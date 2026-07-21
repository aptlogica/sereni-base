package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestExtractUserInfo tests the ExtractUserInfo function
func TestExtractUserInfo(t *testing.T) {
	tests := []struct {
		name           string
		setupCtx       func(*gin.Context)
		expectErr      bool
		expectedUserID string
		expectedSchema string
	}{
		{
			name: "valid user info extraction",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			expectErr:      false,
			expectedUserID: "user123",
			expectedSchema: "test_schema",
		},
		{
			name: "missing user_id",
			setupCtx: func(c *gin.Context) {
				c.Set("schema", "test_schema")
			},
			expectErr: true,
		},
		{
			name: "missing schema",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", "user123")
			},
			expectErr: true,
		},
		{
			name: "both missing",
			setupCtx: func(c *gin.Context) {
				// Don't set anything
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			tt.setupCtx(c)

			userInfo, err := middleware.ExtractUserInfo(c)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, userInfo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, userInfo)
				assert.Equal(t, tt.expectedUserID, userInfo.UserID)
				assert.Equal(t, tt.expectedSchema, userInfo.Schema)
			}
		})
	}
}

// TestExtractScopeInfosFromDatabase tests the ExtractScopeInfosFromDatabase function
func TestExtractScopeInfosFromDatabase(t *testing.T) {
	tests := []struct {
		name           string
		userInfo       *middleware.UserInfo
		mockSetup      func(*MockAccessMemberService)
		expectedScopes int
		expectedTypes  []string
	}{
		{
			name: "user with multiple access members",
			userInfo: &middleware.UserInfo{
				UserID: "user123",
				Schema: "test_schema",
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "workspace",
						ScopeID:   ptrString("scope123"),
					},
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role456",
						ScopeType: "base",
						ScopeID:   ptrString("base456"),
					},
				}, nil)
			},
			expectedScopes: 2,
			expectedTypes:  []string{"workspace", "base"},
		},
		{
			name: "user with no access members",
			userInfo: &middleware.UserInfo{
				UserID: "user123",
				Schema: "test_schema",
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{}, nil)
			},
			expectedScopes: 1,
			expectedTypes:  []string{"workspace"},
		},
		{
			name: "user with nil scope id",
			userInfo: &middleware.UserInfo{
				UserID: "user123",
				Schema: "test_schema",
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "workspace",
						ScopeID:   nil,
					},
				}, nil)
			},
			expectedScopes: 1,
			expectedTypes:  []string{"workspace"},
		},
		{
			name: "database error returns default scope",
			userInfo: &middleware.UserInfo{
				UserID: "user123",
				Schema: "test_schema",
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{}, errors.New("db error"))
			},
			expectedScopes: 1,
			expectedTypes:  []string{"workspace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			scopes := middleware.ExtractScopeInfosFromDatabase(c, tt.userInfo, mockService)

			assert.Len(t, scopes, tt.expectedScopes)
			for i, expectedType := range tt.expectedTypes {
				assert.Equal(t, expectedType, scopes[i].ScopeType)
			}
			mockService.AssertExpectations(t)
		})
	}
}

// TestDefaultMiddleware tests the DefaultMiddleware function
func TestDefaultMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		guardCheck     func() (bool, error)
		expectedStatus int
		shouldAbort    bool
	}{
		{
			name: "guard allows access",
			guardCheck: func() (bool, error) {
				return true, nil
			},
			expectedStatus: http.StatusOK,
			shouldAbort:    false,
		},
		{
			name: "guard denies access without error",
			guardCheck: func() (bool, error) {
				return false, nil
			},
			expectedStatus: http.StatusUnauthorized,
			shouldAbort:    true,
		},
		{
			name: "guard denies access with error",
			guardCheck: func() (bool, error) {
				return false, errors.New("permission denied")
			},
			expectedStatus: http.StatusInternalServerError,
			shouldAbort:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGuard := new(MockGuard)
			allowed, err := tt.guardCheck()
			mockGuard.On("Check", mock.Anything).Return(allowed, err)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			middlewareFn := middleware.DefaultMiddleware(mockGuard)
			r.GET("/test",
				middlewareFn,
				func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"status": "ok"})
				})

			req := httptest.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockGuard.AssertExpectations(t)
		})
	}
}

