package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
)

// RBACManagementService provides a unified interface for all RBAC operations
type RBACManagementService interface {
	// System initialization
	InitializeRBACSystem(ctx context.Context, schema string) error
	GetRBACSystemStatus(ctx context.Context, schemaName string) (dto.RBACSystemStatus, error)

	// Role management
	CreateRole(ctx context.Context, schemaName string, req dto.AccessRoleDTO) (tenant.AccessRole, error)
	GetRoleByID(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error)
	GetRoleByName(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error)
	GetRolesByScope(ctx context.Context, schemaName string, scopeLevel string) ([]tenant.AccessRole, error)
	ListRoles(ctx context.Context, schemaName string, limit, offset int) ([]tenant.AccessRole, int64, error)
	UpdateRole(ctx context.Context, schemaName string, roleID uuid.UUID, req dto.AccessRoleDTO) (tenant.AccessRole, error)
	DeleteRole(ctx context.Context, schemaName string, roleID uuid.UUID) error
	CountRoles(ctx context.Context, schemaName string) (int64, error)

	// Resource management
	CreateResource(ctx context.Context, schemaName string, req dto.ResourceDTO) (tenant.Resource, error)
	GetResourceByID(ctx context.Context, schemaName string, resourceID uuid.UUID) (tenant.Resource, error)
	GetResourceByCode(ctx context.Context, schemaName string, code string) (tenant.Resource, error)
	ListResources(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Resource, int64, error)
	UpdateResource(ctx context.Context, schemaName string, resourceID uuid.UUID, req dto.ResourceDTO) (tenant.Resource, error)
	DeleteResource(ctx context.Context, schemaName string, resourceID uuid.UUID) error
	GetOrCreateResource(ctx context.Context, schemaName string, code string, description *string) (tenant.Resource, error)
	CountResources(ctx context.Context, schemaName string) (int64, error)

	// Action management
	CreateAction(ctx context.Context, schemaName string, req dto.ActionDTO) (tenant.Action, error)
	GetActionByID(ctx context.Context, schemaName string, actionID uuid.UUID) (tenant.Action, error)
	GetActionByCode(ctx context.Context, schemaName string, code string) (tenant.Action, error)
	ListActions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Action, int64, error)
	UpdateAction(ctx context.Context, schemaName string, actionID uuid.UUID, req dto.ActionDTO) (tenant.Action, error)
	DeleteAction(ctx context.Context, schemaName string, actionID uuid.UUID) error
	GetOrCreateAction(ctx context.Context, schemaName string, code string, description *string) (tenant.Action, error)
	CountActions(ctx context.Context, schemaName string) (int64, error)

	// Permission management
	CreatePermission(ctx context.Context, schemaName string, req dto.PermissionDTO) (tenant.Permission, error)
	GetPermissionByID(ctx context.Context, schemaName string, permissionID uuid.UUID) (tenant.Permission, error)
	GetPermissionByResourceAndAction(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error)
	ListPermissions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Permission, int64, error)
	DeletePermission(ctx context.Context, schemaName string, permissionID uuid.UUID) error
	GetOrCreatePermission(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error)
	GetPermissionsByResource(ctx context.Context, schemaName string, resourceID uuid.UUID) ([]tenant.Permission, error)
	CountPermissions(ctx context.Context, schemaName string) (int64, error)

	// Role-Permission mapping
	AssignPermissionToRole(ctx context.Context, schemaName string, req dto.RolePermissionDTO) (tenant.RolePermission, error)
	RemovePermissionFromRole(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) error
	GetRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) ([]tenant.RolePermission, error)
	GetPermissionsByRole(ctx context.Context, schemaName string, roleID uuid.UUID) ([]dto.PermissionWithDetails, error)
	GetRolesByPermission(ctx context.Context, schemaName string, permissionID uuid.UUID) ([]tenant.AccessRole, error)
	CheckRoleHasPermission(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) (bool, error)
	BulkAssignPermissionsToRole(ctx context.Context, schemaName string, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	CountRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) (int64, error)

	// User access management (tenant-scoped)
	AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error)
	RemoveRoleFromUser(ctx context.Context, schemaName string, userID, scopeID string, scopeType string) error
	GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error)
	GetUserAccessByScope(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error)
	GetScopeMembers(ctx context.Context, schemaName string, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error)

	// User permission checks
	GetUserPermissions(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.PermissionWithDetails, error)
	CheckUserPermission(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, resourceCode, actionCode string) (bool, error)
	GetUserHighestRole(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) (*dto.AccessRoleDTO, error)

	// Bulk operations
	BulkAssignRoleToUsers(ctx context.Context, schemaName string, req dto.BulkAssignRoleRequest) error
	BulkRemoveRoleFromUsers(ctx context.Context, schemaName string, userIDs []string, scopeType string, scopeID *string, roleID string) error

	// RBAC analytics and reporting
	GetRBACAnalytics(ctx context.Context, schemaName string) (dto.RBACAnalytics, error)
	GetRoleUsageStats(ctx context.Context, schemaName string, roleID uuid.UUID) (dto.RoleUsageStats, error)
	GetPermissionUsageStats(ctx context.Context, schemaName string, permissionID uuid.UUID) (dto.PermissionUsageStats, error)
	GetResourceAccessMatrix(ctx context.Context, schemaName string) ([]dto.ResourceAccessMatrix, error)

	// Validation and auditing
	ValidateRoleConfiguration(ctx context.Context, schemaName string, roleID uuid.UUID) (dto.RoleValidationResult, error)
	AuditUserAccess(ctx context.Context, schemaName string, userID string) (dto.UserAccessAudit, error)
	GetOrphanedPermissions(ctx context.Context, schemaName string) ([]tenant.Permission, error)
	GetUnusedRoles(ctx context.Context, schemaName string) ([]tenant.AccessRole, error)

	// Membership processing
	ProcessUserMemberships(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error)
}
