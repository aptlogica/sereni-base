package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	services "serenibase/internal/services/auth"

	"github.com/google/uuid"
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

func setupMockDB() *pkg.DatabaseService {
	mockTable := &MockTableService{}

	return &pkg.DatabaseService{
		TableService: mockTable,
	}
}

func TestNewUserService(t *testing.T) {
	db := setupMockDB()

	service := services.NewUserService(db)

	assert.NotNil(t, service, "NewUserService should return a non-nil service")
}

func TestCreateUser_Success(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	now := time.Now()
	userID := uuid.New()

	req := dto.RegisterRequest{
		Email:         "test@example.com",
		Password:      "hashedpassword",
		FirstName:     "John",
		LastName:      "Doe",
		EmailVerified: false,
		Status:        "pending",
	}

	expectedReturn := map[string]interface{}{
		"id":                 userID.String(),
		"email":              "test@example.com",
		"password":           "hashedpassword",
		"first_name":         "John",
		"last_name":          "Doe",
		"display_name":       "John Doe",
		"email_verified":     false,
		"status":             "pending",
		"created_time":       now,
		"last_modified_time": now,
	}

	mockTable.On("CreateRecord", "\"test_schema\".users", mock.Anything).Return(expectedReturn, nil)

	result, err := service.CreateUser(ctx, schema, req)

	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "John", result.FirstName)
	assert.Equal(t, "Doe", result.LastName)
	mockTable.AssertExpectations(t)
}

