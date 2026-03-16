package base_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/base"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTableService is a mock implementation of TableService
// (copied pattern from other service tests)
type MockTableService struct {
	mock.Mock
}

func (m *MockTableService) GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
	args := m.Called(tableName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockTableService) CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableService) UpdateRecord(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, id, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableService) DeleteRecord(tableName string, id interface{}) error {
	args := m.Called(tableName, id)
	return args.Error(0)
}

func (m *MockTableService) GetTables(schema string) ([]dbModels.Table, error) {
	args := m.Called(schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbModels.Table), args.Error(1)
}

func (m *MockTableService) CreateTable(req dbModels.CreateTableRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockTableService) AddColumn(tableName string, req dbModels.AddColumnRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableService) AlterTable(tableName string, req dbModels.AlterTableRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableService) BuildComplexQuery(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error) {
	args := m.Called(tableName, filters)
	return args.Get(0).(dbModels.QueryParams), args.Error(1)
}

func (m *MockTableService) CreateSchema(ctx context.Context, schemaName string) error {
	args := m.Called(ctx, schemaName)
	return args.Error(0)
}

func (m *MockTableService) DropTable(ctx context.Context, tableName string) error {
	args := m.Called(ctx, tableName)
	return args.Error(0)
}

func (m *MockTableService) CreateView(ctx context.Context, viewName string, viewSQL string) error {
	args := m.Called(ctx, viewName, viewSQL)
	return args.Error(0)
}

func (m *MockTableService) CreateFunction(ctx context.Context, functionName string, functionSQL string) error {
	args := m.Called(ctx, functionName, functionSQL)
	return args.Error(0)
}

func (m *MockTableService) GetByFunction(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error) {
	mockArgs := m.Called(ctx, functionName, args)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).([]map[string]interface{}), mockArgs.Error(1)
}

func setupMockDB() (*pkg.DatabaseService, *MockTableService) {
	mockTable := &MockTableService{}

	db := &pkg.DatabaseService{
		TableService: mockTable,
	}

	return db, mockTable
}

func TestNewBaseService(t *testing.T) {
	db, _ := setupMockDB()

	service := services.NewBaseService(db)

	assert.NotNil(t, service)
}

func TestBaseInsertion_DefaultsAndSuccess(t *testing.T) {
	db, mockTable := setupMockDB()
	service := services.NewBaseService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	workspaceID := uuid.New()
	createdBy := "user-123"

	req := dto.BaseInsertion{
		WorkspaceID: workspaceID,
		Title:       "Test Base",
		CreatedBy:   createdBy,
		UpdatedBy:   "",
	}

	baseID := uuid.New()
	now := time.Now().UTC()
	returned := map[string]interface{}{
		"id":                 baseID.String(),
		"workspace_id":       workspaceID.String(),
		"title":              "Test Base",
		"description":        nil,
		"type":               "internal",
		"config":             map[string]interface{}{},
		"settings":           map[string]interface{}{},
		"meta":               map[string]interface{}{},
		"status":             "active",
		"visibility":         "private",
		"table_count":        0,
		"row_count":          int64(0),
		"storage_used_bytes": int64(0),
		"created_by":         createdBy,
		"last_modified_by":   createdBy,
		"created_time":       now,
		"last_modified_time": now,
	}

	var captured map[string]interface{}
	mockTable.On("CreateRecord", tenant.Base{}.TableName(schemaName), mock.Anything).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(map[string]interface{})
		}).
		Return(returned, nil)

	result, err := service.BaseInsertion(ctx, req, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, baseID, result.ID)
	assert.Equal(t, workspaceID.String(), result.WorkspaceID)
	assert.NotNil(t, captured)
	assert.Equal(t, "internal", captured["type"])
	assert.Equal(t, "active", captured["status"])
	assert.Equal(t, "private", captured["visibility"])
	assert.Equal(t, "{}", captured["config"])
	assert.Equal(t, "{}", captured["settings"])
	assert.Equal(t, "{}", captured["meta"])
	assert.Equal(t, createdBy, captured["last_modified_by"])

	mockTable.AssertExpectations(t)
}

func TestBaseInsertion_CreateRecordError(t *testing.T) {
	db, mockTable := setupMockDB()
	service := services.NewBaseService(db)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("CreateRecord", tenant.Base{}.TableName(schemaName), mock.Anything).
		Return(nil, errors.New("db error"))

	_, err := service.BaseInsertion(ctx, dto.BaseInsertion{WorkspaceID: uuid.New(), Title: "t"}, schemaName)

	assert.ErrorIs(t, err, app_errors.DatabaseError)
	mockTable.AssertExpectations(t)
}

func TestBaseInsertion_MapToStructError(t *testing.T) {
	db, mockTable := setupMockDB()
	service := services.NewBaseService(db)

	ctx := context.Background()
	schemaName := "test_schema"

	badReturned := map[string]interface{}{
		"id":     uuid.New().String(),
		"config": func() {},
	}

	mockTable.On("CreateRecord", tenant.Base{}.TableName(schemaName), mock.Anything).
		Return(badReturned, nil)

	_, err := service.BaseInsertion(ctx, dto.BaseInsertion{WorkspaceID: uuid.New(), Title: "t"}, schemaName)

	assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	mockTable.AssertExpectations(t)
}

func TestCreateBase(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("CreateTable", mock.Anything).Return(errors.New("fail"))

		_, err := service.CreateBase(context.Background(), "schema")

		assert.Error(t, err)
		mockTable.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("CreateTable", mock.Anything).Return(nil)

		_, err := service.CreateBase(context.Background(), "schema")

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})
}

