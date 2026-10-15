// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"strings"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/google/uuid"
)

// Constraint types stored in table_constraints.type
const (
	ConstraintTypeUnique = "unique"
	ConstraintTypeCheck  = "check"
)

// TableConstraint describes a constraint a user added to a table (model). Postgres enforces the
// rule; this row keeps what Postgres doesn't: the user-facing title and who made it.
type TableConstraint struct {
	ID      uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	ModelID string    `db:"model_id" json:"model_id,omitempty" mapstructure:"model_id"`
	Type    string    `db:"type" json:"type,omitempty" mapstructure:"type"`
	Title   string    `db:"title" json:"title,omitempty" mapstructure:"title"`
	// Name of the constraint in Postgres, e.g. uq_employee_3f9a2c1b
	ConstraintName string `db:"constraint_name" json:"constraint_name,omitempty" mapstructure:"constraint_name"`
	// Columns of the key, in order, by columns.id so renaming a field doesn't break it
	ColumnIDs []string `db:"column_ids" json:"column_ids" mapstructure:"column_ids"`

	CreatedBy string    `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string    `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`
	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (TableConstraint) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".table_constraints", prefix)
}

func (tbl TableConstraint) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "model_id", DataType: "varchar", NotNull: true},
			{Name: "type", DataType: "varchar", NotNull: true},
			{Name: "title", DataType: "varchar", NotNull: true},
			{Name: "constraint_name", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "column_ids", DataType: "jsonb", NotNull: true},
			{Name: "created_by", DataType: "varchar"},
			{Name: "last_modified_by", DataType: "varchar"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_table_constraints_model_id", Columns: []string{"model_id"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_table_constraints_model_id",
				Columns:           []string{"model_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".models", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}

// Field types that can't be part of a key, by lower-cased uidt: computed, multi-valued or file fields
const systemFieldBlockReason = "system fields"

var keyBlockedTypes = map[string]string{
	"links":            "link fields",
	"lookup":           "lookup fields",
	"formula":          "formula fields",
	"attachment":       "attachments",
	"json":             "JSON fields",
	"multiselect":      "multi select fields",
	"rating":           "rating fields",
	"createdtime":      systemFieldBlockReason,
	"lastmodifiedtime": systemFieldBlockReason,
	"createdby":        systemFieldBlockReason,
	"lastmodifiedby":   systemFieldBlockReason,
}

// KeyColumnBlockReason says why a column can't be part of a key, or "" when it can.
// Title is a system column but allowed: links and lookups show it. Computed fields are
// recognised by type, not by the virtual flag, which is also set on columns made with a table.
func KeyColumnBlockReason(c Column) string {
	if c.ColumnName == "id" {
		return "it is already unique"
	}
	if kind, ok := keyBlockedTypes[strings.ToLower(c.UIDT)]; ok {
		return kind + " can't be part of a key"
	}
	if c.System && c.ColumnName != "title" {
		return "system fields can't be part of a key"
	}
	return ""
}

// HasDefaultValue reports whether the column has a user-set default (meta.defaultValue) that
// would be written to every new record that doesn't give a value.
func HasDefaultValue(c Column) bool {
	switch v := c.Meta["defaultValue"].(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(v) != ""
	case []interface{}:
		return len(v) > 0
	default:
		return true
	}
}

// UniqueDefaultConflictMessage is shown when a unique key and default values can't be combined
const UniqueDefaultConflictMessage = "This column has a UNIQUE constraint. Its default value may be duplicated when inserting multiple rows. Remove the default value or the UNIQUE constraint to avoid duplicate-value errors."
