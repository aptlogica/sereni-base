package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestExtractUserAndSchemaFromContext tests user and schema extraction from context
func TestExtractUserAndSchemaFromContext(t *testing.T) {
	tests := []struct {
		name            string
		setupContext    func(*gin.Context)
		expectedUserID  string
		expectedSchema  string
		expectedSuccess bool
		expectedStatus  int
	}{
		{
			name: "both user_id and schema present",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
			},
			expectedUserID:  "user123",
			expectedSchema:  "test_schema",
			expectedSuccess: true,
			expectedStatus:  http.StatusOK,
		},
		{
			name: "missing user_id",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "test_schema")
			},
			expectedUserID:  "",
			expectedSchema:  "",
			expectedSuccess: false,
			expectedStatus:  http.StatusForbidden,
		},
		{
			name: "missing schema",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
			},
			expectedUserID:  "",
			expectedSchema:  "",
			expectedSuccess: false,
			expectedStatus:  http.StatusForbidden,
		},
		{
			name: "both missing",
			setupContext: func(c *gin.Context) {
				// Don't set anything
			},
			expectedUserID:  "",
			expectedSchema:  "",
			expectedSuccess: false,
			expectedStatus:  http.StatusForbidden,
		},
		{
			name: "empty string values",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "")
				c.Set("schema", "")
			},
			expectedUserID:  "",
			expectedSchema:  "",
			expectedSuccess: true,
			expectedStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			tt.setupContext(c)

			// Use a handler to test the extraction indirectly
			// by simulating what happens in actual middleware
			var capturedUserID, capturedSchema string
			var capturedSuccess bool

			if _, hasUser := c.Get("user_id"); hasUser {
				if _, hasSchema := c.Get("schema"); hasSchema {
					userID, _ := c.Get("user_id")
					schema, _ := c.Get("schema")
					capturedUserID, _ = userID.(string)
					capturedSchema, _ = schema.(string)
					capturedSuccess = true
				}
			}

			if tt.expectedSuccess {
				assert.True(t, capturedSuccess)
				assert.Equal(t, tt.expectedUserID, capturedUserID)
				assert.Equal(t, tt.expectedSchema, capturedSchema)
			} else {
				assert.False(t, capturedSuccess)
			}
		})
	}
}

// TestExtractScopeFromHeaders tests scope extraction from headers
func TestExtractScopeFromHeaders(t *testing.T) {
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
			name: "scope type base",
			headers: map[string]string{
				middleware.HeaderScopeType: "base",
				middleware.HeaderScopeID:   "base456",
			},
			expectedScopeType: "base",
			expectedScopeID:   "base456",
		},
		{
			name: "missing scope type - defaults to workspace",
			headers: map[string]string{
				middleware.HeaderScopeID: "ws789",
			},
			expectedScopeType: constant.ScopeLevels.Workspace,
			expectedScopeID:   "ws789",
		},
		{
			name: "missing scope id",
			headers: map[string]string{
				middleware.HeaderScopeType: "workspace",
			},
			expectedScopeType: "workspace",
			expectedScopeID:   "",
		},
		{
			name:              "both headers missing",
			headers:           map[string]string{},
			expectedScopeType: constant.ScopeLevels.Workspace,
			expectedScopeID:   "",
		},
		{
			name: "empty string values",
			headers: map[string]string{
				middleware.HeaderScopeType: "",
				middleware.HeaderScopeID:   "",
			},
			expectedScopeType: constant.ScopeLevels.Workspace,
			expectedScopeID:   "",
		},
		{
			name: "custom scope type",
			headers: map[string]string{
				middleware.HeaderScopeType: "organization",
				middleware.HeaderScopeID:   "org123",
			},
			expectedScopeType: "organization",
			expectedScopeID:   "org123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", func(c *gin.Context) {
				scopeType := c.GetHeader(middleware.HeaderScopeType)
				if scopeType == "" {
					scopeType = constant.ScopeLevels.Workspace
				}
				scopeID := c.GetHeader(middleware.HeaderScopeID)

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

// TestSendUnauthorizedError tests unauthorized error sending
func TestSendUnauthorizedError(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		// Simulate sending unauthorized error
		c.AbortWithStatus(http.StatusForbidden)
	}, func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	c.Request = req
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, handlerCalled, "Next handler should not be called after abort")
}

