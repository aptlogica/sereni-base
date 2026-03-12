// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type FeatureFlag struct {
	ID             uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Name           string    `db:"name" json:"name,omitempty" mapstructure:"name"`
	Description    *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	DefaultEnabled bool      `db:"default_enabled" json:"default_enabled,omitempty" mapstructure:"default_enabled"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (FeatureFlag) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".shared.feature_flags", prefix)
}

func (tbl FeatureFlag) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "name", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "description", DataType: "text"},
			{Name: "default_enabled", DataType: "boolean", DefaultValue: StrPtr("false")},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_feature_flags_name", Columns: []string{"name"}},
		},
	}
}
