package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	dbModels "godbgrest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"strings"

	"github.com/google/uuid"
)

type rbacManagementService struct {
	repo                  *pkg.DatabaseService
	roleService           interfaces.AccessRoleService
	resourceService       interfaces.ResourceService
	actionService         interfaces.ActionService
	permissionService     interfaces.PermissionService
	rolePermissionService interfaces.RolePermissionService
	accessMemberService   interfaces.AccessMemberService
	baseService           interfaces.BaseService
}

// NewRBACManagementService creates a new RBAC management service that consolidates all RBAC operations
func NewRBACManagementService(
	repo *pkg.DatabaseService,
	roleService interfaces.AccessRoleService,
	resourceService interfaces.ResourceService,
	actionService interfaces.ActionService,
	permissionService interfaces.PermissionService,
	rolePermissionService interfaces.RolePermissionService,
	accessMemberService interfaces.AccessMemberService,
	baseService interfaces.BaseService,
) interfaces.RBACManagementService {
	return &rbacManagementService{
		repo:                  repo,
		roleService:           roleService,
		resourceService:       resourceService,
		actionService:         actionService,
		permissionService:     permissionService,
		rolePermissionService: rolePermissionService,
		accessMemberService:   accessMemberService,
		baseService:           baseService,
	}
}

// ==================== System Initialization ====================

func (s *rbacManagementService) InitializeRBACSystem(ctx context.Context, schema string) error {
	fmt.Println("Initializing RBAC System...")

	// Step 1: Create default resources
	fmt.Println("Creating resources...")
	resourceMap := make(map[string]uuid.UUID)
	resources := []struct {
		code string
		desc string
	}{
		{constant.ResourceCodes.Workspace, "Workspace resource"},
		{constant.ResourceCodes.Base, "Base resource"},
		{constant.ResourceCodes.Table, "Table resource"},
		{constant.ResourceCodes.Records, "Records resource"},
		{constant.ResourceCodes.Members, "Members resource"},
		{constant.ResourceCodes.Views, "Views resource"},
		{constant.ResourceCodes.Settings, "Settings resource"},
		{constant.ResourceCodes.ApiTokens, "API Tokens resource"},
		{constant.ResourceCodes.Webhooks, "Webhooks resource"},
		{constant.ResourceCodes.Automations, "Automations resource"},
	}

	for _, r := range resources {
		desc := r.desc
		resource, err := s.resourceService.GetOrCreateResource(ctx, schema, r.code, &desc)
		if err != nil {
			fmt.Printf("Error creating resource %s: %v\n", r.code, err)
			continue
		}
		resourceMap[r.code] = resource.ID
		fmt.Printf("✓ Created resource: %s\n", r.code)
	}

	// Step 2: Create default actions
	fmt.Println("\nCreating actions...")
	actionMap := make(map[string]uuid.UUID)
	actions := []struct {
		code string
		desc string
	}{
		{constant.ActionCodes.Read, "Read access"},
		{constant.ActionCodes.Create, "Create access"},
		{constant.ActionCodes.Update, "Update access"},
		{constant.ActionCodes.Delete, "Delete access"},
		{constant.ActionCodes.Share, "Share access"},
		{constant.ActionCodes.Invite, "Invite access"},
		{constant.ActionCodes.Export, "Export access"},
		{constant.ActionCodes.Import, "Import access"},
		{constant.ActionCodes.Execute, "Execute access"},
		{constant.ActionCodes.Manage, "Manage access"},
	}

	for _, a := range actions {
		desc := a.desc
		action, err := s.actionService.GetOrCreateAction(ctx, schema, a.code, &desc)
		if err != nil {
			fmt.Printf("Error creating action %s: %v\n", a.code, err)
			continue
		}
		actionMap[a.code] = action.ID
		fmt.Printf("✓ Created action: %s\n", a.code)
	}

	// Step 3: Create default roles
	fmt.Println("\nCreating roles...")
	roleMap := make(map[string]uuid.UUID)
	for _, roleReq := range constant.DefaultAccessRoles {
		role, err := s.roleService.CreateAccessRole(ctx, schema, roleReq)
		if err != nil {
			fmt.Printf("Error creating role %s: %v\n", roleReq.Name, err)
			continue
		}
		roleMap[roleReq.Name] = role.ID
		fmt.Printf("✓ Created role: %s (scope: %s, priority: %d)\n", roleReq.Name, roleReq.ScopeLevel, roleReq.Priority)
	}

	// Step 4: Create permissions (resource × action)
	fmt.Println("\nCreating permissions...")
	permissionMap := make(map[string]uuid.UUID)

	permissionCombinations := []struct {
		resource string
		action   string
	}{
		// Workspace permissions
		{constant.ResourceCodes.Workspace, constant.ActionCodes.Read},
		{constant.ResourceCodes.Workspace, constant.ActionCodes.Create},
		{constant.ResourceCodes.Workspace, constant.ActionCodes.Update},
		{constant.ResourceCodes.Workspace, constant.ActionCodes.Delete},
		{constant.ResourceCodes.Workspace, constant.ActionCodes.Share},
		{constant.ResourceCodes.Workspace, constant.ActionCodes.Invite},

		// Base permissions
		{constant.ResourceCodes.Base, constant.ActionCodes.Read},
		{constant.ResourceCodes.Base, constant.ActionCodes.Create},
		{constant.ResourceCodes.Base, constant.ActionCodes.Update},
		{constant.ResourceCodes.Base, constant.ActionCodes.Delete},

		// Records permissions
		{constant.ResourceCodes.Records, constant.ActionCodes.Read},
		{constant.ResourceCodes.Records, constant.ActionCodes.Create},
		{constant.ResourceCodes.Records, constant.ActionCodes.Update},
		{constant.ResourceCodes.Records, constant.ActionCodes.Delete},
		{constant.ResourceCodes.Records, constant.ActionCodes.Export},

		// Members permissions
		{constant.ResourceCodes.Members, constant.ActionCodes.Read},
		{constant.ResourceCodes.Members, constant.ActionCodes.Invite},
		{constant.ResourceCodes.Members, constant.ActionCodes.Manage},

		// Views permissions
		{constant.ResourceCodes.Views, constant.ActionCodes.Read},
		{constant.ResourceCodes.Views, constant.ActionCodes.Create},
		{constant.ResourceCodes.Views, constant.ActionCodes.Update},
		{constant.ResourceCodes.Views, constant.ActionCodes.Delete},

		// Settings permissions
		{constant.ResourceCodes.Settings, constant.ActionCodes.Read},
		{constant.ResourceCodes.Settings, constant.ActionCodes.Update},

		// API Tokens permissions
		{constant.ResourceCodes.ApiTokens, constant.ActionCodes.Read},
		{constant.ResourceCodes.ApiTokens, constant.ActionCodes.Create},
		{constant.ResourceCodes.ApiTokens, constant.ActionCodes.Delete},
		{constant.ResourceCodes.ApiTokens, constant.ActionCodes.Manage},

		// Webhooks permissions
		{constant.ResourceCodes.Webhooks, constant.ActionCodes.Read},
		{constant.ResourceCodes.Webhooks, constant.ActionCodes.Create},
		{constant.ResourceCodes.Webhooks, constant.ActionCodes.Update},
		{constant.ResourceCodes.Webhooks, constant.ActionCodes.Delete},
	}

	for _, combo := range permissionCombinations {
		resourceID, ok := resourceMap[combo.resource]
		if !ok {
			continue
		}
		actionID, ok := actionMap[combo.action]
		if !ok {
			continue
		}

		permission, err := s.permissionService.GetOrCreatePermission(ctx, schema, resourceID, actionID)
		if err != nil {
			fmt.Printf("Error creating permission %s.%s: %v\n", combo.resource, combo.action, err)
			continue
		}
		permissionMap[fmt.Sprintf("%s.%s", combo.resource, combo.action)] = permission.ID
		fmt.Printf("✓ Created permission: %s.%s\n", combo.resource, combo.action)
	}

	// Step 5: Assign permissions to roles
	fmt.Println("\nAssigning permissions to roles...")

	// Owner role permissions (all workspace and base permissions)
	ownerPermissions := []string{
		"workspace.read", "workspace.create", "workspace.update", "workspace.delete", "workspace.share", "workspace.invite",
		"base.read", "base.create", "base.update", "base.delete",
		"records.read", "records.create", "records.update", "records.delete", "records.export",
		"members.read", "members.invite", "members.manage",
		"views.read", "views.create", "views.update", "views.delete",
		"settings.read", "settings.update",
		"api_tokens.read", "api_tokens.create", "api_tokens.delete", "api_tokens.manage",
		"webhooks.read", "webhooks.create", "webhooks.update", "webhooks.delete",
	}

	ownerRoleID, ok := roleMap[constant.RBACRoleNames.Owner]
	if ok {
		for _, permName := range ownerPermissions {
			if permID, ok := permissionMap[permName]; ok {
				_, err := s.rolePermissionService.AssignPermissionToRole(ctx, schema, dto.RolePermissionDTO{
					ID:           uuid.New(),
					RoleID:       ownerRoleID,
					PermissionID: permID,
				})
				if err != nil {
					fmt.Printf("Error assigning permission %s to owner: %v\n", permName, err)
					continue
				}
				fmt.Printf("✓ Assigned permission %s to owner\n", permName)
			}
		}
	}

	// Base Member role permissions
	memberPermissions := []string{
		"base.read",
		"records.read", "records.create", "records.update", "records.delete", "records.export",
		"views.read", "views.create", "views.update", "views.delete",
	}

	memberRoleID, ok := roleMap[constant.RBACRoleNames.BaseMember]
	if ok {
		for _, permName := range memberPermissions {
			if permID, ok := permissionMap[permName]; ok {
				_, err := s.rolePermissionService.AssignPermissionToRole(ctx, schema, dto.RolePermissionDTO{
					ID:           uuid.New(),
					RoleID:       memberRoleID,
					PermissionID: permID,
				})
				if err != nil {
					fmt.Printf("Error assigning permission %s to member: %v\n", permName, err)
					continue
				}
				fmt.Printf("✓ Assigned permission %s to member\n", permName)
			}
		}
	}

	fmt.Println("\n✓ RBAC System initialization completed successfully!")
	return nil
}

