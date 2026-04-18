// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"

	"github.com/google/uuid"
)

// RolePermissionDTO represents a role-permission mapping
type RolePermissionDTO struct {
	ID           uuid.UUID `json:"id,omitempty" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	RoleID       uuid.UUID `json:"role_id" binding:"required" example:"c4e2f3d0-2345-5b67-ac9d-bcdef234567" format:"uuid" mapstructure:"role_id"`
	PermissionID uuid.UUID `json:"permission_id" binding:"required" example:"d5e3f4e0-3456-6c78-bd0e-cdef345678e" format:"uuid" mapstructure:"permission_id"`
	CreatedAt    time.Time `json:"created_time,omitempty" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
}

func (rp RolePermissionDTO) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":            rp.ID,
		"role_id":       rp.RoleID,
		"permission_id": rp.PermissionID,
		"created_time":  rp.CreatedAt,
	}
}

// RolePermissionResponse represents a role-permission mapping for API responses
type RolePermissionResponse struct {
	ID           uuid.UUID              `json:"id"`
	RoleID       uuid.UUID              `json:"role_id"`
	PermissionID uuid.UUID              `json:"permission_id"`
	Permission   *PermissionWithDetails `json:"permission,omitempty"`
	CreatedAt    time.Time              `json:"created_time"`
}

// RolePermissions represents a role with all its permissions
type RolePermissions struct {
	RoleID      uuid.UUID               `json:"role_id"`
	RoleName    string                  `json:"role_name"`
	ScopeLevel  string                  `json:"scope_level"`
	Permissions []PermissionWithDetails `json:"permissions"`
}
