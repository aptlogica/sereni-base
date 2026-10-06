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
	"github.com/aptlogica/sereni-base/internal/middleware"
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
	middlewares.ScopeLookups = testScopeLookups()

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

// testScopeLookups models two workspaces: User B's (ws-attacker) and a victim's (ws-victim),
// each with one base, table, column and view.
func testScopeLookups() middleware.ScopeLookups {
	lookup := func(table map[string]string) func(context.Context, string, string) (string, error) {
		return func(_ context.Context, _, id string) (string, error) {
			if v, ok := table[id]; ok {
				return v, nil
			}
			return "", errors.New("not found")
		}
	}
	str := func(s string) *string { return &s }
	return middleware.ScopeLookups{
		BaseWorkspace: lookup(map[string]string{"base-victim": "ws-victim", "base-own": "ws-attacker"}),
		ModelBase:     lookup(map[string]string{"model-victim": "base-victim", "model-own": "base-own"}),
		ColumnModel:   lookup(map[string]string{"col-victim": "model-victim", "col-own": "model-own"}),
		ViewModel:     lookup(map[string]string{"view-victim": "model-victim", "view-own": "model-own"}),
		AccessMember: func(_ context.Context, _, id string) (dto.AccessMemberDTO, error) {
			switch id {
			case "am-victim":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Workspace, ScopeID: str("ws-victim")}, nil
			case "am-own":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Workspace, ScopeID: str("ws-attacker")}, nil
			case "am-owner":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.System}, nil
			}
			return dto.AccessMemberDTO{}, errors.New("not found")
		},
	}
}

// User B is a member of ws-attacker only. Every table, column, row, view and member route must
// deny objects in ws-victim even though User B holds the permission in their own workspace, and
// must still let User B through on their own objects.
func TestObjectLevelAuthorization_ContentRoutes(t *testing.T) {
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
	middlewares.ScopeLookups = testScopeLookups()
	r := router.Setup(&config.Config{}, createMockHandlers(), middlewares)

	type tc struct{ method, path, body string }
	serve := func(c tc) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	denied := []tc{
		// Tables
		{http.MethodPost, "/api/v1/table/create", `{"base_id":"base-victim","workspace_id":"ws-victim","title":"x"}`},
		{http.MethodPost, "/api/v1/table/create", `{"base_id":"base-victim","workspace_id":"ws-attacker","title":"x"}`},
		{http.MethodGet, "/api/v1/table/model-victim", ""},
		{http.MethodPatch, "/api/v1/table/model-victim", `{}`},
		{http.MethodDelete, "/api/v1/table/model-victim", ""},
		{http.MethodGet, "/api/v1/table/model-victim/records", ""},
		{http.MethodGet, "/api/v1/table/model-victim/columns", ""},
		{http.MethodGet, "/api/v1/table/", ""},
		// Columns
		{http.MethodPost, "/api/v1/column/create", `{"model_id":"model-victim","title":"x"}`},
		{http.MethodPost, "/api/v1/column/create", `{"model_id":"model-own","meta":{"relation":{"with":"model-victim"}}}`},
		{http.MethodGet, "/api/v1/column/col-victim", ""},
		{http.MethodPatch, "/api/v1/column/col-victim", `{}`},
		{http.MethodDelete, "/api/v1/column/col-victim", ""},
		{http.MethodPost, "/api/v1/column/reorder", `{"source_column_id":"col-own","target_column_id":"col-victim"}`},
		{http.MethodPost, "/api/v1/column/reset", `{"model_id":"model-own","column_id":"col-victim"}`},
		{http.MethodPost, "/api/v1/column/trim-whitespace", `{"model_id":"model-victim","columns":["col-victim"]}`},
		{http.MethodGet, "/api/v1/column/", ""},
		// Rows
		{http.MethodPost, "/api/v1/row/create", `{"model_id":"model-victim","rows":[{}]}`},
		{http.MethodPatch, "/api/v1/row/update", `{"model_id":"model-own","row_id":1,"values":{"col-victim":"x"}}`},
		{http.MethodPost, "/api/v1/row/remove", `{"model_id":"model-victim","row_id":1}`},
		{http.MethodPost, "/api/v1/row/bulk-remove", `{"model_id":"model-victim","row_ids":[1]}`},
		{http.MethodPost, "/api/v1/row/data/insert", `{"model_id":"model-own","column_id":"col-victim","row_id":1}`},
		{http.MethodPost, "/api/v1/row/attachment/remove", `{"model_id":"model-victim","column_id":"col-victim","row_id":1}`},
		// Views
		{http.MethodPost, "/api/v1/view/create", `{"model_id":"model-victim","base_id":"base-victim"}`},
		{http.MethodPost, "/api/v1/view/create", `{"model_id":"model-own","base_id":"base-victim"}`},
		{http.MethodGet, "/api/v1/view/view-victim", ""},
		{http.MethodPatch, "/api/v1/view/view-victim", `{}`},
		{http.MethodDelete, "/api/v1/view/view-victim", ""},
		// Members
		{http.MethodPost, "/api/v1/user/assign", `{"user_id":"u","membership":[{"workspace_id":"ws-victim","role":"maintainer"}]}`},
		{http.MethodPut, "/api/v1/user/access/update", `{"user_id":"u","membership":[{"workspace_id":"ws-attacker","bases":[{"base_id":"base-victim","role":"base-member"}]}]}`},
		{http.MethodDelete, "/api/v1/workspace/access/am-victim", ""},
		{http.MethodDelete, "/api/v1/workspace/access/am-owner", ""},
		{http.MethodDelete, "/api/v1/base/access/am-own", ""}, // workspace row via the base route
		{http.MethodPost, "/api/v1/base/base-victim/bulk-add-members", `{}`},
		{http.MethodGet, "/api/v1/workspace/", ""},
		// Assets cannot be scoped, so only owner/co-owner may address them by ID
		{http.MethodDelete, "/api/v1/asset/asset-1", ""},
	}
	for _, c := range denied {
		t.Run("deny "+c.method+" "+c.path+" "+c.body, func(t *testing.T) {
			w := serve(c)
			assert.Contains(t, w.Body.String(), "ERR_0002", "expected UnauthorizedAccess, got %d %s", w.Code, w.Body.String())
		})
	}

	// Own objects pass the guard; the stub handlers then fail, but not with UnauthorizedAccess.
	allowed := []tc{
		{http.MethodPost, "/api/v1/table/create", `{"base_id":"base-own","workspace_id":"ws-attacker","title":"x"}`},
		{http.MethodGet, "/api/v1/table/model-own/records", ""},
		{http.MethodPost, "/api/v1/column/reorder", `{"source_column_id":"col-own","target_column_id":"col-own"}`},
		{http.MethodPatch, "/api/v1/row/update", `{"model_id":"model-own","row_id":1,"values":{"col-own":"x"}}`},
		{http.MethodPatch, "/api/v1/view/view-own", `{}`},
		{http.MethodPost, "/api/v1/user/assign", `{"user_id":"u","membership":[{"workspace_id":"ws-attacker","bases":[{"base_id":"base-own","role":"base-member"}]}]}`},
		{http.MethodDelete, "/api/v1/workspace/access/am-own", ""},
	}
	for _, c := range allowed {
		t.Run("allow "+c.method+" "+c.path, func(t *testing.T) {
			w := serve(c)
			assert.NotContains(t, w.Body.String(), "ERR_0002", "guard denied own object: %d %s", w.Code, w.Body.String())
		})
	}
}
