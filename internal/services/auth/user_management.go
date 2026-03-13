// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	"mime/multipart"
	"path/filepath"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"

	"serenibase/internal/providers/logger"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"strings"
	"time"

	app_errors "serenibase/internal/app-errors"
	authProviderInterface "serenibase/internal/providers/auth"

	appConstant "serenibase/internal/constant"

	"github.com/google/uuid"
)

type userManagementService struct {
	repo                       *pkg.DatabaseService
	userService                interfaces.UserService
	assetManagementService     interfaces.AssetManagementService
	userResetTokenService      interfaces.UserResetTokenService
	workspaceManagementService interfaces.WorkspaceManagementService
	rbacManagementService      interfaces.RBACManagementService
	authProvider               authProviderInterface.AuthProvider
}

func NewUserManagementService(
	repo *pkg.DatabaseService,
	userService interfaces.UserService,
	assetManagementService interfaces.AssetManagementService,
	userResetTokenService interfaces.UserResetTokenService,
	workspaceManagementService interfaces.WorkspaceManagementService,
	rbacManagementService interfaces.RBACManagementService,
	authProvider authProviderInterface.AuthProvider,
) interfaces.UserManagementService {
	return &userManagementService{
		repo:                       repo,
		userService:                userService,
		assetManagementService:     assetManagementService,
		userResetTokenService:      userResetTokenService,
		workspaceManagementService: workspaceManagementService,
		rbacManagementService:      rbacManagementService,
		authProvider:               authProvider,
	}
}

func (s *userManagementService) GetUserProfileByID(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	lg := logger.Get()
	user, err := s.userService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}
	lg.Debug().Interface("user", user).Msg("Retrieved user profile")

	var userResponse dto.UserResponse
	err = helpers.StructToStruct(user, &userResponse) // Assume this helper exists, else use manual mapping
	if err != nil {
		return dto.UserResponse{}, app_errors.ErrStructToStruct
	}
	// // Convert DateOfBirth to string if present
	// if user.DateOfBirth != nil {
	// 	dateStr := user.DateOfBirth.Format("2006-01-02")
	// 	userResponse.DateOfBirth = &dateStr
	// }
	return userResponse, nil
}

func (s *userManagementService) UpdateUserProfile(ctx context.Context, schema string, userID string, updateData dto.UpdateUserProfileRequest) (dto.UserResponse, error) {
	lg := logger.Get()
	updateData.UpdatedAt = time.Now()

	updateFields := updateData.Map()
	if len(updateFields) == 0 {
		return dto.UserResponse{}, app_errors.InvalidPayload
	}

	// // Handle DateOfBirth parsing if provided
	// if updateData.DateOfBirth != nil && *updateData.DateOfBirth != "" {
	// 	parsed, err := time.Parse("2006-01-02", *updateData.DateOfBirth)
	// 	if err != nil {
	// 		return dto.UserResponse{}, fmt.Errorf("invalid date of birth format: %w", err)
	// 	}
	// 	// keep only date: YYYY-MM-DD 00:00:00 UTC
	// 	onlyDate := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	// 	updateFields["date_of_birth"] = onlyDate
	// }

	updatedUser, err := s.userService.UpdateUser(ctx, schema, userID, updateFields)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to update user")
		return dto.UserResponse{}, err
	}

	var userResponse dto.UserResponse
	err = helpers.StructToStruct(updatedUser, &userResponse)
	if err != nil {
		return dto.UserResponse{}, app_errors.ErrStructToStruct
	}
	// // Convert DateOfBirth to string if present
	// if updatedUser.DateOfBirth != nil {
	// 	dateStr := updatedUser.DateOfBirth.Format("2006-01-02")
	// 	userResponse.DateOfBirth = &dateStr
	// }

	return userResponse, nil
}

