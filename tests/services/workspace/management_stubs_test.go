package workspace_test

import (
	"context"
	"mime/multipart"

	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
)

// StubWorkspaceService provides optional function overrides for WorkspaceService.
type StubWorkspaceService struct {
	CreateWorkspaceFn    func(ctx context.Context, schemaName string) (tenant.Workspace, error)
	WorkspaceInsertionFn func(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string) (tenant.Workspace, error)
	GetWorkspaceByIDFn   func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error)
	GetAllWorkspacesFn   func(ctx context.Context, schemaName string) ([]tenant.Workspace, error)
	UpdateWorkspaceFn    func(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate) (tenant.Workspace, error)
	DeleteWorkspaceFn    func(ctx context.Context, schemaName string, id string) error
	GetBulkWorkspacesFn  func(ctx context.Context, schemaName string, ids []string) ([]tenant.Workspace, error)
}

func (s *StubWorkspaceService) CreateWorkspace(ctx context.Context, schemaName string) (tenant.Workspace, error) {
	if s.CreateWorkspaceFn != nil {
		return s.CreateWorkspaceFn(ctx, schemaName)
	}
	return tenant.Workspace{}, nil
}
func (s *StubWorkspaceService) WorkspaceInsertion(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string) (tenant.Workspace, error) {
	if s.WorkspaceInsertionFn != nil {
		return s.WorkspaceInsertionFn(ctx, req, schemaName)
	}
	return tenant.Workspace{}, nil
}
func (s *StubWorkspaceService) GetWorkspaceByID(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
	if s.GetWorkspaceByIDFn != nil {
		return s.GetWorkspaceByIDFn(ctx, schemaName, id)
	}
	return tenant.Workspace{}, nil
}
func (s *StubWorkspaceService) GetAllWorkspaces(ctx context.Context, schemaName string) ([]tenant.Workspace, error) {
	if s.GetAllWorkspacesFn != nil {
		return s.GetAllWorkspacesFn(ctx, schemaName)
	}
	return []tenant.Workspace{}, nil
}
func (s *StubWorkspaceService) UpdateWorkspace(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate) (tenant.Workspace, error) {
	if s.UpdateWorkspaceFn != nil {
		return s.UpdateWorkspaceFn(ctx, schemaName, id, req)
	}
	return tenant.Workspace{}, nil
}
func (s *StubWorkspaceService) DeleteWorkspace(ctx context.Context, schemaName string, id string) error {
	if s.DeleteWorkspaceFn != nil {
		return s.DeleteWorkspaceFn(ctx, schemaName, id)
	}
	return nil
}
func (s *StubWorkspaceService) GetBulkWorkspaces(ctx context.Context, schemaName string, ids []string) ([]tenant.Workspace, error) {
	if s.GetBulkWorkspacesFn != nil {
		return s.GetBulkWorkspacesFn(ctx, schemaName, ids)
	}
	return []tenant.Workspace{}, nil
}

// StubWorkspaceMemberService provides optional function overrides for WorkspaceMemberService.
type StubWorkspaceMemberService struct {
	GetAllWorkspaceMembersByUserFn         func(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error)
	DeleteWorkspaceMemberFn                func(ctx context.Context, schemaName string, id string) error
	GetWorkspaceMemberByUserAndWorkspaceFn func(ctx context.Context, schemaName string, userId string, workspaceId string) (*tenant.WorkspaceMember, error)
	GetWorkspaceMemberByUserFn             func(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error)
	GetWorkspaceMembersByWorkspaceFn       func(ctx context.Context, schemaName string, workspaceId string) ([]tenant.WorkspaceMember, error)
	DeleteUserMappingsFn                   func(ctx context.Context, schemaName string, userId string) error
	UpdateWorkspaceMemberBasesFn           func(ctx context.Context, schemaName string, workspaceId string, userId string, accessLevel string, basesIds string) error
}

