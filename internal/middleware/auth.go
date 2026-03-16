// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"fmt"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/providers/auth"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

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
		fmt.Println("err ValidateToken:::::::", err)
		fmt.Println("err userClaims:::::::", userClaims)
		if err != nil {
			response.CheckAndSendError(c, err)
			c.Abort()
			return
		}

		// Check if user exists in database
		user, err := userManagementService.GetUserByID(c.Request.Context(), constant.MasterDatabase, userClaims.UserId)
		if err != nil {
			fmt.Println("err GetUserByID:::::::", err)
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
		fmt.Println("userClaims.UserId: ", userClaims.UserId)

		c.Set("user_id", userClaims.UserId)
		c.Set("schema", constant.MasterDatabase)
		c.Set("roles", userClaims.Roles)
		c.Next()
	}
}
