package table_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/table"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewModelService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewModelService(db)

	assert.NotNil(t, svc)
}

func TestCreateModel(t *testing.T) {
	t.Run("repo not initialized", func(t *testing.T) {
		svc := services.NewModelService(nil)
		cm, ok := svc.(interface {
			CreateModel(ctx context.Context, schemaName string) (tenant.Model, error)
		})
		assert.True(t, ok)

		_, err := cm.CreateModel(context.Background(), "schema")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository not initialized")
	})

	t.Run("create table error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)
		cm, ok := svc.(interface {
			CreateModel(ctx context.Context, schemaName string) (tenant.Model, error)
		})
		assert.True(t, ok)

		mockTable.On("CreateTable", mock.Anything).Return(errors.New("fail"))

		_, err := cm.CreateModel(context.Background(), "schema")

		assert.Error(t, err)
		mockTable.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)
		cm, ok := svc.(interface {
			CreateModel(ctx context.Context, schemaName string) (tenant.Model, error)
		})
		assert.True(t, ok)

		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("AddColumn", "\"schema\".\"models\"", mock.Anything).
			Return(nil).Twice()

		_, err := cm.CreateModel(context.Background(), "schema")

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})
}

func TestCreate(t *testing.T) {
	t.Run("create record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("AddColumn", "\"schema\".\"models\"", mock.Anything).
			Return(nil).Twice()
		mockTable.On("CreateRecord", tenant.Model{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.Create(context.Background(), dto.ModelInsertion{ID: "1", BaseID: "b", WorkspaceID: "w", Title: "T", Alias: "a"}, "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("AddColumn", "\"schema\".\"models\"", mock.Anything).
			Return(nil).Twice()
		mockTable.On("CreateRecord", tenant.Model{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		_, err := svc.Create(context.Background(), dto.ModelInsertion{ID: "1", BaseID: "b", WorkspaceID: "w", Title: "T", Alias: "a"}, "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		id := uuid.New()
		mockTable.On("AddColumn", "\"schema\".\"models\"", mock.Anything).
			Return(nil).Twice()
		mockTable.On("CreateRecord", tenant.Model{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": id.String(), "title": "T"}, nil)

		model, err := svc.Create(context.Background(), dto.ModelInsertion{ID: id.String(), BaseID: "b", WorkspaceID: "w", Title: "T", Alias: "a"}, "schema")

		assert.NoError(t, err)
		assert.Equal(t, id, model.ID)
	})
}

func TestGetModelByID(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		svc := services.NewModelService(nil)

		_, err := svc.GetModelByID(context.Background(), "schema", "id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository not initialized")
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetModelByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.TableNotFound)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "title": "T"}}, nil)

		model, err := svc.GetModelByID(context.Background(), "schema", id.String())

		assert.NoError(t, err)
		assert.Equal(t, id, model.ID)
	})
}

func TestFetchModels(t *testing.T) {
	t.Run("repo nil", func(t *testing.T) {
		svc := services.NewModelService(nil)

		_, err := svc.GetAllModels(context.Background(), "schema")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository not initialized")
	})

	t.Run("get data error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetAllModels(context.Background(), "schema")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch models")
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetAllModels(context.Background(), "schema")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to map model data")
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "T"}}, nil)

		models, err := svc.GetAllModels(context.Background(), "schema")

		assert.NoError(t, err)
		assert.Len(t, models, 1)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("table not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		alias := "a"
		_, err := svc.Update(context.Background(), "schema", "id", dto.UpdateModelRequest{Alias: &alias})

		assert.ErrorIs(t, err, app_errors.TableNotFound)
	})

	t.Run("update error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)
		id := uuid.New().String()

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "T"}}, nil)

		mockTable.On("UpdateRecord", tenant.Model{}.TableName("schema"), "id", mock.Anything).
			Return(nil, errors.New("db error"))

		alias := "a"
		_, err := svc.Update(context.Background(), "schema", "id", dto.UpdateModelRequest{Alias: &alias})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)
		id := uuid.New().String()

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "T"}}, nil)

		mockTable.On("UpdateRecord", tenant.Model{}.TableName("schema"), "id", mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		alias := "a"
		_, err := svc.Update(context.Background(), "schema", "id", dto.UpdateModelRequest{Alias: &alias})

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "title": "T"}}, nil)
		mockTable.On("UpdateRecord", tenant.Model{}.TableName("schema"), id.String(), mock.Anything).
			Return(map[string]interface{}{"id": id.String(), "title": "T"}, nil)

		alias := "a"
		model, err := svc.Update(context.Background(), "schema", id.String(), dto.UpdateModelRequest{Alias: &alias})

		assert.NoError(t, err)
		assert.Equal(t, id, model.ID)
	})
}

func TestDeleteModels(t *testing.T) {
	t.Run("repo nil", func(t *testing.T) {
		svc := services.NewModelService(nil)

		err := svc.DeleteModels(context.Background(), "schema", "id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), services.ErrRepositoryNotInitialized)
	})

	t.Run("empty id", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewModelService(db)

		err := svc.DeleteModels(context.Background(), "schema", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model ID cannot be empty")
	})

	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("DeleteRecord", tenant.Model{}.TableName("schema"), "id").
			Return(errors.New("db error"))

		err := svc.DeleteModels(context.Background(), "schema", "id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete model")
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("DeleteRecord", tenant.Model{}.TableName("schema"), "id").
			Return(nil)

		err := svc.DeleteModels(context.Background(), "schema", "id")

		assert.NoError(t, err)
	})
}

func TestGetModelByBaseID(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		svc := services.NewModelService(nil)

		_, err := svc.GetModelByBaseID(context.Background(), "schema", "base")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Filters) == 1 && q.Filters[0].Column == "base_id"
		})).Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "T"}}, nil)

		models, err := svc.GetModelByBaseID(context.Background(), "schema", "base")

		assert.NoError(t, err)
		assert.Len(t, models, 1)
	})
}

func TestGetModelByWorkspaceID(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		svc := services.NewModelService(nil)

		_, err := svc.GetModelByWorkspaceID(context.Background(), "schema", "ws")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("GetTableData", tenant.Model{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Filters) == 1 && q.Filters[0].Column == "workspace_id"
		})).Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "T"}}, nil)

		models, err := svc.GetModelByWorkspaceID(context.Background(), "schema", "ws")

		assert.NoError(t, err)
		assert.Len(t, models, 1)
	})
}

func TestDeleteModel(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("DeleteRecord", tenant.Model{}.TableName("schema"), "id").
			Return(errors.New("db error"))

		err := svc.DeleteModel(context.Background(), "schema", "id")

		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "failed to delete model"))
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewModelService(db)

		mockTable.On("DeleteRecord", tenant.Model{}.TableName("schema"), "id").
			Return(nil)

		err := svc.DeleteModel(context.Background(), "schema", "id")

		assert.NoError(t, err)
	})
}

// Ensure mock type satisfies the table interface used by DatabaseService
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