func (s *StubWorkspaceMemberService) GetAllWorkspaceMembersByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error) {
	if s.GetAllWorkspaceMembersByUserFn != nil {
		return s.GetAllWorkspaceMembersByUserFn(ctx, schemaName, userId)
	}
	return []tenant.WorkspaceMember{}, nil
}
func (s *StubWorkspaceMemberService) DeleteWorkspaceMember(ctx context.Context, schemaName string, id string) error {
	if s.DeleteWorkspaceMemberFn != nil {
		return s.DeleteWorkspaceMemberFn(ctx, schemaName, id)
	}
	return nil
}
func (s *StubWorkspaceMemberService) GetWorkspaceMemberByUserAndWorkspace(ctx context.Context, schemaName string, userId string, workspaceId string) (*tenant.WorkspaceMember, error) {
	if s.GetWorkspaceMemberByUserAndWorkspaceFn != nil {
		return s.GetWorkspaceMemberByUserAndWorkspaceFn(ctx, schemaName, userId, workspaceId)
	}
	return nil, nil
}
func (s *StubWorkspaceMemberService) GetWorkspaceMemberByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error) {
	if s.GetWorkspaceMemberByUserFn != nil {
		return s.GetWorkspaceMemberByUserFn(ctx, schemaName, userId)
	}
	return []tenant.WorkspaceMember{}, nil
}
func (s *StubWorkspaceMemberService) GetWorkspaceMembersByWorkspace(ctx context.Context, schemaName string, workspaceId string) ([]tenant.WorkspaceMember, error) {
	if s.GetWorkspaceMembersByWorkspaceFn != nil {
		return s.GetWorkspaceMembersByWorkspaceFn(ctx, schemaName, workspaceId)
	}
	return []tenant.WorkspaceMember{}, nil
}
func (s *StubWorkspaceMemberService) DeleteUserMappings(ctx context.Context, schemaName string, userId string) error {
	if s.DeleteUserMappingsFn != nil {
		return s.DeleteUserMappingsFn(ctx, schemaName, userId)
	}
	return nil
}
func (s *StubWorkspaceMemberService) UpdateWorkspaceMemberBases(ctx context.Context, schemaName string, workspaceId string, userId string, accessLevel string, basesIds string) error {
	if s.UpdateWorkspaceMemberBasesFn != nil {
		return s.UpdateWorkspaceMemberBasesFn(ctx, schemaName, workspaceId, userId, accessLevel, basesIds)
	}
	return nil
}

// StubBaseManagementService provides optional function overrides for BaseManagementService.
type StubBaseManagementService struct {
	CreateBaseFn            func(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error)
	GetAllBasesWithAccessFn func(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error)
	GetBasesByWorkspaceFn   func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error)
	GetBaseByIDFn           func(ctx context.Context, schemaName string, id string) (tenant.Base, error)
}

func (s *StubBaseManagementService) CreateBase(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
	if s.CreateBaseFn != nil {
		return s.CreateBaseFn(ctx, req, schemaName, userId)
	}
	return tenant.Base{}, nil
}
func (s *StubBaseManagementService) CreateBaseWithoutTable(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s *StubBaseManagementService) CreateBaseWithImage(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string, fileHeader *multipart.FileHeader) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s *StubBaseManagementService) GetBaseByID(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
	if s.GetBaseByIDFn != nil {
		return s.GetBaseByIDFn(ctx, schemaName, id)
	}
	return tenant.Base{}, nil
}
func (s *StubBaseManagementService) GetAllBasesWithAccess(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error) {
	if s.GetAllBasesWithAccessFn != nil {
		return s.GetAllBasesWithAccessFn(ctx, schemaName, workspaceMemberData)
	}
	return []tenant.Base{}, nil
}
func (s *StubBaseManagementService) UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate, userId string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s *StubBaseManagementService) DeleteBase(ctx context.Context, schemaName string, id string) error {
	return nil
}
func (s *StubBaseManagementService) GetTablesByBaseId(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
	return []dto.TableResponse{}, nil
}
func (s *StubBaseManagementService) GetBasesByWorkspace(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
	if s.GetBasesByWorkspaceFn != nil {
		return s.GetBasesByWorkspaceFn(ctx, schemaName, workspaceID)
	}
	return []tenant.Base{}, nil
}
func (s *StubBaseManagementService) AddBaseImage(ctx context.Context, schema string, baseID string, fileHeader *multipart.FileHeader, userId string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s *StubBaseManagementService) RemoveBaseImage(ctx context.Context, schema string, baseID string, userId string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s *StubBaseManagementService) RemoveUserFromBase(ctx context.Context, schemaName string, baseID string, userID string) error {
	return nil
}

