package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"godbgrest/pkg"
	"serenibase/internal/config"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"strings"
	"time"

	app_errors "serenibase/internal/app-errors"
	authProviderInterface "serenibase/internal/providers/auth"
	emailProvider "serenibase/internal/providers/email"
	"serenibase/internal/providers/logger"
	otpProvider "serenibase/internal/providers/otp"
	"serenibase/internal/utils/helpers"

	appConstant "serenibase/internal/constant"

	appConfig "serenibase/internal/config"

	"github.com/google/uuid"
)

type authManagementService struct {
	userDefaultPassword appConfig.TemporaryAddedUserPasswordConfig
	repo                *pkg.DatabaseService

	userManagementService      interfaces.UserManagementService
	workspaceManagementService interfaces.WorkspaceManagementService
	userResetTokenService      interfaces.UserResetTokenService
	rbacManagementService      interfaces.RBACManagementService

	otpProviderService   otpProvider.OtpService
	emailTemplateService emailProvider.EmailTemplateService
	emailProviderService emailProvider.EmailService
	authProviderService  authProviderInterface.AuthProvider
}

func NewAuthManagementService(
	userDefaultPassword appConfig.TemporaryAddedUserPasswordConfig,
	repo *pkg.DatabaseService,
	userManagementService interfaces.UserManagementService,
	workspaceManagementService interfaces.WorkspaceManagementService,
	userResetTokenService interfaces.UserResetTokenService,
	rbacManagementService interfaces.RBACManagementService,
	otpProviderService otpProvider.OtpService,
	emailTemplateService emailProvider.EmailTemplateService,
	emailProviderService emailProvider.EmailService,
	authProviderService authProviderInterface.AuthProvider,
) interfaces.AuthManagementService {
	return &authManagementService{
		userDefaultPassword:        userDefaultPassword,
		repo:                       repo,
		userManagementService:      userManagementService,
		workspaceManagementService: workspaceManagementService,
		userResetTokenService:      userResetTokenService,
		rbacManagementService:      rbacManagementService,
		otpProviderService:         otpProviderService,
		emailTemplateService:       emailTemplateService,
		emailProviderService:       emailProviderService,
		authProviderService:        authProviderService,
	}
}

func (s *authManagementService) sendOtpViaEmail(email string) {
	otp := s.otpProviderService.Generate(email)
	emailData := s.emailTemplateService.EmailVerificationOTPBody(otp)
	s.emailProviderService.Enqueue(emailProvider.EmailJob{To: email, Subject: emailData.Subject, Body: emailData.Body})
}

func (a *authManagementService) RegisterOwner(ctx context.Context, req dto.RegisterRequest) (dto.LoginResponse, error) {
	// 1. Check if user already exists
	if existingUser, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, req.Email); err == nil {
		fmt.Println("User already exists with ID:", existingUser.ID)
		// If user exists, we might want to check if they are already an owner/verified.
		// For now, return error or handle gracefully. Script handled it by printing.
		// But here we return error to let caller decide.
		return dto.LoginResponse{}, app_errors.UserAlreadyExists
	} else if err != app_errors.UserNotFound {
		return dto.LoginResponse{}, err
	}

	// 2. Hash password
	hashed, err := helpers.HashPassword(req.Password)
	if err != nil {
		return dto.LoginResponse{}, app_errors.ErrHashed
	}
	req.Password = hashed

	// 3. Create user in master schema
	insertedUser, err := a.userManagementService.CreateUser(ctx, appConstant.MasterDatabase, req)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	// 4. Verify Email (Skip OTP for owner) && Initialize
	userData, err := a.initializeOwner(ctx, insertedUser.ID.String(), insertedUser)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	createDefaultWorkspace := dto.CreateWorkspaceRequest{
		Title:       "Default Workspace",
		Description: helpers.StringPtr(""),
		CreatedBy:   insertedUser.ID.String(),
	}

	_, err = a.workspaceManagementService.Create(ctx, createDefaultWorkspace, appConstant.MasterDatabase, insertedUser.ID.String())
	if err != nil {
		return dto.LoginResponse{}, err
	}

	roleData, err := a.rbacManagementService.GetRoleByName(ctx, appConstant.MasterDatabase, appConstant.RBACRoleNames.Owner)
	if err != nil {
		fmt.Println("err: ------- ", err)
		return dto.LoginResponse{}, err
	}

	fmt.Println("Assigning role", roleData.Name, "to user", insertedUser.Email)

	accessMemberReq := dto.AccessMemberDTO{
		UserID:     insertedUser.ID.String(),
		ScopeType:  appConstant.ScopeLevels.System,
		ScopeID:    nil,
		RoleID:     roleData.ID.String(),
		AssignedBy: helpers.StringPtr(insertedUser.ID.String()),
	}

	_, err = a.rbacManagementService.AssignRoleToUser(ctx, appConstant.MasterDatabase, accessMemberReq)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	loginResponse := dto.LoginResponse{
		User: &userData,
	}

	return loginResponse, nil
}

