// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg/models"
)

// WorkspaceAPI holds the public API settings of a workspace. A row exists only once the API
// has been enabled for that workspace.
type WorkspaceAPI struct {
	WorkspaceID   string     `db:"workspace_id" json:"workspace_id" mapstructure:"workspace_id"`
	PublicID      string     `db:"public_id" json:"public_id" mapstructure:"public_id"`
	Enabled       bool       `db:"enabled" json:"enabled" mapstructure:"enabled"`
	EnabledAt     *time.Time `db:"enabled_at" json:"enabled_at,omitempty" mapstructure:"enabled_at"`
	SchemaVersion int64      `db:"schema_version" json:"schema_version" mapstructure:"schema_version"`

	CreatedAt time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
}

func (WorkspaceAPI) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".workspace_api", prefix)
}

func (tbl WorkspaceAPI) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "workspace_id", DataType: "varchar", NotNull: true},
			{Name: "public_id", DataType: "varchar(16)", NotNull: true, Unique: true},
			{Name: "enabled", DataType: "boolean", NotNull: true, DefaultValue: StrPtr("false")},
			CreateTimestampColumn("enabled_at", false, true),
			// go-postgres-rest does not accept bigint here; integer is ample for a per-workspace counter
			{Name: "schema_version", DataType: "integer", NotNull: true, DefaultValue: StrPtr("1")},
			CreateTimestampColumn("created_time", true, false),
			CreateTimestampColumn("last_modified_time", true, false),
		},
		PrimaryKey: []string{"workspace_id"},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_workspace_api_workspace_id",
				Columns:           []string{"workspace_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".workspaces", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}

// CheckConstraints returns the CHECK constraints for the table. go-postgres-rest only accepts
// single-column comparisons, so these are added with raw SQL after the table is created.
func (tbl WorkspaceAPI) CheckConstraints(prefix string) []string {
	table := tbl.TableName(prefix)
	return []string{
		fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT workspace_api_public_id_format CHECK (public_id ~ '^ws_[0-9a-z]{10}$')`, table),
		fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT workspace_api_enabled_at CHECK (enabled = false OR enabled_at IS NOT NULL)`, table),
	}
}
