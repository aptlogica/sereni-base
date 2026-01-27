package services

import (
	"context"
	"fmt"
	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
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

func (a *authManagementService) Login(ctx context.Context, email string, password string) (dto.LoginResponse, error) {
	// Check user existence
	masterUser, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, email)
	if err != nil {
		if err == app_errors.UserNotFound {
			return dto.LoginResponse{}, app_errors.InvalidCredentials
		}
		return dto.LoginResponse{}, err
	}

	if masterUser.Status != "active" {
		return dto.LoginResponse{}, app_errors.UserNotActive
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

func (a *authManagementService) ValidateToken(ctx context.Context, token string) (dto.TokenValidationResponse, error) {
	claims, err := a.authProviderService.ValidateToken(ctx, token)
	if err != nil {
		return dto.TokenValidationResponse{Valid: false}, err
	}

	return dto.TokenValidationResponse{
		Valid:  true,
		UserID: claims.UserId,
		Roles:  claims.Roles,
	}, nil
}

func (a *authManagementService) VerifyToken(ctx context.Context, token string) (dto.TokenValidationResponse, error) {
	return a.ValidateToken(ctx, token)
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

	resetURLTemplate := appConfig.AppConfig.Auth.ResetPasswordURL
	if resetURLTemplate == "" {
		resetURLTemplate = "http://localhost:5050/reset-password?token=%s"
	}
	resetLink := fmt.Sprintf(resetURLTemplate, token) // Use token directly
	emailData := a.emailTemplateService.PasswordResetBody(resetLink)

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})

	return nil
}

func (a *authManagementService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	userResetToken, err := a.fetchValidResetToken(ctx, req.Token)
	if err != nil {
		return err
	}

	userId := userResetToken.UserID.String()

	hashedPassword, err := a.hashNewPassword(req.NewPassword)
	if err != nil {
		return err
	}

	if err := a.validateResetTokenClaims(userResetToken.Token, userId); err != nil {
		return err
	}

	if err := a.updateUserPassword(ctx, userId, hashedPassword); err != nil {
		return err
	}

	return a.cleanUserResetTokens(ctx, userId)
}

func (a *authManagementService) fetchValidResetToken(ctx context.Context, token string) (tenant.UserResetToken, error) {
	userResetToken, err := a.userResetTokenService.GetUserResetToken(ctx, token)
	if err != nil {
		return tenant.UserResetToken{}, app_errors.TokenInvalid
	}

	if time.Now().After(userResetToken.Expiry) {
		return tenant.UserResetToken{}, app_errors.TokenExpired
	}

	return userResetToken, nil
}

func (a *authManagementService) hashNewPassword(newPassword string) (string, error) {
	hashedPassword, err := helpers.HashPassword(newPassword)
	if err != nil {
		return "", app_errors.ErrHashed
	}

	return hashedPassword, nil
}

func (a *authManagementService) validateResetTokenClaims(token string, userId string) error {
	if _, err := helpers.DecodeJWT(token); err != nil {
		logger.Get().Error().
			Err(err).
			Str("user_id", userId).
			Msg("Failed to decode JWT from reset token")
		return app_errors.TokenInvalid
	}

	return nil
}