func (a *authManagementService) generateToken(ctx context.Context, user tenant.User) (dto.TokenResponse, error) {
	// No more calling authProvider.AddUser (Keycloak sync)

	tokens, err := a.authProviderService.GenerateToken(ctx, user)
	if err != nil {
		return dto.TokenResponse{}, err
	}

	return dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (a *authManagementService) extractValuesFromToken(tokenStr string, claimKeys []string) (map[string]interface{}, error) {
	if tokenStr == "" {
		return nil, app_errors.TokenAuthorizationHeaderRequired
	}
	// If tokenStr uses "Bearer ..." prefix, remove it
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	parsedClaims := map[string]interface{}{}
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, app_errors.TokenInvalid
	}
	payload, err := a.jwtDecodeSegment(parts[1])
	if err != nil {
		return nil, app_errors.AuthProviderTokenDecodeFailed
	}
	if err := json.Unmarshal(payload, &parsedClaims); err != nil {
		return nil, app_errors.AuthProviderTokenDecodeFailed
	}

	result := make(map[string]interface{})
	for _, key := range claimKeys {
		if val, ok := parsedClaims[key]; ok {
			result[key] = val
		}
	}
	return result, nil
}

func (a *authManagementService) jwtDecodeSegment(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}
	return base64.URLEncoding.DecodeString(seg)
}

func (a *authManagementService) Login(ctx context.Context, email string, password string) (dto.LoginResponse, error) {
	// Check user existence
	masterUser, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, email)
	if err != nil {
		if err == app_errors.UserNotFound {
			return dto.LoginResponse{}, app_errors.InvalidCredentials
		}
		return dto.LoginResponse{}, err
	}

	// Verify Password
	if !helpers.CheckPasswordHash(password, masterUser.Password) {
		return dto.LoginResponse{}, app_errors.InvalidCredentials
	}

	// Generate Token
	tokens, err := a.authProviderService.GenerateToken(ctx, masterUser)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	// If email not verified:
	if !masterUser.EmailVerified {
		return dto.LoginResponse{
			User: &dto.UserResponse{
				ID: masterUser.ID,
			},
			Token: &dto.TokenResponse{
				RefreshToken: tokens.RefreshToken,
			},
		}, nil
	}

	var userResponse dto.UserResponse
	if err := helpers.StructToStruct(masterUser, &userResponse); err != nil {
		return dto.LoginResponse{}, app_errors.ErrMapToStruct
	}

	tokenData := dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}

	loginResponse := dto.LoginResponse{
		User:  &userResponse,
		Token: &tokenData,
	}

	return loginResponse, nil
}

