// serenibase/internal/dto/view_dto.go
// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"

	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
)

// ViewInsertion is used when inserting a new view
type ViewInsertion struct {
	ID      uuid.UUID `db:"id" json:"id,omitempty"`
	ModelID uuid.UUID `db:"model_id" json:"model_id,omitempty"`
	BaseID  uuid.UUID `db:"base_id" json:"base_id,omitempty"`

	// Basic info
	Title       string  `db:"title" json:"title,omitempty"`
	Description *string `db:"description" json:"description,omitempty"`
	Alias       *string `db:"alias" json:"alias,omitempty"`
	Type        string  `db:"type" json:"type,omitempty"` // grid, gallery, form, kanban

	// Configuration
	IsDefault bool    `db:"is_default" json:"is_default,omitempty"`
	LockType  *string `db:"lock_type" json:"lock_type,omitempty"`
	Password  *string `db:"password" json:"password,omitempty"`

	// Sharing
	Public bool    `db:"public" json:"public,omitempty"`
	UUID   *string `db:"uuid" json:"uuid,omitempty"`

	// Settings
	Meta       map[string]interface{} `db:"meta" json:"meta,omitempty"` // JSON config
	OrderIndex *float64               `db:"order_index" json:"order_index,omitempty"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty"`
	CreatedBy string    `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy string    `db:"last_modified_by" json:"last_modified_by,omitempty"`
}

// Map converts ViewInsertion → map[string]interface{} for DB insert
func (v *ViewInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 v.ID,
		"model_id":           v.ModelID,
		"base_id":            v.BaseID,
		"title":              v.Title,
		"description":        v.Description,
		"alias":              v.Alias,
		"type":               v.Type,
		"is_default":         v.IsDefault,
		"lock_type":          v.LockType,
		"password":           v.Password,
		"public":             v.Public,
		"uuid":               v.UUID,
		"meta":               helpers.InterfaceToJSONString(v.Meta),
		"order_index":        v.OrderIndex,
		"created_time":       v.CreatedAt,
		"last_modified_time": v.UpdatedAt,
		"created_by":         v.CreatedBy,
		"last_modified_by":   v.UpdatedBy,
	}
}

// ViewUpdate is used when updating an existing view
type ViewUpdate struct {
	Title       *string                 `db:"title" json:"title,omitempty"`
	Description *string                 `db:"description" json:"description,omitempty"`
	Type        *string                 `db:"type" json:"type,omitempty"`
	IsDefault   *bool                   `db:"is_default" json:"is_default,omitempty"`
	LockType    *string                 `db:"lock_type" json:"lock_type,omitempty"`
	Password    *string                 `db:"password" json:"password,omitempty"`
	Public      *bool                   `db:"public" json:"public,omitempty"`
	UUID        *string                 `db:"uuid" json:"uuid,omitempty"`
	Meta        *map[string]interface{} `db:"meta" json:"meta,omitempty"`
	OrderIndex  *float64                `db:"order_index" json:"order_index,omitempty"`
	UpdatedAt   time.Time               `db:"last_modified_time" json:"last_modified_time,omitempty"`
	UpdatedBy   string                  `db:"last_modified_by" json:"last_modified_by,omitempty"`
}

// Map converts ViewUpdate → map[string]interface{} for DB update
func (v *ViewUpdate) Map() map[string]interface{} {
	result := make(map[string]interface{})
	if v.Title != nil {
		result["title"] = *v.Title
	}
	if v.Description != nil {
		result["description"] = *v.Description
	}
	if v.Type != nil {
		result["type"] = *v.Type
	}
	if v.IsDefault != nil {
		result["is_default"] = *v.IsDefault
	}
	if v.LockType != nil {
		result["lock_type"] = *v.LockType
	}
	if v.Password != nil {
		result["password"] = *v.Password
	}
	if v.Public != nil {
		result["public"] = *v.Public
	}
	if v.UUID != nil {
		result["uuid"] = *v.UUID
	}
	if v.Meta != nil {
		result["meta"] = helpers.InterfaceToJSONString(*v.Meta)
	}
	if v.OrderIndex != nil {
		result["order_index"] = *v.OrderIndex
	}
	if v.UpdatedBy != "" {
		result["last_modified_by"] = v.UpdatedBy
	}
	result["last_modified_time"] = v.UpdatedAt
	return result
}

type CreateViewRequest struct {
	ModelID     uuid.UUID               `db:"model_id" json:"model_id" mapstructure:"model_id" binding:"required"`
	BaseID      uuid.UUID               `db:"base_id" json:"base_id" mapstructure:"base_id" binding:"required"`
	Title       string                  `db:"title" json:"title" mapstructure:"title" binding:"required"`
	Description string                  `db:"description" json:"description" mapstructure:"description" binding:"omitempty"`
	Type        string                  `db:"type" json:"type" mapstructure:"type" binding:"required"`
	Meta        *map[string]interface{} `db:"meta" json:"meta" mapstructure:"meta" binding:"required"`
	OrderIndex  *float64                `db:"order_index" json:"order_index,omitempty" mapstructure:"order_index"`
	CreatedBy   string                  `json:"created_by,omitempty"`
}

type ViewResponse struct {
	ID          uuid.UUID               `db:"id" json:"id" mapstructure:"id"`
	ModelID     uuid.UUID               `db:"model_id" json:"model_id" mapstructure:"model_id"`
	BaseID      uuid.UUID               `db:"base_id" json:"base_id" mapstructure:"base_id"`
	Title       string                  `db:"title" json:"title" mapstructure:"title"`
	Description *string                 `db:"description" json:"description" mapstructure:"description"`
	Alias       *string                 `db:"alias" json:"alias" mapstructure:"alias"`
	Type        string                  `db:"type" json:"type" mapstructure:"type"`
	IsDefault   *bool                   `db:"is_default" json:"is_default" mapstructure:"is_default"`
	LockType    *string                 `db:"lock_type" json:"lock_type" mapstructure:"lock_type"`
	Password    *string                 `db:"password" json:"password" mapstructure:"password"`
	Public      *bool                   `db:"public" json:"public" mapstructure:"public"`
	UUID        *string                 `db:"uuid" json:"uuid" mapstructure:"uuid"`
	Meta        *map[string]interface{} `db:"meta" json:"meta" mapstructure:"meta"`
	OrderIndex  *float64                `db:"order_index" json:"order_index" mapstructure:"order_index"`
	CreatedAt   time.Time               `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt   time.Time               `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
	CreatedBy   string                  `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy   string                  `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`
}
