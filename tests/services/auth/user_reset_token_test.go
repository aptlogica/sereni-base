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

// MockTableServiceResetToken is a mock implementation of TableService
type MockTableServiceResetToken struct {
	mock.Mock
}

func (m *MockTableServiceResetToken) GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
	args := m.Called(tableName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceResetToken) CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceResetToken) UpdateRecord(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, id, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceResetToken) DeleteRecord(tableName string, id interface{}) error {
	args := m.Called(tableName, id)
	return args.Error(0)
}

func (m *MockTableServiceResetToken) GetTables(schema string) ([]dbModels.Table, error) {
	args := m.Called(schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbModels.Table), args.Error(1)
}

func (m *MockTableServiceResetToken) CreateTable(req dbModels.CreateTableRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockTableServiceResetToken) AddColumn(tableName string, req dbModels.AddColumnRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableServiceResetToken) AlterTable(tableName string, req dbModels.AlterTableRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableServiceResetToken) BuildComplexQuery(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error) {
	args := m.Called(tableName, filters)
	return args.Get(0).(dbModels.QueryParams), args.Error(1)
}

func (m *MockTableServiceResetToken) CreateSchema(ctx context.Context, schemaName string) error {
	args := m.Called(ctx, schemaName)
	return args.Error(0)
}

func (m *MockTableServiceResetToken) DropTable(ctx context.Context, tableName string) error {
	args := m.Called(ctx, tableName)
	return args.Error(0)
}

func (m *MockTableServiceResetToken) CreateView(ctx context.Context, viewName string, viewSQL string) error {
	args := m.Called(ctx, viewName, viewSQL)
	return args.Error(0)
}

func (m *MockTableServiceResetToken) CreateFunction(ctx context.Context, functionName string, functionSQL string) error {
	args := m.Called(ctx, functionName, functionSQL)
	return args.Error(0)
}

func (m *MockTableServiceResetToken) GetByFunction(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error) {
	mockArgs := m.Called(ctx, functionName, args)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).([]map[string]interface{}), mockArgs.Error(1)
}

func setupMockDBResetToken() *pkg.DatabaseService {
	mockTable := &MockTableServiceResetToken{}

	return &pkg.DatabaseService{
		TableService: mockTable,
	}
}

func TestNewUserResetTokenService(t *testing.T) {
	db := setupMockDBResetToken()

	service := services.NewUserResetTokenService(db)

	assert.NotNil(t, service, "NewUserResetTokenService should return a non-nil service")
}

func TestCreateUserResetToken_Success_NoExisting(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()
	token := "reset-token-123"
	tokenID := uuid.New().String()
	now := time.Now()

	req := dto.UserResetTokenInsertion{
		UserID: userID,
		Token:  token,
	}

	// Mock GetTableData to return no existing records
	mockTable.On("GetTableData", mock.Anything, mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "user_id" &&
			params.Filters[0].Value == userID
	})).Return([]map[string]interface{}{}, nil)

	// Mock CreateRecord
	mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(map[string]interface{}{
		"id":           tokenID,
		"user_id":      userID,
		"token":        token,
		"created_time": now,
	}, nil)

	result, err := service.CreateUserResetToken(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, token, result.Token)
	assert.Equal(t, userID, result.UserID.String())
	mockTable.AssertExpectations(t)
}

func TestCreateUserResetToken_Success_WithExisting(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()
	token := "new-reset-token"
	existingID := uuid.New().String()
	newID := uuid.New().String()
	now := time.Now()

	req := dto.UserResetTokenInsertion{
		UserID: userID,
		Token:  token,
	}

	// Mock GetTableData to return existing record
	mockTable.On("GetTableData", mock.Anything, mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "user_id" &&
			params.Filters[0].Value == userID
	})).Return([]map[string]interface{}{
		{
			"id":      existingID,
			"user_id": userID,
			"token":   "old-token",
		},
	}, nil)

	// Mock DeleteRecord for existing token
	mockTable.On("DeleteRecord", mock.Anything, existingID).Return(nil)

	// Mock CreateRecord for new token
	mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(map[string]interface{}{
		"id":           newID,
		"user_id":      userID,
		"token":        token,
		"created_time": now,
	}, nil)

	result, err := service.CreateUserResetToken(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, token, result.Token)
	mockTable.AssertExpectations(t)
}

