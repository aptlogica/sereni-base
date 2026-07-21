package table_test

import (
	"context"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	services "github.com/aptlogica/sereni-base/internal/services/table"

	"github.com/stretchr/testify/mock"
)

// MockModelService implements interfaces.ModelService
// Uses testify/mock for behavior setup.
type MockModelService struct{ mock.Mock }

func (m *MockModelService) Create(ctx context.Context, tableData dto.ModelInsertion, schemaName string) (tenant.Model, error) {
	args := m.Called(ctx, tableData, schemaName)
	return args.Get(0).(tenant.Model), args.Error(1)
}
func (m *MockModelService) GetModelByID(ctx context.Context, schemaName string, id string) (tenant.Model, error) {
	args := m.Called(ctx, schemaName, id)
	return args.Get(0).(tenant.Model), args.Error(1)
}
func (m *MockModelService) GetAllModels(ctx context.Context, schemaName string) ([]tenant.Model, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).([]tenant.Model), args.Error(1)
}
func (m *MockModelService) Update(ctx context.Context, schemaName string, id string, req dto.UpdateModelRequest) (tenant.Model, error) {
	args := m.Called(ctx, schemaName, id, req)
	return args.Get(0).(tenant.Model), args.Error(1)
}
func (m *MockModelService) DeleteModels(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}
func (m *MockModelService) GetModelByBaseID(ctx context.Context, schemaName string, base_id string) ([]tenant.Model, error) {
	args := m.Called(ctx, schemaName, base_id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Model), args.Error(1)
}
func (m *MockModelService) GetModelByWorkspaceID(ctx context.Context, schemaName string, workspace_id string) ([]tenant.Model, error) {
	args := m.Called(ctx, schemaName, workspace_id)
	return args.Get(0).([]tenant.Model), args.Error(1)
}
func (m *MockModelService) DeleteModel(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}

// MockColumnService implements interfaces.ColumnService
// Uses testify/mock for behavior setup.
type MockColumnService struct{ mock.Mock }

func (m *MockColumnService) Create(ctx context.Context, req dto.ColumnInsertion, schemaName string) (tenant.Column, error) {
	args := m.Called(ctx, req, schemaName)
	return args.Get(0).(tenant.Column), args.Error(1)
}
func (m *MockColumnService) GetColumnByID(ctx context.Context, schemaName string, id string) (tenant.Column, error) {
	args := m.Called(ctx, schemaName, id)
	return args.Get(0).(tenant.Column), args.Error(1)
}
func (m *MockColumnService) GetColumnByModelID(ctx context.Context, schemaName, modelID string) ([]tenant.Column, error) {
	args := m.Called(ctx, schemaName, modelID)
	return args.Get(0).([]tenant.Column), args.Error(1)
}
func (m *MockColumnService) GetAllColumns(ctx context.Context, schemaName string) ([]tenant.Column, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).([]tenant.Column), args.Error(1)
}
func (m *MockColumnService) UpdateColumn(ctx context.Context, schemaName string, id string, req dto.ColumnUpdate) (tenant.Column, error) {
	args := m.Called(ctx, schemaName, id, req)
	return args.Get(0).(tenant.Column), args.Error(1)
}
func (m *MockColumnService) DeleteColumn(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}
func (m *MockColumnService) BulkInsert(reqs []dto.ColumnInsertion, schemaName string) ([]tenant.Column, error) {
	args := m.Called(reqs, schemaName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Column), args.Error(1)
}
func (m *MockColumnService) GetMaxOrderIndexOfColumn(ctx context.Context, schemaName string, modelId string) (float64, error) {
	args := m.Called(ctx, schemaName, modelId)
	return args.Get(0).(float64), args.Error(1)
}
func (m *MockColumnService) BulkUpdate(ctx context.Context, schemaName string, tableName string, columnName string, updates []dto.UpdateColumnsRequest) error {
	args := m.Called(ctx, schemaName, tableName, columnName, updates)
	return args.Error(0)
}
func (m *MockColumnService) ResetColumn(ctx context.Context, schemaName string, tableName string, columnName string) error {
	args := m.Called(ctx, schemaName, tableName, columnName)
	return args.Error(0)
}

func (m *MockColumnService) BulkUpdateByColumns(ctx context.Context, schemaName string, tableName string, updates []dto.UpdateColumnValueRequest) error {
	args := m.Called(ctx, schemaName, tableName, updates)
	return args.Error(0)
}

// MockViewService implements interfaces.ViewService
type MockViewService struct{ mock.Mock }

func (m *MockViewService) Create(ctx context.Context, req dto.ViewInsertion, schemaName string) (tenant.View, error) {
	args := m.Called(ctx, req, schemaName)
	return args.Get(0).(tenant.View), args.Error(1)
}
func (m *MockViewService) GetViewByID(ctx context.Context, schemaName, id string) (tenant.View, error) {
	args := m.Called(ctx, schemaName, id)
	return args.Get(0).(tenant.View), args.Error(1)
}
func (m *MockViewService) GetAllViews(ctx context.Context, schemaName string) ([]tenant.View, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).([]tenant.View), args.Error(1)
}
func (m *MockViewService) GetViewsByModelID(ctx context.Context, schemaName string, modelID string) ([]tenant.View, error) {
	args := m.Called(ctx, schemaName, modelID)
	return args.Get(0).([]tenant.View), args.Error(1)
}
func (m *MockViewService) UpdateView(ctx context.Context, schemaName, id string, req dto.ViewUpdate) (tenant.View, error) {
	args := m.Called(ctx, schemaName, id, req)
	return args.Get(0).(tenant.View), args.Error(1)
}
func (m *MockViewService) DeleteView(ctx context.Context, schemaName, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}

