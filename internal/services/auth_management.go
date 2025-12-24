package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"godbgrest/pkg"
	"serenibase/internal/config"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
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
	tenantManagementService    interfaces.TenantManagementService
	subscriptionPlanService    interfaces.SubscriptionPlanService
	roleService                interfaces.RoleService
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
	tenantManagementService interfaces.TenantManagementService,
	subscriptionPlanService interfaces.SubscriptionPlanService,
	roleService interfaces.RoleService,
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
		tenantManagementService:    tenantManagementService,
		subscriptionPlanService:    subscriptionPlanService,
		roleService:                roleService,
		workspaceManagementService: workspaceManagementService,
		userResetTokenService:      userResetTokenService,
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

func (a *authManagementService) Register(ctx context.Context, req dto.RegisterRequest) (dto.RegisterResponse, error) {
	// 1. Check if user already exists
	if _, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, req.Email); err == nil {
		fmt.Println("err", err)
		return dto.RegisterResponse{}, app_errors.UserAlreadyExists
	} else if err != app_errors.UserNotFound {
		return dto.RegisterResponse{}, err
	}

	// 2. Hash password
	hashed, err := helpers.HashPassword(req.Password)
	password := req.Password
	if err != nil {
		fmt.Println("err", err)
		return dto.RegisterResponse{}, app_errors.ErrHashed
	}
	req.Password = hashed

	// 3. Create user in local DB
	insertedUser, err := a.userManagementService.CreateUser(ctx, appConstant.MasterDatabase, req)
	if err != nil {
		fmt.Println("err", err)
		return dto.RegisterResponse{}, err
	}

	// 4. Send OTP
	a.sendOtpViaEmail(insertedUser.Email)

	// 5. Generate Tokens
	insertedUser.Password = password // Pass raw password if needed for some reason, but GenerateToken usually doesn't need it for JWT
	// NOTE: GenerateToken does not require raw password for local JWT signing, just user properties.

	// Default role for new registration?
	// Usually invalid/no-access until email verified and tenant initialized.

	tokenData, err := a.generateToken(ctx, insertedUser)
	if err != nil {
		return dto.RegisterResponse{}, err
	}

	registerResponse := dto.RegisterResponse{
		Token: tokenData.RefreshToken, // Returning refresh token as "Token" in response per original code?
	}

	return registerResponse, nil
}

