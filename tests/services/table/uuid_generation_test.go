package table_test

import (
	"context"
	"net"
	"sort"
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

func TestNewUUID_Versions(t *testing.T) {
	for version, want := range map[string]uuid.Version{"v7": 7, "v4": 4, "v1": 1} {
		t.Run(version, func(t *testing.T) {
			u, err := services.NewUUID(version)
			require.NoError(t, err)
			assert.Equal(t, want, u.Version())
			assert.Equal(t, uuid.RFC4122, u.Variant())
		})
	}

	t.Run("unknown version", func(t *testing.T) {
		for _, version := range []string{"", "v5", "v8", "7"} {
			_, err := services.NewUUID(version)
			assert.ErrorIs(t, err, app_errors.UUIDConfigInvalid, version)
		}
	})

	t.Run("v7 values are unique and in creation order", func(t *testing.T) {
		values := make([]string, 1000)
		for i := range values {
			u, err := services.NewUUID("v7")
			require.NoError(t, err)
			values[i] = u.String()
		}
		assert.True(t, sort.StringsAreSorted(values))
		assert.Len(t, uniqueStrings(values), len(values))
	})

	t.Run("v4 values are unique", func(t *testing.T) {
		values := make([]string, 10000)
		for i := range values {
			u, err := services.NewUUID("v4")
			require.NoError(t, err)
			values[i] = u.String()
		}
		assert.Len(t, uniqueStrings(values), len(values))
	})

	t.Run("v1 carries a random node, not a network address", func(t *testing.T) {
		u, err := services.NewUUID("v1")
		require.NoError(t, err)
		node := u.NodeID()
		assert.Equal(t, byte(1), node[0]&0x01, "multicast bit marks a random node")
		interfaces, _ := net.Interfaces()
		for _, iface := range interfaces {
			assert.NotEqual(t, []byte(iface.HardwareAddr), node)
		}
	})
}

func uniqueStrings(values []string) map[string]struct{} {
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		seen[v] = struct{}{}
	}
	return seen
}

func TestUUIDCellValue(t *testing.T) {
	accepted := map[string]interface{}{
		"v4":            "3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63",
		"v7":            "019a0c4e-8f21-7b3d-a9c4-5e2f1d8b6a70",
		"v1":            "c232ab00-9414-11ec-b3c8-9f6bdeced846",
		"v8":            "320c3d4d-cc00-875b-8ec9-32d5f69181c0",
		"braces, upper": "{3F6C2A9E-1B47-4D8A-9E05-7C2B4F1A8D63}",
		"urn":           "urn:uuid:3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63",
		"spaces":        "  3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63 ",
	}
	for name, in := range accepted {
		t.Run("accepts "+name, func(t *testing.T) {
			got, err := services.UUIDCellValue(in)
			require.NoError(t, err)
			parsed := uuid.MustParse(got.(string))
			assert.Equal(t, parsed.String(), got, "stored in canonical lowercase form")
		})
	}

	t.Run("normalises braces and case", func(t *testing.T) {
		got, err := services.UUIDCellValue("{3F6C2A9E-1B47-4D8A-9E05-7C2B4F1A8D63}")
		require.NoError(t, err)
		assert.Equal(t, "3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63", got)
	})

	for _, empty := range []interface{}{nil, "", "   "} {
		got, err := services.UUIDCellValue(empty)
		assert.NoError(t, err)
		assert.Nil(t, got)
	}

	rejected := []interface{}{
		"ORD-2291-A",
		"3f6c2a9e-1b47-4d8a-9e05",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
		42,
	}
	for _, in := range rejected {
		_, err := services.UUIDCellValue(in)
		assert.ErrorIs(t, err, app_errors.UUIDInvalid, in)
	}
}

