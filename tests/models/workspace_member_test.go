package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkspaceMember_TableName(t *testing.T) {
	member := tenant.WorkspaceMember{}
	schema := "test_schema"

	tableName := member.TableName(schema)

	assert.Equal(t, `"test_schema".workspace_members`, tableName)
}

func TestWorkspaceMember_Fields(t *testing.T) {
	memberID := uuid.New()
	workspaceID := uuid.New().String()
	userID := uuid.New().String()
	now := time.Now().UTC()

	member := tenant.WorkspaceMember{
		ID:          memberID,
		WorkspaceID: workspaceID,
		UserID:      userID,
		BasesIds:    "base1,base2,base3",
		AccessLevel: "editor",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, memberID, member.ID)
	assert.Equal(t, workspaceID, member.WorkspaceID)
	assert.Equal(t, userID, member.UserID)
	assert.Equal(t, "base1,base2,base3", member.BasesIds)
	assert.Equal(t, "editor", member.AccessLevel)
}

func TestWorkspaceMember_AccessLevels(t *testing.T) {
	testCases := []struct {
		name        string
		accessLevel string
	}{
		{"owner", "owner"},
		{"editor", "editor"},
		{"viewer", "viewer"},
		{"maintainer", "maintainer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			member := tenant.WorkspaceMember{
				ID:          uuid.New(),
				WorkspaceID: uuid.New().String(),
				UserID:      uuid.New().String(),
				AccessLevel: tc.accessLevel,
			}

			assert.Equal(t, tc.accessLevel, member.AccessLevel)
		})
	}
}

func TestWorkspaceMember_WithMultipleBases(t *testing.T) {
	member := tenant.WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: uuid.New().String(),
		UserID:      uuid.New().String(),
		BasesIds:    "base-uuid-1,base-uuid-2,base-uuid-3,base-uuid-4",
		AccessLevel: "editor",
	}

	assert.Contains(t, member.BasesIds, "base-uuid-1")
	assert.Contains(t, member.BasesIds, "base-uuid-4")
}

func TestWorkspaceMember_WithNoBases(t *testing.T) {
	member := tenant.WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: uuid.New().String(),
		UserID:      uuid.New().String(),
		BasesIds:    "",
		AccessLevel: "viewer",
	}

	assert.Empty(t, member.BasesIds)
}

func TestWorkspaceMember_TableSchema(t *testing.T) {
	member := tenant.WorkspaceMember{}
	schema := "test_schema"

	tableSchema := member.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".workspace_members`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
}
