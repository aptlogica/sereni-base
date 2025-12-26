package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
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
)

type userManagementService struct {
	repo                       *pkg.DatabaseService
	userService                interfaces.UserService
	assetManagementService     interfaces.AssetManagementService
	userResetTokenService      interfaces.UserResetTokenService
	workspaceManagementService interfaces.WorkspaceManagementService
	authProvider               authProviderInterface.AuthProvider
}

func NewUserManagementService(
	repo *pkg.DatabaseService,
	userService interfaces.UserService,
	assetManagementService interfaces.AssetManagementService,
	userResetTokenService interfaces.UserResetTokenService,
	workspaceManagementService interfaces.WorkspaceManagementService,
	authProvider authProviderInterface.AuthProvider,
) interfaces.UserManagementService {
	return &userManagementService{
		repo:                       repo,
		userService:                userService,
		assetManagementService:     assetManagementService,
		userResetTokenService:      userResetTokenService,
		workspaceManagementService: workspaceManagementService,
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
	lg := logger.Get()
	lg.Debug().Str("roles", roles).Msg("Fetching workspaces for user")
	if roles == appConstant.RBACRoleNames.CoOwner || roles == appConstant.RBACRoleNames.Owner {
		fmt.Println("User is Owner or CoOwner, fetching all workspaces")
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

	memberships, err := s.workspaceManagementService.GetWorkspaceMemberByUser(ctx, schema, userID)
	if err != nil {
		if err == app_errors.WorkspaceMemberNotFound {
			return []dto.UserWorkspaceResponse{}, nil
		}
		return nil, err
	}

	workspaceIDs := make([]string, 0, len(memberships))
	workspaceAccess := map[string]string{}
	for _, membership := range memberships {
		workspaceIDs = append(workspaceIDs, membership.WorkspaceID)
		workspaceAccess[membership.WorkspaceID] = membership.AccessLevel
	}

	workspaces, err := s.workspaceManagementService.GetBulkWorkspaces(ctx, schema, workspaceIDs)
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
		wsResp.AccessLevel = workspaceAccess[wsResp.ID.String()]
		res = append(res, wsResp)
	}

	return res, nil
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
		return nil, app_errors.DatabaseError
	}
	fmt.Println("result records: ---- ", records)
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
	if workspaceID != "" {
		filteredMemberships := []tenant.WorkspaceMember{}
		for _, membership := range memberships {
			if membership.WorkspaceID == workspaceID {
				filteredMemberships = append(filteredMemberships, membership)
				break
			}
		}
		memberships = filteredMemberships

		// If no membership found for the specified workspace, return empty response
		if len(memberships) == 0 {
			return response, nil
		}
	}

	// Build workspace IDs and access map from memberships
	workspaceIDs := make([]string, 0, len(memberships))
	workspaceAccess := make(map[string]string)
	membershipMap := make(map[string]*tenant.WorkspaceMember)

	for i := range memberships {
		workspaceIDs = append(workspaceIDs, memberships[i].WorkspaceID)
		workspaceAccess[memberships[i].WorkspaceID] = memberships[i].AccessLevel
		membershipMap[memberships[i].WorkspaceID] = &memberships[i]
	}

	// Get workspaces
	workspaces, err := s.workspaceManagementService.GetBulkWorkspaces(ctx, schema, workspaceIDs)
	if err != nil {
		return dto.UserAccessDetailsResponse{}, err
	}

	// Build workspace access info with bases
	for _, ws := range workspaces {
		accessLevel := workspaceAccess[ws.ID.String()]
		membership := membershipMap[ws.ID.String()]

		// Only get bases for limited_access users
		baseAccessInfos := []dto.BaseAccessInfo{}

		// For full_access and admin users, return empty bases array (they have access to all bases)
		if accessLevel == appConstant.RBACRoleNames.Owner && membership != nil {
			// Get bases only for limited access users
			bases, err := s.workspaceManagementService.GetBasesByWorkspaceId(ctx, schema, membership)
			if err != nil && err != app_errors.BaseNotFound {
				return dto.UserAccessDetailsResponse{}, err
			}

			// Build base access info for limited access users
			for _, base := range bases {
				// Check if this base is in the user's allowed bases
				if membership.BasesIds == "*" || strings.Contains(membership.BasesIds, base.ID.String()) {
					baseAccessInfos = append(baseAccessInfos, dto.BaseAccessInfo{
						ID:    base.ID,
						Title: base.Title,
					})
				}
			}
		}
		// For full_access and admin: bases array remains empty to indicate full workspace access

		response.Workspaces = append(response.Workspaces, dto.WorkspaceAccessInfo{
			ID:          ws.ID,
			Title:       ws.Title,
			AccessLevel: accessLevel,
			Bases:       baseAccessInfos,
		})
	}

	return response, nil
}
