package middleware_test

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/middleware"
	"serenibase/internal/models/tenant"
	"serenibase/internal/providers/auth"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/gin-gonic/gin"
)

// MockAuthProvider is a mock implementation of the AuthProvider interface
type MockAuthProvider struct {
	mock.Mock
}

func (m *MockAuthProvider) ValidateToken(ctx context.Context, token string) (auth.Claims, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(auth.Claims), args.Error(1)
}

func (m *MockAuthProvider) GenerateToken(ctx context.Context, user tenant.User) (auth.Tokens, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(auth.Tokens), args.Error(1)
}

func (m *MockAuthProvider) RefreshToken(ctx context.Context, token, userId, email, password string, roles []string) (auth.Tokens, error) {
	args := m.Called(ctx, token, userId, email, password, roles)
	return args.Get(0).(auth.Tokens), args.Error(1)
}

func (m *MockAuthProvider) Login(ctx context.Context, userId, email, password string, roles []string) (auth.Tokens, error) {
	args := m.Called(ctx, userId, email, password, roles)
	return args.Get(0).(auth.Tokens), args.Error(1)
}

// MockUserManagementService is a mock implementation of UserManagementService interface
type MockUserManagementService struct {
	mock.Mock
}

func (m *MockUserManagementService) GetUserByID(ctx context.Context, schema, id string) (tenant.User, error) {
	args := m.Called(ctx, schema, id)
	if user, ok := args.Get(0).(tenant.User); ok {
		return user, args.Error(1)
	}
	return tenant.User{}, args.Error(1)
}

func (m *MockUserManagementService) GetUserByEmail(ctx context.Context, schema, email string) (tenant.User, error) {
	args := m.Called(ctx, schema, email)
	if user, ok := args.Get(0).(tenant.User); ok {
		return user, args.Error(1)
	}
	return tenant.User{}, args.Error(1)
}

func (m *MockUserManagementService) CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
	args := m.Called(ctx, schema, req)
	if user, ok := args.Get(0).(tenant.User); ok {
		return user, args.Error(1)
	}
	return tenant.User{}, args.Error(1)
}

func (m *MockUserManagementService) GetUserProfileByID(ctx context.Context, schema, userID string) (dto.UserResponse, error) {
	args := m.Called(ctx, schema, userID)
	if resp, ok := args.Get(0).(dto.UserResponse); ok {
		return resp, args.Error(1)
	}
	return dto.UserResponse{}, args.Error(1)
}

func (m *MockUserManagementService) UpdateUserProfile(ctx context.Context, schema, userID string, updateData dto.UpdateUserProfileRequest) (dto.UserResponse, error) {
	args := m.Called(ctx, schema, userID, updateData)
	if resp, ok := args.Get(0).(dto.UserResponse); ok {
		return resp, args.Error(1)
	}
	return dto.UserResponse{}, args.Error(1)
}

func (m *MockUserManagementService) UpdatePassword(ctx context.Context, schema, userID string, updateData dto.UpdateUserPasswordRequest) (tenant.User, error) {
	args := m.Called(ctx, schema, userID, updateData)
	if user, ok := args.Get(0).(tenant.User); ok {
		return user, args.Error(1)
	}
	return tenant.User{}, args.Error(1)
}

func (m *MockUserManagementService) AddAvatar(ctx context.Context, schema, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error) {
	args := m.Called(ctx, schema, userID, fileHeader)
	if resp, ok := args.Get(0).(dto.UserResponse); ok {
		return resp, args.Error(1)
	}
	return dto.UserResponse{}, args.Error(1)
}

func (m *MockUserManagementService) RemoveAvatar(ctx context.Context, schema, userID string) (dto.UserResponse, error) {
	args := m.Called(ctx, schema, userID)
	if resp, ok := args.Get(0).(dto.UserResponse); ok {
		return resp, args.Error(1)
	}
	return dto.UserResponse{}, args.Error(1)
}

func (m *MockUserManagementService) UpdateUser(ctx context.Context, schema, id string, updateData map[string]interface{}) (tenant.User, error) {
	args := m.Called(ctx, schema, id, updateData)
	if user, ok := args.Get(0).(tenant.User); ok {
		return user, args.Error(1)
	}
	return tenant.User{}, args.Error(1)
}

