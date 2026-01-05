package middleware

import (
	"strconv"
	"strings"

	"serenibase/internal/config"

	"github.com/gin-gonic/gin"
)

// func CORS() gin.HandlerFunc {
// 	return gin.HandlerFunc(func(c *gin.Context) {
// 		c.Header("Access-Control-Allow-Origin", "*")
// 		c.Header("Access-Control-Allow-Credentials", "true")
// 		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
// 		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

// 		if c.Request.Method == "OPTIONS" {
// 			c.AbortWithStatus(204)
// 			return
// 		}

// 		c.Next()
// 	})
// }

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
