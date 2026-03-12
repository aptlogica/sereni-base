// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"

	"github.com/google/uuid"
)

type RoleInsertion struct {
	ID          uuid.UUID `json:"id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	Name        string    `json:"name" binding:"required" example:"Admin" format:"string" mapstructure:"name"`
	Description *string   `json:"description,omitempty" example:"Administrator role with all permissions" format:"string" mapstructure:"description"`
	IsDefault   bool      `json:"is_default" example:"false" format:"boolean" mapstructure:"is_default"`
	CreatedAt   time.Time `json:"created_time" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
	UpdatedAt   time.Time `json:"last_modified_time" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"last_modified_time"`
}

func (r RoleInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 r.ID,
		"name":               r.Name,
		"description":        r.Description,
		"is_default":         r.IsDefault,
		"created_time":       r.CreatedAt,
		"last_modified_time": r.UpdatedAt,
	}
}

type UserRoleInsertion struct {
	ID     uuid.UUID `json:"id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	RoleID uuid.UUID `json:"role_id" binding:"required" example:"e4eaaaf2-d142-11e1-b3e4-080027620cdd" format:"uuid" mapstructure:"role_id"`
	UserID uuid.UUID `json:"user_id" binding:"required" example:"5a6b7c8d-9e0f-1234-5678-abcdef987654" format:"uuid" mapstructure:"user_id"`
}

func (u UserRoleInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":      u.ID,
		"role_id": u.RoleID,
		"user_id": u.UserID,
	}
}
