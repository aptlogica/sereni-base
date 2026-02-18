package middleware

import (
	"serenibase/internal/constant"
	"serenibase/internal/providers/auth"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(authProviderService auth.AuthProvider, userManagementService interfaces.UserManagementService) gin.HandlerFunc {
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

		// Check if user exists in database
		user, err := userManagementService.GetUserByID(c.Request.Context(), constant.MasterDatabase, userClaims.UserId)
		if err != nil {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		// Check if user status is "active"
		if user.Status != "active" {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		c.Set("user_id", userClaims.UserId)
		c.Set("schema", constant.MasterDatabase)
		c.Set("roles", userClaims.Roles)
		c.Next()
	}
}