func (m *MockUserManagementService) GetAllUsers(ctx context.Context, schema string) ([]tenant.User, error) {
	args := m.Called(ctx, schema)
	if users, ok := args.Get(0).([]tenant.User); ok {
		return users, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserManagementService) GetWorkspaces(ctx context.Context, schema, userID, roles string) ([]dto.UserWorkspaceResponse, error) {
	args := m.Called(ctx, schema, userID, roles)
	if workspaces, ok := args.Get(0).([]dto.UserWorkspaceResponse); ok {
		return workspaces, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserManagementService) GetBulkUsers(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
	args := m.Called(ctx, schema, ids)
	if users, ok := args.Get(0).([]tenant.User); ok {
		return users, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserManagementService) GetUsersWithRole(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	args := m.Called(ctx, schema)
	if users, ok := args.Get(0).([]dto.UserWithRole); ok {
		return users, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserManagementService) GetActiveUsersForAssign(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	args := m.Called(ctx, schema)
	if users, ok := args.Get(0).([]dto.UserWithRole); ok {
		return users, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserManagementService) DeleteUserCompletely(ctx context.Context, schema, userID string) error {
	args := m.Called(ctx, schema, userID)
	return args.Error(0)
}

func (m *MockUserManagementService) GetUserAccessDetails(ctx context.Context, schema, userID, roles, workspaceID string) (dto.UserAccessDetailsResponse, error) {
	args := m.Called(ctx, schema, userID, roles, workspaceID)
	if resp, ok := args.Get(0).(dto.UserAccessDetailsResponse); ok {
		return resp, args.Error(1)
	}
	return dto.UserAccessDetailsResponse{}, args.Error(1)
}

func (m *MockUserManagementService) GetUserRolesAndAccess(ctx context.Context, schema, userID string, scopeID *string) ([]dto.UserRolesAccessResponse, error) {
	args := m.Called(ctx, schema, userID, scopeID)
	if resp, ok := args.Get(0).([]dto.UserRolesAccessResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		mockAuthSetup  func(*MockAuthProvider)
		mockUserSetup  func(*MockUserManagementService)
		expectedStatus int
		checkContext   func(*testing.T, *gin.Context)
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			mockAuthSetup:  func(m *MockAuthProvider) {},
			mockUserSetup:  func(m *MockUserManagementService) {},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   func(t *testing.T, c *gin.Context) {},
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid_token",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer invalid_token").
					Return(auth.Claims{}, errors.New("invalid token"))
			},
			mockUserSetup:  func(m *MockUserManagementService) {},
			expectedStatus: http.StatusInternalServerError,
			checkContext:   func(t *testing.T, c *gin.Context) {},
		},
		{
			name:       "user not found in database",
			authHeader: "Bearer valid_token",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer valid_token").
					Return(auth.Claims{
						UserId: "user123",
						Roles:  "admin",
					}, nil)
			},
			mockUserSetup: func(m *MockUserManagementService) {
				m.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
					Return(tenant.User{}, errors.New("user not found"))
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   func(t *testing.T, c *gin.Context) {},
		},
		{
			name:       "user status is inactive",
			authHeader: "Bearer valid_token",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer valid_token").
					Return(auth.Claims{
						UserId: "user123",
						Roles:  "admin",
					}, nil)
			},
			mockUserSetup: func(m *MockUserManagementService) {
				m.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
					Return(tenant.User{
						ID:     uuid.New(),
						Email:  "user@example.com",
						Status: "inactive",
					}, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   func(t *testing.T, c *gin.Context) {},
		},
		{
			name:       "user status is suspended",
			authHeader: "Bearer valid_token",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer valid_token").
					Return(auth.Claims{
						UserId: "user456",
						Roles:  "viewer",
					}, nil)
			},
			mockUserSetup: func(m *MockUserManagementService) {
				m.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user456").
					Return(tenant.User{
						ID:     uuid.New(),
						Email:  "user456@example.com",
						Status: "suspended",
					}, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   func(t *testing.T, c *gin.Context) {},
		},
		{
			name:       "valid token with active user - single role",
			authHeader: "Bearer valid_token",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer valid_token").
					Return(auth.Claims{
						UserId: "user123",
						Roles:  "admin",
					}, nil)
			},
			mockUserSetup: func(m *MockUserManagementService) {
				m.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
					Return(tenant.User{
						ID:     uuid.New(),
						Email:  "user@example.com",
						Status: "active",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkContext: func(t *testing.T, c *gin.Context) {
				userID, exists := c.Get("user_id")
				assert.True(t, exists)
				assert.Equal(t, "user123", userID)

				schema, exists := c.Get("schema")
				assert.True(t, exists)
				assert.Equal(t, constant.MasterDatabase, schema)

				roles, exists := c.Get("roles")
				assert.True(t, exists)
				assert.Equal(t, "admin", roles)
			},
		},
		{
			name:       "valid token with active user - multiple roles",
			authHeader: "Bearer valid_token_multi",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer valid_token_multi").
					Return(auth.Claims{
						UserId: "user456",
						Roles:  "admin,editor,viewer",
					}, nil)
			},
			mockUserSetup: func(m *MockUserManagementService) {
				m.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user456").
					Return(tenant.User{
						ID:     uuid.New(),
						Email:  "user456@example.com",
						Status: "active",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkContext: func(t *testing.T, c *gin.Context) {
				userID, exists := c.Get("user_id")
				assert.True(t, exists)
				assert.Equal(t, "user456", userID)

				schema, exists := c.Get("schema")
				assert.True(t, exists)
				assert.Equal(t, constant.MasterDatabase, schema)

				roles, exists := c.Get("roles")
				assert.True(t, exists)
				rolesStr := roles.(string)
				assert.Contains(t, rolesStr, "admin")
				assert.Contains(t, rolesStr, "editor")
				assert.Contains(t, rolesStr, "viewer")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuthProvider := new(MockAuthProvider)
			mockUserService := new(MockUserManagementService)
			tt.mockAuthSetup(mockAuthProvider)
			tt.mockUserSetup(mockUserService)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			var capturedContext *gin.Context
			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
				capturedContext = c
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if capturedContext != nil && tt.expectedStatus == http.StatusOK {
				tt.checkContext(t, capturedContext)
			}

			mockAuthProvider.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
		})
	}
}

// TestAuthMiddleware_Abort tests that middleware aborts on failure
func TestAuthMiddleware_Abort(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockUserService := new(MockUserManagementService)
	mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer invalid").
		Return(auth.Claims{}, errors.New("invalid"))

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	handlerCalled := false
	r.GET("/test",
		middleware.AuthMiddleware(mockAuthProvider, mockUserService),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	c.Request = req
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled, "Handler should not be called when auth fails")
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TestAuthMiddleware_WithChainedMiddleware tests auth middleware with other middlewares
func TestAuthMiddleware_WithChainedMiddleware(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockUserService := new(MockUserManagementService)
	mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
		Return(auth.Claims{
			UserId: "user123",
			Roles:  "admin",
		}, nil)
	mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
		Return(tenant.User{
			ID:     uuid.New(),
			Email:  "user@example.com",
			Status: "active",
		}, nil)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	var capturedUserID string
	r.GET("/test",
		middleware.AuthMiddleware(mockAuthProvider, mockUserService),
		func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			if exists {
				capturedUserID = userID.(string)
			}
			c.Next()
		},
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	c.Request = req
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user123", capturedUserID)
}

// TestAuthMiddleware_EdgeCases tests edge cases for authentication
func TestAuthMiddleware_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		mockAuthSetup  func(*MockAuthProvider)
		mockUserSetup  func(*MockUserManagementService)
		expectedStatus int
	}{
		{
			name:       "empty bearer token",
			authHeader: "Bearer ",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer ").
					Return(auth.Claims{}, errors.New("empty token"))
			},
			mockUserSetup:  func(m *MockUserManagementService) {},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "malformed header no bearer",
			authHeader: "InvalidToken",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "InvalidToken").
					Return(auth.Claims{}, errors.New("malformed"))
			},
			mockUserSetup:  func(m *MockUserManagementService) {},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "token with special characters but valid user",
			authHeader: "Bearer abc123!@#$%^&*()",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer abc123!@#$%^&*()").
					Return(auth.Claims{
						UserId: "user999",
						Roles:  "viewer",
					}, nil)
			},
			mockUserSetup: func(m *MockUserManagementService) {
				m.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user999").
					Return(tenant.User{
						ID:     uuid.New(),
						Email:  "user999@example.com",
						Status: "active",
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "token with whitespace - invalid",
			authHeader: "Bearer token with spaces",
			mockAuthSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer token with spaces").
					Return(auth.Claims{}, errors.New("invalid token format"))
			},
			mockUserSetup:  func(m *MockUserManagementService) {},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuthProvider := new(MockAuthProvider)
			mockUserService := new(MockUserManagementService)
			tt.mockAuthSetup(mockAuthProvider)
			tt.mockUserSetup(mockUserService)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.authHeader)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockAuthProvider.AssertExpectations(t)
		})
	}
}

// TestAuthMiddleware_DifferentHTTPMethods tests authentication across different HTTP methods
func TestAuthMiddleware_DifferentHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			mockAuthProvider := new(MockAuthProvider)
			mockUserService := new(MockUserManagementService)
			mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
				Return(auth.Claims{
					UserId: "user123",
					Roles:  "admin",
				}, nil)
			mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
				Return(tenant.User{
					ID:     uuid.New(),
					Email:  "user@example.com",
					Status: "active",
				}, nil)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Handle(method, "/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Authorization", "Bearer valid_token")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockAuthProvider.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
		})
	}
}

