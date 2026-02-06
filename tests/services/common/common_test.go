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

// MockTableService is a mock implementation of TableService
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

// MockServiceWithSingle is a mock that implements the service interface used by GetSingleRecord
type MockServiceWithSingle struct {
	mock.Mock
}

func (m *MockServiceWithSingle) GetSingleRecord(ctx interface{}, tableName string, query dbModels.QueryParams, errorMsg string) (map[string]interface{}, error) {
	args := m.Called(ctx, tableName, query, errorMsg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockServiceWithMultiple is a mock that implements the service interface used by ListRecords
type MockServiceWithMultiple struct {
	mock.Mock
}

func (m *MockServiceWithMultiple) GetMultipleRecords(ctx interface{}, tableName string, query dbModels.QueryParams, errorMsg string) ([]map[string]interface{}, error) {
	args := m.Called(ctx, tableName, query, errorMsg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

// MockServiceWithCount is a mock that implements the service interface used by CountRecords
type MockServiceWithCount struct {
	mock.Mock
}

func (m *MockServiceWithCount) CountRecords(ctx interface{}, tableName string, errorMsg string) (int64, error) {
	args := m.Called(ctx, tableName, errorMsg)
	return args.Get(0).(int64), args.Error(1)
}

// TestStruct is a test struct for mapping
type TestStruct struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// TestGetSingleRecord tests the GetSingleRecord function
func TestGetSingleRecord(t *testing.T) {
	t.Run("Success - Single record fetched and mapped", func(t *testing.T) {
		mockService := new(MockServiceWithSingle)
		query := dbModels.QueryParams{}
		data := map[string]interface{}{
			"id":   "123",
			"name": "John Doe",
			"age":  float64(30),
		}

		mockService.On("GetSingleRecord", nil, "users", query, "error msg").Return(data, nil)

		result, err := services.GetSingleRecord[TestStruct](mockService, "users", query, "error msg")

		assert.NoError(t, err)
		assert.Equal(t, "123", result.ID)
		assert.Equal(t, "John Doe", result.Name)
		assert.Equal(t, 30, result.Age)
		mockService.AssertExpectations(t)
	})

	t.Run("Error - Service returns error", func(t *testing.T) {
		mockService := new(MockServiceWithSingle)
		query := dbModels.QueryParams{}
		expectedErr := errors.New("database error")

		mockService.On("GetSingleRecord", nil, "users", query, "error msg").Return(nil, expectedErr)

		result, err := services.GetSingleRecord[TestStruct](mockService, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, TestStruct{}, result)
		mockService.AssertExpectations(t)
	})

	t.Run("Error - MapToStruct fails", func(t *testing.T) {
		mockService := new(MockServiceWithSingle)
		query := dbModels.QueryParams{}
		// Invalid data that will fail to map
		data := map[string]interface{}{
			"id":   123,            // should be string
			"name": make(chan int), // cannot be marshaled
		}

		mockService.On("GetSingleRecord", nil, "users", query, "error msg").Return(data, nil)

		result, err := services.GetSingleRecord[TestStruct](mockService, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrMapToStruct, err)
		assert.Equal(t, TestStruct{}, result)
		mockService.AssertExpectations(t)
	})
}

// TestGetSingleRecordWithRepo tests the GetSingleRecordWithRepo function
func TestGetSingleRecordWithRepo(t *testing.T) {
	t.Run("Success - Single record fetched from repo", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{
				"id":   "456",
				"name": "Jane Smith",
				"age":  float64(25),
			},
		}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		result, err := services.GetSingleRecordWithRepo[TestStruct](repo, "users", query, "error msg")

		assert.NoError(t, err)
		assert.Equal(t, "456", result.ID)
		assert.Equal(t, "Jane Smith", result.Name)
		assert.Equal(t, 25, result.Age)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Database error", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		query := dbModels.QueryParams{}
		expectedErr := errors.New("db error")

		mockTableService.On("GetTableData", "users", query).Return(nil, expectedErr)

		result, err := services.GetSingleRecordWithRepo[TestStruct](repo, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.DatabaseError, err)
		assert.Equal(t, TestStruct{}, result)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - No records found", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		result, err := services.GetSingleRecordWithRepo[TestStruct](repo, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrRecordNotFound, err)
		assert.Equal(t, TestStruct{}, result)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - MapToStruct fails", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{
				"invalid": make(chan int),
			},
		}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		result, err := services.GetSingleRecordWithRepo[TestStruct](repo, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrMapToStruct, err)
		assert.Equal(t, TestStruct{}, result)
		mockTableService.AssertExpectations(t)
	})
}