func (s *rbacManagementService) GetRBACSystemStatus(ctx context.Context, schemaName string) (dto.RBACSystemStatus, error) {
	roleCount, err := s.CountRoles(ctx, schemaName)
	if err != nil {
		roleCount = 0
	}

	resourceCount, err := s.CountResources(ctx, schemaName)
	if err != nil {
		resourceCount = 0
	}

	actionCount, err := s.CountActions(ctx, schemaName)
	if err != nil {
		actionCount = 0
	}

	permissionCount, err := s.CountPermissions(ctx, schemaName)
	if err != nil {
		permissionCount = 0
	}

	status := "healthy"
	if roleCount == 0 || resourceCount == 0 || actionCount == 0 || permissionCount == 0 {
		status = "not_initialized"
	}

	return dto.RBACSystemStatus{
		Initialized:         roleCount > 0 && resourceCount > 0 && actionCount > 0,
		TotalRoles:          roleCount,
		TotalResources:      resourceCount,
		TotalActions:        actionCount,
		TotalPermissions:    permissionCount,
		DefaultRolesCreated: roleCount >= 3, // at least owner, member, viewer
		Status:              status,
	}, nil
}

// ==================== Role Management ====================

func (s *rbacManagementService) CreateRole(ctx context.Context, schemaName string, req dto.AccessRoleDTO) (tenant.AccessRole, error) {
	return s.roleService.CreateAccessRole(ctx, schemaName, req)
}

func (s *rbacManagementService) GetRoleByID(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
	return s.roleService.GetAccessRoleByID(ctx, schemaName, roleID)
}

func (s *rbacManagementService) GetRoleByName(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
	return s.roleService.GetAccessRoleByName(ctx, schemaName, name)
}

func (s *rbacManagementService) GetRolesByScope(ctx context.Context, schemaName string, scopeLevel string) ([]tenant.AccessRole, error) {
	return s.roleService.GetAccessRolesByScope(ctx, schemaName, scopeLevel)
}

func (s *rbacManagementService) ListRoles(ctx context.Context, schemaName string, limit, offset int) ([]tenant.AccessRole, int64, error) {
	return s.roleService.ListAccessRoles(ctx, schemaName, limit, offset)
}

func (s *rbacManagementService) UpdateRole(ctx context.Context, schemaName string, roleID uuid.UUID, req dto.AccessRoleDTO) (tenant.AccessRole, error) {
	return s.roleService.UpdateAccessRole(ctx, schemaName, roleID, req)
}

func (s *rbacManagementService) DeleteRole(ctx context.Context, schemaName string, roleID uuid.UUID) error {
	return s.roleService.DeleteAccessRole(ctx, schemaName, roleID)
}

func (s *rbacManagementService) CountRoles(ctx context.Context, schemaName string) (int64, error) {
	tableName := tenant.AccessRole{}.TableName(schemaName)
	countQuery := dbModels.QueryParams{
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}
	countData, err := s.repo.TableService.GetTableData(ctx, tableName, countQuery)
	if err != nil {
		return 0, app_errors.DatabaseError
	}

	if len(countData) == 0 {
		return 0, nil
	}

	if total, ok := countData[0]["total"].(int64); ok {
		return total, nil
	}

	return 0, nil
}

// ==================== Resource Management ====================

func (s *rbacManagementService) CreateResource(ctx context.Context, schemaName string, req dto.ResourceDTO) (tenant.Resource, error) {
	return s.resourceService.CreateResource(ctx, schemaName, req)
}