// StubTableManagementService provides optional function overrides for TableManagementService.
type StubTableManagementService struct {
	GetModelByWorkspaceIDFn func(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error)
}

func (s *StubTableManagementService) CreateTableWithDefaults(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
	return dto.TableResponse{}, nil
}
func (s *StubTableManagementService) UpdateTable(ctx context.Context, id string, tableData dto.UpdateTableRequest, schemaName string) (dto.TableResponse, error) {
	return dto.TableResponse{}, nil
}
func (s *StubTableManagementService) GetTableByID(ctx context.Context, id string, schemaName string) (dto.TableResponse, error) {
	return dto.TableResponse{}, nil
}
func (s *StubTableManagementService) GetAllTables(ctx context.Context, schemaName string) ([]dto.TableResponse, error) {
	return []dto.TableResponse{}, nil
}
func (s *StubTableManagementService) GetModelByBaseID(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
	return []dto.TableResponse{}, nil
}
func (s *StubTableManagementService) GetModelByWorkspaceID(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
	if s.GetModelByWorkspaceIDFn != nil {
		return s.GetModelByWorkspaceIDFn(ctx, schemaName, workspaceID)
	}
	return []dto.TableResponse{}, nil
}
func (s *StubTableManagementService) DeleteTable(ctx context.Context, schemaName string, modelID string) error {
	return nil
}

func (s *StubTableManagementService) AddColumn(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
	return dto.ColumnResponse{}, nil
}
func (s *StubTableManagementService) GetColumnById(ctx context.Context, schemaName string, id string) (dto.ColumnResponse, error) {
	return dto.ColumnResponse{}, nil
}
func (s *StubTableManagementService) GetAllColumns(ctx context.Context, schemaName string) ([]dto.ColumnResponse, error) {
	return []dto.ColumnResponse{}, nil
}
func (s *StubTableManagementService) GetColumnsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ColumnResponse, error) {
	return []dto.ColumnResponse{}, nil
}
func (s *StubTableManagementService) UpdateColumn(ctx context.Context, schemaName string, id string, req dto.ColumnUpdate) (dto.ColumnResponse, error) {
	return dto.ColumnResponse{}, nil
}
func (s *StubTableManagementService) DeleteColumn(ctx context.Context, schemaName string, id string) error {
	return nil
}
func (s *StubTableManagementService) ReorderColumn(ctx context.Context, schemaName string, req dto.ReorderColumnRequest) ([]dto.ColumnResponse, error) {
	return []dto.ColumnResponse{}, nil
}

func (s *StubTableManagementService) CreateView(ctx context.Context, schemaName string, viewData dto.CreateViewRequest) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (s *StubTableManagementService) GetViewByID(ctx context.Context, schemaName string, id string) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (s *StubTableManagementService) GetAllViews(ctx context.Context, schemaName string) ([]dto.ViewResponse, error) {
	return []dto.ViewResponse{}, nil
}
func (s *StubTableManagementService) GetViewsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ViewResponse, error) {
	return []dto.ViewResponse{}, nil
}
func (s *StubTableManagementService) UpdateView(ctx context.Context, schemaName string, id string, req dto.ViewUpdate) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (s *StubTableManagementService) DeleteView(ctx context.Context, schemaName string, id string) error {
	return nil
}

