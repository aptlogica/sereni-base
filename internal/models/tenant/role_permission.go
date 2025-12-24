package tenant

import (
	"fmt"
	"godbgrest/pkg/models"
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

func (tbl RolePermission) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "role_id", DataType: "uuid", NotNull: true},
			{Name: "permission_id", DataType: "uuid", NotNull: true},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_role_permissions_role_id", Columns: []string{"role_id"}},
			{Name: "idx_role_permissions_permission_id", Columns: []string{"permission_id"}},
			{Name: "idx_role_permissions_role_permission", Columns: []string{"role_id", "permission_id"}, Unique: true},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_role_permissions_role_id",
				Columns:           []string{"role_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".access_roles", "SCHEMA_PREFIX"),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
			{
				Name:              "fk_role_permissions_permission_id",
				Columns:           []string{"permission_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".permissions", "SCHEMA_PREFIX"),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
