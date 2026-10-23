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

type Column struct {
	ID      uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	ModelID string    `db:"model_id" json:"model_id,omitempty" mapstructure:"model_id"`
	BaseID  string    `db:"base_id" json:"base_id,omitempty" mapstructure:"base_id"`

	// Column identification
	ColumnName string `db:"column_name" json:"column_name,omitempty" mapstructure:"column_name"`
	Title      string `db:"title" json:"title,omitempty" mapstructure:"title"`

	// Data type information
	UIDT string  `db:"uidt" json:"uidt,omitempty" mapstructure:"uidt"`
	DT   *string `db:"dt" json:"dt,omitempty" mapstructure:"dt"`

	Description *string                `db:"description" json:"description,omitempty" mapstructure:"description"`
	Meta        map[string]interface{} `db:"meta" json:"meta,omitempty" mapstructure:"meta"`

	// // Column properties
	// PK               bool `db:"pk" json:"pk,omitempty" mapstructure:"pk"`
	// PV               bool `db:"pv" json:"pv,omitempty" mapstructure:"pv"`
	// RQD              bool `db:"rqd" json:"rqd,omitempty" mapstructure:"rqd"`
	// UN               bool `db:"un" json:"un,omitempty" mapstructure:"un"`
	// AI               bool `db:"ai" json:"ai,omitempty" mapstructure:"ai"`
	// UniqueConstraint bool `db:"unique_constraint" json:"unique_constraint,omitempty" mapstructure:"unique_constraint"`

	// // Data type parameters
	// MaxLength      *string `db:"max_length" json:"max_length,omitempty" mapstructure:"max_length"`
	// PrecisionValue *string `db:"precision_value" json:"precision_value,omitempty" mapstructure:"precision_value"`
	// ScaleValue     *string `db:"scale_value" json:"scale_value,omitempty" mapstructure:"scale_value"`

	// // Default and validation
	// DefaultValue    *string `db:"default_value" json:"default_value,omitempty" mapstructure:"default_value"`
	// ValidationRules *string `db:"validation_rules" json:"validation_rules,omitempty" mapstructure:"validation_rules"`

	// Special column types
	Virtual bool `db:"virtual" json:"virtual,omitempty" mapstructure:"virtual"`
	System  bool `db:"system" json:"system,omitempty" mapstructure:"system"`
	Deleted bool `db:"deleted" json:"deleted,omitempty" mapstructure:"deleted"`

	// Display
	OrderIndex *float64 `db:"order_index" json:"order_index,omitempty" mapstructure:"order_index"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (Column) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".columns", prefix)
}

// createColumnFK creates a foreign key definition for columns table
func createColumnFK(prefix, column, table string) models.ForeignKeyDef {
	return models.ForeignKeyDef{
		Name:              fmt.Sprintf("fk_columns_%s", column),
		Columns:           []string{column},
		ReferencedTable:   fmt.Sprintf("\"%s\".%s", prefix, table),
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
	}
}

func (tbl Column) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "model_id", DataType: "varchar", NotNull: true},
			{Name: "base_id", DataType: "varchar", NotNull: true},
			{Name: "column_name", DataType: "varchar", NotNull: true},
			{Name: "title", DataType: "varchar", NotNull: true},
			{Name: "uidt", DataType: "varchar", NotNull: true},
			{Name: "description", DataType: "text"},
			{Name: "meta", DataType: "jsonb"},
			{Name: "dt", DataType: "varchar"},
			// {Name: "pk", DataType: "boolean", DefaultValue: StrPtr("false")},
			// {Name: "pv", DataType: "boolean", DefaultValue: StrPtr("false")},
			// {Name: "rqd", DataType: "boolean", DefaultValue: StrPtr("false")},
			// {Name: "un", DataType: "boolean", DefaultValue: StrPtr("false")},
			// {Name: "ai", DataType: "boolean", DefaultValue: StrPtr("false")},
			// {Name: "unique_constraint", DataType: "boolean", DefaultValue: StrPtr("false")},
			// {Name: "max_length", DataType: "varchar"},
			// {Name: "precision_value", DataType: "varchar"},
			// {Name: "scale_value", DataType: "varchar"},
			// {Name: "default_value", DataType: "text"},
			// {Name: "validation_rules", DataType: "text"},
			CreateBooleanColumn("virtual"),
			CreateBooleanColumn("system"),
			CreateBooleanColumn("deleted"),
			{Name: "order_index", DataType: "real"},
			{Name: "created_by", DataType: "varchar"},
			{Name: "last_modified_by", DataType: "varchar"},
			CreateTimestampColumn("created_time", true, false),
			CreateTimestampColumn("last_modified_time", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_columns_model_id", Columns: []string{"model_id"}},
			{Name: "idx_columns_base_id", Columns: []string{"base_id"}},
			{Name: "idx_columns_column_name", Columns: []string{"column_name"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			createColumnFK(prefix, "model_id", "models"),
			createColumnFK(prefix, "base_id", "bases"),
		},
	}
}

// LegacyTitleColumnName is the physical name of the Title column that tables created before
// Title became an ordinary column still have.
const LegacyTitleColumnName = "title"

// IsLegacyTitle reports whether the column is the system Title column of a table created before
// Title became an ordinary column. Such a column is writable and may be a key field until a
// migration turns it into a normal column.
func (c Column) IsLegacyTitle() bool {
	return c.System && c.ColumnName == LegacyTitleColumnName
}

// IsDisplayColumnType reports whether a field of this type can be the label of a record.
// Links, lookups, rollups, attachments and JSON have no single readable value.
func IsDisplayColumnType(uidt string) bool {
	switch strings.ToLower(uidt) {
	case "links", "lookup", "rollup", "attachment", "json":
		return false
	}
	return true
}

// IsDisplayColumn reports whether the column can label a record: a plain, live field. System fields
// are left out, except the Id column and the Title column of tables created before Title became an ordinary column.
func (c Column) IsDisplayColumn() bool {
	if c.Deleted || !IsDisplayColumnType(c.UIDT) {
		return false
	}
	return !c.System || c.IsLegacyTitle() || c.IsIDColumn()
}

// IsIDColumn reports whether the column is the system Id column every table has. A new link shows it
// as the label of linked records unless another column is chosen.
func (c Column) IsIDColumn() bool {
	return c.System && c.ColumnName == "id"
}

// IDColumn returns the system Id column of a table.
func IDColumn(columns []Column) (Column, bool) {
	for _, col := range columns {
		if col.IsIDColumn() && !col.Deleted {
			return col, true
		}
	}
	return Column{}, false
}

// IsTextColumn reports whether the column is a single or long text field that can label a record.
func (c Column) IsTextColumn() bool {
	uidt := strings.ToLower(c.UIDT)
	return (uidt == "text" || uidt == "longtext") && c.IsDisplayColumn()
}

// FirstTextColumn returns the first text column of a table by order index. It labels a record
// when no display column is chosen, and in messages about a record of its own table.
func FirstTextColumn(columns []Column) (Column, bool) {
	var first Column
	found := false
	for _, col := range columns {
		if !col.IsTextColumn() {
			continue
		}
		if !found || columnOrder(col) < columnOrder(first) {
			first, found = col, true
		}
	}
	return first, found
}

func columnOrder(c Column) float64 {
	if c.OrderIndex == nil {
		return 1e18
	}
	return *c.OrderIndex
}
