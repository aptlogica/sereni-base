package models_test

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRolePermission_TableName(t *testing.T) {
	rolePerm := tenant.RolePermission{}
	schema := "test_schema"

	tableName := rolePerm.TableName(schema)

	assert.Equal(t, `"test_schema".role_permissions`, tableName)
}

func TestRolePermission_Fields(t *testing.T) {
	rolePermID := uuid.New()
	roleID := uuid.New()
	permissionID := uuid.New()
	now := time.Now().UTC()

	rolePerm := tenant.RolePermission{
		ID:           rolePermID,
		RoleID:       roleID,
		PermissionID: permissionID,
		CreatedAt:    now,
	}

	assert.Equal(t, rolePermID, rolePerm.ID)
	assert.Equal(t, roleID, rolePerm.RoleID)
	assert.Equal(t, permissionID, rolePerm.PermissionID)
	assert.Equal(t, now, rolePerm.CreatedAt)
}

func TestRolePermission_AdminRole(t *testing.T) {
	adminRoleID := uuid.New()

	// Admin has multiple permissions
	permissions := []tenant.RolePermission{
		{
			ID:           uuid.New(),
			RoleID:       adminRoleID,
			PermissionID: uuid.New(), // workspace.read
			CreatedAt:    time.Now().UTC(),
		},
		{
			ID:           uuid.New(),
			RoleID:       adminRoleID,
			PermissionID: uuid.New(), // workspace.create
			CreatedAt:    time.Now().UTC(),
		},
		{
			ID:           uuid.New(),
			RoleID:       adminRoleID,
			PermissionID: uuid.New(), // workspace.update
			CreatedAt:    time.Now().UTC(),
		},
		{
			ID:           uuid.New(),
			RoleID:       adminRoleID,
			PermissionID: uuid.New(), // workspace.delete
			CreatedAt:    time.Now().UTC(),
		},
	}

	for _, perm := range permissions {
		assert.Equal(t, adminRoleID, perm.RoleID)
		assert.NotEqual(t, uuid.Nil, perm.PermissionID)
	}
}

func TestRolePermission_ViewerRole(t *testing.T) {
	viewerRoleID := uuid.New()

	// Viewer has limited permissions
	readPermission := tenant.RolePermission{
		ID:           uuid.New(),
		RoleID:       viewerRoleID,
		PermissionID: uuid.New(), // workspace.read only
		CreatedAt:    time.Now().UTC(),
	}

	assert.Equal(t, viewerRoleID, readPermission.RoleID)
}

func TestRolePermission_TableSchema(t *testing.T) {
	rolePerm := tenant.RolePermission{}
	schema := "test_schema"

	tableSchema := rolePerm.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".role_permissions`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
