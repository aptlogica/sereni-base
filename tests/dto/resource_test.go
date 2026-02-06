package tests

import (
	"serenibase/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestResourceDTOMap(t *testing.T) {
	id := uuid.New()
	description := "Workspace resource"
	now := time.Now()

	resource := dto.ResourceDTO{
		ID:          id,
		Code:        "workspace",
		Description: &description,
		CreatedAt:   now,
	}

	m := resource.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["code"] != "workspace" {
		t.Errorf("Map() code = %v, want %v", m["code"], "workspace")
	}
	if m["description"] != &description {
		t.Errorf("Map() description = %v, want %v", m["description"], &description)
	}
	if m["created_time"] != now {
		t.Errorf("Map() created_time = %v, want %v", m["created_time"], now)
	}
}

func TestResourceDTOMapMinimal(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	resource := dto.ResourceDTO{
		ID:        id,
		Code:      "base",
		CreatedAt: now,
	}

	m := resource.Map()

	if m["code"] != "base" {
		t.Errorf("Map() code = %v, want %v", m["code"], "base")
	}
	if m["description"] != (*string)(nil) {
		t.Errorf("Map() description = %v, want %v", m["description"], (*string)(nil))
	}
}

func TestResourceResponseFields(t *testing.T) {
	id := uuid.New()
	description := "Records resource"
	now := time.Now()

	resp := dto.ResourceResponse{
		ID:          id,
		Code:        "records",
		Description: &description,
		CreatedAt:   now,
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Code != "records" {
		t.Errorf("Code = %v, want %v", resp.Code, "records")
	}
	if resp.Description == nil || *resp.Description != "Records resource" {
		t.Errorf("Description = %v, want %v", resp.Description, &description)
	}
	if resp.CreatedAt != now {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, now)
	}
}

func TestResourceDTOMapAllFields(t *testing.T) {
	id := uuid.New()
	description := "Members resource"
	now := time.Now()

	resource := dto.ResourceDTO{
		ID:          id,
		Code:        "members",
		Description: &description,
		CreatedAt:   now,
	}

	m := resource.Map()

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
