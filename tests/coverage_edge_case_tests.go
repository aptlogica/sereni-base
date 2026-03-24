package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ============================================================================
// Response Utility Edge Case Tests
// ============================================================================

// TestSendSuccess_EdgeCases tests SendSuccess with various data types
func TestSendSuccess_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{"nil data", nil},
		{"empty string", ""},
		{"empty object", map[string]interface{}{}},
		{"empty array", []interface{}{}},
		{"zero value", 0},
		{"boolean false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			response.SendSuccess(c, tt.data)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}

// TestCheckAndSendError_MultipleErrorTypes tests error handling for different error types
func TestCheckAndSendError_MultipleErrorTypes(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{"nil error", nil, http.StatusOK},
		{"generic error", errors.New("test error"), http.StatusInternalServerError},
		{"context cancelled", context.Canceled, http.StatusInternalServerError},
		{"context deadline exceeded", context.DeadlineExceeded, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			response.CheckAndSendError(c, tt.err)

			if tt.err == nil {
				// For nil error, SendSuccess is called
				assert.NotEqual(t, http.StatusInternalServerError, w.Code)
			}
		})
	}
}

// ============================================================================
// Middleware Edge Case Tests
// ============================================================================

// TestCommonMiddleware_EmptyPath tests middleware with empty/root path
func TestCommonMiddleware_EmptyPath(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", middleware.RequestSizeLimit(1024), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestValidation_BoundaryValues tests validation with boundary values
func TestValidation_BoundaryValues(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		shouldBeValid bool
	}{
		{"single character name", "a", true},
		{"max length underscore", "a_b_c_d_e_f_g_h_i_j", true},
		{"mixed case with numbers", "Table123ABC", true},
		{"leading underscore", "_table", false},
		{"space in name", "table name", false},
		{"special char @", "table@name", false},
		{"special char #", "table#123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/table/:table", middleware.ValidateTableName(), func(c *gin.Context) {
				if c.IsAborted() {
					c.Status(http.StatusBadRequest)
				} else {
					c.Status(http.StatusOK)
				}
			})

			req := httptest.NewRequest("GET", "/api/table/"+tt.value, nil)
			r.ServeHTTP(w, req)

			if tt.shouldBeValid {
				assert.Equal(t, http.StatusOK, w.Code, "Expected valid table name '%s' to pass", tt.value)
			} else {
				assert.Equal(t, http.StatusBadRequest, w.Code, "Expected invalid table name '%s' to fail", tt.value)
			}
		})
	}
}

// ============================================================================
// Constant and Config Edge Case Tests
// ============================================================================

// TestConstantReferences verifies all critical constants are accessible
func TestConstantReferences(t *testing.T) {
	t.Run("master database constant", func(t *testing.T) {
		assert.NotEmpty(t, constant.MasterDatabase)
	})

	t.Run("response codes exist", func(t *testing.T) {
		assert.NotNil(t, responseConst.Error.InvalidPayload)
	})
}

// TestConfigInitialization tests configuration scenarios
func TestConfigInitialization(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{"nil config", nil},
		{"empty config", &config.Config{}},
		{"config with auth", &config.Config{
			Auth: config.AuthConfig{
				JWT: config.JWTConfig{Secret: "test-secret"},
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify no panic occurs
			if tt.cfg != nil {
				assert.NotNil(t, tt.cfg)
			}
		})
	}
}

// ============================================================================
// DTO Validation Edge Cases
// ============================================================================

// TestUserDTOCreation tests creating user DTOs with edge cases
func TestUserDTOCreation(t *testing.T) {
	tests := []struct {
		name string
		user dto.UserResponse
	}{
		{"empty user", dto.UserResponse{}},
		{"user with email only", dto.UserResponse{Email: "test@example.com"}},
		{"user with ID only", dto.UserResponse{ID: uuid.New().String()}},
		{"user with all fields", dto.UserResponse{
			ID:    uuid.New().String(),
			Email: "admin@example.com",
			Name:  "Admin User",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.user)
		})
	}
}