func (s *StubTableManagementService) CreateRow(ctx context.Context, schemaName string, req dto.CreateRowRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s *StubTableManagementService) CreateRowWithRecords(ctx context.Context, schemaName string, modelAlias string, record map[string]interface{}) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s *StubTableManagementService) CreateRowsWithRecordsBulk(ctx context.Context, schemaName string, modelAlias string, records []map[string]interface{}) ([]dto.RecordResponse, error) {
	return []dto.RecordResponse{}, nil
}
func (s *StubTableManagementService) GetAllRecords(ctx context.Context, schemaName string, modelID string) (dto.RecordsResponse, error) {
	return dto.RecordsResponse{}, nil
}
func (s *StubTableManagementService) InsertRowData(ctx context.Context, schemaName string, req dto.InsertRowDataRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s *StubTableManagementService) DeleteRow(ctx context.Context, schemaName string, req dto.DeleteRowDataRequest) error {
	return nil
}
func (s *StubTableManagementService) UpdateRawDataForLinks(ctx context.Context, schemaName string, req dto.UpdateRowDataLinksRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s *StubTableManagementService) AddAttachment(ctx context.Context, schemaName string, req dto.AddAttachmentRequest, files []*multipart.FileHeader) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s *StubTableManagementService) BulkDeleteRows(ctx context.Context, schemaName string, req dto.BulkDeleteRowsRequest) (int, error) {
	return 0, nil
}
func (s *StubTableManagementService) RemoveAttachments(ctx context.Context, schemaName string, req dto.RemoveAttachmentsRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}

// StubRBACManagementService provides optional function overrides for RBACManagementService.
type StubRBACManagementService struct {
	GetUserAccessMembersFn func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error)
	GetRoleByIDFn          func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error)
}

func (s *StubRBACManagementService) InitializeRBACSystem(ctx context.Context, schema string) error {
	return nil
}
func (s *StubRBACManagementService) GetRBACSystemStatus(ctx context.Context, schemaName string) (dto.RBACSystemStatus, error) {
	return dto.RBACSystemStatus{}, nil
}
func (s *StubRBACManagementService) CreateRole(ctx context.Context, schemaName string, req dto.AccessRoleDTO) (tenant.AccessRole, error) {
	return tenant.AccessRole{}, nil
}
func (s *StubRBACManagementService) GetRoleByID(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
	if s.GetRoleByIDFn != nil {
		return s.GetRoleByIDFn(ctx, schemaName, roleID)
	}
	return tenant.AccessRole{}, nil
}
func (s *StubRBACManagementService) GetRoleByName(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
	return tenant.AccessRole{}, nil
}
func (s *StubRBACManagementService) GetRolesByScope(ctx context.Context, schemaName string, scopeLevel string) ([]tenant.AccessRole, error) {
	return []tenant.AccessRole{}, nil
}
func (s *StubRBACManagementService) ListRoles(ctx context.Context, schemaName string, limit, offset int) ([]tenant.AccessRole, int64, error) {
	return []tenant.AccessRole{}, 0, nil
}
func (s *StubRBACManagementService) UpdateRole(ctx context.Context, schemaName string, roleID uuid.UUID, req dto.AccessRoleDTO) (tenant.AccessRole, error) {
	return tenant.AccessRole{}, nil
}
func (s *StubRBACManagementService) DeleteRole(ctx context.Context, schemaName string, roleID uuid.UUID) error {
	return nil
}
func (s *StubRBACManagementService) CountRoles(ctx context.Context, schemaName string) (int64, error) {
	return 0, nil
}