func (a *authManagementService) initializeOwner(ctx context.Context, userId string, user tenant.User) (dto.UserResponse, error) {

	var userResp dto.UserResponse
	// Basic mapping
	userResp.ID = user.ID
	userResp.Email = user.Email
	userResp.FirstName = user.FirstName
	userResp.LastName = user.LastName
	// ... (other fields as needed)

	// Update User
	updateData := map[string]interface{}{
		"status":         "active",
		"email_verified": true,
		"last_login_at":  time.Now(),
	}

	updatedUser, err := a.userManagementService.UpdateUser(ctx, appConstant.MasterDatabase, userId, updateData)
	fmt.Println("Updated user after initialization:", updatedUser)
	if err != nil {
		return dto.UserResponse{}, err
	}

	var userData dto.UserResponse
	if err := helpers.StructToStruct(updatedUser, &userData); err != nil {
		return dto.UserResponse{}, app_errors.ErrStructToStruct
	}

	return userData, nil
}

func (a *authManagementService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) (dto.LoginResponse, error) {
	// Original logic: RefreshToken(req.Token) -> Extract claims -> Check OTP -> addUserWithTenant -> SetEmailVerified

	// We need to validate the token (which is likely a RefreshToken from Register response)
	tokenData, err := a.authProviderService.RefreshToken(ctx, req.Token)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	// Extract info from token
	// Assuming RefreshToken has UserID in claims
	claims, err := a.authProviderService.ValidateToken(ctx, tokenData.RefreshToken)
	// ValidateToken might fail if it expects access token format?
	// But our local JWT ValidateToken works for any valid JWT signed by us.
	if err != nil {
		return dto.LoginResponse{}, err
	}

	userId := claims.UserId

	user, err := a.userManagementService.GetUserByID(ctx, appConstant.MasterDatabase, userId)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	ok := a.otpProviderService.Verify(user.Email, req.OTP)
	if !ok {
		return dto.LoginResponse{}, app_errors.InvalidOTP
	}

	userData, err := a.initializeOwner(ctx, user.ID.String(), user)
	if err != nil {
		fmt.Println(err)
		return dto.LoginResponse{}, err
	}

	// Generate FRESH tokens
	user.Roles = "Admin" // Default for new owner? Or regular user? verifyEmail usually implies general user.

	// We also need to update user.EmailVerified = true locally for token generation
	user.EmailVerified = true

	tokens, err := a.authProviderService.GenerateToken(ctx, user)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	// Build response with fresh tokens
	tokenResponse := dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}

	loginResponse := dto.LoginResponse{
		User:  &userData,
		Token: &tokenResponse,
	}

	return loginResponse, nil
}

func (a *authManagementService) ResendOTP(ctx context.Context, req dto.ResendOTPRequest) error {
	// Similar logic to VerifyEmail to get user
	tokenData, err := a.authProviderService.RefreshToken(ctx, req.Token)
	if err != nil {
		return err
	}
	claims, err := a.authProviderService.ValidateToken(ctx, tokenData.RefreshToken)
	if err != nil {
		return err
	}

	userId := claims.UserId

	user, err := a.userManagementService.GetUserByID(ctx, appConstant.MasterDatabase, userId)
	if err != nil {
		return err
	}

	if user.EmailVerified {
		return app_errors.EmailAlreadyVerified
	}

	a.sendOtpViaEmail(user.Email)

	return nil
}

func (a *authManagementService) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.TokenResponse, error) {
	tokens, err := a.authProviderService.RefreshToken(ctx, req.RefeshToken)
	if err != nil {
		return dto.TokenResponse{}, err
	}

	return dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (a *authManagementService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error {
	// Local lookup instead of Keycloak

	// Original used: CheckUserExistsByEmailAndReturnUser(ctx, req.Email) from Provider.
	// Now we use `GetUserByEmail`.

	user, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, req.Email)
	if err != nil {
		return app_errors.UserNotFound
	}

	if !user.EmailVerified {
		return app_errors.InvalidCredentials
	}

	// Removing strict "pending" check if verified, as verified users are "active" usually.
	if user.Status == "pending" && !user.EmailVerified {
		return app_errors.InvalidCredentials
	}

	tokenAttrs := map[string]interface{}{
		"user_id": user.ID.String(),
	}

	token, err := helpers.GenerateCustomJWT(tokenAttrs, user.ID.String(), 3600) // 1 hour expiry
	if err != nil {
		return err
	}

	dataToInsert := dto.UserResetTokenInsertion{
		ID:     uuid.NewString(),
		UserID: user.ID.String(),
		Token:  token,
		Expiry: time.Now().Add(1 * time.Hour),
	}
	_, err = a.userResetTokenService.CreateUserResetToken(ctx, dataToInsert)
	// data unused was triggering lint?
	if err != nil {
		return app_errors.UserNotFound
	}

	resetURLTemplate := config.AppConfig.Auth.ResetPasswordURL
	if resetURLTemplate == "" {
		resetURLTemplate = "http://localhost:5050/reset-password?token=%s"
	}
	resetLink := fmt.Sprintf(resetURLTemplate, token) // Use token directly
	emailData := a.emailTemplateService.PasswordResetBody(resetLink)

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})

	return nil
}

