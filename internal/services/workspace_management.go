package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	app_errors "serenibase/internal/app-errors"
	appConstant "serenibase/internal/constant"
)

type workspaceManagementService struct {
	repo                   *pkg.DatabaseService
	workspaceService       interfaces.WorkspaceService
	workspaceMember        interfaces.WorkspaceMemberService
	baseManagementService  interfaces.BaseManagementService
	tableManagementService interfaces.TableManagementService
}

func NewWorkspaceManagementService(
	repo *pkg.DatabaseService,
	workspaceService interfaces.WorkspaceService,
	workspaceMember interfaces.WorkspaceMemberService,
	baseManagementService interfaces.BaseManagementService,
	tableManagementService interfaces.TableManagementService,
) interfaces.WorkspaceManagementService {
	return &workspaceManagementService{
		repo:                   repo,
		workspaceService:       workspaceService,
		workspaceMember:        workspaceMember,
		baseManagementService:  baseManagementService,
		tableManagementService: tableManagementService,
	}
}

func (s workspaceManagementService) Create(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string, userId string) (dto.WorkspaceResponse, error) {
	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}
	insertedWorkspace, err := s.workspaceService.WorkspaceInsertion(ctx, req, schemaName)
	if err != nil {
		return dto.WorkspaceResponse{}, err
	}

	var workspace dto.WorkspaceResponse
	if err := helpers.StructToStruct(insertedWorkspace, &workspace); err != nil {
		return dto.WorkspaceResponse{}, app_errors.ErrStructToStruct
	}

	baseInsertionData := dto.CreateBaseRequest{
		WorkspaceID: insertedWorkspace.ID.String(),
		Title:       "Default Base",
		Description: helpers.StringPtr(""),
		CreatedBy:   req.CreatedBy,
	}

	insertedBase, err := s.baseManagementService.CreateBase(ctx, baseInsertionData, schemaName, userId)
	if err != nil {
		return dto.WorkspaceResponse{}, err
	}

	var base dto.BaseResponse
	if err := helpers.StructToStruct(insertedBase, &base); err != nil {
		return dto.WorkspaceResponse{}, app_errors.ErrStructToStruct
	}

	tableInsertionData := dto.CreateTableRequest{
		BaseID:      insertedBase.ID.String(),
		WorkspaceID: insertedWorkspace.ID.String(),
		Title:       "Default Table",
		Description: "",
		OrderIndex:  0,
		CreatedBy:   req.CreatedBy,
	}

	insertedTable, err := s.tableManagementService.CreateTableWithDefaults(ctx, tableInsertionData, schemaName)
	if err != nil {
		return dto.WorkspaceResponse{}, err
	}

	if err := helpers.StructToStruct(insertedBase, &base); err != nil {
		return dto.WorkspaceResponse{}, app_errors.ErrStructToStruct
	}

	base.Tables = []dto.TableResponse{
		insertedTable,
	}

	workspace.Bases = []dto.BaseResponse{
		base,
	}

	return workspace, nil
}

func (s workspaceManagementService) GetByID(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
	workspace, err := s.workspaceService.GetWorkspaceByID(ctx, schemaName, id)
	if err != nil {
		return tenant.Workspace{}, err
	}
	return workspace, nil
}

func (s workspaceManagementService) GetAll(ctx context.Context, schemaName string) ([]tenant.Workspace, error) {
	workspaces, err := s.workspaceService.GetAllWorkspaces(ctx, schemaName)
	if err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (s workspaceManagementService) Update(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate, userId string) (tenant.Workspace, error) {
	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}
	updatedWorkspace, err := s.workspaceService.UpdateWorkspace(ctx, schemaName, id, req)
	if err != nil {
		return tenant.Workspace{}, err
	}
	return updatedWorkspace, nil
}

func (s workspaceManagementService) Delete(ctx context.Context, schemaName string, id string) error {
	err := s.workspaceService.DeleteWorkspace(ctx, schemaName, id)
	if err != nil {
		return err
	}
	return nil
}

