package auth_test

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"testing"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	appConfig "github.com/aptlogica/sereni-base/internal/config"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	authProviderInterface "github.com/aptlogica/sereni-base/internal/providers/auth"
	emailProvider "github.com/aptlogica/sereni-base/internal/providers/email"
	services "github.com/aptlogica/sereni-base/internal/services/auth"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupAuthManagementService() (
	interfaces.AuthManagementService,
	*userManagementServiceMock,
	*workspaceManagementServiceMock,
	*userResetTokenServiceMock,
	*rbacManagementServiceMock,
	*otpServiceMock,
	*emailTemplateServiceMock,
	*emailServiceMock,
	*authProviderMock,
	*MockTableService,
) {
	if appConfig.AppConfig == nil {
		appConfig.AppConfig = &appConfig.Config{Auth: appConfig.AuthConfig{ResetPasswordURL: ""}}
	}

	userMgmt := &userManagementServiceMock{}
	workspaceMgmt := &workspaceManagementServiceMock{}
	resetTokenSvc := &userResetTokenServiceMock{}
	rbacSvc := &rbacManagementServiceMock{}
	otpSvc := &otpServiceMock{}
	emailTpl := &emailTemplateServiceMock{}
	emailSvc := &emailServiceMock{}
	authProv := &authProviderMock{}
	tableSvc := &MockTableService{}

	repo := &pkg.DatabaseService{TableService: tableSvc}

	service := services.NewAuthManagementService(
		appConfig.TemporaryAddedUserPasswordConfig{Value: "TempPass123!"},
		repo,
		services.AuthManagementServiceDeps{
			UserManagementService:      userMgmt,
			WorkspaceManagementService: workspaceMgmt,
			UserResetTokenService:      resetTokenSvc,
			RBACManagementService:      rbacSvc,
		},
		services.AuthManagementProviderDeps{
			OTPProviderService:   otpSvc,
			EmailTemplateService: emailTpl,
			EmailProviderService: emailSvc,
			AuthProviderService:  authProv,
		},
	)

	return service, userMgmt, workspaceMgmt, resetTokenSvc, rbacSvc, otpSvc, emailTpl, emailSvc, authProv, tableSvc
}

// TestNewAuthManagementService tests the constructor with nil dependencies
// This provides basic coverage for the constructor
func TestNewAuthManagementService(t *testing.T) {
	t.Parallel()
	mockDB := &pkg.DatabaseService{}
	userDefaultPassword := appConfig.TemporaryAddedUserPasswordConfig{}

	serviceDeps := services.AuthManagementServiceDeps{
		UserManagementService:      nil,
		WorkspaceManagementService: nil,
		UserResetTokenService:      nil,
		RBACManagementService:      nil,
	}

	providerDeps := services.AuthManagementProviderDeps{
		OTPProviderService:   nil,
		EmailTemplateService: nil,
		EmailProviderService: nil,
		AuthProviderService:  nil,
	}

	service := services.NewAuthManagementService(
		userDefaultPassword,
		mockDB,
		serviceDeps,
		providerDeps,
	)

	assert.NotNil(t, service)
}

func TestAuthManagement_RegisterOwner_Success(t *testing.T) {
	t.Parallel()
	service, userMgmt, workspaceMgmt, _, rbacSvc, _, _, _, authProv, tableSvc := setupAuthManagementService()

	ctx := context.Background()
	userID := uuid.New()
	workspaceID := uuid.New()
	user := tenant.User{ID: userID, Email: "owner@example.com", FirstName: "Owner", LastName: "One"}

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return user, nil
	}
	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		updated := user
		updated.Status = "active"
		updated.EmailVerified = true
		return updated, nil
	}
	workspaceMgmt.CreateFn = func(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string, userId string) (dto.WorkspaceResponse, error) {
		return dto.WorkspaceResponse{ID: workspaceID, Title: "Default Workspace"}, nil
	}
	rbacSvc.GetRoleByNameFn = func(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: uuid.New(), Name: appConstant.RBACRoleNames.Owner}, nil
	}
	rbacSvc.AssignRoleToUserFn = func(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
		return nil, nil
	}
	// Mock JWT service registration - should be called
	authProv.RegisterFn = func(ctx context.Context, userId string, email string, password string, roles []string) error {
		assert.Equal(t, "owner@example.com", email)
		assert.Equal(t, "plain", password)
		assert.Contains(t, roles, appConstant.RBACRoleNames.Owner)
		return nil
	}
	// Mock workspace_members table creation
	tableSvc.On("CreateRecord", mock.Anything, mock.Anything).Return(map[string]interface{}{}, nil)

	resp, err := service.RegisterOwner(ctx, dto.RegisterRequest{
		Email:     "owner@example.com",
		Password:  "plain",
		FirstName: "Owner",
		LastName:  "One",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp.User)
	assert.Equal(t, "owner@example.com", resp.User.Email)
}

func TestAuthManagement_RegisterOwner_UserExists(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return tenant.User{}, app_errors.UserAlreadyExists
	}

	_, err := service.RegisterOwner(ctx, dto.RegisterRequest{Email: "exists@example.com", Password: "pass123"})
	assert.ErrorIs(t, err, app_errors.UserAlreadyExists)
}

func TestAuthManagement_Login_EmailNotVerified(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	hashed, _ := helpers.HashPassword("pass123")
	user := tenant.User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Password:      hashed,
		Status:        "active",
		EmailVerified: false,
	}

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return user, nil
	}
	authProv.LoginFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceLoginRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{AccessToken: "a", RefreshToken: "r"}, nil
	}

	resp, err := service.Login(ctx, "test@example.com", "pass123")
	assert.NoError(t, err)
	assert.NotNil(t, resp.User)
	assert.Equal(t, user.ID, resp.User.ID)
	assert.NotNil(t, resp.Token)
	assert.Equal(t, "r", resp.Token.RefreshToken)
	assert.Equal(t, "", resp.Token.AccessToken)
}

func TestAuthManagement_Login_Verified(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	hashed, _ := helpers.HashPassword("pass123")
	user := tenant.User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Password:      hashed,
		Status:        "active",
		EmailVerified: true,
	}

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return user, nil
	}
	// Mock JWT service login - should be called
	authProv.LoginFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceLoginRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{AccessToken: "a", RefreshToken: "r"}, nil
	}

	resp, err := service.Login(ctx, "test@example.com", "pass123")
	assert.NoError(t, err)
	assert.Equal(t, "a", resp.Token.AccessToken)
	assert.Equal(t, "r", resp.Token.RefreshToken)
}

func TestAuthManagement_Login_WithJWTServiceSync(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	hashed, _ := helpers.HashPassword("pass123")
	user := tenant.User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Password:      hashed,
		Status:        "active",
		EmailVerified: true,
	}

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return user, nil
	}

	loginCallCount := 0
	authProv.LoginFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceLoginRequest) (authProviderInterface.Tokens, error) {
		loginCallCount++
		return authProviderInterface.Tokens{AccessToken: "a", RefreshToken: "r"}, nil
	}

	resp, err := service.Login(ctx, "test@example.com", "pass123")
	assert.NoError(t, err)
	assert.Equal(t, "a", resp.Token.AccessToken)
	assert.Equal(t, "r", resp.Token.RefreshToken)
	assert.Equal(t, 1, loginCallCount, "Login should be called once")
}

func TestAuthManagement_Login_InvalidPassword(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	hashed, _ := helpers.HashPassword("pass123")
	user := tenant.User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		Password:      hashed,
		Status:        "active",
		EmailVerified: true,
	}

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return user, nil
	}

	_, err := service.Login(ctx, "test@example.com", "wrong")
	assert.ErrorIs(t, err, app_errors.InvalidCredentials)
}

