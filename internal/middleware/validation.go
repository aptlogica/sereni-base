// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"net/http"
	"regexp"
	"strings"
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

// RateLimiter provides basic rate limiting
func RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	// This is a simple in-memory rate limiter
	// For production, use Redis or similar
	clients := make(map[string][]int64)

	return gin.HandlerFunc(func(c *gin.Context) {
		clientIP := c.ClientIP()
		// now := gin.H{"timestamp": nil}["timestamp"].(int64)
		now := time.Now().Unix()
		if requests, exists := clients[clientIP]; exists {
			// Remove old requests (older than 1 minute)
			var validRequests []int64
			for _, timestamp := range requests {
				if now-timestamp < 60 {
					validRequests = append(validRequests, timestamp)
				}
			}

			if len(validRequests) >= requestsPerMinute {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Rate limit exceeded",
				})
				c.Abort()
				return
			}

			validRequests = append(validRequests, now)
			clients[clientIP] = validRequests
		} else {
			clients[clientIP] = []int64{now}
		}

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
