// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"

	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
)

// ModelDTO represents the Models table structure for API usage
type ModelDTO struct {
	ID          string                 `json:"id"`
	BaseID      string                 `json:"base_id"`
	WorkspaceID string                 `json:"workspace_id"`
	Title       string                 `json:"title"`
	Alias       string                 `json:"alias"`
	Type        string                 `json:"type"`             // table, view, junction
	Meta        map[string]interface{} `json:"meta,omitempty"`   // JSON metadata
	Schema      string                 `json:"schema,omitempty"` // JSON schema definition
	Enabled     bool                   `json:"enabled"`
	MM          bool                   `json:"mm"` // many-to-many junction table
	Pinned      bool                   `json:"pinned"`
	Deleted     bool                   `json:"deleted"`
	Tags        string                 `json:"tags,omitempty"`
	OrderIndex  float64                `json:"order_index,omitempty"`
	RowCount    int64                  `json:"row_count"`
	ColumnCount int                    `json:"column_count"`
	StorageUsed int64                  `json:"storage_used_bytes"`
	CreatedBy   string                 `json:"created_by"`
	UpdatedBy   string                 `json:"last_modified_by"`
	CreatedAt   time.Time              `json:"created_time"`
	UpdatedAt   time.Time              `json:"last_modified_time"`
}

type ModelInsertion struct {
	ID               string                 `json:"id" binding:"required"`
	BaseID           string                 `json:"base_id" binding:"required"`
	WorkspaceID      string                 `json:"workspace_id" binding:"required"`
	Title            string                 `json:"title" binding:"required"`
	Description      string                 `json:"description" binding:"omitempty"`
	Alias            string                 `json:"alias" binding:"required"`
	Type             string                 `json:"type" binding:"omitempty,oneof=table view junction"`
	Meta             map[string]interface{} `json:"meta,omitempty"`
	Schema           string                 `json:"schema,omitempty"`
	Tags             string                 `json:"tags,omitempty"`
	OrderIndex       float64                `json:"order_index,omitempty"`
	CreatedBy        string                 `json:"created_by"`
	UpdatedBy        string                 `json:"last_modified_by"`
	CreatedTime      time.Time              `json:"created_time"`
	LastModifiedTime time.Time              `json:"last_modified_time"`
}

// UpdateModelRequest DTO for updating an existing model
type UpdateModelRequest struct {
	Title       *string                 `json:"title,omitempty"`
	Description *string                 `json:"description" binding:"omitempty"`
	Alias       *string                 `json:"alias,omitempty"`
	Type        *string                 `json:"type,omitempty" binding:"omitempty,oneof=table view junction"`
	Meta        *map[string]interface{} `json:"meta,omitempty"`
	Schema      *string                 `json:"schema,omitempty"`
	Enabled     *bool                   `json:"enabled,omitempty"`
	MM          *bool                   `json:"mm,omitempty"`
	Pinned      *bool                   `json:"pinned,omitempty"`
	Deleted     *bool                   `json:"deleted,omitempty"`
	Tags        *string                 `json:"tags,omitempty"`
	OrderIndex  *float64                `json:"order_index,omitempty"`
	UpdatedBy   string                  `json:"last_modified_by,omitempty"`
}

func (r *ModelInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 r.ID,
		"base_id":            r.BaseID,
		"workspace_id":       r.WorkspaceID,
		"description":        r.Description,
		"title":              r.Title,
		"alias":              r.Alias,
		"type":               r.Type,
		"meta":               helpers.InterfaceToJSONString(r.Meta),
		"schema":             r.Schema,
		"tags":               r.Tags,
		"order_index":        r.OrderIndex,
		"created_by":         r.CreatedBy,
		"last_modified_by":   r.UpdatedBy,
		"created_time":       r.CreatedTime,
		"last_modified_time": r.LastModifiedTime,
	}
}

// Map converts UpdateModelRequest into a map with only non-nil fields
func (r *UpdateModelRequest) Map() map[string]interface{} {
	updates := make(map[string]interface{})
	if r.Title != nil {
		updates["title"] = *r.Title
	}
	if r.Description != nil {
		updates["description"] = r.Description
	}
	if r.Alias != nil {
		updates["alias"] = *r.Alias
	}
	if r.Type != nil {
		updates["type"] = *r.Type
	}
	if r.Meta != nil {
		updates["meta"] = helpers.InterfaceToJSONString(*r.Meta)
	}
	if r.Schema != nil {
		updates["schema"] = *r.Schema
	}
	if r.Enabled != nil {
		updates["enabled"] = *r.Enabled
	}
	if r.MM != nil {
		updates["mm"] = *r.MM
	}
	if r.Pinned != nil {
		updates["pinned"] = *r.Pinned
	}
	if r.Deleted != nil {
		updates["deleted"] = *r.Deleted
	}
	if r.Tags != nil {
		updates["tags"] = *r.Tags
	}
	if r.OrderIndex != nil {
		updates["order_index"] = *r.OrderIndex
	}
	updates["last_modified_by"] = r.UpdatedBy

	return updates
}

type ModelResponse struct {
	ID          uuid.UUID `db:"id" json:"id" mapstructure:"id"`
	BaseID      uuid.UUID `db:"base_id" json:"base_id" mapstructure:"base_id"`
	WorkspaceID uuid.UUID `db:"workspace_id" json:"workspace_id" mapstructure:"workspace_id"`

	Title       string `db:"title" json:"title" mapstructure:"title"`
	Description string `db:"description" json:"description" mapstructure:"description"`

	Alias string                 `db:"alias" json:"alias" mapstructure:"alias"`
	Meta  map[string]interface{} `db:"meta" json:"meta" mapstructure:"meta"`

	OrderIndex *float64 `db:"order_index" json:"order_index" mapstructure:"order_index"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`

	CreatedAt time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
}
