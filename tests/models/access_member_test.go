package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccessMember_TableName(t *testing.T) {
	member := tenant.AccessMember{}
	schema := "test_schema"

	tableName := member.TableName(schema)

	assert.Equal(t, `"test_schema".access_members`, tableName)
}

func TestAccessMember_SystemLevel(t *testing.T) {
	memberID := uuid.New()
	userID := uuid.New().String()
	roleID := uuid.New().String()
	assignedBy := "admin@example.com"
	now := time.Now().UTC()

	member := tenant.AccessMember{
		ID:          memberID,
		UserID:      userID,
		ScopeType:   "system",
		ScopeID:     nil,
		RoleID:      roleID,
		WorkspaceID: nil,
		AssignedBy:  &assignedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, memberID, member.ID)
	assert.Equal(t, userID, member.UserID)
	assert.Equal(t, "system", member.ScopeType)
	assert.Nil(t, member.ScopeID)
	assert.Equal(t, roleID, member.RoleID)
	assert.Nil(t, member.WorkspaceID)
	assert.Equal(t, "admin@example.com", *member.AssignedBy)
}

func TestAccessMember_WorkspaceLevel(t *testing.T) {
	workspaceID := uuid.New().String()
	scopeID := workspaceID

	member := tenant.AccessMember{
		ID:          uuid.New(),
		UserID:      uuid.New().String(),
		ScopeType:   "workspace",
		ScopeID:     &scopeID,
		RoleID:      uuid.New().String(),
		WorkspaceID: &workspaceID,
	}

	assert.Equal(t, "workspace", member.ScopeType)
	assert.NotNil(t, member.ScopeID)
	assert.Equal(t, workspaceID, *member.ScopeID)
	assert.NotNil(t, member.WorkspaceID)
	assert.Equal(t, workspaceID, *member.WorkspaceID)
}

func TestAccessMember_BaseLevel(t *testing.T) {
	baseID := uuid.New().String()
	workspaceID := uuid.New().String()
	scopeID := baseID

	member := tenant.AccessMember{
		ID:          uuid.New(),
		UserID:      uuid.New().String(),
		ScopeType:   "base",
		ScopeID:     &scopeID,
		RoleID:      uuid.New().String(),
		WorkspaceID: &workspaceID,
	}

	assert.Equal(t, "base", member.ScopeType)
	assert.NotNil(t, member.ScopeID)
	assert.Equal(t, baseID, *member.ScopeID)
	assert.NotNil(t, member.WorkspaceID)
	assert.Equal(t, workspaceID, *member.WorkspaceID)
}

func TestAccessMember_WithoutAssignedBy(t *testing.T) {
	member := tenant.AccessMember{
		ID:         uuid.New(),
		UserID:     uuid.New().String(),
		ScopeType:  "workspace",
		RoleID:     uuid.New().String(),
		AssignedBy: nil,
	}

	assert.Nil(t, member.AssignedBy)
}

func TestAccessMember_TableSchema(t *testing.T) {
	member := tenant.AccessMember{}
	schema := "test_schema"

	tableSchema := member.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".access_members`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
