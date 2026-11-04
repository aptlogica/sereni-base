// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// PublicWorkspaceIDKey is the gin context key holding the workspace public ID (ws_...) on public API requests
const PublicWorkspaceIDKey = "public_workspace_id"

// RequestLogger logs all requests with timing information, the request ID and, on public API
// requests, the workspace public ID
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		line := fmt.Sprintf("[%s] %s %s %d %s \"%s\" %s \"%s\" %s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.ClientIP,
			param.Method,
			param.StatusCode,
			param.Latency,
			param.Path,
			param.Request.Proto,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
		if requestID, ok := param.Keys[RequestIDKey].(string); ok && requestID != "" {
			line += " request_id=" + requestID
		}
		if publicID, ok := param.Keys[PublicWorkspaceIDKey].(string); ok && publicID != "" {
			line += " ws=" + publicID
		}
		return line + "\n"
	})
}

// DatabaseQueryLogger logs database queries (for debugging)
func DatabaseQueryLogger() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// Log slow queries (> 1 second)
		duration := time.Since(start)
		if duration > time.Second {
			log.Printf("SLOW QUERY: %s %s took %v", c.Request.Method, c.Request.URL.Path, duration)
		}
	})
}
