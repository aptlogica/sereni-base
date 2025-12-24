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
	role := appConstant.RoleNames.Admin
	if _, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, req.Email); err == nil {
		fmt.Println("err", err)
		return dto.RegisterResponse{}, app_errors.UserAlreadyExists
	} else if err != app_errors.UserNotFound {
		return dto.RegisterResponse{}, err
	}
	hashed, err := helpers.HashPassword(req.Password)
	password := req.Password
	if err != nil {
		fmt.Println("err", err)
		return dto.RegisterResponse{}, app_errors.ErrHashed
	}
	req.Password = hashed

	insertedUser, err := a.userManagementService.CreateUser(ctx, appConstant.MasterDatabase, req)
	if err != nil {
		fmt.Println("err", err)
		return dto.RegisterResponse{}, err
	}

	a.sendOtpViaEmail(insertedUser.Email)

	insertedUser.Password = password
	tokenData, err := a.generateToken(ctx, insertedUser, uuid.NewString(), role)
	if err != nil {
		return dto.RegisterResponse{}, err
	}

	registerResponse := dto.RegisterResponse{
		Token: tokenData.RefreshToken,
	}

	return registerResponse, nil
}

func (a *authManagementService) generateToken(ctx context.Context, user master.User, tenant_id string, roles string) (dto.TokenResponse, error) {
	_, err := a.authProviderService.AddUser(ctx, user, tenant_id, roles)
	if err != nil {
		return dto.TokenResponse{}, err
	}

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
	userData := master.User{
		Email:    email,
		Password: password,
	}
	tokens, err := a.authProviderService.GenerateToken(ctx, userData)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	claims, err := a.extractValuesFromToken(tokens.AccessToken, []string{"email_verified", "tenant_id", "user_id"})
	emailVerified := false
	if err == nil {
		if val, ok := claims["email_verified"]; ok {
			switch v := val.(type) {
			case bool:
				emailVerified = v
			case string:
				emailVerified = v == "true"
			}
		}
	}

	tenantID, ok := claims["tenant_id"].(string)
	if !ok || tenantID == "" {
		return dto.LoginResponse{}, app_errors.InvalidCredentials
	}
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return dto.LoginResponse{}, app_errors.InvalidCredentials
	}

	parsedUUID, err := uuid.Parse(userID)
	if err != nil {
		return dto.LoginResponse{}, app_errors.InvalidCredentials
	}

	if !emailVerified {
		return dto.LoginResponse{
			User: &dto.UserResponse{
				ID: parsedUUID,
			},
			Token: &dto.TokenResponse{
				RefreshToken: tokens.RefreshToken,
			},
		}, nil
	}

	tenantData, err := a.tenantManagementService.GetTenant(ctx, tenantID)
	if err != nil {
		if err == app_errors.TenantNotFound {
			return dto.LoginResponse{}, app_errors.TenantNotFound
		}
		return dto.LoginResponse{}, app_errors.ErrMapToStruct
	}

	var tenantResponse dto.TenantResponse
	if err := helpers.StructToStruct(tenantData, &tenantResponse); err != nil {
		return dto.LoginResponse{}, app_errors.ErrMapToStruct
	}

	user, err := a.userManagementService.GetUserByID(ctx, tenantData.Schema, userID)
	if err != nil {
		if err == app_errors.UserNotFound {
			return dto.LoginResponse{}, app_errors.InvalidCredentials
		}
		return dto.LoginResponse{}, err
	}

	fmt.Println("user--->", user)

	// Update last_login_at on every login
	updateData := map[string]interface{}{
		"last_login_at": time.Now(),
	}
	updatedUser, err := a.userManagementService.UpdateUser(ctx, tenantData.Schema, userID, updateData)
	if err != nil {
		// Log the error but don't fail the login
		fmt.Println("Failed to update last_login_at:", err)
	} else {
		user = updatedUser
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

	tenantData, err := a.tenantManagementService.InitializeTenant(ctx, tenantReq, plan.ID, role.ID)
	if err != nil {
		return dto.UserResponse{}, err
	}
	fmt.Println("tenantData--->", tenantData)

	err = a.rbacManagementService.InitializeRBACSystem(ctx, tenantData.Schema)
	if err != nil {
		return dto.UserResponse{}, err
	}
	fmt.Println("tenantData--->", tenantData)

	_, err = a.userManagementService.CreateUser(ctx, tenantData.Schema, dto.RegisterRequest{
		ID:            user.ID,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Password:      user.Password,
		AuthProvider:  user.AuthProvider,
		Status:        "active",
		EmailVerified: true,
		DateOfBirth:   user.DateOfBirth,
		Country:       user.Country,
		Timezone:      user.Timezone,
	})

	if err != nil {
		return dto.UserResponse{}, err
	}
	fmt.Println("user created--->")

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
	refreshToken := req.Token
	tokenData, err := a.authProviderService.RefreshToken(ctx, refreshToken)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	fmt.Println("tokenData--->", tokenData)

	claims, err := a.extractValuesFromToken(tokenData.AccessToken, []string{"sub", "user_id", "tenant_id"})
	if err != nil {
		return dto.LoginResponse{}, err
	}

	var userId, tenantId, authProviderUserId string
	if claims != nil {
		if val, ok := claims["sub"].(string); ok {
			authProviderUserId = val
		}
		if val, ok := claims["user_id"].(string); ok {
			userId = val
		}
		if val, ok := claims["tenant_id"].(string); ok {
			tenantId = val
		}
	}
	fmt.Println("userId, tenantId, authProviderUserId--->", userId, tenantId, authProviderUserId)

	user, err := a.userManagementService.GetUserByID(ctx, appConstant.MasterDatabase, userId)
	if err != nil {
		fmt.Println(err)
		return dto.LoginResponse{}, err
	}

	ok := a.otpProviderService.Verify(user.Email, req.OTP)
	if !ok {
		return dto.LoginResponse{}, app_errors.InvalidOTP
	}

	userData, err := a.addUserWithTenant(ctx, user.ID.String(), user, tenantId)
	if err != nil {
		fmt.Println(err)
		return dto.LoginResponse{}, err
	}
	fmt.Println("userData--->", userData)

	err = a.authProviderService.SetEmailVerified(ctx, authProviderUserId)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	tokenResponse := dto.TokenResponse{
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
	}

	loginResponse := dto.LoginResponse{
		User:  &userData,
		Token: &tokenResponse,
	}

	return loginResponse, nil
}

