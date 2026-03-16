package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorkspaceMemberService is a mock implementation of the WorkspaceMemberService
type MockWorkspaceMemberService struct {
	mock.Mock
}

func (m *MockWorkspaceMemberService) GetWorkspaceMemberByUserAndWorkspace(ctx context.Context, schema, userId, workspaceId string) (*tenant.WorkspaceMember, error) {
	args := m.Called(ctx, schema, userId, workspaceId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceMemberService) GetAllWorkspaceMembersByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error) {
	args := m.Called(ctx, schemaName, userId)
	return args.Get(0).([]tenant.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceMemberService) DeleteWorkspaceMember(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}

func (m *MockWorkspaceMemberService) GetWorkspaceMemberByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error) {
	args := m.Called(ctx, schemaName, userId)
	return args.Get(0).([]tenant.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceMemberService) GetWorkspaceMembersByWorkspace(ctx context.Context, schemaName string, workspaceId string) ([]tenant.WorkspaceMember, error) {
	args := m.Called(ctx, schemaName, workspaceId)
	return args.Get(0).([]tenant.WorkspaceMember), args.Error(1)
}

func (m *MockWorkspaceMemberService) DeleteUserMappings(ctx context.Context, schemaName string, userId string) error {
	args := m.Called(ctx, schemaName, userId)
	return args.Error(0)
}

func (m *MockWorkspaceMemberService) UpdateWorkspaceMemberBases(ctx context.Context, schemaName string, workspaceId string, userId string, accessLevel string, basesIds string) error {
	args := m.Called(ctx, schemaName, workspaceId, userId, accessLevel, basesIds)
	return args.Error(0)
}

// MockAccessMemberService is a mock implementation of the AccessMemberService
type MockAccessMemberService struct {
	mock.Mock
}

func (m *MockAccessMemberService) CheckUserPermission(ctx context.Context, schema, userId, scopeType string, scopeID *string, resourceCode, actionCode string) (bool, error) {
	args := m.Called(ctx, schema, userId, scopeType, scopeID, resourceCode, actionCode)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccessMemberService) GetUserHighestRole(ctx context.Context, schema, userId, scopeType string, scopeID *string) (*dto.AccessRoleDTO, error) {
	args := m.Called(ctx, schema, userId, scopeType, scopeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AccessRoleDTO), args.Error(1)
}

func (m *MockAccessMemberService) GetUserAccessByScope(ctx context.Context, schema, userId, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	args := m.Called(ctx, schema, userId, scopeType, scopeID)
	return args.Get(0).([]dto.AccessMemberDTO), args.Error(1)
}

func (m *MockAccessMemberService) AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0), args.Error(1)
}

func (m *MockAccessMemberService) RemoveRoleFromUser(ctx context.Context, schemaName string, userID, scopeID string, scopeType string) error {
	args := m.Called(ctx, schemaName, userID, scopeID, scopeType)
	return args.Error(0)
}

func (m *MockAccessMemberService) RemoveAccessMemberByID(ctx context.Context, schemaName string, memberID string) error {
	args := m.Called(ctx, schemaName, memberID)
	return args.Error(0)
}

func (m *MockAccessMemberService) UpdateRoleForUser(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, newRoleID string) error {
	args := m.Called(ctx, schemaName, userID, scopeType, scopeID, newRoleID)
	return args.Error(0)
}

func (m *MockAccessMemberService) GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
	args := m.Called(ctx, schemaName, userID)
	return args.Get(0).([]dto.AccessMemberDTO), args.Error(1)
}

func (m *MockAccessMemberService) GetScopeMembers(ctx context.Context, schemaName string, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	args := m.Called(ctx, schemaName, scopeType, scopeID)
	return args.Get(0).([]dto.AccessMemberDTO), args.Error(1)
}

func (m *MockAccessMemberService) GetUserPermissions(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.PermissionWithDetails, error) {
	args := m.Called(ctx, schemaName, userID, scopeType, scopeID)
	return args.Get(0).([]dto.PermissionWithDetails), args.Error(1)
}

func (m *MockAccessMemberService) BulkAssignRoleToUsers(ctx context.Context, schemaName string, req dto.BulkAssignRoleRequest) error {
	args := m.Called(ctx, schemaName, req)
	return args.Error(0)
}

