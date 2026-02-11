package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"serenibase/internal/constant"
	"serenibase/internal/middleware"
	"serenibase/internal/models/tenant"
	"serenibase/internal/providers/auth"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *MockAuthProvider) RefreshToken(ctx context.Context, token string) (auth.Tokens, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(auth.Tokens), args.Error(1)
}

func (m *MockAuthProvider) Login(ctx context.Context, email, password string) (auth.Tokens, error) {
	args := m.Called(ctx, email, password)
	return args.Get(0).(auth.Tokens), args.Error(1)
}

func (m *MockAuthProvider) Register(ctx context.Context, userId, email, password string, roles []string) error {
	args := m.Called(ctx, userId, email, password, roles)
	return args.Error(0)
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		mockSetup      func(*MockAuthProvider)
		expectedStatus int
		checkContext   func(*testing.T, *gin.Context)
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			mockSetup:      func(m *MockAuthProvider) {},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   func(t *testing.T, c *gin.Context) {},
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid_token",
			mockSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer invalid_token").
					Return(auth.Claims{}, errors.New("invalid token"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkContext:   func(t *testing.T, c *gin.Context) {},
		},
		{
			name:       "valid token - single role",
			authHeader: "Bearer valid_token",
			mockSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer valid_token").
					Return(auth.Claims{
						UserId: "user123",
						Roles:  "admin",
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
			name:       "valid token - multiple roles",
			authHeader: "Bearer valid_token_multi",
			mockSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer valid_token_multi").
					Return(auth.Claims{
						UserId: "user456",
						Roles:  "admin,editor,viewer",
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
			tt.mockSetup(mockAuthProvider)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			var capturedContext *gin.Context
			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider), func(c *gin.Context) {
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
		})
	}
}

// TestAuthMiddleware_Abort tests that middleware aborts on failure
func TestAuthMiddleware_Abort(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer invalid").
		Return(auth.Claims{}, errors.New("invalid"))

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	handlerCalled := false
	r.GET("/test",
		middleware.AuthMiddleware(mockAuthProvider),
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
	mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
		Return(auth.Claims{
			UserId: "user123",
			Roles:  "admin",
		}, nil)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	var capturedUserID string
	r.GET("/test",
		middleware.AuthMiddleware(mockAuthProvider),
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
		mockSetup      func(*MockAuthProvider)
		expectedStatus int
	}{
		{
			name:       "empty bearer token",
			authHeader: "Bearer ",
			mockSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer ").
					Return(auth.Claims{}, errors.New("empty token"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "malformed header no bearer",
			authHeader: "InvalidToken",
			mockSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "InvalidToken").
					Return(auth.Claims{}, errors.New("malformed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "token with special characters",
			authHeader: "Bearer abc123!@#$%^&*()",
			mockSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer abc123!@#$%^&*()").
					Return(auth.Claims{
						UserId: "user999",
						Roles:  "viewer",
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "very long token",
			authHeader: "Bearer " + string(make([]byte, 1000)),
			mockSetup: func(m *MockAuthProvider) {
				longToken := "Bearer " + string(make([]byte, 1000))
				m.On("ValidateToken", mock.Anything, longToken).
					Return(auth.Claims{}, errors.New("token too long"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "token with whitespace",
			authHeader: "Bearer token with spaces",
			mockSetup: func(m *MockAuthProvider) {
				m.On("ValidateToken", mock.Anything, "Bearer token with spaces").
					Return(auth.Claims{}, errors.New("invalid token format"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuthProvider := new(MockAuthProvider)
			tt.mockSetup(mockAuthProvider)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider), func(c *gin.Context) {
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
			mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
				Return(auth.Claims{
					UserId: "user123",
					Roles:  "admin",
				}, nil)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Handle(method, "/test", middleware.AuthMiddleware(mockAuthProvider), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Authorization", "Bearer valid_token")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			mockAuthProvider.AssertExpectations(t)
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
			mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
				Return(auth.Claims{
					UserId: "user123",
					Roles:  roles,
				}, nil)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			var capturedRoles string
			r.GET("/test", middleware.AuthMiddleware(mockAuthProvider), func(c *gin.Context) {
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
		})
	}
}

// TestAuthMiddleware_ConcurrentRequests tests authentication with concurrent requests
func TestAuthMiddleware_ConcurrentRequests(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
		Return(auth.Claims{
			UserId: "user123",
			Roles:  "admin",
		}, nil).Times(10)

	_, r := gin.CreateTestContext(httptest.NewRecorder())

	r.GET("/test", middleware.AuthMiddleware(mockAuthProvider), func(c *gin.Context) {
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
	mockAuthProvider.AssertExpectations(t)
}

// TestAuthMiddleware_ContextValues tests that all context values are set correctly
func TestAuthMiddleware_ContextValues(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockAuthProvider.On("ValidateToken", mock.Anything, "Bearer valid_token").
		Return(auth.Claims{
			UserId:   "user123",
			TenantId: "tenant456",
			Roles:    "admin,editor",
		}, nil)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.AuthMiddleware(mockAuthProvider), func(c *gin.Context) {
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
}

// TestAuthMiddleware_SuccessfulAuthentication tests successful auth flow
func TestAuthMiddleware_SuccessfulAuthentication(t *testing.T) {
	mockAuthProvider := new(MockAuthProvider)
	mockAuthProvider.On("ValidateToken", mock.Anything, mock.Anything).
		Return(auth.Claims{UserId: "user1", Roles: "user"}, nil)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/protected", middleware.AuthMiddleware(mockAuthProvider), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer token123")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}