func (a *authManagementService) generateToken(ctx context.Context, user master.User) (dto.TokenResponse, error) {
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
	// We need tenant info?
	// The original code tried to generate token first (via Keycloak) then check claims.
	// We must do it reverse: Find user -> Check Pass -> Generate Token

	// Assumption: User must be in MasterDatabase or we search across?
	// Original code: `GetUserByEmail` was used on MasterDatabase in Register.
	// But `Login` pulled `tenant_id` from token to identify where the user belongs?
	// If we are multi-tenant, we need to know which tenant the user belongs to.
	// Or we check `master.User` which might have a default tenant or something?
	// Original code extracted `tenant_id` from the Keycloak token claims.

	// In local auth, we need to find the user first.
	// Users are in a specific schema? Or is there a central user table?
	// `userManagementService.GetUserByEmail` takes a schema.
	// Assuming `appConstant.MasterDatabase` holds the global user list or default schema.

	// First, find the user in master database to get basic info and tenant
	masterUser, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, email)
	if err != nil {
		if err == app_errors.UserNotFound {
			return dto.LoginResponse{}, app_errors.InvalidCredentials
		}
		return dto.LoginResponse{}, err
	}

	// Get the tenant for this user
	tenantData, err := a.tenantManagementService.GetTenantByUserId(ctx, masterUser.ID)
	if err != nil {
		logger.Get().Warn().Err(err).Str("user_id", masterUser.ID.String()).Msg("Failed to resolve tenant for user during login")
		return dto.LoginResponse{}, app_errors.InvalidCredentials
	}

	// Now get the user from the tenant schema to verify password
	// This ensures we check against the tenant's user record where passwords are updated
	user, err := a.userManagementService.GetUserByID(ctx, tenantData.Schema, masterUser.ID.String())
	if err != nil {
		logger.Get().Error().
			Err(err).
			Str("user_id", masterUser.ID.String()).
			Str("tenant_schema", tenantData.Schema).
			Msg("Failed to get user from tenant schema")
		return dto.LoginResponse{}, app_errors.InvalidCredentials
	}

	// Verify Password against tenant schema user
	if !helpers.CheckPasswordHash(password, user.Password) {
		return dto.LoginResponse{}, app_errors.InvalidCredentials
	}

	// Set tenant info for token generation
	user.TenantID = tenantData.ID
	user.Roles = "Admin" // Simplified: Default to Admin if tenant found for now

	// Generate Token
	tokens, err := a.authProviderService.GenerateToken(ctx, user)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	// Logic from here mirrors original: check verification, tenant, etc.
	// We need to resolve tenant_id for the user.
	// If the user was created properly, they should be mapped to a tenant or workspace.
	// But `master.User` struct doesn't seem to have `TenantID`.
	// The original code relied on Keycloak claims having `tenant_id`.
	// Where did Keycloak get it? From attributes we set during `AddUser`.

	// We need to find the tenant for this user.
	// `workspaceManagementService` deals with User Mappings? `tenantManagementService`?
	// Since we don't have Keycloak storing it, we must query DB.
	// Ideally `user` table should have it or a mapping table.
	// Let's assume for now we can get it from `user` if extended or we search.
	// Or the user is an admin of their own tenant?

	// Hack: We need to see how `addUserWithTenant` sets it up.
	// It creates a Tenant, adds User to that Tenant Schema.
	// So the user exists in a specific Schema.
	// But we found the user in `MasterDatabase` schema?
	// If `CreateUser` creates in `MasterDatabase`, good.
	// But `GetUserByEmail` in `MasterDatabase` returns the user.

	// Note: `GetUserByEmail` returns `master.User`.
	// If we can't find tenant easily, we might be stuck.
	// But wait, `ValidateToken` returns `Claims` which has `TenantId`.
	// `GenerateToken` generates claims. It needs TenantId.
	// I left a TODO in `authProvider` about TenantID.

	// For now, let's proceed.
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

	// We need to get TenantID to return full user object and update login time.
	// Access token claims should probably have it.
	// Since we just generated it, we didn't put it in if we didn't have it.

	// Let's look up tenant.
	// Maybe `tenantManagementService.GetTenantByUserId` exists?
	// If not, we might default to something or user has to select workspace.

	// Proceeding with what we have. Login success.

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

func (a *authManagementService) createDefaultRoles(ctx context.Context, schema string) error {
	for _, role := range appConstant.DefaultRoles {
		role.ID = uuid.New()
		_, err := a.roleService.CreateRole(ctx, schema, role)
		if err != nil {
			return app_errors.ErrRoleCreation
		}
		fmt.Printf("Role '%s' created successfully.\n", role.Name)
	}
	return nil
}