func TestAuthManagement_VerifyEmail_InvalidOTP(t *testing.T) {
	service, userMgmt, _, _, _, otpSvc, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	user := tenant.User{ID: uuid.New(), Email: "user@example.com"}
	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{RefreshToken: "refresh"}, nil
	}
	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: user.ID.String()}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return user, nil
	}
	otpSvc.VerifyFn = func(identifier, input string) bool {
		return false
	}

	_, err := service.VerifyEmail(ctx, dto.VerifyEmailRequest{Token: "t", OTP: "0000"})
	assert.ErrorIs(t, err, app_errors.InvalidOTP)
}

func TestAuthManagement_VerifyEmail_Success(t *testing.T) {
	service, userMgmt, _, _, _, otpSvc, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	user := tenant.User{ID: uuid.New(), Email: "user@example.com"}
	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{RefreshToken: "refresh"}, nil
	}
	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: user.ID.String()}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return user, nil
	}
	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		updated := user
		updated.EmailVerified = true
		return updated, nil
	}
	otpSvc.VerifyFn = func(identifier, input string) bool {
		return true
	}
	authProv.LoginFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceLoginRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{AccessToken: "a", RefreshToken: "r"}, nil
	}

	resp, err := service.VerifyEmail(ctx, dto.VerifyEmailRequest{Token: "t", OTP: "1234"})
	assert.NoError(t, err)
	assert.Equal(t, "a", resp.Token.AccessToken)
}

func TestAuthManagement_ResendOTP(t *testing.T) {
	service, userMgmt, _, _, _, otpSvc, emailTpl, emailSvc, authProv, _ := setupAuthManagementService()
	ctx := context.Background()
	user := tenant.User{ID: uuid.New(), Email: "user@example.com", EmailVerified: false}

	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{RefreshToken: "refresh"}, nil
	}
	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: user.ID.String()}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return user, nil
	}
	otpSvc.GenerateFn = func(identifier string) string { return "1111" }
	emailTpl.EmailVerificationOTPBodyFn = func(otp string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "otp", Body: otp}
	}
	var enqueued emailProvider.EmailJob
	emailSvc.EnqueueFn = func(job emailProvider.EmailJob) {
		enqueued = job
	}

	err := service.ResendOTP(ctx, dto.ResendOTPRequest{Token: "t"})
	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", enqueued.To)
	assert.Equal(t, "otp", enqueued.Subject)
}

func TestAuthManagement_ResendOTP_AlreadyVerified(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()
	user := tenant.User{ID: uuid.New(), Email: "user@example.com", EmailVerified: true}

	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{RefreshToken: "refresh"}, nil
	}
	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: user.ID.String()}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return user, nil
	}

	err := service.ResendOTP(ctx, dto.ResendOTPRequest{Token: "t"})
	assert.ErrorIs(t, err, app_errors.EmailAlreadyVerified)
}

func TestAuthManagement_RefreshToken_ValidateToken(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		if reqBody.RefreshToken == "bad" {
			return authProviderInterface.Tokens{}, errors.New("bad")
		}
		return authProviderInterface.Tokens{AccessToken: "a", RefreshToken: "r"}, nil
	}
	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		if tokenStr == "bad" {
			return authProviderInterface.Claims{}, errors.New("bad")
		}
		return authProviderInterface.Claims{UserId: "u1", Roles: "admin"}, nil
	}

	_, err := service.RefreshToken(ctx, dto.RefreshTokenRequest{RefreshToken: "bad"})
	assert.Error(t, err)

	resp, err := service.RefreshToken(ctx, dto.RefreshTokenRequest{RefreshToken: "ok"})
	assert.NoError(t, err)
	assert.Equal(t, "a", resp.AccessToken)

	vr, err := service.ValidateToken(ctx, "ok")
	assert.NoError(t, err)
	assert.True(t, vr.Valid)

	vr, err = service.ValidateToken(ctx, "bad")
	assert.Error(t, err)
	assert.False(t, vr.Valid)

	vr, err = service.VerifyToken(ctx, "ok")
	assert.NoError(t, err)
	assert.True(t, vr.Valid)
}

func TestAuthManagement_ForgotPassword_Success(t *testing.T) {
	service, userMgmt, _, resetSvc, _, _, emailTpl, emailSvc, _, _ := setupAuthManagementService()
	ctx := context.Background()

	user := tenant.User{ID: uuid.New(), Email: "user@example.com", EmailVerified: true, Status: "active"}
	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return user, nil
	}
	resetSvc.CreateUserResetTokenFn = func(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{Token: req.Token}, nil
	}
	emailTpl.PasswordResetBodyFn = func(resetLink string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "reset", Body: resetLink}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.ForgotPassword(ctx, dto.ForgotPasswordRequest{Email: "user@example.com"})
	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", job.To)
	assert.Contains(t, job.Body, "token=")
}

func TestAuthManagement_ForgotPassword_Invalid(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{EmailVerified: false}, nil
	}

	err := service.ForgotPassword(ctx, dto.ForgotPasswordRequest{Email: "user@example.com"})
	assert.ErrorIs(t, err, app_errors.InvalidCredentials)
}

func TestAuthManagement_ResetPassword(t *testing.T) {
	service, userMgmt, _, resetSvc, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	token, _ := helpers.GenerateCustomJWT(map[string]interface{}{"user_id": "u1"}, "u1", 3600)
	claims, _ := helpers.DecodeJWT(token)
	issuedAt := fmt.Sprintf("%d", int64(claims["iat"].(float64)))

	resetSvc.GetUserResetTokenFn = func(ctx context.Context, token string) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Token: token, IssuedAt: issuedAt}, nil
	}
	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Email: "user@example.com"}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Email: "user@example.com"}, nil
	}
	authProv.RegisterFn = func(ctx context.Context, userId string, email, password string, roles []string) error {
		assert.Equal(t, "user@example.com", email)
		assert.Equal(t, "NewPass123!", password)
		return nil
	}

	err := service.ResetPassword(ctx, dto.ResetPasswordRequest{Token: token, NewPassword: "NewPass123!"})
	assert.NoError(t, err)
}

func TestAuthManagement_ResetPassword_InvalidToken(t *testing.T) {
	service, _, _, resetSvc, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	resetSvc.GetUserResetTokenFn = func(ctx context.Context, token string) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{}, errors.New("not found")
	}

	err := service.ResetPassword(ctx, dto.ResetPasswordRequest{Token: "bad", NewPassword: "NewPass123!"})
	assert.ErrorIs(t, err, app_errors.TokenInvalid)
}

func TestAuthManagement_ResetPassword_IssuedAtMismatch(t *testing.T) {
	service, _, _, resetSvc, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	token, _ := helpers.GenerateCustomJWT(map[string]interface{}{"user_id": "u1"}, "u1", 3600)
	resetSvc.GetUserResetTokenFn = func(ctx context.Context, token string) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Token: token, IssuedAt: "0"}, nil
	}

	err := service.ResetPassword(ctx, dto.ResetPasswordRequest{Token: token, NewPassword: "NewPass123!"})
	assert.Error(t, err)
}

func TestAuthManagement_AddUser_CoOwnerAndMember(t *testing.T) {
	service, userMgmt, _, resetSvc, rbacSvc, _, emailTpl, emailSvc, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	user := tenant.User{ID: uuid.New(), Email: "new@example.com", FirstName: "New", LastName: "User"}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return user, nil
	}
	userMgmt.AddAvatarFn = func(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error) {
		return dto.UserResponse{}, nil
	}
	resetSvc.CreateUserResetTokenFn = func(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{Token: req.Token}, nil
	}
	rbacSvc.GetRoleByNameFn = func(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: uuid.New(), Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	rbacSvc.AssignRoleToUserFn = func(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
		return nil, nil
	}
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, nil
	}
	emailTpl.PlatformInvitationBodyFn = func(firstName, tenantName, resetLink string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "invite", Body: resetLink}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email:      "new@example.com",
		FirstName:  "New",
		LastName:   "User",
		ProfilePic: &multipart.FileHeader{Filename: "avatar.png"},
		IsCoOwner:  true,
	}, "admin-id")
	assert.NoError(t, err)
	assert.Equal(t, "new@example.com", job.To)
}

