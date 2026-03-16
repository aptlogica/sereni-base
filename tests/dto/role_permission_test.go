package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRolePermissionDTOMap(t *testing.T) {
	id := uuid.New()
	roleID := uuid.New()
	permissionID := uuid.New()
	now := time.Now()

	rp := dto.RolePermissionDTO{
		ID:           id,
		RoleID:       roleID,
		PermissionID: permissionID,
		CreatedAt:    now,
	}

	m := rp.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["role_id"] != roleID {
		t.Errorf("Map() role_id = %v, want %v", m["role_id"], roleID)
	}
	if m["permission_id"] != permissionID {
		t.Errorf("Map() permission_id = %v, want %v", m["permission_id"], permissionID)
	}
	if m["created_time"] != now {
		t.Errorf("Map() created_time = %v, want %v", m["created_time"], now)
	}
}

func TestRolePermissionResponseFields(t *testing.T) {
	id := uuid.New()
	roleID := uuid.New()
	permissionID := uuid.New()
	now := time.Now()

	permission := &dto.PermissionWithDetails{
		ID:           permissionID,
		ResourceCode: "workspace",
		ActionCode:   "read",
		CreatedAt:    now,
	}

	resp := dto.RolePermissionResponse{
		ID:           id,
		RoleID:       roleID,
		PermissionID: permissionID,
		Permission:   permission,
		CreatedAt:    now,
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.RoleID != roleID {
		t.Errorf("RoleID = %v, want %v", resp.RoleID, roleID)
	}
	if resp.PermissionID != permissionID {
		t.Errorf("PermissionID = %v, want %v", resp.PermissionID, permissionID)
	}
	if resp.Permission == nil {
		t.Error("Permission should not be nil")
	}
	if resp.Permission != nil && resp.Permission.ResourceCode != "workspace" {
		t.Errorf("Permission.ResourceCode = %v, want %v", resp.Permission.ResourceCode, "workspace")
	}
	if resp.Permission != nil && resp.Permission.ActionCode != "read" {
		t.Errorf("Permission.ActionCode = %v, want %v", resp.Permission.ActionCode, "read")
	}
}

func TestRolePermissionsFields(t *testing.T) {
	roleID := uuid.New()
	now := time.Now()

	permissions := []dto.PermissionWithDetails{
		{
			ID:           uuid.New(),
			ResourceCode: "workspace",
			ActionCode:   "read",
			CreatedAt:    now,
		},
		{
			ID:           uuid.New(),
			ResourceCode: "base",
			ActionCode:   "create",
			CreatedAt:    now,
		},
	}

	rp := dto.RolePermissions{
		RoleID:      roleID,
		RoleName:    "editor",
		ScopeLevel:  "workspace",
		Permissions: permissions,
	}

	if rp.RoleID != roleID {
		t.Errorf("RoleID = %v, want %v", rp.RoleID, roleID)
	}
	if rp.RoleName != "editor" {
		t.Errorf("RoleName = %v, want %v", rp.RoleName, "editor")
	}
	if rp.ScopeLevel != "workspace" {
		t.Errorf("ScopeLevel = %v, want %v", rp.ScopeLevel, "workspace")
	}
	if len(rp.Permissions) != 2 {
		t.Errorf("len(Permissions) = %v, want %v", len(rp.Permissions), 2)
	}
}

func TestRolePermissionDTOMapAllFields(t *testing.T) {
	id := uuid.New()
	roleID := uuid.New()
	permissionID := uuid.New()
	now := time.Now()

	rp := dto.RolePermissionDTO{
		ID:           id,
		RoleID:       roleID,
		PermissionID: permissionID,
		CreatedAt:    now,
	}

	m := rp.Map()

	// Verify all fields are present in map
	if _, ok := m["id"]; !ok {
		t.Error("Map() should contain 'id' key")
	}
	if _, ok := m["role_id"]; !ok {
		t.Error("Map() should contain 'role_id' key")
	}
	if _, ok := m["permission_id"]; !ok {
		t.Error("Map() should contain 'permission_id' key")
	}
	if _, ok := m["created_time"]; !ok {
		t.Error("Map() should contain 'created_time' key")
	}
}