func (a *authManagementService) ResendOTP(ctx context.Context, req dto.ResendOTPRequest) error {
	refreshToken := req.Token
	tokenData, err := a.authProviderService.RefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	claims, err := a.extractValuesFromToken(tokenData.AccessToken, []string{"user_id"})
	if err != nil {
		return err
	}

	var userId string
	if claims != nil {
		if val, ok := claims["user_id"].(string); ok {
			userId = val
		}
	}

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
	ok, providerUserID, attributes, err := a.authProviderService.CheckUserExistsByEmailAndReturnUser(ctx, req.Email)
	if err != nil {
		return err
	}

	if !ok {
		return app_errors.UserNotFound
	}

	tenantId, ok := attributes["tenant_id"]
	if !ok || tenantId == "" {
		return app_errors.UserNotFound
	}

	tenant, err := a.tenantManagementService.GetTenant(ctx, tenantId)
	if err != nil {
		return app_errors.TenantNotFound
	}

	fmt.Println("tenant", tenant)

	user, err := a.userManagementService.GetUserByEmail(ctx, tenant.Schema, req.Email)
	if err != nil {
		return app_errors.UserNotFound
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

	token, err := helpers.GenerateCustomJWT(tokenAttrs, providerUserID, 3600) // 1 hour expiry
	if err != nil {
		return err
	}

	dataToInsert := dto.UserResetTokenInsertion{
		ID:     uuid.NewString(),
		UserID: user.ID.String(),
		Token:  token,
		Expiry: time.Now().Add(1 * time.Hour),
	}
	data, err := a.userResetTokenService.CreateUserResetToken(ctx, dataToInsert)
	if err != nil {
		return app_errors.UserNotFound
	}

	resetURLTemplate := config.AppConfig.Auth.ResetPasswordURL
	if resetURLTemplate == "" {
		resetURLTemplate = "http://localhost:5050/reset-password?token=%s"
	}
	resetLink := fmt.Sprintf(resetURLTemplate, data.Token)
	emailData := a.emailTemplateService.PasswordResetBody(resetLink)
	subject := emailData.Subject
	body := emailData.Body

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: subject, Body: body})

	return nil
}

