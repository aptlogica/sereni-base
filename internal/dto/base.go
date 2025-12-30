// serenibase/internal/dto/workspace_dto.go
package dto

import (
	"encoding/json"
	"fmt"
	"serenibase/internal/utils/helpers"
	"time"

	"github.com/google/uuid"
)

// BaseInsertion is used when inserting a new base
type BaseInsertion struct {
	ID               uuid.UUID              `db:"id" json:"id,omitempty"`
	WorkspaceID      uuid.UUID              `db:"workspace_id" json:"workspace_id,omitempty"`
	Title            string                 `db:"title" json:"title,omitempty"`
	Description      *string                `db:"description" json:"description,omitempty"`
	Type             string                 `db:"type" json:"type,omitempty"`             // internal, external
	Config           map[string]interface{} `db:"config" json:"config,omitempty"`         // JSON database config
	Settings         map[string]interface{} `db:"settings" json:"settings,omitempty"`     // JSON base settings
	Meta             map[string]interface{} `db:"meta" json:"meta,omitempty"`             // Additional metadata
	Status           string                 `db:"status" json:"status,omitempty"`         // active, pending
	Visibility       string                 `db:"visibility" json:"visibility,omitempty"` // private, public, shared
	TableCount       int                    `db:"table_count" json:"table_count,omitempty"`
	RowCount         int64                  `db:"row_count" json:"row_count,omitempty"`
	StorageUsedBytes int64                  `db:"storage_used_bytes" json:"storage_used_bytes,omitempty"`
	CreatedBy        string                 `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy        string                 `db:"last_modified_by" json:"last_modified_by,omitempty"`
	CreatedAt        time.Time              `db:"created_time" json:"created_time,omitempty"`
	UpdatedAt        time.Time              `db:"last_modified_time" json:"last_modified_time,omitempty"`
}

// Map converts BaseInsertion → map[string]interface{} for DB insert
func (b *BaseInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 b.ID,
		"workspace_id":       b.WorkspaceID,
		"title":              b.Title,
		"description":        b.Description,
		"type":               b.Type,
		"config":             helpers.InterfaceToJSONString(b.Config),
		"settings":           helpers.InterfaceToJSONString(b.Settings),
		"meta":               helpers.InterfaceToJSONString(b.Meta),
		"status":             b.Status,
		"visibility":         b.Visibility,
		"table_count":        b.TableCount,
		"row_count":          b.RowCount,
		"storage_used_bytes": b.StorageUsedBytes,
		"created_by":         b.CreatedBy,
		"last_modified_by":   b.UpdatedBy,
		"created_time":       b.CreatedAt,
		"last_modified_time": b.UpdatedAt,
	}
}

// SetConfig sets the Config field from a JSON string
func (b *BaseInsertion) SetConfig(config string) error {
	if config == "" {
		b.Config = nil
		return nil
	}
	// Validate JSON
	if !json.Valid([]byte(config)) {
		return fmt.Errorf("invalid JSON for config")
	}
	b.Config = b.Config
	return nil
}

// SetSettings sets the Settings field from a JSON string
func (b *BaseInsertion) SetSettings(settings string) error {
	if settings == "" {
		b.Settings = nil
		return nil
	}
	// Validate JSON
	if !json.Valid([]byte(settings)) {
		return fmt.Errorf("invalid JSON for settings")
	}
	b.Settings = b.Settings
	return nil
}

// SetMeta sets the Meta field from a JSON string
func (b *BaseInsertion) SetMeta(meta string) error {
	if meta == "" {
		b.Meta = nil
		return nil
	}
	// Validate JSON
	if !json.Valid([]byte(meta)) {
		return fmt.Errorf("invalid JSON for meta")
	}
	b.Meta = b.Meta
	return nil
}

// BaseUpdate is used when updating an existing base
type BaseUpdate struct {
	Title            *string                 `db:"title" json:"title,omitempty"`
	Description      *string                 `db:"description" json:"description,omitempty"`
	Image            *string                 `db:"image" json:"image,omitempty"`
	Type             *string                 `db:"type" json:"type,omitempty"`
	Config           *map[string]interface{} `db:"config" json:"config,omitempty"`
	Settings         *map[string]interface{} `db:"settings" json:"settings,omitempty"`
	Meta             *map[string]interface{} `db:"meta" json:"meta,omitempty"`
	Status           *string                 `db:"status" json:"status,omitempty"`
	Visibility       *string                 `db:"visibility" json:"visibility,omitempty"`
	TableCount       *int                    `db:"table_count" json:"table_count,omitempty"`
	RowCount         *int64                  `db:"row_count" json:"row_count,omitempty"`
	StorageUsedBytes *int64                  `db:"storage_used_bytes" json:"storage_used_bytes,omitempty"`
	UpdatedBy        string                  `db:"last_modified_by" json:"last_modified_by,omitempty"`
	UpdatedAt        time.Time               `db:"last_modified_time" json:"last_modified_time,omitempty"`
}

// Map converts BaseUpdate → map[string]interface{} for DB update
func (b *BaseUpdate) Map() map[string]interface{} {
	result := make(map[string]interface{})
	if b.Title != nil {
		result["title"] = *b.Title
	}
	if b.Description != nil {
		result["description"] = *b.Description
	}
	if b.Image != nil {
		result["image"] = *b.Image
	}
	if b.Type != nil {
		result["type"] = *b.Type
	}
	if b.Config != nil {
		result["config"] = helpers.InterfaceToJSONString(*b.Config)
	}
	if b.Settings != nil {
		result["settings"] = helpers.InterfaceToJSONString(*b.Settings)
	}
	if b.Meta != nil {
		result["meta"] = helpers.InterfaceToJSONString(*b.Meta)
	}
	if b.Status != nil {
		result["status"] = *b.Status
	}
	if b.Visibility != nil {
		result["visibility"] = *b.Visibility
	}
	if b.TableCount != nil {
		result["table_count"] = *b.TableCount
	}
	if b.RowCount != nil {
		result["row_count"] = *b.RowCount
	}
	if b.StorageUsedBytes != nil {
		result["storage_used_bytes"] = *b.StorageUsedBytes
	}
	if b.UpdatedBy != "" {
		result["last_modified_by"] = b.UpdatedBy
	}
	result["last_modified_time"] = b.UpdatedAt
	return result
}

type CreateBaseRequest struct {
	Title       string  `db:"title" json:"title,omitempty"`
	Description *string `db:"description" json:"description,omitempty"`
	WorkspaceID string  `db:"workspace_id" json:"workspace_id,omitempty"`
	CreatedBy   string  `json:"created_by,omitempty"`
}

type BaseResponse struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	WorkspaceID string    `db:"workspace_id" json:"workspace_id,omitempty" mapstructure:"workspace_id"`
	Title       string    `db:"title" json:"title,omitempty" mapstructure:"title"`
	Description *string   `db:"description" json:"description,omitempty" mapstructure:"description"`

	// Database connection (for external sources)
	Type   string                 `db:"type" json:"type,omitempty" mapstructure:"type"`
	Config map[string]interface{} `db:"config" json:"config,omitempty" mapstructure:"config"`

	// Settings and metadata
	Settings map[string]interface{} `db:"settings" json:"settings,omitempty" mapstructure:"settings"`
	Meta     map[string]interface{} `db:"meta" json:"meta,omitempty" mapstructure:"meta"`

	// Status and visibility
	Status     string `db:"status" json:"status,omitempty" mapstructure:"status"`
	Visibility string `db:"visibility" json:"visibility,omitempty" mapstructure:"visibility"`

	// Resource tracking
	TableCount       int   `db:"table_count" json:"table_count,omitempty" mapstructure:"table_count"`
	RowCount         int64 `db:"row_count" json:"row_count,omitempty" mapstructure:"row_count"`
	StorageUsedBytes int64 `db:"storage_used_bytes" json:"storage_used_bytes,omitempty" mapstructure:"storage_used_bytes"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`

	AccessLevel string `db:"access_level" json:"access_level" mapstructure:"access_level"`

	Tables []TableResponse `db:"tables" json:"tables" mapstructure:"tables"`
}
