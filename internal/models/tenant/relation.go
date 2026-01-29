package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"serenibase/internal/constant"

	"github.com/google/uuid"
)

type Relation struct {
	ID     uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	BaseID string    `db:"base_id" json:"base_id,omitempty" mapstructure:"base_id"`

	// Source side (foreign key table)
	SourceModelID       string   `db:"source_model_id" json:"source_model_id,omitempty" mapstructure:"source_model_id"`
	SourceColumnID      string   `db:"source_column_id" json:"source_column_id,omitempty" mapstructure:"source_column_id"`
	SourceLookupColumns []string `db:"source_lookup_columns" json:"source_lookup_columns,omitempty" mapstructure:"source_lookup_columns"`

	// Target side (referenced table)
	TargetModelID       string   `db:"target_model_id" json:"target_model_id,omitempty" mapstructure:"target_model_id"`
	TargetColumnID      string   `db:"target_column_id" json:"target_column_id,omitempty" mapstructure:"target_column_id"`
	TargetLookupColumns []string `db:"target_lookup_columns" json:"target_lookup_columns,omitempty" mapstructure:"target_lookup_columns"`

	// Relationship type and rules
	RelationType string `db:"relation_type" json:"relation_type,omitempty" mapstructure:"relation_type"`
	UpdateRule   string `db:"update_rule" json:"update_rule,omitempty" mapstructure:"update_rule"`
	DeleteRule   string `db:"delete_rule" json:"delete_rule,omitempty" mapstructure:"delete_rule"`

	// Many-to-many junction table (if applicable)
	JunctionModelID *string `db:"junction_model_id" json:"junction_model_id,omitempty" mapstructure:"junction_model_id"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (Relation) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".relations", prefix)
}

// createRelationFK creates a foreign key definition for relations table
func createRelationFK(prefix, column, table string) models.ForeignKeyDef {
	return models.ForeignKeyDef{
		Name:              fmt.Sprintf("fk_relations_%s", column),
		Columns:           []string{column},
		ReferencedTable:   fmt.Sprintf("\"%s\".%s", prefix, table),
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
	}
}

// createRelationModelFK creates a foreign key definition for model references in relations table
func createRelationModelFK(prefix, column string) models.ForeignKeyDef {
	return models.ForeignKeyDef{
		Name:              fmt.Sprintf("fk_relations_%s", column),
		Columns:           []string{column},
		ReferencedTable:   fmt.Sprintf(constant.ModelsTableFormat, prefix),
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
	}
}

func (tbl Relation) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "base_id", DataType: "varchar", NotNull: true},
			{Name: "source_model_id", DataType: "varchar", NotNull: true},
			{Name: "source_column_id", DataType: "varchar", NotNull: true},
			{Name: "target_model_id", DataType: "varchar", NotNull: true},
			{Name: "target_column_id", DataType: "varchar", NotNull: true},
			{Name: "relation_type", DataType: "varchar", NotNull: true},
			{Name: "update_rule", DataType: "varchar", DefaultValue: strPtr("'CASCADE'")},
			{Name: "delete_rule", DataType: "varchar", DefaultValue: strPtr("'RESTRICT'")},
			{Name: "junction_model_id", DataType: "varchar"},
			{Name: "source_lookup_columns", DataType: "text[]", DefaultValue: strPtr("null")},
			{Name: "target_lookup_columns", DataType: "text[]", DefaultValue: strPtr("null")},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_relations_base_id", Columns: []string{"base_id"}},
			{Name: "idx_relations_source_model", Columns: []string{"source_model_id"}},
			{Name: "idx_relations_target_model", Columns: []string{"target_model_id"}},
			{Name: "idx_relations_type", Columns: []string{"relation_type"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			createRelationFK(prefix, "base_id", "bases"),
			createRelationModelFK(prefix, "source_model_id"),
			createRelationModelFK(prefix, "target_model_id"),
			createRelationFK(prefix, "source_column_id", "columns"),
			createRelationFK(prefix, "target_column_id", "columns"),
			createRelationModelFK(prefix, "junction_model_id"),
		},
	}
}