func (a *authManagementService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	userResetToken, err := a.userResetTokenService.GetUserResetToken(ctx, req.Token)
	if err != nil {
		return app_errors.TokenInvalid
	}

	claims, err := helpers.DecodeJWT(userResetToken.Token)
	if err != nil {
		return app_errors.TokenInvalid
	}

	tenantID, ok := claims["tenant_id"].(string)
	if !ok || tenantID == "" {
		return app_errors.InvalidCredentials
	}

	authProviderUserID, ok := claims["sub"].(string)
	if !ok || authProviderUserID == "" {
		return app_errors.InvalidCredentials
	}

	if time.Now().After(userResetToken.Expiry) {
		return app_errors.TokenExpired
	}

	userId := userResetToken.UserID.String()
	hashedPassword, err := helpers.HashPassword(req.NewPassword)
	if err != nil {
		return app_errors.ErrHashed
	}

	tenantData, err := a.tenantManagementService.GetTenant(ctx, tenantID)
	if err != nil {
		return app_errors.TenantNotFound
	}

	updateFields := map[string]interface{}{
		"password":            hashedPassword,
		"password_changed_at": time.Now(),
		"last_modified_time":  time.Now(),
	}

	updatedUser, err := a.userManagementService.UpdateUser(ctx, tenantData.Schema, userId, updateFields)
	if err != nil {
		return err
	}

	err = a.authProviderService.ResetPassword(ctx, updatedUser.Email, req.NewPassword)
	if err != nil {
		fmt.Println("err: ", err)
		return err
	}

	if !updatedUser.EmailVerified {
		updateFields := map[string]interface{}{
			"email_verified": true,
			"status":         "active",
		}

		_, err := a.userManagementService.UpdateUser(ctx, tenantData.Schema, userId, updateFields)
		if err != nil {
			return err
		}

		err = a.authProviderService.SetEmailVerified(ctx, authProviderUserID)
		if err != nil {
			return err
		}
	}

	err = a.userResetTokenService.DeleteTokensByUserId(ctx, userId)
	if err != nil {
		return err
	}

	return nil
}

func (a *authManagementService) HandleKeycloakCallback(ctx context.Context, code string) (dto.LoginResponse, error) {
	result, err := a.authProviderService.HandleCallback(ctx, code)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	user, err := a.userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, result.Email)
	if err != nil && err != app_errors.UserNotFound {
		return dto.LoginResponse{}, err
	}
	var userResponse dto.UserResponse
	if err == app_errors.UserNotFound {
		creationReq := dto.RegisterRequest{
			Email:        result.Email,
			FirstName:    result.FirstName,
			LastName:     result.LastName,
			Password:     uuid.NewString(),
			AuthProvider: result.IdentityProvider,
		}
		user, err = a.userManagementService.CreateUser(ctx, appConstant.MasterDatabase, creationReq)
		if err != nil {
			return dto.LoginResponse{}, err
		}

		tenantId := uuid.NewString()
		userResponse, err = a.addUserWithTenant(ctx, user.ID.String(), user, tenantId)
		if err != nil {
			return dto.LoginResponse{}, err
		}

	} else {
		if err := helpers.StructToStruct(user, &userResponse); err != nil {
			return dto.LoginResponse{}, app_errors.ErrMapToStruct
		}
	}

	tokenResp := dto.TokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}
	return dto.LoginResponse{
		User:  &userResponse,
		Token: &tokenResp,
	}, nil
}

