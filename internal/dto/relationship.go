package dto

import (
	"time"

	"github.com/google/uuid"
)

type RelationInsertion struct {
	ID     uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	BaseID string    `db:"base_id" json:"base_id,omitempty" mapstructure:"base_id"`

	SourceModelID       string   `db:"source_model_id" json:"source_model_id,omitempty" mapstructure:"source_model_id"`
	SourceColumnID      string   `db:"source_column_id" json:"source_column_id,omitempty" mapstructure:"source_column_id"`
	SourceLookupColumns []string `db:"source_lookup_columns" json:"source_lookup_columns,omitempty" mapstructure:"source_lookup_columns"`

	// Target side (referenced table)
	TargetModelID       string   `db:"target_model_id" json:"target_model_id,omitempty" mapstructure:"target_model_id"`
	TargetColumnID      string   `db:"target_column_id" json:"target_column_id,omitempty" mapstructure:"target_column_id"`
	TargetLookupColumns []string `db:"target_lookup_columns" json:"target_lookup_columns,omitempty" mapstructure:"target_lookup_columns"`

	// Relationship type and rules
	RelationType string `db:"relation_type" json:"relation_type,omitempty" mapstructure:"relation_type"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (r *RelationInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                    r.ID,
		"base_id":               r.BaseID,
		"source_model_id":       r.SourceModelID,
		"source_column_id":      r.SourceColumnID,
		"target_model_id":       r.TargetModelID,
		"target_column_id":      r.TargetColumnID,
		"source_lookup_columns": r.SourceLookupColumns,
		"target_lookup_columns": r.TargetLookupColumns,
		"relation_type":         r.RelationType,
		"created_time":            r.CreatedAt,
		"last_modified_time":            r.UpdatedAt,
	}
}


type RelationUpdate struct {
	SourceLookupColumns interface{} `db:"source_lookup_columns" json:"source_lookup_columns,omitempty" mapstructure:"source_lookup_columns"`
	TargetLookupColumns interface{} `db:"target_lookup_columns" json:"target_lookup_columns,omitempty" mapstructure:"target_lookup_columns"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (r *RelationUpdate) Map() map[string]interface{} {
	m := map[string]interface{}{}
	if r.SourceLookupColumns != nil {
		m["source_lookup_columns"] = r.SourceLookupColumns
	}
	if r.TargetLookupColumns != nil {
		m["target_lookup_columns"] = r.TargetLookupColumns
	}
	m["last_modified_time"] = r.UpdatedAt
	return m
}
