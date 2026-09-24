// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"

	"github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

// Automation types stored in automations.type
const (
	AutomationTypeTrigger  = "trigger"
	AutomationTypeWebhook  = "webhook"
	AutomationTypeFunction = "function"
)

// Automation stores a trigger or webhook of a table (model).
// For a trigger, Context holds the full CREATE TRIGGER query (optionally preceded by its function);
// for a function, a single CREATE FUNCTION ... RETURNS trigger query.
type Automation struct {
	ID      uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	ModelID string    `db:"model_id" json:"model_id,omitempty" mapstructure:"model_id"`
	Title   string    `db:"title" json:"title,omitempty" mapstructure:"title"`
	Type    string    `db:"type" json:"type,omitempty" mapstructure:"type"`
	Context string    `db:"context" json:"context,omitempty" mapstructure:"context"`
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