func TestCreateUserResetToken_GetExisting_DatabaseError(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()

	req := dto.UserResetTokenInsertion{
		UserID: userID,
		Token:  "token",
	}

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.CreateUserResetToken(ctx, req)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestCreateUserResetToken_DeleteExisting_MissingID(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()

	req := dto.UserResetTokenInsertion{
		UserID: userID,
		Token:  "token",
	}

	// Mock GetTableData to return existing record without ID
	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"user_id": userID,
			"token":   "old-token",
			// No "id" field
		},
	}, nil)

	_, err := service.CreateUserResetToken(ctx, req)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestCreateUserResetToken_DeleteExisting_DatabaseError(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()
	existingID := uuid.New().String()

	req := dto.UserResetTokenInsertion{
		UserID: userID,
		Token:  "token",
	}

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"id":      existingID,
			"user_id": userID,
			"token":   "old-token",
		},
	}, nil)

	mockTable.On("DeleteRecord", mock.Anything, existingID).Return(errors.New("delete error"))

	_, err := service.CreateUserResetToken(ctx, req)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestCreateUserResetToken_CreateRecord_DatabaseError(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()

	req := dto.UserResetTokenInsertion{
		UserID: userID,
		Token:  "token",
	}

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
	mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(nil, errors.New("create error"))

	_, err := service.CreateUserResetToken(ctx, req)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestCreateUserResetToken_MapStructError(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()

	req := dto.UserResetTokenInsertion{
		UserID: userID,
		Token:  "token",
	}

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
	mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(map[string]interface{}{
		"id": "invalid-uuid",
	}, nil)

	_, err := service.CreateUserResetToken(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserResetToken_Success(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	token := "reset-token-123"
	tokenID := uuid.New().String()
	userID := uuid.New().String()
	now := time.Now()

	mockTable.On("GetTableData", mock.Anything, mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "token" &&
			params.Filters[0].Value == token
	})).Return([]map[string]interface{}{
		{
			"id":           tokenID,
			"user_id":      userID,
			"token":        token,
			"created_time": now,
		},
	}, nil)

	result, err := service.GetUserResetToken(ctx, token)

	assert.NoError(t, err)
	assert.Equal(t, token, result.Token)
	assert.Equal(t, userID, result.UserID.String())
	mockTable.AssertExpectations(t)
}

func TestGetUserResetToken_NotFound(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	token := "nonexistent-token"

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.GetUserResetToken(ctx, token)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrRecordNotFound, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserResetToken_DatabaseError(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	token := "reset-token"

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.GetUserResetToken(ctx, token)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserResetToken_MapStructError(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	token := "reset-token"

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"id": "invalid-uuid",
		},
	}, nil)

	_, err := service.GetUserResetToken(ctx, token)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteTokensByUserId_Success_WithRecords(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()
	token1ID := uuid.New().String()
	token2ID := uuid.New().String()

	// Mock GetTableData to return multiple records
	mockTable.On("GetTableData", mock.Anything, mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "user_id" &&
			params.Filters[0].Value == userID
	})).Return([]map[string]interface{}{
		{
			"id":      token1ID,
			"user_id": userID,
			"token":   "token1",
		},
		{
			"id":      token2ID,
			"user_id": userID,
			"token":   "token2",
		},
	}, nil)

	// Mock DeleteRecord for each token
	mockTable.On("DeleteRecord", mock.Anything, token1ID).Return(nil)
	mockTable.On("DeleteRecord", mock.Anything, token2ID).Return(nil)

	err := service.DeleteTokensByUserId(ctx, userID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteTokensByUserId_Success_NoRecords(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	err := service.DeleteTokensByUserId(ctx, userID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteTokensByUserId_GetRecords_DatabaseError(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	err := service.DeleteTokensByUserId(ctx, userID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteTokensByUserId_MissingID(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"user_id": userID,
			"token":   "token",
			// No "id" field
		},
	}, nil)

	err := service.DeleteTokensByUserId(ctx, userID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteTokensByUserId_DeleteRecord_DatabaseError(t *testing.T) {
	db := setupMockDBResetToken()
	mockTable := db.TableService.(*MockTableServiceResetToken)
	service := services.NewUserResetTokenService(db)

	ctx := context.Background()
	userID := uuid.New().String()
	tokenID := uuid.New().String()

	mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{
		{
			"id":      tokenID,
			"user_id": userID,
			"token":   "token",
		},
	}, nil)

	mockTable.On("DeleteRecord", mock.Anything, tokenID).Return(errors.New("delete error"))

	err := service.DeleteTokensByUserId(ctx, userID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}
