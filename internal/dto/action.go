// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"

	"github.com/google/uuid"
)

// ActionDTO represents a system action
type ActionDTO struct {
	ID          uuid.UUID `json:"id,omitempty" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	Code        string    `json:"code" binding:"required" example:"read" mapstructure:"code"` // read, create, update, delete, share, etc.
	Description *string   `json:"description,omitempty" example:"Read action" mapstructure:"description"`
	CreatedAt   time.Time `json:"created_time,omitempty" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
}

func (a ActionDTO) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":           a.ID,
		"code":         a.Code,
		"description":  a.Description,
		"created_time": a.CreatedAt,
	}
}

// ActionResponse represents an action for API responses
type ActionResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_time"`
}