func (a *authManagementService) updateUserPassword(ctx context.Context, userId string, hashedPassword string) error {
	_, err := a.userManagementService.UpdateUser(ctx, appConstant.MasterDatabase, userId, map[string]interface{}{
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

	return nil
}

func (a *authManagementService) cleanUserResetTokens(ctx context.Context, userId string) error {
	return a.userResetTokenService.DeleteTokensByUserId(ctx, userId)
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

	// Check if email already exists
	_, err := a.userManagementService.GetUserByEmail(ctx, schema, userData.Email)
	if err == nil {
		// User exists with this email
		return tenant.User{}, app_errors.UserAlreadyExists
	} else if err != app_errors.UserNotFound {
		// Some other error occurred
		return tenant.User{}, err
	}
	// If err == app_errors.UserNotFound, the email is available, continue

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

	// Handle profile picture file upload
	if userData.ProfilePic != nil {
		_, err := a.userManagementService.AddAvatar(ctx, schema, user.ID.String(), userData.ProfilePic)
		if err != nil {
			return tenant.User{}, err
		}
	}

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
	if roles == appConstant.RBACRoleNames.CoOwner {
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
		_, err = a.rbacManagementService.ProcessUserMemberships(ctx, schema, user.ID.String(), user.ID.String(), userData.Membership)
		if err != nil {
			return tenant.User{}, err
		}

	}

	resetURLTemplate := appConfig.AppConfig.Auth.ResetPasswordURL
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

// EditUser updates user details with support for profile, avatar, membership, and co-owner status changes
func (a *authManagementService) EditUser(ctx context.Context, schema string, userData dto.EditUserRequest, reqBy string) (dto.UserResponse, error) {
	// Get existing user
	_, err := a.userManagementService.GetUserByID(ctx, schema, userData.UserID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	if err := a.updateUserNames(ctx, schema, userData); err != nil {
		return dto.UserResponse{}, err
	}

	if err := a.updateUserAvatar(ctx, schema, userData); err != nil {
		return dto.UserResponse{}, err
	}

	if err := a.processCoOwnerChanges(ctx, schema, userData, reqBy); err != nil {
		return dto.UserResponse{}, err
	}

	if err := a.updateMemberships(ctx, schema, userData, reqBy); err != nil {
		return dto.UserResponse{}, err
	}

	return a.buildUpdatedUserResponse(ctx, schema, userData.UserID)
}

func (a *authManagementService) updateUserNames(ctx context.Context, schema string, userData dto.EditUserRequest) error {
	if userData.FirstName == nil && userData.LastName == nil {
		return nil
	}

	updateReq := dto.UpdateUserProfileRequest{UpdatedAt: time.Now()}
	if userData.FirstName != nil {
		updateReq.FirstName = userData.FirstName
	}
	if userData.LastName != nil {
		updateReq.LastName = userData.LastName
	}

	_, err := a.userManagementService.UpdateUserProfile(ctx, schema, userData.UserID, updateReq)
	return err
}

func (a *authManagementService) updateUserAvatar(ctx context.Context, schema string, userData dto.EditUserRequest) error {
	if userData.ProfilePic == nil {
		return nil
	}

	if _, err := a.userManagementService.RemoveAvatar(ctx, schema, userData.UserID); err != nil && err != app_errors.AssetNotFound {
		return err
	}

	_, err := a.userManagementService.AddAvatar(ctx, schema, userData.UserID, userData.ProfilePic)
	return err
}

func (a *authManagementService) processCoOwnerChanges(ctx context.Context, schema string, userData dto.EditUserRequest, reqBy string) error {
	if userData.IsCoOwner == nil {
		return nil
	}

	currentAccessMembers, _ := a.rbacManagementService.GetUserAccessMembers(ctx, schema, userData.UserID)
	isCurrentCoOwner, currentCoOwnerAccessID := a.detectCoOwner(ctx, schema, currentAccessMembers)

	if *userData.IsCoOwner && !isCurrentCoOwner {
		if err := a.promoteToCoOwner(ctx, schema, reqBy, userData.UserID, currentAccessMembers); err != nil {
			return err
		}
	}

	if !*userData.IsCoOwner && isCurrentCoOwner {
		if err := a.demoteFromCoOwner(ctx, schema, reqBy, userData, currentCoOwnerAccessID); err != nil {
			return err
		}
	}

	return nil
}

func (a *authManagementService) detectCoOwner(ctx context.Context, schema string, accessMembers []dto.AccessMemberDTO) (bool, string) {
	for _, member := range accessMembers {
		if member.RoleID == "" {
			continue
		}

		roleUUID, parseErr := uuid.Parse(member.RoleID)
		if parseErr != nil {
			continue
		}

		role, roleErr := a.rbacManagementService.GetRoleByID(ctx, schema, roleUUID)
		if roleErr == nil && role.Name == appConstant.RBACRoleNames.CoOwner {
			return true, member.ID.String()
		}
	}
	return false, ""
}

func (a *authManagementService) promoteToCoOwner(
	ctx context.Context,
	schema string,
	reqBy string,
	userID string,
	currentAccessMembers []dto.AccessMemberDTO,
) error {
	for _, member := range currentAccessMembers {
		_ = a.RemoveAccessMemberByID(ctx, schema, member.ID.String(), reqBy)
	}

	roleData, err := a.rbacManagementService.GetRoleByName(ctx, appConstant.MasterDatabase, appConstant.RBACRoleNames.CoOwner)
	if err != nil {
		return err
	}

	accessMemberReq := dto.AccessMemberDTO{
		UserID:     userID,
		ScopeType:  appConstant.ScopeLevels.System,
		ScopeID:    nil,
		RoleID:     roleData.ID.String(),
		AssignedBy: helpers.StringPtr(reqBy),
	}

	_, err = a.rbacManagementService.AssignRoleToUser(ctx, appConstant.MasterDatabase, accessMemberReq)
	return err
}

func (a *authManagementService) demoteFromCoOwner(
	ctx context.Context,
	schema string,
	reqBy string,
	userData dto.EditUserRequest,
	coOwnerAccessID string,
) error {
	if coOwnerAccessID != "" {
		_ = a.RemoveAccessMemberByID(ctx, schema, coOwnerAccessID, reqBy)
	}

	if len(userData.Membership) == 0 {
		return nil
	}

	_, err := a.rbacManagementService.ProcessUserMemberships(ctx, schema, userData.UserID, reqBy, userData.Membership)
	return err
}

func (a *authManagementService) updateMemberships(ctx context.Context, schema string, userData dto.EditUserRequest, reqBy string) error {
	if len(userData.Membership) == 0 {
		return nil
	}

	_, err := a.rbacManagementService.ProcessUserMemberships(ctx, schema, userData.UserID, reqBy, userData.Membership)
	return err
}

func (a *authManagementService) buildUpdatedUserResponse(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	updatedUser, err := a.userManagementService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	var userResponse dto.UserResponse
	if err := helpers.StructToStruct(updatedUser, &userResponse); err != nil {
		return dto.UserResponse{}, app_errors.ErrStructToStruct
	}

	return userResponse, nil
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
	// Check if user has Owner role - owners cannot be deactivated
	accessMembers, err := a.rbacManagementService.GetUserAccessMembers(ctx, schema, userID)
	if err == nil {
		for _, member := range accessMembers {
			if member.RoleID != "" {
				roleUUID, parseErr := uuid.Parse(member.RoleID)
				if parseErr == nil {
					role, roleErr := a.rbacManagementService.GetRoleByID(ctx, schema, roleUUID)
					if roleErr == nil && role.Name == appConstant.RBACRoleNames.Owner {
						return dto.UserResponse{}, app_errors.OwnerCannotBeDeactivated
					}
				}
			}
		}
	}

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

func (a *authManagementService) GetActiveUsersForAssign(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
	return a.userManagementService.GetActiveUsersForAssign(ctx, schema)
}

func (a *authManagementService) AssignUserToWorkspace(ctx context.Context, schema string, req dto.CreateMemberRequest, reqBy string) error {
	_, err := a.rbacManagementService.ProcessUserMemberships(ctx, schema, req.UserID, reqBy, req.Membership)
	if err != nil {
		return err
	}
	return nil
}

func (a *authManagementService) RemoveUserFromWorkspace(ctx context.Context, schema string, workspaceID string, userID string, reqBy string) error {
	// Try to remove from access_members table (RBAC system)
	// Find all access_members records for this user-workspace combination
	accessMembersTableName := fmt.Sprintf("\"%s\".access_members", schema)
	params := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
			{
				Column:   "scope_type",
				Operator: "eq",
				Value:    "workspace",
			},
			{
				Column:   "scope_id",
				Operator: "eq",
				Value:    workspaceID,
			},
		},
	}

	accessRecords, err := a.repo.TableService.GetTableData(accessMembersTableName, params)
	if err != nil {
		// If RBAC doesn't exist, try legacy workspace_members table
		return a.workspaceManagementService.RemoveUserFromWorkspace(ctx, schema, workspaceID, userID)
	}

	if len(accessRecords) == 0 {
		// If no RBAC records, try legacy workspace_members table
		return a.workspaceManagementService.RemoveUserFromWorkspace(ctx, schema, workspaceID, userID)
	}

	// Delete all access_members records for this user-workspace combination
	for _, record := range accessRecords {
		var accessMemberID string
		// Handle both string and uuid.UUID types
		switch v := record["id"].(type) {
		case string:
			accessMemberID = v
		case uuid.UUID:
			accessMemberID = v.String()
		default:
			return fmt.Errorf("unexpected type for id field: %T", v)
		}

		deleteErr := a.repo.TableService.DeleteRecord(accessMembersTableName, accessMemberID)
		if deleteErr != nil {
			return deleteErr
		}
	}

	// Send email notification
	user, userErr := a.userManagementService.GetUserByID(ctx, schema, userID)
	if userErr != nil {
		return nil
	}

	workspace, workspaceErr := a.workspaceManagementService.GetByID(ctx, schema, workspaceID)
	workspaceLabel := workspaceID
	if workspaceErr == nil {
		workspaceLabel = workspace.Title
	}

	emailData := a.emailTemplateService.RemovedFromWorkspaceBody(workspaceLabel)
	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})

	return nil
}

func (a *authManagementService) RemoveUserFromBase(ctx context.Context, schema string, baseID string, userID string, reqBy string) error {
	// For base removal, we need to find the access_member record and delete it
	// Find the access_members record for this user-base combination
	accessMembersTableName := fmt.Sprintf("\"%s\".access_members", schema)
	params := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
			{
				Column:   "scope_type",
				Operator: "eq",
				Value:    "base",
			},
			{
				Column:   "scope_id",
				Operator: "eq",
				Value:    baseID,
			},
		},
	}

	accessRecords, err := a.repo.TableService.GetTableData(accessMembersTableName, params)
	if err != nil {
		return err
	}

	if len(accessRecords) == 0 {
		return app_errors.ErrRecordNotFound
	}

	// Delete the access_members record
	var accessMemberID string
	// Handle both string and uuid.UUID types
	switch v := accessRecords[0]["id"].(type) {
	case string:
		accessMemberID = v
	case uuid.UUID:
		accessMemberID = v.String()
	default:
		return fmt.Errorf("unexpected type for id field: %T", v)
	}

	deleteErr := a.repo.TableService.DeleteRecord(accessMembersTableName, accessMemberID)
	if deleteErr != nil {
		return deleteErr
	}

	// Send email notification
	user, userErr := a.userManagementService.GetUserByID(ctx, schema, userID)
	if userErr != nil {
		return nil
	}

	// Get the base title for the email
	baseTableName := fmt.Sprintf("\"%s\".bases", schema)
	baseParams := dbModels.QueryParams{
		Select: []string{"title"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    baseID,
			},
		},
	}

	baseRecords, baseErr := a.repo.TableService.GetTableData(baseTableName, baseParams)
	baseLabel := baseID
	if baseErr == nil && len(baseRecords) > 0 {
		if title, ok := baseRecords[0]["title"].(string); ok {
			baseLabel = title
		}
	}

	emailData := a.emailTemplateService.RemovedFromWorkspaceBody(baseLabel)
	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})

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
	// Check if user exists and has pending status
	user, err := a.userManagementService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return err
	}

	// Only allow deletion if user status is "pending"
	if user.Status != "pending" {
		return app_errors.OnlyPendingUsersCanBeDeleted
	}

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

