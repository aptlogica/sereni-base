package models_test

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestModel_TableName(t *testing.T) {
	model := tenant.Model{}
	schema := "test_schema"

	tableName := model.TableName(schema)

	assert.Equal(t, `"test_schema".models`, tableName)
}

func TestModel_Fields(t *testing.T) {
	modelID := uuid.New()
	baseID := uuid.New()
	workspaceID := uuid.New()
	now := time.Now().UTC()
	desc := "Test model"
	tags := "tag1,tag2"

	model := tenant.Model{
		ID:          modelID,
		BaseID:      baseID,
		WorkspaceID: workspaceID,
		Title:       "Users Table",
		Description: &desc,
		Alias:       "users",
		Type:        "table",
		Enabled:     true,
		MM:          false,
		Pinned:      true,
		Deleted:     false,
		Tags:        &tags,
		CreatedBy:   "user123",
		UpdatedBy:   "user456",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, modelID, model.ID)
	assert.Equal(t, baseID, model.BaseID)
	assert.Equal(t, workspaceID, model.WorkspaceID)
	assert.Equal(t, "Users Table", model.Title)
	assert.Equal(t, "Test model", *model.Description)
	assert.Equal(t, "users", model.Alias)
	assert.Equal(t, "table", model.Type)
	assert.True(t, model.Enabled)
	assert.False(t, model.MM)
	assert.True(t, model.Pinned)
	assert.False(t, model.Deleted)
	assert.Equal(t, "tag1,tag2", *model.Tags)
}

func TestModel_ManyToMany(t *testing.T) {
	model := tenant.Model{
		ID:      uuid.New(),
		BaseID:  uuid.New(),
		Title:   "Junction Table",
		Alias:   "user_groups",
		Type:    "junction",
		MM:      true,
		Enabled: true,
	}

	assert.True(t, model.MM)
	assert.Equal(t, "junction", model.Type)
}

func TestModel_WithMeta(t *testing.T) {
	meta := map[string]interface{}{
		"icon":  "table",
		"color": "blue",
	}

	model := tenant.Model{
		ID:      uuid.New(),
		BaseID:  uuid.New(),
		Title:   "Tasks",
		Alias:   "tasks",
		Type:    "table",
		Meta:    meta,
		Enabled: true,
	}

	assert.Equal(t, "table", model.Meta["icon"])
	assert.Equal(t, "blue", model.Meta["color"])
}

func TestModel_ResourceTracking(t *testing.T) {
	model := tenant.Model{
		ID:               uuid.New(),
		BaseID:           uuid.New(),
		Title:            "Large Table",
		Alias:            "large_table",
		RowCount:         100000,
		ColumnCount:      25,
		StorageUsedBytes: 10485760, // 10 MB
	}

	assert.Equal(t, int64(100000), model.RowCount)
	assert.Equal(t, 25, model.ColumnCount)
	assert.Equal(t, int64(10485760), model.StorageUsedBytes)
}

func TestModel_WithOrderIndex(t *testing.T) {
	orderIndex := 2.5

	model := tenant.Model{
		ID:         uuid.New(),
		BaseID:     uuid.New(),
		Title:      "Ordered Model",
		Alias:      "ordered_model",
		OrderIndex: &orderIndex,
	}

	assert.Equal(t, 2.5, *model.OrderIndex)
}

func TestModel_DeletedModel(t *testing.T) {
	model := tenant.Model{
		ID:      uuid.New(),
		BaseID:  uuid.New(),
		Title:   "Old Model",
		Alias:   "old_model",
		Deleted: true,
		Enabled: false,
	}

	assert.True(t, model.Deleted)
	assert.False(t, model.Enabled)
}

func TestModel_TableSchema(t *testing.T) {
	model := tenant.Model{}
	schema := "test_schema"

	tableSchema := model.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".models`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