func TestAuthManagement_AddUser_MembershipFlow(t *testing.T) {
	service, userMgmt, _, resetSvc, rbacSvc, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	user := tenant.User{ID: uuid.New(), Email: "new@example.com", FirstName: "New", LastName: "User"}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return user, nil
	}
	resetSvc.CreateUserResetTokenFn = func(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{Token: req.Token}, nil
	}
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, nil
	}

	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email:     "new@example.com",
		FirstName: "New",
		LastName:  "User",
		IsCoOwner: false,
		Membership: []dto.MembershipRequest{
			{WorkspaceID: "w1", Role: "editor"},
		},
	}, "admin-id")
	assert.NoError(t, err)
}

func TestAuthManagement_EditUser_FullFlow(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(userID), Email: "u@example.com"}, nil
	}
	userMgmt.UpdateUserProfileFn = func(ctx context.Context, schema string, userID string, updateData dto.UpdateUserProfileRequest) (dto.UserResponse, error) {
		return dto.UserResponse{ID: uuid.MustParse(userID)}, nil
	}
	userMgmt.RemoveAvatarFn = func(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
		return dto.UserResponse{}, app_errors.AssetNotFound
	}
	userMgmt.AddAvatarFn = func(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error) {
		return dto.UserResponse{}, nil
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		roleID := uuid.New().String()
		return []dto.AccessMemberDTO{
			{ID: uuid.New(), RoleID: roleID, ScopeType: "workspace", ScopeID: helpers.StringPtr("w1")},
		}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	rbacSvc.GetRoleByNameFn = func(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: uuid.New(), Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	rbacSvc.AssignRoleToUserFn = func(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
		return nil, nil
	}
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)

	resp, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:     userID,
		FirstName:  helpers.StringPtr("New"),
		ProfilePic: &multipart.FileHeader{Filename: "avatar.png"},
		IsCoOwner:  helpers.BoolPtr(true),
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "editor"}},
	}, "admin-id")
	assert.NoError(t, err)
	assert.Equal(t, uuid.MustParse(userID), resp.ID)
}

func TestAuthManagement_RemoveUser_Activate_Deactivate(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Email: "u@example.com"}, nil
	}

	// Mock GetByFunction for RemoveUser (checks for Owner and Co-Owner)
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.NoAccess,
			},
		},
	}, nil)

	err := service.RemoveUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.NoError(t, err)

	_, err = service.ActivateUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.NoError(t, err)

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return nil, nil
	}
	_, err = service.DeactivateUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.NoError(t, err)
}

func TestAuthManagement_DeactivateOwnerBlocked(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	ownerRoleID := uuid.New()

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{{RoleID: ownerRoleID.String()}}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.Owner}, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.Owner,
			},
		},
	}, nil)

	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.ErrorIs(t, err, app_errors.OwnerCannotBeDeactivated)
}

func TestAuthManagement_DeactivateCoOwnerBlocked(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	coOwnerRoleID := uuid.New()

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{{RoleID: coOwnerRoleID.String()}}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)

	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.ErrorIs(t, err, app_errors.CoOwnerCannotBeDeactivated)
}

func TestAuthManagement_RemoveOwnerBlocked(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	ownerRoleID := uuid.New()

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{{RoleID: ownerRoleID.String()}}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.Owner}, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.Owner,
			},
		},
	}, nil)

	err := service.RemoveUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.ErrorIs(t, err, app_errors.OwnerCannotBeRemoved)
}

func TestAuthManagement_RemoveCoOwnerBlocked(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	coOwnerRoleID := uuid.New()

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{{RoleID: coOwnerRoleID.String()}}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)

	err := service.RemoveUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.ErrorIs(t, err, app_errors.CoOwnerCannotBeRemoved)
}

func TestAuthManagement_ActivateOwnerBlocked(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	ownerRoleID := uuid.New()

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{{RoleID: ownerRoleID.String()}}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.Owner}, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.Owner,
			},
		},
	}, nil)

	_, err := service.ActivateUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.ErrorIs(t, err, app_errors.OwnerCannotBeDeactivated)
}

func TestAuthManagement_ActivateCoOwnerBlocked(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	coOwnerRoleID := uuid.New()

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{{RoleID: coOwnerRoleID.String()}}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)

	_, err := service.ActivateUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.ErrorIs(t, err, app_errors.CoOwnerCannotBeDeactivated)
}

func TestAuthManagement_ActivateCoOwnerAllowedForOwner(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	targetUserID := uuid.New().String()
	ownerReqBy := uuid.New().String()

	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Email: "u@example.com"}, nil
	}

	tableSvc.On("GetByFunction", mock.Anything, "public.get_user_role_by_id", map[string]interface{}{"p_user_id": targetUserID}).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)
	tableSvc.On("GetByFunction", mock.Anything, "public.get_user_role_by_id", map[string]interface{}{"p_user_id": ownerReqBy}).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.Owner,
			},
		},
	}, nil)

	_, err := service.ActivateUser(ctx, appConstant.MasterDatabase, targetUserID, ownerReqBy)
	assert.NoError(t, err)
}

func TestAuthManagement_DeactivateCoOwnerAllowedForOwner(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	targetUserID := uuid.New().String()
	ownerReqBy := uuid.New().String()

	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Email: "u@example.com"}, nil
	}

	tableSvc.On("GetByFunction", mock.Anything, "public.get_user_role_by_id", map[string]interface{}{"p_user_id": targetUserID}).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)
	tableSvc.On("GetByFunction", mock.Anything, "public.get_user_role_by_id", map[string]interface{}{"p_user_id": ownerReqBy}).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.Owner,
			},
		},
	}, nil)

	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, targetUserID, ownerReqBy)
	assert.NoError(t, err)
}

func TestAuthManagement_RemoveCoOwnerAllowedForOwner(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	targetUserID := uuid.New().String()
	ownerReqBy := uuid.New().String()

	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Email: "u@example.com"}, nil
	}

	tableSvc.On("GetByFunction", mock.Anything, "public.get_user_role_by_id", map[string]interface{}{"p_user_id": targetUserID}).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)
	tableSvc.On("GetByFunction", mock.Anything, "public.get_user_role_by_id", map[string]interface{}{"p_user_id": ownerReqBy}).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.Owner,
			},
		},
	}, nil)

	err := service.RemoveUser(ctx, appConstant.MasterDatabase, targetUserID, ownerReqBy)
	assert.NoError(t, err)
}

func TestAuthManagement_DeleteUserCompletely_CoOwnerAllowedForOwner(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	targetUserID := uuid.New().String()
	ownerReqBy := uuid.New().String()

	tableSvc.On("GetByFunction", mock.Anything, "public.get_user_role_by_id", map[string]interface{}{"p_user_id": targetUserID}).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)
	tableSvc.On("GetByFunction", mock.Anything, "public.get_user_role_by_id", map[string]interface{}{"p_user_id": ownerReqBy}).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.Owner,
			},
		},
	}, nil)

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Status: "pending"}, nil
	}
	userMgmt.DeleteUserCompletelyFn = func(ctx context.Context, schema string, userID string) error {
		return nil
	}
	workspaceMgmt.DeleteUserMappingsFn = func(ctx context.Context, schemaName string, userID string) error {
		return nil
	}

	err := service.DeleteUserCompletely(ctx, appConstant.MasterDatabase, targetUserID, ownerReqBy)
	assert.NoError(t, err)
}

