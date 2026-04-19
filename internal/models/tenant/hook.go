// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

type Hook struct {
	ID      uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	ModelID *string   `db:"model_id" json:"model_id,omitempty" mapstructure:"model_id"`
	BaseID  string    `db:"base_id" json:"base_id,omitempty" mapstructure:"base_id"`

	Title       string  `db:"title" json:"title,omitempty" mapstructure:"title"`
	Description *string `db:"description" json:"description,omitempty" mapstructure:"description"`
	Type        string  `db:"type" json:"type,omitempty" mapstructure:"type"`
	Event       string  `db:"event" json:"event,omitempty" mapstructure:"event"`
	Operation   string  `db:"operation" json:"operation,omitempty" mapstructure:"operation"`

	// Configuration
	URL       *string `db:"url" json:"url,omitempty" mapstructure:"url"`
	Headers   *string `db:"headers" json:"headers,omitempty" mapstructure:"headers"`
	Payload   *string `db:"payload" json:"payload,omitempty" mapstructure:"payload"`
	Condition *string `db:"condition" json:"condition,omitempty" mapstructure:"condition"`

	// Settings
	AsyncProcessing bool `db:"async_processing" json:"async_processing,omitempty" mapstructure:"async_processing"`
	Retries         int  `db:"retries" json:"retries,omitempty" mapstructure:"retries"`
	RetryInterval   int  `db:"retry_interval" json:"retry_interval,omitempty" mapstructure:"retry_interval"`
	Timeout         int  `db:"timeout" json:"timeout,omitempty" mapstructure:"timeout"`
	Active          bool `db:"active" json:"active,omitempty" mapstructure:"active"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (Hook) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".hooks", prefix)
}

// CreateBooleanColumnTrue creates a boolean column with default true
func CreateBooleanColumnTrue(name string) models.ColumnDefinition {
	return models.ColumnDefinition{Name: name, DataType: "boolean", DefaultValue: StrPtr("true")}
}

func (tbl Hook) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "model_id", DataType: "varchar"},
			{Name: "base_id", DataType: "varchar", NotNull: true},
			{Name: "title", DataType: "varchar", NotNull: true},
			{Name: "description", DataType: "text"},
			{Name: "type", DataType: "varchar", NotNull: true},
			{Name: "event", DataType: "varchar", NotNull: true},
			{Name: "operation", DataType: "varchar", NotNull: true},
			{Name: "url", DataType: "varchar"},
			{Name: "headers", DataType: "text"},
			{Name: "payload", DataType: "text"},
			{Name: "condition", DataType: "text"},
			CreateBooleanColumnTrue("async_processing"),
			{Name: "retries", DataType: "integer", DefaultValue: StrPtr("3")},
			{Name: "retry_interval", DataType: "integer", DefaultValue: StrPtr("60")},
			{Name: "timeout", DataType: "integer", DefaultValue: StrPtr("30")},
			CreateBooleanColumnTrue("active"),
			CreateTimestampColumn("created_time", true, false),
			CreateTimestampColumn("last_modified_time", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_hooks_base_id", Columns: []string{"base_id"}},
			{Name: "idx_hooks_model_id", Columns: []string{"model_id"}},
			{Name: "idx_hooks_type", Columns: []string{"type"}},
			{Name: "idx_hooks_event", Columns: []string{"event"}},
			{Name: "idx_hooks_active", Columns: []string{"active"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_hooks_base_id",
				Columns:           []string{"base_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".bases", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
			{
				Name:              "fk_hooks_model_id",
				Columns:           []string{"model_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".models", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
