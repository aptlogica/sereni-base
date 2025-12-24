package master

import (
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

// Permission represents a permission = resource × action combination
// Example: workspace.read, workspace.create, base.update, records.delete, etc.
type Permission struct {
	ID         uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	ResourceID uuid.UUID `db:"resource_id" json:"resource_id,omitempty" mapstructure:"resource_id"`
	ActionID   uuid.UUID `db:"action_id" json:"action_id,omitempty" mapstructure:"action_id"`
	CreatedAt  time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
}

func (Permission) TableName() string {
	return "\"master\".permissions"
}

func (tbl Permission) TableSchema() models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "resource_id", DataType: "uuid", NotNull: true},
			{Name: "action_id", DataType: "uuid", NotNull: true},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_permissions_resource_id", Columns: []string{"resource_id"}},
			{Name: "idx_permissions_action_id", Columns: []string{"action_id"}},
			{Name: "idx_permissions_resource_action", Columns: []string{"resource_id", "action_id"}, Unique: true},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_permissions_resource_id",
				Columns:           []string{"resource_id"},
				ReferencedTable:   "\"master\".resources",
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
			{
				Name:              "fk_permissions_action_id",
				Columns:           []string{"action_id"},
				ReferencedTable:   "\"master\".actions",
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