func (s *userManagementService) UpdatePassword(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) (tenant.User, error) {
	if updateData.OldPassword == updateData.NewPassword {
		return tenant.User{}, app_errors.NewPasswordInvalid
	}

	// Fetch user by ID
	user, err := s.userService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return tenant.User{}, err
	}

	// Check if old password matches
	if !helpers.CheckPasswordHash(updateData.OldPassword, user.Password) {
		return tenant.User{}, app_errors.InvalidOldPassword
	}

	// Hash the new password
	hashedPassword, err := helpers.HashPassword(updateData.NewPassword)
	if err != nil {
		return tenant.User{}, app_errors.ErrHashed
	}

	updateFields := map[string]interface{}{
		"password":            hashedPassword,
		"password_changed_at": time.Now(),
		"last_modified_time":  time.Now(),
	}

	// Update password in tenant schema only
	updatedUser, err := s.userService.UpdateUser(ctx, schema, userID, updateFields)
	if err != nil {
		return tenant.User{}, err
	}

	return updatedUser, nil
}

func (s *userManagementService) AddAvatar(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error) {
	lg := logger.Get()
	err := s.deleteAvatarIfExists(ctx, schema, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	if fileHeader == nil {
		return dto.UserResponse{}, app_errors.InvalidPayload
	}

	filename := fileHeader.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	if !allowedExtensions[ext] {
		return dto.UserResponse{}, app_errors.NewPasswordInvalid // Consider app_errors.InvalidPayload or a new error for unsupported file type.
	}

	uploadReq := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}
	assets, err := s.assetManagementService.Upload(ctx, uploadReq, schema)
	if err != nil || len(assets) == 0 {
		lg.Error().Stack().Err(err).Msg("Failed to upload avatar asset")
		return dto.UserResponse{}, err
	}
	avatarPath := assets[0].Url

	updateFields := map[string]interface{}{
		"avatar":             avatarPath,
		"last_modified_time": time.Now(),
	}
	updatedUser, err := s.userService.UpdateUser(ctx, schema, userID, updateFields)
	if err != nil {
		return dto.UserResponse{}, err
	}

	var userResponse dto.UserResponse
	err = helpers.StructToStruct(updatedUser, &userResponse)
	if err != nil {
		return dto.UserResponse{}, app_errors.ErrStructToStruct
	}
	// // Convert DateOfBirth to string if present
	// if updatedUser.DateOfBirth != nil {
	// 	dateStr := updatedUser.DateOfBirth.Format("2006-01-02")
	// 	userResponse.DateOfBirth = &dateStr
	// }

	return userResponse, nil
}

