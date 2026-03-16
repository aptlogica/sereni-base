package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRelationInsertionMap(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	sourceLookup := []string{"col1", "col2"}
	targetLookup := []string{"col3", "col4"}

	relation := dto.RelationInsertion{
		ID:                  id,
		BaseID:              "base-123",
		SourceModelID:       "model-123",
		SourceColumnID:      "col-123",
		SourceLookupColumns: sourceLookup,
		TargetModelID:       "model-456",
		TargetColumnID:      "col-456",
		TargetLookupColumns: targetLookup,
		RelationType:        "one-to-many",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	m := relation.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["base_id"] != "base-123" {
		t.Errorf("Map() base_id = %v, want %v", m["base_id"], "base-123")
	}
	if m["source_model_id"] != "model-123" {
		t.Errorf("Map() source_model_id = %v, want %v", m["source_model_id"], "model-123")
	}
	if m["source_column_id"] != "col-123" {
		t.Errorf("Map() source_column_id = %v, want %v", m["source_column_id"], "col-123")
	}
	if m["target_model_id"] != "model-456" {
		t.Errorf("Map() target_model_id = %v, want %v", m["target_model_id"], "model-456")
	}
	if m["target_column_id"] != "col-456" {
		t.Errorf("Map() target_column_id = %v, want %v", m["target_column_id"], "col-456")
	}
	if m["relation_type"] != "one-to-many" {
		t.Errorf("Map() relation_type = %v, want %v", m["relation_type"], "one-to-many")
	}
}

func TestRelationUpdateMap(t *testing.T) {
	now := time.Now()

	sourceLookup := []string{"new_col1"}
	targetLookup := []string{"new_col2"}

	update := dto.RelationUpdate{
		SourceLookupColumns: sourceLookup,
		TargetLookupColumns: targetLookup,
		UpdatedAt:           now,
	}

	m := update.Map()

	if m["source_lookup_columns"] == nil {
		t.Error("Map() source_lookup_columns should not be nil")
	}
	if m["target_lookup_columns"] == nil {
		t.Error("Map() target_lookup_columns should not be nil")
	}
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}

func TestRelationUpdateMapPartial(t *testing.T) {
	now := time.Now()

	sourceLookup := []string{"col1", "col2"}

	update := dto.RelationUpdate{
		SourceLookupColumns: sourceLookup,
		UpdatedAt:           now,
	}

	m := update.Map()

	if m["source_lookup_columns"] == nil {
		t.Error("Map() should contain 'source_lookup_columns' key")
	}
	if _, ok := m["target_lookup_columns"]; ok {
		t.Error("Map() should not contain 'target_lookup_columns' key when it's nil")
	}
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}

func TestRelationUpdateMapOnlyTime(t *testing.T) {
	now := time.Now()

	update := dto.RelationUpdate{
		UpdatedAt: now,
	}

	m := update.Map()

	if _, ok := m["source_lookup_columns"]; ok {
		t.Error("Map() should not contain 'source_lookup_columns' key when it's nil")
	}
	if _, ok := m["target_lookup_columns"]; ok {
		t.Error("Map() should not contain 'target_lookup_columns' key when it's nil")
	}
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}

func TestRelationInsertionAllFields(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	relation := dto.RelationInsertion{
		ID:                  id,
		BaseID:              "base-456",
		SourceModelID:       "model-src",
		SourceColumnID:      "col-src",
		SourceLookupColumns: []string{"lookup1"},
		TargetModelID:       "model-tgt",
		TargetColumnID:      "col-tgt",
		TargetLookupColumns: []string{"lookup2"},
		RelationType:        "many-to-many",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	m := relation.Map()

	// Verify all fields are present in map
	if _, ok := m["id"]; !ok {
		t.Error("Map() should contain 'id' key")
	}
	if _, ok := m["base_id"]; !ok {
		t.Error("Map() should contain 'base_id' key")
	}
	if _, ok := m["source_model_id"]; !ok {
		t.Error("Map() should contain 'source_model_id' key")
	}
	if _, ok := m["source_column_id"]; !ok {
		t.Error("Map() should contain 'source_column_id' key")
	}
	if _, ok := m["target_model_id"]; !ok {
		t.Error("Map() should contain 'target_model_id' key")
	}
	if _, ok := m["target_column_id"]; !ok {
		t.Error("Map() should contain 'target_column_id' key")
	}
	if _, ok := m["source_lookup_columns"]; !ok {
		t.Error("Map() should contain 'source_lookup_columns' key")
	}
	if _, ok := m["target_lookup_columns"]; !ok {
		t.Error("Map() should contain 'target_lookup_columns' key")
	}
	if _, ok := m["relation_type"]; !ok {
		t.Error("Map() should contain 'relation_type' key")
	}
	if _, ok := m["created_time"]; !ok {
		t.Error("Map() should contain 'created_time' key")
	}
	if _, ok := m["last_modified_time"]; !ok {
		t.Error("Map() should contain 'last_modified_time' key")
	}
}
