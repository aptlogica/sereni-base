package table_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errDBDown = errors.New("db down")

// autoV7Meta is fresh column meta of an auto v7 uuid field; a fresh map, since the service may store into it
func autoV7Meta() map[string]interface{} {
	return map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v7"}}
}

// sqlFixture is a table service whose repo also has a SQL mock, for the queries the uuid code sends itself
type sqlFixture struct {
	table  *MockTableService
	model  *MockModelService
	column *MockColumnService
	sql    sqlmock.Sqlmock
	svc    interfaces.TableManagementService
}

func newSQLFixture(t *testing.T, keyDefaults bool) *sqlFixture {
	t.Helper()
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	f := &sqlFixture{table: &MockTableService{}, model: &MockModelService{}, column: &MockColumnService{}, sql: sqlMock}
	if keyDefaults {
		expectKeyDefaults(f.table)
	}
	repo := &pkg.DatabaseService{TableService: f.table, BulkService: &MockBulkService{}, DB: db}
	f.svc = services.NewTableManagementService("postgres", repo, f.model, f.column, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})
	return f
}

// emptyCells matches the read of records whose value in the column is empty
func emptyCells(column string) interface{} {
	return mock.MatchedBy(func(p dbModels.QueryParams) bool {
		return len(p.Filters) == 1 && p.Filters[0].Column == column && p.Filters[0].Operator == "is_null"
	})
}

func idRows(n int) []map[string]interface{} {
	rows := make([]map[string]interface{}, n)
	for i := range rows {
		rows[i] = map[string]interface{}{"id": int64(i + 1)}
	}
	return rows
}

