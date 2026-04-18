// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"mime/multipart"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

type UserManagementService interface {
	GetUserProfileByID(ctx context.Context, schema string, userID string) (dto.UserResponse, error)
	UpdateUserProfile(ctx context.Context, schema string, userID string, updateData dto.UpdateUserProfileRequest) (dto.UserResponse, error)
	UpdatePassword(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) (tenant.User, error)
	AddAvatar(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error)
	RemoveAvatar(ctx context.Context, schema string, userID string) (dto.UserResponse, error)
	GetUserByEmail(ctx context.Context, schema string, email string) (tenant.User, error)
	// StartRegistration(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, tenant.Tenant, error)
	CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error)
	UpdateUser(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error)
	GetUserByID(ctx context.Context, schema string, id string) (tenant.User, error)
	GetAllUsers(ctx context.Context, schema string) ([]tenant.User, error)
	GetWorkspaces(ctx context.Context, schema string, userID string, roles string) ([]dto.UserWorkspaceResponse, error)
	GetBulkUsers(ctx context.Context, schema string, ids []string) ([]tenant.User, error)
	GetUsersWithRole(ctx context.Context, schema string) ([]dto.UserWithRole, error)
	GetActiveUsersForAssign(ctx context.Context, schema string) ([]dto.UserWithRole, error)
	DeleteUserCompletely(ctx context.Context, schema string, userID string) error
	GetUserAccessDetails(ctx context.Context, schema string, userID string, roles string, workspaceID string) (dto.UserAccessDetailsResponse, error)
	GetUserRolesAndAccess(ctx context.Context, schema string, userID string, scopeID *string) ([]dto.UserRolesAccessResponse, error)
}
