package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	guardUserID     = "user-b"
	guardSchema     = "tenant_schema"
	ownWorkspaceID  = "ws-own"
	victimWorkspace = "ws-victim"
	ownBaseID       = "base-own"
	victimBaseID    = "base-victim"
)

func strPtr(s string) *string { return &s }

// runGuard serves a single request through guard and reports whether the handler ran.
func runGuard(t *testing.T, route, path string, guard gin.HandlerFunc, setUser bool) (int, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if setUser {
			c.Set("user_id", guardUserID)
			c.Set("schema", guardSchema)
		}
		c.Next()
	})
	reached := false
	r.GET(route, guard, func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code, reached
}

func membersMock(members []dto.AccessMemberDTO, err error) *MockAccessMemberService {
	m := new(MockAccessMemberService)
	m.On("GetUserAccessMembers", mock.Anything, guardSchema, guardUserID).Return(members, err)
	return m
}

// GHSA-pm5h-jv59-hcmw: GET /workspace/{id} must not disclose workspaces the user is not a member of.
func TestScopeAccessGuard_Workspace(t *testing.T) {
	tests := []struct {
		name    string
		members []dto.AccessMemberDTO
		err     error
		target  string
		allowed bool
	}{
		{"system owner may access any workspace", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.System}}, nil, victimWorkspace, true},
		{"workspace member may access own workspace", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}}, nil, ownWorkspaceID, true},
		{"workspace member is denied another workspace", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}}, nil, victimWorkspace, false},
		{"base member may access the parent workspace", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(ownBaseID), WorkspaceID: strPtr(ownWorkspaceID)}}, nil, ownWorkspaceID, true},
		{"base member is denied an unrelated workspace", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(ownBaseID), WorkspaceID: strPtr(ownWorkspaceID)}}, nil, victimWorkspace, false},
		{"base ID equal to workspace param does not grant access", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(victimWorkspace)}}, nil, victimWorkspace, false},
		{"user with no memberships is denied", []dto.AccessMemberDTO{}, nil, victimWorkspace, false},
		{"membership of unknown scope type is ignored", []dto.AccessMemberDTO{{ScopeType: "organization", ScopeID: strPtr(victimWorkspace)}}, nil, victimWorkspace, false},
		{"membership lookup error is denied", []dto.AccessMemberDTO{}, errors.New("db down"), victimWorkspace, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := membersMock(tt.members, tt.err)
			guard := middleware.NewScopeAccessGuard(constant.ScopeLevels.Workspace, svc, nil)

			code, reached := runGuard(t, "/workspace/:id", "/workspace/"+tt.target, guard, true)

			assert.Equal(t, tt.allowed, reached)
			if !tt.allowed {
				assert.NotEqual(t, http.StatusOK, code)
			}
		})
	}
}

// GHSA-g566-29mc-jprr: GET /base/{id} and /base/{id}/tables must not disclose bases outside the user's scope.
func TestScopeAccessGuard_Base(t *testing.T) {
	resolveTo := func(ws string, err error) middleware.BaseWorkspaceResolver {
		return func(_ context.Context, schema, baseID string) (string, error) {
			assert.Equal(t, guardSchema, schema)
			return ws, err
		}
	}
	failIfCalled := func(t *testing.T) middleware.BaseWorkspaceResolver {
		return func(context.Context, string, string) (string, error) {
			t.Fatal("resolver should not be called")
			return "", nil
		}
	}

	tests := []struct {
		name     string
		members  []dto.AccessMemberDTO
		resolver func(t *testing.T) middleware.BaseWorkspaceResolver
		target   string
		allowed  bool
	}{
		{"system owner may access any base without resolving", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.System}}, failIfCalled, victimBaseID, true},
		{"base member may access own base without resolving", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(ownBaseID)}}, failIfCalled, ownBaseID, true},
		{"base member is denied another base", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(ownBaseID), WorkspaceID: strPtr(ownWorkspaceID)}}, failIfCalled, victimBaseID, false},
		{"workspace member may access bases in that workspace", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}},
			func(*testing.T) middleware.BaseWorkspaceResolver { return resolveTo(ownWorkspaceID, nil) }, ownBaseID, true},
		{"workspace member is denied bases in another workspace", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}},
			func(*testing.T) middleware.BaseWorkspaceResolver { return resolveTo(victimWorkspace, nil) }, victimBaseID, false},
		{"unknown base is denied", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}},
			func(*testing.T) middleware.BaseWorkspaceResolver { return resolveTo("", errors.New("not found")) }, victimBaseID, false},
		{"nil resolver denies workspace members", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}},
			func(*testing.T) middleware.BaseWorkspaceResolver { return nil }, ownBaseID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := membersMock(tt.members, nil)
			guard := middleware.NewScopeAccessGuard(constant.ScopeLevels.Base, svc, tt.resolver(t))

			code, reached := runGuard(t, "/base/:id/tables", "/base/"+tt.target+"/tables", guard, true)

			assert.Equal(t, tt.allowed, reached)
			if !tt.allowed {
				assert.NotEqual(t, http.StatusOK, code)
			}
		})
	}
}

func TestScopeAccessGuard_MissingUserContext(t *testing.T) {
	svc := new(MockAccessMemberService)
	guard := middleware.NewScopeAccessGuard(constant.ScopeLevels.Workspace, svc, nil)

	code, reached := runGuard(t, "/workspace/:id", "/workspace/"+ownWorkspaceID, guard, false)

	assert.False(t, reached)
	assert.NotEqual(t, http.StatusOK, code)
	svc.AssertNotCalled(t, "GetUserAccessMembers", mock.Anything, mock.Anything, mock.Anything)
}