func TestIDGenerationFromMeta(t *testing.T) {
	cfg, ok := dto.IDGenerationFromMeta(map[string]interface{}{"idGeneration": map[string]interface{}{}})
	assert.True(t, ok)
	assert.Equal(t, "manual", cfg.Mode, "an empty mode is manual")

	cfg, ok = dto.IDGenerationFromMeta(nil)
	assert.True(t, ok)
	assert.Equal(t, dto.IDGeneration{Mode: "manual"}, cfg)

	_, ok = dto.IDGenerationFromMeta(map[string]interface{}{"idGeneration": map[string]interface{}{"mode": 1}})
	assert.False(t, ok, "a mode that isn't text")

	_, ok = dto.IDGenerationFromMeta(map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": 7}})
	assert.False(t, ok, "a version that isn't text")
}

func TestAddColumn_UUIDField(t *testing.T) {
	modelID, baseID := uuid.New(), uuid.New()
	newCol := func(meta map[string]interface{}) tenant.Column {
		return tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID.String(), ColumnName: "order_ref", Title: "Order Ref", UIDT: "uuid", DT: helpers.StringPtr("UUID"), Meta: meta}
	}
	request := func(meta map[string]interface{}) dto.AddColumnRequest {
		return dto.AddColumnRequest{ModelID: modelID, BaseID: baseID, Title: "Order Ref", UIDT: "uuid", Meta: meta}
	}
	setup := func(t *testing.T, col tenant.Column) *sqlFixture {
		f := newSQLFixture(t, true)
		f.column.On("Create", mock.Anything, mock.Anything, "schema").Return(col, nil)
		f.model.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "orders"}, nil)
		f.table.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
		return f
	}

	t.Run("rejects invalid settings before creating anything", func(t *testing.T) {
		f := newSQLFixture(t, true)
		_, err := f.svc.AddColumn(context.Background(), "schema", request(map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v5"}}))
		assert.ErrorIs(t, err, app_errors.UUIDConfigInvalid)
		f.column.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a new auto field fills every existing record and stores its settings", func(t *testing.T) {
		col := newCol(autoV7Meta())
		f := setup(t, col)
		f.table.On("GetTableData", `"schema"."orders"`, emptyCells("order_ref")).Return(idRows(2), nil)
		f.sql.ExpectExec(`UPDATE "schema"."orders" AS t SET "order_ref" = v.value::uuid`).WillReturnResult(sqlmock.NewResult(0, 2))

		_, err := f.svc.AddColumn(context.Background(), "schema", request(map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto"}}))

		require.NoError(t, err)
		assert.NoError(t, f.sql.ExpectationsWereMet())
		created := f.column.Calls[0].Arguments.Get(1).(dto.ColumnInsertion)
		assert.Equal(t, dto.IDGeneration{Mode: "auto", Version: "v7"}, created.Meta["idGeneration"])
	})

	t.Run("a new manual field fills nothing", func(t *testing.T) {
		f := setup(t, newCol(nil))

		_, err := f.svc.AddColumn(context.Background(), "schema", request(nil))

		require.NoError(t, err)
		f.table.AssertNotCalled(t, "GetTableData", `"schema"."orders"`, mock.Anything)
		created := f.column.Calls[0].Arguments.Get(1).(dto.ColumnInsertion)
		assert.Equal(t, dto.IDGeneration{Mode: "manual"}, created.Meta["idGeneration"])
	})

	t.Run("removes the field when the records can't be filled", func(t *testing.T) {
		col := newCol(autoV7Meta())
		f := setup(t, col)
		f.table.On("GetTableData", `"schema"."orders"`, emptyCells("order_ref")).Return([]map[string]interface{}(nil), errDBDown)
		f.column.On("DeleteColumn", mock.Anything, "schema", col.ID.String()).Return(nil)
		f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil)

		_, err := f.svc.AddColumn(context.Background(), "schema", request(autoV7Meta()))

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		f.column.AssertCalled(t, "DeleteColumn", mock.Anything, "schema", col.ID.String())
	})
}

func TestUpdateColumn_AutoStartFailures(t *testing.T) {
	modelID := uuid.NewString()
	manual := map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "manual"}}
	model := tenant.Model{ID: uuid.MustParse(modelID), Alias: "orders"}
	autoMeta := func() *map[string]interface{} {
		return &map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v4"}}
	}
	setup := func(t *testing.T, keyDefaults bool) *sqlFixture {
		f := newSQLFixture(t, keyDefaults)
		col := uuidCol(modelID, "external_id", manual)
		f.column.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
		f.column.On("UpdateColumn", mock.Anything, "schema", "cid", mock.Anything).Return(col, nil)
		f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
		return f
	}

	t.Run("the model can't be read", func(t *testing.T) {
		f := setup(t, true)
		f.model.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil).Once()
		f.model.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{}, errDBDown)

		_, err := f.svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: autoMeta(), Backfill: helpers.BoolPtr(true)})
		assert.ErrorIs(t, err, errDBDown)
	})

	t.Run("the primary key can't be read", func(t *testing.T) {
		f := setup(t, false)
		f.model.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
		f.table.On("GetByFunction", mock.Anything, "public.link_key_columns", mock.Anything).Return([]map[string]interface{}(nil), errDBDown)

		_, err := f.svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: autoMeta(), Backfill: helpers.BoolPtr(true)})
		assert.Error(t, err)
	})

	t.Run("a failed fill puts the old settings back", func(t *testing.T) {
		f := setup(t, true)
		f.model.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
		f.table.On("GetTableData", `"schema"."orders"`, emptyCells("external_id")).Return([]map[string]interface{}(nil), errDBDown)

		_, err := f.svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: autoMeta(), Backfill: helpers.BoolPtr(true)})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		last := f.column.Calls[len(f.column.Calls)-1]
		require.Equal(t, "UpdateColumn", last.Method)
		assert.Equal(t, manual, *last.Arguments.Get(3).(dto.ColumnUpdate).Meta)
	})

	t.Run("changing a field away from uuid drops its settings", func(t *testing.T) {
		f := setup(t, true)
		f.model.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
		f.table.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil).Maybe()
		f.table.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{{}}, nil).Maybe()
		text := "text"
		meta := autoMeta()

		_, _ = f.svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{UIDT: &text, Meta: meta})

		assert.NotContains(t, *meta, "idGeneration")
	})
}

