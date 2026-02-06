package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestViewColumn_TableName(t *testing.T) {
	viewCol := tenant.ViewColumn{}
	schema := "test_schema"

	tableName := viewCol.TableName(schema)

	assert.Equal(t, `"test_schema".view_columns`, tableName)
}

func TestViewColumn_Fields(t *testing.T) {
	viewColID := uuid.New()
	viewID := uuid.New().String()
	columnID := uuid.New().String()
	now := time.Now().UTC()
	orderIndex := 1.5
	width := "200px"

	viewCol := tenant.ViewColumn{
		ID:         viewColID,
		ViewID:     viewID,
		ColumnID:   columnID,
		ShowColumn: true,
		OrderIndex: &orderIndex,
		Width:      &width,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	assert.Equal(t, viewColID, viewCol.ID)
	assert.Equal(t, viewID, viewCol.ViewID)
	assert.Equal(t, columnID, viewCol.ColumnID)
	assert.True(t, viewCol.ShowColumn)
	assert.Equal(t, 1.5, *viewCol.OrderIndex)
	assert.Equal(t, "200px", *viewCol.Width)
}

func TestViewColumn_HiddenColumn(t *testing.T) {
	viewCol := tenant.ViewColumn{
		ID:         uuid.New(),
		ViewID:     uuid.New().String(),
		ColumnID:   uuid.New().String(),
		ShowColumn: false,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	assert.False(t, viewCol.ShowColumn)
}

func TestViewColumn_WithOrderIndex(t *testing.T) {
	columns := []tenant.ViewColumn{
		{
			ID:         uuid.New(),
			ViewID:     uuid.New().String(),
			ColumnID:   uuid.New().String(),
			ShowColumn: true,
			OrderIndex: floatPtr(1.0),
		},
		{
			ID:         uuid.New(),
			ViewID:     uuid.New().String(),
			ColumnID:   uuid.New().String(),
			ShowColumn: true,
			OrderIndex: floatPtr(2.0),
		},
		{
			ID:         uuid.New(),
			ViewID:     uuid.New().String(),
			ColumnID:   uuid.New().String(),
			ShowColumn: true,
			OrderIndex: floatPtr(3.0),
		},
	}

	assert.Equal(t, 1.0, *columns[0].OrderIndex)
	assert.Equal(t, 2.0, *columns[1].OrderIndex)
	assert.Equal(t, 3.0, *columns[2].OrderIndex)
}

func TestViewColumn_WithCustomWidth(t *testing.T) {
	testCases := []struct {
		name  string
		width string
	}{
		{"small", "100px"},
		{"medium", "200px"},
		{"large", "400px"},
		{"percentage", "25%"},
		{"auto", "auto"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			viewCol := tenant.ViewColumn{
				ID:         uuid.New(),
				ViewID:     uuid.New().String(),
				ColumnID:   uuid.New().String(),
				ShowColumn: true,
				Width:      &tc.width,
			}

			assert.Equal(t, tc.width, *viewCol.Width)
		})
	}
}

func TestViewColumn_WithoutOptionalFields(t *testing.T) {
	viewCol := tenant.ViewColumn{
		ID:         uuid.New(),
		ViewID:     uuid.New().String(),
		ColumnID:   uuid.New().String(),
		ShowColumn: true,
		OrderIndex: nil,
		Width:      nil,
	}

	assert.Nil(t, viewCol.OrderIndex)
	assert.Nil(t, viewCol.Width)
}

func TestViewColumn_TableSchema(t *testing.T) {
	viewCol := tenant.ViewColumn{}
	schema := "test_schema"

	tableSchema := viewCol.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".view_columns`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}

// Helper function
func floatPtr(f float64) *float64 {
	return &f
}
