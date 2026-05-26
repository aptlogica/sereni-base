package table_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aptlogica/go-postgres-rest/pkg"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/table"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBulkService struct {
	mock.Mock
}

func (m *MockBulkService) BulkInsert(tableName string, records []map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(tableName, records)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockBulkService) Upsert(tableName string, data map[string]interface{}, conflictColumns []string, updateColumns []string) (map[string]interface{}, error) {
	args := m.Called(tableName, data, conflictColumns, updateColumns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockBulkService) BulkUpdate(tableName string, updates []map[string]interface{}, whereColumn string) (int64, error) {
	args := m.Called(tableName, updates, whereColumn)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBulkService) BulkDelete(tableName string, ids []interface{}, idColumn string) (int64, error) {
	args := m.Called(tableName, ids, idColumn)
	return args.Get(0).(int64), args.Error(1)
}

// setupColumnServiceWithBulk is kept for local helpers when needed.
// nolint:unused
func setupColumnServiceWithBulk() (*MockTableService, *MockBulkService) {
	mockTable := &MockTableService{}
	mockBulk := &MockBulkService{}
	_ = &pkg.DatabaseService{TableService: mockTable, BulkService: mockBulk}
	return mockTable, mockBulk
}

func TestNewColumnService(t *testing.T) {
	db, _ := setupMockDB()
	service := services.NewColumnService(db)
	assert.NotNil(t, service)
}

func TestCreateColumn(t *testing.T) {
	t.Run("create table error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)
		cm, ok := svc.(interface {
			CreateColumn(ctx context.Context, schemaName string) (tenant.Column, error)
		})
		assert.True(t, ok)

		mockTable.On("CreateTable", mock.Anything).Return(errors.New("fail"))

		_, err := cm.CreateColumn(context.Background(), "schema")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)
		cm, ok := svc.(interface {
			CreateColumn(ctx context.Context, schemaName string) (tenant.Column, error)
		})
		assert.True(t, ok)

		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("AddColumn", "\"schema\".\"columns\"", mock.Anything).
			Return(nil).Twice()

		_, err := cm.CreateColumn(context.Background(), "schema")

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})
}

func TestCreateColumnRecord(t *testing.T) {
	t.Run("create record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("AddColumn", "\"schema\".\"columns\"", mock.Anything).
			Return(errors.New("already exists")).Twice()
		mockTable.On("CreateRecord", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.Create(context.Background(), dto.ColumnInsertion{ModelID: uuid.New(), BaseID: uuid.New(), ColumnName: "c", Title: "C"}, "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("AddColumn", "\"schema\".\"columns\"", mock.Anything).
			Return(errors.New("boom")).Twice()
		mockTable.On("CreateRecord", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		_, err := svc.Create(context.Background(), dto.ColumnInsertion{ModelID: uuid.New(), BaseID: uuid.New(), ColumnName: "c", Title: "C"}, "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		var captured map[string]interface{}
		mockTable.On("AddColumn", "\"schema\".\"columns\"", mock.Anything).
			Return(nil).Twice()
		mockTable.On("CreateRecord", tenant.Column{}.TableName("schema"), mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(1).(map[string]interface{}) }).
			Return(map[string]interface{}{"id": uuid.New().String(), "title": "C"}, nil)

		col, err := svc.Create(context.Background(), dto.ColumnInsertion{ModelID: uuid.New(), BaseID: uuid.New(), ColumnName: "c", Title: "C", CreatedBy: "u"}, "schema")

		assert.NoError(t, err)
		assert.Equal(t, "C", col.Title)
		assert.Equal(t, "u", captured["last_modified_by"])
	})
}

func TestGetColumnByID(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetColumnByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetColumnByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.ColumnNotFound)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "title": "C"}}, nil)

		col, err := svc.GetColumnByID(context.Background(), "schema", id.String())

		assert.NoError(t, err)
		assert.Equal(t, id, col.ID)
	})
}

func TestFetchColumns_MapError(t *testing.T) {
	db, mockTable := setupMockDB()
	svc := services.NewColumnService(db)

	mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
		Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

	_, err := svc.GetAllColumns(context.Background(), "schema")

	assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
}

