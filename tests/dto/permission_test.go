package tests

import (
	"serenibase/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPermissionDTOMap(t *testing.T) {
	id := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()
	now := time.Now()

	permission := dto.PermissionDTO{
		ID:         id,
		ResourceID: resourceID,
		ActionID:   actionID,
		CreatedAt:  now,
	}

	m := permission.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["resource_id"] != resourceID {
		t.Errorf("Map() resource_id = %v, want %v", m["resource_id"], resourceID)
	}
	if m["action_id"] != actionID {
		t.Errorf("Map() action_id = %v, want %v", m["action_id"], actionID)
	}
	if m["created_time"] != now {
		t.Errorf("Map() created_time = %v, want %v", m["created_time"], now)
	}
}

func TestPermissionResponseFields(t *testing.T) {
	id := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()
	now := time.Now()

	resourceDesc := "Workspace resource"
	resource := &dto.ResourceResponse{
		ID:          resourceID,
		Code:        "workspace",
		Description: &resourceDesc,
		CreatedAt:   now,
	}

	actionDesc := "Read action"
	action := &dto.ActionResponse{
		ID:          actionID,
		Code:        "read",
		Description: &actionDesc,
		CreatedAt:   now,
	}

	resp := dto.PermissionResponse{
		ID:         id,
		ResourceID: resourceID,
		ActionID:   actionID,
		Resource:   resource,
		Action:     action,
		CreatedAt:  now,
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.ResourceID != resourceID {
		t.Errorf("ResourceID = %v, want %v", resp.ResourceID, resourceID)
	}
	if resp.ActionID != actionID {
		t.Errorf("ActionID = %v, want %v", resp.ActionID, actionID)
	}
	if resp.Resource == nil {
		t.Error("Resource should not be nil")
	}
	if resp.Resource != nil && resp.Resource.Code != "workspace" {
		t.Errorf("Resource.Code = %v, want %v", resp.Resource.Code, "workspace")
	}
	if resp.Action == nil {
		t.Error("Action should not be nil")
	}
	if resp.Action != nil && resp.Action.Code != "read" {
		t.Errorf("Action.Code = %v, want %v", resp.Action.Code, "read")
	}
}

func TestPermissionWithDetailsFields(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	details := dto.PermissionWithDetails{
		ID:           id,
		ResourceCode: "base",
		ActionCode:   "update",
		CreatedAt:    now,
	}

	if details.ID != id {
		t.Errorf("ID = %v, want %v", details.ID, id)
	}
	if details.ResourceCode != "base" {
		t.Errorf("ResourceCode = %v, want %v", details.ResourceCode, "base")
	}
	if details.ActionCode != "update" {
		t.Errorf("ActionCode = %v, want %v", details.ActionCode, "update")
	}
	if details.CreatedAt != now {
		t.Errorf("CreatedAt = %v, want %v", details.CreatedAt, now)
	}
}

func TestPermissionDTOMapAllFields(t *testing.T) {
	id := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()
	now := time.Now()

	permission := dto.PermissionDTO{
		ID:         id,
		ResourceID: resourceID,
		ActionID:   actionID,
		CreatedAt:  now,
	}

	m := permission.Map()

	// Verify all fields are present in map
	if _, ok := m["id"]; !ok {
		t.Error("Map() should contain 'id' key")
	}
	if _, ok := m["resource_id"]; !ok {
		t.Error("Map() should contain 'resource_id' key")
	}
	if _, ok := m["action_id"]; !ok {
		t.Error("Map() should contain 'action_id' key")
	}
	if _, ok := m["created_time"]; !ok {
		t.Error("Map() should contain 'created_time' key")
	}
}
