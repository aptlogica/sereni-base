package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestActionDTOMap(t *testing.T) {
	id := uuid.New()
	description := "Read operation"
	now := time.Now()

	action := dto.ActionDTO{
		ID:          id,
		Code:        "read",
		Description: &description,
		CreatedAt:   now,
	}

	m := action.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["code"] != "read" {
		t.Errorf("Map() code = %v, want %v", m["code"], "read")
	}
	if m["description"] != &description {
		t.Errorf("Map() description = %v, want %v", m["description"], &description)
	}
	if m["created_time"] != now {
		t.Errorf("Map() created_time = %v, want %v", m["created_time"], now)
	}
}

func TestActionDTOMapMinimal(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	action := dto.ActionDTO{
		ID:        id,
		Code:      "delete",
		CreatedAt: now,
	}

	m := action.Map()

	if m["code"] != "delete" {
		t.Errorf("Map() code = %v, want %v", m["code"], "delete")
	}
	if m["description"] != (*string)(nil) {
		t.Errorf("Map() description = %v, want %v", m["description"], (*string)(nil))
	}
}

func TestActionResponseFields(t *testing.T) {
	id := uuid.New()
	description := "Update operation"
	now := time.Now()

	resp := dto.ActionResponse{
		ID:          id,
		Code:        "update",
		Description: &description,
		CreatedAt:   now,
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Code != "update" {
		t.Errorf("Code = %v, want %v", resp.Code, "update")
	}
	if resp.Description == nil || *resp.Description != "Update operation" {
		t.Errorf("Description = %v, want %v", resp.Description, &description)
	}
	if resp.CreatedAt != now {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, now)
	}
}

func TestActionDTOMapAllFields(t *testing.T) {
	id := uuid.New()
	description := "Create new resource"
	now := time.Now()

	action := dto.ActionDTO{
		ID:          id,
		Code:        "create",
		Description: &description,
		CreatedAt:   now,
	}

	m := action.Map()

	// Verify all fields are present in map
	if _, ok := m["id"]; !ok {
		t.Error("Map() should contain 'id' key")
	}
	if _, ok := m["code"]; !ok {
		t.Error("Map() should contain 'code' key")
	}
	if _, ok := m["description"]; !ok {
		t.Error("Map() should contain 'description' key")
	}
	if _, ok := m["created_time"]; !ok {
		t.Error("Map() should contain 'created_time' key")
	}
}
