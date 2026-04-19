// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/google/uuid"
)

// AccessRoleService manages role operations with scope-based access
type AccessRoleService interface {
	// Role operations
	CreateAccessRole(ctx context.Context, schemaName string, req dto.AccessRoleDTO) (tenant.AccessRole, error)
	GetAccessRoleByID(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error)
	GetAccessRoleByName(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error)
	GetAccessRolesByScope(ctx context.Context, schemaName string, scopeLevel string) ([]tenant.AccessRole, error)
	ListAccessRoles(ctx context.Context, schemaName string, limit, offset int) ([]tenant.AccessRole, int64, error)
	UpdateAccessRole(ctx context.Context, schemaName string, roleID uuid.UUID, req dto.AccessRoleDTO) (tenant.AccessRole, error)
	DeleteAccessRole(ctx context.Context, schemaName string, roleID uuid.UUID) error
}

// ResourceService manages resource operations
type ResourceService interface {
	CreateResource(ctx context.Context, schemaName string, req dto.ResourceDTO) (tenant.Resource, error)
	GetResourceByID(ctx context.Context, schemaName string, resourceID uuid.UUID) (tenant.Resource, error)
	GetResourceByCode(ctx context.Context, schemaName string, code string) (tenant.Resource, error)
	ListResources(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Resource, int64, error)
	UpdateResource(ctx context.Context, schemaName string, resourceID uuid.UUID, req dto.ResourceDTO) (tenant.Resource, error)
	DeleteResource(ctx context.Context, schemaName string, resourceID uuid.UUID) error
	GetOrCreateResource(ctx context.Context, schemaName string, code string, description *string) (tenant.Resource, error)
}

// ActionService manages action operations
type ActionService interface {
	CreateAction(ctx context.Context, schemaName string, req dto.ActionDTO) (tenant.Action, error)
	GetActionByID(ctx context.Context, schemaName string, actionID uuid.UUID) (tenant.Action, error)
	GetActionByCode(ctx context.Context, schemaName string, code string) (tenant.Action, error)
	ListActions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Action, int64, error)
	UpdateAction(ctx context.Context, schemaName string, actionID uuid.UUID, req dto.ActionDTO) (tenant.Action, error)
	DeleteAction(ctx context.Context, schemaName string, actionID uuid.UUID) error
	GetOrCreateAction(ctx context.Context, schemaName string, code string, description *string) (tenant.Action, error)
}

// PermissionService manages permission operations
type PermissionService interface {
	CreatePermission(ctx context.Context, schemaName string, req dto.PermissionDTO) (tenant.Permission, error)
	GetPermissionByID(ctx context.Context, schemaName string, permissionID uuid.UUID) (tenant.Permission, error)
	GetPermissionByResourceAndAction(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error)
	ListPermissions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Permission, int64, error)
	DeletePermission(ctx context.Context, schemaName string, permissionID uuid.UUID) error
	GetOrCreatePermission(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error)
	GetPermissionsByResource(ctx context.Context, schemaName string, resourceID uuid.UUID) ([]tenant.Permission, error)
}

// RolePermissionService manages role-permission mappings
type RolePermissionService interface {
	AssignPermissionToRole(ctx context.Context, schemaName string, req dto.RolePermissionDTO) (tenant.RolePermission, error)
	RemovePermissionFromRole(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) error
	GetRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) ([]tenant.RolePermission, error)
	GetPermissionsByRole(ctx context.Context, schemaName string, roleID uuid.UUID) ([]dto.PermissionWithDetails, error)
	GetRolesByPermission(ctx context.Context, schemaName string, permissionID uuid.UUID) ([]tenant.AccessRole, error)
	CheckRoleHasPermission(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) (bool, error)
}

// AccessMemberService manages user access assignments across scopes
type AccessMemberService interface {
	// User access assignment
	AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error)
	RemoveRoleFromUser(ctx context.Context, schemaName string, userID, scopeID string, scopeType string) error
	RemoveAccessMemberByID(ctx context.Context, schemaName string, memberID string) error
	UpdateRoleForUser(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, newRoleID string) error
	GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error)
	GetUserAccessByScope(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error)
	GetScopeMembers(ctx context.Context, schemaName string, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error)

	// User permissions
	GetUserPermissions(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.PermissionWithDetails, error)
	CheckUserPermission(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, resourceCode, actionCode string) (bool, error)
	GetUserHighestRole(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) (*dto.AccessRoleDTO, error)

	// Bulk operations
	BulkAssignRoleToUsers(ctx context.Context, schemaName string, req dto.BulkAssignRoleRequest) error
	BulkRemoveRoleFromUsers(ctx context.Context, schemaName string, userIDs []string, scopeType string, scopeID *string, roleID string) error
}
