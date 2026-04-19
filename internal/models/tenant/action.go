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

// Action represents a system action that can be performed on resources
// Examples: read, create, update, delete, share, invite, etc.
type Action struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Code        string    `db:"code" json:"code,omitempty" mapstructure:"code"` // read, create, update, delete, share, etc.
	Description *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	CreatedAt   time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
}

func (Action) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".actions", prefix)
}

// createUUIDColumn creates a UUID column definition
func createUUIDColumn(name string, notNull, unique bool) models.ColumnDefinition {
	return models.ColumnDefinition{Name: name, DataType: "uuid", NotNull: notNull, Unique: unique}
}

func (tbl Action) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			createUUIDColumn("id", true, true),
			{Name: "code", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "description", DataType: "text"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_actions_code", Columns: []string{"code"}},
		},
	}
}