func (s *rbacManagementService) GetResourceByID(ctx context.Context, schemaName string, resourceID uuid.UUID) (tenant.Resource, error) {
	return s.resourceService.GetResourceByID(ctx, schemaName, resourceID)
}

func (s *rbacManagementService) GetResourceByCode(ctx context.Context, schemaName string, code string) (tenant.Resource, error) {
	return s.resourceService.GetResourceByCode(ctx, schemaName, code)
}

func (s *rbacManagementService) ListResources(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Resource, int64, error) {
	return s.resourceService.ListResources(ctx, schemaName, limit, offset)
}

func (s *rbacManagementService) UpdateResource(ctx context.Context, schemaName string, resourceID uuid.UUID, req dto.ResourceDTO) (tenant.Resource, error) {
	return s.resourceService.UpdateResource(ctx, schemaName, resourceID, req)
}

func (s *rbacManagementService) DeleteResource(ctx context.Context, schemaName string, resourceID uuid.UUID) error {
	return s.resourceService.DeleteResource(ctx, schemaName, resourceID)
}

func (s *rbacManagementService) GetOrCreateResource(ctx context.Context, schemaName string, code string, description *string) (tenant.Resource, error) {
	return s.resourceService.GetOrCreateResource(ctx, schemaName, code, description)
}

func (s *rbacManagementService) CountResources(ctx context.Context, schemaName string) (int64, error) {
	tableName := tenant.Resource{}.TableName(schemaName)
	countQuery := dbModels.QueryParams{
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}
	countData, err := s.repo.TableService.GetTableData(ctx, tableName, countQuery)
	if err != nil {
		return 0, app_errors.DatabaseError
	}

	if len(countData) == 0 {
		return 0, nil
	}

	if total, ok := countData[0]["total"].(int64); ok {
		return total, nil
	}

	return 0, nil
}

// ==================== Action Management ====================

func (s *rbacManagementService) CreateAction(ctx context.Context, schemaName string, req dto.ActionDTO) (tenant.Action, error) {
	return s.actionService.CreateAction(ctx, schemaName, req)
}

func (s *rbacManagementService) GetActionByID(ctx context.Context, schemaName string, actionID uuid.UUID) (tenant.Action, error) {
	return s.actionService.GetActionByID(ctx, schemaName, actionID)
}

func (s *rbacManagementService) GetActionByCode(ctx context.Context, schemaName string, code string) (tenant.Action, error) {
	return s.actionService.GetActionByCode(ctx, schemaName, code)
}

func (s *rbacManagementService) ListActions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Action, int64, error) {
	return s.actionService.ListActions(ctx, schemaName, limit, offset)
}

func (s *rbacManagementService) UpdateAction(ctx context.Context, schemaName string, actionID uuid.UUID, req dto.ActionDTO) (tenant.Action, error) {
	return s.actionService.UpdateAction(ctx, schemaName, actionID, req)
}

func (s *rbacManagementService) DeleteAction(ctx context.Context, schemaName string, actionID uuid.UUID) error {
	return s.actionService.DeleteAction(ctx, schemaName, actionID)
}

func (s *rbacManagementService) GetOrCreateAction(ctx context.Context, schemaName string, code string, description *string) (tenant.Action, error) {
	return s.actionService.GetOrCreateAction(ctx, schemaName, code, description)
}

func (s *rbacManagementService) CountActions(ctx context.Context, schemaName string) (int64, error) {
	tableName := tenant.Action{}.TableName(schemaName)
	countQuery := dbModels.QueryParams{
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}
	countData, err := s.repo.TableService.GetTableData(ctx, tableName, countQuery)
	if err != nil {
		return 0, app_errors.DatabaseError
	}

	if len(countData) == 0 {
		return 0, nil
	}

	if total, ok := countData[0]["total"].(int64); ok {
		return total, nil
	}

	return 0, nil
}

// ==================== Permission Management ====================

func (s *rbacManagementService) CreatePermission(ctx context.Context, schemaName string, req dto.PermissionDTO) (tenant.Permission, error) {
	return s.permissionService.CreatePermission(ctx, schemaName, req)
}

func (s *rbacManagementService) GetPermissionByID(ctx context.Context, schemaName string, permissionID uuid.UUID) (tenant.Permission, error) {
	return s.permissionService.GetPermissionByID(ctx, schemaName, permissionID)
}

func (s *rbacManagementService) GetPermissionByResourceAndAction(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error) {
	return s.permissionService.GetPermissionByResourceAndAction(ctx, schemaName, resourceID, actionID)
}

func (s *rbacManagementService) ListPermissions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Permission, int64, error) {
	return s.permissionService.ListPermissions(ctx, schemaName, limit, offset)
}

func (s *rbacManagementService) DeletePermission(ctx context.Context, schemaName string, permissionID uuid.UUID) error {
	return s.permissionService.DeletePermission(ctx, schemaName, permissionID)
}

func (s *rbacManagementService) GetOrCreatePermission(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error) {
	return s.permissionService.GetOrCreatePermission(ctx, schemaName, resourceID, actionID)
}

func (s *rbacManagementService) GetPermissionsByResource(ctx context.Context, schemaName string, resourceID uuid.UUID) ([]tenant.Permission, error) {
	return s.permissionService.GetPermissionsByResource(ctx, schemaName, resourceID)
}

func (s *rbacManagementService) CountPermissions(ctx context.Context, schemaName string) (int64, error) {
	tableName := tenant.Permission{}.TableName(schemaName)
	countQuery := dbModels.QueryParams{
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}
	countData, err := s.repo.TableService.GetTableData(ctx, tableName, countQuery)
	if err != nil {
		return 0, app_errors.DatabaseError
	}

	if len(countData) == 0 {
		return 0, nil
	}

	if total, ok := countData[0]["total"].(int64); ok {
		return total, nil
	}

	return 0, nil
}

// ==================== Role-Permission Mapping ====================

func (s *rbacManagementService) AssignPermissionToRole(ctx context.Context, schemaName string, req dto.RolePermissionDTO) (tenant.RolePermission, error) {
	return s.rolePermissionService.AssignPermissionToRole(ctx, schemaName, req)
}

func (s *rbacManagementService) RemovePermissionFromRole(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) error {
	return s.rolePermissionService.RemovePermissionFromRole(ctx, schemaName, roleID, permissionID)
}

func (s *rbacManagementService) GetRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) ([]tenant.RolePermission, error) {
	return s.rolePermissionService.GetRolePermissions(ctx, schemaName, roleID)
}

func (s *rbacManagementService) GetPermissionsByRole(ctx context.Context, schemaName string, roleID uuid.UUID) ([]dto.PermissionWithDetails, error) {
	return s.rolePermissionService.GetPermissionsByRole(ctx, schemaName, roleID)
}

func (s *rbacManagementService) GetRolesByPermission(ctx context.Context, schemaName string, permissionID uuid.UUID) ([]tenant.AccessRole, error) {
	return s.rolePermissionService.GetRolesByPermission(ctx, schemaName, permissionID)
}