func (a *authManagementService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	userResetToken, err := a.userResetTokenService.GetUserResetToken(ctx, req.Token)
	if err != nil {
		return app_errors.TokenInvalid
	}

	if time.Now().After(userResetToken.Expiry) {
		return app_errors.TokenExpired
	}

	// Assuming Check claims helper also verifies signature?
	// The token is stored, so it was valid when created.

	userId := userResetToken.UserID.String()

	hashedPassword, err := helpers.HashPassword(req.NewPassword)
	if err != nil {
		return app_errors.ErrHashed
	}

	// Extract claims if needed validation
	_, err = helpers.DecodeJWT(userResetToken.Token)
	if err != nil {
		logger.Get().Error().
			Err(err).
			Str("user_id", userId).
			Msg("Failed to decode JWT from reset token")
		return app_errors.TokenInvalid
	}

	// Update password in master schema
	_, err = a.userManagementService.UpdateUser(ctx, appConstant.MasterDatabase, userId, map[string]interface{}{
		"password":            hashedPassword,
		"status":              "active",
		"email_verified":      true,
		"password_changed_at": time.Now(),
	})
	if err != nil {
		logger.Get().Error().
			Err(err).
			Str("user_id", userId).
			Msg("Failed to update password in master schema")
		return err
	}

	// Clean tokens
	err = a.userResetTokenService.DeleteTokensByUserId(ctx, userId)
	return err
}

func (a *authManagementService) HandleKeycloakCallback(ctx context.Context, code string) (dto.LoginResponse, error) {
	return dto.LoginResponse{}, fmt.Errorf("social login removed")
}

func (a *authManagementService) GetAuthProviderUrl(provider string) string {
	return ""
}

func (a *authManagementService) Logout(ctx context.Context, refreshToken string) error {
	// Stateless logout, maybe can blacklist if needed
	return nil
}