// MockRelationshipService implements interfaces.RelationshipService
type MockRelationshipService struct{ mock.Mock }

func (m *MockRelationshipService) Create(ctx context.Context, req dto.RelationInsertion, schemaName string) (tenant.Relation, error) {
	args := m.Called(ctx, req, schemaName)
	return args.Get(0).(tenant.Relation), args.Error(1)
}
func (m *MockRelationshipService) GetRelationByID(ctx context.Context, id string, schemaName string) (tenant.Relation, error) {
	args := m.Called(ctx, id, schemaName)
	return args.Get(0).(tenant.Relation), args.Error(1)
}
func (m *MockRelationshipService) DeleteRelation(ctx context.Context, relationId string, schemaName string) error {
	args := m.Called(ctx, relationId, schemaName)
	return args.Error(0)
}
func (m *MockRelationshipService) UpdateRelation(ctx context.Context, relationId string, relationData dto.RelationUpdate, schemaName string) (tenant.Relation, error) {
	args := m.Called(ctx, relationId, relationData, schemaName)
	return args.Get(0).(tenant.Relation), args.Error(1)
}

// MockAssetManagementService implements interfaces.AssetManagementService
type MockAssetManagementService struct{ mock.Mock }

func (m *MockAssetManagementService) Upload(ctx context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error) {
	args := m.Called(ctx, req, schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Assets), args.Error(1)
}
func (m *MockAssetManagementService) UploadImage(ctx context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error) {
	args := m.Called(ctx, req, schema)
	return args.Get(0).([]tenant.Assets), args.Error(1)
}
func (m *MockAssetManagementService) GetBulkAssets(ctx context.Context, schemaName string, ids []string) ([]tenant.Assets, error) {
	args := m.Called(ctx, schemaName, ids)
	return args.Get(0).([]tenant.Assets), args.Error(1)
}
func (m *MockAssetManagementService) UpdateAsset(ctx context.Context, assetId string, assetData dto.AssetUpdate, schemaName string) (tenant.Assets, error) {
	args := m.Called(ctx, assetId, assetData, schemaName)
	return args.Get(0).(tenant.Assets), args.Error(1)
}
func (m *MockAssetManagementService) DeleteAsset(ctx context.Context, assetId string, schemaName string) error {
	args := m.Called(ctx, assetId, schemaName)
	return args.Error(0)
}
func (m *MockAssetManagementService) GetAssetByURL(ctx context.Context, schemaName string, url string) (tenant.Assets, error) {
	args := m.Called(ctx, schemaName, url)
	return args.Get(0).(tenant.Assets), args.Error(1)
}

// Helper to build a table management service with mocks.
func setupTableManagementService() (*pkg.DatabaseService, *MockTableService, *MockBulkService, *MockModelService, *MockColumnService, *MockViewService, *MockRelationshipService, *MockAssetManagementService, interfaces.TableManagementService) {
	mockTable := &MockTableService{}
	mockBulk := &MockBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	db := &pkg.DatabaseService{TableService: mockTable, BulkService: mockBulk}

	svc := services.NewTableManagementService("postgres", db, mockModel, mockColumn, mockView, mockRel, mockAsset)

	return db, mockTable, mockBulk, mockModel, mockColumn, mockView, mockRel, mockAsset, svc
}

// Ensure the mock bulk service satisfies the interface used by DatabaseService
var _ interface {
	BulkInsert(tableName string, records []map[string]interface{}) ([]map[string]interface{}, error)
	Upsert(tableName string, data map[string]interface{}, conflictColumns []string, updateColumns []string) (map[string]interface{}, error)
	BulkUpdate(tableName string, updates []map[string]interface{}, whereColumn string) (int64, error)
	BulkDelete(tableName string, ids []interface{}, idColumn string) (int64, error)
} = &MockBulkService{}

// Ensure mock table service satisfies the table interface used by DatabaseService
var _ interface {
	GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error)
	CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error)
	UpdateRecord(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error)
	DeleteRecord(tableName string, id interface{}) error
	GetTables(schema string) ([]dbModels.Table, error)
	CreateTable(req dbModels.CreateTableRequest) error
	AddColumn(tableName string, req dbModels.AddColumnRequest) error
	AlterTable(tableName string, req dbModels.AlterTableRequest) error
	BuildComplexQuery(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error)
	CreateSchema(ctx context.Context, schemaName string) error
	DropTable(ctx context.Context, tableName string) error
	CreateView(ctx context.Context, viewName string, viewSQL string) error
	CreateFunction(ctx context.Context, functionName string, functionSQL string) error
	GetByFunction(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error)
} = &MockTableService{}

// Ensure interface compliance where needed
var _ interfaces.TableManagementService = services.NewTableManagementService("postgres", &pkg.DatabaseService{}, &MockModelService{}, &MockColumnService{}, &MockViewService{}, &MockRelationshipService{}, &MockAssetManagementService{})
