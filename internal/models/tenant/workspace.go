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

type Workspace struct {
	ID          uuid.UUID              `db:"id" json:"id" mapstructure:"id"`
	Title       string                 `db:"title" json:"title" mapstructure:"title"`
	Description *string                `db:"description" json:"description" mapstructure:"description"`
	Slug        string                 `db:"slug" json:"slug" mapstructure:"slug"`
	Meta        map[string]interface{} `db:"meta" json:"meta" mapstructure:"meta"`
	IsDefault   bool                   `db:"is_default" json:"is_default" mapstructure:"is_default"`
	Status      string                 `db:"status" json:"status" mapstructure:"status"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`

	CreatedAt time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
}

func (Workspace) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".workspaces", prefix)
}

func (tbl Workspace) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "title", DataType: "varchar", NotNull: true},
			{Name: "description", DataType: "text"},
			{Name: "slug", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "meta", DataType: "jsonb"},
			CreateBooleanColumn("is_default"),
			{Name: "status", DataType: "varchar", DefaultValue: StrPtr("'active'")},
			{Name: "created_by", DataType: "varchar"},
			{Name: "last_modified_by", DataType: "varchar"},
			CreateTimestampColumn("created_time", true, false),
			CreateTimestampColumn("last_modified_time", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_workspaces_slug", Columns: []string{"slug"}},
			{Name: "idx_workspaces_status", Columns: []string{"status"}},
		},
	}
}