func (a *authManagementService) AddUser(ctx context.Context, schema string, userData dto.AddUserRequest, reqBy string) (tenant.User, error) {
	// Admin adding user
	roles := appConstant.RBACRoleNames.NoAccess
	if userData.IsCoOwner {
		roles = appConstant.RBACRoleNames.CoOwner
	}

	userCreationReq := dto.RegisterRequest{
		ID:            uuid.New(),
		Email:         userData.Email,
		FirstName:     userData.FirstName,
		LastName:      userData.LastName,
		Password:      a.userDefaultPassword.Value,
		AuthProvider:  "email",
		EmailVerified: false,
		Status:        "pending",
		Roles:         roles,
	}

	user, err := a.userManagementService.CreateUser(ctx, schema, userCreationReq)
	if err != nil {
		return tenant.User{}, err
	}

	// // Handle profile picture file upload
	// if userData.ProfilePic != nil {
	// 	_, err := a.userManagementService.AddAvatar(ctx, schema, user.ID.String(), userData.ProfilePic)
	// 	if err != nil {
	// 		return tenant.User{}, err
	// 	}
	// }

	// Send invitation
	tokenAttrs := map[string]interface{}{
		"user_id": user.ID.String(),
	}

	token, err := helpers.GenerateCustomJWT(tokenAttrs, user.ID.String(), 3600)
	if err != nil {
		return tenant.User{}, err
	}

	dataToInsert := dto.UserResetTokenInsertion{
		ID:     uuid.NewString(),
		UserID: user.ID.String(),
		Token:  token,
		Expiry: time.Now().Add(1 * time.Hour),
	}

	data, err := a.userResetTokenService.CreateUserResetToken(ctx, dataToInsert)
	if err != nil {
		return tenant.User{}, err
	}

	// handle membership invitations if any
	if roles == appConstant.RBACRoleNames.CoOwner || len(userData.Membership) == 0 {
		roleData, err := a.rbacManagementService.GetRoleByName(ctx, appConstant.MasterDatabase, roles)
		if err != nil {
			return tenant.User{}, err
		}

		accessMemberReq := dto.AccessMemberDTO{
			UserID:     user.ID.String(),
			ScopeType:  appConstant.ScopeLevels.System,
			ScopeID:    nil,
			RoleID:     roleData.ID.String(),
			AssignedBy: helpers.StringPtr(reqBy),
		}

		_, err = a.rbacManagementService.AssignRoleToUser(ctx, appConstant.MasterDatabase, accessMemberReq)
		if err != nil {
			return tenant.User{}, err
		}
	} else {
		fmt.Println("-------------------")
		_, err = a.rbacManagementService.ProcessUserMemberships(ctx, schema, user.ID.String(), user.ID.String(), userData.Membership)
		if err != nil {
			return tenant.User{}, err
		}

	}

	resetURLTemplate := config.AppConfig.Auth.ResetPasswordURL
	if resetURLTemplate == "" {
		resetURLTemplate = "http://localhost:5050/reset-password?token=%s"
	}
	resetLink := fmt.Sprintf(resetURLTemplate, data.Token)

	// Assuming PlatformInvitationBody expects (FirstName, OrganizationName, Link).
	// Since tenantData.Name is gone, we can use a hardcoded name or config name.
	// For now, using empty string or "SereniBase" placeholder until config is better.
	emailData := a.emailTemplateService.PlatformInvitationBody(user.FirstName, "SereniBase", resetLink)

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})

	return user, nil
}

func (a *authManagementService) RemoveUser(ctx context.Context, schema string, userID string) error {
	// Local delete only
	updateData := map[string]interface{}{
		"is_deleted": true,
		"deleted_at": time.Now(),
	}
	_, err := a.userManagementService.UpdateUser(ctx, schema, userID, updateData)
	return err
}

func (a *authManagementService) ActivateUser(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	updateFields := map[string]interface{}{
		"status":             "active",
		"last_modified_time": time.Now(),
	}

	updatedUser, err := a.userManagementService.UpdateUser(ctx, schema, userID, updateFields)
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

func (a *authManagementService) DeactivateUser(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	updateFields := map[string]interface{}{
		"status":             "deactivated",
		"last_modified_time": time.Now(),
	}

	updatedUser, err := a.userManagementService.UpdateUser(ctx, schema, userID, updateFields)
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

func (a *authManagementService) GetUsers(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	return a.userManagementService.GetUsersWithRole(ctx, schema)
}

func (a *authManagementService) AssignUserToWorkspace(ctx context.Context, schema string, req dto.CreateMemberRequest, reqBy string) error {
	_, err := a.rbacManagementService.ProcessUserMemberships(ctx, schema, req.UserID, reqBy, req.Membership)
	if err != nil {
		return err
	}
	return nil
}

func (a *authManagementService) RemoveUserFromWorkspace(ctx context.Context, schema string, req dto.RemoveMemberRequest, reqBy string) error {
	// err := a.workspaceManagementService.RemoveUserFromWorkspace(ctx, schema, req.WorkspaceID, req.UserID)
	// if err != nil {
	// 	return err
	// }

	user, userErr := a.userManagementService.GetUserByID(ctx, schema, req.UserID)
	if userErr != nil {
		return nil
	}

	workspace, workspaceErr := a.workspaceManagementService.GetByID(ctx, schema, req.WorkspaceID)
	workspaceLabel := req.WorkspaceID
	if workspaceErr == nil {
		workspaceLabel = workspace.Title
	}

	emailData := a.emailTemplateService.RemovedFromWorkspaceBody(workspaceLabel)
	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})

	return nil
}

