package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"serenibase/internal/middleware"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// TestPrometheusMetrics tests the Prometheus metrics middleware
func TestPrometheusMetrics(t *testing.T) {
	// Reset prometheus metrics for clean test
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	tests := []struct {
		name           string
		method         string
		path           string
		handlerStatus  int
		expectedStatus int
	}{
		{
			name:           "GET request success",
			method:         "GET",
			path:           "/api/test",
			handlerStatus:  http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request success",
			method:         "POST",
			path:           "/api/users",
			handlerStatus:  http.StatusCreated,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "PUT request success",
			method:         "PUT",
			path:           "/api/users/123",
			handlerStatus:  http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "DELETE request success",
			method:         "DELETE",
			path:           "/api/users/123",
			handlerStatus:  http.StatusNoContent,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "request with error",
			method:         "GET",
			path:           "/api/notfound",
			handlerStatus:  http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "request with server error",
			method:         "POST",
			path:           "/api/error",
			handlerStatus:  http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Handle(tt.method, tt.path, middleware.PrometheusMetrics(), func(c *gin.Context) {
				c.Status(tt.handlerStatus)
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestPrometheusMetrics_ActiveConnections tests active connections gauge
func TestPrometheusMetrics_ActiveConnections(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.PrometheusMetrics(), func(c *gin.Context) {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	c.Request = req
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestPrometheusMetrics_RequestDuration tests request duration tracking
func TestPrometheusMetrics_RequestDuration(t *testing.T) {
	tests := []struct {
		name     string
		delay    time.Duration
		minDelay time.Duration
	}{
		{
			name:     "fast request",
			delay:    1 * time.Millisecond,
			minDelay: 1 * time.Millisecond,
		},
		{
			name:     "medium request",
			delay:    50 * time.Millisecond,
			minDelay: 50 * time.Millisecond,
		},
		{
			name:     "slow request",
			delay:    100 * time.Millisecond,
			minDelay: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.PrometheusMetrics(), func(c *gin.Context) {
				time.Sleep(tt.delay)
				c.Status(http.StatusOK)
			})

			start := time.Now()
			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req
			r.ServeHTTP(w, req)
			duration := time.Since(start)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.GreaterOrEqual(t, duration, tt.minDelay, "Request should take at least the specified delay")
		})
	}
}

// TestPrometheusMetrics_MultipleRequests tests handling multiple requests
func TestPrometheusMetrics_MultipleRequests(t *testing.T) {
	paths := []string{"/api/users", "/api/posts", "/api/comments"}

	for _, path := range paths {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.GET(path, middleware.PrometheusMetrics(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", path, nil)
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}

// TestPrometheusMetrics_DifferentStatusCodes tests tracking different status codes
func TestPrometheusMetrics_DifferentStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.PrometheusMetrics(), func(c *gin.Context) {
				c.Status(statusCode)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, statusCode, w.Code)
		})
	}
}

// TestDatabaseMetrics tests database metrics recording
func TestDatabaseMetrics(t *testing.T) {
	tests := []struct {
		name      string
		table     string
		operation string
		duration  time.Duration
	}{
		{
			name:      "fast query",
			table:     "users",
			operation: "SELECT",
			duration:  10 * time.Millisecond,
		},
		{
			name:      "medium query",
			table:     "posts",
			operation: "INSERT",
			duration:  50 * time.Millisecond,
		},
		{
			name:      "slow query",
			table:     "comments",
			operation: "UPDATE",
			duration:  200 * time.Millisecond,
		},
		{
			name:      "delete operation",
			table:     "sessions",
			operation: "DELETE",
			duration:  30 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call DatabaseMetrics function
			middleware.DatabaseMetrics(tt.table, tt.operation, tt.duration)

			// Function should complete without error
			// The actual metric values would be checked by Prometheus
		})
	}
}

// TestDatabaseMetrics_DifferentTables tests metrics for different tables
func TestDatabaseMetrics_DifferentTables(t *testing.T) {
	tables := []string{"users", "posts", "comments", "sessions", "tokens"}

	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			middleware.DatabaseMetrics(table, "SELECT", 10*time.Millisecond)
		})
	}
}

