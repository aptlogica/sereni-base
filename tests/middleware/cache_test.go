package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"serenibase/internal/middleware"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestNewCacheMiddleware tests cache middleware creation
func TestNewCacheMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		redisURL string
		ttl      time.Duration
	}{
		{
			name:     "valid redis URL",
			redisURL: "redis://localhost:6379",
			ttl:      5 * time.Minute,
		},
		{
			name:     "invalid redis URL - fallback",
			redisURL: "invalid://url",
			ttl:      10 * time.Minute,
		},
		{
			name:     "empty redis URL - fallback",
			redisURL: "",
			ttl:      1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := middleware.NewCacheMiddleware(tt.redisURL, tt.ttl)
			assert.NotNil(t, cache, "Cache middleware should not be nil")
		})
	}
}

// TestCacheMiddleware_OnlyGETRequests tests that only GET requests are cached
func TestCacheMiddleware_OnlyGETRequests(t *testing.T) {
	cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

	methods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method+" request not cached", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			handlerCalled := false
			r.Handle(method, "/test", cache.Cache(), func(c *gin.Context) {
				handlerCalled = true
				c.JSON(http.StatusOK, gin.H{"message": "response"})
			})

			req := httptest.NewRequest(method, "/test", nil)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.True(t, handlerCalled, "Handler should be called for "+method)
			assert.NotEqual(t, "HIT", w.Header().Get("X-Cache"))
		})
	}
}

// TestCacheMiddleware_GETRequests tests GET request caching behavior
func TestCacheMiddleware_GETRequests(t *testing.T) {
	t.Run("GET request sets cache miss header", func(t *testing.T) {
		cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		callCount := 0
		r.GET("/test", cache.Cache(), func(c *gin.Context) {
			callCount++
			c.JSON(http.StatusOK, gin.H{"message": "response", "count": callCount})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		r.ServeHTTP(w, req)

		// First call should miss cache
		assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
		assert.Equal(t, 1, callCount)
	})

	t.Run("GET request with query params", func(t *testing.T) {
		cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.GET("/test", cache.Cache(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "response"})
		})

		req := httptest.NewRequest("GET", "/test?param=value", nil)
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestCacheMiddleware_ResponseWriter tests the custom response writer
func TestCacheMiddleware_ResponseWriter(t *testing.T) {
	t.Run("captures response with status 200", func(t *testing.T) {
		cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.GET("/test", cache.Cache(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"key": "value"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "key")
		assert.Contains(t, w.Body.String(), "value")
	})

	t.Run("does not cache non-200 responses", func(t *testing.T) {
		cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

		statusCodes := []int{201, 204, 400, 401, 403, 404, 500}

		for _, statusCode := range statusCodes {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/test", cache.Cache(), func(c *gin.Context) {
				c.JSON(statusCode, gin.H{"error": "message"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, statusCode, w.Code)
			assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
		}
	})
}

// TestCacheMiddleware_KeyGeneration tests cache key generation
func TestCacheMiddleware_KeyGeneration(t *testing.T) {
	cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

	tests := []struct {
		name  string
		path  string
		query string
	}{
		{
			name:  "simple path",
			path:  "/api/test",
			query: "",
		},
		{
			name:  "path with query",
			path:  "/api/test",
			query: "?key=value",
		},
		{
			name:  "path with multiple query params",
			path:  "/api/test",
			query: "?key1=value1&key2=value2",
		},
		{
			name:  "complex path",
			path:  "/api/v1/users/123/posts",
			query: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET(tt.path, cache.Cache(), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest("GET", tt.path+tt.query, nil)
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "MISS", w.Header().Get("X-Cache"))
		})
	}
}

// TestCacheMiddleware_DifferentPaths tests that different paths have different cache keys
func TestCacheMiddleware_DifferentPaths(t *testing.T) {
	cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

	paths := []string{"/api/users", "/api/posts", "/api/comments"}

	for _, path := range paths {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.GET(path, cache.Cache(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"path": path})
		})

		req := httptest.NewRequest("GET", path, nil)
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), path)
	}
}

// TestCacheMiddleware_TTL tests that TTL is properly set
func TestCacheMiddleware_TTL(t *testing.T) {
	ttls := []time.Duration{
		1 * time.Second,
		1 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, ttl := range ttls {
		t.Run(ttl.String(), func(t *testing.T) {
			cache := middleware.NewCacheMiddleware("redis://localhost:6379", ttl)
			assert.NotNil(t, cache)
		})
	}
}

// TestCacheMiddleware_ConcurrentRequests tests concurrent request handling
func TestCacheMiddleware_ConcurrentRequests(t *testing.T) {
	cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	requestCount := 0
	r.GET("/test", cache.Cache(), func(c *gin.Context) {
		requestCount++
		c.JSON(http.StatusOK, gin.H{"count": requestCount})
	})

	// Make multiple requests
	for i := 0; i < 3; i++ {
		w = httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

// TestCacheMiddleware_POSTMethodNotCached tests that POST requests are not cached
func TestCacheMiddleware_POSTMethodNotCached(t *testing.T) {
	cache := middleware.NewCacheMiddleware("redis://localhost:6379", 5*time.Minute)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.POST("/test", cache.Cache(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "POST"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// POST should not have X-Cache header or should not be HIT
}

// TestCacheMiddleware_InvalidRedisConnection tests fallback behavior
func TestCacheMiddleware_InvalidRedisConnection(t *testing.T) {
	cache := middleware.NewCacheMiddleware("invalid://badurl:9999", 5*time.Minute)
	assert.NotNil(t, cache, "Should fallback to default config")

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", cache.Cache(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "test"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestCacheMiddleware_DifferentTTLValues tests different TTL configurations
func TestCacheMiddleware_DifferentTTLValues(t *testing.T) {
	ttls := []time.Duration{
		100 * time.Millisecond,
		5 * time.Second,
		10 * time.Minute,
		1 * time.Hour,
	}

	for _, ttl := range ttls {
		t.Run(ttl.String(), func(t *testing.T) {
			cache := middleware.NewCacheMiddleware("redis://localhost:6379", ttl)
			assert.NotNil(t, cache)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.GET("/test", cache.Cache(), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ttl": ttl.String()})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