func (s *userManagementService) deleteAvatarIfExists(ctx context.Context, schema string, userID string) error {
	user, err := s.userService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return err
	}

	if user.Avatar != "" {
		asset, err := s.assetManagementService.GetAssetByURL(ctx, schema, user.Avatar)
		if err == nil {
			if asset.Url == user.Avatar {
				avatarAssetId := asset.ID.String()
				err = s.assetManagementService.DeleteAsset(ctx, avatarAssetId, schema)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *userManagementService) RemoveAvatar(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	err := s.deleteAvatarIfExists(ctx, schema, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	updateFields := map[string]interface{}{
		"avatar":             "",
		"last_modified_time": time.Now(),
	}

	updatedUser, err := s.userService.UpdateUser(ctx, schema, userID, updateFields)
	if err != nil {
		return dto.UserResponse{}, err
	}

	var userResponse dto.UserResponse
	err = helpers.StructToStruct(updatedUser, &userResponse)
	if err != nil {
		return dto.UserResponse{}, app_errors.ErrStructToStruct
	}

	return userResponse, nil
}

func (s *userManagementService) GetUserByEmail(ctx context.Context, schema string, email string) (tenant.User, error) {
	return s.userService.GetUserByEmail(ctx, schema, email)
}

func (s *userManagementService) CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
	return s.userService.CreateUser(ctx, schema, req)
}

func (s *userManagementService) UpdateUser(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
	return s.userService.UpdateUser(ctx, schema, id, updateData)
}

func (s *userManagementService) GetUserByID(ctx context.Context, schema string, id string) (tenant.User, error) {
	return s.userService.GetUserByID(ctx, schema, id)
}

func (s *userManagementService) GetAllUsers(ctx context.Context, schema string) ([]tenant.User, error) {
	return s.userService.GetAllUsers(ctx, schema)
}

func (s *userManagementService) GetWorkspaces(ctx context.Context, schema string, userID string, roles string) ([]dto.UserWorkspaceResponse, error) {
	// Get user's workspace access from RBAC access_members table
	// This includes both workspace-level and base-level access
	accessMembers, err := s.rbacManagementService.GetUserAccessMembers(ctx, schema, userID)
	if err != nil {
		return []dto.UserWorkspaceResponse{}, nil
	}

	// Check if user has system-level Owner or CoOwner role in access_members
	systemRole := s.getSystemLevelRole(ctx, schema, accessMembers)
	if systemRole == appConstant.RBACRoleNames.CoOwner || systemRole == appConstant.RBACRoleNames.Owner {
		return s.getAllWorkspacesForOwner(ctx, schema, systemRole)
	}

	workspaceAccess := s.buildWorkspaceAccessMapForWorkspaces(accessMembers)

	// If no workspace access found, return empty list
	if len(workspaceAccess) == 0 {
		return []dto.UserWorkspaceResponse{}, nil
	}

	// Get unique workspace IDs
	workspaceIDs := make([]string, 0, len(workspaceAccess))
	for wsID := range workspaceAccess {
		workspaceIDs = append(workspaceIDs, wsID)
	}

	// Get workspace details in bulk
	workspaces, err := s.workspaceManagementService.GetBulkWorkspaces(ctx, schema, workspaceIDs)
	if err != nil {
		return nil, err
	}

	// Build response with workspace details and access roles
	return s.buildWorkspaceResponses(ctx, schema, workspaces, workspaceAccess)
}

func (s *userManagementService) getAllWorkspacesForOwner(ctx context.Context, schema string, roles string) ([]dto.UserWorkspaceResponse, error) {
	workspaces, err := s.workspaceManagementService.GetAll(ctx, schema)
	if err != nil {
		return nil, err
	}
	var res []dto.UserWorkspaceResponse
	for _, ws := range workspaces {
		var wsResp dto.UserWorkspaceResponse
		err := helpers.StructToStruct(ws, &wsResp)
		if err != nil {
			return nil, app_errors.ErrStructToStruct
		}
		wsResp.AccessLevel = roles
		res = append(res, wsResp)
	}
	return res, nil
}

// getSystemLevelRole checks if user has system-level Owner or CoOwner role
func (s *userManagementService) getSystemLevelRole(ctx context.Context, schema string, accessMembers []dto.AccessMemberDTO) string {
	for _, member := range accessMembers {
		if member.ScopeType == appConstant.ScopeLevels.System && member.ScopeID == nil {
			// Get the role name from role_id
			roleName := s.getRoleNameByID(ctx, schema, member.RoleID)
			if roleName == appConstant.RBACRoleNames.Owner || roleName == appConstant.RBACRoleNames.CoOwner {
				return roleName
			}
		}
	}
	return ""
}

func (s *userManagementService) buildWorkspaceAccessMapForWorkspaces(accessMembers []dto.AccessMemberDTO) map[string]string {
	// Map to store workspace IDs and their access levels
	// Key: workspace_id, Value: role_id or "base"
	workspaceAccess := map[string]string{}
	for _, member := range accessMembers {
		switch member.ScopeType {
		case "base":
			// Base-level access - get workspace_id from workspace_id column
			if member.WorkspaceID != nil && *member.WorkspaceID != "" {
				workspaceID := *member.WorkspaceID
				// Only set if not already set with workspace-level access (workspace-level has priority)
				if _, exists := workspaceAccess[workspaceID]; !exists {
					workspaceAccess[workspaceID] = "base"
				}
			}
		case "workspace":
			// Workspace-level access - get workspace_id from scope_id column
			if member.ScopeID != nil && *member.ScopeID != "" {
				workspaceID := *member.ScopeID
				// Workspace-level access has priority, always set/override
				workspaceAccess[workspaceID] = member.RoleID
			}
		}
	}
	return workspaceAccess
}

func (s *userManagementService) buildWorkspaceResponses(ctx context.Context, schema string, workspaces []tenant.Workspace, workspaceAccess map[string]string) ([]dto.UserWorkspaceResponse, error) {
	var res []dto.UserWorkspaceResponse
	for _, ws := range workspaces {
		var wsResp dto.UserWorkspaceResponse
		err := helpers.StructToStruct(ws, &wsResp)
		if err != nil {
			return nil, app_errors.ErrStructToStruct
		}

		// Get role ID or access level for this workspace
		roleIDOrLevel, exists := workspaceAccess[wsResp.ID.String()]
		if !exists {
			continue
		}

		wsResp.AccessLevel = s.determineAccessLevel(ctx, schema, roleIDOrLevel)
		res = append(res, wsResp)
	}
	return res, nil
}

// determineAccessLevel determines the access level from a role ID or direct level
func (s *userManagementService) determineAccessLevel(ctx context.Context, schema string, roleIDOrLevel string) string {
	// If it's "base", set as access level directly
	if roleIDOrLevel == "base" {
		return "base"
	}

	// Try to parse as UUID and get role name
	roleUUID, parseErr := uuid.Parse(roleIDOrLevel)
	if parseErr != nil {
		return roleIDOrLevel
	}

	role, roleErr := s.rbacManagementService.GetRoleByID(ctx, schema, roleUUID)
	if roleErr != nil {
		return roleIDOrLevel
	}

	return role.Name
}

func (s *userManagementService) GetBulkUsers(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
	return s.userService.GetBulkUsers(ctx, schema, ids)
}

func (s *userManagementService) GetUsersWithRole(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	lg := logger.Get()
	functionName := "get_users_with_role"
	schemaFunctionName := fmt.Sprintf("%s.%s", appConstant.MasterDatabase, functionName)

	records, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		nil,
	)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get users with role")
	}
	var result []dto.UserWithRole
	for _, record := range records {
		if rec, ok := record[functionName].(map[string]interface{}); ok {
			var user dto.UserWithRole
			if err := helpers.MapToStruct(rec, &user); err == nil {
				result = append(result, user)
			} else {
				lg.Warn().Err(err).Msg("Failed to convert record to UserWithRole")
			}
		}
	}
	lg.Debug().Interface("result", result).Msg("Retrieved users with roles")
	return result, nil
}

