package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestViewInsertionMap(t *testing.T) {
	id := uuid.New()
	modelID := uuid.New()
	baseID := uuid.New()
	description := "Grid view"
	alias := "grid_view"
	lockType := "locked"
	password := "secret"
	viewUUID := "view-uuid-123"
	orderIndex := 1.5
	now := time.Now()

	meta := map[string]interface{}{
		"columns": []string{"col1", "col2"},
		"filters": map[string]interface{}{},
	}

	view := dto.ViewInsertion{
		ID:          id,
		ModelID:     modelID,
		BaseID:      baseID,
		Title:       "Main Grid",
		Description: &description,
		Alias:       &alias,
		Type:        "grid",
		IsDefault:   true,
		LockType:    &lockType,
		Password:    &password,
		Public:      false,
		UUID:        &viewUUID,
		Meta:        meta,
		OrderIndex:  &orderIndex,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "user-123",
		UpdatedBy:   "user-123",
	}

	m := view.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["model_id"] != modelID {
		t.Errorf("Map() model_id = %v, want %v", m["model_id"], modelID)
	}
	if m["base_id"] != baseID {
		t.Errorf("Map() base_id = %v, want %v", m["base_id"], baseID)
	}
	if m["title"] != "Main Grid" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Main Grid")
	}
	if m["type"] != "grid" {
		t.Errorf("Map() type = %v, want %v", m["type"], "grid")
	}
	if m["is_default"] != true {
		t.Errorf("Map() is_default = %v, want %v", m["is_default"], true)
	}
	if m["public"] != false {
		t.Errorf("Map() public = %v, want %v", m["public"], false)
	}
}

func TestViewUpdateMap(t *testing.T) {
	title := "Updated View"
	description := "Updated description"
	viewType := "kanban"
	isDefault := true
	lockType := "collaborative"
	password := "newpass"
	public := true
	viewUUID := "new-uuid"
	orderIndex := 2.5
	now := time.Now()

	meta := map[string]interface{}{
		"updated": true,
	}

	update := dto.ViewUpdate{
		Title:       &title,
		Description: &description,
		Type:        &viewType,
		IsDefault:   &isDefault,
		LockType:    &lockType,
		Password:    &password,
		Public:      &public,
		UUID:        &viewUUID,
		Meta:        &meta,
		OrderIndex:  &orderIndex,
		UpdatedAt:   now,
		UpdatedBy:   "user-456",
	}

	m := update.Map()

	if m["title"] != "Updated View" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Updated View")
	}
	if m["type"] != "kanban" {
		t.Errorf("Map() type = %v, want %v", m["type"], "kanban")
	}
	if m["is_default"] != true {
		t.Errorf("Map() is_default = %v, want %v", m["is_default"], true)
	}
	if m["public"] != true {
		t.Errorf("Map() public = %v, want %v", m["public"], true)
	}
	if m["last_modified_by"] != "user-456" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user-456")
	}
}

func TestViewUpdateMapPartial(t *testing.T) {
	title := "New Title"
	now := time.Now()

	update := dto.ViewUpdate{
		Title:     &title,
		UpdatedAt: now,
		UpdatedBy: "user-789",
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
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}

func TestCreateViewRequestFields(t *testing.T) {
	modelID := uuid.New()
	baseID := uuid.New()
	orderIndex := 1.0

	meta := map[string]interface{}{
		"key": "value",
	}

	req := dto.CreateViewRequest{
		ModelID:     modelID,
		BaseID:      baseID,
		Title:       "New View",
		Description: "View description",
		Type:        "gallery",
		Meta:        &meta,
		OrderIndex:  &orderIndex,
		CreatedBy:   "user-123",
	}

	if req.ModelID != modelID {
		t.Errorf("ModelID = %v, want %v", req.ModelID, modelID)
	}
	if req.BaseID != baseID {
		t.Errorf("BaseID = %v, want %v", req.BaseID, baseID)
	}
	if req.Title != "New View" {
		t.Errorf("Title = %v, want %v", req.Title, "New View")
	}
	if req.Type != "gallery" {
		t.Errorf("Type = %v, want %v", req.Type, "gallery")
	}
	if req.OrderIndex == nil || *req.OrderIndex != 1.0 {
		t.Errorf("OrderIndex = %v, want %v", req.OrderIndex, &orderIndex)
	}
}

func TestViewResponseFields(t *testing.T) {
	id := uuid.New()
	modelID := uuid.New()
	baseID := uuid.New()
	description := "Response view"
	alias := "resp_view"
	isDefault := true
	lockType := "locked"
	password := "pass"
	public := true
	viewUUID := "uuid-123"
	orderIndex := 2.0
	now := time.Now()

	meta := map[string]interface{}{
		"test": true,
	}

	resp := dto.ViewResponse{
		ID:          id,
		ModelID:     modelID,
		BaseID:      baseID,
		Title:       "Response View",
		Description: &description,
		Alias:       &alias,
		Type:        "form",
		IsDefault:   &isDefault,
		LockType:    &lockType,
		Password:    &password,
		Public:      &public,
		UUID:        &viewUUID,
		Meta:        &meta,
		OrderIndex:  &orderIndex,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "user-123",
		UpdatedBy:   "user-456",
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Title != "Response View" {
		t.Errorf("Title = %v, want %v", resp.Title, "Response View")
	}
	if resp.Type != "form" {
		t.Errorf("Type = %v, want %v", resp.Type, "form")
	}
	if resp.IsDefault == nil || *resp.IsDefault != true {
		t.Errorf("IsDefault = %v, want %v", resp.IsDefault, &isDefault)
	}
	if resp.Public == nil || *resp.Public != true {
		t.Errorf("Public = %v, want %v", resp.Public, &public)
	}
}
