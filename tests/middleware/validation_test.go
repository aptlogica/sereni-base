package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"serenibase/internal/middleware"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestValidateTableName tests table name validation middleware
func TestValidateTableName(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		expectedStatus int
	}{
		{"valid table name", "users", http.StatusOK},
		{"valid with underscore", "user_accounts", http.StatusOK},
		{"valid with number", "table123", http.StatusOK},
		{"invalid - starts with number", "123table", http.StatusBadRequest},
		{"invalid - special char", "user-table", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/table/:table", middleware.ValidateTableName(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/api/table/"+tt.tableName, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

// TestValidateColumnName tests column name validation middleware
func TestValidateColumnName(t *testing.T) {
	tests := []struct {
		name           string
		columnName     string
		expectedStatus int
	}{
		{"valid column name", "email", http.StatusOK},
		{"valid with underscore", "first_name", http.StatusOK},
		{"valid with number", "col123", http.StatusOK},
		{"invalid - starts with number", "1column", http.StatusBadRequest},
		{"invalid - dash", "col-name", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/column/:column", middleware.ValidateColumnName(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/api/column/"+tt.columnName, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

// TestRequestSizeLimit tests request size limiting middleware
func TestRequestSizeLimit(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int64
		bodySize int
	}{
		{"within limit", 1024, 512},
		{"at limit", 1024, 1024},
		{"small request", 10000, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.POST("/api/test", middleware.RequestSizeLimit(tt.maxSize), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			body := strings.Repeat("a", tt.bodySize)
			req := httptest.NewRequest("POST", "/api/test", strings.NewReader(body))
			r.ServeHTTP(w, req)
		})
	}
}

// TestRateLimiter tests rate limiting middleware
func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.GET("/api/test", middleware.RateLimiter(5), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		for i := 0; i < 5; i++ {
			w = httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			r.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				t.Errorf("Request %d should not be rate limited", i+1)
			}
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.GET("/api/test", middleware.RateLimiter(2), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		for i := 0; i < 3; i++ {
			w = httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = "127.0.0.1:5678"
			r.ServeHTTP(w, req)

			if i < 2 && w.Code == http.StatusTooManyRequests {
				t.Errorf("Request %d should have been allowed", i+1)
			}

			if i >= 2 && w.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d should have been blocked", i+1)
			}
		}
	})
}

// TestRequestLogger tests the request logger middleware
func TestRequestLogger(t *testing.T) {
	logger := middleware.RequestLogger()
	if logger == nil {
		t.Fatal("RequestLogger() returned nil")
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/api/test", logger, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	r.ServeHTTP(w, req)
}

// TestDatabaseQueryLogger tests the database query logger middleware
func TestDatabaseQueryLogger(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/api/test", middleware.DatabaseQueryLogger(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("DatabaseQueryLogger middleware interfered with request")
	}
}

// TestValidationMiddlewareChaining tests chaining multiple validation middlewares
func TestValidationMiddlewareChaining(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/api/table/:table/column/:column",
		middleware.ValidateTableName(),
		middleware.ValidateColumnName(),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

	req := httptest.NewRequest("GET", "/api/table/valid_table/column/valid_column", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Chained middleware failed, status = %d", w.Code)
	}
}

// TestValidateTableNameReservedWords tests reserved SQL keywords
func TestValidateTableNameReservedWords(t *testing.T) {
	reservedWords := []string{"select", "from", "where", "table"}

	for _, word := range reservedWords {
		t.Run("reserved_"+word, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/table/:table", middleware.ValidateTableName(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/api/table/"+word, nil)
			r.ServeHTTP(w, req)

			// Reserved words should be rejected
			if w.Code != http.StatusBadRequest {
				t.Errorf("Reserved word %q should be rejected, got status %d", word, w.Code)
			}
		})
	}
}

// TestRateLimiterDifferentClients tests rate limiting for different clients
func TestRateLimiterDifferentClients(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/api/test", middleware.RateLimiter(2), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Client 1
	for i := 0; i < 2; i++ {
		w = httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Client 1 request %d should be allowed", i+1)
		}
	}

	// Client 2 should have separate limit
	w = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.2:5678"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Client 2 request should be allowed (separate from client 1)")
	}
}

// TestValidateTableName_EdgeCases tests edge cases for table name validation
func TestValidateTableName_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		tableName      string
		expectedStatus int
	}{
		{"empty string", "", http.StatusNotFound},
		{"very long name", strings.Repeat("a", 100), http.StatusOK},
		{"underscore prefix", "_users", http.StatusOK},
		{"double underscore", "user__table", http.StatusOK},
		{"all caps", "USERS", http.StatusOK},
		{"mixed case", "UserAccounts", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/table/:table", middleware.ValidateTableName(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// URL encode the table name to handle special characters
			encodedName := url.PathEscape(tt.tableName)
			req := httptest.NewRequest("GET", "/api/table/"+encodedName, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

// TestValidateColumnName_EdgeCases tests edge cases for column name validation
func TestValidateColumnName_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		columnName     string
		expectedStatus int
	}{
		{"empty string", "", http.StatusNotFound},
		{"very long name", strings.Repeat("b", 100), http.StatusOK},
		{"underscore only", "_", http.StatusOK},
		{"double underscore", "col__name", http.StatusOK},
		{"all caps", "EMAIL", http.StatusOK},
		{"mixed case", "firstName", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/column/:column", middleware.ValidateColumnName(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// URL encode the column name to handle special characters
			encodedName := url.PathEscape(tt.columnName)
			req := httptest.NewRequest("GET", "/api/column/"+encodedName, nil)
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status code = %d, want %d for column %q", w.Code, tt.expectedStatus, tt.columnName)
			}
		})
	}
}