// TestListRecords tests the ListRecords function
func TestListRecords(t *testing.T) {
	t.Run("Success - Multiple records fetched and mapped", func(t *testing.T) {
		mockService := new(MockServiceWithMultiple)
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{
				"id":   "1",
				"name": "User1",
				"age":  float64(20),
			},
			{
				"id":   "2",
				"name": "User2",
				"age":  float64(30),
			},
		}

		mockService.On("GetMultipleRecords", nil, "users", query, "error msg").Return(data, nil)

		results, err := services.ListRecords[TestStruct](mockService, "users", query, "error msg")

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "1", results[0].ID)
		assert.Equal(t, "User1", results[0].Name)
		assert.Equal(t, "2", results[1].ID)
		assert.Equal(t, "User2", results[1].Name)
		mockService.AssertExpectations(t)
	})

	t.Run("Success - Empty list", func(t *testing.T) {
		mockService := new(MockServiceWithMultiple)
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{}

		mockService.On("GetMultipleRecords", nil, "users", query, "error msg").Return(data, nil)

		results, err := services.ListRecords[TestStruct](mockService, "users", query, "error msg")

		assert.NoError(t, err)
		assert.Len(t, results, 0)
		mockService.AssertExpectations(t)
	})

	t.Run("Error - Service returns error", func(t *testing.T) {
		mockService := new(MockServiceWithMultiple)
		query := dbModels.QueryParams{}
		expectedErr := errors.New("service error")

		mockService.On("GetMultipleRecords", nil, "users", query, "error msg").Return(nil, expectedErr)

		results, err := services.ListRecords[TestStruct](mockService, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, results)
		mockService.AssertExpectations(t)
	})

	t.Run("Error - MapToStruct fails for one record", func(t *testing.T) {
		mockService := new(MockServiceWithMultiple)
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{
				"id":   "1",
				"name": "User1",
				"age":  float64(20),
			},
			{
				"invalid": make(chan int),
			},
		}

		mockService.On("GetMultipleRecords", nil, "users", query, "error msg").Return(data, nil)

		results, err := services.ListRecords[TestStruct](mockService, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrMapToStruct, err)
		assert.Nil(t, results)
		mockService.AssertExpectations(t)
	})
}

// TestListRecordsWithRepo tests the ListRecordsWithRepo function
func TestListRecordsWithRepo(t *testing.T) {
	t.Run("Success - Multiple records from repo", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{
				"id":   "10",
				"name": "Alice",
				"age":  float64(28),
			},
			{
				"id":   "20",
				"name": "Bob",
				"age":  float64(32),
			},
		}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		results, err := services.ListRecordsWithRepo[TestStruct](repo, "users", query, "error msg")

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "10", results[0].ID)
		assert.Equal(t, "Alice", results[0].Name)
		assert.Equal(t, "20", results[1].ID)
		assert.Equal(t, "Bob", results[1].Name)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - Empty result", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		results, err := services.ListRecordsWithRepo[TestStruct](repo, "users", query, "error msg")

		assert.NoError(t, err)
		assert.Len(t, results, 0)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Database error", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		query := dbModels.QueryParams{}
		expectedErr := errors.New("db error")

		mockTableService.On("GetTableData", "users", query).Return(nil, expectedErr)

		results, err := services.ListRecordsWithRepo[TestStruct](repo, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.DatabaseError, err)
		assert.Nil(t, results)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - MapToStruct fails", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		query := dbModels.QueryParams{}
		data := []map[string]interface{}{
			{
				"invalid": make(chan int),
			},
		}

		mockTableService.On("GetTableData", "users", query).Return(data, nil)

		results, err := services.ListRecordsWithRepo[TestStruct](repo, "users", query, "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrMapToStruct, err)
		assert.Nil(t, results)
		mockTableService.AssertExpectations(t)
	})
}