// GHSA-wfm4-mmrh-5h9c: /user/profile/{id} must only allow the user themselves (or admins for reads).
func TestSelfOrRoleGuard(t *testing.T) {
	ownerRoles := []string{constant.RBACRoleNames.Owner, constant.RBACRoleNames.CoOwner}

	t.Run("user may access their own profile without a role lookup", func(t *testing.T) {
		svc := new(MockAccessMemberService)
		code, reached := runGuard(t, "/user/profile/:id", "/user/profile/"+guardUserID, middleware.NewSelfOrRoleGuard(nil, svc), true)
		assert.True(t, reached)
		assert.Equal(t, http.StatusOK, code)
		svc.AssertNotCalled(t, "GetUserAccessMembers", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("self-only guard denies another user's profile", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.System}}, nil)
		svc.On("GetUserHighestRole", mock.Anything, guardSchema, guardUserID, constant.ScopeLevels.System, mock.Anything).
			Return(&dto.AccessRoleDTO{Name: constant.RBACRoleNames.Owner}, nil)
		code, reached := runGuard(t, "/user/profile/:id", "/user/profile/victim", middleware.NewSelfOrRoleGuard(nil, svc), true)
		assert.False(t, reached)
		assert.NotEqual(t, http.StatusOK, code)
	})

	t.Run("regular user is denied another user's profile", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}}, nil)
		svc.On("GetUserHighestRole", mock.Anything, guardSchema, guardUserID, constant.ScopeLevels.Workspace, mock.Anything).
			Return(&dto.AccessRoleDTO{Name: constant.RBACRoleNames.BaseMember}, nil)
		code, reached := runGuard(t, "/user/profile/:id", "/user/profile/victim", middleware.NewSelfOrRoleGuard(ownerRoles, svc), true)
		assert.False(t, reached)
		assert.NotEqual(t, http.StatusOK, code)
	})

	t.Run("owner may read another user's profile", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.System}}, nil)
		svc.On("GetUserHighestRole", mock.Anything, guardSchema, guardUserID, constant.ScopeLevels.System, mock.Anything).
			Return(&dto.AccessRoleDTO{Name: constant.RBACRoleNames.Owner}, nil)
		code, reached := runGuard(t, "/user/profile/:id", "/user/profile/victim", middleware.NewSelfOrRoleGuard(ownerRoles, svc), true)
		assert.True(t, reached)
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("unauthenticated context is denied", func(t *testing.T) {
		svc := new(MockAccessMemberService)
		code, reached := runGuard(t, "/user/profile/:id", "/user/profile/"+guardUserID, middleware.NewSelfOrRoleGuard(nil, svc), false)
		assert.False(t, reached)
		assert.NotEqual(t, http.StatusOK, code)
	})
}

// RateLimiter now guards login; concurrent requests must not crash it or exceed the limit.
func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const limit = 10
	r := gin.New()
	r.POST("/login", middleware.RateLimiter(limit), func(c *gin.Context) { c.Status(http.StatusOK) })

	var mu sync.Mutex
	ok := 0
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", nil))
			if w.Code == http.StatusOK {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, limit, ok)
}

// POST /base/create takes workspace_id from the form; Base:Create must be granted in that workspace,
// not in any workspace the user happens to maintain.
func TestWorkspaceFormPermissionGuard_CreateBase(t *testing.T) {
	ownMaintainer := dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}
	victimReadOnly := dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(victimWorkspace)}

	tests := []struct {
		name    string
		members []dto.AccessMemberDTO
		grants  map[string]bool // scopeID -> has Base:Create there
		target  string
		allowed bool
	}{
		{"system owner may create in any workspace", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.System}}, map[string]bool{"": true}, victimWorkspace, true},
		{"maintainer may create in own workspace", []dto.AccessMemberDTO{ownMaintainer}, map[string]bool{ownWorkspaceID: true}, ownWorkspaceID, true},
		{"maintainer is denied another workspace", []dto.AccessMemberDTO{ownMaintainer}, map[string]bool{ownWorkspaceID: true}, victimWorkspace, false},
		{"maintainer elsewhere and read-only member of target is denied", []dto.AccessMemberDTO{ownMaintainer, victimReadOnly},
			map[string]bool{ownWorkspaceID: true, victimWorkspace: false}, victimWorkspace, false},
		{"base member of target workspace is denied", []dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(ownBaseID), WorkspaceID: strPtr(victimWorkspace)}},
			map[string]bool{ownBaseID: true}, victimWorkspace, false},
		{"missing workspace_id is denied", []dto.AccessMemberDTO{ownMaintainer}, map[string]bool{ownWorkspaceID: true}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := membersMock(tt.members, nil)
			for scopeID, granted := range tt.grants {
				id := scopeID
				svc.On("CheckUserPermission", mock.Anything, guardSchema, guardUserID, mock.Anything,
					mock.MatchedBy(func(s *string) bool { return s != nil && *s == id }),
					constant.ResourceCodes.Base, constant.ActionCodes.Create).Return(granted, nil).Maybe()
			}
			guard := middleware.NewWorkspaceFormPermissionGuard("workspace_id", constant.ResourceCodes.Base, constant.ActionCodes.Create, svc)

			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("user_id", guardUserID)
				c.Set("schema", guardSchema)
				c.Next()
			})
			reached := false
			r.POST("/base/create", guard, func(c *gin.Context) {
				reached = true
				c.Status(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodPost, "/base/create", strings.NewReader(url.Values{"workspace_id": {tt.target}, "title": {"B"}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.allowed, reached)
			if !tt.allowed {
				assert.NotEqual(t, http.StatusOK, w.Code)
			}
		})
	}
}
