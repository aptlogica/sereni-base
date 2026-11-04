// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"strconv"
	"strings"

	"github.com/aptlogica/sereni-base/internal/config"

	"github.com/gin-gonic/gin"
)

// PublicAPIPathPrefix starts every workspace public API path (/api/v1/{ws}/...). No internal route may start with it.
const PublicAPIPathPrefix = "/api/v1/ws_"

const (
	publicAPIAllowMethods   = "GET, POST, PATCH, DELETE, OPTIONS"
	publicAPIAllowHeaders   = "Content-Type, Idempotency-Key, X-Request-Id, Authorization"
	publicAPIExposeHeaders  = "X-Request-Id, Retry-After, Idempotent-Replayed"
	publicAPIPreflightCache = "600"
)

// publicAPICORS lets a site on any origin call the workspace public API. It uses no cookies or sessions, so
// browsers send no credentials and "*" is safe; access will be decided by API keys sent in headers.
func publicAPICORS(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", publicAPIAllowMethods)
	// Preflights get back whatever headers the client asked for, so client libraries' own headers pass too
	if requested := c.Request.Header.Get("Access-Control-Request-Headers"); requested != "" {
		c.Header("Access-Control-Allow-Headers", requested)
		c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
	} else {
		c.Header("Access-Control-Allow-Headers", publicAPIAllowHeaders)
	}
	c.Header("Access-Control-Expose-Headers", publicAPIExposeHeaders)
	c.Header("Access-Control-Max-Age", publicAPIPreflightCache)
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, PublicAPIPathPrefix) {
			publicAPICORS(c)
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			c.Next()
			return
		}

		corsConfig := config.AppConfig.CORS
		origin := c.Request.Header.Get("Origin")

		// Check if the request origin is allowed
		allowedOrigin := ""
		if corsConfig.AllowedOrigins == "*" {
			allowedOrigin = "*"
		} else {
			origins := strings.Split(corsConfig.AllowedOrigins, ",")
			for _, o := range origins {
				if strings.TrimSpace(o) == origin {
					allowedOrigin = origin
					break
				}
			}
		}

		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Headers", corsConfig.AllowedHeaders)
			c.Header("Access-Control-Allow-Methods", corsConfig.AllowedMethods)
			c.Header("Access-Control-Allow-Credentials", strconv.FormatBool(corsConfig.AllowCredentials))
			// Let browser apps read the request ID and how long to wait after a 429
			c.Header("Access-Control-Expose-Headers", "X-Request-Id, Retry-After")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