func (s *StubRBACManagementService) CreateResource(ctx context.Context, schemaName string, req dto.ResourceDTO) (tenant.Resource, error) {
	return tenant.Resource{}, nil
}
func (s *StubRBACManagementService) GetResourceByID(ctx context.Context, schemaName string, resourceID uuid.UUID) (tenant.Resource, error) {
	return tenant.Resource{}, nil
}
func (s *StubRBACManagementService) GetResourceByCode(ctx context.Context, schemaName string, code string) (tenant.Resource, error) {
	return tenant.Resource{}, nil
}
func (s *StubRBACManagementService) ListResources(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Resource, int64, error) {
	return []tenant.Resource{}, 0, nil
}
func (s *StubRBACManagementService) UpdateResource(ctx context.Context, schemaName string, resourceID uuid.UUID, req dto.ResourceDTO) (tenant.Resource, error) {
	return tenant.Resource{}, nil
}
func (s *StubRBACManagementService) DeleteResource(ctx context.Context, schemaName string, resourceID uuid.UUID) error {
	return nil
}
func (s *StubRBACManagementService) GetOrCreateResource(ctx context.Context, schemaName string, code string, description *string) (tenant.Resource, error) {
	return tenant.Resource{}, nil
}
func (s *StubRBACManagementService) CountResources(ctx context.Context, schemaName string) (int64, error) {
	return 0, nil
}

func (s *StubRBACManagementService) CreateAction(ctx context.Context, schemaName string, req dto.ActionDTO) (tenant.Action, error) {
	return tenant.Action{}, nil
}
func (s *StubRBACManagementService) GetActionByID(ctx context.Context, schemaName string, actionID uuid.UUID) (tenant.Action, error) {
	return tenant.Action{}, nil
}
func (s *StubRBACManagementService) GetActionByCode(ctx context.Context, schemaName string, code string) (tenant.Action, error) {
	return tenant.Action{}, nil
}
func (s *StubRBACManagementService) ListActions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Action, int64, error) {
	return []tenant.Action{}, 0, nil
}
func (s *StubRBACManagementService) UpdateAction(ctx context.Context, schemaName string, actionID uuid.UUID, req dto.ActionDTO) (tenant.Action, error) {
	return tenant.Action{}, nil
}
func (s *StubRBACManagementService) DeleteAction(ctx context.Context, schemaName string, actionID uuid.UUID) error {
	return nil
}
func (s *StubRBACManagementService) GetOrCreateAction(ctx context.Context, schemaName string, code string, description *string) (tenant.Action, error) {
	return tenant.Action{}, nil
}
func (s *StubRBACManagementService) CountActions(ctx context.Context, schemaName string) (int64, error) {
	return 0, nil
}

func (s *StubRBACManagementService) CreatePermission(ctx context.Context, schemaName string, req dto.PermissionDTO) (tenant.Permission, error) {
	return tenant.Permission{}, nil
}
func (s *StubRBACManagementService) GetPermissionByID(ctx context.Context, schemaName string, permissionID uuid.UUID) (tenant.Permission, error) {
	return tenant.Permission{}, nil
}
func (s *StubRBACManagementService) GetPermissionByResourceAndAction(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error) {
	return tenant.Permission{}, nil
}
func (s *StubRBACManagementService) ListPermissions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Permission, int64, error) {
	return []tenant.Permission{}, 0, nil
}
func (s *StubRBACManagementService) DeletePermission(ctx context.Context, schemaName string, permissionID uuid.UUID) error {
	return nil
}
func (s *StubRBACManagementService) GetOrCreatePermission(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error) {
	return tenant.Permission{}, nil
}
func (s *StubRBACManagementService) GetPermissionsByResource(ctx context.Context, schemaName string, resourceID uuid.UUID) ([]tenant.Permission, error) {
	return []tenant.Permission{}, nil
}
func (s *StubRBACManagementService) CountPermissions(ctx context.Context, schemaName string) (int64, error) {
	return 0, nil
}

