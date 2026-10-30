package table_test

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
		{"single select", "TEXT", "2", "'2'", false},
		{"multi select array", "TEXT[]", []interface{}{"12", "13"}, `'{"12","13"}'`, false},
		{"array escapes quotes", "TEXT[]", []interface{}{`a"b`, "O'Brien"}, `'{"a\"b","O''Brien"}'`, false},
		{"array on non-multiSelect array column", "JSONB[]", []interface{}{"a"}, "", true},
		{"array on non-array column", "TEXT", []interface{}{"a"}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, sqlMock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { db.Close() })
			mockTable, mockModel, mockColumn := &MockTableService{}, &MockModelService{}, &MockColumnService{}
			repo := &pkg.DatabaseService{TableService: mockTable, BulkService: &MockBulkService{}, DB: db}
			svc := services.NewTableManagementService("postgres", repo, mockModel, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})
			modelID, baseID := uuid.New(), uuid.New()
			uidt := "text"
			if tc.dt == "TEXT[]" {
				uidt = "multiSelect"
			}
			if tc.name == "single select" {
				uidt = "select"
			}
			col := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID.String(), ColumnName: "col", Title: "T", UIDT: uidt, DT: helpers.StringPtr(tc.dt)}
			mockColumn.On("Create", mock.Anything, mock.Anything, "schema").Return(col, nil)
			mockColumn.On("DeleteColumn", mock.Anything, "schema", col.ID.String()).Return(nil).Maybe()
			mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
			mockTable.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
			mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)

			if !tc.wantErr {
				// A new column also gives its default to the records that already exist
				sqlMock.ExpectExec(`UPDATE "schema"."tbl" SET "col" = ` + regexp.QuoteMeta(tc.want) + ` WHERE "col" IS NULL`).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			_, err = svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{
				ModelID: modelID, BaseID: baseID, Title: "T", UIDT: "text",
				Meta: map[string]interface{}{"defaultValue": tc.value},
			})

			if tc.wantErr {
				assert.ErrorIs(t, err, app_errors.InvalidPayload)
				mockTable.AssertNotCalled(t, "AlterTable", mock.Anything, mock.MatchedBy(func(r dbModels.AlterTableRequest) bool { return r.Action == "modify_column" }))
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, sqlMock.ExpectationsWereMet())
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
		mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
		return mockTable, mockColumn, func(meta map[string]interface{}) error {
			_, err := svc.UpdateColumn(context.Background(), "schema", colID.String(), dto.ColumnUpdate{Meta: &meta})
			return err
		}
	}

	t.Run("sets default", func(t *testing.T) {
		mockTable, _, update := setup(nil)
		assert.NoError(t, update(map[string]interface{}{"defaultValue": float64(10)}))
		req := modifyColumnReq(t, mockTable.Calls[len(mockTable.Calls)-1].Arguments)
		assert.Equal(t, "10", *req.SetDefault)
	})

	t.Run("empty drops default", func(t *testing.T) {
		mockTable, _, update := setup(nil)
		assert.NoError(t, update(map[string]interface{}{"defaultValue": ""}))
		assert.True(t, modifyColumnReq(t, mockTable.Calls[len(mockTable.Calls)-1].Arguments).DropDefault)
	})

	t.Run("null drops default", func(t *testing.T) {
		mockTable, _, update := setup(nil)
		assert.NoError(t, update(map[string]interface{}{"defaultValue": nil}))
		assert.True(t, modifyColumnReq(t, mockTable.Calls[0].Arguments).DropDefault)
	})

	t.Run("meta without key drops default", func(t *testing.T) {
		mockTable, _, update := setup(nil)
		assert.NoError(t, update(map[string]interface{}{"other": 1}))
		assert.True(t, modifyColumnReq(t, mockTable.Calls[0].Arguments).DropDefault)
	})

	t.Run("nil meta leaves default alone", func(t *testing.T) {
		_, mockTable2, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
		modelID, colID := uuid.New(), uuid.New()
		col := tenant.Column{ID: colID, ModelID: modelID.String(), BaseID: uuid.New().String(), ColumnName: "col", UIDT: "number", DT: helpers.StringPtr("INTEGER")}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", colID.String()).Return(col, nil)
		mockColumn.On("UpdateColumn", mock.Anything, "schema", colID.String(), mock.Anything).Return(col, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
		title := "new title"
		_, err := svc.UpdateColumn(context.Background(), "schema", colID.String(), dto.ColumnUpdate{Title: &title})
		assert.NoError(t, err)
		mockTable2.AssertNotCalled(t, "AlterTable", mock.Anything, mock.Anything)
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

func TestAddColumn_FillFailureRollsBack(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	mockTable, mockModel, mockColumn := &MockTableService{}, &MockModelService{}, &MockColumnService{}
	repo := &pkg.DatabaseService{TableService: mockTable, BulkService: &MockBulkService{}, DB: db}
	svc := services.NewTableManagementService("postgres", repo, mockModel, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})
	modelID, baseID := uuid.New(), uuid.New()
	col := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID.String(), ColumnName: "col", UIDT: "select", DT: helpers.StringPtr("TEXT")}
	mockColumn.On("Create", mock.Anything, mock.Anything, "schema").Return(col, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", col.ID.String()).Return(nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
	mockTable.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
	mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
	sqlMock.ExpectExec(`UPDATE "schema"."tbl" SET "col" = 'a' WHERE "col" IS NULL`).WillReturnError(errors.New("boom"))

	_, err = svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{
		ModelID: modelID, BaseID: baseID, Title: "T", UIDT: "select",
		Meta: map[string]interface{}{"defaultValue": "a"},
	})

	assert.ErrorIs(t, err, app_errors.DatabaseError)
	mockColumn.AssertCalled(t, "DeleteColumn", mock.Anything, "schema", col.ID.String())
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestUpdateColumn_RemovedSelectOptions(t *testing.T) {
	options := func(names ...string) []interface{} {
		out := make([]interface{}, 0, len(names))
		for _, n := range names {
			out = append(out, map[string]interface{}{"option": n, "color": "#fff"})
		}
		return out
	}
	type fixture struct {
		sql     sqlmock.Sqlmock
		model   *MockModelService
		modelID string
		column  *MockColumnService
		update  func(uidt string, meta map[string]interface{}, newUIDT *string) error
	}
	setup := func(t *testing.T, uidt, dt string) fixture {
		db, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })
		mockTable, mockModel, mockColumn := &MockTableService{}, &MockModelService{}, &MockColumnService{}
		repo := &pkg.DatabaseService{TableService: mockTable, BulkService: &MockBulkService{}, DB: db}
		svc := services.NewTableManagementService("postgres", repo, mockModel, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})
		modelID, colID := uuid.New(), uuid.New()
		stored := tenant.Column{
			ID: colID, ModelID: modelID.String(), BaseID: uuid.NewString(), ColumnName: "col", UIDT: uidt, DT: helpers.StringPtr(dt),
			Meta: map[string]interface{}{"options": options("1", "2", "3"), "defaultValue": "2"},
		}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", colID.String()).Return(stored, nil)
		mockColumn.On("UpdateColumn", mock.Anything, "schema", colID.String(), mock.Anything).Return(stored, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
		mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
		mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
		mockTable.On("GetByFunction", mock.Anything, "public.link_key_columns", mock.Anything).
			Return([]map[string]interface{}{{"link_key_columns": "{id}"}}, nil).Maybe()
		return fixture{sql: sqlMock, model: mockModel, modelID: modelID.String(), column: mockColumn, update: func(_ string, meta map[string]interface{}, newUIDT *string) error {
			_, err := svc.UpdateColumn(context.Background(), "schema", colID.String(), dto.ColumnUpdate{Meta: &meta, UIDT: newUIDT})
			return err
		}}
	}
	savedMeta := func(t *testing.T, f fixture) map[string]interface{} {
		t.Helper()
		for _, c := range f.column.Calls {
			if c.Method == "UpdateColumn" {
				return *c.Arguments.Get(3).(dto.ColumnUpdate).Meta
			}
		}
		t.Fatal("UpdateColumn not called")
		return nil
	}

	t.Run("select: removes the option from the default and clears the cells", func(t *testing.T) {
		f := setup(t, "select", "TEXT")
		f.sql.ExpectExec(regexp.QuoteMeta(`UPDATE "schema"."tbl" SET "col" = NULL WHERE "col" = ANY($1::text[])`)).
			WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 2))

		err := f.update("select", map[string]interface{}{"options": options("1", "3"), "defaultValue": "2"}, nil)

		require.NoError(t, err)
		assert.NoError(t, f.sql.ExpectationsWereMet())
		assert.NotContains(t, savedMeta(t, f), "defaultValue")
	})

	t.Run("select: keeps a default that is still an option", func(t *testing.T) {
		f := setup(t, "select", "TEXT")
		f.sql.ExpectExec(`UPDATE "schema"."tbl" SET "col" = NULL`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))

		err := f.update("select", map[string]interface{}{"options": options("1", "2"), "defaultValue": "2"}, nil)

		require.NoError(t, err)
		assert.Equal(t, "2", savedMeta(t, f)["defaultValue"])
		assert.NoError(t, f.sql.ExpectationsWereMet())
	})

	t.Run("multiSelect: removes the option from each array", func(t *testing.T) {
		f := setup(t, "multiSelect", "TEXT[]")
		f.sql.ExpectExec(`UPDATE "schema"."tbl" SET "col" = NULLIF\(ARRAY\(SELECT x FROM unnest\("col"\) AS x WHERE x <> ALL\(\$1::text\[\]\)\), '\{\}'\) WHERE "col" && \$1::text\[\]`).
			WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 2))

		err := f.update("multiSelect", map[string]interface{}{"options": options("1", "3"), "defaultValue": []interface{}{"2", "3"}}, nil)

		require.NoError(t, err)
		assert.Equal(t, []interface{}{"3"}, savedMeta(t, f)["defaultValue"])
		assert.NoError(t, f.sql.ExpectationsWereMet())
	})

	t.Run("reports a database error from clearing the cells", func(t *testing.T) {
		f := setup(t, "select", "TEXT")
		f.sql.ExpectExec(`UPDATE "schema"."tbl" SET "col" = NULL`).WithArgs(sqlmock.AnyArg()).WillReturnError(errors.New("boom"))

		err := f.update("select", map[string]interface{}{"options": options("1", "3")}, nil)

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("reports a failed model lookup while clearing the cells", func(t *testing.T) {
		f := setup(t, "select", "TEXT")
		// The default sync reads the model first; the lookup for clearing the cells fails
		f.model.ExpectedCalls = nil
		f.model.On("GetModelByID", mock.Anything, "schema", f.modelID).Return(tenant.Model{Alias: "tbl"}, nil).Once()
		f.model.On("GetModelByID", mock.Anything, "schema", f.modelID).Return(tenant.Model{}, errors.New("model gone")).Once()

		err := f.update("select", map[string]interface{}{"options": options("1", "3")}, nil)

		assert.EqualError(t, err, "model gone")
		assert.NoError(t, f.sql.ExpectationsWereMet())
	})

	t.Run("touches no cells when no option is removed", func(t *testing.T) {
		f := setup(t, "select", "TEXT")

		err := f.update("select", map[string]interface{}{"options": options("1", "2", "3", "4"), "defaultValue": "2"}, nil)

		require.NoError(t, err)
		assert.NoError(t, f.sql.ExpectationsWereMet())
	})

	t.Run("leaves the cells alone when the field type changes", func(t *testing.T) {
		f := setup(t, "select", "TEXT")
		newType := "text"

		err := f.update("select", map[string]interface{}{"options": options("1")}, &newType)

		require.NoError(t, err)
		assert.NoError(t, f.sql.ExpectationsWereMet())
	})

	t.Run("ignores other field types", func(t *testing.T) {
		f := setup(t, "text", "TEXT")

		err := f.update("text", map[string]interface{}{"options": options("1")}, nil)

		require.NoError(t, err)
		assert.NoError(t, f.sql.ExpectationsWereMet())
	})
}
