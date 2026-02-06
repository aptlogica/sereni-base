package tests

import (
	"serenibase/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBaseInsertionMap(t *testing.T) {
	id := uuid.New()
	workspaceID := uuid.New()
	description := "Test base description"
	now := time.Now()

	config := map[string]interface{}{
		"host": "localhost",
		"port": 5432,
	}

	settings := map[string]interface{}{
		"timezone": "UTC",
	}

	meta := map[string]interface{}{
		"icon": "database",
	}

	base := dto.BaseInsertion{
		ID:               id,
		WorkspaceID:      workspaceID,
		Title:            "Test Base",
		Description:      &description,
		Type:             "internal",
		Config:           config,
		Settings:         settings,
		Meta:             meta,
		Status:           "active",
		Visibility:       "private",
		TableCount:       5,
		RowCount:         1000,
		StorageUsedBytes: 1024000,
		CreatedBy:        "user-123",
		UpdatedBy:        "user-123",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	m := base.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["workspace_id"] != workspaceID {
		t.Errorf("Map() workspace_id = %v, want %v", m["workspace_id"], workspaceID)
	}
	if m["title"] != "Test Base" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Test Base")
	}
	if m["type"] != "internal" {
		t.Errorf("Map() type = %v, want %v", m["type"], "internal")
	}
	if m["status"] != "active" {
		t.Errorf("Map() status = %v, want %v", m["status"], "active")
	}
	if m["visibility"] != "private" {
		t.Errorf("Map() visibility = %v, want %v", m["visibility"], "private")
	}
	if m["table_count"] != 5 {
		t.Errorf("Map() table_count = %v, want %v", m["table_count"], 5)
	}
	if m["row_count"] != int64(1000) {
		t.Errorf("Map() row_count = %v, want %v", m["row_count"], int64(1000))
	}
	if m["storage_used_bytes"] != int64(1024000) {
		t.Errorf("Map() storage_used_bytes = %v, want %v", m["storage_used_bytes"], int64(1024000))
	}
}

func TestBaseInsertionSetConfig(t *testing.T) {
	base := dto.BaseInsertion{}

	// Valid JSON
	err := base.SetConfig(`{"key": "value"}`)
	if err != nil {
		t.Errorf("SetConfig() with valid JSON should not error, got %v", err)
	}

	// Invalid JSON
	err = base.SetConfig(`{invalid json}`)
	if err == nil {
		t.Error("SetConfig() with invalid JSON should return error")
	}

	// Empty string
	err = base.SetConfig("")
	if err != nil {
		t.Errorf("SetConfig() with empty string should not error, got %v", err)
	}
	if base.Config != nil {
		t.Error("SetConfig() with empty string should set Config to nil")
	}
}

func TestBaseInsertionSetSettings(t *testing.T) {
	base := dto.BaseInsertion{}

	// Valid JSON
	err := base.SetSettings(`{"setting": "value"}`)
	if err != nil {
		t.Errorf("SetSettings() with valid JSON should not error, got %v", err)
	}

	// Invalid JSON
	err = base.SetSettings(`{bad json}`)
	if err == nil {
		t.Error("SetSettings() with invalid JSON should return error")
	}

	// Empty string
	err = base.SetSettings("")
	if err != nil {
		t.Errorf("SetSettings() with empty string should not error, got %v", err)
	}
	if base.Settings != nil {
		t.Error("SetSettings() with empty string should set Settings to nil")
	}
}

func TestBaseInsertionSetMeta(t *testing.T) {
	base := dto.BaseInsertion{}

	// Valid JSON
	err := base.SetMeta(`{"meta": "data"}`)
	if err != nil {
		t.Errorf("SetMeta() with valid JSON should not error, got %v", err)
	}

	// Invalid JSON
	err = base.SetMeta(`{not valid}`)
	if err == nil {
		t.Error("SetMeta() with invalid JSON should return error")
	}

	// Empty string
	err = base.SetMeta("")
	if err != nil {
		t.Errorf("SetMeta() with empty string should not error, got %v", err)
	}
	if base.Meta != nil {
		t.Error("SetMeta() with empty string should set Meta to nil")
	}
}