func TestAuthManagement_RemoveUserFromWorkspace_Fallback(t *testing.T) {
	service, _, workspaceMgmt, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return(nil, errors.New("no table"))
	workspaceMgmt.RemoveUserFromWorkspaceFn = func(ctx context.Context, schemaName string, workspaceID string, userID string) error {
		return nil
	}

	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.NoError(t, err)

	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	emailSvc.EnqueueFn = func(job emailProvider.EmailJob) {}
}

func TestAuthManagement_RemoveUserFromWorkspace_Success(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	workspaceMgmt.GetByIDFn = func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
		return tenant.Workspace{Title: "Workspace"}, nil
	}
	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "u@example.com", job.To)
}

func TestAuthManagement_RemoveUserFromWorkspace_TypeMismatch(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": 123},
	}, nil)

	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserFromBase_Flows(t *testing.T) {
	service, userMgmt, _, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil).Once()
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil).Once()
	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"title": "Base One"},
	}, nil).Once()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "u@example.com", job.To)
}

func TestAuthManagement_RemoveUserFromBase_Empty(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
}

func TestAuthManagement_GetMembersAndDeleteUserCompletely(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	workspaceMgmt.GetWorkspaceMembersFn = func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error) {
		return []tenant.WorkspaceMember{{UserID: "u1", AccessLevel: "editor"}}, nil
	}
	userMgmt.GetBulkUsersFn = func(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
		return []tenant.User{{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Email: "u1@example.com"}}, nil
	}
	_, err := service.GetWorkspaceMembers(ctx, appConstant.MasterDatabase, "w1")
	assert.NoError(t, err)

	workspaceMgmt.GetWorkspaceBaseMembersFn = func(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error) {
		return []tenant.WorkspaceMember{{UserID: "u1", AccessLevel: "viewer"}}, nil
	}
	_, err = service.GetBaseMembers(ctx, appConstant.MasterDatabase, "b1")
	assert.NoError(t, err)

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Status: "pending"}, nil
	}
	userMgmt.DeleteUserCompletelyFn = func(ctx context.Context, schema string, userID string) error { return nil }
	workspaceMgmt.DeleteUserMappingsFn = func(ctx context.Context, schemaName string, userID string) error {
		return app_errors.WorkspaceMemberNotFound
	}
	err = service.DeleteUserCompletely(ctx, appConstant.MasterDatabase, "00000000-0000-0000-0000-000000000001", "admin-id")
	assert.NoError(t, err)
}

func TestAuthManagement_GetUsersAndAssign(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUsersWithRoleFn = func(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
		return []dto.UserWithRole{{ID: uuid.New()}}, nil
	}
	userMgmt.GetActiveUsersForAssignFn = func(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
		return []dto.UserWithRole{{ID: uuid.New()}}, nil
	}
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, nil
	}

	_, err := service.GetUsers(ctx, appConstant.MasterDatabase)
	assert.NoError(t, err)
	_, err = service.GetActiveUsersForAssign(ctx, appConstant.MasterDatabase)
	assert.NoError(t, err)

	err = service.AssignUserToWorkspace(ctx, appConstant.MasterDatabase, dto.CreateMemberRequest{
		UserID:     "u1",
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "viewer"}},
	}, "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_FunctionQueriesAndRemoveAccess(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"get_workspace_members_with_role": map[string]interface{}{"id": "u1", "email": "u@example.com"}},
	}, nil)
	_, err := service.GetWorkspaceMembersWithRole(ctx, appConstant.MasterDatabase, "w1")
	assert.NoError(t, err)

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"get_base_members_with_role": map[string]interface{}{"id": "u2", "email": "u2@example.com"}},
	}, nil)
	_, err = service.GetBaseMembersWithRole(ctx, appConstant.MasterDatabase, "b1")
	assert.NoError(t, err)

	tableSvc.On("DeleteRecord", mock.Anything, "a1").Return(nil)
	err = service.RemoveAccessMemberByID(ctx, appConstant.MasterDatabase, "a1", "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_BulkAddsAndUpdatePassword(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		if userID == "fail" {
			return nil, errors.New("fail")
		}
		return nil, nil
	}
	_, err := service.BulkAddMembers(ctx, appConstant.MasterDatabase, dto.BulkAddMembersRequest{
		Members: []dto.BulkMemberRequest{{UserID: "ok"}, {UserID: "fail"}},
	}, "admin")
	assert.NoError(t, err)

	_, err = service.BulkAddBaseMembers(ctx, appConstant.MasterDatabase, "b1", dto.BulkAddBaseMembersRequest{
		Members: []dto.BulkMemberRequest{{UserID: "ok"}, {UserID: "fail"}},
	}, "admin")
	assert.NoError(t, err)

	userMgmt.UpdatePasswordFn = func(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(userID), Email: "user@example.com"}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Email: "user@example.com"}, nil
	}
	authProv.RegisterFn = func(ctx context.Context, userId string, email, password string, roles []string) error {
		assert.Equal(t, "user@example.com", email)
		assert.Equal(t, "new", password)
		return nil
	}
	err = service.UpdatePassword(ctx, appConstant.MasterDatabase, "00000000-0000-0000-0000-000000000001", dto.UpdateUserPasswordRequest{
		OldPassword: "old",
		NewPassword: "new",
	})
	assert.NoError(t, err)
}

func TestAuthManagement_Misc(t *testing.T) {
	service, _, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	_, err := service.HandleKeycloakCallback(ctx, "code")
	assert.Error(t, err)
	assert.Equal(t, "", service.GetAuthProviderUrl("any"))
	assert.NoError(t, service.Logout(ctx, "refresh"))
}

func TestAuthManagement_RemoveUserFromBase_TypeMismatch(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": 123},
	}, nil)
	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.Error(t, err)
}

func TestAuthManagement_WorkspaceMemberNotFound(t *testing.T) {
	service, _, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	workspaceMgmt.GetWorkspaceMembersFn = func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error) {
		return nil, app_errors.WorkspaceMemberNotFound
	}
	members, err := service.GetWorkspaceMembers(ctx, appConstant.MasterDatabase, "w1")
	assert.NoError(t, err)
	assert.Len(t, members, 0)

	workspaceMgmt.GetWorkspaceBaseMembersFn = func(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error) {
		return nil, app_errors.WorkspaceMemberNotFound
	}
	members, err = service.GetBaseMembers(ctx, appConstant.MasterDatabase, "b1")
	assert.NoError(t, err)
	assert.Len(t, members, 0)
}

func TestAuthManagement_RemoveAccessMember_Error(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("DeleteRecord", mock.Anything, "bad").Return(errors.New("bad"))
	err := service.RemoveAccessMemberByID(ctx, appConstant.MasterDatabase, "bad", "admin")
	assert.Error(t, err)
}

func TestAuthManagement_GetWorkspaceBaseMembersWithRole_Error(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db"))
	_, err := service.GetWorkspaceMembersWithRole(ctx, appConstant.MasterDatabase, "w1")
	assert.Error(t, err)

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db"))
	_, err = service.GetBaseMembersWithRole(ctx, appConstant.MasterDatabase, "b1")
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserCompletely_NotPending(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Status: "active"}, nil
	}
	err := service.DeleteUserCompletely(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.ErrorIs(t, err, app_errors.OnlyPendingUsersCanBeDeleted)
}

func TestAuthManagement_RemoveUserFromWorkspace_EmptyRecords(t *testing.T) {
	service, _, workspaceMgmt, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
	workspaceMgmt.RemoveUserFromWorkspaceFn = func(ctx context.Context, schemaName string, workspaceID string, userID string) error {
		return nil
	}
	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_RemoveUserFromBase_TableError(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return(nil, errors.New("db"))
	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.Error(t, err)
}

func TestAuthManagement_ResetPassword_InvalidJWT(t *testing.T) {
	service, _, _, resetSvc, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	resetSvc.GetUserResetTokenFn = func(ctx context.Context, token string) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Token: "invalid", IssuedAt: "0"}, nil
	}

	err := service.ResetPassword(ctx, dto.ResetPasswordRequest{Token: "invalid", NewPassword: "NewPass123!"})
	assert.ErrorIs(t, err, app_errors.TokenInvalid)
}

