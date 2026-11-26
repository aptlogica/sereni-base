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

// BaseAPI holds the public API settings of a base. A row exists only once the API
// has been enabled for that base.
type BaseAPI struct {
	BaseID        string     `db:"base_id" json:"base_id" mapstructure:"base_id"`
	PublicID      string     `db:"public_id" json:"public_id" mapstructure:"public_id"`
	Enabled       bool       `db:"enabled" json:"enabled" mapstructure:"enabled"`
	EnabledAt     *time.Time `db:"enabled_at" json:"enabled_at,omitempty" mapstructure:"enabled_at"`
	SchemaVersion int64      `db:"schema_version" json:"schema_version" mapstructure:"schema_version"`
	// Settings holds one key per feature, e.g. base_management; a missing key means the feature is off.
	// It is read with its own query (platform.baseAPIService.settings), not mapped from the row.
	Settings map[string]interface{} `db:"settings" json:"settings" mapstructure:"-"`

	CreatedAt time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
}

func (BaseAPI) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".base_api", prefix)
}

func (tbl BaseAPI) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "base_id", DataType: "varchar", NotNull: true},
			{Name: "public_id", DataType: "varchar(16)", NotNull: true, Unique: true},
			{Name: "enabled", DataType: "boolean", NotNull: true, DefaultValue: StrPtr("false")},
			CreateTimestampColumn("enabled_at", false, true),
			// go-postgres-rest does not accept bigint here; integer is ample for a per-base counter
			{Name: "schema_version", DataType: "integer", NotNull: true, DefaultValue: StrPtr("1")},
			{Name: "settings", DataType: "jsonb", NotNull: true, DefaultValue: StrPtr("'{}'::jsonb")},
			CreateTimestampColumn("created_time", true, false),
			CreateTimestampColumn("last_modified_time", true, false),
		},
		PrimaryKey: []string{"base_id"},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_base_api_base_id",
				Columns:           []string{"base_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".bases", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}

// CheckConstraints returns the CHECK constraints for the table. go-postgres-rest only accepts
// single-column comparisons, so these are added with raw SQL after the table is created.
func (tbl BaseAPI) CheckConstraints(prefix string) []string {
	table := tbl.TableName(prefix)
	return []string{
		fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT base_api_public_id_format CHECK (public_id ~ '^bs_[0-9a-z]{10}$')`, table),
		// Databases created before settings existed get the column here; new ones already have it
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS settings jsonb NOT NULL DEFAULT '{}'::jsonb`, table),
		fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT base_api_settings_object CHECK (jsonb_typeof(settings) = 'object')`, table),
		fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT base_api_enabled_at CHECK (enabled = false OR enabled_at IS NOT NULL)`, table),
	}
}
