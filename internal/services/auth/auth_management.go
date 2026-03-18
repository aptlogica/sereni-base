// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	rbac "github.com/aptlogica/sereni-base/internal/services/rbac"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	authProviderInterface "github.com/aptlogica/sereni-base/internal/providers/auth"
	emailProvider "github.com/aptlogica/sereni-base/internal/providers/email"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	otpProvider "github.com/aptlogica/sereni-base/internal/providers/otp"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	appConstant "github.com/aptlogica/sereni-base/internal/constant"

	appConfig "github.com/aptlogica/sereni-base/internal/config"

	"github.com/google/uuid"
)

// AuthManagementServiceDeps holds all service dependencies
type AuthManagementServiceDeps struct {
	UserManagementService      interfaces.UserManagementService
	WorkspaceManagementService interfaces.WorkspaceManagementService
	UserResetTokenService      interfaces.UserResetTokenService
	RBACManagementService      interfaces.RBACManagementService
}

// AuthManagementProviderDeps holds all provider dependencies
type AuthManagementProviderDeps struct {
	OTPProviderService   otpProvider.OtpService
	EmailTemplateService emailProvider.EmailTemplateService
	EmailProviderService emailProvider.EmailService
	AuthProviderService  authProviderInterface.AuthProvider
}

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
	serviceDeps AuthManagementServiceDeps,
	providerDeps AuthManagementProviderDeps,
) interfaces.AuthManagementService {
	return &authManagementService{
		userDefaultPassword:        userDefaultPassword,
		repo:                       repo,
		userManagementService:      serviceDeps.UserManagementService,
		workspaceManagementService: serviceDeps.WorkspaceManagementService,
		userResetTokenService:      serviceDeps.UserResetTokenService,
		rbacManagementService:      serviceDeps.RBACManagementService,
		otpProviderService:         providerDeps.OTPProviderService,
		emailTemplateService:       providerDeps.EmailTemplateService,
		emailProviderService:       providerDeps.EmailProviderService,
		authProviderService:        providerDeps.AuthProviderService,
	}
}

func (s *authManagementService) sendOtpViaEmail(email string) {
	otp := s.otpProviderService.Generate(email)
	emailData := s.emailTemplateService.EmailVerificationOTPBody(otp)
	s.emailProviderService.Enqueue(emailProvider.EmailJob{To: email, Subject: emailData.Subject, Body: emailData.Body})
}

func (a *authManagementService) RegisterOwner(ctx context.Context, req dto.RegisterRequest) (dto.LoginResponse, error) {
	// Hash password for local storage
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

	// Use the RBAC role names that match the system

	// 5. Verify Email (Skip OTP for owner) && Initialize
	userData, err := a.initializeOwner(ctx, insertedUser.ID.String(), insertedUser)
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("failed to initialize owner: %w", err)
	}

	// 6. Assign Owner role to user at system level
	roleData, err := a.rbacManagementService.GetRoleByName(ctx, appConstant.MasterDatabase, appConstant.RBACRoleNames.Owner)
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("failed to get Owner role: %w", err)
	}

	accessMemberReq := dto.AccessMemberDTO{
		UserID:     insertedUser.ID.String(),
		ScopeType:  appConstant.ScopeLevels.System,
		ScopeID:    nil,
		RoleID:     roleData.ID.String(),
		AssignedBy: helpers.StringPtr(insertedUser.ID.String()),
	}

	_, err = a.rbacManagementService.AssignRoleToUser(ctx, appConstant.MasterDatabase, accessMemberReq)
	if err != nil {

		return dto.LoginResponse{}, fmt.Errorf("failed to assign Owner role: %w", err)
	}

	// 7. Create default workspace
	createDefaultWorkspace := dto.CreateWorkspaceRequest{
		Title:       "Default Workspace",
		Description: helpers.StringPtr(""),
		CreatedBy:   insertedUser.ID.String(),
	}

	workspace, err := a.workspaceManagementService.Create(ctx, createDefaultWorkspace, appConstant.MasterDatabase, insertedUser.ID.String())
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("failed to create default workspace: %w", err)
	}

	// 8. Add owner to workspace_members table for legacy compatibility
	workspaceMemberData := dto.WorkspaceMemberInsertion{
		ID:          uuid.New(),
		WorkspaceID: workspace.ID.String(),
		UserID:      insertedUser.ID.String(),
		AccessLevel: "owner",
		BasesIds:    "",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	tableName := tenant.WorkspaceMember{}.TableName(appConstant.MasterDatabase)
	_, _ = a.repo.TableService.CreateRecord(tableName, workspaceMemberData.Map())

	loginResponse := dto.LoginResponse{
		User: &userData,
	}

	return loginResponse, nil
}

