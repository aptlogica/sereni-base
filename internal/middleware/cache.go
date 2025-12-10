package middleware

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type CacheMiddleware struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCacheMiddleware(redisURL string, ttl time.Duration) *CacheMiddleware {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fallback to default configuration
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opt)

	return &CacheMiddleware{
		client: client,
		ttl:    ttl,
	}
}

func (c *CacheMiddleware) Cache() gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		// Only cache GET requests
		if ctx.Request.Method != "GET" {
			ctx.Next()
			return
		}

		// Generate cache key
		key := c.generateCacheKey(ctx)

		// Try to get from cache
		cached, err := c.client.Get(context.Background(), key).Result()
		if err == nil {
			ctx.Header("X-Cache", "HIT")
			ctx.Data(200, "application/json", []byte(cached))
			ctx.Abort()
			return
		}

		// If not in cache, continue with request
		ctx.Header("X-Cache", "MISS")

		// Capture response
		w := &responseWriter{
			ResponseWriter: ctx.Writer,
			body:           &bytes.Buffer{},
		}
		ctx.Writer = w

		ctx.Next()

		// Cache the response if successful
		if w.status == 200 {
			c.client.Set(context.Background(), key, w.body.String(), c.ttl)
		}
	})
}

func (c *CacheMiddleware) generateCacheKey(ctx *gin.Context) string {
	data := fmt.Sprintf("%s:%s:%s", ctx.Request.Method, ctx.Request.URL.Path, ctx.Request.URL.RawQuery)
	return fmt.Sprintf("postgrest:%x", md5.Sum([]byte(data)))
}

type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
