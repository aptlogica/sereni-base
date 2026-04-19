// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

type WorkspaceMember struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	WorkspaceID string    `db:"workspace_id" json:"workspace_id,omitempty" mapstructure:"workspace_id"`
	UserID      string    `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
	BasesIds    string    `db:"bases_ids" json:"bases_ids,omitempty" mapstructure:"bases_ids"`
	AccessLevel string    `db:"access_level" json:"access_level,omitempty" mapstructure:"access_level"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (WorkspaceMember) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".workspace_members", prefix)
}

func (tbl WorkspaceMember) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "workspace_id", DataType: "varchar", NotNull: true},
			{Name: "user_id", DataType: "varchar", NotNull: true},
			{Name: "access_level", DataType: "varchar", NotNull: true},
			{Name: "bases_ids", DataType: "text"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_workspace_members_workspace_user", Columns: []string{"workspace_id", "user_id"}, Unique: true},
			{Name: "idx_workspace_members_user_id", Columns: []string{"user_id"}},
			{Name: "idx_workspace_members_access_level", Columns: []string{"access_level"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_workspace_members_workspace_id",
				Columns:           []string{"workspace_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".workspaces", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