func (a *authManagementService) Login(ctx context.Context, email string, password string) (dto.LoginResponse, error) {
	// Check user existence in sereni-base
	user, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, email)
	if err != nil {
		if err == app_errors.UserNotFound {
			return dto.LoginResponse{}, app_errors.InvalidCredentials
		}
		return dto.LoginResponse{}, err
	}

	if user.Status != "active" {
		return dto.LoginResponse{}, app_errors.UserNotActive
	}

	// Verify Password locally first (as source of truth)
	if !helpers.CheckPasswordHash(password, user.Password) {
		return dto.LoginResponse{}, app_errors.InvalidCredentials
	}

	// Call JWT service login endpoint to get tokens
	reqBody := authProviderInterface.AuthServiceLoginRequest{
		Id:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Roles:         []string{user.Roles},
	}
	tokens, err := a.authProviderService.Login(ctx, reqBody)
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("failed to authenticate with JWT service: %w", err)
	}

	// If email not verified:
	if !user.EmailVerified {
		return dto.LoginResponse{
			User: &dto.UserResponse{
				ID: user.ID,
			},
			Token: &dto.TokenResponse{
				RefreshToken: tokens.RefreshToken,
			},
		}, nil
	}

	var userResponse dto.UserResponse
	if err := helpers.StructToStruct(user, &userResponse); err != nil {
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

	// Extract info from token
	// Assuming RefreshToken has UserID in claims
	claims, err := a.authProviderService.ValidateToken(ctx, req.Token)
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
		return dto.LoginResponse{}, err
	}

	// Generate FRESH tokens
	user.Roles = "Admin" // Default for new owner? Or regular user? verifyEmail usually implies general user.

	// We also need to update user.EmailVerified = true locally for token generation
	user.EmailVerified = true

	reqBody := authProviderInterface.AuthServiceLoginRequest{
		Id:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Roles:         []string{user.Roles},
	}
	tokens, err := a.authProviderService.Login(ctx, reqBody)
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

	claims, err := a.authProviderService.ValidateToken(ctx, req.Token)
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

	reqBody := authProviderInterface.AuthServiceRefreshRequest{
		RefreshToken: req.RefreshToken,
	}
	tokens, err := a.authProviderService.RefreshToken(ctx, reqBody)
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

// extractIssuedAtFromToken extracts the iat claim as a string from a JWT token
func extractIssuedAtFromToken(token string) (string, error) {
	claims, err := helpers.DecodeJWT(token)
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT for iat: %w", err)
	}
	if iat, ok := claims["iat"].(float64); ok {
		return fmt.Sprintf("%d", int64(iat)), nil
	}
	return "", fmt.Errorf("iat claim not found in token")
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

	token, err := helpers.GenerateCustomJWT(tokenAttrs, user.ID.String(), 259200)
	if err != nil {
		return err
	}

	issuedAtStr, err := extractIssuedAtFromToken(token)
	if err != nil {
		issuedAtStr = fmt.Sprintf("%d", time.Now().Unix())
	}
	dataToInsert := dto.UserResetTokenInsertion{
		ID:       uuid.NewString(),
		UserID:   user.ID.String(),
		Token:    token,
		IssuedAt: issuedAtStr,
	}
	_, err = a.userResetTokenService.CreateUserResetToken(ctx, dataToInsert)
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

	tokenIssuedAt, err := extractIssuedAtFromToken(userResetToken.Token)
	if err != nil {
		return app_errors.TokenInvalid
	}
	latestIssuedAt := userResetToken.IssuedAt
	if tokenIssuedAt != latestIssuedAt {
		return fmt.Errorf("reset token issued date does not match latest issued date in user_reset_tokens")
	}

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
	// No expiry check here, only validity of token
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
	if err := a.validateEmailAvailability(ctx, schema, userData.Email); err != nil {
		return tenant.User{}, err
	}

	// Admin adding user
	roles := a.determineUserRole(userData.IsCoOwner)

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
	if err := a.handleProfilePicture(ctx, schema, user.ID.String(), userData.ProfilePic); err != nil {
		return tenant.User{}, err
	}

	// Send invitation
	token, err := a.generateInvitationToken(ctx, user.ID.String())
	if err != nil {
		return tenant.User{}, err
	}

	// handle membership invitations if any
	if err := a.handleMembershipSetup(ctx, schema, user.ID.String(), roles, reqBy, userData.Membership); err != nil {
		return tenant.User{}, err
	}

	// Send invitation email
	a.sendInvitationEmail(user, token)

	return user, nil
}