func (s *userManagementService) GetActiveUsersForAssign(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	functionName := "get_active_users_for_assign"
	schemaFunctionName := fmt.Sprintf("%s.%s", appConstant.MasterDatabase, functionName)

	records, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		nil,
	)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get active users for assign")
	}
	var result []dto.UserWithRole
	for _, record := range records {
		if rec, ok := record[functionName].(map[string]interface{}); ok {
			var user dto.UserWithRole
			if err := helpers.MapToStruct(rec, &user); err == nil {
				result = append(result, user)
			}
		}
	}
	return result, nil
}

func (s *userManagementService) DeleteUserCompletely(ctx context.Context, schema string, userID string) error {
	return s.userService.DeleteUser(ctx, schema, userID)

}

func (s *userManagementService) GetUserAccessDetails(ctx context.Context, schema string, userID string, roles string, workspaceID string) (dto.UserAccessDetailsResponse, error) {
	response := dto.UserAccessDetailsResponse{
		Workspaces: []dto.WorkspaceAccessInfo{},
	}

	// Get workspace memberships for the target user
	memberships, err := s.workspaceManagementService.GetWorkspaceMemberByUser(ctx, schema, userID)
	if err != nil {
		if err == app_errors.WorkspaceMemberNotFound {
			return response, nil
		}
		return dto.UserAccessDetailsResponse{}, err
	}

	// Filter memberships by workspace_id if provided
	memberships = s.filterMembershipsByWorkspaceID(memberships, workspaceID)
	if len(memberships) == 0 {
		return response, nil
	}

	// Build workspace IDs and access map from memberships
	workspaceIDs, workspaceAccess, membershipMap := s.buildWorkspaceData(memberships)

	// Get workspaces
	workspaces, err := s.workspaceManagementService.GetBulkWorkspaces(ctx, schema, workspaceIDs)
	if err != nil {
		return dto.UserAccessDetailsResponse{}, err
	}

	// Build workspace access info with bases
	for _, ws := range workspaces {
		accessLevel := workspaceAccess[ws.ID.String()]
		membership := membershipMap[ws.ID.String()]

		baseAccessInfos, err := s.getBaseAccessInfos(ctx, schema, accessLevel, membership)
		if err != nil {
			return dto.UserAccessDetailsResponse{}, err
		}

		response.Workspaces = append(response.Workspaces, dto.WorkspaceAccessInfo{
			ID:          ws.ID,
			Title:       ws.Title,
			AccessLevel: accessLevel,
			Bases:       baseAccessInfos,
		})
	}

	return response, nil
}

