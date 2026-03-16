package workspace_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/workspace"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewWorkspaceService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewWorkspaceService(db)

	assert.NotNil(t, svc)
}

func TestCreateWorkspace(t *testing.T) {
	t.Run("repo not initialized", func(t *testing.T) {
		svc := services.NewWorkspaceService(nil)

		_, err := svc.CreateWorkspace(context.Background(), "schema")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository not initialized")
	})

	t.Run("create table error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("CreateTable", mock.Anything).Return(errors.New("fail"))

		_, err := svc.CreateWorkspace(context.Background(), "schema")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockTable.On("AddColumn", "\"schema\".\"workspaces\"", mock.Anything).
			Return(nil).Twice()

		_, err := svc.CreateWorkspace(context.Background(), "schema")

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})
}

func TestWorkspaceInsertion(t *testing.T) {
	t.Run("create record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("AddColumn", "\"schema\".\"workspaces\"", mock.Anything).
			Return(errors.New("already exists")).Twice()
		mockTable.On("CreateRecord", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.WorkspaceInsertion(context.Background(), dto.CreateWorkspaceRequest{Title: "My Title"}, "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("AddColumn", "\"schema\".\"workspaces\"", mock.Anything).
			Return(errors.New("boom")).Twice()
		mockTable.On("CreateRecord", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		_, err := svc.WorkspaceInsertion(context.Background(), dto.CreateWorkspaceRequest{Title: "My Title"}, "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		var captured map[string]interface{}
		mockTable.On("AddColumn", "\"schema\".\"workspaces\"", mock.Anything).
			Return(nil).Twice()
		mockTable.On("CreateRecord", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(1).(map[string]interface{}) }).
			Return(map[string]interface{}{"id": uuid.New().String(), "title": "My Title"}, nil)

		ws, err := svc.WorkspaceInsertion(context.Background(), dto.CreateWorkspaceRequest{Title: "My Title", CreatedBy: "u"}, "schema")

		assert.NoError(t, err)
		assert.Equal(t, "My Title", ws.Title)
		slug, _ := captured["slug"].(string)
		assert.True(t, strings.HasPrefix(slug, "my_title-workspace-"))
		assert.Len(t, slug, len("my_title-workspace-")+8)
		assert.Equal(t, "u", captured["last_modified_by"])
	})
}

func TestGetWorkspaceByID(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewWorkspaceService(db)

		_, err := svc.GetWorkspaceByID(context.Background(), "schema", "")

		assert.Error(t, err)
	})

	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetWorkspaceByID(context.Background(), "schema", "id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch workspace")
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetWorkspaceByID(context.Background(), "schema", "id")

		assert.Error(t, err)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetWorkspaceByID(context.Background(), "schema", "id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to map workspace data")
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "title": "T"}}, nil)

		ws, err := svc.GetWorkspaceByID(context.Background(), "schema", id.String())

		assert.NoError(t, err)
		assert.Equal(t, id, ws.ID)
	})
}

func TestGetAllWorkspaces(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetAllWorkspaces(context.Background(), "schema")

		assert.Error(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		rows, err := svc.GetAllWorkspaces(context.Background(), "schema")

		assert.NoError(t, err)
		assert.Empty(t, rows)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetAllWorkspaces(context.Background(), "schema")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to map workspace data")
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "T"}}, nil)

		rows, err := svc.GetAllWorkspaces(context.Background(), "schema")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestUpdateWorkspace(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewWorkspaceService(db)

		_, err := svc.UpdateWorkspace(context.Background(), "schema", "", dto.WorkspaceUpdate{})

		assert.Error(t, err)
	})

	t.Run("get workspace error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateWorkspace(context.Background(), "schema", "id", dto.WorkspaceUpdate{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found")
	})

	t.Run("update error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "T"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Workspace{}.TableName("schema"), id, mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateWorkspace(context.Background(), "schema", id, dto.WorkspaceUpdate{})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("empty update result", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "T"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Workspace{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{}, nil)

		_, err := svc.UpdateWorkspace(context.Background(), "schema", id, dto.WorkspaceUpdate{})

		assert.ErrorIs(t, err, app_errors.InvalidPayload)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "T"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.Workspace{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{"id": id}, nil)
		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "Updated"}}, nil).Once()

		ws, err := svc.UpdateWorkspace(context.Background(), "schema", id, dto.WorkspaceUpdate{})

		assert.NoError(t, err)
		assert.Equal(t, "Updated", ws.Title)
	})
}

func TestDeleteWorkspace(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewWorkspaceService(db)

		err := svc.DeleteWorkspace(context.Background(), "schema", "")

		assert.Error(t, err)
	})

	t.Run("get workspace error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		err := svc.DeleteWorkspace(context.Background(), "schema", "id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found")
	})

	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "T"}}, nil).Once()
		mockTable.On("DeleteRecord", tenant.Workspace{}.TableName("schema"), id).
			Return(errors.New("db error"))

		err := svc.DeleteWorkspace(context.Background(), "schema", id)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete workspace")
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "T"}}, nil).Once()
		mockTable.On("DeleteRecord", tenant.Workspace{}.TableName("schema"), id).
			Return(nil)

		err := svc.DeleteWorkspace(context.Background(), "schema", id)

		assert.NoError(t, err)
	})
}

func TestGetBulkWorkspaces(t *testing.T) {
	t.Run("empty ids", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewWorkspaceService(db)

		rows, err := svc.GetBulkWorkspaces(context.Background(), "schema", []string{})

		assert.NoError(t, err)
		assert.Empty(t, rows)
	})

	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetBulkWorkspaces(context.Background(), "schema", []string{"id"})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("empty rows", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		rows, err := svc.GetBulkWorkspaces(context.Background(), "schema", []string{"id"})

		assert.NoError(t, err)
		assert.Empty(t, rows)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetBulkWorkspaces(context.Background(), "schema", []string{"id"})

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceService(db)

		mockTable.On("GetTableData", tenant.Workspace{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "T"}}, nil)

		rows, err := svc.GetBulkWorkspaces(context.Background(), "schema", []string{"id"})

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
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
