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

type View struct {
	ID      uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	ModelID string    `db:"model_id" json:"model_id,omitempty" mapstructure:"model_id"`
	BaseID  string    `db:"base_id" json:"base_id,omitempty" mapstructure:"base_id"`

	Title       string  `db:"title" json:"title,oxitempty" mapstructure:"title"`
	Alias       *string `db:"alias" json:"alias,omitempty" mapstructure:"alias"`
	Description *string `db:"description" json:"description,omitempty" mapstructure:"description"`
	Type        string  `db:"type" json:"type,omitempty" mapstructure:"type"`

	// View configuration
	IsDefault bool    `db:"is_default" json:"is_default,omitempty" mapstructure:"is_default"`
	LockType  *string `db:"lock_type" json:"lock_type,omitempty" mapstructure:"lock_type"`
	Password  *string `db:"password" json:"password,omitempty" mapstructure:"password"`

	// Sharing
	Public bool    `db:"public" json:"public,omitempty" mapstructure:"public"`
	UUID   *string `db:"uuid" json:"uuid,omitempty" mapstructure:"uuid"`

	// View settings
	Meta       map[string]interface{} `db:"meta" json:"meta,omitempty" mapstructure:"meta"`
	OrderIndex *float64               `db:"order_index" json:"order_index,omitempty" mapstructure:"order_index"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`
}

func (View) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".views", prefix)
}

func (tbl View) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "model_id", DataType: "varchar", NotNull: true},
			{Name: "base_id", DataType: "varchar", NotNull: true},
			{Name: "title", DataType: "varchar", NotNull: true},
			{Name: "alias", DataType: "varchar"},
			{Name: "description", DataType: "text"},
			{Name: "type", DataType: "varchar", DefaultValue: StrPtr("'grid'")},
			{Name: "is_default", DataType: "boolean", DefaultValue: StrPtr("false")},
			{Name: "lock_type", DataType: "varchar"},
			{Name: "password", DataType: "varchar"},
			{Name: "public", DataType: "boolean", DefaultValue: StrPtr("false")},
			{Name: "uuid", DataType: "varchar"},
			{Name: "meta", DataType: "jsonb"},
			{Name: "order_index", DataType: "real"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "created_by", DataType: "varchar"},
			{Name: "last_modified_by", DataType: "varchar"},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_views_model_id", Columns: []string{"model_id"}},
			{Name: "idx_views_base_id", Columns: []string{"base_id"}},
			{Name: "idx_views_type", Columns: []string{"type"}},
			{Name: "idx_views_public", Columns: []string{"public"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			createViewFK(prefix, "model_id", "models"),
			createViewFK(prefix, "base_id", "bases"),
		},
	}
}

// createViewFK creates a foreign key definition for views table
func createViewFK(prefix, column, table string) models.ForeignKeyDef {
	return models.ForeignKeyDef{
		Name:              fmt.Sprintf("fk_views_%s", column),
		Columns:           []string{column},
		ReferencedTable:   fmt.Sprintf("\"%s\".%s", prefix, table),
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
	}
}
