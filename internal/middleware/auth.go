package middleware

import (
	"fmt"
	"serenibase/internal/providers/auth"
	"serenibase/internal/utils/response"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(authProviderService auth.AuthProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the Authorization header
		authHeader := c.GetHeader("Authorization")
		userClaims, err := authProviderService.ValidateToken(c.Request.Context(), authHeader)
		if err != nil {
			fmt.Println("ValidateToken err: ", err)
			response.CheckAndSendError(c, err)
			c.Abort()
			return
		}
		fmt.Println("userClaims: ", userClaims)
		c.Set("user_id", userClaims.UserId)
		c.Set("tenant_id", userClaims.TenantId)
		c.Set("roles", userClaims.Roles)
		c.Next()
	}
}
