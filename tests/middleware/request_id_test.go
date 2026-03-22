package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestRequestID tests request ID middleware
func TestRequestID(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		expectNewUUID bool
	}{
		{
			name:          "generates new ID when not provided",
			headerValue:   "",
			expectNewUUID: true,
		},
		{
			name:          "uses provided ID",
			headerValue:   "custom-request-id-123",
			expectNewUUID: false,
		},
		{
			name:          "trims whitespace from provided ID",
			headerValue:   "  trimmed-id  ",
			expectNewUUID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			var capturedRequestID string
			r.GET("/test", middleware.RequestID(), func(c *gin.Context) {
				requestID, exists := c.Get("request_id")
				if !exists {
					t.Error("request_id not found in context")
					return
				}
				capturedRequestID = requestID.(string)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Request-Id", tt.headerValue)
			}
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			responseID := w.Header().Get("X-Request-Id")
			if responseID == "" {
				t.Error("X-Request-Id header not set in response")
			}

			if tt.expectNewUUID {
				if capturedRequestID == "" {
					t.Error("Expected new UUID to be generated")
				}
				// UUID format validation: basic length check
				if len(capturedRequestID) != 36 {
					t.Errorf("Expected UUID format (36 chars), got %d chars", len(capturedRequestID))
				}
			} else {
				expected := "custom-request-id-123"
				if tt.headerValue == "  trimmed-id  " {
					expected = "trimmed-id"
				}
				if capturedRequestID != expected {
					t.Errorf("Expected request ID '%s', got '%s'", expected, capturedRequestID)
				}
			}
		})
	}
}

// TestRequestIDKey tests that the key constant is correct
func TestRequestIDKey(t *testing.T) {
	expected := "request_id"
	if middleware.RequestIDKey != expected {
		t.Errorf("Expected RequestIDKey to be '%s', got '%s'", expected, middleware.RequestIDKey)
	}
}

// TestRequestLogger tests request logging middleware
func TestRequestLogger_RequestID(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.RequestLogger(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestDatabaseQueryLogger tests database query logging middleware
func TestDatabaseQueryLogger_RequestID(t *testing.T) {
	t.Run("normal request", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.GET("/test", middleware.DatabaseQueryLogger(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("slow request", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.GET("/slow", middleware.DatabaseQueryLogger(), func(c *gin.Context) {
			// Simulate slow operation
			time.Sleep(1100 * time.Millisecond)
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/slow", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// TestRequestLoggerWithDifferentMethods tests request logger with different HTTP methods
func TestRequestLoggerWithDifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Handle(method, "/test", middleware.RequestLogger(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/test", nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}

// TestRequestLoggerWithError tests request logger with error responses
func TestRequestLoggerWithError(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/error", middleware.RequestLogger(), func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})

	req := httptest.NewRequest("GET", "/error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// TestRequestIDChain tests middleware chaining with request ID
func TestRequestIDChain(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/chain", middleware.RequestID(), middleware.RequestLogger(), func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		if !exists {
			t.Error("request_id not found after chaining")
		}
		if requestID.(string) == "" {
			t.Error("request_id is empty")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/chain", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}