func (m *MockAccessMemberService) BulkRemoveRoleFromUsers(ctx context.Context, schemaName string, userIDs []string, scopeType string, scopeID *string, roleID string) error {
	args := m.Called(ctx, schemaName, userIDs, scopeType, scopeID, roleID)
	return args.Error(0)
}

// TestScopeHeaderMiddleware tests the ScopeHeaderMiddleware function
func TestScopeHeaderMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		scope     string
		wantScope string
	}{
		{
			name:      "sets workspace scope",
			scope:     "workspace",
			wantScope: "workspace",
		},
		{
			name:      "sets base scope",
			scope:     "base",
			wantScope: "base",
		},
		{
			name:      "sets custom scope",
			scope:     "custom",
			wantScope: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.ScopeHeaderMiddleware(tt.scope), func(c *gin.Context) {
				scope := c.Request.Header.Get("Scope")
				assert.Equal(t, tt.wantScope, scope)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestWorkspaceAndBaseAccessValidationMiddleware tests workspace and base access validation
func TestWorkspaceAndBaseAccessValidationMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		headers        map[string]string
		mockSetup      func(*MockWorkspaceMemberService)
		allowedAccess  []string
		expectedStatus int
	}{
		{
			name: "missing user_id in context",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "test_schema")
			},
			headers:        map[string]string{"workspace": "ws123"},
			mockSetup:      func(m *MockWorkspaceMemberService) {},
			allowedAccess:  []string{"owner", "admin"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing schema in context",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
			},
			headers:        map[string]string{"workspace": "ws123"},
			mockSetup:      func(m *MockWorkspaceMemberService) {},
			allowedAccess:  []string{"owner", "admin"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing workspace header",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers:        map[string]string{},
			mockSetup:      func(m *MockWorkspaceMemberService) {},
			allowedAccess:  []string{"owner", "admin"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "workspace member not found",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{"workspace": "ws123"},
			mockSetup: func(m *MockWorkspaceMemberService) {
				m.On("GetWorkspaceMemberByUserAndWorkspace",
					mock.Anything,
					"test_schema",
					"user123",
					"ws123",
				).Return((*tenant.WorkspaceMember)(nil), errors.New("not found"))
			},
			allowedAccess:  []string{"owner", "admin"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "insufficient access level",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{"workspace": "ws123"},
			mockSetup: func(m *MockWorkspaceMemberService) {
				m.On("GetWorkspaceMemberByUserAndWorkspace",
					mock.Anything,
					"test_schema",
					"user123",
					"ws123",
				).Return(&tenant.WorkspaceMember{
					AccessLevel: "viewer",
					BasesIds:    "*",
				}, nil)
			},
			allowedAccess:  []string{"owner", "admin"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "valid access - owner",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{"workspace": "ws123"},
			mockSetup: func(m *MockWorkspaceMemberService) {
				m.On("GetWorkspaceMemberByUserAndWorkspace",
					mock.Anything,
					"test_schema",
					"user123",
					"ws123",
				).Return(&tenant.WorkspaceMember{
					AccessLevel: "owner",
					BasesIds:    "*",
				}, nil)
			},
			allowedAccess:  []string{"owner", "admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid access - admin",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{"workspace": "ws123"},
			mockSetup: func(m *MockWorkspaceMemberService) {
				m.On("GetWorkspaceMemberByUserAndWorkspace",
					mock.Anything,
					"test_schema",
					"user123",
					"ws123",
				).Return(&tenant.WorkspaceMember{
					AccessLevel: "admin",
					BasesIds:    "*",
				}, nil)
			},
			allowedAccess:  []string{"owner", "admin"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockWorkspaceMemberService)
			tt.mockSetup(mockService)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test",
				func(c *gin.Context) {
					// Setup context in middleware chain
					tt.setupContext(c)
					c.Next()
				},
				middleware.WorkspaceAndBaseAccessValidationMiddleware(mockService, tt.allowedAccess),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestCheckPermissionMiddleware tests the RBAC permission checking middleware
func TestCheckPermissionMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		headers        map[string]string
		mockSetup      func(*MockAccessMemberService)
		resourceCode   string
		actionCode     string
		expectedStatus int
	}{
		{
			name: "missing user_id",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "test_schema")
			},
			headers:        map[string]string{},
			mockSetup:      func(m *MockAccessMemberService) {},
			resourceCode:   "table",
			actionCode:     "read",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing schema",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
			},
			headers:        map[string]string{},
			mockSetup:      func(m *MockAccessMemberService) {},
			resourceCode:   "table",
			actionCode:     "read",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "permission check error",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"table",
					"read",
				).Return(false, errors.New("db error"))
			},
			resourceCode:   "table",
			actionCode:     "read",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "permission denied",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"table",
					"read",
				).Return(false, nil)
			},
			resourceCode:   "table",
			actionCode:     "read",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "permission granted",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"table",
					"read",
				).Return(true, nil)
			},
			resourceCode:   "table",
			actionCode:     "read",
			expectedStatus: http.StatusOK,
		},
		{
			name: "permission granted with default workspace scope",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeID: "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"column",
					"update",
				).Return(true, nil)
			},
			resourceCode:   "column",
			actionCode:     "update",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test",
				func(c *gin.Context) {
					// Setup context in middleware chain
					tt.setupContext(c)
					c.Next()
				},
				middleware.CheckPermissionMiddleware(mockService, tt.resourceCode, tt.actionCode),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestCheckRoleMiddleware tests role-based access control
func TestCheckRoleMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		headers        map[string]string
		mockSetup      func(*MockAccessMemberService)
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name: "user has required role",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("GetUserHighestRole",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
				).Return(&dto.AccessRoleDTO{Name: "admin"}, nil)
			},
			requiredRoles:  []string{"admin", "owner"},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user does not have required role",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("GetUserHighestRole",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
				).Return(&dto.AccessRoleDTO{Name: "viewer"}, nil)
			},
			requiredRoles:  []string{"admin", "owner"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "error getting role",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("GetUserHighestRole",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
				).Return((*dto.AccessRoleDTO)(nil), errors.New("db error"))
			},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test",
				func(c *gin.Context) {
					// Setup context in middleware chain
					tt.setupContext(c)
					c.Next()
				},
				middleware.CheckRoleMiddleware(mockService, tt.requiredRoles),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestValidateAccessScopeMiddleware tests access scope validation
func TestValidateAccessScopeMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		headers        map[string]string
		mockSetup      func(*MockAccessMemberService)
		expectedStatus int
	}{
		{
			name: "missing scope type and id",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers:        map[string]string{},
			mockSetup:      func(m *MockAccessMemberService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user has access to scope",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("GetUserAccessByScope",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
				).Return([]dto.AccessMemberDTO{{UserID: "user123"}}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user does not have access to scope",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("GetUserAccessByScope",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
				).Return([]dto.AccessMemberDTO{}, nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test",
				func(c *gin.Context) {
					// Setup context in middleware chain
					tt.setupContext(c)
					c.Next()
				},
				middleware.ValidateAccessScopeMiddleware(mockService),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestRequirePermissionsMiddleware tests multiple permission checking
func TestRequirePermissionsMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		headers        map[string]string
		mockSetup      func(*MockAccessMemberService)
		permissions    []struct{ Resource, Action string }
		expectedStatus int
	}{
		{
			name: "all permissions granted",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"table",
					"read",
				).Return(true, nil)
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"table",
					"write",
				).Return(true, nil)
			},
			permissions: []struct{ Resource, Action string }{
				{"table", "read"},
				{"table", "write"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "one permission denied",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			headers: map[string]string{
				middleware.HeaderScopeType: constant.ScopeLevels.Workspace,
				middleware.HeaderScopeID:   "scope123",
			},
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"table",
					"read",
				).Return(true, nil)
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"table",
					"delete",
				).Return(false, nil)
			},
			permissions: []struct{ Resource, Action string }{
				{"table", "read"},
				{"table", "delete"},
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test",
				func(c *gin.Context) {
					// Setup context in middleware chain
					tt.setupContext(c)
					c.Next()
				},
				middleware.RequirePermissionsMiddleware(mockService, tt.permissions),
				func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