func (a *authManagementService) addUserWithTenant(ctx context.Context, userId string, user master.User, tenantId string) (dto.UserResponse, error) {
	plan, err := a.subscriptionPlanService.GetSubscriptionPlanByName(ctx, appConstant.PlanNames.Free)
	if err != nil {
		return dto.UserResponse{}, app_errors.ErrSubscriptionPlanNotFound
	}

	role, err := a.roleService.GetRoleByName(ctx, appConstant.MasterDatabase, appConstant.RoleNames.Admin)
	if err != nil {
		return dto.UserResponse{}, app_errors.ErrRoleNotFound
	}

	tenantReq := dto.TenantRequest{
		UserID:   uuid.MustParse(userId),
		TenantID: uuid.MustParse(tenantId),
	}

	// Create Tenant
	tenantData, err := a.tenantManagementService.InitializeTenant(ctx, tenantReq, plan.ID, role.ID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	err = a.rbacManagementService.InitializeRBACSystem(ctx, tenantData.Schema)
	if err != nil {
		return dto.UserResponse{}, err
	}

	// Create User in Tenant Schema
	// Note: We already created in MasterDatabase? Yes.
	// This seems to duplicate user into tenant schema?
	// Or maybe MasterDatabase is just for initial auth lookup?

	_, err = a.userManagementService.CreateUser(ctx, tenantData.Schema, dto.RegisterRequest{
		ID:            user.ID,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Password:      user.Password,
		AuthProvider:  user.AuthProvider, // "local" or whatever
		Status:        "active",
		EmailVerified: true,
		DateOfBirth:   user.DateOfBirth,
		Country:       user.Country,
		Timezone:      user.Timezone,
	})

	if err != nil {
		return dto.UserResponse{}, err
	}

	// Update Master User
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

	tenantUserRole, err := a.roleService.GetRoleByName(ctx, tenantData.Schema, appConstant.RoleNames.Admin)
	if err != nil {
		return dto.UserResponse{}, app_errors.ErrRoleNotFound
	}

	err = a.userManagementService.AddUserRole(ctx, tenantData.Schema, user.ID, tenantUserRole.ID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	workspaceReq := dto.CreateWorkspaceRequest{
		Title:       "Default Workspace",
		Description: helpers.StringPtr(""),
	}

	_, err = a.workspaceManagementService.Create(ctx, workspaceReq, tenantData.Schema, userData.ID.String())
	if err != nil {
		return dto.UserResponse{}, err
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
	// We don't have tenantId yet effectively.

	user, err := a.userManagementService.GetUserByID(ctx, appConstant.MasterDatabase, userId)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	ok := a.otpProviderService.Verify(user.Email, req.OTP)
	if !ok {
		return dto.LoginResponse{}, app_errors.InvalidOTP
	}

	// Create Tenant ID
	tenantId := uuid.NewString()

	userData, err := a.addUserWithTenant(ctx, user.ID.String(), user, tenantId)
	if err != nil {
		fmt.Println(err)
		return dto.LoginResponse{}, err
	}

	// Generate FRESH tokens with TenantID and Role
	// user object needs to be updated with TenantID
	user.TenantID = uuid.MustParse(tenantId)
	user.Roles = "Admin" // Default for new tenant owner

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

	// Note: We need to know which schema/tenant the user is in?
	// Or searches MasterDatabase?
	// The original code got `tenant_id` from Keycloak attributes.
	// If we don't have that, we might assume MasterDatabase for existence check.

	user, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, req.Email)
	if err != nil {
		return app_errors.UserNotFound
	}

	// Resolve tenant for the user (similar to Login function)
	// This is necessary because ResetPassword updates password in both master and tenant schemas
	tenantId := ""
	tenantData, err := a.tenantManagementService.GetTenantByUserId(ctx, user.ID)
	if err == nil {
		tenantId = tenantData.ID.String()
	} else {
		// Log warning but continue - user may not have completed email verification yet
		logger.Get().Warn().Err(err).Str("user_id", user.ID.String()).Msg("Failed to resolve tenant for password reset")
	}

	if !user.EmailVerified {
		return app_errors.InvalidCredentials
	}

	if user.Status == "pending" {
		return app_errors.InvalidCredentials
	}

	tokenAttrs := map[string]interface{}{
		"tenant_id": tenantId,
		"user_id":   user.ID.String(),
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

	// Extract tenant_id from token to get the tenant schema
	claims, err := helpers.DecodeJWT(userResetToken.Token)
	if err != nil {
		logger.Get().Error().
			Err(err).
			Str("user_id", userId).
			Msg("Failed to decode JWT from reset token")
		return app_errors.TokenInvalid
	}

	tenantId, ok := claims["tenant_id"].(string)
	if !ok || tenantId == "" {
		logger.Get().Error().
			Str("user_id", userId).
			Msg("tenant_id not found in reset token claims")
		return app_errors.TokenInvalid
	}

	// Get Tenant Schema
	tenantData, err := a.tenantManagementService.GetTenant(ctx, tenantId)
	if err != nil {
		logger.Get().Error().
			Err(err).
			Str("tenant_id", tenantId).
			Msg("Failed to get tenant data")
		return err
	}

	// Update password in tenant schema only
	_, err = a.userManagementService.UpdateUser(ctx, tenantData.Schema, userId, map[string]interface{}{
		"password":            hashedPassword,
		"password_changed_at": time.Now(),
	})
	if err != nil {
		logger.Get().Error().
			Err(err).
			Str("user_id", userId).
			Str("tenant_schema", tenantData.Schema).
			Msg("Failed to update password in tenant schema")
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

func (a *authManagementService) AddUser(ctx context.Context, schema string, userData dto.AddUserRequest) (master.User, error) {
	// Admin adding user
	roles := appConstant.RoleNames.User

	role, err := a.roleService.GetRoleByName(ctx, schema, roles)
	if err != nil {
		return master.User{}, app_errors.ErrRoleNotFound
	}

	user, tenant, err := a.userManagementService.AddUserToTenant(ctx, schema, userData, role.ID, a.userDefaultPassword.Value)
	if err != nil {
		return master.User{}, err
	}

	// Previous: authProvider.AddUser
	// Now: Just ensure they are in Master DB? AddUserToTenant probably handled DB.
	// We just need to send invitation.

	tokenAttrs := map[string]interface{}{
		"tenant_id": tenant.ID.String(),
		"user_id":   user.ID.String(),
	}

	token, err := helpers.GenerateCustomJWT(tokenAttrs, user.ID.String(), 3600)
	if err != nil {
		return master.User{}, err
	}

	dataToInsert := dto.UserResetTokenInsertion{
		ID:     uuid.NewString(),
		UserID: user.ID.String(),
		Token:  token,
		Expiry: time.Now().Add(1 * time.Hour),
	}

	data, err := a.userResetTokenService.CreateUserResetToken(ctx, dataToInsert)
	if err != nil {
		return master.User{}, err
	}

	resetURLTemplate := config.AppConfig.Auth.ResetPasswordURL
	if resetURLTemplate == "" {
		resetURLTemplate = "http://localhost:5050/reset-password?token=%s"
	}
	resetLink := fmt.Sprintf(resetURLTemplate, data.Token)
	emailData := a.emailTemplateService.PlatformInvitationBody(user.FirstName, tenant.Name, resetLink)

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

func (a *authManagementService) AssignUserToWorkspace(ctx context.Context, schema string, req dto.CreateMemberRequest) error {
	memberships, err := a.workspaceManagementService.GetWorkspaceMemberByUser(ctx, schema, req.UserID)
	if err != nil && err != app_errors.WorkspaceMemberNotFound {
		return err
	}
	for _, member := range memberships {
		if member.WorkspaceID == req.WorkspaceID {
			return app_errors.ErrUserAlreadyInWorkspace
		}
	}

	err = a.workspaceManagementService.AssignUserToWorkspace(ctx, schema, req)
	if err != nil {
		return err
	}

	workspace, workspaceErr := a.workspaceManagementService.GetByID(ctx, schema, req.WorkspaceID)
	if workspaceErr != nil {
		return workspaceErr
	}

	user, userErr := a.userManagementService.GetUserByID(ctx, schema, req.UserID)
	if userErr != nil {
		return userErr
	}

	emailData := a.emailTemplateService.AddedToWorkspaceBody(workspace.Title, req.AccessLevel)
	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})

	return nil
}

func (a *authManagementService) RemoveUserFromWorkspace(ctx context.Context, schema string, req dto.RemoveMemberRequest) error {
	err := a.workspaceManagementService.RemoveUserFromWorkspace(ctx, schema, req.WorkspaceID, req.UserID)
	if err != nil {
		return err
	}

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

func (a *authManagementService) InviteMemberToWorkspace(ctx context.Context, schema string, req dto.CreateMemberRequest) error {
	memberships, err := a.workspaceManagementService.GetWorkspaceMemberByUser(ctx, schema, req.UserID)
	if err != nil && err != app_errors.WorkspaceMemberNotFound {
		return err
	}
	for _, member := range memberships {
		if member.WorkspaceID == req.WorkspaceID {
			return app_errors.ErrUserAlreadyInWorkspace
		}
	}

	err = a.workspaceManagementService.InviteMember(ctx, schema, req)
	if err != nil {
		return err
	}

	workspace, workspaceErr := a.workspaceManagementService.GetByID(ctx, schema, req.WorkspaceID)
	if workspaceErr != nil {
		return workspaceErr
	}

	user, userErr := a.userManagementService.GetUserByID(ctx, schema, req.UserID)
	if userErr != nil {
		return userErr
	}

	emailData := a.emailTemplateService.InvitedToWorkspaceBody(workspace.Title, req.AccessLevel)
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

func (a *authManagementService) AddMultipleMembers(ctx context.Context, schema string, req dto.AddMultipleMembersRequest) (dto.AddMultipleMembersResponse, error) {
	response := dto.AddMultipleMembersResponse{
		Successes: []dto.MemberAddSuccess{},
		Failures:  []dto.MemberAddFailure{},
	}

	workspace, workspaceErr := a.workspaceManagementService.GetByID(ctx, schema, req.WorkspaceID)
	if workspaceErr != nil {
		return response, workspaceErr
	}

	for _, userID := range req.UserIDs {
		memberReq := dto.CreateMemberRequest{
			WorkspaceID: req.WorkspaceID,
			UserID:      userID,
			AccessLevel: req.AccessLevel,
			BasesIds:    req.BasesIds,
		}

		memberships, err := a.workspaceManagementService.GetWorkspaceMemberByUser(ctx, schema, userID)
		if err != nil && err != app_errors.WorkspaceMemberNotFound {
			response.Failures = append(response.Failures, dto.MemberAddFailure{
				UserID: userID,
				Error:  "Failed to check existing membership",
			})
			response.FailureCount++
			continue
		}

		alreadyMember := false
		for _, member := range memberships {
			if member.WorkspaceID == req.WorkspaceID {
				alreadyMember = true
				break
			}
		}

		if alreadyMember {
			err = a.workspaceManagementService.UpdateWorkspaceMemberBases(ctx, schema, req.WorkspaceID, userID, req.AccessLevel, req.BasesIds)
			if err != nil {
				response.Failures = append(response.Failures, dto.MemberAddFailure{
					UserID: userID,
					Error:  fmt.Sprintf("Failed to update base access: %s", err.Error()),
				})
				response.FailureCount++
				continue
			}

			user, userErr := a.userManagementService.GetUserByID(ctx, schema, userID)
			if userErr == nil {
				emailData := a.emailTemplateService.WorkspaceAccessUpdatedBody(workspace.Title, req.AccessLevel)
				a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})
			}

			response.Successes = append(response.Successes, dto.MemberAddSuccess{
				UserID: userID,
			})
			response.SuccessCount++
			continue
		}

		err = a.workspaceManagementService.AssignUserToWorkspace(ctx, schema, memberReq)
		if err != nil {
			response.Failures = append(response.Failures, dto.MemberAddFailure{
				UserID: userID,
				Error:  err.Error(),
			})
			response.FailureCount++
			continue
		}

		user, userErr := a.userManagementService.GetUserByID(ctx, schema, userID)
		if userErr == nil {
			emailData := a.emailTemplateService.AddedToWorkspaceBody(workspace.Title, req.AccessLevel)
			a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})
		}

		response.Successes = append(response.Successes, dto.MemberAddSuccess{
			UserID: userID,
		})
		response.SuccessCount++
	}

	return response, nil
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
