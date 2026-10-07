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

// Automation types stored in automations.type
const (
	AutomationTypeTrigger  = "trigger"
	AutomationTypeWebhook  = "webhook"
	AutomationTypeFunction = "function"
)

// Automation stores a trigger, function or webhook of a table (model).
// For a trigger, Context holds one CREATE TRIGGER query; for a function,
// one CREATE FUNCTION ... RETURNS trigger query.
type Automation struct {
	ID      uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	ModelID string    `db:"model_id" json:"model_id,omitempty" mapstructure:"model_id"`
	Title   string    `db:"title" json:"title,omitempty" mapstructure:"title"`
	Type    string    `db:"type" json:"type,omitempty" mapstructure:"type"`
	Context string    `db:"context" json:"context,omitempty" mapstructure:"context"`

	CreatedBy string    `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string    `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`
	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (Automation) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".automations", prefix)
}

func (tbl Automation) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "model_id", DataType: "varchar", NotNull: true},
			{Name: "title", DataType: "varchar"},
			{Name: "type", DataType: "varchar", NotNull: true},
			{Name: "context", DataType: "text", NotNull: true},
			{Name: "created_by", DataType: "varchar"},
			{Name: "last_modified_by", DataType: "varchar"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_automations_model_id", Columns: []string{"model_id"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_automations_model_id",
				Columns:           []string{"model_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".models", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
