package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRequestLogger_AllMethods tests logging for all HTTP methods
func TestRequestLogger_AllMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Handle(method, "/test", middleware.RequestLogger(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(method, "/test", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestRequestLogger_WithHeaders tests logging with various headers
func TestRequestLogger_WithHeaders(t *testing.T) {
	headers := map[string]string{
		"User-Agent":      "test-agent/1.0",
		"Content-Type":    "application/json",
		"Accept":          "application/json",
		"X-Request-ID":    "req-123",
		"Authorization":   "Bearer token",
		"X-Forwarded-For": "192.168.1.1",
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.RequestLogger(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestDatabaseQueryLogger_LongQueries tests logging of long queries
func TestDatabaseQueryLogger_LongQueries(t *testing.T) {
	queries := []string{
		"SELECT * FROM users",
		"SELECT " + strings.Repeat("column, ", 50) + "id FROM large_table",
		strings.Repeat("SELECT * FROM table UNION ", 10) + "SELECT * FROM table",
	}

	for i, query := range queries {
		t.Run("query_"+strconv.Itoa(i), func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.DatabaseQueryLogger(), func(c *gin.Context) {
				c.Set("db_query", query)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestDatabaseQueryLogger_NoQuery tests when no query is present
func TestDatabaseQueryLogger_NoQuery(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.DatabaseQueryLogger(), func(c *gin.Context) {
		// Don't set db_query
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestDatabaseQueryLogger_EmptyQuery tests with empty query string
func TestDatabaseQueryLogger_EmptyQuery(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.DatabaseQueryLogger(), func(c *gin.Context) {
		c.Set("db_query", "")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestDatabaseQueryLogger_SpecialCharacters tests queries with special characters
func TestDatabaseQueryLogger_SpecialCharacters(t *testing.T) {
	queries := []string{
		"SELECT * FROM users WHERE name = 'O''Neil'",
		"SELECT * FROM users WHERE email = 'test@example.com'",
		"SELECT * FROM users WHERE data = '{\"key\": \"value\"}'",
		"SELECT * FROM users WHERE text LIKE '%test%'",
	}

	for i, query := range queries {
		t.Run("special_char_query_"+strconv.Itoa(i), func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.DatabaseQueryLogger(), func(c *gin.Context) {
				c.Set("db_query", query)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