func TestBackfillUUIDs_Batches(t *testing.T) {
	modelID := uuid.NewString()
	manual := map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "manual"}}
	setup := func(t *testing.T) *sqlFixture {
		f := newSQLFixture(t, true)
		col := uuidCol(modelID, "external_id", manual)
		f.column.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
		f.column.On("UpdateColumn", mock.Anything, "schema", "cid", mock.Anything).Return(col, nil)
		f.model.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{ID: uuid.MustParse(modelID), Alias: "orders"}, nil)
		f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
		return f
	}
	update := func(f *sqlFixture) error {
		meta := &map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v1"}}
		_, err := f.svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: meta, Backfill: helpers.BoolPtr(true)})
		return err
	}

	t.Run("keeps reading until a short batch", func(t *testing.T) {
		f := setup(t)
		f.table.On("GetTableData", `"schema"."orders"`, emptyCells("external_id")).Return(idRows(1000), nil).Once()
		f.table.On("GetTableData", `"schema"."orders"`, emptyCells("external_id")).Return([]map[string]interface{}{}, nil).Once()
		f.sql.ExpectExec(`UPDATE "schema"."orders"`).WillReturnResult(sqlmock.NewResult(0, 1000))

		require.NoError(t, update(f))
		assert.NoError(t, f.sql.ExpectationsWereMet())
	})

	t.Run("a record without an id", func(t *testing.T) {
		f := setup(t)
		f.table.On("GetTableData", `"schema"."orders"`, emptyCells("external_id")).Return([]map[string]interface{}{{"title": "x"}}, nil)

		assert.Error(t, update(f))
	})

	t.Run("the update fails", func(t *testing.T) {
		f := setup(t)
		f.table.On("GetTableData", `"schema"."orders"`, emptyCells("external_id")).Return(idRows(3), nil)
		f.sql.ExpectExec(`UPDATE "schema"."orders"`).WillReturnError(errDBDown)

		assert.ErrorIs(t, update(f), app_errors.DatabaseError)
	})
}

func TestInserts_UUIDLookupFailures(t *testing.T) {
	modelID := uuid.NewString()
	model := tenant.Model{ID: uuid.MustParse(modelID), Alias: "orders"}

	t.Run("a new empty record when the columns can't be read", func(t *testing.T) {
		_, _, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column(nil), errDBDown)

		_, err := svc.CreateRow(context.Background(), "schema", dto.CreateRowRequest{ModelID: modelID, CreatedBy: "u"})
		assert.ErrorIs(t, err, errDBDown)
	})

	t.Run("a new record with values when the columns can't be read", func(t *testing.T) {
		_, _, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column(nil), errDBDown)

		_, err := upsert(svc)(context.Background(), "schema", modelID, 0, map[string]interface{}{}, "u")
		assert.ErrorIs(t, err, errDBDown)
	})

	t.Run("an auto field without a version gets v7", func(t *testing.T) {
		_, _, _, mockModel, mockColumn, _, _, _, _ := setupTableManagementService()
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
		noVersion := uuidCol(modelID, "order_ref", map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto"}})
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{noVersion}, nil)
		var written map[string]interface{}
		stubTable := &StubTableService{CreateRecordFn: func(_ string, data map[string]interface{}) (map[string]interface{}, error) {
			written = data
			return map[string]interface{}{"id": 1}, nil
		}}
		svc := setupTableManagementServiceWithStubs(stubTable, &StubBulkService{}, mockModel, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})

		_, err := svc.CreateRow(context.Background(), "schema", dto.CreateRowRequest{ModelID: modelID, CreatedBy: "u"})
		require.NoError(t, err)
		assert.Equal(t, uuid.Version(7), uuid.MustParse(written[`"order_ref"`].(string)).Version())
	})

	// byAlias builds a service whose models table lists "orders" with the given row
	byAlias := func(modelRow map[string]interface{}, columns []tenant.Column, columnsErr error) (*MockBulkService, interfaces.TableManagementService) {
		mockTable := &MockTableService{}
		mockTable.On("GetTableData", `"schema".models`, mock.Anything).Return([]map[string]interface{}{modelRow}, nil)
		mockColumn := &MockColumnService{}
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return(columns, columnsErr)
		mockBulk := &MockBulkService{}
		repo := &pkg.DatabaseService{TableService: mockTable, BulkService: mockBulk}
		return mockBulk, services.NewTableManagementService("postgres", repo, &MockModelService{}, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})
	}
	modelRow := map[string]interface{}{"id": modelID, "alias": "orders"}
	auto := []tenant.Column{uuidCol(modelID, "order_ref", autoV7Meta())}

	t.Run("a record by alias that gives a value for an auto field", func(t *testing.T) {
		_, svc := byAlias(modelRow, auto, nil)
		_, err := svc.CreateRowWithRecords(context.Background(), "schema", "orders", map[string]interface{}{"order_ref": "3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63"})
		assert.ErrorIs(t, err, app_errors.UUIDReadOnly)
	})

	t.Run("a record by alias when the columns can't be read", func(t *testing.T) {
		_, svc := byAlias(modelRow, nil, errDBDown)
		_, err := svc.CreateRowWithRecords(context.Background(), "schema", "orders", map[string]interface{}{})
		assert.ErrorIs(t, err, errDBDown)
	})

	t.Run("bulk records that give a value for an auto field", func(t *testing.T) {
		mockBulk, svc := byAlias(modelRow, auto, nil)
		_, err := svc.CreateRowsWithRecordsBulk(context.Background(), "schema", "orders", []map[string]interface{}{{"order_ref": "x"}})
		assert.ErrorIs(t, err, app_errors.UUIDReadOnly)
		mockBulk.AssertNotCalled(t, "BulkInsert", mock.Anything, mock.Anything)
	})

	t.Run("bulk records when the columns can't be read", func(t *testing.T) {
		_, svc := byAlias(modelRow, nil, errDBDown)
		_, err := svc.CreateRowsWithRecordsBulk(context.Background(), "schema", "orders", []map[string]interface{}{{}})
		assert.ErrorIs(t, err, errDBDown)
	})

	t.Run("a model row that can't be read", func(t *testing.T) {
		_, svc := byAlias(map[string]interface{}{"id": []int{1}, "alias": "orders"}, nil, nil)
		_, err := svc.CreateRowWithRecords(context.Background(), "schema", "orders", map[string]interface{}{})
		assert.Error(t, err)
	})
}

