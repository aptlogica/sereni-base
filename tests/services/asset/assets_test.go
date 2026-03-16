package asset_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	services "github.com/aptlogica/sereni-base/internal/services/asset"

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

// MockBulkService is a mock implementation of BulkService
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

func setupMockDB() (*pkg.DatabaseService, *MockTableService, *MockBulkService) {
	mockTable := &MockTableService{}
	mockBulk := &MockBulkService{}

	db := &pkg.DatabaseService{
		TableService: mockTable,
		BulkService:  mockBulk,
	}

	return db, mockTable, mockBulk
}

func TestNewAssetsService(t *testing.T) {
	db, _, _ := setupMockDB()

	service := services.NewAssetsService(db)

	assert.NotNil(t, service, "NewAssetsService should return a non-nil service")
}

func TestAssetInsertion_Success(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New()
	now := time.Now()

	assetData := dto.AssetInsertion{
		ID:           assetID,
		Title:        "test.png",
		Url:          "https://example.com/test.png",
		ThumbnailUrl: "https://example.com/thumb.png",
		BasePath:     "/uploads/test.png",
		MimeType:     "image/png",
		Size:         1024,
		Height:       800,
		Width:        600,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	expectedReturn := map[string]interface{}{
		"id":                 assetID.String(),
		"title":              "test.png",
		"url":                "https://example.com/test.png",
		"thumbnail_url":      "https://example.com/thumb.png",
		"base_path":          "/uploads/test.png",
		"mime_type":          "image/png",
		"size":               int64(1024),
		"height":             800,
		"width":              600,
		"created_time":       now,
		"last_modified_time": now,
	}

	mockTable.On("CreateRecord", "\"test_schema\".assets", mock.Anything).Return(expectedReturn, nil)

	result, err := service.AssetInsertion(ctx, assetData, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, assetID.String(), result.ID.String())
	assert.Equal(t, "test.png", result.Title)
	mockTable.AssertExpectations(t)
}

func TestAssetInsertion_DatabaseError(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetData := dto.AssetInsertion{
		ID:    uuid.New(),
		Title: "test.png",
	}

	mockTable.On("CreateRecord", "\"test_schema\".assets", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.AssetInsertion(ctx, assetData, schemaName)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestAssetInsertion_MapStructError(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetData := dto.AssetInsertion{
		ID:    uuid.New(),
		Title: "test.png",
	}

	// Return invalid data that can't be mapped to struct
	invalidReturn := map[string]interface{}{
		"id": "invalid-uuid-format",
	}

	mockTable.On("CreateRecord", "\"test_schema\".assets", mock.Anything).Return(invalidReturn, nil)

	_, err := service.AssetInsertion(ctx, assetData, schemaName)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	mockTable.AssertExpectations(t)
}

func TestGetBulkAssets_Success(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	id1 := uuid.New().String()
	id2 := uuid.New().String()
	ids := []string{id1, id2}
	now := time.Now()

	mockData := []map[string]interface{}{
		{
			"id":                 id1,
			"title":              "asset1.png",
			"url":                "https://example.com/asset1.png",
			"thumbnail_url":      "https://example.com/thumb1.png",
			"base_path":          "/uploads/asset1.png",
			"mime_type":          "image/png",
			"size":               int64(1024),
			"height":             800,
			"width":              600,
			"created_time":       now,
			"last_modified_time": now,
		},
		{
			"id":                 id2,
			"title":              "asset2.png",
			"url":                "https://example.com/asset2.png",
			"thumbnail_url":      "https://example.com/thumb2.png",
			"base_path":          "/uploads/asset2.png",
			"mime_type":          "image/png",
			"size":               int64(2048),
			"height":             1024,
			"width":              768,
			"created_time":       now,
			"last_modified_time": now,
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".assets", mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 && params.Filters[0].Column == "id" && params.Filters[0].Operator == "in"
	})).Return(mockData, nil)

	result, err := service.GetBulkAssets(ctx, schemaName, ids)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "asset1.png", result[0].Title)
	assert.Equal(t, "asset2.png", result[1].Title)
	mockTable.AssertExpectations(t)
}

func TestGetBulkAssets_EmptyIDs(t *testing.T) {
	db, _, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	ids := []string{}

	result, err := service.GetBulkAssets(ctx, schemaName, ids)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetBulkAssets_DatabaseError(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	ids := []string{"id1"}

	mockTable.On("GetTableData", "\"test_schema\".assets", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.GetBulkAssets(ctx, schemaName, ids)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestAssetBulkInsertion_Success(t *testing.T) {
	db, _, mockBulk := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	now := time.Now()

	assetData := []dto.AssetInsertion{
		{
			ID:           uuid.New(),
			Title:        "asset1.png",
			Url:          "https://example.com/asset1.png",
			ThumbnailUrl: "https://example.com/thumb1.png",
			BasePath:     "/uploads/asset1.png",
			MimeType:     "image/png",
			Size:         1024,
			Height:       800,
			Width:        600,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.New(),
			Title:        "asset2.png",
			Url:          "https://example.com/asset2.png",
			ThumbnailUrl: "https://example.com/thumb2.png",
			BasePath:     "/uploads/asset2.png",
			MimeType:     "image/png",
			Size:         2048,
			Height:       1024,
			Width:        768,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	expectedReturn := []map[string]interface{}{
		{
			"id":                 assetData[0].ID.String(),
			"title":              "asset1.png",
			"url":                "https://example.com/asset1.png",
			"thumbnail_url":      "https://example.com/thumb1.png",
			"base_path":          "/uploads/asset1.png",
			"mime_type":          "image/png",
			"size":               int64(1024),
			"height":             800,
			"width":              600,
			"created_time":       now,
			"last_modified_time": now,
		},
		{
			"id":                 assetData[1].ID.String(),
			"title":              "asset2.png",
			"url":                "https://example.com/asset2.png",
			"thumbnail_url":      "https://example.com/thumb2.png",
			"base_path":          "/uploads/asset2.png",
			"mime_type":          "image/png",
			"size":               int64(2048),
			"height":             1024,
			"width":              768,
			"created_time":       now,
			"last_modified_time": now,
		},
	}

	mockBulk.On("BulkInsert", "\"test_schema\".assets", mock.MatchedBy(func(records []map[string]interface{}) bool {
		return len(records) == 2
	})).Return(expectedReturn, nil)

	result, err := service.AssetBulkInsertion(ctx, assetData, schemaName)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "asset1.png", result[0].Title)
	assert.Equal(t, "asset2.png", result[1].Title)
	mockBulk.AssertExpectations(t)
}

func TestAssetBulkInsertion_DatabaseError(t *testing.T) {
	db, _, mockBulk := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetData := []dto.AssetInsertion{
		{
			ID:    uuid.New(),
			Title: "asset1.png",
		},
	}

	mockBulk.On("BulkInsert", "\"test_schema\".assets", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.AssetBulkInsertion(ctx, assetData, schemaName)

	assert.Error(t, err)
	mockBulk.AssertExpectations(t)
}

func TestAssetUpdate_Success(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()
	now := time.Now()
	newTitle := "updated.png"

	assetData := dto.AssetUpdate{
		Title:     &newTitle,
		UpdatedAt: now,
	}

	expectedReturn := map[string]interface{}{
		"id":                 assetID,
		"title":              "updated.png",
		"url":                "https://example.com/test.png",
		"thumbnail_url":      "https://example.com/thumb.png",
		"base_path":          "/uploads/test.png",
		"mime_type":          "image/png",
		"size":               int64(1024),
		"height":             800,
		"width":              600,
		"created_time":       now,
		"last_modified_time": now,
	}

	mockTable.On("UpdateRecord", "\"test_schema\".assets", assetID, mock.MatchedBy(func(data map[string]interface{}) bool {
		return data["title"] == &newTitle
	})).Return(expectedReturn, nil)

	result, err := service.AssetUpdate(ctx, assetID, assetData, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, "updated.png", result.Title)
	mockTable.AssertExpectations(t)
}

func TestAssetUpdate_EmptyPayload(t *testing.T) {
	db, _, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	assetData := dto.AssetUpdate{}

	_, err := service.AssetUpdate(ctx, assetID, assetData, schemaName)

	assert.Error(t, err)
	assert.Equal(t, app_errors.InvalidPayload, err)
}

func TestAssetUpdate_DatabaseError(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()
	newTitle := "updated.png"

	assetData := dto.AssetUpdate{
		Title:     &newTitle,
		UpdatedAt: time.Now(),
	}

	mockTable.On("UpdateRecord", "\"test_schema\".assets", assetID, mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.AssetUpdate(ctx, assetID, assetData, schemaName)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestGetAssetByID_Success(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()
	now := time.Now()

	mockData := []map[string]interface{}{
		{
			"id":                 assetID,
			"title":              "test.png",
			"url":                "https://example.com/test.png",
			"thumbnail_url":      "https://example.com/thumb.png",
			"base_path":          "/uploads/test.png",
			"mime_type":          "image/png",
			"size":               int64(1024),
			"height":             800,
			"width":              600,
			"created_time":       now,
			"last_modified_time": now,
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".assets", mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "id" &&
			params.Filters[0].Operator == "eq" &&
			params.Filters[0].Value == assetID
	})).Return(mockData, nil)

	result, err := service.GetAssetByID(ctx, assetID, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, assetID, result.ID.String())
	assert.Equal(t, "test.png", result.Title)
	mockTable.AssertExpectations(t)
}

func TestGetAssetByID_NotFound(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	mockTable.On("GetTableData", "\"test_schema\".assets", mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.GetAssetByID(ctx, assetID, schemaName)

	assert.Error(t, err)
	assert.Equal(t, app_errors.InvalidPayload, err)
	mockTable.AssertExpectations(t)
}

func TestGetAssetByID_DatabaseError(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	mockTable.On("GetTableData", "\"test_schema\".assets", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.GetAssetByID(ctx, assetID, schemaName)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteAsset_Success(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	mockTable.On("DeleteRecord", "\"test_schema\".assets", assetID).Return(nil)

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteAsset_DatabaseError(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	mockTable.On("DeleteRecord", "\"test_schema\".assets", assetID).Return(errors.New("database error"))

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestGetAssetByURL_Success(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	url := "https://example.com/test.png"
	now := time.Now()

	mockData := []map[string]interface{}{
		{
			"id":                 uuid.New().String(),
			"title":              "test.png",
			"url":                url,
			"thumbnail_url":      "https://example.com/thumb.png",
			"base_path":          "/uploads/test.png",
			"mime_type":          "image/png",
			"size":               int64(1024),
			"height":             800,
			"width":              600,
			"created_time":       now,
			"last_modified_time": now,
		},
	}

	mockTable.On("GetTableData", "\"test_schema\".assets", mock.MatchedBy(func(params dbModels.QueryParams) bool {
		return len(params.Filters) == 1 &&
			params.Filters[0].Column == "url" &&
			params.Filters[0].Operator == "eq" &&
			params.Filters[0].Value == url
	})).Return(mockData, nil)

	result, err := service.GetAssetByURL(ctx, url, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, url, result.Url)
	assert.Equal(t, "test.png", result.Title)
	mockTable.AssertExpectations(t)
}

func TestGetAssetByURL_NotFound(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	url := "https://example.com/notfound.png"

	mockTable.On("GetTableData", "\"test_schema\".assets", mock.Anything).Return([]map[string]interface{}{}, nil)

	_, err := service.GetAssetByURL(ctx, url, schemaName)

	assert.Error(t, err)
	assert.Equal(t, app_errors.InvalidPayload, err)
	mockTable.AssertExpectations(t)
}

func TestGetAssetByURL_DatabaseError(t *testing.T) {
	db, mockTable, _ := setupMockDB()
	service := services.NewAssetsService(db)

	ctx := context.Background()
	schemaName := "test_schema"
	url := "https://example.com/test.png"

	mockTable.On("GetTableData", "\"test_schema\".assets", mock.Anything).Return(nil, errors.New("database error"))

	_, err := service.GetAssetByURL(ctx, url, schemaName)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}
