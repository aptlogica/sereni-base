package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	"mime/multipart"
	"path/filepath"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/models/tenant"
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
	tenantManagementService    interfaces.TenantManagementService
	subscriptionPlanService    interfaces.SubscriptionPlanService
	assetManagementService     interfaces.AssetManagementService
	userResetTokenService      interfaces.UserResetTokenService
	userRoleService            interfaces.UserRoleService
	workspaceManagementService interfaces.WorkspaceManagementService
	authProvider               authProviderInterface.AuthProvider
}

func NewUserManagementService(
	repo *pkg.DatabaseService,
	userService interfaces.UserService,
	tenantManagementService interfaces.TenantManagementService,
	subscriptionPlanService interfaces.SubscriptionPlanService,
	assetManagementService interfaces.AssetManagementService,
	userResetTokenService interfaces.UserResetTokenService,
	userRoleService interfaces.UserRoleService,
	workspaceManagementService interfaces.WorkspaceManagementService,
	authProvider authProviderInterface.AuthProvider,
) interfaces.UserManagementService {
	return &userManagementService{
		repo:                       repo,
		userService:                userService,
		tenantManagementService:    tenantManagementService,
		subscriptionPlanService:    subscriptionPlanService,
		assetManagementService:     assetManagementService,
		userResetTokenService:      userResetTokenService,
		userRoleService:            userRoleService,
		workspaceManagementService: workspaceManagementService,
		authProvider:               authProvider,
	}
}

func (s *userManagementService) GetUserProfileByID(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	user, err := s.userService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}
	fmt.Println(user)

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
		fmt.Println(err)
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

func (s *userManagementService) UpdatePassword(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) (master.User, error) {
	if updateData.OldPassword == updateData.NewPassword {
		return master.User{}, app_errors.NewPasswordInvalid
	}

	// Fetch user by ID
	user, err := s.userService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return master.User{}, err
	}

	// Check if old password matches
	if !helpers.CheckPasswordHash(updateData.OldPassword, user.Password) {
		return master.User{}, app_errors.InvalidOldPassword
	}

	// Hash the new password
	hashedPassword, err := helpers.HashPassword(updateData.NewPassword)
	if err != nil {
		return master.User{}, app_errors.ErrHashed
	}

	updateFields := map[string]interface{}{
		"password":            hashedPassword,
		"password_changed_at": time.Now(),
		"last_modified_time":  time.Now(),
	}

	user, err = s.userService.UpdateUser(ctx, schema, userID, updateFields)
	if err != nil {
		return master.User{}, err
	}

	return user, nil
}

func (s *userManagementService) AddAvatar(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error) {
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
		fmt.Println("assetManagementService upload: ", err)
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

func (s *userManagementService) GetUserByEmail(ctx context.Context, schema string, email string) (master.User, error) {
	return s.userService.GetUserByEmail(ctx, schema, email)
}

func (s *userManagementService) CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (master.User, error) {
	return s.userService.CreateUser(ctx, schema, req)
}

func (s *userManagementService) UpdateUser(ctx context.Context, schema string, id string, updateData map[string]interface{}) (master.User, error) {
	return s.userService.UpdateUser(ctx, schema, id, updateData)
}

func (s *userManagementService) GetUserByID(ctx context.Context, schema string, id string) (master.User, error) {
	return s.userService.GetUserByID(ctx, schema, id)
}

func (s *userManagementService) AddUserRole(ctx context.Context, schema string, userID, roleID uuid.UUID) error {
	userRoleInsertionReq := dto.UserRoleInsertion{
		ID:     uuid.New(),
		UserID: userID,
		RoleID: roleID,
	}

	_, err := s.userRoleService.CreateUserRole(ctx, schema, userRoleInsertionReq)
	if err != nil {
		fmt.Println("CreateUserRole err ----> ", err, schema)
		return app_errors.TableNotFound
	}

	return nil
}

func (t *userManagementService) AddUserToTenant(ctx context.Context, schema string, userData dto.AddUserRequest, roleId uuid.UUID, userPassword string) (master.User, master.Tenant, error) {
	userCreationReq := dto.RegisterRequest{
		ID:            uuid.New(),
		Email:         userData.Email,
		FirstName:     userData.FirstName,
		LastName:      userData.LastName,
		Password:      userPassword,
		AuthProvider:  "email",
		EmailVerified: false,
		Status:        "pending",
	}

	createdUser, err := t.CreateUser(ctx, schema, userCreationReq)
	if err != nil {
		return master.User{}, master.Tenant{}, err
	}

	tenentData, err := t.tenantManagementService.GetTenantBySchema(ctx, schema)
	if err != nil {
		fmt.Println("GetTenantBySchema err ----> ", err)
		return master.User{}, master.Tenant{}, app_errors.TableNotFound
	}

	return createdUser, tenentData, nil
}

func (s *userManagementService) GetAllUsers(ctx context.Context, schema string) ([]master.User, error) {
	return s.userService.GetAllUsers(ctx, schema)
}

func (s *userManagementService) GetWorkspaces(ctx context.Context, schema string, userID string, roles string) ([]dto.UserWorkspaceResponse, error) {
	fmt.Println("roles: ", roles)
	if roles == appConstant.RoleNames.Admin {
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
			wsResp.AccessLevel = appConstant.RoleNames.Admin
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

func (s *userManagementService) GetBulkUsers(ctx context.Context, schema string, ids []string) ([]master.User, error) {
	return s.userService.GetBulkUsers(ctx, schema, ids)
}

func (s *userManagementService) GetUsersWithRole(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	functionName := "get_users_with_role"
	schemaFunctionName := fmt.Sprintf("%s.%s", appConstant.MasterDatabase, functionName)

	args := map[string]interface{}{
		"p_schema_name": schema,
	}

	records, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var result []dto.UserWithRole
	for _, record := range records {
		if rec, ok := record[functionName].(map[string]interface{}); ok {
			var user dto.UserWithRole
			if err := helpers.MapToStruct(rec, &user); err == nil {
				result = append(result, user)
			} else {
				fmt.Println("MapToStruct error:", err)
			}
		}
	}
	fmt.Println("result: ", result)
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
		if accessLevel == appConstant.AccessNames.LimitedAccess && membership != nil {
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

