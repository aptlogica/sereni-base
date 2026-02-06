package tests

import (
	"serenibase/internal/dto"
	"testing"
	"time"
)

func TestUpdateTableRequestMap(t *testing.T) {
	title := "Updated Table"
	description := "New description"
	meta := map[string]interface{}{"key": "value"}
	now := time.Now()

	req := dto.UpdateTableRequest{
		Title:       &title,
		Description: &description,
		Meta:        meta,
		UpdatedBy:   "admin",
		UpdatedAt:   now,
	}

	m := req.Map()

	if m["title"] != &title {
		t.Errorf("Map() title = %v, want %v", m["title"], &title)
	}
	if m["Description"] != &description {
		t.Errorf("Map() Description = %v, want %v", m["Description"], &description)
	}
	if m["last_modified_by"] != "admin" {
		t.Errorf("Map() last_modified_by = %v, want %v", m["last_modified_by"], "admin")
	}
}