// BulkAddMembers adds multiple members to a workspace with their memberships
func (a *authManagementService) BulkAddMembers(ctx context.Context, schema string, req dto.BulkAddMembersRequest, userID string) (dto.BulkAddMembersResponse, error) {
	result := dto.BulkAddMembersResponse{
		Success: []string{},
		Failed:  []dto.MemberAddFailure{},
		Total:   len(req.Members),
	}

	for _, member := range req.Members {
		createReq := dto.CreateMemberRequest{
			UserID:     member.UserID,
			Membership: member.Memberships,
		}

		err := a.AssignUserToWorkspace(ctx, schema, createReq, userID)
		if err != nil {
			result.Failed = append(result.Failed, dto.MemberAddFailure{
				UserID: member.UserID,
				Error:  fmt.Sprintf("failed to assign member: %v", err),
			})
		} else {
			result.Success = append(result.Success, member.UserID)
		}
	}

	return result, nil
}

// BulkAddBaseMembers adds multiple members to bases with their roles
func (a *authManagementService) BulkAddBaseMembers(ctx context.Context, schema string, baseID string, req dto.BulkAddBaseMembersRequest, userID string) (dto.BulkAddMembersResponse, error) {
	result := dto.BulkAddMembersResponse{
		Success: []string{},
		Failed:  []dto.MemberAddFailure{},
		Total:   len(req.Members),
	}

	for _, member := range req.Members {
		createReq := dto.CreateMemberRequest{
			UserID:     member.UserID,
			Membership: member.Memberships,
		}

		err := a.AssignUserToWorkspace(ctx, schema, createReq, userID)
		if err != nil {
			result.Failed = append(result.Failed, dto.MemberAddFailure{
				UserID: member.UserID,
				Error:  fmt.Sprintf("failed to assign member to base: %v", err),
			})
		} else {
			result.Success = append(result.Success, member.UserID)
		}
	}

	return result, nil
}

