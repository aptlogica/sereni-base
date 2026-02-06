package scripts_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	"serenibase/internal/scripts"

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
	if args.Get(0) == nil {
		return dbModels.QueryParams{}, args.Error(1)
	}
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

func (m *MockTableService) GetByFunction(ctx context.Context, functionName string, fnArgs map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, functionName, fnArgs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

// Helper function to create a mock database service
func setupMockDB() (*pkg.DatabaseService, *MockTableService) {
	mockTable := &MockTableService{}
	db := &pkg.DatabaseService{
		TableService: mockTable,
	}
	return db, mockTable
}

// contains checks if a string contains a substring (case-insensitive)
// This is a local helper that mirrors the internal function for testing
func contains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// TestContainsHelper tests the local contains helper function
func TestContainsHelper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{"exact match", "hello", "hello", true},
		{"case insensitive upper in str", "HELLO", "hello", true},
		{"case insensitive upper in substr", "hello", "HELLO", true},
		{"partial match", "hello world", "world", true},
		{"partial match at start", "hello world", "hello", true},
		{"no match", "hello", "world", false},
		{"empty substr", "hello", "", true},
		{"empty str", "", "hello", false},
		{"both empty", "", "", true},
		{"mixed case match", "HeLLo WoRLd", "hello world", true},
		{"already exists error message", "relation already exists", "already exists", true},
		{"already exists uppercase", "RELATION ALREADY EXISTS", "already exists", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.str, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCreateMasterSchema tests the exported CreateMasterSchema function
func TestCreateMasterSchema(t *testing.T) {
	t.Run("successful full master schema creation", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		// Mock CreateSchema
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil)

		// Mock CreateTable for all table creations
		mockTable.On("CreateTable", mock.Anything).Return(nil)

		// Mock CreateFunction for all function creations
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		// Should not panic
		scripts.CreateMasterSchema(dbService)

		mockTable.AssertCalled(t, "CreateSchema", mock.Anything, mock.AnythingOfType("string"))
		mockTable.AssertCalled(t, "CreateTable", mock.Anything)
	})

	t.Run("master schema creation with schema error", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		// Mock CreateSchema with error
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("schema creation failed"))

		// Mock CreateTable - should still be called even if schema fails
		mockTable.On("CreateTable", mock.Anything).Return(nil)

		// Mock CreateFunction
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		// Should not panic even with schema error
		scripts.CreateMasterSchema(dbService)

		mockTable.AssertExpectations(t)
	})

	t.Run("master schema creation with already exists errors", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		// Mock CreateSchema with "already exists" error
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("schema already exists"))

		// Mock CreateTable with "already exists" error
		mockTable.On("CreateTable", mock.Anything).Return(errors.New("relation already exists"))

		// Mock CreateFunction with "already exists" error
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(errors.New("function already exists"))

		// Should not panic - "already exists" errors should be handled gracefully
		scripts.CreateMasterSchema(dbService)

		mockTable.AssertExpectations(t)
	})

	t.Run("master schema creation with table creation errors", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		// Mock CreateSchema - success
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil)

		// Mock CreateTable with various errors
		mockTable.On("CreateTable", mock.Anything).Return(errors.New("database connection failed"))

		// Mock CreateFunction
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		// Should not panic even with table errors
		scripts.CreateMasterSchema(dbService)

		mockTable.AssertExpectations(t)
	})

	t.Run("master schema creation with function creation errors", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		// Mock CreateSchema
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil)

		// Mock CreateTable
		mockTable.On("CreateTable", mock.Anything).Return(nil)

		// Mock CreateFunction with error
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(errors.New("syntax error in function"))

		// Should not panic even with function errors
		scripts.CreateMasterSchema(dbService)

		mockTable.AssertExpectations(t)
	})

	t.Run("master schema creation with mixed errors", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		// Mock CreateSchema with error
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("permission denied"))

		// Mock CreateTable - some succeed, some fail
		mockTable.On("CreateTable", mock.Anything).Return(errors.New("table creation failed")).Once()
		mockTable.On("CreateTable", mock.Anything).Return(nil)

		// Mock CreateFunction - some with "already exists"
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(errors.New("function already exists")).Once()
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		// Should not panic
		scripts.CreateMasterSchema(dbService)
	})

	t.Run("master schema creation verifies table count", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		// Track CreateTable calls
		tableCount := 0
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil)
		mockTable.On("CreateTable", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			tableCount++
		})
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		scripts.CreateMasterSchema(dbService)

		// Verify that multiple tables were created (at least the expected tables)
		assert.Greater(t, tableCount, 0, "Expected at least one table to be created")
	})
}

