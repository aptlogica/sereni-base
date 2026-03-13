// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"

	"github.com/google/uuid"
)

// AccessRoleDTO represents a role with scope-based access control
type AccessRoleDTO struct {
	ID          uuid.UUID `json:"id,omitempty" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	Name        string    `json:"name" binding:"required" example:"owner" mapstructure:"name"`
	ScopeLevel  string    `json:"scope_level" binding:"required" example:"workspace" mapstructure:"scope_level"` // system, workspace, base
	Priority    int       `json:"priority" binding:"required" example:"100" mapstructure:"priority"`             // higher = overrides lower
	Description *string   `json:"description,omitempty" example:"Workspace owner with full control" mapstructure:"description"`
	IsDefault   bool      `json:"is_default" example:"false" mapstructure:"is_default"`
	WorkspaceID *string   `json:"workspace_id,omitempty" example:"ws-123" mapstructure:"workspace_id"` // for base-level roles
	CreatedAt   time.Time `json:"created_time,omitempty" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
	UpdatedAt   time.Time `json:"last_modified_time,omitempty" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"last_modified_time"`
}

func (r AccessRoleDTO) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 r.ID,
		"name":               r.Name,
		"scope_level":        r.ScopeLevel,
		"priority":           r.Priority,
		"description":        r.Description,
		"is_default":         r.IsDefault,
		"workspace_id":       r.WorkspaceID,
		"created_time":       r.CreatedAt,
		"last_modified_time": r.UpdatedAt,
	}
}

// AccessRoleResponse represents a role for API responses
type AccessRoleResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ScopeLevel  string    `json:"scope_level"`
	Priority    int       `json:"priority"`
	Description *string   `json:"description,omitempty"`
	IsDefault   bool      `json:"is_default"`
	WorkspaceID *string   `json:"workspace_id,omitempty"`
	CreatedAt   time.Time `json:"created_time"`
	UpdatedAt   time.Time `json:"last_modified_time"`
}
