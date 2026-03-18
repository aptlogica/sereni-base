// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"strconv"
	"strings"

	"github.com/aptlogica/sereni-base/internal/config"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
