package middleware

import (
	"serenibase/internal/constant"
	"serenibase/internal/providers/auth"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(authProviderService auth.AuthProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}
		userClaims, err := authProviderService.ValidateToken(c.Request.Context(), authHeader)
		if err != nil {
			response.CheckAndSendError(c, err)
			c.Abort()
			return
		}
		c.Set("user_id", userClaims.UserId)
		c.Set("schema", constant.MasterDatabase)
		c.Set("roles", userClaims.Roles)
		c.Next()
	}
}
