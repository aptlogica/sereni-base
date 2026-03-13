// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"
	"strings"

	"github.com/gin-gonic/gin"
)

// PermissionRequest represents the parameters for a permission check
type PermissionRequest struct {
	Schema       string
	UserID       string
	ScopeType    string
	ScopeID      *string
	ResourceCode string
	ActionCode   string
}

// MiddlewareUtil provides common middleware utilities
type MiddlewareUtil struct{}

// NewMiddlewareUtil creates a new middleware utility
func NewMiddlewareUtil() *MiddlewareUtil {
	return &MiddlewareUtil{}
}

// extractUserAndSchemaFromContext extracts user_id and schema from context
func (mu *MiddlewareUtil) ExtractUserAndSchemaFromContext(c *gin.Context) (userID string, schema string, ok bool) {
	userId, hasUser := c.Get("user_id")
	schemaVal, hasSchema := c.Get("schema")
	if !hasUser || !hasSchema {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		c.Abort()
		return "", "", false
	}
	strSchema, _ := schemaVal.(string)
	strUserId, _ := userId.(string)
	return strUserId, strSchema, true
}

// extractScopeFromHeaders extracts scope type and ID from headers
func (mu *MiddlewareUtil) ExtractScopeFromHeaders(c *gin.Context) (scopeType string, scopeID string) {
	scopeType = c.GetHeader(HeaderScopeType)
	scopeID = c.GetHeader(HeaderScopeID)
	return
}

// sendUnauthorizedError sends unauthorized error response
func (mu *MiddlewareUtil) SendUnauthorizedError(c *gin.Context) {
	response.SendError(c, responseConst.Error.UnauthorizedAccess)
	c.Abort()
}

// ValidateBaseAccess validates if user has access to the requested base
func (mu *MiddlewareUtil) ValidateBaseAccess(c *gin.Context, workspaceMemberData interface{}, scopeType string) bool {
	if scopeType == ScopeBase {
		baseID := c.GetHeader("base")
		if baseID == "" {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return false
		}

		// For workspace members with BasesIds field
		if wm, ok := workspaceMemberData.(interface {
			GetBasesIds() string
		}); ok {
			if !mu.isBaseAllowed(baseID, wm.GetBasesIds()) {
				response.SendError(c, responseConst.Error.UnauthorizedAccess)
				c.Abort()
				return false
			}
		}
	}
	return true
}

// isBaseAllowed checks if the baseID is allowed based on basesIds string
func (mu *MiddlewareUtil) isBaseAllowed(baseID, basesIds string) bool {
	if basesIds == "*" {
		return true
	}
	baseIDs := strings.Split(basesIds, ",")
	for _, id := range baseIDs {
		if strings.TrimSpace(id) == baseID {
			return true
		}
	}
	return false
}

// CheckAccessLevel validates if user has required access level
func (mu *MiddlewareUtil) CheckAccessLevel(c *gin.Context, userAccessLevel string, allowedAccess []string) bool {
	accessAllowed := false
	for _, a := range allowedAccess {
		if a == userAccessLevel {
			accessAllowed = true
			break
		}
	}

	if !accessAllowed {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		c.Abort()
		return false
	}
	return true
}

// CheckUserPermission checks if user has required permission
func (mu *MiddlewareUtil) CheckUserPermission(c *gin.Context, accessMemberService interfaces.AccessMemberService, req PermissionRequest) bool {

	hasPermission, err := accessMemberService.CheckUserPermission(
		c.Request.Context(),
		req.Schema,
		req.UserID,
		req.ScopeType,
		req.ScopeID,
		req.ResourceCode,
		req.ActionCode,
	)

	if err != nil || !hasPermission {
		mu.SendUnauthorizedError(c)
		return false
	}
	return true
}

// CheckUserRole checks if user has required role
func (mu *MiddlewareUtil) CheckUserRole(c *gin.Context, accessMemberService interfaces.AccessMemberService,
	schemaStr, userIDStr, scopeType string, scopeID *string, requiredRoles []string) bool {

	highestRole, err := accessMemberService.GetUserHighestRole(
		c.Request.Context(),
		schemaStr,
		userIDStr,
		scopeType,
		scopeID,
	)

	if err != nil {
		mu.SendUnauthorizedError(c)
		return false
	}

	// Check if user's role is in required roles
	hasRole := false
	for _, role := range requiredRoles {
		if role == highestRole.Name {
			hasRole = true
			break
		}
	}

	if !hasRole {
		mu.SendUnauthorizedError(c)
		return false
	}

	c.Set("userRole", highestRole)
	return true
}
