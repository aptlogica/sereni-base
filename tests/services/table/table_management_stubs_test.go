package table_test

import (
	"context"

	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	"serenibase/internal/services/interfaces"
	services "serenibase/internal/services/table"
)

// StubTableService provides function hooks for TableService behavior.
type StubTableService struct {
	GetTableDataFn      func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error)
	CreateRecordFn      func(tableName string, data map[string]interface{}) (map[string]interface{}, error)
	UpdateRecordFn      func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error)
	DeleteRecordFn      func(tableName string, id interface{}) error
	GetTablesFn         func(schema string) ([]dbModels.Table, error)
	CreateTableFn       func(req dbModels.CreateTableRequest) error
	AddColumnFn         func(tableName string, req dbModels.AddColumnRequest) error
	AlterTableFn        func(tableName string, req dbModels.AlterTableRequest) error
	BuildComplexQueryFn func(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error)
	CreateSchemaFn      func(ctx context.Context, schemaName string) error
	DropTableFn         func(ctx context.Context, tableName string) error
	CreateViewFn        func(ctx context.Context, viewName string, viewSQL string) error
	CreateFunctionFn    func(ctx context.Context, functionName string, functionSQL string) error
	GetByFunctionFn     func(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error)
}

func (s *StubTableService) GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
	if s.GetTableDataFn != nil {
		return s.GetTableDataFn(tableName, params)
	}
	return nil, nil
}
func (s *StubTableService) CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	if s.CreateRecordFn != nil {
		return s.CreateRecordFn(tableName, data)
	}
	return map[string]interface{}{}, nil
}
func (s *StubTableService) UpdateRecord(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	if s.UpdateRecordFn != nil {
		return s.UpdateRecordFn(tableName, id, data)
	}
	return map[string]interface{}{}, nil
}
func (s *StubTableService) DeleteRecord(tableName string, id interface{}) error {
	if s.DeleteRecordFn != nil {
		return s.DeleteRecordFn(tableName, id)
	}
	return nil
}
func (s *StubTableService) GetTables(schema string) ([]dbModels.Table, error) {
	if s.GetTablesFn != nil {
		return s.GetTablesFn(schema)
	}
	return nil, nil
}
func (s *StubTableService) CreateTable(req dbModels.CreateTableRequest) error {
	if s.CreateTableFn != nil {
		return s.CreateTableFn(req)
	}
	return nil
}
func (s *StubTableService) AddColumn(tableName string, req dbModels.AddColumnRequest) error {
	if s.AddColumnFn != nil {
		return s.AddColumnFn(tableName, req)
	}
	return nil
}
func (s *StubTableService) AlterTable(tableName string, req dbModels.AlterTableRequest) error {
	if s.AlterTableFn != nil {
		return s.AlterTableFn(tableName, req)
	}
	return nil
}
func (s *StubTableService) BuildComplexQuery(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error) {
	if s.BuildComplexQueryFn != nil {
		return s.BuildComplexQueryFn(tableName, filters)
	}
	return dbModels.QueryParams{}, nil
}
func (s *StubTableService) CreateSchema(ctx context.Context, schemaName string) error {
	if s.CreateSchemaFn != nil {
		return s.CreateSchemaFn(ctx, schemaName)
	}
	return nil
}
func (s *StubTableService) DropTable(ctx context.Context, tableName string) error {
	if s.DropTableFn != nil {
		return s.DropTableFn(ctx, tableName)
	}
	return nil
}
func (s *StubTableService) CreateView(ctx context.Context, viewName string, viewSQL string) error {
	if s.CreateViewFn != nil {
		return s.CreateViewFn(ctx, viewName, viewSQL)
	}
	return nil
}
func (s *StubTableService) CreateFunction(ctx context.Context, functionName string, functionSQL string) error {
	if s.CreateFunctionFn != nil {
		return s.CreateFunctionFn(ctx, functionName, functionSQL)
	}
	return nil
}
func (s *StubTableService) GetByFunction(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error) {
	if s.GetByFunctionFn != nil {
		return s.GetByFunctionFn(ctx, functionName, args)
	}
	return nil, nil
}

// StubBulkService provides function hooks for BulkService behavior.
type StubBulkService struct {
	BulkInsertFn func(tableName string, records []map[string]interface{}) ([]map[string]interface{}, error)
	UpsertFn     func(tableName string, data map[string]interface{}, conflictColumns []string, updateColumns []string) (map[string]interface{}, error)
	BulkUpdateFn func(tableName string, updates []map[string]interface{}, whereColumn string) (int64, error)
	BulkDeleteFn func(tableName string, ids []interface{}, idColumn string) (int64, error)
}

func (s *StubBulkService) BulkInsert(tableName string, records []map[string]interface{}) ([]map[string]interface{}, error) {
	if s.BulkInsertFn != nil {
		return s.BulkInsertFn(tableName, records)
	}
	return nil, nil
}
func (s *StubBulkService) Upsert(tableName string, data map[string]interface{}, conflictColumns []string, updateColumns []string) (map[string]interface{}, error) {
	if s.UpsertFn != nil {
		return s.UpsertFn(tableName, data, conflictColumns, updateColumns)
	}
	return nil, nil
}
func (s *StubBulkService) BulkUpdate(tableName string, updates []map[string]interface{}, whereColumn string) (int64, error) {
	if s.BulkUpdateFn != nil {
		return s.BulkUpdateFn(tableName, updates, whereColumn)
	}
	return 0, nil
}
func (s *StubBulkService) BulkDelete(tableName string, ids []interface{}, idColumn string) (int64, error) {
	if s.BulkDeleteFn != nil {
		return s.BulkDeleteFn(tableName, ids, idColumn)
	}
	return 0, nil
}

func setupTableManagementServiceWithStubs(tableSvc *StubTableService, bulkSvc *StubBulkService, model interfaces.ModelService, column interfaces.ColumnService, view interfaces.ViewService, rel interfaces.RelationshipService, asset interfaces.AssetManagementService) interfaces.TableManagementService {
	db := &pkg.DatabaseService{TableService: tableSvc, BulkService: bulkSvc}
	return services.NewTableManagementService("postgres", db, model, column, view, rel, asset)
}
