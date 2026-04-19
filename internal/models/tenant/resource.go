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

// Resource represents a system resource that can be accessed
// Examples: workspace, base, records, members, settings, etc.
type Resource struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Code        string    `db:"code" json:"code,omitempty" mapstructure:"code"` // workspace, base, records, members, etc.
	Description *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	CreatedAt   time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
}

func (Resource) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".resources", prefix)
}

func (tbl Resource) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "code", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "description", DataType: "text"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_resources_code", Columns: []string{"code"}},
		},
	}
}
