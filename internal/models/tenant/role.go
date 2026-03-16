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

type Role struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Name        string    `db:"name" json:"name,omitempty" mapstructure:"name"`
	Description *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	IsDefault   bool      `db:"is_default" json:"is_default,omitempty" mapstructure:"is_default"`
	CreatedAt   time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt   time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (Role) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".roles", prefix)
}

func (tbl Role) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			CreateUUIDIDColumn(),
			{Name: "name", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "description", DataType: "text"},
			{Name: "is_default", DataType: "boolean", NotNull: true, DefaultValue: StrPtr("false")},
			CreateTimestampColumn("created_time", true, false),
			CreateTimestampColumn("last_modified_time", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_roles_name", Columns: []string{"name"}},
		},
	}
}
