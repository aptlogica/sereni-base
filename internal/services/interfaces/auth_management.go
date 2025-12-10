package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
)

type AuthManagementService interface {
	// Authentication
	Login(ctx context.Context, email string, password string) (dto.LoginResponse, error)
	Register(ctx context.Context, req dto.RegisterRequest) (dto.RegisterResponse, error)
	VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) (dto.LoginResponse, error)
	ResendOTP(ctx context.Context, req dto.ResendOTPRequest) error
	Logout(ctx context.Context, refreshToken string) error

	// crud user
	AddUser(ctx context.Context, schema string, userData dto.AddUserRequest) (master.User, error)
	RemoveUser(ctx context.Context, schema string, userID string) error
	DeleteUserCompletely(ctx context.Context, schema string, userID string) error
	GetUsers(ctx context.Context, schema string) ([]dto.UserWithRole, error)

	// Token Management
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.TokenResponse, error)

	// Password
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error
	HandleKeycloakCallback(ctx context.Context, code string) (dto.LoginResponse, error)
	GetAuthProviderUrl(provider string) string

	AssignUserToWorkspace(ctx context.Context, schema string, req dto.CreateMemberRequest) error
	AddMultipleMembers(ctx context.Context, schema string, req dto.AddMultipleMembersRequest) (dto.AddMultipleMembersResponse, error)
	RemoveUserFromWorkspace(ctx context.Context, schema string, req dto.RemoveMemberRequest) error
	InviteMemberToWorkspace(ctx context.Context, schema string, req dto.CreateMemberRequest) error
	GetWorkspaceMembers(ctx context.Context, schema string, workspaceID string) ([]dto.WorkspaceMemberResponse, error)
	GetBaseMembers(ctx context.Context, schema string, baseID string) ([]dto.WorkspaceMemberResponse, error)
	UpdatePassword(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) error
	ActivateUser(ctx context.Context, schema string, userID string) (dto.UserResponse, error)
	DeactivateUser(ctx context.Context, schema string, userID string) (dto.UserResponse, error)
}