func (s *userManagementService) filterMembershipsByWorkspaceID(memberships []tenant.WorkspaceMember, workspaceID string) []tenant.WorkspaceMember {
	if workspaceID == "" {
		return memberships
	}
	filteredMemberships := []tenant.WorkspaceMember{}
	for _, membership := range memberships {
		if membership.WorkspaceID == workspaceID {
			filteredMemberships = append(filteredMemberships, membership)
			break
		}
	}
	return filteredMemberships
}

func (s *userManagementService) buildWorkspaceData(memberships []tenant.WorkspaceMember) ([]string, map[string]string, map[string]*tenant.WorkspaceMember) {
	workspaceIDs := make([]string, 0, len(memberships))
	workspaceAccess := make(map[string]string)
	membershipMap := make(map[string]*tenant.WorkspaceMember)
	for i := range memberships {
		workspaceIDs = append(workspaceIDs, memberships[i].WorkspaceID)
		workspaceAccess[memberships[i].WorkspaceID] = memberships[i].AccessLevel
		membershipMap[memberships[i].WorkspaceID] = &memberships[i]
	}
	return workspaceIDs, workspaceAccess, membershipMap
}

func (s *userManagementService) getBaseAccessInfos(ctx context.Context, schema string, accessLevel string, membership *tenant.WorkspaceMember) ([]dto.BaseAccessInfo, error) {
	baseAccessInfos := []dto.BaseAccessInfo{}
	if accessLevel == appConstant.RBACRoleNames.Owner && membership != nil {
		bases, err := s.workspaceManagementService.GetBasesByWorkspaceId(ctx, schema, membership)
		if err != nil && err != app_errors.BaseNotFound {
			return nil, err
		}
		for _, base := range bases {
			if membership.BasesIds == "*" || strings.Contains(membership.BasesIds, base.ID.String()) {
				baseAccessInfos = append(baseAccessInfos, dto.BaseAccessInfo{
					ID:    base.ID,
					Title: base.Title,
				})
			}
		}
	}
	return baseAccessInfos, nil
}

// If scopeID is provided, only returns access for that specific scope (workspace or base)
func (s *userManagementService) GetUserRolesAndAccess(ctx context.Context, schema string, userID string, scopeID *string) ([]dto.UserRolesAccessResponse, error) {
	accessMembers := s.fetchAccessMembers(ctx, schema, userID)
	if len(accessMembers) == 0 {
		return []dto.UserRolesAccessResponse{}, nil
	}

	workspaceAccessMap := s.buildWorkspaceAccessMap(ctx, schema, accessMembers, scopeID)
	return s.workspaceAccessMapToSlice(workspaceAccessMap), nil
}

func (s *userManagementService) fetchAccessMembers(ctx context.Context, schema string, userID string) []dto.AccessMemberDTO {
	lg := logger.Get()
	accessMembers, err := s.rbacManagementService.GetUserAccessMembers(ctx, schema, userID)
	if err != nil {
		lg.Error().Err(err).Str("userID", userID).Msg("Failed to get user access members")
		return nil
	}
	return accessMembers
}

func (s *userManagementService) buildWorkspaceAccessMap(
	ctx context.Context,
	schema string,
	accessMembers []dto.AccessMemberDTO,
	scopeID *string,
) map[string]*dto.UserRolesAccessResponse {
	workspaceAccessMap := make(map[string]*dto.UserRolesAccessResponse)

	for _, member := range accessMembers {
		if !s.matchesScopeFilter(member, scopeID) {
			continue
		}

		switch member.ScopeType {
		case "workspace":
			s.handleWorkspaceAccess(ctx, schema, member, workspaceAccessMap)
		case "base":
			s.handleBaseAccess(ctx, schema, member, workspaceAccessMap)
		}
	}

	return workspaceAccessMap
}

func (s *userManagementService) matchesScopeFilter(member dto.AccessMemberDTO, scopeID *string) bool {
	if scopeID == nil || *scopeID == "" {
		return true
	}

	if member.ScopeID != nil && *member.ScopeID == *scopeID {
		return true
	}

	if member.WorkspaceID != nil && *member.WorkspaceID == *scopeID {
		return true
	}

	return false
}

