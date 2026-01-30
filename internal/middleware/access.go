package middleware

import (
	"fmt"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"
	"strings"

	"github.com/gin-gonic/gin"
)

// ScopeLevel constants
const (
	ScopeWorkspace = "workspace"
	ScopeBase      = "base"
)

// Header constants
const (
	HeaderScopeType = "scope-type"
	HeaderScopeID   = "scope-id"
)

func ScopeHeaderMiddleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("ScopeHeaderMiddleware-------------------")
		c.Request.Header.Set("Scope", scope)
		c.Next()
	}
}

func WorkspaceAndBaseAccessValidationMiddleware(workspaceMemberService interfaces.WorkspaceMemberService, allowedAccess []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		strSchema, strUserId, ok := validateUserAndSchema(c)
		if !ok {
			return
		}

		if !validateWorkspaceAccess(c, workspaceMemberService, strSchema, strUserId, allowedAccess) {
			return
		}

		workspaceMemberData, _ := c.Get("workspaceMemberData")

		if !validateBaseAccess(c, workspaceMemberData.(tenant.WorkspaceMember)) {
			return
		}

		c.Next()
	}
}

func validateUserAndSchema(c *gin.Context) (string, string, bool) {
	userId, hasUser := c.Get("user_id")
	schema, hasSchema := c.Get("schema")
	if !hasUser || !hasSchema {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		c.Abort()
		return "", "", false
	}
	strSchema, _ := schema.(string)
	strUserId, _ := userId.(string)
	return strSchema, strUserId, true
}

func validateWorkspaceAccess(c *gin.Context, workspaceMemberService interfaces.WorkspaceMemberService, strSchema, strUserId string, allowedAccess []string) bool {
	workspaceID := c.GetHeader("workspace")
	if workspaceID == "" {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		c.Abort()
		return false
	}

	workspaceMemberData, err := workspaceMemberService.GetWorkspaceMemberByUserAndWorkspace(
		c.Request.Context(),
		strSchema,
		strUserId,
		workspaceID,
	)

	if err != nil {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		c.Abort()
		return false
	}

	// Check allowed access
	accessAllowed := false
	for _, a := range allowedAccess {
		if a == workspaceMemberData.AccessLevel {
			accessAllowed = true
			break
		}
	}
	if !accessAllowed {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		c.Abort()
		return false
	}

	c.Set("workspaceMemberData", workspaceMemberData)
	return true
}

func validateBaseAccess(c *gin.Context, workspaceMemberData tenant.WorkspaceMember) bool {
	scope, hasScope := c.Get("scope")
	_ = hasScope

	if scope == ScopeBase {
		baseID := c.GetHeader("base")
		if baseID == "" {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return false
		}

		if workspaceMemberData.BasesIds != "*" {
			baseIDs := strings.Split(workspaceMemberData.BasesIds, ",")
			baseAllowed := false
			for _, id := range baseIDs {
				if strings.TrimSpace(id) == baseID {
					baseAllowed = true
					break
				}
			}
			if !baseAllowed {
				response.SendError(c, responseConst.Error.UnauthorizedAccess)
				c.Abort()
				return false
			}
		}
	}
	return true
}

// ========== RBAC Middleware Functions ==========

// CheckPermissionMiddleware verifies if user has required permission for a resource-action combination
// This is the new RBAC-based permission check that supports granular permissions
func CheckPermissionMiddleware(accessMemberService interfaces.AccessMemberService, resourceCode, actionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, schemaStr, ok := extractUserAndSchemaFromContext(c)
		if !ok {
			return
		}

		scopeType, scopeID := extractScopeFromHeaders(c)

		// Check if user has permission
		hasPermission, err := accessMemberService.CheckUserPermission(
			c.Request.Context(),
			schemaStr,
			userIDStr,
			scopeType,
			&scopeID,
			resourceCode,
			actionCode,
		)

		if err != nil || !hasPermission {
			sendUnauthorizedError(c)
			return
		}

		c.Next()
	}
}

// CheckRoleMiddleware verifies if user has a specific role at a scope level
func CheckRoleMiddleware(accessMemberService interfaces.AccessMemberService, requiredRoles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, schemaStr, ok := extractUserAndSchemaFromContext(c)
		if !ok {
			return
		}

		scopeType, scopeID := extractScopeFromHeaders(c)

		// Get user's highest role for this scope
		highestRole, err := accessMemberService.GetUserHighestRole(
			c.Request.Context(),
			schemaStr,
			userIDStr,
			scopeType,
			&scopeID,
		)

		if err != nil {
			sendUnauthorizedError(c)
			return
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
			sendUnauthorizedError(c)
			return
		}

		c.Set("userRole", highestRole)
		c.Next()
	}
}

// ValidateAccessScopeMiddleware ensures user has access to the requested scope
// Supports both legacy workspace_members and new access_members
func ValidateAccessScopeMiddleware(accessMemberService interfaces.AccessMemberService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, schemaStr, ok := extractUserAndSchemaFromContext(c)
		if !ok {
			return
		}

		scopeType, scopeID := extractScopeFromHeaders(c)

		if scopeType == "" || scopeID == "" {
			sendUnauthorizedError(c)
			return
		}

		// Get user access for this scope
		accessMembers, err := accessMemberService.GetUserAccessByScope(
			c.Request.Context(),
			schemaStr,
			userIDStr,
			scopeType,
			&scopeID,
		)

		if err != nil || len(accessMembers) == 0 {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		// Store access member data for downstream handlers
		c.Set("accessMembers", accessMembers)
		c.Next()
	}
}

// RequirePermissionsMiddleware is a convenience middleware that checks multiple permissions
// Useful for operations that require multiple permissions
func RequirePermissionsMiddleware(accessMemberService interfaces.AccessMemberService, permissions []struct{ Resource, Action string }) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, schemaStr, ok := extractUserAndSchemaFromContext(c)
		if !ok {
			return
		}

		scopeType, scopeID := extractScopeFromHeaders(c)

		ctx := c.Request.Context()

		// Check all required permissions
		for _, perm := range permissions {
			hasPermission, err := accessMemberService.CheckUserPermission(
				ctx,
				schemaStr,
				userIDStr,
				scopeType,
				&scopeID,
				perm.Resource,
				perm.Action,
			)

			if err != nil || !hasPermission {
				sendUnauthorizedError(c)
				return
			}
		}

		c.Next()
	}
}