// TestCreateMasterSchemaEdgeCases tests edge cases
func TestCreateMasterSchemaEdgeCases(t *testing.T) {
	t.Run("nil error from CreateSchema", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil)
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		// Should not panic with nil error
		assert.NotPanics(t, func() {
			scripts.CreateMasterSchema(dbService)
		})
	})

	t.Run("empty error message handling", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(errors.New(""))
		mockTable.On("CreateTable", mock.Anything).Return(errors.New(""))
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(errors.New(""))

		// Should not panic with empty error messages
		assert.NotPanics(t, func() {
			scripts.CreateMasterSchema(dbService)
		})
	})

	t.Run("special characters in error message", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("error with 'special' chars & <symbols>"))
		mockTable.On("CreateTable", mock.Anything).Return(errors.New("relation \"test\" already exists"))
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		// Should not panic with special characters
		assert.NotPanics(t, func() {
			scripts.CreateMasterSchema(dbService)
		})
	})

	t.Run("unicode in error message", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("错误: 表已存在"))
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		// Should not panic with unicode error messages
		assert.NotPanics(t, func() {
			scripts.CreateMasterSchema(dbService)
		})
	})

	t.Run("very long error message", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		longError := strings.Repeat("error ", 1000)
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(errors.New(longError))
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		// Should not panic with very long error messages
		assert.NotPanics(t, func() {
			scripts.CreateMasterSchema(dbService)
		})
	})
}

// TestCreateMasterSchemaContextHandling tests context usage
func TestCreateMasterSchemaContextHandling(t *testing.T) {
	t.Run("context is passed to CreateSchema", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		var capturedCtx context.Context
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
		})
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		scripts.CreateMasterSchema(dbService)

		assert.NotNil(t, capturedCtx, "Context should be passed to CreateSchema")
	})

	t.Run("context is passed to CreateFunction", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		var capturedCtx context.Context
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil)
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
		})

		scripts.CreateMasterSchema(dbService)

		assert.NotNil(t, capturedCtx, "Context should be passed to CreateFunction")
	})
}

// TestCreateMasterSchemaSchemaName tests that the correct schema name is used
func TestCreateMasterSchemaSchemaName(t *testing.T) {
	t.Run("schema name is passed correctly", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		var capturedSchemaName string
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil).Run(func(args mock.Arguments) {
			capturedSchemaName = args.String(1)
		})
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		scripts.CreateMasterSchema(dbService)

		assert.NotEmpty(t, capturedSchemaName, "Schema name should not be empty")
	})
}

// TestCreateMasterSchemaTableRequests tests the table creation requests
func TestCreateMasterSchemaTableRequests(t *testing.T) {
	t.Run("table requests have names", func(t *testing.T) {
		dbService, mockTable := setupMockDB()

		tableNames := make([]string, 0)
		mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil)
		mockTable.On("CreateTable", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			req := args.Get(0).(dbModels.CreateTableRequest)
			tableNames = append(tableNames, req.Name)
		})
		mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		scripts.CreateMasterSchema(dbService)

		// Verify all tables have names
		for _, name := range tableNames {
			assert.NotEmpty(t, name, "Table name should not be empty")
		}
	})
}

// BenchmarkCreateMasterSchema benchmarks the CreateMasterSchema function
func BenchmarkCreateMasterSchema(b *testing.B) {
	dbService, mockTable := setupMockDB()

	mockTable.On("CreateSchema", mock.Anything, mock.AnythingOfType("string")).Return(nil)
	mockTable.On("CreateTable", mock.Anything).Return(nil)
	mockTable.On("CreateFunction", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scripts.CreateMasterSchema(dbService)
	}
}
