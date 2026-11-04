package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiterByKey(t *testing.T) {
	var retryAfter time.Duration
	limiter := middleware.RateLimiterBy(
		func(c *gin.Context) string { return c.GetHeader("X-Key") },
		2,
		func(c *gin.Context, wait time.Duration) {
			retryAfter = wait
			c.AbortWithStatus(http.StatusTooManyRequests)
		})
	_, r := gin.CreateTestContext(httptest.NewRecorder())
	r.GET("/test", limiter, func(c *gin.Context) { c.Status(http.StatusOK) })

	send := func(key string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Key", key)
		r.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, send("a"))
	assert.Equal(t, http.StatusOK, send("a"))
	assert.Equal(t, http.StatusTooManyRequests, send("a"))
	assert.Greater(t, retryAfter, time.Duration(0))
	assert.LessOrEqual(t, retryAfter, time.Minute)

	// Other keys have their own budget
	assert.Equal(t, http.StatusOK, send("b"))
}

func TestRateLimiterKeepsIPBehaviour(t *testing.T) {
	_, r := gin.CreateTestContext(httptest.NewRecorder())
	r.GET("/test", middleware.RateLimiter(1), func(c *gin.Context) { c.Status(http.StatusOK) })

	codes := make([]int, 2)
	for i := range codes {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		r.ServeHTTP(w, req)
		codes[i] = w.Code
	}
	assert.Equal(t, []int{http.StatusOK, http.StatusTooManyRequests}, codes)
}
