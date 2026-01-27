package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

// AccessRole represents a role with scope-based access control
// Scopes: system (global), workspace, base
// Examples: owner, co-owner, maintainer, member, viewer
// WorkspaceID is only populated for base-level roles to track which workspace the base belongs to
type AccessRole struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Name        string    `db:"name" json:"name,omitempty" mapstructure:"name"`
	ScopeLevel  string    `db:"scope_level" json:"scope_level,omitempty" mapstructure:"scope_level"` // system, workspace, base
	Priority    int       `db:"priority" json:"priority,omitempty" mapstructure:"priority"`          // higher = overrides lower
	Description *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	IsDefault   bool      `db:"is_default" json:"is_default,omitempty" mapstructure:"is_default"`
	WorkspaceID *string   `db:"workspace_id" json:"workspace_id,omitempty" mapstructure:"workspace_id"` // for base-level roles, stores workspace that owns the base
	CreatedAt   time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt   time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (AccessRole) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".access_roles", prefix)
}

func (tbl AccessRole) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "name", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "scope_level", DataType: "varchar", NotNull: true}, // system, workspace, base
			{Name: "priority", DataType: "integer", NotNull: true, DefaultValue: StrPtr("0")},
			{Name: "description", DataType: "varchar", NotNull: false},
			{Name: "is_default", DataType: "boolean", NotNull: true, DefaultValue: StrPtr("false")},
			{Name: "workspace_id", DataType: "varchar", NotNull: false}, // for base-level roles only
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_access_roles_name", Columns: []string{"name"}},
			{Name: "idx_access_roles_scope_level", Columns: []string{"scope_level"}},
			{Name: "idx_access_roles_priority", Columns: []string{"priority"}},
			{Name: "idx_access_roles_workspace", Columns: []string{"workspace_id"}},
		},
	}
}
