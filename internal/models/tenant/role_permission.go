package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

// RolePermission maps roles to their permissions
// This represents which permissions are granted to each role
type RolePermission struct {
	ID           uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	RoleID       uuid.UUID `db:"role_id" json:"role_id,omitempty" mapstructure:"role_id"`
	PermissionID uuid.UUID `db:"permission_id" json:"permission_id,omitempty" mapstructure:"permission_id"`
	CreatedAt    time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
}

func (RolePermission) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".role_permissions", prefix)
}

// createRolePermissionFK creates a foreign key definition for role_permissions table
func createRolePermissionFK(prefix, column, table string) models.ForeignKeyDef {
	return models.ForeignKeyDef{
		Name:              fmt.Sprintf("fk_role_permissions_%s", column),
		Columns:           []string{column},
		ReferencedTable:   fmt.Sprintf("\"%s\".%s", prefix, table),
		ReferencedColumns: []string{"id"},
		OnDelete:          "CASCADE",
	}
}

func (tbl RolePermission) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			createUUIDIDColumn(),
			{Name: "role_id", DataType: "uuid", NotNull: true},
			{Name: "permission_id", DataType: "uuid", NotNull: true},
			createTimestampColumn("created_time", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_role_permissions_role_id", Columns: []string{"role_id"}},
			{Name: "idx_role_permissions_permission_id", Columns: []string{"permission_id"}},
			{Name: "idx_role_permissions_role_permission", Columns: []string{"role_id", "permission_id"}, Unique: true},
		},
		ForeignKeys: []models.ForeignKeyDef{
			createRolePermissionFK(prefix, "role_id", "access_roles"),
			createRolePermissionFK(prefix, "permission_id", "permissions"),
		},
	}
}
