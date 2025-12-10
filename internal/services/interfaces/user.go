package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
)

type UserService interface {
	// CRUD
	CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (master.User, error)
	GetUserByEmail(ctx context.Context, schema string, email string) (master.User, error)
	GetUserByID(ctx context.Context, schema string, id string) (master.User, error)
	UpdateUser(ctx context.Context, schema string, id string, updateData map[string]interface{}) (master.User, error)
	GetAllUsers(ctx context.Context, schema string) ([]master.User, error)
	GetBulkUsers(ctx context.Context, schema string, ids []string) ([]master.User, error)
	DeleteUser(ctx context.Context, id string, schema string) error

	// // Authentication
	// RegisterUser(dto RegisterDTO) (User, error)
	// LoginUser(dto LoginDTO) (AuthToken, error)
	// LogoutUser(userID string) error
	// RefreshToken(token string) (AuthToken, error)

	// // Authorization
	// AssignRole(userID string, role string) error
	// RevokeRole(userID string, role string) error
	// GetUserRoles(userID string) ([]string, error)
	// HasPermission(userID string, permission string) (bool, error)

	// // Profile & Security
	// UpdateProfile(userID string, dto UpdateProfileDTO) (User, error)
	// ChangePassword(userID, oldPassword, newPassword string) error
	// ResetPassword(email string) error
	// ActivateUser(userID string) error
	// DeactivateUser(userID string) error

	// // Admin & Management
	// ListUsers(filter UserFilter) ([]User, error)
	// SearchUsers(query string) ([]User, error)
	// SuspendUser(userID string) error
	// RestoreUser(userID string) error
}