func (a *authManagementService) GetAuthProviderUrl(provider string) string {
	return a.authProviderService.GetProviderURL(provider)
}

func (a *authManagementService) Logout(ctx context.Context, refreshToken string) error {
	err := a.authProviderService.Logout(ctx, refreshToken)
	if err != nil {
		return err
	}

	return nil
}

// need to shift inside tenant management in code refactor

func (a *authManagementService) AddUser(ctx context.Context, schema string, userData dto.AddUserRequest) (master.User, error) {
	roles := appConstant.RoleNames.User

	role, err := a.roleService.GetRoleByName(ctx, schema, roles)
	if err != nil {
		return master.User{}, app_errors.ErrRoleNotFound
	}

	user, tenant, err := a.userManagementService.AddUserToTenant(ctx, schema, userData, role.ID, a.userDefaultPassword.Value)
	if err != nil {
		return master.User{}, err
	}

	tokens, err := a.authProviderService.AddUser(ctx, user, tenant.ID.String(), roles)
	if err != nil {
		return master.User{}, err
	}

	claims, err := a.extractValuesFromToken(tokens.AccessToken, []string{"sub"})
	if err != nil {
		return master.User{}, err
	}

	fmt.Println("claims--->>>: ", claims)

	providerUserID, ok := claims["sub"].(string)
	if !ok || providerUserID == "" {
		return master.User{}, app_errors.InvalidCredentials
	}

	err = a.userManagementService.AddUserRole(ctx, schema, user.ID, role.ID)
	if err != nil {
		return master.User{}, err
	}

	tokenAttrs := map[string]interface{}{
		"tenant_id": tenant.ID.String(),
		"user_id":   user.ID.String(),
	}

	token, err := helpers.GenerateCustomJWT(tokenAttrs, providerUserID, 3600) // 1 hour expiry
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
		fmt.Println("err: ", err)
		return master.User{}, err
	}

	resetURLTemplate := config.AppConfig.Auth.ResetPasswordURL
	if resetURLTemplate == "" {
		resetURLTemplate = "http://localhost:5050/reset-password?token=%s"
	}
	resetLink := fmt.Sprintf(resetURLTemplate, data.Token)
	emailData := a.emailTemplateService.PlatformInvitationBody(user.FirstName, tenant.Name, resetLink)
	subject := emailData.Subject
	body := emailData.Body

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: subject, Body: body})

	return user, nil
}

