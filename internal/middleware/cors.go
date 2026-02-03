package middleware

import (
	"fmt"
	"strconv"
	"strings"

	"serenibase/internal/config"

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
			fmt.Println("allowedOrigin:", allowedOrigin)
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
