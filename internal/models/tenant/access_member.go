// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"github.com/aptlogica/go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

// AccessMember represents user-to-role assignment with scope awareness
// Allows assigning roles at different levels: system (admin), workspace, base
// The scope_id can be null for system-level, workspace_id for workspace-level, or base_id for base-level access
// For base-level access, workspace_id stores the workspace that the base belongs to
type AccessMember struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	UserID      string    `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
	ScopeType   string    `db:"scope_type" json:"scope_type,omitempty" mapstructure:"scope_type"` // system, workspace, base
	ScopeID     *string   `db:"scope_id" json:"scope_id,omitempty" mapstructure:"scope_id"`       // null for system, workspace_id or base_id otherwise
	RoleID      string    `db:"role_id" json:"role_id,omitempty" mapstructure:"role_id"`
	WorkspaceID *string   `db:"workspace_id" json:"workspace_id,omitempty" mapstructure:"workspace_id"` // workspace that owns this access (for base-level, stores workspace_id)
	AssignedBy  *string   `db:"assigned_by" json:"assigned_by,omitempty" mapstructure:"assigned_by"`    // who assigned this role
	CreatedAt   time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt   time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (AccessMember) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".access_members", prefix)
}

func (tbl AccessMember) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "user_id", DataType: "varchar", NotNull: true},
			{Name: "scope_type", DataType: "varchar", NotNull: true}, // system, workspace, base
			{Name: "scope_id", DataType: "varchar"},                  // null for system level, workspace_id or base_id otherwise
			{Name: "role_id", DataType: "varchar", NotNull: true},
			{Name: "workspace_id", DataType: "varchar"}, // workspace owner (for base-level records stores workspace_id)
			{Name: "assigned_by", DataType: "varchar"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_access_members_user_id", Columns: []string{"user_id"}},
			{Name: "idx_access_members_scope", Columns: []string{"scope_type", "scope_id"}},
			{Name: "idx_access_members_role_id", Columns: []string{"role_id"}},
			{Name: "idx_access_members_workspace", Columns: []string{"workspace_id"}},
			{Name: "idx_access_members_user_scope_role", Columns: []string{"user_id", "scope_type", "scope_id", "role_id"}, Unique: true},
		},
	}
}
