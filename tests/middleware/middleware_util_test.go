package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestNewMiddlewareUtil tests creating a new middleware utility
func TestNewMiddlewareUtil(t *testing.T) {
	util := middleware.NewMiddlewareUtil()
	assert.NotNil(t, util, "MiddlewareUtil should not be nil")
}

// TestMiddlewareUtil_ExtractUserAndSchemaFromContext tests user and schema extraction
func TestMiddlewareUtil_ExtractUserAndSchemaFromContext(t *testing.T) {
	util := middleware.NewMiddlewareUtil()

	tests := []struct {
		name            string
		setupContext    func(*gin.Context)
		expectedUserID  string
		expectedSchema  string
		expectedSuccess bool
	}{
		{
			name: "both present",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			expectedUserID:  "user123",
			expectedSchema:  "test_schema",
			expectedSuccess: true,
		},
		{
			name: "missing user_id",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "test_schema")
			},
			expectedUserID:  "",
			expectedSchema:  "",
			expectedSuccess: false,
		},
		{
			name: "missing schema",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
			},
			expectedUserID:  "",
			expectedSchema:  "",
			expectedSuccess: false,
		},
		{
			name: "both missing",
			setupContext: func(c *gin.Context) {
				// Don't set anything
			},
			expectedUserID:  "",
			expectedSchema:  "",
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			tt.setupContext(c)

			userID, schema, ok := util.ExtractUserAndSchemaFromContext(c)

			assert.Equal(t, tt.expectedSuccess, ok)
			if tt.expectedSuccess {
				assert.Equal(t, tt.expectedUserID, userID)
				assert.Equal(t, tt.expectedSchema, schema)
			}
		})
	}
}

// TestMiddlewareUtil_ExtractScopeFromHeaders tests scope extraction
func TestMiddlewareUtil_ExtractScopeFromHeaders(t *testing.T) {
	util := middleware.NewMiddlewareUtil()

	tests := []struct {
		name              string
		headers           map[string]string
		expectedScopeType string
		expectedScopeID   string
	}{
		{
			name: "both headers present",
			headers: map[string]string{
				middleware.HeaderScopeType: "workspace",
				middleware.HeaderScopeID:   "ws123",
			},
			expectedScopeType: "workspace",
			expectedScopeID:   "ws123",
		},
		{
			name: "missing scope type",
			headers: map[string]string{
				middleware.HeaderScopeID: "ws123",
			},
			expectedScopeType: "",
			expectedScopeID:   "ws123",
		},
		{
			name: "missing scope id",
			headers: map[string]string{
				middleware.HeaderScopeType: "base",
			},
			expectedScopeType: "base",
			expectedScopeID:   "",
		},
		{
			name:              "both missing",
			headers:           map[string]string{},
			expectedScopeType: "",
			expectedScopeID:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", func(c *gin.Context) {
				scopeType, scopeID := util.ExtractScopeFromHeaders(c)
				assert.Equal(t, tt.expectedScopeType, scopeType)
				assert.Equal(t, tt.expectedScopeID, scopeID)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			c.Request = req
			r.ServeHTTP(w, req)
		})
	}
}

// TestMiddlewareUtil_SendUnauthorizedError tests sending unauthorized error
func TestMiddlewareUtil_SendUnauthorizedError(t *testing.T) {
	util := middleware.NewMiddlewareUtil()

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		util.SendUnauthorizedError(c)
	}, func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	c.Request = req
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled, "Handler should not be called after SendUnauthorizedError")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Mock type for workspace member with BasesIds getter
type MockWorkspaceMemberWithBases struct {
	basesIds string
}

func (m MockWorkspaceMemberWithBases) GetBasesIds() string {
	return m.basesIds
}

