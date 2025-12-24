package dto

import (
	"time"

	"github.com/google/uuid"
)

// PermissionDTO represents a permission (resource × action)
type PermissionDTO struct {
	ID         uuid.UUID `json:"id,omitempty" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	ResourceID uuid.UUID `json:"resource_id" binding:"required" example:"c4e2f3d0-2345-5b67-ac9d-bcdef234567" format:"uuid" mapstructure:"resource_id"`
	ActionID   uuid.UUID `json:"action_id" binding:"required" example:"d5e3f4e0-3456-6c78-bd0e-cdef345678e" format:"uuid" mapstructure:"action_id"`
	CreatedAt  time.Time `json:"created_time,omitempty" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
}

func (p PermissionDTO) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":           p.ID,
		"resource_id":  p.ResourceID,
		"action_id":    p.ActionID,
		"created_time": p.CreatedAt,
	}
}

// PermissionResponse represents a permission for API responses
type PermissionResponse struct {
	ID         uuid.UUID         `json:"id"`
	ResourceID uuid.UUID         `json:"resource_id"`
	ActionID   uuid.UUID         `json:"action_id"`
	Resource   *ResourceResponse `json:"resource,omitempty"`
	Action     *ActionResponse   `json:"action,omitempty"`
	CreatedAt  time.Time         `json:"created_time"`
}

// PermissionWithDetails includes resource and action details
type PermissionWithDetails struct {
	ID           uuid.UUID `json:"id"`
	ResourceCode string    `json:"resource_code"`
	ActionCode   string    `json:"action_code"`
	CreatedAt    time.Time `json:"created_time"`
}
