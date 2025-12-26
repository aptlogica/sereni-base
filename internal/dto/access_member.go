package dto

import (
	"time"

	"github.com/google/uuid"
)

// AccessMemberDTO represents a user-role assignment with scope
type AccessMemberDTO struct {
	ID          uuid.UUID `json:"id,omitempty" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	UserID      string    `json:"user_id" binding:"required" example:"5a6b7c8d-9e0f-1234-5678-abcdef987654" mapstructure:"user_id"`
	ScopeType   string    `json:"scope_type" binding:"required" example:"workspace" mapstructure:"scope_type"`              // system, workspace, base
	ScopeID     *string   `json:"scope_id,omitempty" example:"e5e3f4e0-3456-6c78-bd0e-cdef345678e" mapstructure:"scope_id"` // null for system
	RoleID      string    `json:"role_id" binding:"required" example:"c4e2f3d0-2345-5b67-ac9d-bcdef234567" mapstructure:"role_id"`
	WorkspaceID *string   `json:"workspace_id,omitempty" example:"w-123456" mapstructure:"workspace_id"` // workspace that owns this access (for base-level, stores workspace_id)
	AssignedBy  *string   `json:"assigned_by,omitempty" example:"5a6b7c8d-9e0f-1234-5678-abcdef987654" mapstructure:"assigned_by"`
	CreatedAt   time.Time `json:"created_time,omitempty" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
	UpdatedAt   time.Time `json:"last_modified_time,omitempty" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"last_modified_time"`
}

func (am AccessMemberDTO) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 am.ID,
		"user_id":            am.UserID,
		"scope_type":         am.ScopeType,
		"scope_id":           am.ScopeID,
		"role_id":            am.RoleID,
		"workspace_id":       am.WorkspaceID,
		"assigned_by":        am.AssignedBy,
		"created_time":       am.CreatedAt,
		"last_modified_time": am.UpdatedAt,
	}
}

// AccessMemberResponse represents a user-role assignment for API responses
type AccessMemberResponse struct {
	ID         uuid.UUID           `json:"id"`
	UserID     string              `json:"user_id"`
	ScopeType  string              `json:"scope_type"`
	ScopeID    *string             `json:"scope_id,omitempty"`
	RoleID     string              `json:"role_id"`
	Role       *AccessRoleResponse `json:"role,omitempty"`
	AssignedBy *string             `json:"assigned_by,omitempty"`
	CreatedAt  time.Time           `json:"created_time"`
	UpdatedAt  time.Time           `json:"last_modified_time"`
}

// UserAccessInfo provides user access information across scopes
type UserAccessInfo struct {
	UserID          string            `json:"user_id"`
	SystemRole      *AccessRoleDTO    `json:"system_role,omitempty"` // system-level role if any
	WorkspaceAccess []AccessMemberDTO `json:"workspace_access"`      // workspace and base-level roles
}

// BulkAssignRoleRequest assigns a role to multiple users in a scope
type BulkAssignRoleRequest struct {
	UserIDs    []string `json:"user_ids" binding:"required" example:"[\"5a6b7c8d\",\"6b7c8d9e\"]"`
	ScopeType  string   `json:"scope_type" binding:"required" example:"workspace"`
	ScopeID    *string  `json:"scope_id,omitempty" example:"e5e3f4e0-3456-6c78-bd0e-cdef345678e"`
	RoleID     string   `json:"role_id" binding:"required" example:"c4e2f3d0-2345-5b67-ac9d-bcdef234567"`
	AssignedBy *string  `json:"assigned_by,omitempty"`
}
