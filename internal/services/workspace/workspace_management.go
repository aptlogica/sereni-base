// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"

	"github.com/google/uuid"
)

type workspaceManagementService struct {
	repo                   *pkg.DatabaseService
	workspaceService       interfaces.WorkspaceService
	workspaceMember        interfaces.WorkspaceMemberService
	baseManagementService  interfaces.BaseManagementService
	tableManagementService interfaces.TableManagementService
	rbacManagementService  interfaces.RBACManagementService
}

func NewWorkspaceManagementService(
	repo *pkg.DatabaseService,
	workspaceService interfaces.WorkspaceService,
	workspaceMember interfaces.WorkspaceMemberService,
	baseManagementService interfaces.BaseManagementService,
	tableManagementService interfaces.TableManagementService,
	rbacManagementService interfaces.RBACManagementService,
) interfaces.WorkspaceManagementService {
	return &workspaceManagementService{
		repo:                   repo,
		workspaceService:       workspaceService,
		workspaceMember:        workspaceMember,
		baseManagementService:  baseManagementService,
		tableManagementService: tableManagementService,
		rbacManagementService:  rbacManagementService,
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

	_, err = s.baseManagementService.CreateBase(ctx, baseInsertionData, schemaName, userId)
	if err != nil {
		return dto.WorkspaceResponse{}, err
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

// isWorkspaceLevelRole checks if the role is a workspace-level role that grants access to all bases
func isWorkspaceLevelRole(role string) bool {
	return role == appConstant.RBACRoleNames.Owner ||
		role == appConstant.RBACRoleNames.CoOwner
}

// getAllBasesWithWorkspaceRole retrieves all bases in a workspace for users with workspace-level roles
func (s workspaceManagementService) getAllBasesWithWorkspaceRole(ctx context.Context, schemaName string, workspaceID string, role string) ([]dto.BaseResponse, error) {
	// Get all bases in workspace
	bases, err := s.baseManagementService.GetBasesByWorkspace(ctx, schemaName, workspaceID)
	if err != nil {
		return nil, err
	}

	// Convert to BaseResponse and add workspace-level access level
	var response []dto.BaseResponse
	for _, base := range bases {
		var baseResp dto.BaseResponse
		if err := helpers.StructToStruct(base, &baseResp); err != nil {
			return nil, app_errors.ErrStructToStruct
		}
		// Workspace-level users get the workspace role as access level
		baseResp.AccessLevel = role
		response = append(response, baseResp)
	}
	return response, nil
}

// normalizeBaseID turns a base id value from a query row into a comparable key
func normalizeBaseID(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(t))
	case []byte:
		return strings.ToLower(strings.TrimSpace(string(t)))
	default:
		return strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
	}
}

// markAPIEnabled sets APIEnabled on the bases whose public API is switched on, using one lookup.
// A failed lookup is logged and leaves every flag false, so the list never fails because of it.
func (s workspaceManagementService) markAPIEnabled(schemaName string, bases []dto.BaseResponse) {
	if len(bases) == 0 {
		return
	}
	ids := make([]string, 0, len(bases))
	for _, b := range bases {
		ids = append(ids, b.ID.String())
	}
	lg := logger.Get()
	rows, err := s.repo.TableService.GetTableData(tenant.BaseAPI{}.TableName(schemaName), dbModels.QueryParams{
		Select: []string{"base_id"},
		Filters: []dbModels.QueryFilter{
			{Column: "base_id", Operator: "in", Value: ids},
			{Column: "enabled", Operator: "eq", Value: true},
		},
	})
	if err != nil {
		lg.Warn().Err(err).Msg("Failed to look up API-enabled bases")
		return
	}
	// Match ids ignoring case and surrounding spaces, and accept text returned as bytes
	enabled := make(map[string]bool, len(rows))
	for _, row := range rows {
		enabled[normalizeBaseID(row["base_id"])] = true
	}
	// TEMPORARY debug: remove once the API tag is confirmed working
	lg.Debug().Strs("searched_base_ids", ids).Int("enabled_rows", len(rows)).Msg("API-enabled base lookup")
	for i := range bases {
		bases[i].APIEnabled = enabled[normalizeBaseID(bases[i].ID.String())]
	}
}

// getSystemLevelRole checks if user has system-level Owner or CoOwner role
func (s workspaceManagementService) getSystemLevelRole(ctx context.Context, schemaName string, accessMembers []dto.AccessMemberDTO) string {
	for _, member := range accessMembers {
		if member.ScopeType == appConstant.ScopeLevels.System && member.ScopeID == nil {
			// Get the role name from role_id
			roleName := s.getRoleName(ctx, schemaName, member.RoleID)
			if roleName == appConstant.RBACRoleNames.Owner || roleName == appConstant.RBACRoleNames.CoOwner {
				return roleName
			}
		}
	}
	return ""
}

// checkWorkspaceLevelAccess checks if user has workspace-level access and returns the role name
func (s workspaceManagementService) checkWorkspaceLevelAccess(ctx context.Context, schemaName string, workspaceID string, accessMembers []dto.AccessMemberDTO) (string, bool) {
	for _, member := range accessMembers {
		if s.isWorkspaceMember(member, workspaceID) {
			if roleName := s.getRoleName(ctx, schemaName, member.RoleID); roleName != "" {
				return roleName, true
			}
		}
	}
	return "", false
}

// isWorkspaceMember checks if the member has workspace-level access for the given workspace
func (s workspaceManagementService) isWorkspaceMember(member dto.AccessMemberDTO, workspaceID string) bool {
	return member.ScopeType == "workspace" && member.ScopeID != nil && *member.ScopeID == workspaceID
}

// getRoleName retrieves the role name from a role ID, returns empty string if invalid
func (s workspaceManagementService) getRoleName(ctx context.Context, schemaName string, roleID string) string {
	if roleID == "" {
		return ""
	}

	roleUUID, parseErr := uuid.Parse(roleID)
	if parseErr != nil {
		return ""
	}

	roleData, roleErr := s.rbacManagementService.GetRoleByID(ctx, schemaName, roleUUID)
	if roleErr != nil {
		return ""
	}

	return roleData.Name
}

// buildBaseAccessMap creates a map of base IDs to role names for base-level access
func (s workspaceManagementService) buildBaseAccessMap(ctx context.Context, schemaName string, workspaceID string, accessMembers []dto.AccessMemberDTO) map[string]string {
	baseAccessMap := make(map[string]string)
	for _, member := range accessMembers {
		if s.isBaseMember(member, workspaceID) {
			if roleName := s.getRoleName(ctx, schemaName, member.RoleID); roleName != "" {
				baseAccessMap[*member.ScopeID] = roleName
			} else {
				baseAccessMap[*member.ScopeID] = member.RoleID
			}
		}
	}
	return baseAccessMap
}

// isBaseMember checks if the member has base-level access for the given workspace
func (s workspaceManagementService) isBaseMember(member dto.AccessMemberDTO, workspaceID string) bool {
	return member.ScopeType == "base" &&
		member.WorkspaceID != nil &&
		*member.WorkspaceID == workspaceID &&
		member.ScopeID != nil
}

// getBasesWithAccess retrieves bases with their access levels
func (s workspaceManagementService) getBasesWithAccess(ctx context.Context, schemaName string, baseAccessMap map[string]string, userID string, workspaceID string) ([]dto.BaseResponse, error) {
	var response []dto.BaseResponse
	for baseID, roleName := range baseAccessMap {
		base, err := s.baseManagementService.GetBaseByID(ctx, schemaName, baseID)
		if err != nil {
			continue
		}

		var baseResp dto.BaseResponse
		if err := helpers.StructToStruct(base, &baseResp); err != nil {
			continue
		}
		baseResp.AccessLevel = roleName
		response = append(response, baseResp)
	}

	return response, nil
}

// GetAllBasesByWorkspaceId returns the bases the user can see in a workspace, each flagged with
// whether its public API is on (one lookup for the whole list, whichever access branch built it).
func (s workspaceManagementService) GetAllBasesByWorkspaceId(ctx context.Context, schemaName string, workspaceID string, role string, userID string) ([]dto.BaseResponse, error) {
	response, err := s.collectBasesByWorkspaceId(ctx, schemaName, workspaceID, role, userID)
	if err != nil {
		return nil, err
	}
	fmt.Println("response: ", response)
	s.markAPIEnabled(schemaName, response)
	fmt.Println("response: ", response)
	return response, nil
}

// collectBasesByWorkspaceId builds the visible base list by access level (system, workspace or base)
func (s workspaceManagementService) collectBasesByWorkspaceId(ctx context.Context, schemaName string, workspaceID string, role string, userID string) ([]dto.BaseResponse, error) {
	// Get user's access members to check for system/workspace/base-level access
	accessMembers, err := s.rbacManagementService.GetUserAccessMembers(ctx, schemaName, userID)
	if err != nil {
		return nil, err
	}

	// Check if user has system-level Owner or CoOwner role - they can see all bases
	if systemRole := s.getSystemLevelRole(ctx, schemaName, accessMembers); systemRole != "" {
		return s.getAllBasesWithWorkspaceRole(ctx, schemaName, workspaceID, systemRole)
	}

	// Check if user has workspace-level role from JWT token - they can see all bases in workspace
	if isWorkspaceLevelRole(role) {
		return s.getAllBasesWithWorkspaceRole(ctx, schemaName, workspaceID, role)
	}

	// Check if user has workspace-level access in accessMembers
	if workspaceRole, hasWorkspaceAccess := s.checkWorkspaceLevelAccess(ctx, schemaName, workspaceID, accessMembers); hasWorkspaceAccess {
		return s.getAllBasesWithWorkspaceRole(ctx, schemaName, workspaceID, workspaceRole)
	}

	// For base-member and base-read: Get only bases where user has explicit access
	baseAccessMap := s.buildBaseAccessMap(ctx, schemaName, workspaceID, accessMembers)

	// If no base access found, return empty list
	if len(baseAccessMap) == 0 {
		return []dto.BaseResponse{}, nil
	}

	// Get all bases with user's access
	return s.getBasesWithAccess(ctx, schemaName, baseAccessMap, userID, workspaceID)
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
		return nil, app_errors.LogDatabaseError(err, "failed to get workspace base users")
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