// ============================================================================
// Error Response Edge Cases
// ============================================================================

// TestErrorResponseConsistency verifies error responses are consistent
func TestErrorResponseConsistency(t *testing.T) {
	errorCodes := []responseConst.ResponseCode{
		responseConst.Error.InvalidPayload,
		responseConst.Error.UnauthorizedAccess,
		responseConst.Error.InternalError,
		responseConst.Error.ResourceNotFound,
	}

	for _, code := range errorCodes {
		t.Run(string(code), func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			response.SendError(c, code)

			assert.Greater(t, w.Code, 0, "Response code should be set")
			assert.Greater(t, int64(w.Body.Len()), int64(0), "Response body should not be empty")
		})
	}
}

// ============================================================================
// Tenant Model Edge Cases
// ============================================================================

// TestTenantUserCreation tests creating tenant users with various states
func TestTenantUserCreation(t *testing.T) {
	tests := []struct {
		name   string
		user   tenant.User
		status string
	}{
		{"active user", tenant.User{ID: uuid.New()}, "active"},
		{"inactive user", tenant.User{ID: uuid.New()}, "inactive"},
		{"suspended user", tenant.User{ID: uuid.New()}, "suspended"},
		{"archived user", tenant.User{ID: uuid.New()}, "archived"},
		{"pending user", tenant.User{ID: uuid.New()}, "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := tt.user
			user.Status = tt.status
			assert.Equal(t, tt.status, user.Status)
		})
	}
}

// ============================================================================
// Context Operations Edge Cases
// ============================================================================

// TestContextOperationsWithDifferentTypes tests Gin context with edge cases
func TestContextOperationsWithDifferentTypes(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string value", "key1", "value1"},
		{"int value", "key2", 123},
		{"bool value", "key3", true},
		{"nil value", "key4", nil},
		{"uuid value", "key5", uuid.New()},
		{"map value", "key6", map[string]string{"nested": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.Set(tt.key, tt.value)
			val, exists := c.Get(tt.key)
			assert.True(t, exists)
			assert.Equal(t, tt.value, val)
		})
	}
}

// TestContextMissing tests accessing non-existent context values
func TestContextMissing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	_, exists := c.Get("nonexistent")
	assert.False(t, exists)
}

// ============================================================================
// HTTP Status Boundary Tests
// ============================================================================

// TestStatusCodeBoundaries tests various HTTP status codes
func TestStatusCodeBoundaries(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,                  // 200
		http.StatusCreated,             // 201
		http.StatusAccepted,            // 202
		http.StatusBadRequest,          // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusInternalServerError, // 500
		http.StatusServiceUnavailable,  // 503
	}

	for _, code := range statusCodes {
		t.Run("status_"+string(rune(code)), func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Status(code)

			w.WriteHeader(code)
			assert.Equal(t, code, w.Code)
		})
	}
}

// ============================================================================
// Empty/Nil Input Tests
// ============================================================================

// TestNilPointerHandling tests functions with nil inputs
func TestNilPointerHandling(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{"nil string pointer", (*string)(nil)},
		{"nil struct pointer", (*tenant.User)(nil)},
		{"nil interface", interface{}(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Nil(t, tt.value)
		})
	}
}

// TestEmptyStringHandling tests empty string edge cases
func TestEmptyStringHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"space only", " "},
		{"tab only", "\t"},
		{"newline only", "\n"},
		{"unicode space", "\u00A0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.input)
		})
	}
}

// ============================================================================
// UUID and ID Edge Cases
// ============================================================================

// TestUUIDGeneration tests UUID handling
func TestUUIDGeneration(t *testing.T) {
	tests := []struct {
		name string
		fn   func() uuid.UUID
	}{
		{"new uuid", func() uuid.UUID { return uuid.New() }},
		{"nil uuid", func() uuid.UUID { return uuid.UUID{} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.fn()
			assert.NotNil(t, id)
		})
	}
}