func TestValidateIDGeneration(t *testing.T) {
	gen := func(v map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{"idGeneration": v}
	}

	cases := []struct {
		name    string
		uidt    string
		meta    map[string]interface{}
		want    dto.IDGeneration
		wantErr bool
	}{
		{"no settings is manual", "uuid", map[string]interface{}{}, dto.IDGeneration{Mode: "manual"}, false},
		{"auto without a version gets v7", "uuid", gen(map[string]interface{}{"mode": "auto"}), dto.IDGeneration{Mode: "auto", Version: "v7"}, false},
		{"auto v4", "uuid", gen(map[string]interface{}{"mode": "auto", "version": "v4"}), dto.IDGeneration{Mode: "auto", Version: "v4"}, false},
		{"auto v1", "uuid", gen(map[string]interface{}{"mode": "auto", "version": "v1"}), dto.IDGeneration{Mode: "auto", Version: "v1"}, false},
		{"manual drops a version", "uuid", gen(map[string]interface{}{"mode": "manual", "version": "v4"}), dto.IDGeneration{Mode: "manual"}, false},
		{"a non-uuid field without settings", "text", map[string]interface{}{}, dto.IDGeneration{Mode: "manual"}, false},
		{"version not offered", "uuid", gen(map[string]interface{}{"mode": "auto", "version": "v5"}), dto.IDGeneration{}, true},
		{"unknown mode", "uuid", gen(map[string]interface{}{"mode": "sometimes"}), dto.IDGeneration{}, true},
		{"settings that aren't an object", "uuid", map[string]interface{}{"idGeneration": "auto"}, dto.IDGeneration{}, true},
		{"auto with a default value", "uuid", map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto"}, "defaultValue": "x"}, dto.IDGeneration{}, true},
		{"settings on a non-uuid field", "text", gen(map[string]interface{}{"mode": "auto"}), dto.IDGeneration{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := services.ValidateIDGeneration(tc.uidt, tc.meta)
			if tc.wantErr {
				assert.ErrorIs(t, err, app_errors.UUIDConfigInvalid)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFillGeneratedUUIDs(t *testing.T) {
	cols := []services.AutoUUIDColumn{{ColumnName: "order_ref", Version: "v7"}, {ColumnName: "public_id", Version: "v4"}}

	t.Run("fills every auto field", func(t *testing.T) {
		record := map[string]interface{}{"title": "Desk"}
		require.NoError(t, services.FillGeneratedUUIDs(cols, record))
		assert.Equal(t, uuid.Version(7), uuid.MustParse(record["order_ref"].(string)).Version())
		assert.Equal(t, uuid.Version(4), uuid.MustParse(record["public_id"].(string)).Version())
		assert.Equal(t, "Desk", record["title"])
	})

	t.Run("refuses a value given for an auto field", func(t *testing.T) {
		record := map[string]interface{}{"order_ref": "3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63"}
		assert.ErrorIs(t, services.FillGeneratedUUIDs(cols, record), app_errors.UUIDReadOnly)
	})

	t.Run("an empty value counts as not given", func(t *testing.T) {
		record := map[string]interface{}{"order_ref": nil, "public_id": ""}
		require.NoError(t, services.FillGeneratedUUIDs(cols, record))
		assert.NotEmpty(t, record["order_ref"])
		assert.NotEmpty(t, record["public_id"])
	})
}

func uuidCol(modelID, name string, meta map[string]interface{}) tenant.Column {
	return tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.NewString(), ColumnName: name, Title: name, UIDT: "uuid", DT: helpers.StringPtr("UUID"), Meta: meta}
}

func TestRowWrites_UUIDFields(t *testing.T) {
	autoMeta := map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v7"}}
	manualMeta := map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "manual"}}

	// fixture: an "orders" table with an auto (v7) and a manual uuid field; written holds the last record saved
	type fixture struct {
		svc          interfaces.TableManagementService
		modelID      string
		auto, manual tenant.Column
		written      map[string]interface{}
	}
	setup := func() *fixture {
		f := &fixture{modelID: uuid.NewString()}
		f.auto = uuidCol(f.modelID, "order_ref", autoMeta)
		f.manual = uuidCol(f.modelID, "external_id", manualMeta)

		mockModel := &MockModelService{}
		mockModel.On("GetModelByID", mock.Anything, "schema", f.modelID).Return(tenant.Model{ID: uuid.MustParse(f.modelID), Alias: "orders"}, nil)
		mockColumn := &MockColumnService{}
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", f.modelID).Return([]tenant.Column{f.auto, f.manual}, nil)
		mockColumn.On("GetColumnByID", mock.Anything, "schema", f.auto.ID.String()).Return(f.auto, nil)
		mockColumn.On("GetColumnByID", mock.Anything, "schema", f.manual.ID.String()).Return(f.manual, nil)

		stubTable := &StubTableService{}
		stubTable.CreateRecordFn = func(_ string, data map[string]interface{}) (map[string]interface{}, error) {
			f.written = data
			return map[string]interface{}{"id": 1}, nil
		}
		stubTable.UpdateRecordFn = func(_ string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
			f.written = data
			return map[string]interface{}{"id": id}, nil
		}
		f.svc = setupTableManagementServiceWithStubs(stubTable, &StubBulkService{}, mockModel, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})
		return f
	}

	t.Run("a new empty record gets a value in the auto field", func(t *testing.T) {
		f := setup()

		_, err := f.svc.CreateRow(context.Background(), "schema", dto.CreateRowRequest{ModelID: f.modelID, CreatedBy: "u"})
		require.NoError(t, err)
		value, ok := f.written[`"order_ref"`].(string)
		require.True(t, ok, "auto field is written")
		assert.Equal(t, uuid.Version(7), uuid.MustParse(value).Version())
		assert.NotContains(t, f.written, `"external_id"`, "manual field is left empty")
	})

	t.Run("a new record with values gets a value in the auto field and keeps the manual one", func(t *testing.T) {
		f := setup()

		_, err := upsert(f.svc)(context.Background(), "schema", f.modelID, 0,
			map[string]interface{}{f.manual.ID.String(): "{3F6C2A9E-1B47-4D8A-9E05-7C2B4F1A8D63}"}, "u")
		require.NoError(t, err)
		assert.Equal(t, "3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63", f.written[`"external_id"`])
		assert.NotEmpty(t, f.written[`"order_ref"`])
	})

	t.Run("the auto field can't be written", func(t *testing.T) {
		f := setup()
		value := interface{}("3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63")

		_, err := f.svc.InsertRowData(context.Background(), "schema", dto.InsertRowDataRequest{ModelID: f.modelID, ColumnId: f.auto.ID.String(), RowId: 1, Value: &value})
		assert.ErrorIs(t, err, app_errors.UUIDReadOnly)

		_, err = upsert(f.svc)(context.Background(), "schema", f.modelID, 0, map[string]interface{}{f.auto.ID.String(): value}, "u")
		assert.ErrorIs(t, err, app_errors.UUIDReadOnly)
	})

	t.Run("the manual field only takes a valid UUID", func(t *testing.T) {
		f := setup()
		value := interface{}("ORD-2291-A")

		_, err := f.svc.InsertRowData(context.Background(), "schema", dto.InsertRowDataRequest{ModelID: f.modelID, ColumnId: f.manual.ID.String(), RowId: 1, Value: &value})
		assert.ErrorIs(t, err, app_errors.UUIDInvalid)
	})

	t.Run("records created by table alias (import) get a value in the auto field", func(t *testing.T) {
		modelID := uuid.NewString()
		auto := uuidCol(modelID, "order_ref", autoMeta)
		mockTable := &MockTableService{}
		mockTable.On("GetTableData", `"schema".models`, mock.Anything).Return([]map[string]interface{}{{"id": modelID, "alias": "orders"}}, nil)
		mockColumn := &MockColumnService{}
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{auto}, nil)
		var inserted []map[string]interface{}
		mockBulk := &MockBulkService{}
		mockBulk.On("BulkInsert", `"schema"."orders"`, mock.Anything).Run(func(args mock.Arguments) {
			inserted = args.Get(1).([]map[string]interface{})
		}).Return([]map[string]interface{}{{"id": 1}, {"id": 2}}, nil)
		repo := &pkg.DatabaseService{TableService: mockTable, BulkService: mockBulk}
		svc := services.NewTableManagementService("postgres", repo, &MockModelService{}, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})

		_, err := svc.CreateRowsWithRecordsBulk(context.Background(), "schema", "orders", []map[string]interface{}{{"title": "a"}, {"title": "b"}})
		require.NoError(t, err)
		require.Len(t, inserted, 2)
		assert.NotEqual(t, inserted[0]["order_ref"], inserted[1]["order_ref"])
		assert.Equal(t, uuid.Version(7), uuid.MustParse(inserted[0]["order_ref"].(string)).Version())
	})
}