func TestCreateUser_DatabaseError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"

	req := dto.RegisterRequest{
		Email:     "test@example.com",
		Password:  "hashedpassword",
		FirstName: "John",
		LastName:  "Doe",
	}

	mockTable.On("CreateRecord", "\"test_schema\".users", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.CreateUser(ctx, schema, req)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestCreateUser_MapStructError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"

	req := dto.RegisterRequest{
		Email:     "test@example.com",
		Password:  "hashedpassword",
		FirstName: "John",
		LastName:  "Doe",
	}

	invalidReturn := map[string]interface{}{
		"id": "invalid-uuid",
	}

	mockTable.On("CreateRecord", "\"test_schema\".users", mock.Anything).Return(invalidReturn, nil)

	_, err := service.CreateUser(ctx, schema, req)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserByEmail_Success(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	email := "test@example.com"
	userID := uuid.New().String()
	now := time.Now()

	mockData := []map[string]interface{}{
		{
			"id":                 userID,
			"email":              email,
			"password":           "hashedpassword",
			"first_name":         "John",
			"last_name":          "Doe",
			"display_name":       "John Doe",
			"email_verified":     true,
			"status":             "active",
			"created_time":       now,
			"last_modified_time": now,
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "email" &&
			params.Filters[0].Operator == "eq" &&
			params.Filters[0].Value == email
	})).Return(mockData, nil)

	result, err := service.GetUserByEmail(ctx, schema, email)

	assert.NoError(t, err)
	assert.Equal(t, email, result.Email)
	assert.Equal(t, "John", result.FirstName)
	mockTable.AssertExpectations(t)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	email := "notfound@example.com"

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.GetUserByEmail(ctx, schema, email)

	assert.Error(t, err)
	assert.Equal(t, app_errors.UserNotFound, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserByEmail_DatabaseError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	email := "test@example.com"

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.GetUserByEmail(ctx, schema, email)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserByID_Success(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()
	now := time.Now()

	mockData := []map[string]interface{}{
		{
			"id":                 userID,
			"email":              "test@example.com",
			"password":           "hashedpassword",
			"first_name":         "John",
			"last_name":          "Doe",
			"display_name":       "John Doe",
			"email_verified":     true,
			"status":             "active",
			"created_time":       now,
			"last_modified_time": now,
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "id" &&
			params.Filters[0].Operator == "eq" &&
			params.Filters[0].Value == userID
	})).Return(mockData, nil)

	result, err := service.GetUserByID(ctx, schema, userID)

	assert.NoError(t, err)
	assert.Equal(t, userID, result.ID.String())
	assert.Equal(t, "test@example.com", result.Email)
	mockTable.AssertExpectations(t)
}

func TestGetUserByID_NotFound(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.GetUserByID(ctx, schema, userID)

	assert.Error(t, err)
	assert.Equal(t, app_errors.UserNotFound, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserByID_DatabaseError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.GetUserByID(ctx, schema, userID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestUpdateUser_Success(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()
	now := time.Now()

	updateData := map[string]interface{}{
		"first_name": "Jane",
		"last_name":  "Smith",
	}

	expectedReturn := map[string]interface{}{
		"id":                 userID,
		"email":              "test@example.com",
		"password":           "hashedpassword",
		"first_name":         "Jane",
		"last_name":          "Smith",
		"display_name":       "Jane Smith",
		"email_verified":     true,
		"status":             "active",
		"created_time":       now,
		"last_modified_time": now,
	}

	mockTable.On("UpdateRecord", "\"test_schema\".users", userID, updateData).Return(expectedReturn, nil)

	result, err := service.UpdateUser(ctx, schema, userID, updateData)

	assert.NoError(t, err)
	assert.Equal(t, "Jane", result.FirstName)
	assert.Equal(t, "Smith", result.LastName)
	mockTable.AssertExpectations(t)
}

func TestUpdateUser_DatabaseError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()

	updateData := map[string]interface{}{
		"first_name": "Jane",
	}

	mockTable.On("UpdateRecord", "\"test_schema\".users", userID, updateData).Return(nil, errors.New("database error"))

	_, err := service.UpdateUser(ctx, schema, userID, updateData)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestUpdateUser_MapStructError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()

	updateData := map[string]interface{}{
		"first_name": "Jane",
	}

	invalidReturn := map[string]interface{}{
		"id": "invalid-uuid",
	}

	mockTable.On("UpdateRecord", "\"test_schema\".users", userID, updateData).Return(invalidReturn, nil)

	_, err := service.UpdateUser(ctx, schema, userID, updateData)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}

func TestGetAllUsers_Success(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	now := time.Now()

	mockData := []map[string]interface{}{
		{
			"id":                 uuid.New().String(),
			"email":              "user1@example.com",
			"first_name":         "User",
			"last_name":          "One",
			"email_verified":     true,
			"status":             "active",
			"is_deleted":         false,
			"created_time":       now,
			"last_modified_time": now,
		},
		{
			"id":                 uuid.New().String(),
			"email":              "user2@example.com",
			"first_name":         "User",
			"last_name":          "Two",
			"email_verified":     true,
			"status":             "active",
			"is_deleted":         false,
			"created_time":       now,
			"last_modified_time": now,
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "is_deleted" &&
			params.Filters[0].Value == false
	})).Return(mockData, nil)

	result, err := service.GetAllUsers(ctx, schema)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "user1@example.com", result[0].Email)
	assert.Equal(t, "user2@example.com", result[1].Email)
	mockTable.AssertExpectations(t)
}

func TestGetAllUsers_Empty(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return([]map[string]interface{}{}, nil)

	result, err := service.GetAllUsers(ctx, schema)

	assert.NoError(t, err)
	assert.Empty(t, result)
	mockTable.AssertExpectations(t)
}

func TestGetAllUsers_DatabaseError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.GetAllUsers(ctx, schema)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestGetBulkUsers_Success(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	id1 := uuid.New().String()
	id2 := uuid.New().String()
	ids := []string{id1, id2}
	now := time.Now()

	mockData := []map[string]interface{}{
		{
			"id":                 id1,
			"email":              "user1@example.com",
			"first_name":         "User",
			"last_name":          "One",
			"email_verified":     true,
			"status":             "active",
			"created_time":       now,
			"last_modified_time": now,
		},
		{
			"id":                 id2,
			"email":              "user2@example.com",
			"first_name":         "User",
			"last_name":          "Two",
			"email_verified":     true,
			"status":             "active",
			"created_time":       now,
			"last_modified_time": now,
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "id" &&
			params.Filters[0].Operator == "in"
	})).Return(mockData, nil)

	result, err := service.GetBulkUsers(ctx, schema, ids)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "user1@example.com", result[0].Email)
	assert.Equal(t, "user2@example.com", result[1].Email)
	mockTable.AssertExpectations(t)
}

