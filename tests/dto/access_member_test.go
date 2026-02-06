package tests

import (
	"serenibase/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessMemberDTOMap(t *testing.T) {
	id := uuid.New()
	scopeID := "scope-123"
	workspaceID := "ws-123"
	assignedBy := "user-456"
	now := time.Now()

	member := dto.AccessMemberDTO{
		ID:          id,
		UserID:      "user-123",
		ScopeType:   "workspace",
		ScopeID:     &scopeID,
		RoleID:      "role-123",
		WorkspaceID: &workspaceID,
		AssignedBy:  &assignedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	m := member.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["user_id"] != "user-123" {
		t.Errorf("Map() user_id = %v, want %v", m["user_id"], "user-123")
	}
	if m["scope_type"] != "workspace" {
		t.Errorf("Map() scope_type = %v, want %v", m["scope_type"], "workspace")
	}
	if m["scope_id"] != &scopeID {
		t.Errorf("Map() scope_id = %v, want %v", m["scope_id"], &scopeID)
	}
	if m["role_id"] != "role-123" {
		t.Errorf("Map() role_id = %v, want %v", m["role_id"], "role-123")
	}
	if m["workspace_id"] != &workspaceID {
		t.Errorf("Map() workspace_id = %v, want %v", m["workspace_id"], &workspaceID)
	}
	if m["assigned_by"] != &assignedBy {
		t.Errorf("Map() assigned_by = %v, want %v", m["assigned_by"], &assignedBy)
	}
	if m["created_time"] != now {
		t.Errorf("Map() created_time = %v, want %v", m["created_time"], now)
	}
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}

func TestAccessMemberDTOMapMinimal(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	member := dto.AccessMemberDTO{
		ID:        id,
		UserID:    "user-123",
		ScopeType: "system",
		RoleID:    "role-123",
		CreatedAt: now,
		UpdatedAt: now,
	}

	m := member.Map()

	if m["user_id"] != "user-123" {
		t.Errorf("Map() user_id = %v, want %v", m["user_id"], "user-123")
	}
	if m["scope_type"] != "system" {
		t.Errorf("Map() scope_type = %v, want %v", m["scope_type"], "system")
	}
	if m["scope_id"] != (*string)(nil) {
		t.Errorf("Map() scope_id = %v, want %v", m["scope_id"], (*string)(nil))
	}
	if m["workspace_id"] != (*string)(nil) {
		t.Errorf("Map() workspace_id = %v, want %v", m["workspace_id"], (*string)(nil))
	}
	if m["assigned_by"] != (*string)(nil) {
		t.Errorf("Map() assigned_by = %v, want %v", m["assigned_by"], (*string)(nil))
	}
}

func TestAccessMemberResponseFields(t *testing.T) {
	id := uuid.New()
	scopeID := "scope-123"
	assignedBy := "user-456"
	now := time.Now()

	role := &dto.AccessRoleResponse{
		ID:         uuid.New(),
		Name:       "editor",
		ScopeLevel: "workspace",
		Priority:   50,
		IsDefault:  true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	resp := dto.AccessMemberResponse{
		ID:         id,
		UserID:     "user-123",
		ScopeType:  "workspace",
		ScopeID:    &scopeID,
		RoleID:     "role-123",
		Role:       role,
		AssignedBy: &assignedBy,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.UserID != "user-123" {
		t.Errorf("UserID = %v, want %v", resp.UserID, "user-123")
	}
	if resp.Role == nil {
		t.Error("Role should not be nil")
	}
	if resp.Role != nil && resp.Role.Name != "editor" {
		t.Errorf("Role.Name = %v, want %v", resp.Role.Name, "editor")
	}
}

func TestUserAccessInfoFields(t *testing.T) {
	desc := "System admin"
	sysRole := &dto.AccessRoleDTO{
		ID:          uuid.New(),
		Name:        "admin",
		ScopeLevel:  "system",
		Priority:    100,
		Description: &desc,
		IsDefault:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	wsAccess := []dto.AccessMemberDTO{
		{
			ID:        uuid.New(),
			UserID:    "user-123",
			ScopeType: "workspace",
			RoleID:    "role-456",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	info := dto.UserAccessInfo{
		UserID:          "user-123",
		SystemRole:      sysRole,
		WorkspaceAccess: wsAccess,
	}

	if info.UserID != "user-123" {
		t.Errorf("UserID = %v, want %v", info.UserID, "user-123")
	}
	if info.SystemRole == nil {
		t.Error("SystemRole should not be nil")
	}
	if info.SystemRole != nil && info.SystemRole.Name != "admin" {
		t.Errorf("SystemRole.Name = %v, want %v", info.SystemRole.Name, "admin")
	}
	if len(info.WorkspaceAccess) != 1 {
		t.Errorf("len(WorkspaceAccess) = %v, want %v", len(info.WorkspaceAccess), 1)
	}
}

func TestBulkAssignRoleRequestFields(t *testing.T) {
	scopeID := "scope-123"
	assignedBy := "user-admin"

	req := dto.BulkAssignRoleRequest{
		UserIDs:    []string{"user-1", "user-2", "user-3"},
		ScopeType:  "workspace",
		ScopeID:    &scopeID,
		RoleID:     "role-123",
		AssignedBy: &assignedBy,
	}

	if len(req.UserIDs) != 3 {
		t.Errorf("len(UserIDs) = %v, want %v", len(req.UserIDs), 3)
	}
	if req.ScopeType != "workspace" {
		t.Errorf("ScopeType = %v, want %v", req.ScopeType, "workspace")
	}
	if req.ScopeID == nil || *req.ScopeID != "scope-123" {
		t.Errorf("ScopeID = %v, want %v", req.ScopeID, &scopeID)
	}
	if req.RoleID != "role-123" {
		t.Errorf("RoleID = %v, want %v", req.RoleID, "role-123")
	}
	if req.AssignedBy == nil || *req.AssignedBy != "user-admin" {
		t.Errorf("AssignedBy = %v, want %v", req.AssignedBy, &assignedBy)
	}
}

func TestBulkAssignRoleRequestMinimal(t *testing.T) {
	req := dto.BulkAssignRoleRequest{
		UserIDs:   []string{"user-1"},
		ScopeType: "system",
		RoleID:    "role-123",
	}

	if len(req.UserIDs) != 1 {
		t.Errorf("len(UserIDs) = %v, want %v", len(req.UserIDs), 1)
	}
	if req.ScopeType != "system" {
		t.Errorf("ScopeType = %v, want %v", req.ScopeType, "system")
	}
	if req.ScopeID != nil {
		t.Errorf("ScopeID = %v, want nil", req.ScopeID)
	}
	if req.AssignedBy != nil {
		t.Errorf("AssignedBy = %v, want nil", req.AssignedBy)
	}
}