func (a *authManagementService) InviteMemberToWorkspace(ctx context.Context, schema string, req dto.CreateMemberRequest, reqBy string) error {
	// workspace, workspaceErr := a.workspaceManagementService.GetByID(ctx, schema, req.WorkspaceID)
	// if workspaceErr != nil {
	// 	return workspaceErr
	// }

	// user, userErr := a.userManagementService.GetUserByID(ctx, schema, req.UserID)
	// if userErr != nil {
	// 	return userErr
	// }

	// emailData := a.emailTemplateService.InvitedToWorkspaceBody(workspace.Title, req.AccessLevel)
	// a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})

	return nil
}

func (a *authManagementService) GetWorkspaceMembers(ctx context.Context, schema string, workspaceID string) ([]dto.WorkspaceMemberResponse, error) {
	members, err := a.workspaceManagementService.GetWorkspaceMembers(ctx, schema, workspaceID)
	if err != nil {
		if err == app_errors.WorkspaceMemberNotFound {
			return []dto.WorkspaceMemberResponse{}, nil
		}
		return nil, err
	}

	userIDs := make([]string, 0, len(members))
	userAccess := map[string]string{}
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
		userAccess[m.UserID] = m.AccessLevel
	}

	users, err := a.userManagementService.GetBulkUsers(ctx, schema, userIDs)
	if err != nil {
		return nil, err
	}

	var res []dto.WorkspaceMemberResponse
	for _, user := range users {
		var memberResp dto.WorkspaceMemberResponse
		err := helpers.StructToStruct(user, &memberResp)
		if err != nil {
			return nil, err
		}
		memberResp.AccessLevel = userAccess[user.ID.String()]
		res = append(res, memberResp)
	}

	return res, nil
}

func (a *authManagementService) GetBaseMembers(ctx context.Context, schema string, baseID string) ([]dto.WorkspaceMemberResponse, error) {
	members, err := a.workspaceManagementService.GetWorkspaceBaseMembers(ctx, schema, baseID)
	if err != nil {
		if err == app_errors.WorkspaceMemberNotFound {
			return []dto.WorkspaceMemberResponse{}, nil
		}
		return nil, err
	}

	userIDs := make([]string, 0, len(members))
	userAccess := map[string]string{}
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
		userAccess[m.UserID] = m.AccessLevel
	}

	users, err := a.userManagementService.GetBulkUsers(ctx, schema, userIDs)
	if err != nil {
		return nil, err
	}

	var res []dto.WorkspaceMemberResponse
	for _, user := range users {
		var memberResp dto.WorkspaceMemberResponse
		err := helpers.StructToStruct(user, &memberResp)
		if err != nil {
			return nil, err
		}
		memberResp.AccessLevel = userAccess[user.ID.String()]
		res = append(res, memberResp)
	}

	return res, nil
}

func (a *authManagementService) DeleteUserCompletely(ctx context.Context, schema string, userID string) error {
	// Local delete only
	deleteUserErr := a.userManagementService.DeleteUserCompletely(ctx, schema, userID)
	if deleteUserErr != nil {
		return deleteUserErr
	}

	removeMappingErr := a.workspaceManagementService.DeleteUserMappings(ctx, schema, userID)
	if removeMappingErr != nil {
		if removeMappingErr == app_errors.WorkspaceMemberNotFound {
			return nil
		}
		return removeMappingErr
	}

	return nil
}


func (a *authManagementService) UpdatePassword(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) error {
	user, err := a.userManagementService.UpdatePassword(ctx, schema, userID, updateData)
	if err != nil {
		return err
	}
	// No external sync
	_ = user
	return nil
}
