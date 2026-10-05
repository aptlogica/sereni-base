package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/router"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// regularMemberAccess models "User B": a plain member of their own workspace only.
type regularMemberAccess struct{}

func (regularMemberAccess) GetUserAccessMembers(context.Context, string, string) ([]dto.AccessMemberDTO, error) {
	ws := "ws-attacker"
	return []dto.AccessMemberDTO{{UserID: "user-b", ScopeType: constant.ScopeLevels.Workspace, ScopeID: &ws}}, nil
}
func (regularMemberAccess) GetUserHighestRole(context.Context, string, string, string, *string) (*dto.AccessRoleDTO, error) {
	return &dto.AccessRoleDTO{Name: constant.RBACRoleNames.BaseMember}, nil
}
func (regularMemberAccess) CheckUserPermission(context.Context, string, string, string, *string, string, string) (bool, error) {
	return true, nil // even with the permission somewhere, object-level checks must still deny
}
func (regularMemberAccess) AssignRoleToUser(context.Context, string, dto.AccessMemberDTO) (interface{}, error) {
	return nil, nil
}
func (regularMemberAccess) RemoveRoleFromUser(context.Context, string, string, string, string) error {
	return nil
}
func (regularMemberAccess) RemoveAccessMemberByID(context.Context, string, string) error { return nil }
func (regularMemberAccess) UpdateRoleForUser(context.Context, string, string, string, *string, string) error {
	return nil
}
func (regularMemberAccess) GetUserAccessByScope(context.Context, string, string, string, *string) ([]dto.AccessMemberDTO, error) {
	return nil, nil
}
func (regularMemberAccess) GetScopeMembers(context.Context, string, string, *string) ([]dto.AccessMemberDTO, error) {
	return nil, nil
}
func (regularMemberAccess) GetUserPermissions(context.Context, string, string, string, *string) ([]dto.PermissionWithDetails, error) {
	return nil, nil
}
func (regularMemberAccess) BulkAssignRoleToUsers(context.Context, string, dto.BulkAssignRoleRequest) error {
	return nil
}
func (regularMemberAccess) BulkRemoveRoleFromUsers(context.Context, string, []string, string, *string, string) error {
	return nil
}

// TestObjectLevelAuthorizationWiring drives the real router as User B against another
// user's workspace, base and profile (GHSA-pm5h-jv59-hcmw, GHSA-g566-29mc-jprr, GHSA-wfm4-mmrh-5h9c).
func TestObjectLevelAuthorizationWiring(t *testing.T) {
	gin.SetMode(gin.TestMode)

	middlewares := createMockMiddlewares()
	middlewares.AuthMiddleware = func() gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("user_id", "user-b")
			c.Set("schema", "tenant")
			c.Next()
		}
	}
	middlewares.AccessMemberService = regularMemberAccess{}
	middlewares.BaseWorkspaceID = func(_ context.Context, _, baseID string) (string, error) {
		if baseID == "base-victim" {
			return "ws-victim", nil
		}
		return "", errors.New("base not found")
	}

	r := router.Setup(&config.Config{}, createMockHandlers(), middlewares)

	denied := []struct{ method, path string }{
		// Workspace
		{http.MethodGet, "/api/v1/workspace/ws-victim"},
		{http.MethodGet, "/api/v1/workspace/ws-victim/bases"},
		{http.MethodGet, "/api/v1/workspace/ws-victim/tables"},
		{http.MethodPut, "/api/v1/workspace/ws-victim"},
		{http.MethodDelete, "/api/v1/workspace/ws-victim"},
		{http.MethodGet, "/api/v1/workspace/ws-victim/members"},
		{http.MethodPost, "/api/v1/workspace/ws-victim/remove"},
		// Base
		{http.MethodGet, "/api/v1/base/base-victim"},
		{http.MethodGet, "/api/v1/base/base-victim/tables"},
		{http.MethodPut, "/api/v1/base/base-victim"},
		{http.MethodDelete, "/api/v1/base/base-victim"},
		{http.MethodPost, "/api/v1/base/base-victim/image"},
		{http.MethodGet, "/api/v1/base/base-victim/members"},
		{http.MethodGet, "/api/v1/base/unknown-base"},
		// User profile
		{http.MethodGet, "/api/v1/user/profile/user-a"},
		{http.MethodPatch, "/api/v1/user/profile/user-a"},
		{http.MethodPost, "/api/v1/user/profile/user-a/avatar"},
		{http.MethodDelete, "/api/v1/user/profile/user-a/avatar"},
		{http.MethodPost, "/api/v1/user/change-password/user-a"},
		{http.MethodGet, "/api/v1/user/roles-and-access/user-a"},
	}

	for _, tc := range denied {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}")))

			assert.NotEqual(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), "ERR_0002", "expected UnauthorizedAccess, got %d %s", w.Code, w.Body.String())
		})
	}
}