func TestGetBulkUsers_EmptyIDs(t *testing.T) {
	db := setupMockDB()
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	ids := []string{}

	result, err := service.GetBulkUsers(ctx, schema, ids)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetBulkUsers_Empty(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	ids := []string{uuid.New().String()}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return([]map[string]interface{}{}, nil)

	result, err := service.GetBulkUsers(ctx, schema, ids)

	assert.NoError(t, err)
	assert.Empty(t, result)
	mockTable.AssertExpectations(t)
}

func TestGetBulkUsers_DatabaseError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	ids := []string{uuid.New().String()}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.GetBulkUsers(ctx, schema, ids)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()

	mockTable.On("DeleteRecord", "\"test_schema\".users", userID).Return(nil)

	err := service.DeleteUser(ctx, schema, userID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteUser_DatabaseError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()

	mockTable.On("DeleteRecord", "\"test_schema\".users", userID).Return(errors.New("database error"))

	err := service.DeleteUser(ctx, schema, userID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

// Additional tests for missing coverage

func TestCreateUser_WithCustomID(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	now := time.Now()
	customID := uuid.New()

	req := dto.RegisterRequest{
		ID:            customID,
		Email:         "test@example.com",
		Password:      "hashedpassword",
		FirstName:     "John",
		LastName:      "Doe",
		EmailVerified: true,
		Status:        "active",
		AuthProvider:  "google",
	}

	expectedReturn := map[string]interface{}{
		"id":                 customID.String(),
		"email":              "test@example.com",
		"password":           "hashedpassword",
		"first_name":         "John",
		"last_name":          "Doe",
		"display_name":       "John Doe",
		"email_verified":     true,
		"status":             "active",
		"auth_provider":      "google",
		"created_time":       now,
		"last_modified_time": now,
	}

	mockTable.On("CreateRecord", "\"test_schema\".users", mock.Anything).Return(expectedReturn, nil)

	result, err := service.CreateUser(ctx, schema, req)

	assert.NoError(t, err)
	assert.Equal(t, customID, result.ID)
	assert.Equal(t, "test@example.com", result.Email)
	mockTable.AssertExpectations(t)
}

func TestGetUserByEmail_MapStructError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	email := "test@example.com"

	invalidData := []map[string]interface{}{
		{
			"id": "invalid-uuid",
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return(invalidData, nil)

	_, err := service.GetUserByEmail(ctx, schema, email)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserByID_MapStructError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()

	invalidData := []map[string]interface{}{
		{
			"id": "invalid-uuid",
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return(invalidData, nil)

	_, err := service.GetUserByID(ctx, schema, userID)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}

func TestUpdateUser_WithDateOfBirth(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()
	now := time.Now()

	updateData := map[string]interface{}{
		"first_name":  "Jane",
		"DateOfBirth": "1990-05-15",
	}

	expectedReturn := map[string]interface{}{
		"id":                 userID,
		"email":              "test@example.com",
		"first_name":         "Jane",
		"last_name":          "Doe",
		"date_of_birth":      "1990-05-15",
		"created_time":       now,
		"last_modified_time": now,
	}

	mockTable.On("UpdateRecord", "\"test_schema\".users", userID, mock.MatchedBy(func(data map[string]interface{}) bool {
		// Should have date_of_birth after conversion
		_, hasDateOfBirth := data["date_of_birth"]
		_, hasDateOfBirthCamel := data["DateOfBirth"]
		return hasDateOfBirth && !hasDateOfBirthCamel
	})).Return(expectedReturn, nil)

	result, err := service.UpdateUser(ctx, schema, userID, updateData)

	assert.NoError(t, err)
	assert.Equal(t, "Jane", result.FirstName)
	mockTable.AssertExpectations(t)
}

func TestUpdateUser_InvalidDateOfBirth(t *testing.T) {
	db := setupMockDB()
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	userID := uuid.New().String()

	updateData := map[string]interface{}{
		"first_name":  "Jane",
		"DateOfBirth": "invalid-date",
	}

	_, err := service.UpdateUser(ctx, schema, userID, updateData)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dob format")
}

func TestGetAllUsers_MapStructError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"

	invalidData := []map[string]interface{}{
		{
			"id": "valid-uuid-" + uuid.New().String(),
		},
		{
			"id": "invalid-uuid",
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return(invalidData, nil)

	_, err := service.GetAllUsers(ctx, schema)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}

func TestGetBulkUsers_MapStructError(t *testing.T) {
	db := setupMockDB()
	mockTable := db.TableService.(*MockTableService)
	service := services.NewUserService(db)

	ctx := context.Background()
	schema := "test_schema"
	ids := []string{uuid.New().String()}

	invalidData := []map[string]interface{}{
		{
			"id": "invalid-uuid",
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".users", mock.Anything).Return(invalidData, nil)

	_, err := service.GetBulkUsers(ctx, schema, ids)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}
