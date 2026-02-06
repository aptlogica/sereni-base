package common_test

import (
	"context"
	"errors"
	"testing"

	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	services "serenibase/internal/services/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTableServiceGeneric is a mock implementation of TableService
type MockTableServiceGeneric struct {
	mock.Mock
}

func (m *MockTableServiceGeneric) GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
	args := m.Called(tableName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceGeneric) CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceGeneric) UpdateRecord(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, id, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceGeneric) DeleteRecord(tableName string, id interface{}) error {
	args := m.Called(tableName, id)
	return args.Error(0)
}

func (m *MockTableServiceGeneric) GetTables(schema string) ([]dbModels.Table, error) {
	args := m.Called(schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbModels.Table), args.Error(1)
}

func (m *MockTableServiceGeneric) CreateTable(req dbModels.CreateTableRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockTableServiceGeneric) AddColumn(tableName string, req dbModels.AddColumnRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableServiceGeneric) AlterTable(tableName string, req dbModels.AlterTableRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableServiceGeneric) BuildComplexQuery(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error) {
	args := m.Called(tableName, filters)
	return args.Get(0).(dbModels.QueryParams), args.Error(1)
}

func (m *MockTableServiceGeneric) CreateSchema(ctx context.Context, schemaName string) error {
	args := m.Called(ctx, schemaName)
	return args.Error(0)
}

func (m *MockTableServiceGeneric) DropTable(ctx context.Context, tableName string) error {
	args := m.Called(ctx, tableName)
	return args.Error(0)
}

func (m *MockTableServiceGeneric) CreateView(ctx context.Context, viewName string, viewSQL string) error {
	args := m.Called(ctx, viewName, viewSQL)
	return args.Error(0)
}

func (m *MockTableServiceGeneric) CreateFunction(ctx context.Context, functionName string, functionSQL string) error {
	args := m.Called(ctx, functionName, functionSQL)
	return args.Error(0)
}

func (m *MockTableServiceGeneric) GetByFunction(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error) {
	mockArgs := m.Called(ctx, functionName, args)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).([]map[string]interface{}), mockArgs.Error(1)
}

// TestStruct for mapping tests
type TestUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// TestNewGenericBaseService tests the NewGenericBaseService constructor
func TestNewGenericBaseService(t *testing.T) {
	t.Run("Create new generic base service", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}

		service := services.NewGenericBaseService(repo)

		assert.NotNil(t, service)
		assert.Equal(t, repo, service.GetRepository())
	})
}

// TestGenericBaseService_CreateRecord tests the CreateRecord method
func TestGenericBaseService_CreateRecord(t *testing.T) {
	t.Run("Success - Create record", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		data := map[string]interface{}{
			"email": "test@example.com",
			"age":   30,
		}
		expectedResult := map[string]interface{}{
			"id":    "123",
			"email": "test@example.com",
			"age":   30,
		}

		mockTableService.On("CreateRecord", "users", data).Return(expectedResult, nil)

		result, err := service.CreateRecord("users", data)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Create record fails", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		data := map[string]interface{}{
			"email": "test@example.com",
		}
		expectedErr := errors.New("create error")

		mockTableService.On("CreateRecord", "users", data).Return(nil, expectedErr)

		result, err := service.CreateRecord("users", data)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
		mockTableService.AssertExpectations(t)
	})
}

// TestGenericBaseService_GetSingleRecord tests the GetSingleRecord method
func TestGenericBaseService_GetSingleRecord(t *testing.T) {
	t.Run("Success - Get single record", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{
				"id":    "456",
				"email": "user@example.com",
				"age":   float64(25),
			},
		}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		result, err := service.GetSingleRecord(context.Background(), "users", query, "error msg")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "456", result["id"])
		assert.Equal(t, "user@example.com", result["email"])
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Database error", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		query := dbModels.QueryParams{}
		expectedErr := errors.New("db error")

		mockTableService.On("GetTableData", "users", query).Return(nil, expectedErr)

		result, err := service.GetSingleRecord(context.Background(), "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.DatabaseError, err)
		assert.Nil(t, result)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Record not found", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		query := dbModels.QueryParams{}
		data := []map[string]interface{}{}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		result, err := service.GetSingleRecord(context.Background(), "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrRecordNotFound, err)
		assert.Nil(t, result)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - With nil context", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{
				"id": "789",
			},
		}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		result, err := service.GetSingleRecord(nil, "users", query, "error msg")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "789", result["id"])
		mockTableService.AssertExpectations(t)
	})
}