// upsert reaches UpsertRowValues, which the service has but its interface doesn't list
func upsert(svc interfaces.TableManagementService) func(ctx context.Context, schemaName, modelID string, rowID int, values map[string]interface{}, user string) (dto.RecordResponse, error) {
	return svc.(interface {
		UpsertRowValues(ctx context.Context, schemaName string, modelID string, rowID int, values map[string]interface{}, user string) (dto.RecordResponse, error)
	}).UpsertRowValues
}

func TestUpdateColumn_ManualToAutoFillsEmptyRecords(t *testing.T) {
	autoMeta := func() *map[string]interface{} {
		return &map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v4"}}
	}
	setup := func(t *testing.T) (*MockTableService, *MockColumnService, sqlmock.Sqlmock, interfaces.TableManagementService, tenant.Column) {
		db, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })

		modelID := uuid.NewString()
		col := uuidCol(modelID, "external_id", map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "manual"}})
		mockTable := &MockTableService{}
		expectKeyDefaults(mockTable)
		mockTable.On("AlterTable", `"schema"."orders"`, mock.Anything).Return(nil)
		mockModel := &MockModelService{}
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{ID: uuid.MustParse(modelID), Alias: "orders"}, nil)
		mockColumn := &MockColumnService{}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
		mockColumn.On("UpdateColumn", mock.Anything, "schema", "cid", mock.Anything).Return(col, nil)

		repo := &pkg.DatabaseService{TableService: mockTable, BulkService: &MockBulkService{}, DB: db}
		svc := services.NewTableManagementService("postgres", repo, mockModel, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})
		return mockTable, mockColumn, sqlMock, svc, col
	}
	emptyRows := mock.MatchedBy(func(p dbModels.QueryParams) bool {
		return len(p.Filters) == 1 && p.Filters[0].Column == "external_id" && p.Filters[0].Operator == "is_null"
	})

	t.Run("fills the empty records when asked", func(t *testing.T) {
		mockTable, mockColumn, sqlMock, svc, _ := setup(t)
		mockTable.On("GetTableData", `"schema"."orders"`, emptyRows).Return([]map[string]interface{}{{"id": int64(3)}, {"id": int64(7)}}, nil).Once()
		sqlMock.ExpectExec(`UPDATE "schema"."orders" AS t SET "external_id" = v.value::uuid`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 2))

		_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: autoMeta(), Backfill: helpers.BoolPtr(true)})

		require.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
		// The stored settings name the version
		req := mockColumn.Calls[len(mockColumn.Calls)-1].Arguments.Get(3).(dto.ColumnUpdate)
		assert.Equal(t, dto.IDGeneration{Mode: "auto", Version: "v4"}, (*req.Meta)["idGeneration"])
	})

	t.Run("leaves empty records when not asked", func(t *testing.T) {
		mockTable, _, sqlMock, svc, _ := setup(t)

		_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: autoMeta()})

		require.NoError(t, err)
		mockTable.AssertNotCalled(t, "GetTableData", `"schema"."orders"`, mock.Anything)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("always fills a primary key field", func(t *testing.T) {
		mockTable, _, sqlMock, svc, _ := setup(t)
		mockTable.ExpectedCalls = nil
		mockTable.On("GetByFunction", mock.Anything, "public.link_key_columns", mock.Anything).
			Return([]map[string]interface{}{{"link_key_columns": "{external_id}"}}, nil)
		mockTable.On("GetTableData", mock.MatchedBy(func(s string) bool { return s != `"schema"."orders"` }), mock.Anything).Return([]map[string]interface{}{}, nil).Maybe()
		mockTable.On("AlterTable", `"schema"."orders"`, mock.Anything).Return(nil)
		mockTable.On("GetTableData", `"schema"."orders"`, emptyRows).Return([]map[string]interface{}{{"id": int64(1)}}, nil).Once()
		sqlMock.ExpectExec(`UPDATE "schema"."orders"`).WillReturnResult(sqlmock.NewResult(0, 1))

		_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: autoMeta(), Backfill: helpers.BoolPtr(false)})

		require.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("rejects a version that isn't offered", func(t *testing.T) {
		_, mockColumn, _, svc, _ := setup(t)
		meta := &map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v5"}}

		_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: meta})

		assert.ErrorIs(t, err, app_errors.UUIDConfigInvalid)
		mockColumn.AssertNotCalled(t, "UpdateColumn", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestDataTools_AutoUUIDFields(t *testing.T) {
	autoMeta := map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v7"}}
	manualMeta := map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "manual"}}
	modelID := uuid.NewString()
	auto := uuidCol(modelID, "order_ref", autoMeta)
	manual := uuidCol(modelID, "external_id", manualMeta)

	t.Run("data tools refuse an auto field and allow a manual one", func(t *testing.T) {
		_, _, _, _, mockColumn, _, _, _, svc := setupTableManagementService()
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{auto, manual}, nil)

		assert.ErrorIs(t, svc.ValidateColumnsAllowed(context.Background(), "schema", modelID, []string{auto.ID.String()}), app_errors.UUIDReadOnly)
		assert.NoError(t, svc.ValidateColumnsAllowed(context.Background(), "schema", modelID, []string{manual.ID.String()}))
	})

	setupWrites := func() (*MockColumnService, interfaces.TableManagementService) {
		_, _, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{ID: uuid.MustParse(modelID), Alias: "orders"}, nil)
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "auto").Return(auto, nil)
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "manual").Return(manual, nil)
		return mockColumn, svc
	}

	t.Run("bulk update refuses an auto field", func(t *testing.T) {
		mockColumn, svc := setupWrites()
		err := svc.BulkUpdateColumns(context.Background(), "schema", modelID, "auto", []dto.UpdateColumnsRequest{{Id: 1, Value: "x"}})
		assert.ErrorIs(t, err, app_errors.UUIDReadOnly)
		mockColumn.AssertNotCalled(t, "BulkUpdate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("bulk update checks and normalises a manual field", func(t *testing.T) {
		mockColumn, svc := setupWrites()
		err := svc.BulkUpdateColumns(context.Background(), "schema", modelID, "manual", []dto.UpdateColumnsRequest{{Id: 1, Value: "nope"}})
		assert.ErrorIs(t, err, app_errors.UUIDInvalid)

		mockColumn.On("BulkUpdate", mock.Anything, "schema", "orders", "external_id", mock.Anything).Return(nil)
		err = svc.BulkUpdateColumns(context.Background(), "schema", modelID, "manual", []dto.UpdateColumnsRequest{{Id: 1, Value: "{3F6C2A9E-1B47-4D8A-9E05-7C2B4F1A8D63}"}})
		require.NoError(t, err)
		updates := mockColumn.Calls[len(mockColumn.Calls)-1].Arguments.Get(4).([]dto.UpdateColumnsRequest)
		assert.Equal(t, "3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63", updates[0].Value)
	})

	t.Run("reset refuses an auto field", func(t *testing.T) {
		_, svc := setupWrites()
		assert.ErrorIs(t, svc.ResetColumnValues(context.Background(), "schema", modelID, "auto"), app_errors.UUIDReadOnly)
	})
}

func TestCountEmptyCells(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	modelID := uuid.NewString()
	col := uuidCol(modelID, "external_id", nil)
	mockModel := &MockModelService{}
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{ID: uuid.MustParse(modelID), Alias: "orders"}, nil)
	mockColumn := &MockColumnService{}
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
	repo := &pkg.DatabaseService{TableService: &MockTableService{}, BulkService: &MockBulkService{}, DB: db}
	svc := services.NewTableManagementService("postgres", repo, mockModel, mockColumn, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})

	sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FILTER \(WHERE "external_id" IS NULL\), COUNT\("external_id"\) FROM "schema"."orders"`).
		WillReturnRows(sqlmock.NewRows([]string{"empty", "filled"}).AddRow(128, 1872))

	counts, err := svc.(interface {
		CountEmptyCells(ctx context.Context, schemaName string, columnID string) (dto.EmptyCountResponse, error)
	}).CountEmptyCells(context.Background(), "schema", "cid")

	require.NoError(t, err)
	assert.Equal(t, dto.EmptyCountResponse{Empty: 128, Filled: 1872}, counts)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}
