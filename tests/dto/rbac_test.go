package tests

import (
	"serenibase/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRoleInsertionMap(t *testing.T) {
	id := uuid.New()
	description := "Admin role"
	now := time.Now()

	role := &dto.RoleInsertion{
		ID:          id,
		Name:        "Admin",
		Description: &description,
		IsDefault:   true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m := role.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["name"] != "Admin" {
		t.Errorf("Map() name = %v, want %v", m["name"], "Admin")
	}
	if m["description"] != &description {
		t.Errorf("Map() description = %v, want %v", m["description"], &description)
	}
	if m["is_default"] != true {
		t.Errorf("Map() is_default = %v, want %v", m["is_default"], true)
	}
}

func TestUserRoleInsertionMap(t *testing.T) {
	id := uuid.New()
	roleID := uuid.New()
	userID := uuid.New()

	userRole := &dto.UserRoleInsertion{
		ID:     id,
		RoleID: roleID,
		UserID: userID,
	}

	m := userRole.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["role_id"] != roleID {
		t.Errorf("Map() role_id = %v, want %v", m["role_id"], roleID)
	}
	if m["user_id"] != userID {
		t.Errorf("Map() user_id = %v, want %v", m["user_id"], userID)
	}
}