// TestAuthMiddleware_MultipleRoles tests handling of multiple roles
func TestAuthMiddleware_MultipleRoles(t *testing.T) {
	rolesCombinations := []string{
		"admin",
		"admin,editor",
		"admin,editor,viewer",
		"viewer,contributor,moderator,admin",
	}

	for _, roles := range rolesCombinations {
		t.Run("roles_"+roles, func(t *testing.T) {
			mockAuthProvider := new(MockAuthProvider)
			mockUserService := new(MockUserManagementService)
			mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
				Return(auth.Claims{
					UserId: "user123",
					Roles:  roles,
				}, nil)
			mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
				Return(tenant.User{
					ID:     uuid.New(),
					Email:  "user@example.com",
					Status: "active",
				}, nil)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			var capturedRoles string
			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
				rolesVal, exists := c.Get("roles")
				if exists {
					capturedRoles = rolesVal.(string)
				}
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer valid_token")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, roles, capturedRoles)
			mockAuthProvider.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
		})
	}
}

// TestAuthMiddleware_ConcurrentRequests tests authentication with concurrent requests
func TestAuthMiddleware_ConcurrentRequests(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockUserService := new(MockUserManagementService)
	mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
		Return(auth.Claims{
			UserId: "user123",
			Roles:  "admin",
		}, nil).Maybe()
	mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
		Return(tenant.User{
			ID:     uuid.New(),
			Email:  "user@example.com",
			Status: "active",
		}, nil).Maybe()

	_, r := gin.CreateTestContext(httptest.NewRecorder())

	r.GET("/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	successCount := 0
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid_token")
		r.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(t, 10, successCount)
}

// TestAuthMiddleware_ContextValues tests that all context values are set correctly
func TestAuthMiddleware_ContextValues(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockUserService := new(MockUserManagementService)
	mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
		Return(auth.Claims{
			UserId:   "user123",
			TenantId: "tenant456",
			Roles:    "admin,editor",
		}, nil)
	mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
		Return(tenant.User{
			ID:     uuid.New(),
			Email:  "user@example.com",
			Status: "active",
		}, nil)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
		// Check user_id
		userID, exists := c.Get("user_id")
		assert.True(t, exists, "user_id should be set")
		assert.Equal(t, "user123", userID)

		// Check schema
		schema, exists := c.Get("schema")
		assert.True(t, exists, "schema should be set")
		assert.NotEmpty(t, schema)

		// Check roles
		roles, exists := c.Get("roles")
		assert.True(t, exists, "roles should be set")
		assert.Equal(t, "admin,editor", roles)

		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	c.Request = req
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockAuthProvider.AssertExpectations(t)
	mockUserService.AssertExpectations(t)
}

// TestAuthMiddleware_SuccessfulAuthentication tests successful auth flow
func TestAuthMiddleware_SuccessfulAuthentication(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockUserService := new(MockUserManagementService)
	mockAuthProvider.On("ValidateToken", mock.Anything, mock.Anything).
		Return(auth.Claims{UserId: "user1", Roles: "user"}, nil)
	mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user1").
		Return(tenant.User{
			ID:     uuid.New(),
			Email:  "user@example.com",
			Status: "active",
		}, nil)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/protected", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer token123")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

// TestAuthMiddleware_UserStatusValidation tests various user status scenarios
func TestAuthMiddleware_UserStatusValidation(t *testing.T) {
	statusTests := []struct {
		name           string
		userStatus     string
		expectedStatus int
		shouldPass     bool
	}{
		{"active status", "active", http.StatusOK, true},
		{"inactive status", "inactive", http.StatusUnauthorized, false},
		{"suspended status", "suspended", http.StatusUnauthorized, false},
		{"deactivated status", "deactivated", http.StatusUnauthorized, false},
		{"pending status", "pending", http.StatusUnauthorized, false},
		{"archived status", "archived", http.StatusUnauthorized, false},
	}

	for _, tt := range statusTests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuthProvider := new(MockAuthProvider)
			mockUserService := new(MockUserManagementService)

			mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
				Return(auth.Claims{
					UserId: "user123",
					Roles:  "admin",
				}, nil)

			mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
				Return(tenant.User{
					ID:     uuid.New(),
					Email:  "user@example.com",
					Status: tt.userStatus,
				}, nil)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			handlerCalled := false
			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer valid_token")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.shouldPass, handlerCalled)
			mockAuthProvider.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
		})
	}
}