func TestAuthManagement_UpdateUserPassword_Error(t *testing.T) {
	service, userMgmt, _, resetSvc, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	token, _ := helpers.GenerateCustomJWT(map[string]interface{}{"user_id": "u1"}, "u1", 3600)
	claims, _ := helpers.DecodeJWT(token)
	issuedAt := fmt.Sprintf("%d", int64(claims["iat"].(float64)))

	resetSvc.GetUserResetTokenFn = func(ctx context.Context, token string) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Token: token, IssuedAt: issuedAt}, nil
	}
	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		return tenant.User{}, errors.New("update failed")
	}

	err := service.ResetPassword(ctx, dto.ResetPasswordRequest{Token: token, NewPassword: "NewPass123!"})
	assert.Error(t, err)
}

func TestAuthManagement_Login_UserNotActive(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{Status: "pending"}, nil
	}

	_, err := service.Login(ctx, "user@example.com", "pass")
	assert.ErrorIs(t, err, app_errors.UserNotActive)
}

func TestAuthManagement_RemoveUserFromWorkspace_DeleteError(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(errors.New("delete failed"))
	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.Error(t, err)
}

func TestAuthManagement_AssignUserToWorkspace_Error(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, errors.New("fail")
	}
	err := service.AssignUserToWorkspace(ctx, appConstant.MasterDatabase, dto.CreateMemberRequest{
		UserID:     "u1",
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "viewer"}},
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_GetUsersErrors(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUsersWithRoleFn = func(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
		return nil, errors.New("db")
	}
	_, err := service.GetUsers(ctx, appConstant.MasterDatabase)
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserFromBase_UserNotFound(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil).Once()
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil).Once()
	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"title": "Base One"},
	}, nil).Once()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{}, errors.New("not found")
	}

	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_RefreshTokenRequest(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{AccessToken: "a", RefreshToken: "r"}, nil
	}

	resp, err := service.RefreshToken(ctx, dto.RefreshTokenRequest{RefreshToken: "r"})
	assert.NoError(t, err)
	assert.Equal(t, "a", resp.AccessToken)
}

func TestAuthManagement_ValidateToken_Error(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{}, errors.New("bad")
	}

	resp, err := service.ValidateToken(ctx, "bad")
	assert.Error(t, err)
	assert.False(t, resp.Valid)
}

func TestAuthManagement_GetUsersWithRole_EmptyRecord(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"get_workspace_members_with_role": "not-map"},
	}, nil)
	_, err := service.GetWorkspaceMembersWithRole(ctx, appConstant.MasterDatabase, "w1")
	assert.NoError(t, err)
}

func TestAuthManagement_GetBaseMembersWithRole_EmptyRecord(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"get_base_members_with_role": "not-map"},
	}, nil)
	_, err := service.GetBaseMembersWithRole(ctx, appConstant.MasterDatabase, "b1")
	assert.NoError(t, err)
}

func TestAuthManagement_RemoveUserFromBase_DeleteError(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(errors.New("delete failed"))

	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserFromWorkspace_UserNotFound(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{}, errors.New("not found")
	}
	workspaceMgmt.RemoveUserFromWorkspaceFn = func(ctx context.Context, schemaName string, workspaceID string, userID string) error {
		return nil
	}

	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_RemoveUserFromWorkspace_WorkspaceNotFound(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	workspaceMgmt.GetByIDFn = func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
		return tenant.Workspace{}, errors.New("not found")
	}
	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "u@example.com", job.To)
}

func TestAuthManagement_RemoveUserCompletely_DeleteError(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Status: "pending"}, nil
	}
	userMgmt.DeleteUserCompletelyFn = func(ctx context.Context, schema string, userID string) error {
		return errors.New("delete failed")
	}
	err := service.DeleteUserCompletely(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserCompletely_MappingError(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id), Status: "pending"}, nil
	}
	userMgmt.DeleteUserCompletelyFn = func(ctx context.Context, schema string, userID string) error {
		return nil
	}
	workspaceMgmt.DeleteUserMappingsFn = func(ctx context.Context, schemaName string, userID string) error {
		return errors.New("mapping failed")
	}
	err := service.DeleteUserCompletely(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.Error(t, err)
}

func TestAuthManagement_UpdateUserPassword_InvalidTokenBranch(t *testing.T) {
	service, _, _, resetSvc, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	resetSvc.GetUserResetTokenFn = func(ctx context.Context, token string) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{Token: "invalid", IssuedAt: "0"}, nil
	}

	err := service.ResetPassword(ctx, dto.ResetPasswordRequest{Token: "invalid", NewPassword: "NewPass123!"})
	assert.ErrorIs(t, err, app_errors.TokenInvalid)
}

func TestAuthManagement_GetUsersWithRole_AndGetActiveUsersForAssign_Error(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetActiveUsersForAssignFn = func(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
		return nil, errors.New("db")
	}
	_, err := service.GetActiveUsersForAssign(ctx, appConstant.MasterDatabase)
	assert.Error(t, err)
}

func TestAuthManagement_CheckIfUserIsOwner_ErrorIgnored(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db"))
	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.NoError(t, err)
}

func TestAuthManagement_RemoveUserFromBase_BaseTitleFallback(t *testing.T) {
	service, userMgmt, _, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil).Once()
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil).Once()
	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return(nil, errors.New("db")).Once()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "u@example.com", job.To)
}

func TestAuthManagement_InitializeOwner_ErrorOnUpdate(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()
	user := tenant.User{ID: uuid.New(), Email: "u@example.com"}

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return user, nil
	}
	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		return tenant.User{}, errors.New("update failed")
	}

	_, err := service.RegisterOwner(ctx, dto.RegisterRequest{Email: "u@example.com", Password: "pass"})
	assert.Error(t, err)
}

func TestAuthManagement_RefreshToken_Empty(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{}, nil
	}
	_, err := service.RefreshToken(ctx, dto.RefreshTokenRequest{RefreshToken: "r"})
	assert.NoError(t, err)
}

func TestAuthManagement_SendOtpViaEmail_CalledThroughResend(t *testing.T) {
	service, userMgmt, _, _, _, otpSvc, emailTpl, emailSvc, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	user := tenant.User{ID: uuid.New(), Email: "user@example.com", EmailVerified: false}
	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{RefreshToken: "refresh"}, nil
	}
	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: user.ID.String()}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return user, nil
	}
	otpSvc.GenerateFn = func(identifier string) string { return "2222" }
	emailTpl.EmailVerificationOTPBodyFn = func(otp string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "otp", Body: otp}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.ResendOTP(ctx, dto.ResendOTPRequest{Token: "t"})
	assert.NoError(t, err)
	assert.Equal(t, "user@example.com", job.To)
}

func TestAuthManagement_Login_UserNotFound(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	_, err := service.Login(ctx, "missing@example.com", "pass")
	assert.ErrorIs(t, err, app_errors.InvalidCredentials)
}

func TestAuthManagement_Login_ErrorFromUserLookup(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, errors.New("db")
	}
	_, err := service.Login(ctx, "err@example.com", "pass")
	assert.Error(t, err)
}

func TestAuthManagement_VerifyEmail_TokenErrors(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{}, errors.New("bad")
	}
	_, err := service.VerifyEmail(ctx, dto.VerifyEmailRequest{Token: "bad", OTP: "1234"})
	assert.Error(t, err)

	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{}, errors.New("bad")
	}
	_, err = service.VerifyEmail(ctx, dto.VerifyEmailRequest{Token: "bad", OTP: "1234"})
	assert.Error(t, err)
}

