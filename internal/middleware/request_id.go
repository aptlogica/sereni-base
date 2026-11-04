// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

// A client-supplied ID is kept only when it is short and safe to write into logs
var clientRequestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// NewRequestID returns req_ followed by a UUID without dashes
func NewRequestID() string {
	return "req_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

// RequestID ensures every request has a stable ID and exposes it via context + header.
// A client's X-Request-Id is reused when it is at most 64 characters of [A-Za-z0-9_-]; otherwise a new ID replaces it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-Id"))
		if !clientRequestIDPattern.MatchString(requestID) {
			requestID = NewRequestID()
		}

		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-Id", requestID)
		c.Next()
	}
}
