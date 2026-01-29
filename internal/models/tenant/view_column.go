package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type ViewColumn struct {
	ID       uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	ViewID   string    `db:"view_id" json:"view_id,omitempty" mapstructure:"view_id"`
	ColumnID string    `db:"column_id" json:"column_id,omitempty" mapstructure:"column_id"`

	ShowColumn bool     `db:"show_column" json:"show_column,omitempty" mapstructure:"show_column"`
	OrderIndex *float64 `db:"order_index" json:"order_index,omitempty" mapstructure:"order_index"`
	Width      *string  `db:"width" json:"width,omitempty" mapstructure:"width"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (ViewColumn) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".view_columns", prefix)
}

// createViewColumnFK creates a foreign key definition for view_columns table
func createViewColumnFK(prefix, column, table string) models.ForeignKeyDef {
	return models.ForeignKeyDef{
		Name:              fmt.Sprintf("fk_view_columns_%s", column),
		Columns:           []string{column},
		ReferencedTable:   fmt.Sprintf("\"%s\".%s", prefix, table),
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
	}
}

func (tbl ViewColumn) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "view_id", DataType: "varchar", NotNull: true},
			{Name: "column_id", DataType: "varchar", NotNull: true},
			{Name: "show_column", DataType: "boolean", DefaultValue: strPtr("true")},
			{Name: "order_index", DataType: "real"},
			{Name: "width", DataType: "varchar"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_view_columns_view_id", Columns: []string{"view_id"}},
			{Name: "idx_view_columns_column_id", Columns: []string{"column_id"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			createViewColumnFK(prefix, "view_id", "views"),
			createViewColumnFK(prefix, "column_id", "columns"),
		},
	}
}