// TestMiddlewareUtil_ValidateBaseAccess tests base access validation
func TestMiddlewareUtil_ValidateBaseAccess(t *testing.T) {
	util := middleware.NewMiddlewareUtil()

	tests := []struct {
		name                string
		scopeType           string
		baseHeader          string
		workspaceMemberData interface{}
		expectedResult      bool
		expectedStatus      int
	}{
		{
			name:      "workspace scope - no base check",
			scopeType: middleware.ScopeWorkspace,
			workspaceMemberData: MockWorkspaceMemberWithBases{
				basesIds: "base1,base2",
			},
			expectedResult: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:       "base scope - missing base header",
			scopeType:  middleware.ScopeBase,
			baseHeader: "",
			workspaceMemberData: MockWorkspaceMemberWithBases{
				basesIds: "*",
			},
			expectedResult: false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "base scope - wildcard access",
			scopeType:  middleware.ScopeBase,
			baseHeader: "base123",
			workspaceMemberData: MockWorkspaceMemberWithBases{
				basesIds: "*",
			},
			expectedResult: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:       "base scope - allowed base",
			scopeType:  middleware.ScopeBase,
			baseHeader: "base2",
			workspaceMemberData: MockWorkspaceMemberWithBases{
				basesIds: "base1,base2,base3",
			},
			expectedResult: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:       "base scope - not allowed base",
			scopeType:  middleware.ScopeBase,
			baseHeader: "base999",
			workspaceMemberData: MockWorkspaceMemberWithBases{
				basesIds: "base1,base2,base3",
			},
			expectedResult: false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "base scope - with spaces in basesIds",
			scopeType:  middleware.ScopeBase,
			baseHeader: "base2",
			workspaceMemberData: MockWorkspaceMemberWithBases{
				basesIds: "base1, base2, base3",
			},
			expectedResult: true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", func(c *gin.Context) {
				result := util.ValidateBaseAccess(c, tt.workspaceMemberData, tt.scopeType)
				assert.Equal(t, tt.expectedResult, result)
				if result {
					c.Status(http.StatusOK)
				}
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.baseHeader != "" {
				req.Header.Set("base", tt.baseHeader)
			}
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestMiddlewareUtil_CheckAccessLevel tests access level checking
func TestMiddlewareUtil_CheckAccessLevel(t *testing.T) {
	util := middleware.NewMiddlewareUtil()

	tests := []struct {
		name            string
		userAccessLevel string
		allowedAccess   []string
		expectedResult  bool
		expectedStatus  int
	}{
		{
			name:            "access allowed - owner",
			userAccessLevel: "owner",
			allowedAccess:   []string{"owner", "admin"},
			expectedResult:  true,
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "access allowed - admin",
			userAccessLevel: "admin",
			allowedAccess:   []string{"owner", "admin"},
			expectedResult:  true,
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "access denied - viewer",
			userAccessLevel: "viewer",
			allowedAccess:   []string{"owner", "admin"},
			expectedResult:  false,
			expectedStatus:  http.StatusUnauthorized,
		},
		{
			name:            "access allowed - single level",
			userAccessLevel: "editor",
			allowedAccess:   []string{"editor"},
			expectedResult:  true,
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "access denied - empty allowed list",
			userAccessLevel: "admin",
			allowedAccess:   []string{},
			expectedResult:  false,
			expectedStatus:  http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", func(c *gin.Context) {
				result := util.CheckAccessLevel(c, tt.userAccessLevel, tt.allowedAccess)
				assert.Equal(t, tt.expectedResult, result)
				if result {
					c.Status(http.StatusOK)
				}
			})

			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestMiddlewareUtil_CheckUserPermission tests user permission checking
func TestMiddlewareUtil_CheckUserPermission(t *testing.T) {
	util := middleware.NewMiddlewareUtil()

	tests := []struct {
		name           string
		mockSetup      func(*MockAccessMemberService)
		resourceCode   string
		actionCode     string
		expectedResult bool
	}{
		{
			name: "permission granted",
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
			expectedResult: true,
		},
		{
			name: "permission denied",
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
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
			resourceCode:   "table",
			actionCode:     "delete",
			expectedResult: false,
		},
		{
			name: "permission check error",
			mockSetup: func(m *MockAccessMemberService) {
				scopeID := "scope123"
				m.On("CheckUserPermission",
					mock.Anything,
					"test_schema",
					"user123",
					constant.ScopeLevels.Workspace,
					&scopeID,
					"table",
					"write",
				).Return(false, errors.New("db error"))
			},
			resourceCode:   "table",
			actionCode:     "write",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			scopeID := "scope123"
			req := middleware.PermissionRequest{
				Schema:       "test_schema",
				UserID:       "user123",
				ScopeType:    constant.ScopeLevels.Workspace,
				ScopeID:      &scopeID,
				ResourceCode: tt.resourceCode,
				ActionCode:   tt.actionCode,
			}
			result := util.CheckUserPermission(c, mockService, req)

			assert.Equal(t, tt.expectedResult, result)
			mockService.AssertExpectations(t)
		})
	}
}

// TestMiddlewareUtil_CheckUserRole tests user role checking
func TestMiddlewareUtil_CheckUserRole(t *testing.T) {
	util := middleware.NewMiddlewareUtil()

	tests := []struct {
		name           string
		mockSetup      func(*MockAccessMemberService)
		requiredRoles  []string
		expectedResult bool
	}{
		{
			name: "user has required role",
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
			expectedResult: true,
		},
		{
			name: "user does not have required role",
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
			expectedResult: false,
		},
		{
			name: "error getting role",
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
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockAccessMemberService)
			tt.mockSetup(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			scopeID := "scope123"
			result := util.CheckUserRole(
				c,
				mockService,
				"test_schema",
				"user123",
				constant.ScopeLevels.Workspace,
				&scopeID,
				tt.requiredRoles,
			)

			assert.Equal(t, tt.expectedResult, result)
			mockService.AssertExpectations(t)
		})
	}
}

// TestMiddlewareUtil_Integration tests integration of multiple utility methods
func TestMiddlewareUtil_Integration(t *testing.T) {
	util := middleware.NewMiddlewareUtil()

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user_id", "user123")
		c.Set("schema", "test_schema")

		userID, schema, ok := util.ExtractUserAndSchemaFromContext(c)
		assert.True(t, ok)
		assert.Equal(t, "user123", userID)
		assert.Equal(t, "test_schema", schema)

		scopeType, scopeID := util.ExtractScopeFromHeaders(c)
		assert.Equal(t, "workspace", scopeType)
		assert.Equal(t, "ws123", scopeID)

		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(middleware.HeaderScopeType, "workspace")
	req.Header.Set(middleware.HeaderScopeID, "ws123")
	c.Request = req
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