func TestUpdateColumn(t *testing.T) {
	t.Run("get column error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateColumn(context.Background(), "schema", "id", dto.ColumnUpdate{})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("update record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "C"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Column{}.TableName("schema"), id, mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateColumn(context.Background(), "schema", id, dto.ColumnUpdate{})

		assert.ErrorIs(t, err, app_errors.ColumnUpdateFailed)
	})

	t.Run("empty update result", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "C"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Column{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{}, nil)

		_, err := svc.UpdateColumn(context.Background(), "schema", id, dto.ColumnUpdate{})

		assert.ErrorIs(t, err, app_errors.InvalidPayload)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "C"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Column{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{"id": id}, nil)
		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "Updated"}}, nil).Once()

		col, err := svc.UpdateColumn(context.Background(), "schema", id, dto.ColumnUpdate{})

		assert.NoError(t, err)
		assert.Equal(t, "Updated", col.Title)
	})
}

func TestDeleteColumn(t *testing.T) {
	t.Run("get column error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		err := svc.DeleteColumn(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "C"}}, nil).Once()
		mockTable.On("DeleteRecord", tenant.Column{}.TableName("schema"), id).
			Return(errors.New("db error"))

		err := svc.DeleteColumn(context.Background(), "schema", id)

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "C"}}, nil).Once()
		mockTable.On("DeleteRecord", tenant.Column{}.TableName("schema"), id).
			Return(nil)

		err := svc.DeleteColumn(context.Background(), "schema", id)

		assert.NoError(t, err)
	})
}

func TestGetColumnByModelID(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetColumnByModelID(context.Background(), "schema", "model")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "C"}}, nil)

		cols, err := svc.GetColumnByModelID(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Len(t, cols, 1)
	})
}

func TestBulkInsert(t *testing.T) {
	t.Run("bulk error", func(t *testing.T) {
		mockTable := &MockTableService{}
		mockBulk := &MockBulkService{}
		db := &pkg.DatabaseService{TableService: mockTable, BulkService: mockBulk}
		svc := services.NewColumnService(db)

		mockBulk.On("BulkInsert", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.BulkInsert([]dto.ColumnInsertion{{ID: uuid.New()}}, "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		mockTable := &MockTableService{}
		mockBulk := &MockBulkService{}
		db := &pkg.DatabaseService{TableService: mockTable, BulkService: mockBulk}
		svc := services.NewColumnService(db)

		mockBulk.On("BulkInsert", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.BulkInsert([]dto.ColumnInsertion{{ID: uuid.New()}}, "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		mockTable := &MockTableService{}
		mockBulk := &MockBulkService{}
		db := &pkg.DatabaseService{TableService: mockTable, BulkService: mockBulk}
		svc := services.NewColumnService(db)

		mockBulk.On("BulkInsert", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "C"}}, nil)

		cols, err := svc.BulkInsert([]dto.ColumnInsertion{{ID: uuid.New()}}, "schema")

		assert.NoError(t, err)
		assert.Len(t, cols, 1)
	})
}

func TestGetMaxOrderIndex(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		getMax, ok := svc.(interface {
			GetMaxOrderIndex(ctx context.Context, schemaName, modelID string) ([]tenant.Column, error)
		})
		assert.True(t, ok)

		_, err := getMax.GetMaxOrderIndex(context.Background(), "schema", "model")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String()}}, nil)

		getMax, ok := svc.(interface {
			GetMaxOrderIndex(ctx context.Context, schemaName, modelID string) ([]tenant.Column, error)
		})
		assert.True(t, ok)

		cols, err := getMax.GetMaxOrderIndex(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Len(t, cols, 1)
	})
}

func TestGetMaxOrderIndexOfColumn(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetMaxOrderIndexOfColumn(context.Background(), "schema", "model")

		assert.Error(t, err)
	})

	t.Run("no data", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		val, err := svc.GetMaxOrderIndexOfColumn(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Equal(t, float64(0), val)
	})

	t.Run("int", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"max": int(3)}}, nil)

		val, err := svc.GetMaxOrderIndexOfColumn(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Equal(t, float64(3), val)
	})

	t.Run("int64", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"max": int64(4)}}, nil)

		val, err := svc.GetMaxOrderIndexOfColumn(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Equal(t, float64(4), val)
	})

	t.Run("float64", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"max": float64(5.5)}}, nil)

		val, err := svc.GetMaxOrderIndexOfColumn(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Equal(t, float64(5.5), val)
	})

	t.Run("float32", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"max": float32(6.5)}}, nil)

		val, err := svc.GetMaxOrderIndexOfColumn(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Equal(t, float64(6.5), val)
	})

	t.Run("nil max", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetTableData", tenant.Column{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"max": nil}}, nil)

		val, err := svc.GetMaxOrderIndexOfColumn(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Equal(t, float64(0), val)
	})
}

