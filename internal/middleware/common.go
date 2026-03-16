// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

// extractUserAndSchemaFromContext extracts user_id and schema from context
// Returns (userID, schema, success)
func extractUserAndSchemaFromContext(c *gin.Context) (string, string, bool) {
	userID, hasUser := c.Get("user_id")
	schema, hasSchema := c.Get("schema")

	if !hasUser || !hasSchema {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		c.Abort()
		return "", "", false
	}

	userIDStr, _ := userID.(string)
	schemaStr, _ := schema.(string)
	return userIDStr, schemaStr, true
}

// extractScopeFromHeaders extracts scope type and ID from headers
// If scopeType is not provided, defaults to Workspace
func extractScopeFromHeaders(c *gin.Context) (string, string) {
	scopeType := c.GetHeader(HeaderScopeType)
	if scopeType == "" {
		scopeType = constant.ScopeLevels.Workspace
	}

	scopeID := c.GetHeader(HeaderScopeID)
	return scopeType, scopeID
}

// sendUnauthorizedError sends unauthorized error and aborts
func sendUnauthorizedError(c *gin.Context) {
	response.SendError(c, responseConst.Error.UnauthorizedAccess)
	c.Abort()
}