func TestAuthManagement_ResendOTP_TokenErrors(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{}, errors.New("bad")
	}
	err := service.ResendOTP(ctx, dto.ResendOTPRequest{Token: "bad"})
	assert.Error(t, err)

	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{}, errors.New("bad")
	}
	err = service.ResendOTP(ctx, dto.ResendOTPRequest{Token: "bad"})
	assert.Error(t, err)
}

func TestAuthManagement_GetWorkspaceMembersWithRole_LogError(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db"))
	_, err := service.GetWorkspaceMembersWithRole(ctx, appConstant.MasterDatabase, "w1")
	assert.Error(t, err)
}

func TestAuthManagement_GetBaseMembersWithRole_LogError(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db"))
	_, err := service.GetBaseMembersWithRole(ctx, appConstant.MasterDatabase, "b1")
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserFromWorkspace_UsesUuidID(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	id := uuid.New()
	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": id},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, id.String()).Return(nil)
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	workspaceMgmt.GetByIDFn = func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
		return tenant.Workspace{Title: "Workspace"}, nil
	}
	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) {}

	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_RemoveUserFromBase_UsesUuidID(t *testing.T) {
	service, userMgmt, _, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	id := uuid.New()
	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": id},
	}, nil).Once()
	tableSvc.On("DeleteRecord", mock.Anything, id.String()).Return(nil).Once()
	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"title": "Base One"},
	}, nil).Once()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) {}

	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_GetActiveUsersForAssign_Error(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetActiveUsersForAssignFn = func(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
		return nil, errors.New("db")
	}
	_, err := service.GetActiveUsersForAssign(ctx, appConstant.MasterDatabase)
	assert.Error(t, err)
}

func TestAuthManagement_VerifyToken_UsesValidateToken(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: "u1", Roles: "admin"}, nil
	}
	resp, err := service.VerifyToken(ctx, "token")
	assert.NoError(t, err)
	assert.True(t, resp.Valid)
}

func TestAuthManagement_RemoveUserFromBase_EmailTemplateFallback(t *testing.T) {
	service, userMgmt, _, _, _, _, _, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil).Once()
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil).Once()
	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"title": "Base One"},
	}, nil).Once()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "u@example.com", job.To)
}

func TestAuthManagement_RemoveUserFromWorkspace_EmailTemplateFallback(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	workspaceMgmt.GetByIDFn = func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
		return tenant.Workspace{Title: "Workspace"}, nil
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "u@example.com", job.To)
}

func TestAuthManagement_VerifyEmail_UserLookupError(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{RefreshToken: "refresh"}, nil
	}
	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: "u1"}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{}, errors.New("db")
	}

	_, err := service.VerifyEmail(ctx, dto.VerifyEmailRequest{Token: "t", OTP: "1234"})
	assert.Error(t, err)
}

func TestAuthManagement_ResendOTP_UserLookupError(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{RefreshToken: "refresh"}, nil
	}
	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: "u1"}, nil
	}
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{}, errors.New("db")
	}

	err := service.ResendOTP(ctx, dto.ResendOTPRequest{Token: "t"})
	assert.Error(t, err)
}

func TestAuthManagement_ForgotPassword_UserNotFound(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, errors.New("db")
	}
	err := service.ForgotPassword(ctx, dto.ForgotPasswordRequest{Email: "missing@example.com"})
	assert.ErrorIs(t, err, app_errors.UserNotFound)
}

func TestAuthManagement_ForgotPassword_PendingAndUnverified(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{EmailVerified: false, Status: "pending"}, nil
	}
	err := service.ForgotPassword(ctx, dto.ForgotPasswordRequest{Email: "user@example.com"})
	assert.ErrorIs(t, err, app_errors.InvalidCredentials)
}

func TestAuthManagement_GetUsersAndActiveUsers_ForAssign_Success(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUsersWithRoleFn = func(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
		return []dto.UserWithRole{{ID: uuid.New()}}, nil
	}
	userMgmt.GetActiveUsersForAssignFn = func(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
		return []dto.UserWithRole{{ID: uuid.New()}}, nil
	}

	users, err := service.GetUsers(ctx, appConstant.MasterDatabase)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	active, err := service.GetActiveUsersForAssign(ctx, appConstant.MasterDatabase)
	assert.NoError(t, err)
	assert.Len(t, active, 1)
}

func TestAuthManagement_GetWorkspaceMembersWithRole_EmptyMap(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"get_workspace_members_with_role": map[string]interface{}{}},
	}, nil)
	_, err := service.GetWorkspaceMembersWithRole(ctx, appConstant.MasterDatabase, "w1")
	assert.NoError(t, err)
}

func TestAuthManagement_GetBaseMembersWithRole_EmptyMap(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"get_base_members_with_role": map[string]interface{}{}},
	}, nil)
	_, err := service.GetBaseMembersWithRole(ctx, appConstant.MasterDatabase, "b1")
	assert.NoError(t, err)
}

func TestAuthManagement_DeleteUserCompletely_ErrorOnUserLookup(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{}, errors.New("db")
	}
	err := service.DeleteUserCompletely(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.Error(t, err)
}

func TestAuthManagement_GetBaseMembers_Error(t *testing.T) {
	service, _, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	workspaceMgmt.GetWorkspaceBaseMembersFn = func(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error) {
		return nil, errors.New("db")
	}
	_, err := service.GetBaseMembers(ctx, appConstant.MasterDatabase, "b1")
	assert.Error(t, err)
}

func TestAuthManagement_GetWorkspaceMembers_Error(t *testing.T) {
	service, _, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	workspaceMgmt.GetWorkspaceMembersFn = func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error) {
		return nil, errors.New("db")
	}
	_, err := service.GetWorkspaceMembers(ctx, appConstant.MasterDatabase, "w1")
	assert.Error(t, err)
}

func TestAuthManagement_GetBaseMembers_BulkUsersError(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	workspaceMgmt.GetWorkspaceBaseMembersFn = func(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error) {
		return []tenant.WorkspaceMember{{UserID: "u1", AccessLevel: "viewer"}}, nil
	}
	userMgmt.GetBulkUsersFn = func(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
		return nil, errors.New("db")
	}
	_, err := service.GetBaseMembers(ctx, appConstant.MasterDatabase, "b1")
	assert.Error(t, err)
}

func TestAuthManagement_GetWorkspaceMembers_BulkUsersError(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	workspaceMgmt.GetWorkspaceMembersFn = func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error) {
		return []tenant.WorkspaceMember{{UserID: "u1", AccessLevel: "viewer"}}, nil
	}
	userMgmt.GetBulkUsersFn = func(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
		return nil, errors.New("db")
	}
	_, err := service.GetWorkspaceMembers(ctx, appConstant.MasterDatabase, "w1")
	assert.Error(t, err)
}

func TestAuthManagement_GetBaseMembers_MapError(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	workspaceMgmt.GetWorkspaceBaseMembersFn = func(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error) {
		return []tenant.WorkspaceMember{{UserID: "u1", AccessLevel: "viewer"}}, nil
	}
	userMgmt.GetBulkUsersFn = func(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
		return []tenant.User{{}}, nil
	}

	_, err := service.GetBaseMembers(ctx, appConstant.MasterDatabase, "b1")
	assert.NoError(t, err)
}

func TestAuthManagement_GetWorkspaceMembers_MapError(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	workspaceMgmt.GetWorkspaceMembersFn = func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error) {
		return []tenant.WorkspaceMember{{UserID: "u1", AccessLevel: "viewer"}}, nil
	}
	userMgmt.GetBulkUsersFn = func(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
		return []tenant.User{{}}, nil
	}

	_, err := service.GetWorkspaceMembers(ctx, appConstant.MasterDatabase, "w1")
	assert.NoError(t, err)
}