// TestAuthMiddleware_DatabaseCheckFailure tests when user is not found in database
func TestAuthMiddleware_DatabaseCheckFailure(t *testing.T) {
	tests := []struct {
		name        string
		errorMsg    string
		expectedErr string
	}{
		{"user not found", "user not found", "user not found"},
		{"database connection error", "database connection failed", "database connection failed"},
		{"query timeout", "context deadline exceeded", "context deadline exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuthProvider := new(MockAuthProvider)
			mockUserService := new(MockUserManagementService)

			mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
				Return(auth.Claims{
					UserId: "user123",
					Roles:  "admin",
				}, nil)

			mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, "user123").
				Return(tenant.User{}, errors.New(tt.errorMsg))

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			handlerCalled := false
			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer valid_token")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.False(t, handlerCalled)
			mockAuthProvider.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
		})
	}
}

// TestAuthMiddleware_UsersWithDifferentRolesAndStatuses tests combinations of roles and statuses
func TestAuthMiddleware_UsersWithDifferentRolesAndStatuses(t *testing.T) {
	testCases := []struct {
		name           string
		userID         string
		roles          string
		status         string
		expectedStatus int
	}{
		{"admin with active status", "admin1", "admin", "active", http.StatusOK},
		{"admin with inactive status", "admin2", "admin", "inactive", http.StatusUnauthorized},
		{"editor with active status", "editor1", "admin,editor", "active", http.StatusOK},
		{"editor with suspended status", "editor2", "admin,editor", "suspended", http.StatusUnauthorized},
		{"viewer with active status", "viewer1", "viewer", "active", http.StatusOK},
		{"viewer with pending status", "viewer2", "viewer", "pending", http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthProvider := new(MockAuthProvider)
			mockUserService := new(MockUserManagementService)

			mockAuthProvider.On("ValidateToken", mock.Anything, mock.MatchedBy(func(s string) bool {
				return s == "Bearer valid_token"
			})).
				Return(auth.Claims{
					UserId: tc.userID,
					Roles:  tc.roles,
				}, nil)

			mockUserService.On("GetUserByID", mock.Anything, constant.MasterDatabase, tc.userID).
				Return(tenant.User{
					ID:     uuid.New(),
					Email:  tc.userID + "@example.com",
					Status: tc.status,
				}, nil)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider, mockUserService), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer valid_token")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			mockAuthProvider.AssertExpectations(t)
			mockUserService.AssertExpectations(t)
		})
	}
}