func TestBaseUpdateMap(t *testing.T) {
	title := "Updated Base"
	description := "Updated description"
	image := "image.png"
	baseType := "external"
	status := "inactive"
	visibility := "public"
	tableCount := 10
	rowCount := int64(2000)
	storageUsed := int64(2048000)
	now := time.Now()

	config := map[string]interface{}{
		"host": "remote",
	}

	settings := map[string]interface{}{
		"backup": true,
	}

	meta := map[string]interface{}{
		"color": "blue",
	}

	update := dto.BaseUpdate{
		Title:            &title,
		Description:      &description,
		Image:            &image,
		Type:             &baseType,
		Config:           &config,
		Settings:         &settings,
		Meta:             &meta,
		Status:           &status,
		Visibility:       &visibility,
		TableCount:       &tableCount,
		RowCount:         &rowCount,
		StorageUsedBytes: &storageUsed,
		UpdatedBy:        "user-456",
		UpdatedAt:        now,
	}

	m := update.Map()

	if m["title"] != "Updated Base" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Updated Base")
	}
	if m["type"] != "external" {
		t.Errorf("Map() type = %v, want %v", m["type"], "external")
	}
	if m["status"] != "inactive" {
		t.Errorf("Map() status = %v, want %v", m["status"], "inactive")
	}
	if m["visibility"] != "public" {
		t.Errorf("Map() visibility = %v, want %v", m["visibility"], "public")
	}
	if m["table_count"] != 10 {
		t.Errorf("Map() table_count = %v, want %v", m["table_count"], 10)
	}
	if m["last_modified_by"] != "user-456" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user-456")
	}
}

func TestBaseUpdateMapPartial(t *testing.T) {
	title := "New Title"
	now := time.Now()

	update := dto.BaseUpdate{
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
	if _, ok := m["type"]; ok {
		t.Error("Map() should not contain 'type' key when it's nil")
	}
	if m["last_modified_by"] != "user-789" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user-789")
	}
}

func TestCreateBaseRequestFields(t *testing.T) {
	description := "Test description"

	req := dto.CreateBaseRequest{
		Title:       "New Base",
		Description: &description,
		WorkspaceID: "ws-123",
		CreatedBy:   "user-123",
	}

	if req.Title != "New Base" {
		t.Errorf("Title = %v, want %v", req.Title, "New Base")
	}
	if req.Description == nil || *req.Description != "Test description" {
		t.Errorf("Description = %v, want %v", req.Description, &description)
	}
	if req.WorkspaceID != "ws-123" {
		t.Errorf("WorkspaceID = %v, want %v", req.WorkspaceID, "ws-123")
	}
	if req.CreatedBy != "user-123" {
		t.Errorf("CreatedBy = %v, want %v", req.CreatedBy, "user-123")
	}
}

func TestBaseResponseFields(t *testing.T) {
	id := uuid.New()
	description := "Base description"
	now := time.Now()

	config := map[string]interface{}{
		"host": "localhost",
	}

	tables := []dto.TableResponse{
		{
			Model: dto.ModelResponse{
				ID:    uuid.New(),
				Title: "Table 1",
			},
		},
	}

	resp := dto.BaseResponse{
		ID:               id,
		WorkspaceID:      "ws-123",
		Title:            "Test Base",
		Description:      &description,
		Image:            "image.png",
		Type:             "internal",
		Config:           config,
		Status:           "active",
		Visibility:       "private",
		TableCount:       5,
		RowCount:         1000,
		StorageUsedBytes: 1024000,
		CreatedBy:        "user-123",
		UpdatedBy:        "user-456",
		CreatedAt:        now,
		UpdatedAt:        now,
		AccessLevel:      "owner",
		Tables:           tables,
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Title != "Test Base" {
		t.Errorf("Title = %v, want %v", resp.Title, "Test Base")
	}
	if resp.Type != "internal" {
		t.Errorf("Type = %v, want %v", resp.Type, "internal")
	}
	if resp.AccessLevel != "owner" {
		t.Errorf("AccessLevel = %v, want %v", resp.AccessLevel, "owner")
	}
	if len(resp.Tables) != 1 {
		t.Errorf("len(Tables) = %v, want %v", len(resp.Tables), 1)
	}
}