func (s *StubRBACManagementService) AssignPermissionToRole(ctx context.Context, schemaName string, req dto.RolePermissionDTO) (tenant.RolePermission, error) {
	return tenant.RolePermission{}, nil
}
func (s *StubRBACManagementService) RemovePermissionFromRole(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) error {
	return nil
}
func (s *StubRBACManagementService) GetRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) ([]tenant.RolePermission, error) {
	return []tenant.RolePermission{}, nil
}
func (s *StubRBACManagementService) GetPermissionsByRole(ctx context.Context, schemaName string, roleID uuid.UUID) ([]dto.PermissionWithDetails, error) {
	return []dto.PermissionWithDetails{}, nil
}
func (s *StubRBACManagementService) GetRolesByPermission(ctx context.Context, schemaName string, permissionID uuid.UUID) ([]tenant.AccessRole, error) {
	return []tenant.AccessRole{}, nil
}
func (s *StubRBACManagementService) CheckRoleHasPermission(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) (bool, error) {
	return false, nil
}
func (s *StubRBACManagementService) BulkAssignPermissionsToRole(ctx context.Context, schemaName string, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	return nil
}
func (s *StubRBACManagementService) CountRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) (int64, error) {
	return 0, nil
}

func (s *StubRBACManagementService) AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
	return nil, nil
}
func (s *StubRBACManagementService) RemoveRoleFromUser(ctx context.Context, schemaName string, userID, scopeID string, scopeType string) error {
	return nil
}
func (s *StubRBACManagementService) GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
	if s.GetUserAccessMembersFn != nil {
		return s.GetUserAccessMembersFn(ctx, schemaName, userID)
	}
	return []dto.AccessMemberDTO{}, nil
}
func (s *StubRBACManagementService) GetUserAccessByScope(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	return []dto.AccessMemberDTO{}, nil
}
func (s *StubRBACManagementService) GetScopeMembers(ctx context.Context, schemaName string, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	return []dto.AccessMemberDTO{}, nil
}

func (s *StubRBACManagementService) GetUserPermissions(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.PermissionWithDetails, error) {
	return []dto.PermissionWithDetails{}, nil
}
func (s *StubRBACManagementService) CheckUserPermission(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, resourceCode, actionCode string) (bool, error) {
	return false, nil
}
func (s *StubRBACManagementService) GetUserHighestRole(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) (*dto.AccessRoleDTO, error) {
	return nil, nil
}

func (s *StubRBACManagementService) BulkAssignRoleToUsers(ctx context.Context, schemaName string, req dto.BulkAssignRoleRequest) error {
	return nil
}
func (s *StubRBACManagementService) BulkRemoveRoleFromUsers(ctx context.Context, schemaName string, userIDs []string, scopeType string, scopeID *string, roleID string) error {
	return nil
}

func (s *StubRBACManagementService) GetRBACAnalytics(ctx context.Context, schemaName string) (dto.RBACAnalytics, error) {
	return dto.RBACAnalytics{}, nil
}
func (s *StubRBACManagementService) GetRoleUsageStats(ctx context.Context, schemaName string, roleID uuid.UUID) (dto.RoleUsageStats, error) {
	return dto.RoleUsageStats{}, nil
}
func (s *StubRBACManagementService) GetPermissionUsageStats(ctx context.Context, schemaName string, permissionID uuid.UUID) (dto.PermissionUsageStats, error) {
	return dto.PermissionUsageStats{}, nil
}
func (s *StubRBACManagementService) GetResourceAccessMatrix(ctx context.Context, schemaName string) ([]dto.ResourceAccessMatrix, error) {
	return []dto.ResourceAccessMatrix{}, nil
}

func (s *StubRBACManagementService) ValidateRoleConfiguration(ctx context.Context, schemaName string, roleID uuid.UUID) (dto.RoleValidationResult, error) {
	return dto.RoleValidationResult{}, nil
}
func (s *StubRBACManagementService) AuditUserAccess(ctx context.Context, schemaName string, userID string) (dto.UserAccessAudit, error) {
	return dto.UserAccessAudit{}, nil
}
func (s *StubRBACManagementService) GetOrphanedPermissions(ctx context.Context, schemaName string) ([]tenant.Permission, error) {
	return []tenant.Permission{}, nil
}
func (s *StubRBACManagementService) GetUnusedRoles(ctx context.Context, schemaName string) ([]tenant.AccessRole, error) {
	return []tenant.AccessRole{}, nil
}

func (s *StubRBACManagementService) ProcessUserMemberships(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
	return nil, nil
}
