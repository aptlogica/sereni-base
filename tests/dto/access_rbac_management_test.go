package tests

import (
	"serenibase/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessRoleDTOMapRBAC(t *testing.T) {
	id := uuid.New()
	description := "Test role description"
	workspaceID := "ws-123"
	now := time.Now()

	role := dto.AccessRoleDTO{
		ID:          id,
		Name:        "editor",
		ScopeLevel:  "workspace",
		Priority:    50,
		Description: &description,
		IsDefault:   true,
		WorkspaceID: &workspaceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m := role.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["name"] != "editor" {
		t.Errorf("Map() name = %v, want %v", m["name"], "editor")
	}
	if m["scope_level"] != "workspace" {
		t.Errorf("Map() scope_level = %v, want %v", m["scope_level"], "workspace")
	}
	if m["priority"] != 50 {
		t.Errorf("Map() priority = %v, want %v", m["priority"], 50)
	}
	if m["description"] != &description {
		t.Errorf("Map() description = %v, want %v", m["description"], &description)
	}
	if m["is_default"] != true {
		t.Errorf("Map() is_default = %v, want %v", m["is_default"], true)
	}
	if m["workspace_id"] != &workspaceID {
		t.Errorf("Map() workspace_id = %v, want %v", m["workspace_id"], &workspaceID)
	}
}

func TestAccessRoleDTOMapMinimal(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	role := dto.AccessRoleDTO{
		ID:         id,
		Name:       "viewer",
		ScopeLevel: "base",
		Priority:   10,
		IsDefault:  false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	m := role.Map()

	if m["name"] != "viewer" {
		t.Errorf("Map() name = %v, want %v", m["name"], "viewer")
	}
	if m["scope_level"] != "base" {
		t.Errorf("Map() scope_level = %v, want %v", m["scope_level"], "base")
	}
	if m["priority"] != 10 {
		t.Errorf("Map() priority = %v, want %v", m["priority"], 10)
	}
	if m["is_default"] != false {
		t.Errorf("Map() is_default = %v, want %v", m["is_default"], false)
	}
	// description and workspace_id can be nil pointers
	if m["description"] != (*string)(nil) {
		t.Errorf("Map() description = %v, want %v", m["description"], (*string)(nil))
	}
	if m["workspace_id"] != (*string)(nil) {
		t.Errorf("Map() workspace_id = %v, want %v", m["workspace_id"], (*string)(nil))
	}
}

func TestAccessRoleResponseFields(t *testing.T) {
	id := uuid.New()
	description := "Test role"
	workspaceID := "ws-123"
	now := time.Now()

	resp := dto.AccessRoleResponse{
		ID:          id,
		Name:        "admin",
		ScopeLevel:  "system",
		Priority:    100,
		Description: &description,
		IsDefault:   true,
		WorkspaceID: &workspaceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Name != "admin" {
		t.Errorf("Name = %v, want %v", resp.Name, "admin")
	}
	if resp.ScopeLevel != "system" {
		t.Errorf("ScopeLevel = %v, want %v", resp.ScopeLevel, "system")
	}
	if resp.Priority != 100 {
		t.Errorf("Priority = %v, want %v", resp.Priority, 100)
	}
	if *resp.Description != "Test role" {
		t.Errorf("Description = %v, want %v", *resp.Description, "Test role")
	}
	if !resp.IsDefault {
		t.Errorf("IsDefault = %v, want %v", resp.IsDefault, true)
	}
	if *resp.WorkspaceID != "ws-123" {
		t.Errorf("WorkspaceID = %v, want %v", *resp.WorkspaceID, "ws-123")
	}
}

func TestRBACSystemStatusFields(t *testing.T) {
	status := dto.RBACSystemStatus{
		Initialized:          true,
		TotalRoles:           5,
		TotalResources:       10,
		TotalActions:         20,
		TotalPermissions:     50,
		TotalRoleAssignments: 100,
		DefaultRolesCreated:  true,
		Status:               "healthy",
	}

	if !status.Initialized {
		t.Errorf("Initialized = %v, want %v", status.Initialized, true)
	}
	if status.TotalRoles != 5 {
		t.Errorf("TotalRoles = %v, want %v", status.TotalRoles, 5)
	}
	if status.Status != "healthy" {
		t.Errorf("Status = %v, want %v", status.Status, "healthy")
	}
}

func TestRoleUsageStatsFields(t *testing.T) {
	roleID := uuid.New()
	lastAssigned := "2024-01-01T12:00:00Z"

	stats := dto.RoleUsageStats{
		RoleID:          roleID,
		RoleName:        "editor",
		ScopeLevel:      "workspace",
		UserCount:       10,
		PermissionCount: 25,
		LastAssigned:    &lastAssigned,
	}

	if stats.RoleID != roleID {
		t.Errorf("RoleID = %v, want %v", stats.RoleID, roleID)
	}
	if stats.RoleName != "editor" {
		t.Errorf("RoleName = %v, want %v", stats.RoleName, "editor")
	}
	if stats.UserCount != 10 {
		t.Errorf("UserCount = %v, want %v", stats.UserCount, 10)
	}
}

func TestPermissionUsageStatsFields(t *testing.T) {
	permID := uuid.New()

	stats := dto.PermissionUsageStats{
		PermissionID: permID,
		ResourceCode: "workspace",
		ActionCode:   "read",
		RoleCount:    3,
		IsOrphaned:   false,
	}

	if stats.PermissionID != permID {
		t.Errorf("PermissionID = %v, want %v", stats.PermissionID, permID)
	}
	if stats.ResourceCode != "workspace" {
		t.Errorf("ResourceCode = %v, want %v", stats.ResourceCode, "workspace")
	}
	if stats.IsOrphaned {
		t.Errorf("IsOrphaned = %v, want %v", stats.IsOrphaned, false)
	}
}

func TestResourceUsageStatsFields(t *testing.T) {
	resID := uuid.New()

	stats := dto.ResourceUsageStats{
		ResourceID:      resID,
		ResourceCode:    "table",
		PermissionCount: 15,
		RoleCount:       4,
	}

	if stats.ResourceID != resID {
		t.Errorf("ResourceID = %v, want %v", stats.ResourceID, resID)
	}
	if stats.ResourceCode != "table" {
		t.Errorf("ResourceCode = %v, want %v", stats.ResourceCode, "table")
	}
	if stats.PermissionCount != 15 {
		t.Errorf("PermissionCount = %v, want %v", stats.PermissionCount, 15)
	}
}

func TestRoleValidationResultFields(t *testing.T) {
	roleID := uuid.New()

	result := dto.RoleValidationResult{
		RoleID:          roleID,
		RoleName:        "admin",
		IsValid:         true,
		HasPermissions:  true,
		PermissionCount: 50,
		HasUsers:        true,
		UserCount:       10,
		Issues:          []string{},
		Warnings:        []string{"High permission count"},
	}

	if result.RoleID != roleID {
		t.Errorf("RoleID = %v, want %v", result.RoleID, roleID)
	}
	if !result.IsValid {
		t.Errorf("IsValid = %v, want %v", result.IsValid, true)
	}
	if result.UserCount != 10 {
		t.Errorf("UserCount = %v, want %v", result.UserCount, 10)
	}
}