func (a *authManagementService) RemoveUser(ctx context.Context, schema string, userID string) error {
	user, err := a.userManagementService.GetUserByID(ctx, schema, userID)
	if err != nil {
		fmt.Println("GetUserByID: ", err)
		return app_errors.UserNotFound
	}

	ok, providerUserID, _, err := a.authProviderService.CheckUserExistsByEmailAndReturnUser(ctx, user.Email)
	if !ok {
		fmt.Println("CheckUserExistsByEmailAndReturnUser: ", err, user.Email)
		return app_errors.UserNotFound
	}
	if err != nil {
		return err
	}

	updateData := map[string]interface{}{
		"is_deleted": true,
		"deleted_at": time.Now(),
	}

	_, err = a.userManagementService.UpdateUser(ctx, schema, userID, updateData)
	if err != nil {
		return err
	}

	err = a.authProviderService.DisableUser(ctx, providerUserID)
	if err != nil {
		return app_errors.ErrUserDisableFailed
	}

	return nil
}
func (a *authManagementService) ActivateUser(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
	user, err := a.userManagementService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	ok, keycloakUserID, _, err := a.authProviderService.CheckUserExistsByEmailAndReturnUser(ctx, user.Email)
	if err != nil {
		return dto.UserResponse{}, err
	}
	if !ok {
		return dto.UserResponse{}, app_errors.UserNotFound
	}

	err = a.authProviderService.EnableUser(ctx, keycloakUserID)
	if err != nil {
		return dto.UserResponse{}, err
	}

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
	user, err := a.userManagementService.GetUserByID(ctx, schema, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	ok, keycloakUserID, _, err := a.authProviderService.CheckUserExistsByEmailAndReturnUser(ctx, user.Email)
	if err != nil {
		return dto.UserResponse{}, err
	}
	if !ok {
		return dto.UserResponse{}, app_errors.UserNotFound
	}

	// Then, disable the user in Keycloak
	err = a.authProviderService.DisableUser(ctx, keycloakUserID)
	if err != nil {
		return dto.UserResponse{}, err
	}

	// Update the user status in the database
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

// need to shift inside user management in code refactor
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
		fmt.Println("err InviteMember: ", err)
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
	subject := emailData.Subject
	body := emailData.Body

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: subject, Body: body})

	return nil
}

func (a *authManagementService) RemoveUserFromWorkspace(ctx context.Context, schema string, req dto.RemoveMemberRequest) error {
	err := a.workspaceManagementService.RemoveUserFromWorkspace(ctx, schema, req.WorkspaceID, req.UserID)
	if err != nil {
		fmt.Println("RemoveUserFromWorkspace: ", err)
		return err
	}

	user, userErr := a.userManagementService.GetUserByID(ctx, schema, req.UserID)
	if userErr != nil {
		fmt.Println("GetUserByID: ", userErr)
		return nil
	}

	workspace, workspaceErr := a.workspaceManagementService.GetByID(ctx, schema, req.WorkspaceID)
	workspaceLabel := req.WorkspaceID
	if workspaceErr == nil {
		workspaceLabel = workspace.Title
	}

	emailData := a.emailTemplateService.RemovedFromWorkspaceBody(workspaceLabel)
	subject := emailData.Subject
	body := emailData.Body

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: subject, Body: body})

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
		fmt.Println("err InviteMember: ", err)
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
	subject := emailData.Subject
	body := emailData.Body

	a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: subject, Body: body})

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

	fmt.Println("get userIDs------------", userIDs)

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
	fmt.Println("members...")
	members, err := a.workspaceManagementService.GetWorkspaceBaseMembers(ctx, schema, baseID)
	if err != nil {
		if err == app_errors.WorkspaceMemberNotFound {
			return []dto.WorkspaceMemberResponse{}, nil
		}
		return nil, err
	}
	fmt.Println("members...", members)

	userIDs := make([]string, 0, len(members))
	userAccess := map[string]string{}
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
		userAccess[m.UserID] = m.AccessLevel
	}
	fmt.Println("userIDs...", userIDs)

	users, err := a.userManagementService.GetBulkUsers(ctx, schema, userIDs)
	if err != nil {
		return nil, err
	}
	fmt.Println("users...", users)

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
	user, err := a.userManagementService.GetUserByID(ctx, schema, userID)
	if err != nil {
		fmt.Println("DeleteUserCompletely - GetUserByID: ", err)
		return app_errors.UserNotFound
	}

	ok, providerUserID, _, err := a.authProviderService.CheckUserExistsByEmailAndReturnUser(ctx, user.Email)
	if err != nil {
		fmt.Println("DeleteUserCompletely - CheckUserExistsByEmailAndReturnUser: ", err, user.Email)
		return err
	}
	// If user doesn't exist in provider, proceed with local deletion
	if !ok {
		fmt.Println("DeleteUserCompletely - User does not exist in provider: ", user.Email)
	}

	deleteUserErr := a.userManagementService.DeleteUserCompletely(ctx, schema, userID)
	if deleteUserErr != nil {
		fmt.Println("DeleteUserCompletely - DeleteUserCompletely DB: ", deleteUserErr)
		return deleteUserErr
	}

	if ok {
		deleteProviderErr := a.authProviderService.DeleteUser(ctx, providerUserID)
		if deleteProviderErr != nil {
			fmt.Println("DeleteUserCompletely - DeleteUser from AuthProvider: ", deleteProviderErr)
			return fmt.Errorf("failed to delete user in auth provider: %w", deleteProviderErr)
		}
	}

	removeMappingErr := a.workspaceManagementService.DeleteUserMappings(ctx, schema, userID)
	if removeMappingErr != nil {
		if removeMappingErr == app_errors.WorkspaceMemberNotFound {
			return nil
		}
		fmt.Println("DeleteUserCompletely - DeleteUserMappings: ", removeMappingErr)
		return removeMappingErr
	}

	return nil
}