// TestGenericBaseService_GetMultipleRecords tests the GetMultipleRecords method
func TestGenericBaseService_GetMultipleRecords(t *testing.T) {
	t.Run("Success - Get multiple records", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{"id": "1", "email": "user1@example.com"},
			{"id": "2", "email": "user2@example.com"},
		}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		results, err := service.GetMultipleRecords(context.Background(), "users", query, "error msg")

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "1", results[0]["id"])
		assert.Equal(t, "2", results[1]["id"])
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - Empty result", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		query := dbModels.QueryParams{}
		data := []map[string]interface{}{}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		results, err := service.GetMultipleRecords(context.Background(), "users", query, "error msg")

		assert.NoError(t, err)
		assert.Len(t, results, 0)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Database error", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		query := dbModels.QueryParams{}
		expectedErr := errors.New("db error")

		mockTableService.On("GetTableData", "users", query).Return(nil, expectedErr)

		results, err := service.GetMultipleRecords(context.Background(), "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.DatabaseError, err)
		assert.Nil(t, results)
		mockTableService.AssertExpectations(t)
	})
}

// TestGenericBaseService_UpdateRecord tests the UpdateRecord method
func TestGenericBaseService_UpdateRecord(t *testing.T) {
	t.Run("Success - Update record", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		updateData := map[string]interface{}{
			"email": "updated@example.com",
		}
		expectedResult := map[string]interface{}{
			"id":    "123",
			"email": "updated@example.com",
		}

		mockTableService.On("UpdateRecord", "users", "123", updateData).Return(expectedResult, nil)

		result, err := service.UpdateRecord("users", "123", updateData)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Update fails", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		updateData := map[string]interface{}{
			"email": "updated@example.com",
		}
		expectedErr := errors.New("update error")

		mockTableService.On("UpdateRecord", "users", "123", updateData).Return(nil, expectedErr)

		result, err := service.UpdateRecord("users", "123", updateData)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
		mockTableService.AssertExpectations(t)
	})
}

// TestGenericBaseService_DeleteRecord tests the DeleteRecord method
func TestGenericBaseService_DeleteRecord(t *testing.T) {
	t.Run("Success - Delete record", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		mockTableService.On("DeleteRecord", "users", "123").Return(nil)

		err := service.DeleteRecord("users", "123")

		assert.NoError(t, err)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - Delete record with filter", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		filter := map[string]interface{}{
			"email": "test@example.com",
		}

		mockTableService.On("DeleteRecord", "users", filter).Return(nil)

		err := service.DeleteRecord("users", filter)

		assert.NoError(t, err)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Delete fails", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		expectedErr := errors.New("delete error")

		mockTableService.On("DeleteRecord", "users", "123").Return(expectedErr)

		err := service.DeleteRecord("users", "123")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockTableService.AssertExpectations(t)
	})
}

// TestGenericBaseService_CountRecords tests the CountRecords method
func TestGenericBaseService_CountRecords(t *testing.T) {
	t.Run("Success - Count records", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		countData := []map[string]interface{}{
			{"total": float64(50)},
		}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT"
		})).Return(countData, nil)

		count, err := service.CountRecords(context.Background(), "users", "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(50), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - Empty result returns 0", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		countData := []map[string]interface{}{}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT"
		})).Return(countData, nil)

		count, err := service.CountRecords(context.Background(), "users", "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - No total field returns 0", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		countData := []map[string]interface{}{
			{"other": "value"},
		}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT"
		})).Return(countData, nil)

		count, err := service.CountRecords(context.Background(), "users", "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Database error", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		expectedErr := errors.New("db error")

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT"
		})).Return(nil, expectedErr)

		count, err := service.CountRecords(context.Background(), "users", "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.DatabaseError, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})
}