func TestAuthManagement_GetWorkspaceMembersWithRole_RecordWarn(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"get_workspace_members_with_role": map[string]interface{}{"id": make(chan int)}},
	}, nil)
	_, err := service.GetWorkspaceMembersWithRole(ctx, appConstant.MasterDatabase, "w1")
	assert.NoError(t, err)
}

func TestAuthManagement_GetBaseMembersWithRole_RecordWarn(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"get_base_members_with_role": map[string]interface{}{"id": make(chan int)}},
	}, nil)
	_, err := service.GetBaseMembersWithRole(ctx, appConstant.MasterDatabase, "b1")
	assert.NoError(t, err)
}

func TestAuthManagement_AddUser_EmailAlreadyExists(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{ID: uuid.New()}, nil
	}
	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email: "exists@example.com",
	}, "admin")
	assert.ErrorIs(t, err, app_errors.UserAlreadyExists)
}

func TestAuthManagement_AddUser_EmailLookupError(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, errors.New("db")
	}
	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email: "err@example.com",
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_AddUser_CreateUserError(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return tenant.User{}, errors.New("create failed")
	}
	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email: "new@example.com",
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_AddUser_HandleProfilePictureError(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return tenant.User{ID: uuid.New()}, nil
	}
	userMgmt.AddAvatarFn = func(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error) {
		return dto.UserResponse{}, errors.New("avatar failed")
	}
	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email:      "new@example.com",
		ProfilePic: &multipart.FileHeader{Filename: "avatar.png"},
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_AddUser_GenerateInvitationError(t *testing.T) {
	service, userMgmt, _, resetSvc, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return tenant.User{ID: uuid.New()}, nil
	}
	resetSvc.CreateUserResetTokenFn = func(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{}, errors.New("token failed")
	}
	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email: "new@example.com",
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_AddUser_MembershipError(t *testing.T) {
	service, userMgmt, _, resetSvc, rbacSvc, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return tenant.User{ID: uuid.New()}, nil
	}
	resetSvc.CreateUserResetTokenFn = func(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{Token: req.Token}, nil
	}
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, errors.New("membership failed")
	}
	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email:      "new@example.com",
		IsCoOwner:  false,
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "viewer"}},
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_AddUser_CoOwnerRoleError(t *testing.T) {
	service, userMgmt, _, resetSvc, rbacSvc, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByEmailFn = func(ctx context.Context, schema string, email string) (tenant.User, error) {
		return tenant.User{}, app_errors.UserNotFound
	}
	userMgmt.CreateUserFn = func(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
		return tenant.User{ID: uuid.New()}, nil
	}
	resetSvc.CreateUserResetTokenFn = func(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{Token: req.Token}, nil
	}
	rbacSvc.GetRoleByNameFn = func(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
		return tenant.AccessRole{}, errors.New("role failed")
	}
	_, err := service.AddUser(ctx, appConstant.MasterDatabase, dto.AddUserRequest{
		Email:     "new@example.com",
		IsCoOwner: true,
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_EditUser_NotFound(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{}, errors.New("not found")
	}

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{UserID: "u1"}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_EditUser_UpdateNamesError(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	userMgmt.UpdateUserProfileFn = func(ctx context.Context, schema string, userID string, updateData dto.UpdateUserProfileRequest) (dto.UserResponse, error) {
		return dto.UserResponse{}, errors.New("update failed")
	}

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:    userID,
		FirstName: helpers.StringPtr("New"),
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_EditUser_UpdateAvatarError(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	userMgmt.RemoveAvatarFn = func(ctx context.Context, schema string, userID string) (dto.UserResponse, error) {
		return dto.UserResponse{}, errors.New("remove failed")
	}

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:     userID,
		ProfilePic: &multipart.FileHeader{Filename: "avatar.png"},
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_EditUser_CoOwnerPromote(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{}, nil
	}
	rbacSvc.GetRoleByNameFn = func(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: uuid.New(), Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	rbacSvc.AssignRoleToUserFn = func(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
		return nil, nil
	}
	// Memberships should NOT be called when promoting to coowner
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		t.Fatal("ProcessUserMemberships should not be called when promoting to CoOwner")
		return nil, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:    userID,
		IsCoOwner: helpers.BoolPtr(true),
	}, "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_EditUser_CoOwnerPromote_WithMemberships(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	roleID := uuid.New()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{
			{ID: uuid.New(), RoleID: roleID.String(), ScopeType: "workspace", ScopeID: helpers.StringPtr("w1")},
		}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, id uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: id, Name: "member"}, nil
	}
	rbacSvc.GetRoleByNameFn = func(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: uuid.New(), Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	rbacSvc.AssignRoleToUserFn = func(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
		return nil, nil
	}
	// Memberships should NOT be called when promoting to coowner, even if provided
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		t.Fatal("ProcessUserMemberships should not be called when promoting to CoOwner")
		return nil, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:     userID,
		IsCoOwner:  helpers.BoolPtr(true),
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "editor"}},
	}, "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_EditUser_CoOwnerPromote_Error(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{}, nil
	}
	rbacSvc.GetRoleByNameFn = func(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
		return tenant.AccessRole{}, errors.New("role not found")
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:    userID,
		IsCoOwner: helpers.BoolPtr(true),
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_EditUser_CoOwnerDemote(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		roleID := uuid.New()
		return []dto.AccessMemberDTO{
			{ID: uuid.New(), RoleID: roleID.String(), ScopeType: "system", ScopeID: nil},
		}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, nil
	}
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:     userID,
		IsCoOwner:  helpers.BoolPtr(false),
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "viewer"}},
	}, "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_EditUser_CoOwnerDemote_NoMemberships(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		roleID := uuid.New()
		return []dto.AccessMemberDTO{
			{ID: uuid.New(), RoleID: roleID.String(), ScopeType: "system", ScopeID: nil},
		}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:    userID,
		IsCoOwner: helpers.BoolPtr(false),
		// No memberships provided - should still work
	}, "admin")
	assert.NoError(t, err)
}

func TestAuthManagement_EditUser_IsCoOwner_Nil_UpdateMemberships(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	membershipCalled := false

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{}, nil
	}
	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		membershipCalled = true
		return nil, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:     userID,
		IsCoOwner:  nil, // Not changing coowner status
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "viewer"}},
	}, "admin")
	assert.NoError(t, err)
	assert.True(t, membershipCalled, "ProcessUserMemberships should be called when IsCoOwner is nil")
}

func TestAuthManagement_EditUser_UpdateMembershipsError(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return nil, errors.New("db")
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID:     userID,
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "viewer"}},
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_EditUser_BuildUpdatedUserResponseError(t *testing.T) {
	service, userMgmt, _, _, rbacSvc, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{}, errors.New("db")
	}
	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return nil, nil
	}

	_, err := service.EditUser(ctx, appConstant.MasterDatabase, dto.EditUserRequest{
		UserID: userID,
	}, "admin")
	assert.Error(t, err)
}

func TestAuthManagement_BulkAddMembers_ErrorMessages(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, errors.New("fail")
	}
	resp, err := service.BulkAddMembers(ctx, appConstant.MasterDatabase, dto.BulkAddMembersRequest{
		Members: []dto.BulkMemberRequest{{UserID: "u1"}},
	}, "admin")
	assert.NoError(t, err)
	assert.Len(t, resp.Failed, 1)
}

func TestAuthManagement_BulkAddBaseMembers_ErrorMessages(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	rbacSvc.ProcessUserMembershipsFn = func(ctx context.Context, schema string, userID string, assignedBy string, memberships []dto.MembershipRequest) (interface{}, error) {
		return nil, errors.New("fail")
	}
	resp, err := service.BulkAddBaseMembers(ctx, appConstant.MasterDatabase, "b1", dto.BulkAddBaseMembersRequest{
		Members: []dto.BulkMemberRequest{{UserID: "u1"}},
	}, "admin")
	assert.NoError(t, err)
	assert.Len(t, resp.Failed, 1)
}

