package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
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

func (Permission) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".permissions", prefix)
}

// createPermissionFK creates a foreign key definition for permission table
func createPermissionFK(prefix, column, table string) models.ForeignKeyDef {
	return models.ForeignKeyDef{
		Name:              fmt.Sprintf("fk_permissions_%s", column),
		Columns:           []string{column},
		ReferencedTable:   fmt.Sprintf("\"%s\".%s", prefix, table),
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
	}
}

func (tbl Permission) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			createIDColumn(),
			{Name: "resource_id", DataType: "uuid", NotNull: true},
			{Name: "action_id", DataType: "uuid", NotNull: true},
			createTimestampColumn("created_time", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_permissions_resource_id", Columns: []string{"resource_id"}},
			{Name: "idx_permissions_action_id", Columns: []string{"action_id"}},
			{Name: "idx_permissions_resource_action", Columns: []string{"resource_id", "action_id"}, Unique: true},
		},
		ForeignKeys: []models.ForeignKeyDef{
			createPermissionFK(prefix, "resource_id", "resources"),
			createPermissionFK(prefix, "action_id", "actions"),
		},
	}
}