// TestCountRecords tests the CountRecords function
func TestCountRecords(t *testing.T) {
	t.Run("Success - Count returned", func(t *testing.T) {
		mockService := new(MockServiceWithCount)

		mockService.On("CountRecords", nil, "users", "error msg").Return(int64(42), nil)

		count, err := services.CountRecords(mockService, "users", "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(42), count)
		mockService.AssertExpectations(t)
	})

	t.Run("Error - Service returns error", func(t *testing.T) {
		mockService := new(MockServiceWithCount)
		expectedErr := errors.New("count error")

		mockService.On("CountRecords", nil, "users", "error msg").Return(int64(0), expectedErr)

		count, err := services.CountRecords(mockService, "users", "error msg")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, int64(0), count)
		mockService.AssertExpectations(t)
	})
}

// TestCountRecordsWithRepo tests the CountRecordsWithRepo function
func TestCountRecordsWithRepo(t *testing.T) {
	t.Run("Success - Count from repo", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		countData := []map[string]interface{}{
			{
				"total": float64(100),
			},
		}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT"
		})).Return(countData, nil)

		count, err := services.CountRecordsWithRepo(repo, "users", "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(100), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - Empty count result returns 0", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		countData := []map[string]interface{}{}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT"
		})).Return(countData, nil)

		count, err := services.CountRecordsWithRepo(repo, "users", "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Success - Count result without total field returns 0", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		countData := []map[string]interface{}{
			{
				"other": "value",
			},
		}

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT"
		})).Return(countData, nil)

		count, err := services.CountRecordsWithRepo(repo, "users", "error msg")

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})

	t.Run("Error - Database error", func(t *testing.T) {
		mockTableService := new(MockTableService)
		repo := &pkg.DatabaseService{
			TableService: mockTableService,
		}
		expectedErr := errors.New("db error")

		mockTableService.On("GetTableData", "users", mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 1 && q.Aggregates[0].Function == "COUNT"
		})).Return(nil, expectedErr)

		count, err := services.CountRecordsWithRepo(repo, "users", "error msg")

		assert.Error(t, err)
		assert.Equal(t, app_errors.DatabaseError, err)
		assert.Equal(t, int64(0), count)
		mockTableService.AssertExpectations(t)
	})
}

// TestCreateSingleFilterQuery tests the CreateSingleFilterQuery function
func TestCreateSingleFilterQuery(t *testing.T) {
	t.Run("Create single filter query", func(t *testing.T) {
		query := services.CreateSingleFilterQuery("email", "=", "test@example.com", 10)

		assert.Len(t, query.Filters, 1)
		assert.Equal(t, "email", query.Filters[0].Column)
		assert.Equal(t, "=", query.Filters[0].Operator)
		assert.Equal(t, "test@example.com", query.Filters[0].Value)
		assert.NotNil(t, query.Limit)
		assert.Equal(t, 10, *query.Limit)
	})

	t.Run("Create filter query with different operator", func(t *testing.T) {
		query := services.CreateSingleFilterQuery("age", ">", "18", 5)

		assert.Len(t, query.Filters, 1)
		assert.Equal(t, "age", query.Filters[0].Column)
		assert.Equal(t, ">", query.Filters[0].Operator)
		assert.Equal(t, "18", query.Filters[0].Value)
		assert.NotNil(t, query.Limit)
		assert.Equal(t, 5, *query.Limit)
	})
}