func TestAuthManagement_ResetPassword_UpdateUserPasswordError(t *testing.T) {
	service, userMgmt, _, resetSvc, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	token, _ := helpers.GenerateCustomJWT(map[string]interface{}{"user_id": "u1"}, "u1", 3600)
	claims, _ := helpers.DecodeJWT(token)
	issuedAt := fmt.Sprintf("%d", int64(claims["iat"].(float64)))
	resetSvc.GetUserResetTokenFn = func(ctx context.Context, token string) (tenant.UserResetToken, error) {
		return tenant.UserResetToken{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Token: token, IssuedAt: issuedAt}, nil
	}
	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
		return tenant.User{}, errors.New("update failed")
	}
	err := service.ResetPassword(ctx, dto.ResetPasswordRequest{Token: token, NewPassword: "NewPass123!"})
	assert.Error(t, err)
}

func TestAuthManagement_GetUsersWithRole_FunctionName(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, fmt.Sprintf("%s.%s", appConstant.MasterDatabase, "get_workspace_members_with_role"), mock.Anything).Return([]map[string]interface{}{}, nil)
	_, err := service.GetWorkspaceMembersWithRole(ctx, appConstant.MasterDatabase, "w1")
	assert.NoError(t, err)
}

func TestAuthManagement_GetBaseMembersWithRole_FunctionName(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, fmt.Sprintf("%s.%s", appConstant.MasterDatabase, "get_base_members_with_role"), mock.Anything).Return([]map[string]interface{}{}, nil)
	_, err := service.GetBaseMembersWithRole(ctx, appConstant.MasterDatabase, "b1")
	assert.NoError(t, err)
}

func TestAuthManagement_GetUsers_ActiveUsersWithRole_Error(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, _ := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUsersWithRoleFn = func(ctx context.Context, schema string) ([]dto.UserWithRole, error) {
		return nil, errors.New("db")
	}
	_, err := service.GetUsers(ctx, appConstant.MasterDatabase)
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserFromWorkspace_UnexpectedIDType(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": time.Now()},
	}, nil)
	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserFromBase_UnexpectedIDType(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": time.Now()},
	}, nil)
	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.Error(t, err)
}

func TestAuthManagement_ValidateToken_Success(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.ValidateTokenFn = func(ctx context.Context, tokenStr string) (authProviderInterface.Claims, error) {
		return authProviderInterface.Claims{UserId: "u1", Roles: "admin"}, nil
	}
	resp, err := service.ValidateToken(ctx, "token")
	assert.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "u1", resp.UserID)
}

func TestAuthManagement_RefreshToken_Error(t *testing.T) {
	service, _, _, _, _, _, _, _, authProv, _ := setupAuthManagementService()
	ctx := context.Background()

	authProv.RefreshTokenFn = func(ctx context.Context, reqBody authProviderInterface.AuthServiceRefreshRequest) (authProviderInterface.Tokens, error) {
		return authProviderInterface.Tokens{}, errors.New("bad")
	}
	_, err := service.RefreshToken(ctx, dto.RefreshTokenRequest{RefreshToken: "bad"})
	assert.Error(t, err)
}

func TestAuthManagement_RemoveUserFromWorkspace_EmailTemplateSet(t *testing.T) {
	service, userMgmt, workspaceMgmt, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil)
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil)
	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	workspaceMgmt.GetByIDFn = func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
		return tenant.Workspace{Title: "Workspace"}, nil
	}
	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.RemoveUserFromWorkspace(ctx, appConstant.MasterDatabase, "w1", "u1", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "u@example.com", job.To)
}

func TestAuthManagement_RemoveUserFromBase_EmailTemplateSet(t *testing.T) {
	service, userMgmt, _, _, _, _, emailTpl, emailSvc, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"id": uuid.New().String()},
	}, nil).Once()
	tableSvc.On("DeleteRecord", mock.Anything, mock.Anything).Return(nil).Once()
	tableSvc.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{"title": "Base One"},
	}, nil).Once()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{Email: "u@example.com"}, nil
	}
	emailTpl.RemovedFromWorkspaceBodyFn = func(workspaceLabel string) emailProvider.EmailContent {
		return emailProvider.EmailContent{Subject: "removed", Body: workspaceLabel}
	}
	var job emailProvider.EmailJob
	emailSvc.EnqueueFn = func(j emailProvider.EmailJob) { job = j }

	err := service.RemoveUserFromBase(ctx, appConstant.MasterDatabase, "b1", "u1", "admin")
	assert.NoError(t, err)
	assert.Equal(t, "u@example.com", job.To)
}

// TestAuthManagement_ParseRoleData tests the parseRoleData helper method
func TestAuthManagement_ParseRoleData_StringValue(t *testing.T) {
	service, userMgmt, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	userMgmt.GetUserByIDFn = func(ctx context.Context, schema string, id string) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}
	userMgmt.UpdateUserFn = func(ctx context.Context, schema string, id string, updateFields map[string]interface{}) (tenant.User, error) {
		return tenant.User{ID: uuid.MustParse(id)}, nil
	}

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": "Viewer", // Non-owner role so deactivation succeeds
			},
		},
	}, nil)

	result, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestAuthManagement_ParseRoleData_MapValue tests parseRoleData when role data is a map
func TestAuthManagement_ParseRoleData_MapValue(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.NoAccess,
			},
		},
	}, nil)

	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.NoError(t, err)
}

// TestAuthManagement_IsOwnerRole_MatchesOwner tests isOwnerRole when role is Owner
func TestAuthManagement_IsOwnerRole_MatchesOwner(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		roleID := uuid.New()
		return []dto.AccessMemberDTO{{RoleID: roleID.String()}}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.Owner}, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.Owner,
			},
		},
	}, nil)

	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.ErrorIs(t, err, app_errors.OwnerCannotBeDeactivated)
}

// TestAuthManagement_IsOwnerRole_DoesNotMatchOwner tests isOwnerRole when role is not Owner
func TestAuthManagement_IsOwnerRole_DoesNotMatchOwner(t *testing.T) {
	service, _, _, _, rbacSvc, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()
	userID := uuid.New().String()
	coOwnerRoleID := uuid.New()

	rbacSvc.GetUserAccessMembersFn = func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
		return []dto.AccessMemberDTO{{RoleID: coOwnerRoleID.String()}}, nil
	}
	rbacSvc.GetRoleByIDFn = func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
		return tenant.AccessRole{ID: roleID, Name: appConstant.RBACRoleNames.CoOwner}, nil
	}
	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": appConstant.RBACRoleNames.CoOwner,
			},
		},
	}, nil)

	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, userID, "admin-id")
	assert.ErrorIs(t, err, app_errors.CoOwnerCannotBeDeactivated)
}

// TestAuthManagement_ParseRoleData_InvalidJSON tests parseRoleData with invalid data
func TestAuthManagement_ParseRoleData_InvalidJSON(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": "invalid",
		},
	}, nil)

	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.NoError(t, err)
}

// TestAuthManagement_IsOwnerRole_EmptyRoleName tests isOwnerRole with empty role name
func TestAuthManagement_IsOwnerRole_EmptyRoleName(t *testing.T) {
	service, _, _, _, _, _, _, _, _, tableSvc := setupAuthManagementService()
	ctx := context.Background()

	tableSvc.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"get_user_role_by_id": map[string]interface{}{
				"role_name": "",
			},
		},
	}, nil)

	_, err := service.DeactivateUser(ctx, appConstant.MasterDatabase, uuid.New().String(), "admin-id")
	assert.NoError(t, err)
}
