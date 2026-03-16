package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestColumnInsertionMap(t *testing.T) {
	id := uuid.New()
	modelID := uuid.New()
	baseID := uuid.New()
	description := "Test column"
	dt := "VARCHAR"
	orderIndex := 1.5
	now := time.Now()

	meta := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	column := dto.ColumnInsertion{
		ID:          id,
		ModelID:     modelID,
		BaseID:      baseID,
		ColumnName:  "test_column",
		Title:       "Test Column",
		Description: &description,
		Meta:        meta,
		UIDT:        "SingleLineText",
		DT:          &dt,
		Virtual:     false,
		System:      false,
		Deleted:     false,
		OrderIndex:  &orderIndex,
		CreatedBy:   "user-123",
		UpdatedBy:   "user-123",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m := column.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["model_id"] != modelID {
		t.Errorf("Map() model_id = %v, want %v", m["model_id"], modelID)
	}
	if m["base_id"] != baseID {
		t.Errorf("Map() base_id = %v, want %v", m["base_id"], baseID)
	}
	if m["column_name"] != "test_column" {
		t.Errorf("Map() column_name = %v, want %v", m["column_name"], "test_column")
	}
	if m["title"] != "Test Column" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Test Column")
	}
	if m["uidt"] != "SingleLineText" {
		t.Errorf("Map() uidt = %v, want %v", m["uidt"], "SingleLineText")
	}
	if m["virtual"] != false {
		t.Errorf("Map() virtual = %v, want %v", m["virtual"], false)
	}
	if m["system"] != false {
		t.Errorf("Map() system = %v, want %v", m["system"], false)
	}
	if m["deleted"] != false {
		t.Errorf("Map() deleted = %v, want %v", m["deleted"], false)
	}
}

func TestColumnInsertionMapMinimal(t *testing.T) {
	id := uuid.New()
	modelID := uuid.New()
	baseID := uuid.New()
	now := time.Now()

	column := dto.ColumnInsertion{
		ID:         id,
		ModelID:    modelID,
		BaseID:     baseID,
		ColumnName: "id",
		Title:      "ID",
		UIDT:       "ID",
		System:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	m := column.Map()

	if m["column_name"] != "id" {
		t.Errorf("Map() column_name = %v, want %v", m["column_name"], "id")
	}
	if m["system"] != true {
		t.Errorf("Map() system = %v, want %v", m["system"], true)
	}
}

func TestColumnUpdateMap(t *testing.T) {
	title := "Updated Title"
	description := "Updated description"
	uidt := "SingleLineText"
	dt := "VARCHAR"
	virtual := true
	system := false
	deleted := false
	orderIndex := 2.5
	now := time.Now()

	meta := map[string]interface{}{
		"updated": true,
	}

	update := dto.ColumnUpdate{
		Title:       &title,
		Description: &description,
		Meta:        &meta,
		UIDT:        &uidt,
		DT:          &dt,
		Virtual:     &virtual,
		System:      &system,
		Deleted:     &deleted,
		OrderIndex:  &orderIndex,
		UpdatedBy:   "user-456",
		UpdatedAt:   now,
	}

	m := update.Map()

	if m["title"] != "Updated Title" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Updated Title")
	}
	if m["description"] != "Updated description" {
		t.Errorf("Map() description = %v, want %v", m["description"], "Updated description")
	}
	if m["uidt"] != "SingleLineText" {
		t.Errorf("Map() uidt = %v, want %v", m["uidt"], "SingleLineText")
	}
	if m["virtual"] != true {
		t.Errorf("Map() virtual = %v, want %v", m["virtual"], true)
	}
	if m["last_modified_by"] != "user-456" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user-456")
	}
}

func TestColumnUpdateMapPartial(t *testing.T) {
	title := "New Title"
	now := time.Now()

	update := dto.ColumnUpdate{
		Title:     &title,
		UpdatedBy: "user-789",
		UpdatedAt: now,
	}

	m := update.Map()

	if m["title"] != "New Title" {
		t.Errorf("Map() title = %v, want %v", m["title"], "New Title")
	}
	if _, ok := m["description"]; ok {
		t.Error("Map() should not contain 'description' key when it's nil")
	}
	if _, ok := m["uidt"]; ok {
		t.Error("Map() should not contain 'uidt' key when it's nil")
	}
	if m["last_modified_by"] != "user-789" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user-789")
	}
}

func TestColumnUpdateMapEmpty(t *testing.T) {
	now := time.Now()

	update := dto.ColumnUpdate{
		UpdatedBy: "user-000",
		UpdatedAt: now,
	}

	m := update.Map()

	// Only UpdatedBy and UpdatedAt should be present
	if m["last_modified_by"] != "user-000" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user-000")
	}
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}
