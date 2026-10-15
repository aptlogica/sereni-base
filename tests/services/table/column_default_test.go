package table_test

import (
	"context"
	"errors"
	"testing"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func modifyColumnReq(t *testing.T, args mock.Arguments) dbModels.ModifyColumnRequest {
	t.Helper()
	req := args.Get(1).(dbModels.AlterTableRequest)
	assert.Equal(t, "modify_column", req.Action)
	return req.Data.(dbModels.ModifyColumnRequest)
}

func TestAddColumn_AppliesMetaDefaultValue(t *testing.T) {
	cases := []struct {
		name    string
		dt      string
		value   interface{}
		want    string
		wantErr bool
	}{
		{"text is quoted and escaped", "TEXT", "O'Brien", "'O''Brien'", false},
		{"integer from float", "INTEGER", float64(5), "5", false},
		{"integer from int", "INTEGER", 8, "8", false},
		{"integer from string", "INTEGER", " 7 ", "7", false},
		{"bool", "BOOLEAN", true, "true", false},
		{"bool from string", "BOOLEAN", "false", "false", false},
		{"invalid number", "INTEGER", "1; DROP TABLE x", "", true},
		{"invalid bool", "BOOLEAN", "maybe", "", true},
		{"unsupported value type", "TEXT", []string{"a"}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
			modelID, baseID := uuid.New(), uuid.New()
			col := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID.String(), ColumnName: "col", Title: "T", UIDT: "text", DT: helpers.StringPtr(tc.dt)}
			mockColumn.On("Create", mock.Anything, mock.Anything, "schema").Return(col, nil)
			mockColumn.On("DeleteColumn", mock.Anything, "schema", col.ID.String()).Return(nil).Maybe()
			mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
			mockTable.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
			mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)

			_, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{
				ModelID: modelID, BaseID: baseID, Title: "T", UIDT: "text",
				Meta: map[string]interface{}{"defaultValue": tc.value},
			})

			if tc.wantErr {
				assert.ErrorIs(t, err, app_errors.InvalidPayload)
				mockTable.AssertNotCalled(t, "AlterTable", mock.Anything, mock.MatchedBy(func(r dbModels.AlterTableRequest) bool { return r.Action == "modify_column" }))
				return
			}
			assert.NoError(t, err)
			for _, c := range mockTable.Calls {
				if c.Method == "AlterTable" && c.Arguments.Get(1).(dbModels.AlterTableRequest).Action == "modify_column" {
					req := modifyColumnReq(t, c.Arguments)
					assert.Equal(t, tc.want, *req.SetDefault)
					assert.False(t, req.DropDefault)
					return
				}
			}
			t.Fatal("expected a modify_column AlterTable call")
		})
	}
}

func TestAddColumn_NilDefaultDropsDefault(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
	modelID, baseID := uuid.New(), uuid.New()
	col := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID.String(), ColumnName: "col", UIDT: "text", DT: helpers.StringPtr("TEXT")}
	mockColumn.On("Create", mock.Anything, mock.Anything, "schema").Return(col, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
	mockTable.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
	mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{
		ModelID: modelID, BaseID: baseID, Title: "T", UIDT: "text",
		Meta: map[string]interface{}{"defaultValue": nil},
	})

	assert.NoError(t, err)
	req := modifyColumnReq(t, mockTable.Calls[len(mockTable.Calls)-1].Arguments)
	assert.True(t, req.DropDefault)
	assert.Nil(t, req.SetDefault)
}

func TestAddColumn_DefaultAlterFailureRollsBack(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
	modelID, baseID := uuid.New(), uuid.New()
	col := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID.String(), ColumnName: "col", UIDT: "text", DT: helpers.StringPtr("TEXT")}
	mockColumn.On("Create", mock.Anything, mock.Anything, "schema").Return(col, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", col.ID.String()).Return(nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
	mockTable.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
	mockTable.On("AlterTable", mock.Anything, mock.MatchedBy(func(r dbModels.AlterTableRequest) bool { return r.Action == "modify_column" })).Return(errors.New("boom"))
	mockTable.On("AlterTable", mock.Anything, mock.MatchedBy(func(r dbModels.AlterTableRequest) bool { return r.Action == "drop_column" })).Return(nil)

	_, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{
		ModelID: modelID, BaseID: baseID, Title: "T", UIDT: "text",
		Meta: map[string]interface{}{"defaultValue": "x"},
	})

	assert.Error(t, err)
	mockColumn.AssertCalled(t, "DeleteColumn", mock.Anything, "schema", col.ID.String())
}

func TestUpdateColumn_AppliesMetaDefaultValue(t *testing.T) {
	setup := func(altErr error) (*MockTableService, *MockColumnService, func(meta map[string]interface{}) error) {
		_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
		modelID, colID := uuid.New(), uuid.New()
		col := tenant.Column{ID: colID, ModelID: modelID.String(), BaseID: uuid.New().String(), ColumnName: "col", UIDT: "number", DT: helpers.StringPtr("INTEGER")}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", colID.String()).Return(col, nil)
		mockColumn.On("UpdateColumn", mock.Anything, "schema", colID.String(), mock.Anything).Return(col, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
		mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(altErr)
		return mockTable, mockColumn, func(meta map[string]interface{}) error {
			_, err := svc.UpdateColumn(context.Background(), "schema", colID.String(), dto.ColumnUpdate{Meta: &meta})
			return err
		}
	}

	t.Run("sets default", func(t *testing.T) {
		mockTable, _, update := setup(nil)
		assert.NoError(t, update(map[string]interface{}{"defaultValue": float64(10)}))
		req := modifyColumnReq(t, mockTable.Calls[0].Arguments)
		assert.Equal(t, "10", *req.SetDefault)
	})

	t.Run("empty drops default", func(t *testing.T) {
		mockTable, _, update := setup(nil)
		assert.NoError(t, update(map[string]interface{}{"defaultValue": ""}))
		assert.True(t, modifyColumnReq(t, mockTable.Calls[0].Arguments).DropDefault)
	})

	t.Run("meta without key leaves default alone", func(t *testing.T) {
		mockTable, _, update := setup(nil)
		assert.NoError(t, update(map[string]interface{}{"other": 1}))
		mockTable.AssertNotCalled(t, "AlterTable", mock.Anything, mock.Anything)
	})

	t.Run("alter failure reverts metadata", func(t *testing.T) {
		_, mockColumn, update := setup(errors.New("boom"))
		assert.Error(t, update(map[string]interface{}{"defaultValue": "1"}))
		mockColumn.AssertNumberOfCalls(t, "UpdateColumn", 2)
	})

	t.Run("invalid value", func(t *testing.T) {
		_, _, update := setup(nil)
		assert.ErrorIs(t, update(map[string]interface{}{"defaultValue": "abc"}), app_errors.InvalidPayload)
	})
}
