package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestModelInsertionMap(t *testing.T) {
	id := "model-123"
	baseID := "base-123"
	workspaceID := "ws-123"
	now := time.Now()

	meta := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	model := dto.ModelInsertion{
		ID:               id,
		BaseID:           baseID,
		WorkspaceID:      workspaceID,
		Title:            "Test Model",
		Description:      "Test description",
		Alias:            "test_model",
		Type:             "table",
		Meta:             meta,
		Schema:           "{}",
		Tags:             "tag1,tag2",
		OrderIndex:       1.5,
		CreatedBy:        "user-123",
		UpdatedBy:        "user-123",
		CreatedTime:      now,
		LastModifiedTime: now,
	}

	m := model.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["base_id"] != baseID {
		t.Errorf("Map() base_id = %v, want %v", m["base_id"], baseID)
	}
	if m["workspace_id"] != workspaceID {
		t.Errorf("Map() workspace_id = %v, want %v", m["workspace_id"], workspaceID)
	}
	if m["title"] != "Test Model" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Test Model")
	}
	if m["alias"] != "test_model" {
		t.Errorf("Map() alias = %v, want %v", m["alias"], "test_model")
	}
	if m["type"] != "table" {
		t.Errorf("Map() type = %v, want %v", m["type"], "table")
	}
	if m["tags"] != "tag1,tag2" {
		t.Errorf("Map() tags = %v, want %v", m["tags"], "tag1,tag2")
	}
}

func TestUpdateModelRequestMap(t *testing.T) {
	title := "Updated Title"
	description := "Updated description"
	alias := "updated_model"
	modelType := "view"
	enabled := true
	mm := false
	pinned := true
	deleted := false
	tags := "newtag"
	orderIndex := 2.5
	schema := "{\"test\": true}"

	meta := map[string]interface{}{
		"updated": true,
	}

	update := dto.UpdateModelRequest{
		Title:       &title,
		Description: &description,
		Alias:       &alias,
		Type:        &modelType,
		Meta:        &meta,
		Schema:      &schema,
		Enabled:     &enabled,
		MM:          &mm,
		Pinned:      &pinned,
		Deleted:     &deleted,
		Tags:        &tags,
		OrderIndex:  &orderIndex,
		UpdatedBy:   "user-456",
	}

	m := update.Map()

	if m["title"] != "Updated Title" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Updated Title")
	}
	if m["alias"] != "updated_model" {
		t.Errorf("Map() alias = %v, want %v", m["alias"], "updated_model")
	}
	if m["type"] != "view" {
		t.Errorf("Map() type = %v, want %v", m["type"], "view")
	}
	if m["enabled"] != true {
		t.Errorf("Map() enabled = %v, want %v", m["enabled"], true)
	}
	if m["mm"] != false {
		t.Errorf("Map() mm = %v, want %v", m["mm"], false)
	}
	if m["pinned"] != true {
		t.Errorf("Map() pinned = %v, want %v", m["pinned"], true)
	}
	if m["deleted"] != false {
		t.Errorf("Map() deleted = %v, want %v", m["deleted"], false)
	}
}

func TestUpdateModelRequestMapPartial(t *testing.T) {
	title := "New Title"

	update := dto.UpdateModelRequest{
		Title:     &title,
		UpdatedBy: "user-789",
	}

	m := update.Map()

	if m["title"] != "New Title" {
		t.Errorf("Map() title = %v, want %v", m["title"], "New Title")
	}
	if _, ok := m["alias"]; ok {
		t.Error("Map() should not contain 'alias' key when it's nil")
	}
	if _, ok := m["type"]; ok {
		t.Error("Map() should not contain 'type' key when it's nil")
	}
	if m["last_modified_by"] != "user-789" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user-789")
	}
}

func TestModelDTOFields(t *testing.T) {
	now := time.Now()

	meta := map[string]interface{}{
		"icon": "table",
	}

	model := dto.ModelDTO{
		ID:          "model-123",
		BaseID:      "base-123",
		WorkspaceID: "ws-123",
		Title:       "Users",
		Alias:       "users",
		Type:        "table",
		Meta:        meta,
		Schema:      "{}",
		Enabled:     true,
		MM:          false,
		Pinned:      true,
		Deleted:     false,
		Tags:        "important",
		OrderIndex:  1.0,
		RowCount:    100,
		ColumnCount: 10,
		StorageUsed: 1024,
		CreatedBy:   "user-123",
		UpdatedBy:   "user-123",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if model.ID != "model-123" {
		t.Errorf("ID = %v, want %v", model.ID, "model-123")
	}
	if model.Title != "Users" {
		t.Errorf("Title = %v, want %v", model.Title, "Users")
	}
	if model.Type != "table" {
		t.Errorf("Type = %v, want %v", model.Type, "table")
	}
	if model.Enabled != true {
		t.Errorf("Enabled = %v, want %v", model.Enabled, true)
	}
	if model.RowCount != 100 {
		t.Errorf("RowCount = %v, want %v", model.RowCount, 100)
	}
	if model.ColumnCount != 10 {
		t.Errorf("ColumnCount = %v, want %v", model.ColumnCount, 10)
	}
}

func TestModelResponseFields(t *testing.T) {
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	baseID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	workspaceID := uuid.MustParse("323e4567-e89b-12d3-a456-426614174002")
	now := time.Now()
	orderIndex := 1.5

	meta := map[string]interface{}{
		"color": "blue",
	}

	response := dto.ModelResponse{
		ID:          id,
		BaseID:      baseID,
		WorkspaceID: workspaceID,
		Title:       "Test Model",
		Description: "Test description",
		Alias:       "test_model",
		Meta:        meta,
		OrderIndex:  &orderIndex,
		CreatedBy:   "user-123",
		UpdatedBy:   "user-456",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if response.ID != id {
		t.Errorf("ID = %v, want %v", response.ID, id)
	}
	if response.Title != "Test Model" {
		t.Errorf("Title = %v, want %v", response.Title, "Test Model")
	}
	if response.OrderIndex == nil || *response.OrderIndex != 1.5 {
		t.Errorf("OrderIndex = %v, want %v", response.OrderIndex, &orderIndex)
	}
}
