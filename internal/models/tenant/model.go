// Copyright (c) 2026 Aptlogica Technologies Private Limited
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

type Model struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	BaseID      uuid.UUID `db:"base_id" json:"base_id,omitempty" mapstructure:"base_id"`
	WorkspaceID uuid.UUID `db:"workspace_id" json:"workspace_id,omitempty" mapstructure:"workspace_id"`

	Title       string  `db:"title" json:"title,omitempty" mapstructure:"title"`
	Description *string `db:"description" json:"description,omitempty" mapstructure:"description"`
	Alias       string  `db:"alias" json:"alias,omitempty" mapstructure:"alias"`
	Type        string  `db:"type" json:"type,omitempty" mapstructure:"type"`

	// Schema and metadata
	Meta   map[string]interface{} `db:"meta" json:"meta,omitempty" mapstructure:"meta"`
	Schema *string                `db:"schema" json:"schema,omitempty" mapstructure:"schema"`

	// Status flags
	Enabled bool `db:"enabled" json:"enabled,omitempty" mapstructure:"enabled"`
	MM      bool `db:"mm" json:"mm,omitempty" mapstructure:"mm"` // many-to-many junction table
	Pinned  bool `db:"pinned" json:"pinned,omitempty" mapstructure:"pinned"`
	Deleted bool `db:"deleted" json:"deleted,omitempty" mapstructure:"deleted"`

	// Organization
	Tags       *string  `db:"tags" json:"tags,omitempty" mapstructure:"tags"`
	OrderIndex *float64 `db:"order_index" json:"order_index,omitempty" mapstructure:"order_index"`

	// Resource tracking
	RowCount         int64 `db:"row_count" json:"row_count,omitempty" mapstructure:"row_count"`
	ColumnCount      int   `db:"column_count" json:"column_count,omitempty" mapstructure:"column_count"`
	StorageUsedBytes int64 `db:"storage_used_bytes" json:"storage_used_bytes,omitempty" mapstructure:"storage_used_bytes"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (Model) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".models", prefix)
}

func (tbl Model) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "base_id", DataType: "varchar", NotNull: true},
			{Name: "workspace_id", DataType: "varchar", NotNull: true},
			{Name: "title", DataType: "varchar", NotNull: true},
			{Name: "description", DataType: "text"},
			{Name: "alias", DataType: "varchar", NotNull: true},
			{Name: "type", DataType: "varchar", DefaultValue: StrPtr("'table'")},
			{Name: "meta", DataType: "jsonb"},
			{Name: "schema", DataType: "text"},
			{Name: "enabled", DataType: "boolean", DefaultValue: StrPtr("true")},
			{Name: "mm", DataType: "boolean", DefaultValue: StrPtr("false")},
			{Name: "pinned", DataType: "boolean", DefaultValue: StrPtr("false")},
			{Name: "deleted", DataType: "boolean", DefaultValue: StrPtr("false")},
			{Name: "tags", DataType: "varchar"},
			{Name: "order_index", DataType: "real"},
			{Name: "row_count", DataType: "integer", DefaultValue: StrPtr("0")},
			{Name: "column_count", DataType: "integer", DefaultValue: StrPtr("0")},
			{Name: "storage_used_bytes", DataType: "integer", DefaultValue: StrPtr("0")},
			{Name: "created_by", DataType: "varchar"},
			{Name: "last_modified_by", DataType: "varchar"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_models_base_id", Columns: []string{"base_id"}},
			{Name: "idx_models_workspace_id", Columns: []string{"workspace_id"}},
			{Name: "idx_models_alias", Columns: []string{"alias"}},
			{Name: "idx_models_type", Columns: []string{"type"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			createModelFK(prefix, "base_id", "bases"),
			createModelFK(prefix, "workspace_id", "workspaces"),
		},
	}
}

// createModelFK creates a foreign key definition for models table
func createModelFK(prefix, column, table string) models.ForeignKeyDef {
	return models.ForeignKeyDef{
		Name:              fmt.Sprintf("fk_models_%s", column),
		Columns:           []string{column},
		ReferencedTable:   fmt.Sprintf("\"%s\".%s", prefix, table),
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
	}
}