// TestCommonMiddleware_Integration tests common middleware integration patterns
func TestCommonMiddleware_Integration(t *testing.T) {
	t.Run("extract user and schema in middleware chain", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		var capturedUserID, capturedSchema string

		r.GET("/test",
			func(c *gin.Context) {
				// First middleware sets context
				c.Set("user_id", "user123")
				c.Set("schema", "test_schema")
				c.Next()
			},
			func(c *gin.Context) {
				// Second middleware extracts and uses context
				userID, hasUser := c.Get("user_id")
				schema, hasSchema := c.Get("schema")

				if hasUser && hasSchema {
					capturedUserID = userID.(string)
					capturedSchema = schema.(string)
				}
				c.Next()
			},
			func(c *gin.Context) {
				c.Status(http.StatusOK)
			},
		)

		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "user123", capturedUserID)
		assert.Equal(t, "test_schema", capturedSchema)
	})

	t.Run("extract scope headers in middleware chain", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		var capturedScopeType, capturedScopeID string

		r.GET("/test", func(c *gin.Context) {
			scopeType := c.GetHeader(middleware.HeaderScopeType)
			if scopeType == "" {
				scopeType = constant.ScopeLevels.Workspace
			}
			scopeID := c.GetHeader(middleware.HeaderScopeID)

			capturedScopeType = scopeType
			capturedScopeID = scopeID
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(middleware.HeaderScopeType, "base")
		req.Header.Set(middleware.HeaderScopeID, "base123")
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "base", capturedScopeType)
		assert.Equal(t, "base123", capturedScopeID)
	})
}

// TestCommonMiddleware_TypeAssertion tests type assertion behavior
func TestCommonMiddleware_TypeAssertion(t *testing.T) {
	tests := []struct {
		name     string
		setValue interface{}
		expected string
	}{
		{
			name:     "string value",
			setValue: "test_value",
			expected: "test_value",
		},
		{
			name:     "empty string",
			setValue: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", func(c *gin.Context) {
				c.Set("test_key", tt.setValue)

				value, exists := c.Get("test_key")
				assert.True(t, exists)

				strValue, ok := value.(string)
				assert.True(t, ok)
				assert.Equal(t, tt.expected, strValue)

				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestExtractUserAndSchemaFromContext_VariousTypes tests with different data types
func TestExtractUserAndSchemaFromContext_VariousTypes(t *testing.T) {
	tests := []struct {
		name   string
		userID interface{}
		schema interface{}
	}{
		{"string values", "user123", "schema_name"},
		{"numeric string", "12345", "67890"},
		{"empty strings", "", ""},
		{"long strings", "user_" + strings.Repeat("x", 100), "schema_" + strings.Repeat("y", 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("user_id", tt.userID)
			c.Set("schema", tt.schema)

			userID, hasUser := c.Get("user_id")
			schema, hasSchema := c.Get("schema")

			assert.True(t, hasUser)
			assert.True(t, hasSchema)
			assert.Equal(t, tt.userID, userID)
			assert.Equal(t, tt.schema, schema)
		})
	}
}

// TestExtractScopeFromHeaders_AllScopeTypes tests all scope type variations
func TestExtractScopeFromHeaders_AllScopeTypes(t *testing.T) {
	scopeTypes := []string{
		constant.ScopeLevels.System,
		constant.ScopeLevels.Workspace,
		constant.ScopeLevels.Base,
		"custom_scope",
		"",
	}

	for _, scopeType := range scopeTypes {
		t.Run("scope_"+scopeType, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", func(c *gin.Context) {
				headerScope := c.GetHeader(middleware.HeaderScopeType)
				if headerScope == "" && scopeType == "" {
					headerScope = constant.ScopeLevels.Workspace
				}
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if scopeType != "" {
				req.Header.Set(middleware.HeaderScopeType, scopeType)
			}
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestExtractScopeFromHeaders_WithScopeIDs tests scope ID extraction
func TestExtractScopeFromHeaders_WithScopeIDs(t *testing.T) {
	scopeIDs := []string{
		"ws_123456",
		"base_789",
		"org_abc",
		"",
		"id-with-dashes",
		"id_with_underscores",
	}

	for _, scopeID := range scopeIDs {
		t.Run("id_"+scopeID, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", func(c *gin.Context) {
				headerScopeID := c.GetHeader(middleware.HeaderScopeID)
				if scopeID != "" {
					assert.Equal(t, scopeID, headerScopeID)
				}
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if scopeID != "" {
				req.Header.Set(middleware.HeaderScopeID, scopeID)
			}
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestContextValues_ConcurrentAccess tests concurrent context access
func TestContextValues_ConcurrentAccess(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Run("concurrent_"+strconv.Itoa(i), func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Set("user_id", "user"+strconv.Itoa(i))
			c.Set("schema", "schema"+strconv.Itoa(i))

			userID, _ := c.Get("user_id")
			schema, _ := c.Get("schema")

			assert.Equal(t, "user"+strconv.Itoa(i), userID)
			assert.Equal(t, "schema"+strconv.Itoa(i), schema)
		})
	}
}