// TestGenericBaseService_CountRecordsWithFilter tests the CountRecordsWithFilter method
func TestGenericBaseService_CountRecordsWithFilter(t *testing.T) {
	t.Run("Success - Count records with filter", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		filters := []dbModels.QueryFilter{
			{Column: "status", Operator: "=", Value: "active"},
		}
		countData := []map[string]interface{}{
			{"total": float64(25)},
		}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT" &&
				len(q.Filters) == 1 && q.Filters[0].Column == "status"
		})).Return(countData, nil)

		count, err := service.CountRecordsWithFilter(context.Background(), "users", filters, "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(25), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - Empty result with filter returns 0", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		filters := []dbModels.QueryFilter{
			{Column: "status", Operator: "=", Value: "inactive"},
		}
		countData := []map[string]interface{}{}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && len(q.Filters) == 1
		})).Return(countData, nil)

		count, err := service.CountRecordsWithFilter(context.Background(), "users", filters, "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - No total field with filter returns 0", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		filters := []dbModels.QueryFilter{
			{Column: "status", Operator: "=", Value: "active"},
		}
		countData := []map[string]interface{}{
			{"other": "value"},
		}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && len(q.Filters) == 1
		})).Return(countData, nil)

		count, err := service.CountRecordsWithFilter(context.Background(), "users", filters, "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Database error with filter", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		filters := []dbModels.QueryFilter{
			{Column: "status", Operator: "=", Value: "active"},
		}
		expectedErr := errors.New("db error")

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && len(q.Filters) == 1
		})).Return(nil, expectedErr)

		count, err := service.CountRecordsWithFilter(context.Background(), "users", filters, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.DatabaseError, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})
}

// TestGenericBaseService_MapToStruct tests the MapToStruct method
func TestGenericBaseService_MapToStruct(t *testing.T) {
	t.Run("Success - Map to struct", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		data := map[string]interface{}{
			"id":    "789",
			"email": "test@example.com",
			"age":   float64(30),
		}
		var result TestUser

		err := service.MapToStruct(data, &result)

		assert.NoError(t, err)
		assert.Equal(t, "789", result.ID)
		assert.Equal(t, "test@example.com", result.Email)
		assert.Equal(t, 30, result.Age)
	})

	t.Run("Error - MapToStruct fails", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		data := map[string]interface{}{
			"invalid": make(chan int),
		}
		var result TestUser

		err := service.MapToStruct(data, &result)

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrMapToStruct, err)
	})
}

// TestGenericBaseService_MapToStructList tests the MapToStructList method
func TestGenericBaseService_MapToStructList(t *testing.T) {
	t.Run("Success - Map to struct list", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		data := []map[string]interface{}{
			{
				"id":    "1",
				"email": "user1@example.com",
				"age":   float64(25),
			},
			{
				"id":    "2",
				"email": "user2@example.com",
				"age":   float64(35),
			},
		}
		var results struct {
			Items []interface{} `json:"items"`
		}

		err := service.MapToStructList(data, &results)

		assert.NoError(t, err)
		// The actual implementation wraps in items, so we verify it processed without error
	})

	t.Run("Success - Empty list", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		data := []map[string]interface{}{}
		var results struct {
			Items []interface{} `json:"items"`
		}

		err := service.MapToStructList(data, &results)

		assert.NoError(t, err)
	})

	t.Run("Error - MapToStruct fails in list", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		data := []map[string]interface{}{
			{
				"invalid": make(chan int),
			},
		}
		var results struct {
			Items []interface{} `json:"items"`
		}

		err := service.MapToStructList(data, &results)

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrMapToStruct, err)
	})
}

// TestGenericBaseService_GetRepository tests the GetRepository method
func TestGenericBaseService_GetRepository(t *testing.T) {
	t.Run("Get repository", func(t *testing.T) {
		mockTableService := new(MockTableServiceGeneric)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		service := services.NewGenericBaseService(repo)

		result := service.GetRepository()

		assert.NotNil(t, result)
		assert.Equal(t, repo, result)
		assert.Equal(t, mockTableService, result.TableService)
	})
}