func (s *userManagementService) handleWorkspaceAccess(
	ctx context.Context,
	schema string,
	member dto.AccessMemberDTO,
	workspaceAccessMap map[string]*dto.UserRolesAccessResponse,
) {

	if member.ScopeID == nil || *member.ScopeID == "" {
		return
	}

	workspaceID := *member.ScopeID
	workspace, err := s.getWorkspaceByID(ctx, schema, workspaceID)
	if err != nil {
		return
	}

	roleName := s.getRoleNameByID(ctx, schema, member.RoleID)

	if _, exists := workspaceAccessMap[workspaceID]; !exists {
		workspaceAccessMap[workspaceID] = &dto.UserRolesAccessResponse{
			WorkspaceId:   workspaceID,
			WorkspaceName: workspace.Title,
			Access:        roleName,
			Bases:         []dto.BaseRoleAccess{},
		}
		return
	}

	workspaceAccessMap[workspaceID].Access = roleName
}

func (s *userManagementService) handleBaseAccess(
	ctx context.Context,
	schema string,
	member dto.AccessMemberDTO,
	workspaceAccessMap map[string]*dto.UserRolesAccessResponse,
) {
	if member.ScopeID == nil || *member.ScopeID == "" || member.WorkspaceID == nil || *member.WorkspaceID == "" {
		return
	}

	baseID := *member.ScopeID
	workspaceID := *member.WorkspaceID

	base, err := s.getBaseByID(ctx, schema, baseID)
	if err != nil {
		return
	}

	roleName := s.getRoleNameByID(ctx, schema, member.RoleID)

	if _, exists := workspaceAccessMap[workspaceID]; !exists {
		workspace, err := s.getWorkspaceByID(ctx, schema, workspaceID)
		if err != nil {
			return
		}

		workspaceAccessMap[workspaceID] = &dto.UserRolesAccessResponse{
			WorkspaceId:   workspaceID,
			WorkspaceName: workspace.Title,
			Access:        "",
			Bases:         []dto.BaseRoleAccess{},
		}
	}

	workspaceAccessMap[workspaceID].Bases = append(
		workspaceAccessMap[workspaceID].Bases,
		dto.BaseRoleAccess{BaseId: baseID, BaseName: base.Title, Access: roleName},
	)
}

func (s *userManagementService) workspaceAccessMapToSlice(workspaceAccessMap map[string]*dto.UserRolesAccessResponse) []dto.UserRolesAccessResponse {
	response := make([]dto.UserRolesAccessResponse, 0, len(workspaceAccessMap))
	for _, wsAccess := range workspaceAccessMap {
		response = append(response, *wsAccess)
	}
	return response
}

// getWorkspaceByID is a helper method to fetch workspace by ID
func (s *userManagementService) getWorkspaceByID(ctx context.Context, schema string, workspaceID string) (tenant.Workspace, error) {
	tableName := tenant.Workspace{}.TableName(schema)
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    workspaceID,
			},
		},
		Limit: &limit,
	}

	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.Workspace{}, err
	}

	if len(data) == 0 {
		return tenant.Workspace{}, app_errors.ErrRecordNotFound
	}

	var workspace tenant.Workspace
	if err := helpers.MapToStruct(data[0], &workspace); err != nil {
		return tenant.Workspace{}, err
	}

	return workspace, nil
}

// getBaseByID is a helper method to fetch base by ID
func (s *userManagementService) getBaseByID(ctx context.Context, schema string, baseID string) (tenant.Base, error) {
	tableName := tenant.Base{}.TableName(schema)
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    baseID,
			},
		},
		Limit: &limit,
	}

	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.Base{}, err
	}

	if len(data) == 0 {
		return tenant.Base{}, app_errors.ErrRecordNotFound
	}

	var base tenant.Base
	if err := helpers.MapToStruct(data[0], &base); err != nil {
		return tenant.Base{}, err
	}

	return base, nil
}

// getRoleNameByID is a helper method to get role name by role ID
func (s *userManagementService) getRoleNameByID(ctx context.Context, schema string, roleID string) string {
	if roleID == "" {
		return ""
	}

	// Try to parse as UUID
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return roleID
	}

	// Get role details
	role, err := s.rbacManagementService.GetRoleByID(ctx, schema, roleUUID)
	if err != nil {
		return roleID
	}

	return role.Name
}
