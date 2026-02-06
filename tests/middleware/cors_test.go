package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"serenibase/internal/config"
	"serenibase/internal/middleware"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupCORSConfig sets up the config for CORS testing
func setupCORSConfig(origins, methods, headers string, credentials bool) {
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{}
	}
	config.AppConfig.CORS = config.CORSConfig{
		AllowedOrigins:   origins,
		AllowedMethods:   methods,
		AllowedHeaders:   headers,
		AllowCredentials: credentials,
	}
}

// TestCORS_Wildcard tests CORS with wildcard origin
func TestCORS_Wildcard(t *testing.T) {
	setupCORSConfig("*", "GET,POST,PUT,DELETE", "Content-Type,Authorization", true)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.CORS(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	c.Request = req
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET,POST,PUT,DELETE", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type,Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

// TestCORS_SpecificOrigin tests CORS with specific allowed origin
func TestCORS_SpecificOrigin(t *testing.T) {
	tests := []struct {
		name                  string
		allowedOrigins        string
		requestOrigin         string
		expectedAllowOrigin   string
		shouldHaveCORSHeaders bool
	}{
		{
			name:                  "matching single origin",
			allowedOrigins:        "https://example.com",
			requestOrigin:         "https://example.com",
			expectedAllowOrigin:   "https://example.com",
			shouldHaveCORSHeaders: true,
		},
		{
			name:                  "non-matching origin",
			allowedOrigins:        "https://example.com",
			requestOrigin:         "https://malicious.com",
			expectedAllowOrigin:   "",
			shouldHaveCORSHeaders: false,
		},
		{
			name:                  "matching one of multiple origins",
			allowedOrigins:        "https://example.com,https://test.com,https://demo.com",
			requestOrigin:         "https://test.com",
			expectedAllowOrigin:   "https://test.com",
			shouldHaveCORSHeaders: true,
		},
		{
			name:                  "not matching any of multiple origins",
			allowedOrigins:        "https://example.com,https://test.com",
			requestOrigin:         "https://unauthorized.com",
			expectedAllowOrigin:   "",
			shouldHaveCORSHeaders: false,
		},
		{
			name:                  "matching with spaces in config",
			allowedOrigins:        "https://example.com, https://test.com, https://demo.com",
			requestOrigin:         "https://demo.com",
			expectedAllowOrigin:   "https://demo.com",
			shouldHaveCORSHeaders: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupCORSConfig(tt.allowedOrigins, "GET,POST", "Content-Type", false)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.CORS(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			if tt.shouldHaveCORSHeaders {
				assert.Equal(t, tt.expectedAllowOrigin, w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "GET,POST", w.Header().Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Content-Type", w.Header().Get("Access-Control-Allow-Headers"))
			} else {
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

// TestCORS_PreflightRequest tests OPTIONS preflight requests
func TestCORS_PreflightRequest(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins string
		requestOrigin  string
		expectedStatus int
	}{
		{
			name:           "preflight with wildcard",
			allowedOrigins: "*",
			requestOrigin:  "https://example.com",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "preflight with allowed origin",
			allowedOrigins: "https://example.com",
			requestOrigin:  "https://example.com",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "preflight with disallowed origin",
			allowedOrigins: "https://example.com",
			requestOrigin:  "https://malicious.com",
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupCORSConfig(tt.allowedOrigins, "GET,POST,PUT,DELETE", "Content-Type,Authorization", true)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			handlerCalled := false
			r.OPTIONS("/test", middleware.CORS(), func(c *gin.Context) {
				handlerCalled = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("OPTIONS", "/test", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.False(t, handlerCalled, "Handler should not be called for OPTIONS requests")
		})
	}
}

// TestCORS_Credentials tests credential handling
func TestCORS_Credentials(t *testing.T) {
	tests := []struct {
		name             string
		allowCredentials bool
		expectedValue    string
	}{
		{
			name:             "credentials allowed",
			allowCredentials: true,
			expectedValue:    "true",
		},
		{
			name:             "credentials not allowed",
			allowCredentials: false,
			expectedValue:    "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupCORSConfig("*", "GET,POST", "Content-Type", tt.allowCredentials)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.CORS(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedValue, w.Header().Get("Access-Control-Allow-Credentials"))
		})
	}
}

// TestCORS_NoOriginHeader tests requests without Origin header
func TestCORS_NoOriginHeader(t *testing.T) {
	setupCORSConfig("https://example.com", "GET,POST", "Content-Type", true)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.CORS(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No Origin header set
	c.Request = req
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Should not set CORS headers when no Origin header is present
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORS_DifferentMethods tests CORS with different HTTP methods
func TestCORS_DifferentMethods(t *testing.T) {
	setupCORSConfig("*", "GET,POST,PUT,DELETE,PATCH", "Content-Type", true)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Handle(method, "/test", middleware.CORS(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

// TestCORS_CustomHeaders tests custom allowed headers
func TestCORS_CustomHeaders(t *testing.T) {
	tests := []struct {
		name           string
		allowedHeaders string
	}{
		{
			name:           "single header",
			allowedHeaders: "Content-Type",
		},
		{
			name:           "multiple headers",
			allowedHeaders: "Content-Type,Authorization,X-Custom-Header",
		},
		{
			name:           "many headers",
			allowedHeaders: "Content-Type,Authorization,X-API-Key,X-Request-ID,Accept",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupCORSConfig("*", "GET,POST", tt.allowedHeaders, true)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.CORS(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.allowedHeaders, w.Header().Get("Access-Control-Allow-Headers"))
		})
	}
}

// TestCORS_CustomMethods tests custom allowed methods
func TestCORS_CustomMethods(t *testing.T) {
	tests := []struct {
		name           string
		allowedMethods string
	}{
		{
			name:           "GET only",
			allowedMethods: "GET",
		},
		{
			name:           "GET and POST",
			allowedMethods: "GET,POST",
		},
		{
			name:           "all standard methods",
			allowedMethods: "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupCORSConfig("*", tt.allowedMethods, "Content-Type", false)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.CORS(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.allowedMethods, w.Header().Get("Access-Control-Allow-Methods"))
		})
	}
}

// TestCORS_Integration tests CORS with multiple middleware
func TestCORS_Integration(t *testing.T) {
	setupCORSConfig("https://example.com", "GET,POST", "Content-Type", true)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test",
		middleware.CORS(),
		func(c *gin.Context) {
			// Another middleware
			c.Next()
		},
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		},
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	c.Request = req
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Body.String(), "success")
}