func (s workspaceManagementService) GetTablesByWorkspaceId(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
	models, err := s.tableManagementService.GetModelByWorkspaceID(ctx, schemaName, workspaceID)
	if err != nil {
		return nil, err
	}

	var tableResponses []dto.TableResponse
	for _, m := range models {
		var modelResponse dto.ModelResponse
		if err := helpers.StructToStruct(m, &modelResponse); err != nil {
			return nil, app_errors.ErrStructToStruct
		}
		tableResponses = append(tableResponses, dto.TableResponse{
			Model: modelResponse,
		})
	}

	return tableResponses, nil
}

func (s workspaceManagementService) GetBasesByWorkspaceId(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error) {
	bases, err := s.baseManagementService.GetAllBasesWithAccess(ctx, schemaName, workspaceMemberData)
	if err != nil {
		return nil, err
	}

	return bases, nil
}

func (s workspaceManagementService) GetAllBasesByWorkspaceId(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
	bases, err := s.baseManagementService.GetBasesByWorkspace(ctx, schemaName, workspaceID)
	if err != nil {
		return nil, err
	}
	return bases, nil
}

func (s workspaceManagementService) RemoveUserFromWorkspace(ctx context.Context, schemaName string, workspaceID string, userID string) error {
	workspaceMemner, err := s.workspaceMember.GetWorkspaceMemberByUserAndWorkspace(ctx, schemaName, userID, workspaceID)
	if err != nil {
		return err
	}

	return s.workspaceMember.DeleteWorkspaceMember(ctx, schemaName, workspaceMemner.ID.String())
}

func (s workspaceManagementService) GetWorkspaceMemberByUser(ctx context.Context, schemaName string, userID string) ([]tenant.WorkspaceMember, error) {
	return s.workspaceMember.GetWorkspaceMemberByUser(ctx, schemaName, userID)
}

func (s workspaceManagementService) GetWorkspaceMembers(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error) {
	return s.workspaceMember.GetWorkspaceMembersByWorkspace(ctx, schemaName, workspaceID)
}

func (s workspaceManagementService) GetBulkWorkspaces(ctx context.Context, schemaName string, workspaceIDs []string) ([]tenant.Workspace, error) {
	return s.workspaceService.GetBulkWorkspaces(ctx, schemaName, workspaceIDs)
}

func (s workspaceManagementService) GetWorkspaceBaseMembers(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error) {

	base, err := s.baseManagementService.GetBaseByID(ctx, schemaName, baseID)
	if err != nil {
		return nil, err
	}

	functionName := "get_workspace_base_users"
	schemaFunctionName := fmt.Sprintf("%s.%s", appConstant.MasterDatabase, functionName)

	args := map[string]interface{}{
		"p_schema_name":  schemaName,
		"p_workspace_id": base.WorkspaceID,
		"p_base_id":      baseID,
	}

	records, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)

	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var result []tenant.WorkspaceMember
	for _, record := range records {
		if rec, ok := record[functionName].(map[string]interface{}); ok {
			var wm tenant.WorkspaceMember
			if err := helpers.MapToStruct(rec, &wm); err == nil {
				result = append(result, wm)
			}
		}
	}

	return result, nil
}

func (s workspaceManagementService) DeleteUserMappings(ctx context.Context, schemaName string, userID string) error {
	err := s.workspaceMember.DeleteUserMappings(ctx, schemaName, userID)
	if err != nil {
		return err
	}

	return nil
}

func (s workspaceManagementService) UpdateWorkspaceMemberBases(ctx context.Context, schemaName string, workspaceID string, userID string, accessLevel string, basesIds string) error {
	// Delegate to workspace member service
	return s.workspaceMember.UpdateWorkspaceMemberBases(ctx, schemaName, workspaceID, userID, accessLevel, basesIds)
}