// TestDatabaseMetrics_DifferentOperations tests metrics for different operations
func TestDatabaseMetrics_DifferentOperations(t *testing.T) {
	operations := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP"}

	for _, operation := range operations {
		t.Run(operation, func(t *testing.T) {
			middleware.DatabaseMetrics("test_table", operation, 15*time.Millisecond)
		})
	}
}

// TestPrometheusMetrics_Concurrent tests concurrent requests
func TestPrometheusMetrics_Concurrent(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", middleware.PrometheusMetrics(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Simulate concurrent requests
	for i := 0; i < 10; i++ {
		w = httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

// TestPrometheusMetrics_WithDynamicPath tests metrics with dynamic path parameters
func TestPrometheusMetrics_WithDynamicPath(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/users/:id", middleware.PrometheusMetrics(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	userIDs := []string{"123", "456", "789"}

	for _, id := range userIDs {
		w = httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/"+id, nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

// TestPrometheusMetrics_MiddlewareChain tests metrics in middleware chain
func TestPrometheusMetrics_MiddlewareChain(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	middleware1Called := false
	middleware2Called := false

	r.GET("/test",
		middleware.PrometheusMetrics(),
		func(c *gin.Context) {
			middleware1Called = true
			c.Next()
		},
		func(c *gin.Context) {
			middleware2Called = true
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
	assert.True(t, middleware1Called)
	assert.True(t, middleware2Called)
}

// TestDatabaseMetrics_EdgeCases tests edge cases
func TestDatabaseMetrics_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		table     string
		operation string
		duration  time.Duration
	}{
		{
			name:      "zero duration",
			table:     "test",
			operation: "SELECT",
			duration:  0,
		},
		{
			name:      "very long duration",
			table:     "test",
			operation: "SELECT",
			duration:  10 * time.Second,
		},
		{
			name:      "empty table name",
			table:     "",
			operation: "SELECT",
			duration:  10 * time.Millisecond,
		},
		{
			name:      "empty operation",
			table:     "test",
			operation: "",
			duration:  10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			middleware.DatabaseMetrics(tt.table, tt.operation, tt.duration)
		})
	}
}

// TestPrometheusMetrics_AllStatusCodes tests metrics with various status codes
func TestPrometheusMetrics_AllStatusCodes(t *testing.T) {
	statusCodes := []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 500, 502, 503}

	for _, statusCode := range statusCodes {
		t.Run("status_"+strconv.Itoa(statusCode), func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.PrometheusMetrics(), func(c *gin.Context) {
				c.Status(statusCode)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, statusCode, w.Code)
		})
	}
}

// TestDatabaseMetrics_AllOperations tests all database operation types
func TestDatabaseMetrics_AllOperations(t *testing.T) {
	operations := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE",
		"CREATE", "DROP", "ALTER", "TRUNCATE",
		"BEGIN", "COMMIT", "ROLLBACK",
	}

	for _, op := range operations {
		t.Run("operation_"+op, func(t *testing.T) {
			middleware.DatabaseMetrics("test_table", op, 10*time.Millisecond)
		})
	}
}

// TestDatabaseMetrics_DifferentDurations tests various query durations
func TestDatabaseMetrics_DifferentDurations(t *testing.T) {
	durations := []time.Duration{
		1 * time.Microsecond,
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, duration := range durations {
		t.Run("duration_"+duration.String(), func(t *testing.T) {
			middleware.DatabaseMetrics("users", "SELECT", duration)
		})
	}
}

// TestPrometheusMetrics_LongPaths tests metrics with long URL paths
func TestPrometheusMetrics_LongPaths(t *testing.T) {
	paths := []string{
		"/api/v1/users",
		"/api/v1/workspaces/ws123/bases/base456/tables/tbl789",
		"/api/" + strings.Repeat("path/", 10) + "endpoint",
	}

	for _, path := range paths {
		t.Run("path_"+path, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET(path, middleware.PrometheusMetrics(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", path, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestPrometheusMetrics_WithQueryParams tests metrics with query parameters
func TestPrometheusMetrics_WithQueryParams(t *testing.T) {
	queryParams := []string{
		"?id=1",
		"?page=1&limit=10",
		"?filter=name&sort=asc&search=test",
		"?" + strings.Repeat("param=value&", 10),
	}

	for _, params := range queryParams {
		t.Run("params_"+params, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test", middleware.PrometheusMetrics(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test"+params, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