// AddMultipleMembers adds multiple users to a workspace at once
func (a *authManagementService) AddMultipleMembers(ctx context.Context, schema string, req dto.AddMultipleMembersRequest) (dto.AddMultipleMembersResponse, error) {
	response := dto.AddMultipleMembersResponse{
		Successes: []dto.MemberAddSuccess{},
		Failures:  []dto.MemberAddFailure{},
	}

	// Get workspace details once for email notifications
	workspace, workspaceErr := a.workspaceManagementService.GetByID(ctx, schema, req.WorkspaceID)
	if workspaceErr != nil {
		return response, workspaceErr
	}

	// Process each user
	for _, userID := range req.UserIDs {
		// Create individual member request
		memberReq := dto.CreateMemberRequest{
			WorkspaceID: req.WorkspaceID,
			UserID:      userID,
			AccessLevel: req.AccessLevel,
			BasesIds:    req.BasesIds,
		}

		// Check if user is already a member
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
			// User already exists - update their base access instead of failing
			err = a.workspaceManagementService.UpdateWorkspaceMemberBases(ctx, schema, req.WorkspaceID, userID, req.AccessLevel, req.BasesIds)
			if err != nil {
				response.Failures = append(response.Failures, dto.MemberAddFailure{
					UserID: userID,
					Error:  fmt.Sprintf("Failed to update base access: %s", err.Error()),
				})
				response.FailureCount++
				continue
			}

			// Get user details for email
			user, userErr := a.userManagementService.GetUserByID(ctx, schema, userID)
			if userErr == nil {
				// Send email notification about updated access
				emailData := a.emailTemplateService.WorkspaceAccessUpdatedBody(workspace.Title, req.AccessLevel)
				a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})
			}

			// Record success
			response.Successes = append(response.Successes, dto.MemberAddSuccess{
				UserID: userID,
			})
			response.SuccessCount++
			continue
		}

		// Add user to workspace
		err = a.workspaceManagementService.AssignUserToWorkspace(ctx, schema, memberReq)
		if err != nil {
			response.Failures = append(response.Failures, dto.MemberAddFailure{
				UserID: userID,
				Error:  err.Error(),
			})
			response.FailureCount++
			continue
		}

		// Get user details for email
		user, userErr := a.userManagementService.GetUserByID(ctx, schema, userID)
		if userErr == nil {
			// Send email notification
			emailData := a.emailTemplateService.AddedToWorkspaceBody(workspace.Title, req.AccessLevel)
			a.emailProviderService.Enqueue(emailProvider.EmailJob{To: user.Email, Subject: emailData.Subject, Body: emailData.Body})
		}

		// Record success
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

	err = a.authProviderService.ResetPassword(ctx, user.Email, updateData.NewPassword)
	if err != nil {
		fmt.Println("UpdatePassword - ResetPassword in provider:", err)
		return err
	}

	return nil
}