// TestRequestSizeLimit_EdgeCases tests edge cases for request size limiting
func TestRequestSizeLimit_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		maxSize        int64
		bodySize       int
		expectedStatus int
	}{
		{"empty body", 1024, 0, http.StatusOK},
		{"exactly at limit", 1024, 1024, http.StatusOK},
		{"one byte over", 1024, 1025, http.StatusRequestEntityTooLarge},
		{"very large limit", 1000000, 5000, http.StatusOK},
		{"zero limit", 0, 1, http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.POST("/api/test", middleware.RequestSizeLimit(tt.maxSize), func(c *gin.Context) {
				// Actually read the body to trigger MaxBytesReader limit
				_, err := io.ReadAll(c.Request.Body)
				if err != nil {
					c.Status(http.StatusRequestEntityTooLarge)
					return
				}
				c.Status(http.StatusOK)
			})

			body := strings.Repeat("x", tt.bodySize)
			req := httptest.NewRequest("POST", "/api/test", strings.NewReader(body))
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}

// TestRateLimiter_EdgeCases tests edge cases for rate limiting
func TestRateLimiter_EdgeCases(t *testing.T) {
	// Note: Zero rate limit is not properly handled by the middleware
	// The first request will always be allowed for a new client

	t.Run("very high rate limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.GET("/api/test", middleware.RateLimiter(1000), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		for i := 0; i < 50; i++ {
			w = httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = "127.0.0.1:2222"
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d should be allowed with high rate limit", i+1)
			}
		}
	})

	t.Run("concurrent requests from same client", func(t *testing.T) {
		_, r := gin.CreateTestContext(httptest.NewRecorder())

		r.GET("/api/test", middleware.RateLimiter(10), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < 15; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = "127.0.0.1:3333"
			r.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		if successCount > 10 {
			t.Errorf("Too many successful requests: %d (expected max 10)", successCount)
		}
		if rateLimitedCount < 5 {
			t.Errorf("Too few rate limited requests: %d (expected at least 5)", rateLimitedCount)
		}
	})
}

// TestValidationMiddleware_AllHTTPMethods tests validation with all HTTP methods
func TestValidationMiddleware_AllHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Handle(method, "/api/table/:table", middleware.ValidateTableName(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/api/table/users", nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Method %s failed validation", method)
			}
		})
	}
}

// TestRequestLogger_DifferentPaths tests request logger with different paths
func TestRequestLogger_DifferentPaths(t *testing.T) {
	paths := []string{"/api/users", "/api/posts/123", "/health", "/api/workspaces/ws1/bases/b1"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET(path, middleware.RequestLogger(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("RequestLogger failed for path %s", path)
			}
		})
	}
}

// TestDatabaseQueryLogger_WithQueries tests database query logger with different query patterns
func TestDatabaseQueryLogger_WithQueries(t *testing.T) {
	queries := []string{
		"SELECT * FROM users",
		"INSERT INTO posts VALUES (1, 'test')",
		"UPDATE users SET name = 'John'",
		"DELETE FROM posts WHERE id = 1",
	}

	for _, query := range queries {
		t.Run(query[:6], func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/test", middleware.DatabaseQueryLogger(), func(c *gin.Context) {
				c.Set("query", query)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/api/test", nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("DatabaseQueryLogger failed for query %s", query)
			}
		})
	}
}

// TestMiddlewareChaining_Complex tests complex middleware chaining scenarios
func TestMiddlewareChaining_Complex(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.POST("/api/table/:table/column/:column",
		middleware.RequestSizeLimit(1024),
		middleware.RateLimiter(10),
		middleware.ValidateTableName(),
		middleware.ValidateColumnName(),
		middleware.RequestLogger(),
		middleware.DatabaseQueryLogger(),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

	body := strings.Repeat("test", 50)
	req := httptest.NewRequest("POST", "/api/table/users/column/email", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:4444"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Complex middleware chain failed, status = %d", w.Code)
	}
}

// TestValidateTableName_WithNumbers tests table names with numbers
func TestValidateTableName_WithNumbers(t *testing.T) {
	tableNames := []string{
		"table1", "table123", "user_accounts_2023", "data_v2",
	}

	for _, tableName := range tableNames {
		t.Run(tableName, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/api/table/:table", middleware.ValidateTableName(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/api/table/"+tableName, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected OK, got %d", w.Code)
			}
		})
	}
}

// TestRequestLogger_WithUserAgent tests request logger with different user agents
func TestRequestLogger_WithUserAgent(t *testing.T) {
	userAgents := []string{
		"Mozilla/5.0",
		"curl/7.64.1",
		"PostmanRuntime/7.26.8",
	}

	for _, ua := range userAgents {
		t.Run(ua, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.RequestLogger(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", ua)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected OK, got %d", w.Code)
			}
		})
	}
}