// TestNewPermissionGuard creates tests for PermissionGuard
func TestNewPermissionGuard(t *testing.T) {
	resourceCode := "table"
	actionCode := "read"

	mockService := new(MockAccessMemberService)

	guard := middleware.NewPermissionGuard(resourceCode, actionCode, mockService)

	assert.NotNil(t, guard)
	assert.NotNil(t, guard.Middleware())
}

// TestPermissionGuardCheck tests the Check method of PermissionGuard
func TestPermissionGuardCheck(t *testing.T) {
	tests := []struct {
		name             string
		resourceCode     string
		actionCode       string
		setupContext     func(*gin.Context)
		mockSetup        func(*MockAccessMemberService)
		expectedAllowed  bool
		expectedHasError bool
	}{
		{
			name:         "missing user info",
			resourceCode: "table",
			actionCode:   "read",
			setupContext: func(c *gin.Context) {
				// Missing user_id and schema
			},
			mockSetup: func(m *MockAccessMemberService) {
			},
			expectedAllowed:  false,
			expectedHasError: true,
		},
		{
			name:         "user has permission",
			resourceCode: "table",
			actionCode:   "read",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "workspace",
						ScopeID:   ptrString("scope123"),
					},
				}, nil)

				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					"workspace",
					&scopeID,
					"table",
					"read",
				).Return(true, nil)
			},
			expectedAllowed:  true,
			expectedHasError: false,
		},
		{
			name:         "user does not have permission",
			resourceCode: "table",
			actionCode:   "delete",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "workspace",
						ScopeID:   ptrString("scope123"),
					},
				}, nil)

				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					"workspace",
					&scopeID,
					"table",
					"delete",
				).Return(false, nil)
			},
			expectedAllowed:  false,
			expectedHasError: false,
		},
		{
			name:         "permission check error",
			resourceCode: "table",
			actionCode:   "read",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "workspace",
						ScopeID:   ptrString("scope123"),
					},
				}, nil)

				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					"workspace",
					&scopeID,
					"table",
					"read",
				).Return(false, errors.New("db error"))
			},
			expectedAllowed:  false,
			expectedHasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			guard := middleware.NewPermissionGuard(tt.resourceCode, tt.actionCode, mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			tt.setupContext(c)

			allowed, err := guard.Check(c)

			assert.Equal(t, tt.expectedAllowed, allowed)
			if tt.expectedHasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockService.AssertExpectations(t)
		})
	}
}

// TestNewRoleGuard creates tests for RoleGuard
func TestNewRoleGuard(t *testing.T) {
	requiredRoles := []string{"owner", "admin"}
	mockService := new(MockAccessMemberService)

	guard := middleware.NewRoleGuard(requiredRoles, mockService, "workspace")

	assert.NotNil(t, guard)
	assert.NotNil(t, guard.Middleware())
}

