package table_test

import (
	"context"
	"errors"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// keyFixture is a table "org" keyed by (code, dept) with one plain column "code"
func keyFixture(t *testing.T) (*relFixture, tenant.Column) {
	t.Helper()
	f := &relFixture{
		table: &MockTableService{}, bulk: &MockBulkService{}, model: &MockModelService{},
		column: &MockColumnService{}, view: &MockViewService{}, rel: &MockRelationshipService{},
	}
	f.table.On("GetByFunction", mock.Anything, "public.link_key_columns", map[string]interface{}{"p_table": `"schema"."org"`}).
		Return([]map[string]interface{}{{"link_key_columns": "{code,dept}"}}, nil).Maybe()
	f = withService(f)

	modelID := uuid.New()
	col := plainCol(uuid.New(), modelID.String(), "code", "text")
	f.model.On("GetModelByID", mock.Anything, relSchema, modelID.String()).Return(tenant.Model{ID: modelID, Alias: "org"}, nil)
	f.column.On("GetColumnByID", mock.Anything, relSchema, col.ID.String()).Return(col, nil)
	return f, col
}

func TestDeleteColumn_BlockedWhileInPrimaryKey(t *testing.T) {
	f, col := keyFixture(t)

	err := f.svc.DeleteColumn(context.Background(), relSchema, col.ID.String())

	assert.ErrorIs(t, err, app_errors.ColumnInPrimaryKey)
	f.column.AssertNotCalled(t, "DeleteColumn", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateColumn_TypeChangeBlockedWhileInPrimaryKey(t *testing.T) {
	t.Run("type change is blocked", func(t *testing.T) {
		f, col := keyFixture(t)
		uidt := "number"

		_, err := f.svc.UpdateColumn(context.Background(), relSchema, col.ID.String(), dto.ColumnUpdate{UIDT: &uidt})

		assert.ErrorIs(t, err, app_errors.ColumnInPrimaryKey)
		f.column.AssertNotCalled(t, "UpdateColumn", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("renaming a key field is allowed", func(t *testing.T) {
		f, col := keyFixture(t)
		title := "Org code"
		f.column.On("UpdateColumn", mock.Anything, relSchema, col.ID.String(), mock.Anything).Return(col, nil)

		_, err := f.svc.UpdateColumn(context.Background(), relSchema, col.ID.String(), dto.ColumnUpdate{Title: &title})

		assert.NoError(t, err)
		f.table.AssertNotCalled(t, "GetByFunction", mock.Anything, "public.link_key_columns", mock.Anything)
	})

	t.Run("key lookup failure stops the change", func(t *testing.T) {
		f := &relFixture{
			table: &MockTableService{}, bulk: &MockBulkService{}, model: &MockModelService{},
			column: &MockColumnService{}, view: &MockViewService{}, rel: &MockRelationshipService{},
		}
		f.table.On("GetByFunction", mock.Anything, "public.link_key_columns", mock.Anything).Return(nil, errors.New("down"))
		f = withService(f)
		modelID := uuid.New()
		col := plainCol(uuid.New(), modelID.String(), "code", "text")
		f.model.On("GetModelByID", mock.Anything, relSchema, modelID.String()).Return(tenant.Model{ID: modelID, Alias: "org"}, nil)
		f.column.On("GetColumnByID", mock.Anything, relSchema, col.ID.String()).Return(col, nil)
		uidt := "number"

		_, err := f.svc.UpdateColumn(context.Background(), relSchema, col.ID.String(), dto.ColumnUpdate{UIDT: &uidt})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})
}
