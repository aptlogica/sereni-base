// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ValidateTableName ensures table names are safe for SQL
func ValidateTableName() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		tableName := c.Param("table")
		if tableName != "" {
			if !isValidIdentifier(tableName) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid table name. Table names must start with a letter and contain only letters, numbers, and underscores.",
				})
				c.Abort()
				return
			}
		}
		c.Next()
	})
}

// ValidateColumnName ensures column names are safe for SQL
func ValidateColumnName() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		columnName := c.Param("column")
		if columnName != "" {
			if !isValidIdentifier(columnName) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid column name. Column names must start with a letter and contain only letters, numbers, and underscores.",
				})
				c.Abort()
				return
			}
		}
		c.Next()
	})
}

// RequestSizeLimit limits the size of request bodies
func RequestSizeLimit(maxSize int64) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	})
}

// RateLimiter provides basic rate limiting per client IP
func RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	return RateLimiterBy(func(c *gin.Context) string { return c.ClientIP() }, requestsPerMinute,
		func(c *gin.Context, _ time.Duration) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
		})
}

// RateLimiterBy allows requestsPerMinute requests per key in any 60 second window. Over the limit it
// calls onLimit, which must write the response and abort, with the time until the next request is allowed.
func RateLimiterBy(key func(*gin.Context) string, requestsPerMinute int, onLimit func(c *gin.Context, retryAfter time.Duration)) gin.HandlerFunc {
	// This is a simple in-memory rate limiter
	// For production, consider using external rate limiting solutions
	clients := make(map[string][]int64)
	var mu sync.Mutex

	return gin.HandlerFunc(func(c *gin.Context) {
		clientKey := key(c)
		now := time.Now().Unix()
		mu.Lock()
		if requests, exists := clients[clientKey]; exists {
			// Remove old requests (older than 1 minute)
			var validRequests []int64
			for _, timestamp := range requests {
				if now-timestamp < 60 {
					validRequests = append(validRequests, timestamp)
				}
			}

			if len(validRequests) >= requestsPerMinute {
				clients[clientKey] = validRequests
				mu.Unlock()
				// The oldest request in the window leaves it first
				retryAfter := time.Minute
				if len(validRequests) > 0 {
					retryAfter = time.Duration(60-(now-validRequests[0])) * time.Second
				}
				onLimit(c, retryAfter)
				return
			}

			validRequests = append(validRequests, now)
			clients[clientKey] = validRequests
		} else {
			clients[clientKey] = []int64{now}
		}
		mu.Unlock()

		c.Next()
	})
}

func isValidIdentifier(name string) bool {
	// PostgreSQL identifier rules: start with letter or underscore, followed by letters, digits, or underscores
	match, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)

	reservedWords := map[string]bool{
		"select": true, "from": true, "where": true, "insert": true, "update": true,
		"delete": true, "create": true, "drop": true, "alter": true, "table": true,
		"column": true, "index": true, "view": true, "user": true, "order": true,
		"group": true, "having": true, "limit": true, "offset": true, "join": true,
		"inner": true, "left": true, "right": true, "full": true, "outer": true,
		"and": true, "or": true, "not": true, "null": true, "true": true, "false": true,
	}

	return match && !reservedWords[strings.ToLower(name)]
}