func (a *authManagementService) validateEmailAvailability(ctx context.Context, schema string, email string) error {
	_, err := a.userManagementService.GetUserByEmail(ctx, schema, email)
	if err == nil {
		return app_errors.UserAlreadyExists
	} else if err != app_errors.UserNotFound {
		return err
	}
	return nil
}

func (a *authManagementService) determineUserRole(isCoOwner bool) string {
	if isCoOwner {
		return appConstant.RBACRoleNames.CoOwner
	}
	return appConstant.RBACRoleNames.NoAccess
}

func (a *authManagementService) handleProfilePicture(ctx context.Context, schema, userID string, profilePic *multipart.FileHeader) error {
	if profilePic != nil {
		_, err := a.userManagementService.AddAvatar(ctx, schema, userID, profilePic)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *authManagementService) generateInvitationToken(ctx context.Context, userID string) (string, error) {
	tokenAttrs := map[string]interface{}{
		"user_id": userID,
	}

	token, err := helpers.GenerateCustomJWT(tokenAttrs, userID, 259200)
	if err != nil {
		return "", err
	}

	issuedAtStr, err := extractIssuedAtFromToken(token)
	if err != nil {
		issuedAtStr = fmt.Sprintf("%d", time.Now().Unix())
	}
	dataToInsert := dto.UserResetTokenInsertion{
		ID:       uuid.NewString(),
		UserID:   userID,
		Token:    token,
		IssuedAt: issuedAtStr,
	}

	data, err := a.userResetTokenService.CreateUserResetToken(ctx, dataToInsert)
	if err != nil {
		return "", err
	}

	return data.Token, nil
}

func (a *authManagementService) handleMembershipSetup(ctx context.Context, schema, userID, roles, reqBy string, membership []dto.MembershipRequest) error {
	if roles == appConstant.RBACRoleNames.CoOwner {
		roleData, err := a.rbacManagementService.GetRoleByName(ctx, appConstant.MasterDatabase, roles)
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
		if err != nil {
			return err
		}
	} else {
		_, err := a.rbacManagementService.ProcessUserMemberships(ctx, schema, userID, userID, membership)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *authManagementService) sendInvitationEmail(user tenant.User, token string) {
	resetURLTemplate := appConfig.AppConfig.Auth.ResetPasswordURL
	if resetURLTemplate == "" {
		resetURLTemplate = "http://localhost:5050/reset-password?token=%s"
	}
	resetLink := fmt.Sprintf(resetURLTemplate, token)

	emailData := a.emailTemplateService.PlatformInvitationBody(user.FirstName, "SereniBase", resetLink)

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})
}

