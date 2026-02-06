package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkspace_TableName(t *testing.T) {
	workspace := tenant.Workspace{}
	schema := "test_schema"

	tableName := workspace.TableName(schema)

	assert.Equal(t, `"test_schema".workspaces`, tableName)
}

func TestWorkspace_Fields(t *testing.T) {
	workspaceID := uuid.New()
	now := time.Now().UTC()
	desc := "Test workspace description"

	workspace := tenant.Workspace{
		ID:          workspaceID,
		Title:       "My Workspace",
		Description: &desc,
		Slug:        "my-workspace",
		IsDefault:   true,
		Status:      "active",
		CreatedBy:   "user123",
		UpdatedBy:   "user456",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, workspaceID, workspace.ID)
	assert.Equal(t, "My Workspace", workspace.Title)
	assert.Equal(t, "Test workspace description", *workspace.Description)
	assert.Equal(t, "my-workspace", workspace.Slug)
	assert.True(t, workspace.IsDefault)
	assert.Equal(t, "active", workspace.Status)
	assert.Equal(t, "user123", workspace.CreatedBy)
	assert.Equal(t, "user456", workspace.UpdatedBy)
}

func TestWorkspace_WithMeta(t *testing.T) {
	meta := map[string]interface{}{
		"color": "blue",
		"icon":  "workspace",
	}

	workspace := tenant.Workspace{
		ID:    uuid.New(),
		Title: "Test Workspace",
		Meta:  meta,
	}

	assert.Equal(t, "blue", workspace.Meta["color"])
	assert.Equal(t, "workspace", workspace.Meta["icon"])
}

func TestWorkspace_WithNilDescription(t *testing.T) {
	workspace := tenant.Workspace{
		ID:          uuid.New(),
		Title:       "Workspace",
		Description: nil,
	}

	assert.Nil(t, workspace.Description)
}

func TestWorkspace_TableSchema(t *testing.T) {
	workspace := tenant.Workspace{}
	schema := "test_schema"

	tableSchema := workspace.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".workspaces`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
}