func (s *rbacManagementService) CheckRoleHasPermission(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) (bool, error) {
	return s.rolePermissionService.CheckRoleHasPermission(ctx, schemaName, roleID, permissionID)
}

func (s *rbacManagementService) BulkAssignPermissionsToRole(ctx context.Context, schemaName string, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	for _, permID := range permissionIDs {
		_, err := s.rolePermissionService.AssignPermissionToRole(ctx, schemaName, dto.RolePermissionDTO{
			ID:           uuid.New(),
			RoleID:       roleID,
			PermissionID: permID,
		})
		if err != nil {
			return fmt.Errorf("failed to assign permission %s: %w", permID, err)
		}
	}
	return nil
}

func (s *rbacManagementService) CountRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) (int64, error) {
	tableName := tenant.RolePermission{}.TableName(schemaName)
	countQuery := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "role_id",
				Operator: "eq",
				Value:    roleID.String(),
			},
		},
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}
	countData, err := s.repo.TableService.GetTableData(ctx, tableName, countQuery)
	if err != nil {
		return 0, app_errors.DatabaseError
	}

	if len(countData) == 0 {
		return 0, nil
	}

	if total, ok := countData[0]["total"].(int64); ok {
		return total, nil
	}

	return 0, nil
}

// ==================== User Access Management (Tenant-Scoped) ====================

func (s *rbacManagementService) AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
	if s.accessMemberService == nil {
		return nil, app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.AssignRoleToUser(ctx, schemaName, req)
}