// EditUser updates user details with support for profile, avatar, membership, and co-owner status changes
func (a *authManagementService) EditUser(ctx context.Context, schema string, userData dto.EditUserRequest, reqBy string) (dto.UserResponse, error) {
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

	if userData.IsCoOwner != nil {
		if err := a.processCoOwnerChanges(ctx, schema, userData, reqBy); err != nil {
			logger.Get().Error().Err(err).Str("user_id", userData.UserID).Msg("Failed to process CoOwner changes")
			return dto.UserResponse{}, err
		}
	}

	if userData.IsCoOwner == nil || !*userData.IsCoOwner {
		if err := a.updateMemberships(ctx, schema, userData, reqBy); err != nil {
			return dto.UserResponse{}, err
		}
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

type membershipKey struct {
	ScopeType string // "workspace" or "base"
	ScopeID   string // workspace_id or base_id
	Role      string // role name
}

func (a *authManagementService) buildNewMembershipsMap(memberships []dto.MembershipRequest) map[membershipKey]struct{} {
	newMemberships := make(map[membershipKey]struct{})
	for _, m := range memberships {
		if len(m.Bases) == 0 {
			key := membershipKey{ScopeType: "workspace", ScopeID: m.WorkspaceID, Role: m.Role}
			newMemberships[key] = struct{}{}
		} else {
			for _, b := range m.Bases {
				key := membershipKey{ScopeType: "base", ScopeID: b.BaseID, Role: b.Role}
				newMemberships[key] = struct{}{}
			}
		}
	}
	return newMemberships
}

func (a *authManagementService) removeObsoleteMemberships(ctx context.Context, schema, reqBy string, currentMembers []dto.AccessMemberDTO, newMemberships map[membershipKey]struct{}) {
	for _, member := range currentMembers {
		key := a.buildMembershipKeyFromAccessMember(member)
		if _, found := newMemberships[key]; !found {
			_ = a.RemoveAccessMemberByID(ctx, schema, member.ID.String(), reqBy)
		}
	}
}

func (a *authManagementService) buildMembershipKeyFromAccessMember(member dto.AccessMemberDTO) membershipKey {
	var key membershipKey
	if member.ScopeID != nil {
		key = membershipKey{ScopeType: member.ScopeType, ScopeID: *member.ScopeID, Role: member.RoleID}
	}
	return key
}

func (a *authManagementService) updateMemberships(ctx context.Context, schema string, userData dto.EditUserRequest, reqBy string) error {
	isOwner, err := a.checkIfUserIsOwner(ctx, schema, userData.UserID)
	if err != nil {
		return err
	}
	if isOwner {
		return nil
	}

	currentAccessMembers, err := a.rbacManagementService.GetUserAccessMembers(ctx, schema, userData.UserID)
	if err != nil {
		return err
	}

	newMemberships := a.buildNewMembershipsMap(userData.Membership)
	a.removeObsoleteMemberships(ctx, schema, reqBy, currentAccessMembers, newMemberships)

	if len(userData.Membership) == 0 {
		return nil
	}

	_, err = a.rbacManagementService.ProcessUserMemberships(ctx, schema, userData.UserID, reqBy, userData.Membership)
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
	isOwner, err := a.checkIfUserIsOwner(ctx, schema, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}
	if isOwner {
		return dto.UserResponse{}, app_errors.OwnerCannotBeDeactivated
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

func (a *authManagementService) checkIfUserIsOwner(ctx context.Context, schema string, userID string) (bool, error) {
	functionName := "get_user_role_by_id"
	schemaFunctionName := fmt.Sprintf("%s.%s", appConstant.MasterDatabase, functionName)

	args := map[string]interface{}{
		"p_user_id": userID,
	}

	records, err := a.repo.TableService.GetByFunction(ctx, schemaFunctionName, args)
	if err != nil {
		return false, err
	}

	for _, record := range records {
		roleData := a.parseRoleData(record, functionName)
		if roleData != nil && a.isOwnerRole(roleData) {
			return true, nil
		}
	}

	return false, nil
}

// parseRoleData extracts and parses role data from a record
func (a *authManagementService) parseRoleData(record map[string]interface{}, functionName string) map[string]interface{} {
	value, exists := record[functionName]
	if !exists {
		return nil
	}

	switch v := value.(type) {
	case string:
		var roleData map[string]interface{}
		if err := json.Unmarshal([]byte(v), &roleData); err == nil {
			return roleData
		}
	case map[string]interface{}:
		return v
	}
	return nil
}

// isOwnerRole checks if the role data indicates an owner role
func (a *authManagementService) isOwnerRole(roleData map[string]interface{}) bool {
	roleName, exists := roleData["role_name"].(string)
	return exists && roleName == appConstant.RBACRoleNames.Owner
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
	accessMembersTableName := fmt.Sprintf(rbac.AccessMembersTableFormat, schema)
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
	accessMembersTableName := fmt.Sprintf(rbac.AccessMembersTableFormat, schema)
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
	_, err := a.userManagementService.UpdatePassword(ctx, schema, userID, updateData)
	return err
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
		return nil, app_errors.LogDatabaseError(err, "failed to get workspace members with roles")
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

// GetBaseMembersWithRole retrieves base members with their roles in UserWithRole format
// It checks for users who have access to the base through:
// 1. base_id in scope_id (base-level access)
// 2. workspace members that have access to this base
func (a *authManagementService) GetBaseMembersWithRole(ctx context.Context, schema string, baseID string) ([]dto.UserWithRole, error) {
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
		return nil, app_errors.LogDatabaseError(err, "failed to get base members with roles")
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

// RemoveAccessMemberByID removes a member from access_members table by access_member_id
func (a *authManagementService) RemoveAccessMemberByID(ctx context.Context, schema string, accessMemberID string, reqBy string) error {

	// Delete from access_members table using TableService
	tableName := fmt.Sprintf(rbac.AccessMembersTableFormat, schema)
	err := a.repo.TableService.DeleteRecord(tableName, accessMemberID)
	if err != nil {
		return err
	}

	return nil
}
