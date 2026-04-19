// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"

	"github.com/google/uuid"
)

// ResourceDTO represents a system resource
type ResourceDTO struct {
	ID          uuid.UUID `json:"id,omitempty" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	Code        string    `json:"code" binding:"required" example:"workspace" mapstructure:"code"` // workspace, base, records, members, etc.
	Description *string   `json:"description,omitempty" example:"Workspace resource" mapstructure:"description"`
	CreatedAt   time.Time `json:"created_time,omitempty" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
}

func (r ResourceDTO) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":           r.ID,
		"code":         r.Code,
		"description":  r.Description,
		"created_time": r.CreatedAt,
	}
}

// ResourceResponse represents a resource for API responses
type ResourceResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_time"`
}