// GetWorkspaceMembersWithRole retrieves workspace members with their roles in UserWithRole format
// It checks for users who have access to the workspace either through:
// 1. workspace_id in scope_id (workspace-level access)
// 2. workspace_id as their membership workspace
func (a *authManagementService) GetWorkspaceMembersWithRole(ctx context.Context, schema string, workspaceID string) ([]dto.UserWithRole, error) {
	lg := logger.Get()
	functionName := "get_workspace_members_with_role"
	schemaFunctionName := fmt.Sprintf("%s.%s", appConstant.MasterDatabase, functionName)

	args := map[string]interface{}{
		"p_workspace_id": workspaceID,
	}

	records, err := a.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		lg.Error().Err(err).Msg("Failed to get workspace members with roles")
		return nil, app_errors.LogDatabaseError(err, "failed to get workspace members with roles")
	}

	fmt.Println("Records from get_workspace_members_with_role:", records)

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

	lg.Debug().Interface("result", result).Msg("Retrieved workspace members with roles")
	return result, nil
}

// GetBaseMembersWithRole retrieves base members with their roles in UserWithRole format
// It checks for users who have access to the base through:
// 1. base_id in scope_id (base-level access)
// 2. workspace members that have access to this base
func (a *authManagementService) GetBaseMembersWithRole(ctx context.Context, schema string, baseID string) ([]dto.UserWithRole, error) {
	lg := logger.Get()
	functionName := "get_base_members_with_role"
	schemaFunctionName := fmt.Sprintf("%s.%s", appConstant.MasterDatabase, functionName)

	args := map[string]interface{}{
		"p_base_id": baseID,
	}

	records, err := a.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		lg.Error().Err(err).Msg("Failed to get base members with roles")
		return nil, app_errors.LogDatabaseError(err, "failed to get base members with roles")
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

	lg.Debug().Interface("result", result).Msg("Retrieved base members with roles")
	return result, nil
}

// RemoveAccessMemberByID removes a member from access_members table by access_member_id
func (a *authManagementService) RemoveAccessMemberByID(ctx context.Context, schema string, accessMemberID string, reqBy string) error {
	lg := logger.Get()

	// Delete from access_members table using TableService
	tableName := fmt.Sprintf("\"%s\".access_members", schema)
	err := a.repo.TableService.DeleteRecord(tableName, accessMemberID)
	if err != nil {
		lg.Error().Err(err).Str("access_member_id", accessMemberID).Msg("Failed to remove access member")
		return err
	}

	lg.Info().Str("access_member_id", accessMemberID).Str("removed_by", reqBy).Msg("Successfully removed access member")
	return nil
}
