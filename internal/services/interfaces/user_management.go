package interfaces

import (
	"context"
	"mime/multipart"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"

	"github.com/google/uuid"
)

type UserManagementService interface {
	GetUserProfileByID(ctx context.Context, schema string, userID string) (dto.UserResponse, error)
	UpdateUserProfile(ctx context.Context, schema string, userID string, updateData dto.UpdateUserProfileRequest) (dto.UserResponse, error)
	UpdatePassword(ctx context.Context, schema string, userID string, updateData dto.UpdateUserPasswordRequest) (master.User, error) 
	AddAvatar(ctx context.Context, schema string, userID string, fileHeader *multipart.FileHeader) (dto.UserResponse, error)
	RemoveAvatar(ctx context.Context, schema string, userID string) (dto.UserResponse, error)
	GetUserByEmail(ctx context.Context, schema string, email string) (master.User, error)
	CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (master.User, error)
	UpdateUser(ctx context.Context, schema string, id string, updateData map[string]interface{}) (master.User, error)
	GetUserByID(ctx context.Context, schema string, id string) (master.User, error)
	AddUserToTenant(ctx context.Context, schema string, userData dto.AddUserRequest, roleId uuid.UUID, userPassword string) (master.User, master.Tenant, error)
	GetAllUsers(ctx context.Context, schema string) ([]master.User, error)
	AddUserRole(ctx context.Context, schema string, userID, roleID uuid.UUID) error
	GetWorkspaces(ctx context.Context, schema string, userID string, roles string) ([]dto.UserWorkspaceResponse, error)
	GetBulkUsers(ctx context.Context, schema string, ids []string) ([]master.User, error)
	GetUsersWithRole(ctx context.Context, schema string) ([]dto.UserWithRole, error)
	DeleteUserCompletely(ctx context.Context, schema string, userID string) error
	GetUserAccessDetails(ctx context.Context, schema string, userID string, roles string, workspaceID string) (dto.UserAccessDetailsResponse, error)
}