// TestAddFilter tests the AddFilter function
func TestAddFilter(t *testing.T) {
	t.Run("Add filter when value is not empty", func(t *testing.T) {
		query := &dbModels.QueryParams{}
		services.AddFilter(query, "name", "LIKE", "John")

		assert.Len(t, query.Filters, 1)
		assert.Equal(t, "name", query.Filters[0].Column)
		assert.Equal(t, "LIKE", query.Filters[0].Operator)
		assert.Equal(t, "John", query.Filters[0].Value)
	})

	t.Run("Do not add filter when value is empty", func(t *testing.T) {
		query := &dbModels.QueryParams{}
		services.AddFilter(query, "name", "LIKE", "")

		assert.Len(t, query.Filters, 0)
	})

	t.Run("Add multiple filters", func(t *testing.T) {
		query := &dbModels.QueryParams{}
		services.AddFilter(query, "status", "=", "active")
		services.AddFilter(query, "role", "=", "admin")
		services.AddFilter(query, "empty", "=", "")

		assert.Len(t, query.Filters, 2)
		assert.Equal(t, "status", query.Filters[0].Column)
		assert.Equal(t, "role", query.Filters[1].Column)
	})
}

// TestCreateMultiFilterQuery tests the CreateMultiFilterQuery function
func TestCreateMultiFilterQuery(t *testing.T) {
	t.Run("Create multi filter query with limit and offset", func(t *testing.T) {
		filters := []dbModels.QueryFilter{
			{Column: "status", Operator: "=", Value: "active"},
			{Column: "age", Operator: ">", Value: "18"},
		}
		limit := 20
		offset := 10

		query := services.CreateMultiFilterQuery(filters, &limit, &offset)

		assert.Len(t, query.Filters, 2)
		assert.Equal(t, "status", query.Filters[0].Column)
		assert.Equal(t, "age", query.Filters[1].Column)
		assert.NotNil(t, query.Limit)
		assert.Equal(t, 20, *query.Limit)
		assert.NotNil(t, query.Offset)
		assert.Equal(t, 10, *query.Offset)
	})

	t.Run("Create multi filter query without limit and offset", func(t *testing.T) {
		filters := []dbModels.QueryFilter{
			{Column: "status", Operator: "=", Value: "active"},
		}

		query := services.CreateMultiFilterQuery(filters, nil, nil)

		assert.Len(t, query.Filters, 1)
		assert.Nil(t, query.Limit)
		assert.Nil(t, query.Offset)
	})

	t.Run("Create query with empty filters", func(t *testing.T) {
		filters := []dbModels.QueryFilter{}
		limit := 10

		query := services.CreateMultiFilterQuery(filters, &limit, nil)

		assert.Len(t, query.Filters, 0)
		assert.NotNil(t, query.Limit)
		assert.Equal(t, 10, *query.Limit)
		assert.Nil(t, query.Offset)
	})
}

// TestMapToStructList tests the MapToStructList function
func TestMapToStructList(t *testing.T) {
	t.Run("Success - Map list to struct list", func(t *testing.T) {
		data := []map[string]interface{}{
			{
				"id":   "1",
				"name": "Test1",
				"age":  float64(25),
			},
			{
				"id":   "2",
				"name": "Test2",
				"age":  float64(35),
			},
		}

		results, err := services.MapToStructList[TestStruct](data)

		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "1", results[0].ID)
		assert.Equal(t, "Test1", results[0].Name)
		assert.Equal(t, 25, results[0].Age)
		assert.Equal(t, "2", results[1].ID)
		assert.Equal(t, "Test2", results[1].Name)
		assert.Equal(t, 35, results[1].Age)
	})

	t.Run("Success - Empty list", func(t *testing.T) {
		data := []map[string]interface{}{}

		results, err := services.MapToStructList[TestStruct](data)

		assert.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("Error - MapToStruct fails", func(t *testing.T) {
		data := []map[string]interface{}{
			{
				"id":   "1",
				"name": "Test1",
				"age":  float64(25),
			},
			{
				"invalid": make(chan int),
			},
		}

		results, err := services.MapToStructList[TestStruct](data)

		assert.Error(t, err)
		assert.Equal(t, app_errors.ErrMapToStruct, err)
		assert.Nil(t, results)
	})
}