func (s *rbacManagementService) RemoveRoleFromUser(ctx context.Context, schemaName string, userID, scopeID string, scopeType string) error {
	if s.accessMemberService == nil {
		return app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.RemoveRoleFromUser(ctx, schemaName, userID, scopeID, scopeType)
}

func (s *rbacManagementService) RemoveAccessMemberByID(ctx context.Context, schemaName string, memberID string) error {
	if s.accessMemberService == nil {
		return app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.RemoveAccessMemberByID(ctx, schemaName, memberID)
}

func (s *rbacManagementService) UpdateRoleForUser(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, newRoleID string) error {
	if s.accessMemberService == nil {
		return app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.UpdateRoleForUser(ctx, schemaName, userID, scopeType, scopeID, newRoleID)
}

func (s *rbacManagementService) GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
	if s.accessMemberService == nil {
		return nil, app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.GetUserAccessMembers(ctx, schemaName, userID)
}

func (s *rbacManagementService) GetUserAccessByScope(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	if s.accessMemberService == nil {
		return nil, app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.GetUserAccessByScope(ctx, schemaName, userID, scopeType, scopeID)
}

func (s *rbacManagementService) GetScopeMembers(ctx context.Context, schemaName string, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	if s.accessMemberService == nil {
		return nil, app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.GetScopeMembers(ctx, schemaName, scopeType, scopeID)
}

// GetAllUserAccessInWorkspace retrieves ALL access records for a user in a specific workspace
// This includes both base-level records (where workspace_id = workspaceID) and
// workspace-level records (where scope_type = workspace AND scope_id = workspaceID)
func (s *rbacManagementService) GetAllUserAccessInWorkspace(ctx context.Context, schemaName string, userID, workspaceID string) ([]dto.AccessMemberDTO, error) {
	if s.accessMemberService == nil {
		return nil, app_errors.ErrServiceNotInitialized
	}

	// Get all records using the internal method from access member service
	allMembers, err := s.GetUserAccessMembers(ctx, schemaName, userID)
	if err != nil {
		return nil, err
	}

	// Filter to only records in this workspace
	var workspaceRecords []dto.AccessMemberDTO
	for _, member := range allMembers {
		// Include base-level records where workspace_id matches
		if member.WorkspaceID != nil && *member.WorkspaceID == workspaceID {
			workspaceRecords = append(workspaceRecords, member)
		}
		// Include workspace-level records where scope_id matches
		if member.ScopeType == constant.ScopeLevels.Workspace && member.ScopeID != nil && *member.ScopeID == workspaceID {
			workspaceRecords = append(workspaceRecords, member)
		}
	}

	return workspaceRecords, nil
}

// ==================== User Permission Checks ====================

func (s *rbacManagementService) GetUserPermissions(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.PermissionWithDetails, error) {
	if s.accessMemberService == nil {
		return nil, app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.GetUserPermissions(ctx, schemaName, userID, scopeType, scopeID)
}

func (s *rbacManagementService) CheckUserPermission(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, resourceCode, actionCode string) (bool, error) {
	if s.accessMemberService == nil {
		return false, app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.CheckUserPermission(ctx, schemaName, userID, scopeType, scopeID, resourceCode, actionCode)
}

func (s *rbacManagementService) GetUserHighestRole(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) (*dto.AccessRoleDTO, error) {
	if s.accessMemberService == nil {
		return nil, app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.GetUserHighestRole(ctx, schemaName, userID, scopeType, scopeID)
}

// ==================== Bulk Operations ====================

func (s *rbacManagementService) BulkAssignRoleToUsers(ctx context.Context, schemaName string, req dto.BulkAssignRoleRequest) error {
	if s.accessMemberService == nil {
		return app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.BulkAssignRoleToUsers(ctx, schemaName, req)
}

func (s *rbacManagementService) BulkRemoveRoleFromUsers(ctx context.Context, schemaName string, userIDs []string, scopeType string, scopeID *string, roleID string) error {
	if s.accessMemberService == nil {
		return app_errors.ErrServiceNotInitialized
	}
	return s.accessMemberService.BulkRemoveRoleFromUsers(ctx, schemaName, userIDs, scopeType, scopeID, roleID)
}

// ==================== RBAC Analytics and Reporting ====================

func (s *rbacManagementService) GetRBACAnalytics(ctx context.Context, schemaName string) (dto.RBACAnalytics, error) {
	systemStatus, err := s.GetRBACSystemStatus(ctx, schemaName)
	if err != nil {
		return dto.RBACAnalytics{}, err
	}

	// Get role distribution
	roles, _, err := s.ListRoles(ctx, schemaName, 100, 0)
	if err != nil {
		return dto.RBACAnalytics{}, err
	}

	roleDistribution := make(map[string]int64)
	topRoles := []dto.RoleUsageStats{}
	for _, role := range roles {
		stats, _ := s.GetRoleUsageStats(ctx, schemaName, role.ID)
		roleDistribution[role.Name] = stats.UserCount
		topRoles = append(topRoles, stats)
	}

	// Get resource distribution
	resources, _, err := s.ListResources(ctx, schemaName, 100, 0)
	if err != nil {
		return dto.RBACAnalytics{}, err
	}

	resourceDistribution := make(map[string]int64)
	topResources := []dto.ResourceUsageStats{}
	for _, resource := range resources {
		permissions, _ := s.GetPermissionsByResource(ctx, schemaName, resource.ID)
		count := int64(len(permissions))
		resourceDistribution[resource.Code] = count
		topResources = append(topResources, dto.ResourceUsageStats{
			ResourceID:      resource.ID,
			ResourceCode:    resource.Code,
			PermissionCount: count,
		})
	}

	// Get orphaned permissions
	orphaned, _ := s.GetOrphanedPermissions(ctx, schemaName)
	unusedPermissions := []uuid.UUID{}
	for _, perm := range orphaned {
		unusedPermissions = append(unusedPermissions, perm.ID)
	}

	return dto.RBACAnalytics{
		SystemStatus:         systemStatus,
		RoleDistribution:     roleDistribution,
		ResourceDistribution: resourceDistribution,
		TopRoles:             topRoles,
		TopResources:         topResources,
		UnusedPermissions:    unusedPermissions,
	}, nil
}

func (s *rbacManagementService) GetRoleUsageStats(ctx context.Context, schemaName string, roleID uuid.UUID) (dto.RoleUsageStats, error) {
	role, err := s.GetRoleByID(ctx, schemaName, roleID)
	if err != nil {
		return dto.RoleUsageStats{}, err
	}

	permCount, _ := s.CountRolePermissions(ctx, schemaName, roleID)

	return dto.RoleUsageStats{
		RoleID:          role.ID,
		RoleName:        role.Name,
		ScopeLevel:      role.ScopeLevel,
		UserCount:       0, // Would need to query access_member table
		PermissionCount: permCount,
	}, nil
}

func (s *rbacManagementService) GetPermissionUsageStats(ctx context.Context, schemaName string, permissionID uuid.UUID) (dto.PermissionUsageStats, error) {
	permission, err := s.GetPermissionByID(ctx, schemaName, permissionID)
	if err != nil {
		return dto.PermissionUsageStats{}, err
	}

	roles, _ := s.GetRolesByPermission(ctx, schemaName, permissionID)
	roleCount := int64(len(roles))

	return dto.PermissionUsageStats{
		PermissionID: permission.ID,
		RoleCount:    roleCount,
		IsOrphaned:   roleCount == 0,
	}, nil
}

func (s *rbacManagementService) GetResourceAccessMatrix(ctx context.Context, schemaName string) ([]dto.ResourceAccessMatrix, error) {
	resources, _, err := s.ListResources(ctx, schemaName, 100, 0)
	if err != nil {
		return nil, err
	}

	matrix := []dto.ResourceAccessMatrix{}
	for _, resource := range resources {
		permissions, _ := s.GetPermissionsByResource(ctx, schemaName, resource.ID)

		actionMap := make(map[string]*dto.ActionAccessMap)
		for _, perm := range permissions {
			// Get roles with this permission
			roles, _ := s.GetRolesByPermission(ctx, schemaName, perm.ID)
			roleNames := []string{}
			roleIDs := []uuid.UUID{}
			for _, role := range roles {
				roleNames = append(roleNames, role.Name)
				roleIDs = append(roleIDs, role.ID)
			}

			// Add to action map (would need action details)
			actionKey := perm.ActionID.String()
			actionMap[actionKey] = &dto.ActionAccessMap{
				ActionID: perm.ActionID,
				Roles:    roleNames,
				RoleIDs:  roleIDs,
			}
		}

		actions := []dto.ActionAccessMap{}
		for _, action := range actionMap {
			actions = append(actions, *action)
		}

		matrix = append(matrix, dto.ResourceAccessMatrix{
			ResourceCode: resource.Code,
			ResourceID:   resource.ID,
			Actions:      actions,
		})
	}

	return matrix, nil
}

// ==================== Validation and Auditing ====================

func (s *rbacManagementService) ValidateRoleConfiguration(ctx context.Context, schemaName string, roleID uuid.UUID) (dto.RoleValidationResult, error) {
	role, err := s.GetRoleByID(ctx, schemaName, roleID)
	if err != nil {
		return dto.RoleValidationResult{
			RoleID:  roleID,
			IsValid: false,
			Issues:  []string{"Role not found"},
		}, err
	}

	permCount, _ := s.CountRolePermissions(ctx, schemaName, roleID)

	issues := []string{}
	warnings := []string{}

	if permCount == 0 {
		warnings = append(warnings, "Role has no permissions assigned")
	}

	isValid := len(issues) == 0

	return dto.RoleValidationResult{
		RoleID:          role.ID,
		RoleName:        role.Name,
		IsValid:         isValid,
		HasPermissions:  permCount > 0,
		PermissionCount: permCount,
		HasUsers:        false, // Would need to check access_member
		UserCount:       0,
		Issues:          issues,
		Warnings:        warnings,
	}, nil
}

func (s *rbacManagementService) AuditUserAccess(ctx context.Context, schemaName string, userID string) (dto.UserAccessAudit, error) {
	if s.accessMemberService == nil {
		return dto.UserAccessAudit{}, app_errors.ErrServiceNotInitialized
	}

	accessMembers, err := s.GetUserAccessMembers(ctx, schemaName, userID)
	if err != nil {
		return dto.UserAccessAudit{}, err
	}

	systemRoles := []dto.AccessRoleDTO{}
	workspaceRoles := []dto.WorkspaceRoleAssignment{}
	baseRoles := []dto.BaseRoleAssignment{}

	for _, member := range accessMembers {
		roleUUID, _ := uuid.Parse(member.RoleID)
		role, _ := s.GetRoleByID(ctx, schemaName, roleUUID)

		roleDTO := dto.AccessRoleDTO{
			ID:         role.ID,
			Name:       role.Name,
			ScopeLevel: role.ScopeLevel,
			Priority:   role.Priority,
		}

		switch member.ScopeType {
		case "system":
			systemRoles = append(systemRoles, roleDTO)
		case "workspace":
			workspaceRoles = append(workspaceRoles, dto.WorkspaceRoleAssignment{
				WorkspaceID: *member.ScopeID,
				Role:        roleDTO,
				AssignedAt:  member.CreatedAt.String(),
			})
		case "base":
			baseRoles = append(baseRoles, dto.BaseRoleAssignment{
				BaseID:     *member.ScopeID,
				Role:       roleDTO,
				AssignedAt: member.CreatedAt.String(),
			})
		}
	}

	return dto.UserAccessAudit{
		UserID:         userID,
		TotalRoles:     len(accessMembers),
		SystemRoles:    systemRoles,
		WorkspaceRoles: workspaceRoles,
		BaseRoles:      baseRoles,
	}, nil
}

func (s *rbacManagementService) GetOrphanedPermissions(ctx context.Context, schemaName string) ([]tenant.Permission, error) {
	permissions, _, err := s.ListPermissions(ctx, schemaName, 1000, 0)
	if err != nil {
		return nil, err
	}

	orphaned := []tenant.Permission{}
	for _, perm := range permissions {
		roles, _ := s.GetRolesByPermission(ctx, schemaName, perm.ID)
		if len(roles) == 0 {
			orphaned = append(orphaned, perm)
		}
	}

	return orphaned, nil
}

func (s *rbacManagementService) GetUnusedRoles(ctx context.Context, schemaName string) ([]tenant.AccessRole, error) {
	roles, _, err := s.ListRoles(ctx, schemaName, 1000, 0)
	if err != nil {
		return nil, err
	}

	unused := []tenant.AccessRole{}
	for _, role := range roles {
		permCount, _ := s.CountRolePermissions(ctx, schemaName, role.ID)
		if permCount == 0 {
			unused = append(unused, role)
		}
	}

	return unused, nil
}

// ==================== Membership Processing ====================

// isSameRole compares two role IDs after normalization
// Normalizes UUIDs by removing hyphens and converting to lowercase
func isSameRole(roleID1, roleID2 string) bool {
	if roleID1 == "" || roleID2 == "" {
		return false
	}
	norm1 := strings.ToLower(strings.ReplaceAll(roleID1, "-", ""))
	norm2 := strings.ToLower(strings.ReplaceAll(roleID2, "-", ""))
	return norm1 == norm2
}

// separateByScope divides access records into base and workspace scope records
func separateByScope(records []dto.AccessMemberDTO) (base, workspace []dto.AccessMemberDTO) {
	for _, record := range records {
		if record.ScopeType == "base" {
			base = append(base, record)
		} else if record.ScopeType == "workspace" {
			workspace = append(workspace, record)
		}
	}
	return
}

// ProcessUserMemberships processes user membership assignments across workspaces and bases
// Handles four scenarios:
// 1. User has base-level access, switching to workspace-level: remove all base records, add workspace
// 2. User has base-level access, changing role: update existing base record
// 3. User has workspace-level access, switching to base-level: remove workspace record, add base records
// 4. User has workspace-level access, changing role: update existing workspace record
func (s *rbacManagementService) ProcessUserMemberships(
	ctx context.Context,
	schema string,
	userID string,
	assignedBy string,
	memberships []dto.MembershipRequest,
) (interface{}, error) {

	summary := &MembershipProcessingSummary{
		UserID:           userID,
		ProcessedCount:   0,
		SkippedCount:     0,
		FailedCount:      0,
		ProcessedMembers: []ProcessedMember{},
		SkippedMembers:   []SkippedMember{},
		FailedMembers:    []FailedMember{},
	}

	// Handle empty membership
	if len(memberships) == 0 {
		fmt.Println("No memberships to process")
		return summary, nil
	}

	// Process each membership request
	for i, membership := range memberships {
		// Validate role is not base-level with workspace-id (common mistake)
		if (membership.Role == constant.RBACRoleNames.BaseMember || membership.Role == constant.RBACRoleNames.BaseMemberReadOnly) && membership.WorkspaceID != "" {
			summary.SkippedCount++
			summary.SkippedMembers = append(summary.SkippedMembers, SkippedMember{
				Index:  i,
				Reason: fmt.Sprintf("invalid request: role '%s' is a base-level role but WorkspaceID is provided. For base-level roles, use the 'bases' array instead", membership.Role),
				Role:   membership.Role,
			})
			continue
		}

		// Case A: Workspace-level membership (role is maintainer or workspace-read)
		if membership.Role == constant.RBACRoleNames.WorkspaceMaintainer || membership.Role == constant.RBACRoleNames.WorkspaceMaintainerRO {
			if membership.WorkspaceID == "" {
				summary.SkippedCount++
				summary.SkippedMembers = append(summary.SkippedMembers, SkippedMember{
					Index:  i,
					Reason: "workspace_id is required for workspace-level roles",
					Role:   membership.Role,
				})
				continue
			}

			// Get the workspace role
			roleData, err := s.GetRoleByName(ctx, schema, membership.Role)
			if err != nil {
				summary.FailedCount++
				summary.FailedMembers = append(summary.FailedMembers, FailedMember{
					Index:  i,
					Reason: fmt.Sprintf("failed to get role: %v", err),
					Role:   membership.Role,
					Error:  err,
				})
				continue
			}

			// Get ALL existing access for user in this workspace (both base and workspace level)
			allExistingAccess, err := s.GetAllUserAccessInWorkspace(ctx, schema, userID, membership.WorkspaceID)
			if err == nil && len(allExistingAccess) > 0 {
				// Separate records by scope type
				baseRecords, workspaceRecords := separateByScope(allExistingAccess)

				fmt.Printf("DEBUG: SCOPE-UPDATE - Found %d base records and %d workspace records for user %s in workspace %s\n",
					len(baseRecords), len(workspaceRecords), userID, membership.WorkspaceID)

				// SCENARIO 1: User has base-level records (need to convert to workspace-level)
				if len(baseRecords) > 0 {
					fmt.Printf("DEBUG: SCOPE-UPDATE - Deleting %d base-level record(s) before converting to workspace-level\n", len(baseRecords))

					// Delete ALL base records from this workspace by ID (more reliable than composite key)
					removalErrors := []error{}
					for idx, baseRecord := range baseRecords {
						fmt.Printf("DEBUG: SCOPE-UPDATE - Base Record %d Details:\n", idx)
						fmt.Printf("  - ID: %v\n", baseRecord.ID)
						fmt.Printf("  - ScopeID: %v\n", baseRecord.ScopeID)
						fmt.Printf("  - WorkspaceID: %v\n", baseRecord.WorkspaceID)
						fmt.Printf("  - RoleID: %s\n", baseRecord.RoleID)
						fmt.Printf("  - ScopeType: %s\n", baseRecord.ScopeType)

						// Delete by ID instead of by composite key (user_id, scope_id, scope_type)
						// This is more reliable and avoids mismatch issues
						fmt.Printf("DEBUG: SCOPE-UPDATE - Attempting to remove base record by ID: %s\n", baseRecord.ID.String())
						errRemove := s.RemoveAccessMemberByID(ctx, schema, baseRecord.ID.String())
						if errRemove != nil {
							fmt.Printf("DEBUG: SCOPE-UPDATE - Failed to remove base record %s: %v (error type: %T)\n", baseRecord.ID.String(), errRemove, errRemove)
							removalErrors = append(removalErrors, errRemove)
						} else {
							fmt.Printf("DEBUG: SCOPE-UPDATE - Successfully removed base record %s\n", baseRecord.ID.String())
						}
					}

					// If deletions failed, log warning but continue
					// Don't abort the operation - attempting to remove may have partially succeeded
					if len(removalErrors) > 0 {
						fmt.Printf("DEBUG: SCOPE-UPDATE - WARNING: Failed to remove %d base-level record(s), but continuing with workspace assignment\n", len(removalErrors))
						for idx, err := range removalErrors {
							fmt.Printf("  - Error %d: %v\n", idx, err)
						}
					}

					fmt.Printf("DEBUG: SCOPE-UPDATE - Proceeding with workspace assignment.\n")
				}

				// SCENARIO 2: User already has workspace-level record (check if role matches)
				if len(workspaceRecords) > 0 {
					existingRecord := workspaceRecords[0]
					newRoleID := roleData.ID.String()

					if isSameRole(existingRecord.RoleID, newRoleID) {
						// Role is already correct - skip
						fmt.Printf("DEBUG: SCOPE-UPDATE - User already has role '%s' in workspace %s. Skipping.\n", membership.Role, membership.WorkspaceID)

						summary.SkippedCount++
						summary.SkippedMembers = append(summary.SkippedMembers, SkippedMember{
							Index:  i,
							Reason: fmt.Sprintf("user already has role '%s' in this workspace", membership.Role),
							Role:   membership.Role,
						})
						continue
					} else {
						// Role is different - update it
						fmt.Printf("DEBUG: SCOPE-UPDATE - Role changed from %s to %s. Updating.\n", existingRecord.RoleID, newRoleID)

						errUpdate := s.UpdateRoleForUser(ctx, schema, userID, constant.ScopeLevels.Workspace, &membership.WorkspaceID, newRoleID)
						if errUpdate != nil {
							summary.FailedCount++
							summary.FailedMembers = append(summary.FailedMembers, FailedMember{
								Index:  i,
								Reason: fmt.Sprintf("failed to update workspace role: %v", errUpdate),
								Role:   membership.Role,
								Error:  errUpdate,
							})
							continue
						}

						fmt.Printf("DEBUG: SCOPE-UPDATE - Role updated successfully\n")

						summary.ProcessedCount++
						summary.ProcessedMembers = append(summary.ProcessedMembers, ProcessedMember{
							Index:     i,
							ScopeType: constant.ScopeLevels.Workspace,
							ScopeID:   membership.WorkspaceID,
							Role:      membership.Role,
							Type:      "workspace-level-updated",
						})
						continue
					}
				}

				// If we deleted base records but no workspace record exists, create one
				if len(baseRecords) > 0 && len(workspaceRecords) == 0 {
					fmt.Printf("DEBUG: SCOPE-UPDATE - Creating new workspace-level record after deleting base records\n")

					accessMemberReq := dto.AccessMemberDTO{
						UserID:     userID,
						ScopeType:  constant.ScopeLevels.Workspace,
						ScopeID:    &membership.WorkspaceID,
						RoleID:     roleData.ID.String(),
						AssignedBy: &assignedBy,
					}

					_, err := s.AssignRoleToUser(ctx, schema, accessMemberReq)
					if err != nil {
						summary.FailedCount++
						summary.FailedMembers = append(summary.FailedMembers, FailedMember{
							Index:  i,
							Reason: fmt.Sprintf("failed to assign workspace role: %v", err),
							Role:   membership.Role,
							Error:  err,
						})
						continue
					}

					fmt.Printf("DEBUG: SCOPE-UPDATE - Workspace-level record created successfully\n")

					summary.ProcessedCount++
					summary.ProcessedMembers = append(summary.ProcessedMembers, ProcessedMember{
						Index:     i,
						ScopeType: constant.ScopeLevels.Workspace,
						ScopeID:   membership.WorkspaceID,
						Role:      membership.Role,
						Type:      "base-to-workspace-conversion",
					})
					continue
				}
			}

			// SCENARIO 3: No existing records in this workspace - create new workspace-level record
			fmt.Printf("DEBUG: SCOPE-UPDATE - No existing records. Creating new workspace-level access for user %s\n", userID)

			accessMemberReq := dto.AccessMemberDTO{
				UserID:     userID,
				ScopeType:  constant.ScopeLevels.Workspace,
				ScopeID:    &membership.WorkspaceID,
				RoleID:     roleData.ID.String(),
				AssignedBy: &assignedBy,
			}

			_, err = s.AssignRoleToUser(ctx, schema, accessMemberReq)
			if err != nil {
				summary.FailedCount++
				summary.FailedMembers = append(summary.FailedMembers, FailedMember{
					Index:  i,
					Reason: fmt.Sprintf("failed to assign role: %v", err),
					Role:   membership.Role,
					Error:  err,
				})
				continue
			}

			fmt.Printf("DEBUG: SCOPE-UPDATE - Workspace-level record created successfully\n")

			summary.ProcessedCount++
			summary.ProcessedMembers = append(summary.ProcessedMembers, ProcessedMember{
				Index:     i,
				ScopeType: constant.ScopeLevels.Workspace,
				ScopeID:   membership.WorkspaceID,
				Role:      membership.Role,
				Type:      "workspace-level",
			})

			continue
		}

		// Case B: Base-level membership (role is empty, check bases array)
		if membership.Role == "" {
			if len(membership.Bases) == 0 {
				summary.SkippedCount++
				summary.SkippedMembers = append(summary.SkippedMembers, SkippedMember{
					Index:  i,
					Reason: "when role is empty, bases array must not be empty",
					Role:   "",
				})
				continue
			}

			// Check if user has workspace-level access - CASE 3: Convert from workspace to base
			workspaceMembers, err := s.GetUserAccessByScope(ctx, schema, userID, constant.ScopeLevels.Workspace, nil)
			if err == nil && len(workspaceMembers) > 0 {
				fmt.Printf("DEBUG: CASE 3 - Found workspace-level access for user %s. Will convert to base-level.\n", userID)

				// For Case 3, we need to determine which workspace we're converting from
				// User is requesting to add bases, so first workspace they mention will tell us which workspace to remove
				if len(membership.Bases) > 0 {
					firstBase := membership.Bases[0]
					if firstBase.BaseID != "" {
						// Get the workspace that owns the first base to determine which workspace to remove from
						baseData, errGetBase := s.baseService.GetBaseByID(ctx, schema, firstBase.BaseID)
						if errGetBase == nil {
							targetWorkspaceID := baseData.WorkspaceID

							// Filter workspace members to find the one we're converting from
							for _, wsMember := range workspaceMembers {
								if wsMember.ScopeID != nil && *wsMember.ScopeID == targetWorkspaceID {
									// Remove workspace-level access for this specific workspace
									fmt.Printf("DEBUG: CASE 3 - Removing workspace-level access from workspace %s (Record ID: %s)\n", targetWorkspaceID, wsMember.ID.String())
									// Use ID-based deletion instead of composite key
									errRemove := s.RemoveAccessMemberByID(ctx, schema, wsMember.ID.String())
									if errRemove != nil {
										fmt.Printf("DEBUG: CASE 3 - Failed to remove workspace-level access: %v\n", errRemove)
										summary.FailedCount++
										summary.FailedMembers = append(summary.FailedMembers, FailedMember{
											Index:  i,
											Reason: fmt.Sprintf("failed to remove workspace-level access: %v", errRemove),
											Role:   "",
											Error:  errRemove,
										})
										continue
									}
									fmt.Printf("DEBUG: CASE 3 - Successfully removed workspace-level access\n")
									break
								}
							}
						}
					}
				}
			}

			// Process each base
			for j, baseMembership := range membership.Bases {
				if baseMembership.BaseID == "" {
					summary.SkippedCount++
					summary.SkippedMembers = append(summary.SkippedMembers, SkippedMember{
						Index:  i,
						Reason: fmt.Sprintf("base_id is required (bases[%d])", j),
						Role:   "",
					})
					continue
				}

				if baseMembership.Role != "base-member" && baseMembership.Role != "base-read" {
					summary.SkippedCount++
					summary.SkippedMembers = append(summary.SkippedMembers, SkippedMember{
						Index:  i,
						Reason: fmt.Sprintf("invalid base role: %s (bases[%d]), must be 'base-member' or 'base-read'", baseMembership.Role, j),
						Role:   "",
					})
					continue
				}

				// Get the base to find its workspace
				baseData, errGetBase := s.baseService.GetBaseByID(ctx, schema, baseMembership.BaseID)
				if errGetBase != nil {
					summary.FailedCount++
					summary.FailedMembers = append(summary.FailedMembers, FailedMember{
						Index:  i,
						Reason: fmt.Sprintf("failed to get base details: %v (bases[%d])", errGetBase, j),
						Role:   baseMembership.Role,
						Error:  errGetBase,
					})
					continue
				}

				// Get the base role
				roleData, err := s.GetRoleByName(ctx, schema, baseMembership.Role)
				if err != nil {
					summary.FailedCount++
					summary.FailedMembers = append(summary.FailedMembers, FailedMember{
						Index:  i,
						Reason: fmt.Sprintf("failed to get base role: %v (bases[%d])", err, j),
						Role:   baseMembership.Role,
						Error:  err,
					})
					continue
				}

				// Check if user has existing access in this base
				existingBaseMembers, err := s.GetUserAccessByScope(ctx, schema, userID, constant.ScopeLevels.Base, &baseMembership.BaseID)
				if err == nil && len(existingBaseMembers) > 0 {
					// CASE 2: User has base-level access and changing only the role
					existingRoleID := existingBaseMembers[0].RoleID
					newRoleID := roleData.ID.String()

					fmt.Printf("DEBUG: Comparing roles - Existing: %s, New: %s\n", existingRoleID, newRoleID)

					// Normalize UUIDs for comparison (remove hyphens and convert to lowercase)
					existingRoleIDNorm := strings.ToLower(strings.ReplaceAll(existingRoleID, "-", ""))
					newRoleIDNorm := strings.ToLower(strings.ReplaceAll(newRoleID, "-", ""))

					if existingRoleIDNorm != newRoleIDNorm {
						// Role changed - update the role directly in the database
						fmt.Printf("DEBUG: Role has changed for user %s in base %s. Updating from %s to %s\n", userID, baseMembership.BaseID, existingRoleID, newRoleID)

						errUpdate := s.UpdateRoleForUser(ctx, schema, userID, constant.ScopeLevels.Base, &baseMembership.BaseID, newRoleID)
						if errUpdate != nil {
							fmt.Printf("DEBUG: Update failed with error: %v\n", errUpdate)
							summary.FailedCount++
							summary.FailedMembers = append(summary.FailedMembers, FailedMember{
								Index:  i,
								Reason: fmt.Sprintf("failed to update base role (bases[%d]): %v", j, errUpdate),
								Role:   baseMembership.Role,
								Error:  errUpdate,
							})
							continue
						}

						fmt.Printf("DEBUG: Role updated successfully for user %s in base %s\n", userID, baseMembership.BaseID)

						summary.ProcessedCount++
						summary.ProcessedMembers = append(summary.ProcessedMembers, ProcessedMember{
							Index:     i,
							ScopeType: constant.ScopeLevels.Base,
							ScopeID:   baseMembership.BaseID,
							Role:      baseMembership.Role,
							Type:      "base-level-updated",
						})
					} else {
						// Same role - skip
						fmt.Printf("DEBUG: Same role '%s' already exists for user %s in base %s. Skipping.\n", baseMembership.Role, userID, baseMembership.BaseID)

						summary.SkippedCount++
						summary.SkippedMembers = append(summary.SkippedMembers, SkippedMember{
							Index:  i,
							Reason: fmt.Sprintf("user already has same role '%s' in this base (bases[%d])", baseMembership.Role, j),
							Role:   baseMembership.Role,
						})
					}
					continue
				}

				// No existing access - assign new base-level role
				accessMemberReq := dto.AccessMemberDTO{
					UserID:      userID,
					ScopeType:   constant.ScopeLevels.Base,
					ScopeID:     &baseMembership.BaseID,
					RoleID:      roleData.ID.String(),
					WorkspaceID: &baseData.WorkspaceID,
					AssignedBy:  &assignedBy,
				}

				_, err = s.AssignRoleToUser(ctx, schema, accessMemberReq)
				if err != nil {
					summary.FailedCount++
					summary.FailedMembers = append(summary.FailedMembers, FailedMember{
						Index:  i,
						Reason: fmt.Sprintf("failed to assign base role: %v (bases[%d])", err, j),
						Role:   baseMembership.Role,
						Error:  err,
					})
					continue
				}

				summary.ProcessedCount++
				summary.ProcessedMembers = append(summary.ProcessedMembers, ProcessedMember{
					Index:     i,
					ScopeType: constant.ScopeLevels.Base,
					ScopeID:   baseMembership.BaseID,
					Role:      baseMembership.Role,
					Type:      "base-level",
				})
			}

			continue
		}

		// Invalid format - role is neither workspace-level nor base-level
		summary.SkippedCount++
		summary.SkippedMembers = append(summary.SkippedMembers, SkippedMember{
			Index:  i,
			Reason: fmt.Sprintf("invalid role format: role is '%s' (not 'maintainer', 'workspace-read', or empty)", membership.Role),
			Role:   membership.Role,
		})
	}

	return summary, nil
}

// MembershipProcessingSummary contains the summary of membership processing
type MembershipProcessingSummary struct {
	UserID           string
	ProcessedCount   int
	SkippedCount     int
	FailedCount      int
	ProcessedMembers []ProcessedMember
	SkippedMembers   []SkippedMember
	FailedMembers    []FailedMember
}

// ProcessedMember represents a successfully processed membership
type ProcessedMember struct {
	Index     int    `json:"index"`
	Type      string `json:"type"`       // "workspace-level" or "base-level"
	ScopeType string `json:"scope_type"` // "workspace" or "base"
	ScopeID   string `json:"scope_id"`
	Role      string `json:"role"`
}

// SkippedMember represents a membership that was skipped
type SkippedMember struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
	Role   string `json:"role"`
}

// FailedMember represents a membership that failed to process
type FailedMember struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
	Role   string `json:"role"`
	Error  error  `json:"error,omitempty"`
}