// TestRoleGuardCheck tests the Check method of RoleGuard
func TestRoleGuardCheck(t *testing.T) {
	tests := []struct {
		name             string
		requiredRoles    []string
		scopeType        string
		setupContext     func(*gin.Context)
		mockSetup        func(*MockAccessMemberService)
		expectedAllowed  bool
		expectedHasError bool
	}{
		{
			name:          "missing user info",
			requiredRoles: []string{"owner"},
			scopeType:     "",
			setupContext: func(c *gin.Context) {
				// Missing user_id and schema
			},
			mockSetup: func(m *MockAccessMemberService) {
			},
			expectedAllowed:  false,
			expectedHasError: true,
		},
		{
			name:          "user has required role",
			requiredRoles: []string{"owner", "admin"},
			scopeType:     "workspace",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "workspace",
						ScopeID:   ptrString("scope123"),
					},
				}, nil)

				scopeID := "scope123"
				m.On("GetUserHighestRole",
					mock.Anything,
					"test_schema",
					"user123",
					"workspace",
					&scopeID,
				).Return(&dto.AccessRoleDTO{
					ID:         uuid.New(),
					Name:       "owner",
					ScopeLevel: "workspace",
					Priority:   1,
				}, nil)
			},
			expectedAllowed:  true,
			expectedHasError: false,
		},
		{
			name:          "user does not have required role",
			requiredRoles: []string{"owner"},
			scopeType:     "workspace",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "workspace",
						ScopeID:   ptrString("scope123"),
					},
				}, nil)

				scopeID := "scope123"
				m.On("GetUserHighestRole",
					mock.Anything,
					"test_schema",
					"user123",
					"workspace",
					&scopeID,
				).Return(&dto.AccessRoleDTO{
					ID:         uuid.New(),
					Name:       "viewer",
					ScopeLevel: "workspace",
					Priority:   100,
				}, nil)
			},
			expectedAllowed:  false,
			expectedHasError: false,
		},
		{
			name:          "user role fetch error",
			requiredRoles: []string{"owner"},
			scopeType:     "workspace",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "workspace",
						ScopeID:   ptrString("scope123"),
					},
				}, nil)

				scopeID := "scope123"
				m.On("GetUserHighestRole",
					mock.Anything,
					"test_schema",
					"user123",
					"workspace",
					&scopeID,
				).Return((*dto.AccessRoleDTO)(nil), errors.New("db error"))
			},
			expectedAllowed:  false,
			expectedHasError: false,
		},
		{
			name:          "scope type checked from access member when not specified",
			requiredRoles: []string{"owner"},
			scopeType:     "",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			mockSetup: func(m *MockAccessMemberService) {
				m.On("GetUserAccessMembers",
					mock.Anything,
					"test_schema",
					"user123",
				).Return([]dto.AccessMemberDTO{
					{
						ID:        uuid.New(),
						UserID:    "user123",
						RoleID:    "role123",
						ScopeType: "base",
						ScopeID:   ptrString("base123"),
					},
				}, nil)

				scopeID := "base123"
				m.On("GetUserHighestRole",
					mock.Anything,
					"test_schema",
					"user123",
					"base",
					&scopeID,
				).Return(&dto.AccessRoleDTO{
					ID:         uuid.New(),
					Name:       "owner",
					ScopeLevel: "base",
					Priority:   1,
				}, nil)
			},
			expectedAllowed:  true,
			expectedHasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			guard := middleware.NewRoleGuard(tt.requiredRoles, mockService, tt.scopeType)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			tt.setupContext(c)

			allowed, err := guard.Check(c)

			assert.Equal(t, tt.expectedAllowed, allowed)
			if tt.expectedHasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockService.AssertExpectations(t)
		})
	}
}

// TestPermissionGuardMiddleware tests the Middleware method of PermissionGuard
func TestPermissionGuardMiddleware(t *testing.T) {
	mockService := new(MockAccessMemberService)

	guard := middleware.NewPermissionGuard("table", "read", mockService)

	assert.NotNil(t, guard.Middleware())
}

// TestRoleGuardMiddleware tests the Middleware method of RoleGuard
func TestRoleGuardMiddleware(t *testing.T) {
	mockService := new(MockAccessMemberService)
	guard := middleware.NewRoleGuard([]string{"owner"}, mockService, "workspace")

	assert.NotNil(t, guard.Middleware())
}

// MockGuard is a mock implementation of the Guard interface
type MockGuard struct {
	mock.Mock
}

func (m *MockGuard) Check(c *gin.Context) (bool, error) {
	args := m.Called(c)
	return args.Bool(0), args.Error(1)
}

func (m *MockGuard) Middleware() gin.HandlerFunc {
	return middleware.DefaultMiddleware(m)
}

// Helper function
func ptrString(s string) *string {
	return &s
}
