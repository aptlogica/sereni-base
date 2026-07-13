// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

type AuthManagementService interface {
	// Authentication
	Login(ctx context.Context, email string, password string) (dto.LoginResponse, error)
	// Register(ctx context.Context, req dto.RegisterRequest) (dto.RegisterResponse, error)
	RegisterOwner(ctx context.Context, req dto.RegisterRequest) (dto.LoginResponse, error)
	VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) (dto.LoginResponse, error)
	ResendOTP(ctx context.Context, req dto.ResendOTPRequest) error
	Logout(ctx context.Context, refreshToken string) error

	// crud user
	AddUser(ctx context.Context, schema string, userData dto.AddUserRequest, reqBy string) (tenant.User, error)
	EditUser(ctx context.Context, schema string, userData dto.EditUserRequest, reqBy string) (dto.UserResponse, error)
	RemoveUser(ctx context.Context, schema string, userID string, reqBy string) error
	DeleteUserCompletely(ctx context.Context, schema string, userID string, reqBy string) error
	GetUsers(ctx context.Context, schema string) ([]dto.UserWithRole, error)
	GetActiveUsersForAssign(ctx context.Context, schema string) ([]dto.UserWithRole, error)

	// Token Management
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.TokenResponse, error)
	ValidateToken(ctx context.Context, token string) (dto.TokenValidationResponse, error)
	VerifyToken(ctx context.Context, token string) (dto.TokenValidationResponse, error)

	// Password
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error
	HandleKeycloakCallback(ctx context.Context, code string) (dto.LoginResponse, error)
	GetAuthProviderUrl(provider string) string

	AssignUserToWorkspace(ctx context.Context, schema string, req dto.CreateMemberRequest, reqBy string) error
	RemoveUserFromWorkspace(ctx context.Context, schema string, workspaceID string, userID string, accessMemberID *string, reqBy string) error
	RemoveUserFromBase(ctx context.Context, schema string, baseID string, userID string, reqBy string) error
	RemoveAccessMemberByID(ctx context.Context, schema string, accessMemberID string, reqBy string) error
	GetWorkspaceMembers(ctx context.Context, schema string, workspaceID string) ([]dto.WorkspaceMemberResponse, error)
	GetBaseMembers(ctx context.Context, schema string, baseID string) ([]dto.WorkspaceMemberResponse, error)
	GetWorkspaceMembersWithRole(ctx context.Context, schema string, workspaceID string) ([]dto.UserWithRole, error)
	GetBaseMembersWithRole(ctx context.Context, schema string, baseID string) ([]dto.UserWithRole, error)
	UpdatePassword(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) error
	ActivateUser(ctx context.Context, schema string, userID string, reqBy string) (dto.UserResponse, error)
	DeactivateUser(ctx context.Context, schema string, userID string, reqBy string) (dto.UserResponse, error)
	BulkAddMembers(ctx context.Context, schema string, req dto.BulkAddMembersRequest, userID string) (dto.BulkAddMembersResponse, error)
	BulkAddBaseMembers(ctx context.Context, schema string, baseID string, req dto.BulkAddBaseMembersRequest, userID string) (dto.BulkAddMembersResponse, error)
}
