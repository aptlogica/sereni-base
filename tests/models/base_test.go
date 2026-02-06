package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBase_TableName(t *testing.T) {
	base := tenant.Base{}
	schema := "test_schema"

	tableName := base.TableName(schema)

	assert.Equal(t, `"test_schema".bases`, tableName)
}

func TestBase_Fields(t *testing.T) {
	baseID := uuid.New()
	workspaceID := uuid.New().String()
	now := time.Now().UTC()
	desc := "Test description"

	base := tenant.Base{
		ID:          baseID,
		WorkspaceID: workspaceID,
		Title:       "Test Base",
		Description: &desc,
		Type:        "internal",
		Status:      "active",
		Visibility:  "public",
		CreatedBy:   "user123",
		UpdatedBy:   "user123",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, baseID, base.ID)
	assert.Equal(t, workspaceID, base.WorkspaceID)
	assert.Equal(t, "Test Base", base.Title)
	assert.Equal(t, "Test description", *base.Description)
	assert.Equal(t, "internal", base.Type)
	assert.Equal(t, "active", base.Status)
	assert.Equal(t, "public", base.Visibility)
}

func TestBase_WithConfig(t *testing.T) {
	config := map[string]interface{}{
		"host": "localhost",
		"port": 5432,
	}

	settings := map[string]interface{}{
		"theme": "dark",
	}

	base := tenant.Base{
		ID:       uuid.New(),
		Config:   config,
		Settings: settings,
	}

	assert.Equal(t, "localhost", base.Config["host"])
	assert.Equal(t, 5432, base.Config["port"])
	assert.Equal(t, "dark", base.Settings["theme"])
}

func TestBase_ResourceTracking(t *testing.T) {
	base := tenant.Base{
		ID:               uuid.New(),
		TableCount:       10,
		RowCount:         1000,
		StorageUsedBytes: 1024000,
	}

	assert.Equal(t, 10, base.TableCount)
	assert.Equal(t, int64(1000), base.RowCount)
	assert.Equal(t, int64(1024000), base.StorageUsedBytes)
}

func TestBase_TableSchema(t *testing.T) {
	base := tenant.Base{}
	schema := "test_schema"

	tableSchema := base.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".bases`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
