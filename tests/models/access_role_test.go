package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccessRole_TableName(t *testing.T) {
	role := tenant.AccessRole{}
	schema := "test_schema"

	tableName := role.TableName(schema)

	assert.Equal(t, `"test_schema".access_roles`, tableName)
}

func TestAccessRole_SystemLevel(t *testing.T) {
	roleID := uuid.New()
	now := time.Now().UTC()
	desc := "System administrator"

	role := tenant.AccessRole{
		ID:          roleID,
		Name:        "admin",
		ScopeLevel:  "system",
		Priority:    100,
		Description: &desc,
		IsDefault:   false,
		WorkspaceID: nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, roleID, role.ID)
	assert.Equal(t, "admin", role.Name)
	assert.Equal(t, "system", role.ScopeLevel)
	assert.Equal(t, 100, role.Priority)
	assert.Equal(t, "System administrator", *role.Description)
	assert.False(t, role.IsDefault)
	assert.Nil(t, role.WorkspaceID)
}

func TestAccessRole_WorkspaceLevel(t *testing.T) {
	role := tenant.AccessRole{
		ID:         uuid.New(),
		Name:       "workspace_owner",
		ScopeLevel: "workspace",
		Priority:   90,
		IsDefault:  false,
	}

	assert.Equal(t, "workspace_owner", role.Name)
	assert.Equal(t, "workspace", role.ScopeLevel)
	assert.Equal(t, 90, role.Priority)
}

func TestAccessRole_BaseLevel(t *testing.T) {
	workspaceID := uuid.New().String()

	role := tenant.AccessRole{
		ID:          uuid.New(),
		Name:        "base_editor",
		ScopeLevel:  "base",
		Priority:    50,
		IsDefault:   true,
		WorkspaceID: &workspaceID,
	}

	assert.Equal(t, "base_editor", role.Name)
	assert.Equal(t, "base", role.ScopeLevel)
	assert.Equal(t, 50, role.Priority)
	assert.True(t, role.IsDefault)
	assert.NotNil(t, role.WorkspaceID)
	assert.Equal(t, workspaceID, *role.WorkspaceID)
}

func TestAccessRole_Priority(t *testing.T) {
	adminRole := tenant.AccessRole{
		ID:         uuid.New(),
		Name:       "admin",
		ScopeLevel: "system",
		Priority:   100,
	}

	memberRole := tenant.AccessRole{
		ID:         uuid.New(),
		Name:       "member",
		ScopeLevel: "workspace",
		Priority:   30,
	}

	viewerRole := tenant.AccessRole{
		ID:         uuid.New(),
		Name:       "viewer",
		ScopeLevel: "base",
		Priority:   10,
	}

	assert.Greater(t, adminRole.Priority, memberRole.Priority)
	assert.Greater(t, memberRole.Priority, viewerRole.Priority)
}

func TestAccessRole_TableSchema(t *testing.T) {
	role := tenant.AccessRole{}
	schema := "test_schema"

	tableSchema := role.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".access_roles`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