func TestFillGeneratedUUIDs_UnknownVersion(t *testing.T) {
	err := services.FillGeneratedUUIDs([]services.AutoUUIDColumn{{ColumnName: "x", Version: "v9"}}, map[string]interface{}{})
	assert.ErrorIs(t, err, app_errors.UUIDConfigInvalid)
}

func TestCountEmptyCells_Failures(t *testing.T) {
	modelID := uuid.NewString()
	count := func(svc interfaces.TableManagementService) error {
		_, err := svc.(interface {
			CountEmptyCells(ctx context.Context, schemaName string, columnID string) (dto.EmptyCountResponse, error)
		}).CountEmptyCells(context.Background(), "schema", "cid")
		return err
	}
	setup := func(t *testing.T) *sqlFixture {
		f := newSQLFixture(t, false)
		f.column.On("GetColumnByID", mock.Anything, "schema", "cid").Return(uuidCol(modelID, "external_id", nil), nil)
		f.model.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{ID: uuid.MustParse(modelID), Alias: "orders"}, nil)
		return f
	}

	t.Run("the column can't be read", func(t *testing.T) {
		f := newSQLFixture(t, false)
		f.column.On("GetColumnByID", mock.Anything, "schema", "cid").Return(tenant.Column{}, errDBDown)
		assert.Error(t, count(f.svc))
	})

	t.Run("the model can't be read", func(t *testing.T) {
		f := newSQLFixture(t, false)
		f.column.On("GetColumnByID", mock.Anything, "schema", "cid").Return(uuidCol(modelID, "external_id", nil), nil)
		f.model.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{}, errDBDown)
		assert.ErrorIs(t, count(f.svc), errDBDown)
	})

	t.Run("the query fails", func(t *testing.T) {
		f := setup(t)
		f.sql.ExpectQuery(`SELECT COUNT`).WillReturnError(errDBDown)
		assert.ErrorIs(t, count(f.svc), app_errors.DatabaseError)
	})

	t.Run("the counts can't be read", func(t *testing.T) {
		f := setup(t)
		f.sql.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"empty", "filled"}).AddRow("many", "few"))
		assert.ErrorIs(t, count(f.svc), app_errors.DatabaseError)
	})
}

func TestIDGenerationFromMeta_TypedSettings(t *testing.T) {
	cfg, ok := dto.IDGenerationFromMeta(map[string]interface{}{"idGeneration": dto.IDGeneration{Mode: "auto", Version: "v4"}})
	assert.True(t, ok)
	assert.Equal(t, dto.IDGeneration{Mode: "auto", Version: "v4"}, cfg)

	cfg, _ = dto.IDGenerationFromMeta(map[string]interface{}{"idGeneration": dto.IDGeneration{}})
	assert.Equal(t, "manual", cfg.Mode)
}