func TestGetBaseByID(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		db, _ := setupMockDB()
		service := services.NewBaseService(db)

		_, err := service.GetBaseByID(context.Background(), "schema", "")

		assert.ErrorIs(t, err, app_errors.InvalidPayload)
	})

	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := service.GetBaseByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := service.GetBaseByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.BaseNotFound)
		mockTable.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		id := uuid.New()
		row := map[string]interface{}{
			"id":           id.String(),
			"workspace_id": "ws",
			"title":        "Title",
		}

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{row}, nil)

		base, err := service.GetBaseByID(context.Background(), "schema", id.String())

		assert.NoError(t, err)
		assert.Equal(t, id, base.ID)
		mockTable.AssertExpectations(t)
	})
}

func TestGetAllBases(t *testing.T) {
	db, mockTable := setupMockDB()
	service := services.NewBaseService(db)

	row := map[string]interface{}{
		"id":           uuid.New().String(),
		"workspace_id": "ws",
		"title":        "Title",
	}

	mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
		Return([]map[string]interface{}{row}, nil)

	bases, err := service.GetAllBases(context.Background(), "schema")

	assert.NoError(t, err)
	assert.Len(t, bases, 1)
	mockTable.AssertExpectations(t)
}

func TestFetchBases_MapToStructError(t *testing.T) {
	db, mockTable := setupMockDB()
	service := services.NewBaseService(db)

	badRow := map[string]interface{}{"id": make(chan int)}
	mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
		Return([]map[string]interface{}{badRow}, nil)

	_, err := service.GetAllBases(context.Background(), "schema")

	assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	mockTable.AssertExpectations(t)
}

func TestUpdateBase(t *testing.T) {
	t.Run("get base error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := service.UpdateBase(context.Background(), "schema", "id", dto.BaseUpdate{})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("update record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		id := uuid.New().String()
		row := map[string]interface{}{
			"id":           id,
			"workspace_id": "ws",
			"title":        "Title",
		}

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{row}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Base{}.TableName("schema"), id, mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := service.UpdateBase(context.Background(), "schema", id, dto.BaseUpdate{})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("update record empty", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		id := uuid.New().String()
		row := map[string]interface{}{
			"id":           id,
			"workspace_id": "ws",
			"title":        "Title",
		}

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{row}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Base{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{}, nil)

		_, err := service.UpdateBase(context.Background(), "schema", id, dto.BaseUpdate{})

		assert.ErrorIs(t, err, app_errors.InvalidPayload)
		mockTable.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		id := uuid.New()
		row := map[string]interface{}{
			"id":           id.String(),
			"workspace_id": "ws",
			"title":        "Title",
		}
		updatedRow := map[string]interface{}{
			"id":           id.String(),
			"workspace_id": "ws",
			"title":        "Updated",
		}

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{row}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Base{}.TableName("schema"), id.String(), mock.Anything).
			Return(map[string]interface{}{"id": id.String()}, nil)
		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{updatedRow}, nil).Once()

		updated, err := service.UpdateBase(context.Background(), "schema", id.String(), dto.BaseUpdate{})

		assert.NoError(t, err)
		assert.Equal(t, "Updated", updated.Title)
		mockTable.AssertExpectations(t)
	})
}

func TestDeleteBase(t *testing.T) {
	t.Run("get base error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		err := service.DeleteBase(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		id := uuid.New().String()
		row := map[string]interface{}{"id": id, "workspace_id": "ws", "title": "Title"}
		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{row}, nil).Once()
		mockTable.On("DeleteRecord", tenant.Base{}.TableName("schema"), id).
			Return(errors.New("db error"))

		err := service.DeleteBase(context.Background(), "schema", id)

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		id := uuid.New().String()
		row := map[string]interface{}{"id": id, "workspace_id": "ws", "title": "Title"}
		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{row}, nil).Once()
		mockTable.On("DeleteRecord", tenant.Base{}.TableName("schema"), id).
			Return(nil)

		err := service.DeleteBase(context.Background(), "schema", id)

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})
}

func TestGetBasesByWorkspace(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := service.GetBasesByWorkspace(context.Background(), "schema", "ws")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		row := map[string]interface{}{"id": uuid.New().String(), "workspace_id": "ws", "title": "Title"}
		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{row}, nil)

		bases, err := service.GetBasesByWorkspace(context.Background(), "schema", "ws")

		assert.NoError(t, err)
		assert.Len(t, bases, 1)
		mockTable.AssertExpectations(t)
	})
}

func TestGetBulkbases(t *testing.T) {
	t.Run("empty ids", func(t *testing.T) {
		db, _ := setupMockDB()
		service := services.NewBaseService(db)

		bases, err := service.GetBulkbases(context.Background(), "schema", []string{})

		assert.NoError(t, err)
		assert.Empty(t, bases)
	})

	t.Run("error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := service.GetBulkbases(context.Background(), "schema", []string{"a"})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertExpectations(t)
	})

	t.Run("empty rows", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		bases, err := service.GetBulkbases(context.Background(), "schema", []string{"a"})

		assert.NoError(t, err)
		assert.Empty(t, bases)
		mockTable.AssertExpectations(t)
	})

	t.Run("map to struct error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		badRow := map[string]interface{}{"id": make(chan int)}
		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{badRow}, nil)

		_, err := service.GetBulkbases(context.Background(), "schema", []string{"a"})

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
		mockTable.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		service := services.NewBaseService(db)

		row := map[string]interface{}{"id": uuid.New().String(), "workspace_id": "ws", "title": "Title"}
		mockTable.On("GetTableData", tenant.Base{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{row}, nil)

		bases, err := service.GetBulkbases(context.Background(), "schema", []string{"a"})

		assert.NoError(t, err)
		assert.Len(t, bases, 1)
		mockTable.AssertExpectations(t)
	})
}
