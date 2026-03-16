package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
)

func TestImportTableRequestFields(t *testing.T) {
	req := dto.ImportTableRequest{
		BaseID:      "base-123",
		WorkspaceID: "ws-123",
		TableName:   "Users",
		OrderIndex:  1,
		CreatedBy:   "user1",
	}

	if req.BaseID != "base-123" {
		t.Errorf("BaseID = %v, want %v", req.BaseID, "base-123")
	}
	if req.WorkspaceID != "ws-123" {
		t.Errorf("WorkspaceID = %v, want %v", req.WorkspaceID, "ws-123")
	}
	if req.TableName != "Users" {
		t.Errorf("TableName = %v, want %v", req.TableName, "Users")
	}
	if req.OrderIndex != 1 {
		t.Errorf("OrderIndex = %v, want %v", req.OrderIndex, 1)
	}
	if req.CreatedBy != "user1" {
		t.Errorf("CreatedBy = %v, want %v", req.CreatedBy, "user1")
	}
}

func TestUserResetTokenInsertionMap(t *testing.T) {
	insertion := dto.UserResetTokenInsertion{
		ID:       "token-123",
		UserID:   "user-456",
		Token:    "reset-token-abc",
		IssuedAt: "2024-01-01T12:00:00Z",
	}

	m := insertion.Map()

	if m["id"] != "token-123" {
		t.Errorf("Map() id = %v, want %v", m["id"], "token-123")
	}
	if m["user_id"] != "user-456" {
		t.Errorf("Map() user_id = %v, want %v", m["user_id"], "user-456")
	}
	if m["token"] != "reset-token-abc" {
		t.Errorf("Map() token = %v, want %v", m["token"], "reset-token-abc")
	}
	if m["issued_at"] != "2024-01-01T12:00:00Z" {
		t.Errorf("Map() issued_at = %v, want %v", m["issued_at"], "2024-01-01T12:00:00Z")
	}
}

func TestBaseRoleAccessFields(t *testing.T) {
	base := dto.BaseRoleAccess{
		BaseId:   "base-123",
		BaseName: "Test Base",
		Access:   "editor",
	}

	if base.BaseId != "base-123" {
		t.Errorf("BaseId = %v, want %v", base.BaseId, "base-123")
	}
	if base.BaseName != "Test Base" {
		t.Errorf("BaseName = %v, want %v", base.BaseName, "Test Base")
	}
	if base.Access != "editor" {
		t.Errorf("Access = %v, want %v", base.Access, "editor")
	}
}

func TestUserRolesAccessResponseFields(t *testing.T) {
	bases := []dto.BaseRoleAccess{
		{
			BaseId:   "base-1",
			BaseName: "Base 1",
			Access:   "viewer",
		},
		{
			BaseId:   "base-2",
			BaseName: "Base 2",
			Access:   "editor",
		},
	}

	response := dto.UserRolesAccessResponse{
		WorkspaceId:   "ws-123",
		WorkspaceName: "Test Workspace",
		Access:        "owner",
		Bases:         bases,
	}

	if response.WorkspaceId != "ws-123" {
		t.Errorf("WorkspaceId = %v, want %v", response.WorkspaceId, "ws-123")
	}
	if response.WorkspaceName != "Test Workspace" {
		t.Errorf("WorkspaceName = %v, want %v", response.WorkspaceName, "Test Workspace")
	}
	if response.Access != "owner" {
		t.Errorf("Access = %v, want %v", response.Access, "owner")
	}
	if len(response.Bases) != 2 {
		t.Errorf("len(Bases) = %v, want %v", len(response.Bases), 2)
	}
}

func TestUserRolesAccessList(t *testing.T) {
	list := dto.UserRolesAccessList{
		{
			WorkspaceId:   "ws-1",
			WorkspaceName: "Workspace 1",
			Access:        "owner",
			Bases:         []dto.BaseRoleAccess{},
		},
		{
			WorkspaceId:   "ws-2",
			WorkspaceName: "Workspace 2",
			Access:        "editor",
			Bases:         []dto.BaseRoleAccess{},
		},
	}

	if len(list) != 2 {
		t.Errorf("len(list) = %v, want %v", len(list), 2)
	}
	if list[0].WorkspaceId != "ws-1" {
		t.Errorf("list[0].WorkspaceId = %v, want %v", list[0].WorkspaceId, "ws-1")
	}
	if list[1].Access != "editor" {
		t.Errorf("list[1].Access = %v, want %v", list[1].Access, "editor")
	}
}
