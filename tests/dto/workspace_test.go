package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWorkspaceInsertionMap(t *testing.T) {
	id := uuid.New()
	description := "Test workspace description"
	now := time.Now()

	ws := &dto.WorkspaceInsertion{
		ID:          id,
		Title:       "Test Workspace",
		Description: &description,
		Slug:        "test-workspace",
		Meta: map[string]interface{}{
			"key": "value",
		},
		IsDefault: true,
		Status:    "active",
		CreatedBy: "user1",
		UpdatedBy: "user1",
		CreatedAt: now,
		UpdatedAt: now,
	}

	m := ws.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["title"] != "Test Workspace" {
		t.Errorf("Map() title = %v, want %v", m["title"], "Test Workspace")
	}
	if m["slug"] != "test-workspace" {
		t.Errorf("Map() slug = %v, want %v", m["slug"], "test-workspace")
	}
	if m["is_default"] != true {
		t.Errorf("Map() is_default = %v, want %v", m["is_default"], true)
	}
	if m["status"] != "active" {
		t.Errorf("Map() status = %v, want %v", m["status"], "active")
	}
}

func TestWorkspaceUpdateMap(t *testing.T) {
	title := "Updated Title"
	description := "Updated Description"
	slug := "updated-slug"
	status := "inactive"
	isDefault := false
	meta := map[string]interface{}{"key": "new_value"}
	now := time.Now()

	wu := &dto.WorkspaceUpdate{
		Title:       &title,
		Description: &description,
		Slug:        &slug,
		Status:      &status,
		IsDefault:   &isDefault,
		Meta:        &meta,
		UpdatedBy:   "user2",
		UpdatedAt:   now,
	}

	m := wu.Map()

	if m["title"] != title {
		t.Errorf("Map() title = %v, want %v", m["title"], title)
	}
	if m["description"] != description {
		t.Errorf("Map() description = %v, want %v", m["description"], description)
	}
	if m["slug"] != slug {
		t.Errorf("Map() slug = %v, want %v", m["slug"], slug)
	}
	if m["status"] != status {
		t.Errorf("Map() status = %v, want %v", m["status"], status)
	}
	if m["is_default"] != isDefault {
		t.Errorf("Map() is_default = %v, want %v", m["is_default"], isDefault)
	}
	if m["last_modified_by"] != "user2" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user2")
	}
}

func TestWorkspaceUpdateMapEmpty(t *testing.T) {
	now := time.Now()
	wu := &dto.WorkspaceUpdate{
		UpdatedBy: "user",
		UpdatedAt: now,
	}

	m := wu.Map()

	if len(m) != 2 {
		t.Errorf("Map() length = %d, want 2", len(m))
	}
	if m["last_modified_by"] != "user" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "user")
	}
}

func TestWorkspaceMemberInsertionMap(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	wmi := &dto.WorkspaceMemberInsertion{
		ID:          id,
		WorkspaceID: "ws-123",
		UserID:      "user-456",
		AccessLevel: "editor",
		BasesIds:    "base1,base2",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m := wmi.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["workspace_id"] != "ws-123" {
		t.Errorf("Map() workspace_id = %v, want %v", m["workspace_id"], "ws-123")
	}
	if m["user_id"] != "user-456" {
		t.Errorf("Map() user_id = %v, want %v", m["user_id"], "user-456")
	}
	if m["access_level"] != "editor" {
		t.Errorf("Map() access_level = %v, want %v", m["access_level"], "editor")
	}
	if m["bases_ids"] != "base1,base2" {
		t.Errorf("Map() bases_ids = %v, want %v", m["bases_ids"], "base1,base2")
	}
	if m["created_time"] != now {
		t.Errorf("Map() created_time = %v, want %v", m["created_time"], now)
	}
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}
