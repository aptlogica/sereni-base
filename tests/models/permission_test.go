package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPermission_TableName(t *testing.T) {
	permission := tenant.Permission{}
	schema := "test_schema"

	tableName := permission.TableName(schema)

	assert.Equal(t, `"test_schema".permissions`, tableName)
}

func TestPermission_Fields(t *testing.T) {
	permissionID := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()
	now := time.Now().UTC()

	permission := tenant.Permission{
		ID:         permissionID,
		ResourceID: resourceID,
		ActionID:   actionID,
		CreatedAt:  now,
	}

	assert.Equal(t, permissionID, permission.ID)
	assert.Equal(t, resourceID, permission.ResourceID)
	assert.Equal(t, actionID, permission.ActionID)
	assert.Equal(t, now, permission.CreatedAt)
}

func TestPermission_ResourceActionCombination(t *testing.T) {
	// workspace.read
	workspaceReadPerm := tenant.Permission{
		ID:         uuid.New(),
		ResourceID: uuid.New(), // workspace resource
		ActionID:   uuid.New(), // read action
		CreatedAt:  time.Now().UTC(),
	}

	// base.update
	baseUpdatePerm := tenant.Permission{
		ID:         uuid.New(),
		ResourceID: uuid.New(), // base resource
		ActionID:   uuid.New(), // update action
		CreatedAt:  time.Now().UTC(),
	}

	// records.delete
	recordsDeletePerm := tenant.Permission{
		ID:         uuid.New(),
		ResourceID: uuid.New(), // records resource
		ActionID:   uuid.New(), // delete action
		CreatedAt:  time.Now().UTC(),
	}

	assert.NotEqual(t, workspaceReadPerm.ResourceID, baseUpdatePerm.ResourceID)
	assert.NotEqual(t, workspaceReadPerm.ActionID, recordsDeletePerm.ActionID)
}

func TestPermission_TableSchema(t *testing.T) {
	permission := tenant.Permission{}
	schema := "test_schema"

	tableSchema := permission.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".permissions`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
