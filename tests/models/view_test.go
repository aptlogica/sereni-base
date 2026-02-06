package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestView_TableName(t *testing.T) {
	view := tenant.View{}
	schema := "test_schema"

	tableName := view.TableName(schema)

	assert.Equal(t, `"test_schema".views`, tableName)
}

func TestView_Fields(t *testing.T) {
	viewID := uuid.New()
	modelID := uuid.New().String()
	baseID := uuid.New().String()
	now := time.Now().UTC()
	alias := "grid_view"
	desc := "Main grid view"

	view := tenant.View{
		ID:          viewID,
		ModelID:     modelID,
		BaseID:      baseID,
		Title:       "Grid View",
		Alias:       &alias,
		Description: &desc,
		Type:        "grid",
		IsDefault:   true,
		Public:      false,
		CreatedBy:   "user123",
		UpdatedBy:   "user123",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, viewID, view.ID)
	assert.Equal(t, modelID, view.ModelID)
	assert.Equal(t, baseID, view.BaseID)
	assert.Equal(t, "Grid View", view.Title)
	assert.Equal(t, "grid_view", *view.Alias)
	assert.Equal(t, "Main grid view", *view.Description)
	assert.Equal(t, "grid", view.Type)
	assert.True(t, view.IsDefault)
	assert.False(t, view.Public)
}

func TestView_WithMeta(t *testing.T) {
	meta := map[string]interface{}{
		"columns":     []string{"col1", "col2"},
		"row_height":  40,
		"show_totals": true,
	}

	view := tenant.View{
		ID:      uuid.New(),
		ModelID: uuid.New().String(),
		BaseID:  uuid.New().String(),
		Title:   "Test View",
		Meta:    meta,
	}

	assert.Equal(t, 40, view.Meta["row_height"])
	assert.Equal(t, true, view.Meta["show_totals"])
}

func TestView_WithLockAndPassword(t *testing.T) {
	lockType := "password"
	password := "encrypted_password_hash"

	view := tenant.View{
		ID:       uuid.New(),
		ModelID:  uuid.New().String(),
		BaseID:   uuid.New().String(),
		Title:    "Locked View",
		LockType: &lockType,
		Password: &password,
	}

	assert.Equal(t, "password", *view.LockType)
	assert.Equal(t, "encrypted_password_hash", *view.Password)
}

func TestView_PublicWithUUID(t *testing.T) {
	publicUUID := uuid.New().String()

	view := tenant.View{
		ID:      uuid.New(),
		ModelID: uuid.New().String(),
		BaseID:  uuid.New().String(),
		Title:   "Public View",
		Public:  true,
		UUID:    &publicUUID,
	}

	assert.True(t, view.Public)
	assert.NotNil(t, view.UUID)
	assert.Equal(t, publicUUID, *view.UUID)
}

func TestView_WithOrderIndex(t *testing.T) {
	orderIndex := 1.5

	view := tenant.View{
		ID:         uuid.New(),
		ModelID:    uuid.New().String(),
		BaseID:     uuid.New().String(),
		Title:      "Ordered View",
		OrderIndex: &orderIndex,
	}

	assert.Equal(t, 1.5, *view.OrderIndex)
}

func TestView_TableSchema(t *testing.T) {
	view := tenant.View{}
	schema := "test_schema"

	tableSchema := view.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".views`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
