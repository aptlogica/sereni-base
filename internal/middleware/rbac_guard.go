// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@sereni-base.com

package middleware

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"
	"github.com/gin-gonic/gin"
)

// Guard base authorization interface
type Guard interface {
	Check(c *gin.Context) (bool, error)
	Middleware() gin.HandlerFunc
}

// PermissionGuard checks resource+action permission
type PermissionGuard interface {
	Guard
}

// RoleGuard checks required roles
type RoleGuard interface {
	Guard
}

// DefaultMiddleware converts Guard to Gin middleware
func DefaultMiddleware(g Guard) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, err := g.Check(c)
		if !allowed {
			if err != nil {
				response.CheckAndSendError(c, err)
			} else {
				response.SendError(c, responseConst.Error.UnauthorizedAccess)
			}
			c.Abort()
			return
		}
		c.Next()
	}
}

// GuardContextKeys holds standardized context key names
type GuardContextKeys struct {
	UserIDKey string
	SchemaKey string
}

// GuardContext provides standardized context key management
var GuardContext = GuardContextKeys{
	UserIDKey: "user_id",
	SchemaKey: "schema",
}

// UserInfo contains extracted user context from request
type UserInfo struct {
	UserID string
	Schema string
}

// ScopeInfo contains extracted scope context from request headers
type ScopeInfo struct {
	ScopeType string
	ScopeID   string
}

// ExtractScopeInfosFromDatabase returns all scope records assigned to the user.
func ExtractScopeInfosFromDatabase(
	c *gin.Context,
	userInfo *UserInfo,
	accessMemberSvc interfaces.AccessMemberService,
) []ScopeInfo {
	accessMembers, err := accessMemberSvc.GetUserAccessMembers(
		c.Request.Context(),
		userInfo.Schema,
		userInfo.UserID,
	)
	if err != nil || len(accessMembers) == 0 {
		return []ScopeInfo{{
			ScopeType: "workspace",
			ScopeID:   "",
		}}
	}

	scopeInfos := make([]ScopeInfo, 0, len(accessMembers))
	for _, member := range accessMembers {
		scopeID := ""
		if member.ScopeID != nil {
			scopeID = *member.ScopeID
		}
		scopeInfos = append(scopeInfos, ScopeInfo{
			ScopeType: member.ScopeType,
			ScopeID:   scopeID,
		})
	}

	return scopeInfos
}

// ExtractUserInfo extracts user information from gin context
// These values are set by AuthMiddleware during token validation
func ExtractUserInfo(c *gin.Context) (*UserInfo, error) {
	userIDVal, ok := c.Get(GuardContext.UserIDKey)
	if !ok {
		return nil, dto.InvalidContextError(GuardContext.UserIDKey)
	}
	userID, _ := userIDVal.(string)

	schemaVal, ok := c.Get(GuardContext.SchemaKey)
	if !ok {
		return nil, dto.InvalidContextError(GuardContext.SchemaKey)
	}
	schema, _ := schemaVal.(string)

	return &UserInfo{
		UserID: userID,
		Schema: schema,
	}, nil
}

// permissionGuardImpl implements PermissionGuard interface
type permissionGuardImpl struct {
	resourceCode    string
	actionCode      string
	accessMemberSvc interfaces.AccessMemberService
}

// NewPermissionGuard creates permission-based guard middleware
func NewPermissionGuard(
	resourceCode string,
	actionCode string,
	accessMemberSvc interfaces.AccessMemberService,
) PermissionGuard {
	return &permissionGuardImpl{
		resourceCode:    resourceCode,
		actionCode:      actionCode,
		accessMemberSvc: accessMemberSvc,
	}
}

// Check validates user has required permission
func (pg *permissionGuardImpl) Check(c *gin.Context) (bool, error) {
	userInfo, err := ExtractUserInfo(c)
	if err != nil {
		return false, err
	}

	// Evaluate permission across all scopes assigned to the user.
	for _, scopeInfo := range ExtractScopeInfosFromDatabase(c, userInfo, pg.accessMemberSvc) {
		hasPermission, checkErr := pg.accessMemberSvc.CheckUserPermission(
			c.Request.Context(),
			userInfo.Schema,
			userInfo.UserID,
			scopeInfo.ScopeType,
			&scopeInfo.ScopeID,
			pg.resourceCode,
			pg.actionCode,
		)
		if checkErr != nil {
			return false, checkErr
		}
		if hasPermission {
			return true, nil
		}
	}

	return false, nil
}

// Middleware returns the Gin handler for this guard
func (pg *permissionGuardImpl) Middleware() gin.HandlerFunc {
	return DefaultMiddleware(pg)
}

// roleGuardImpl implements RoleGuard interface
type roleGuardImpl struct {
	requiredRoles   []string
	accessMemberSvc interfaces.AccessMemberService
	scopeType       string
}

// NewRoleGuard creates role-based guard middleware
// If scopeType is empty, checks user role across all scopes (from database)
// If scopeType is provided, checks user role for that specific scope only
func NewRoleGuard(
	requiredRoles []string,
	accessMemberSvc interfaces.AccessMemberService,
	scopeType string,
) RoleGuard {
	return &roleGuardImpl{
		requiredRoles:   requiredRoles,
		accessMemberSvc: accessMemberSvc,
		scopeType:       scopeType,
	}
}

// Check validates user has required role
func (rg *roleGuardImpl) Check(c *gin.Context) (bool, error) {
	userInfo, err := ExtractUserInfo(c)
	if err != nil {
		return false, err
	}

	for _, scopeInfo := range ExtractScopeInfosFromDatabase(c, userInfo, rg.accessMemberSvc) {
		checkScopeType := rg.scopeType
		if checkScopeType == "" {
			checkScopeType = scopeInfo.ScopeType
		}

		userRole, roleErr := rg.accessMemberSvc.GetUserHighestRole(
			c.Request.Context(),
			userInfo.Schema,
			userInfo.UserID,
			checkScopeType,
			&scopeInfo.ScopeID,
		)
		if roleErr != nil || userRole == nil {
			continue
		}

		for _, req := range rg.requiredRoles {
			if userRole.Name == req {
				return true, nil
			}
		}
	}

	return false, nil
}

// Middleware returns the Gin handler for this guard
func (rg *roleGuardImpl) Middleware() gin.HandlerFunc {
	return DefaultMiddleware(rg)
}