func TestBulkUpdate(t *testing.T) {
	t.Run("empty updates", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewColumnService(db)

		err := svc.BulkUpdate(context.Background(), "schema", "table", "column", []dto.UpdateColumnsRequest{})

		assert.NoError(t, err)
	})

	t.Run("get by function error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetByFunction", mock.Anything, "schema.bulk_update", mock.Anything).
			Return(nil, errors.New("function error"))

		err := svc.BulkUpdate(context.Background(), "schema", "table", "column", []dto.UpdateColumnsRequest{
			{Id: "row1", Value: "newValue1"},
		})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("success with single update", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetByFunction", mock.Anything, "schema.bulk_update", mock.MatchedBy(func(args map[string]interface{}) bool {
			return args["p_schema_name"] == "schema" &&
				args["p_table_name"] == "table" &&
				args["p_column_name"] == "column" &&
				args["p_data"] != nil
		})).
			Return([]map[string]interface{}{}, nil)

		err := svc.BulkUpdate(context.Background(), "schema", "table", "column", []dto.UpdateColumnsRequest{
			{Id: "row1", Value: "newValue1"},
		})

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})

	t.Run("success with multiple updates", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetByFunction", mock.Anything, "schema.bulk_update", mock.MatchedBy(func(args map[string]interface{}) bool {
			return args["p_schema_name"] == "schema" &&
				args["p_table_name"] == "users" &&
				args["p_column_name"] == "status" &&
				args["p_data"] != nil
		})).
			Return([]map[string]interface{}{}, nil)

		err := svc.BulkUpdate(context.Background(), "schema", "users", "status", []dto.UpdateColumnsRequest{
			{Id: "user1", Value: "active"},
			{Id: "user2", Value: "inactive"},
			{Id: "user3", Value: "pending"},
		})

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})

	t.Run("success with different value types", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetByFunction", mock.Anything, "schema.bulk_update", mock.Anything).
			Return([]map[string]interface{}{}, nil)

		// Test with numeric value
		err := svc.BulkUpdate(context.Background(), "schema", "table", "column", []dto.UpdateColumnsRequest{
			{Id: "row1", Value: 42},
		})
		assert.NoError(t, err)

		// Test with boolean value
		err = svc.BulkUpdate(context.Background(), "schema", "table", "column", []dto.UpdateColumnsRequest{
			{Id: "row2", Value: true},
		})
		assert.NoError(t, err)

		// Test with null value
		err = svc.BulkUpdate(context.Background(), "schema", "table", "column", []dto.UpdateColumnsRequest{
			{Id: "row3", Value: nil},
		})
		assert.NoError(t, err)
	})
}

func TestResetColumn(t *testing.T) {
	t.Run("get by function error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetByFunction", mock.Anything, "schema.reset_column", mock.Anything).
			Return(nil, errors.New("function error"))

		err := svc.ResetColumn(context.Background(), "schema", "table", "column")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetByFunction", mock.Anything, "schema.reset_column", mock.MatchedBy(func(args map[string]interface{}) bool {
			return args["p_schema_name"] == "schema" &&
				args["p_table_name"] == "table" &&
				args["p_column_name"] == "column"
		})).
			Return([]map[string]interface{}{}, nil)

		err := svc.ResetColumn(context.Background(), "schema", "table", "column")

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})

	t.Run("success with different table names", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetByFunction", mock.Anything, "myschema.reset_column", mock.MatchedBy(func(args map[string]interface{}) bool {
			return args["p_schema_name"] == "myschema" &&
				args["p_table_name"] == "users" &&
				args["p_column_name"] == "created_at"
		})).
			Return([]map[string]interface{}{}, nil)

		err := svc.ResetColumn(context.Background(), "myschema", "users", "created_at")

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})

	t.Run("success with multiple calls", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewColumnService(db)

		mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).
			Return([]map[string]interface{}{}, nil)

		err1 := svc.ResetColumn(context.Background(), "schema1", "table1", "col1")
		err2 := svc.ResetColumn(context.Background(), "schema2", "table2", "col2")
		err3 := svc.ResetColumn(context.Background(), "schema3", "table3", "col3")

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
		mockTable.AssertNumberOfCalls(t, "GetByFunction", 3)
	})
}
