package table_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateTableWithDefaults_Success(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

	model := tenant.Model{ID: uuid.New(), BaseID: uuid.New(), WorkspaceID: uuid.New(), Alias: "tbl", CreatedBy: "user", Title: "Table"}
	mockModel.On("Create", mock.Anything, mock.Anything, "schema").Return(model, nil)
	mockTable.On("CreateTable", mock.Anything).Return(nil)
	mockColumn.On("BulkInsert", mock.Anything, "schema").Return([]tenant.Column{{ID: uuid.New(), Title: "Title", ColumnName: "title", BaseID: model.BaseID.String(), ModelID: model.ID.String()}}, nil)
	mockView.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.View{ID: uuid.New(), Title: "Default", BaseID: model.BaseID.String(), ModelID: model.ID.String()}, nil)

	mockModel.On("GetModelByID", mock.Anything, "schema", model.ID.String()).Return(model, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", model.ID.String()).Return([]tenant.Column{}, nil)
	mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	resp, err := svc.CreateTableWithDefaults(context.Background(), dto.CreateTableRequest{BaseID: model.BaseID.String(), WorkspaceID: "ws", Title: "Table", CreatedBy: "user"}, "schema")

	assert.NoError(t, err)
	assert.Equal(t, model.ID, resp.Model.ID)
}

func TestCreateTableWithDefaults_Errors(t *testing.T) {
	t.Run("create model error", func(t *testing.T) {
		_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

		mockModel.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.Model{}, errors.New("fail"))

		_, err := svc.CreateTableWithDefaults(context.Background(), dto.CreateTableRequest{Title: "T"}, "schema")

		assert.Error(t, err)
	})

	t.Run("create table error", func(t *testing.T) {
		_, mockTable, _, mockModel, _, _, _, _, svc := setupTableManagementService()

		mockModel.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.Model{Alias: "tbl"}, nil)
		mockTable.On("CreateTable", mock.Anything).Return(errors.New("fail"))

		_, err := svc.CreateTableWithDefaults(context.Background(), dto.CreateTableRequest{Title: "T"}, "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("insert system columns error", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

		mockModel.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.Model{Alias: "tbl"}, nil)
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockColumn.On("BulkInsert", mock.Anything, "schema").Return(nil, errors.New("fail"))

		_, err := svc.CreateTableWithDefaults(context.Background(), dto.CreateTableRequest{Title: "T"}, "schema")

		assert.Error(t, err)
	})

	t.Run("create default view error", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

		model := tenant.Model{ID: uuid.New(), BaseID: uuid.New(), Alias: "tbl", CreatedBy: "user"}
		mockModel.On("Create", mock.Anything, mock.Anything, "schema").Return(model, nil)
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockColumn.On("BulkInsert", mock.Anything, "schema").Return([]tenant.Column{}, nil)
		mockView.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.View{}, errors.New("fail"))

		_, err := svc.CreateTableWithDefaults(context.Background(), dto.CreateTableRequest{Title: "T"}, "schema")

		assert.Error(t, err)
	})

	t.Run("get all records error", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

		model := tenant.Model{ID: uuid.New(), BaseID: uuid.New(), Alias: "tbl", CreatedBy: "user"}
		mockModel.On("Create", mock.Anything, mock.Anything, "schema").Return(model, nil)
		mockTable.On("CreateTable", mock.Anything).Return(nil)
		mockColumn.On("BulkInsert", mock.Anything, "schema").Return([]tenant.Column{}, nil)
		mockView.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.View{ID: uuid.New()}, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", model.ID.String()).Return(tenant.Model{}, errors.New("fail"))

		_, err := svc.CreateTableWithDefaults(context.Background(), dto.CreateTableRequest{Title: "T"}, "schema")

		assert.Error(t, err)
	})
}

func TestInsertSystemColumns_SystemFieldRespected(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

	model := tenant.Model{ID: uuid.New(), BaseID: uuid.New(), WorkspaceID: uuid.New(), Alias: "tbl", CreatedBy: "user", Title: "Table"}
	mockModel.On("Create", mock.Anything, mock.Anything, "schema").Return(model, nil)
	mockTable.On("CreateTable", mock.Anything).Return(nil)

	// Capture the columns being inserted to verify System field
	var insertedColumns []dto.ColumnInsertion
	mockColumn.On("BulkInsert", mock.Anything, "schema").Run(func(args mock.Arguments) {
		insertedColumns = args.Get(0).([]dto.ColumnInsertion)
	}).Return([]tenant.Column{
		{ID: uuid.New(), Title: "Id", ColumnName: "id", BaseID: model.BaseID.String(), ModelID: model.ID.String(), System: true},
		{ID: uuid.New(), Title: "Title", ColumnName: "title", BaseID: model.BaseID.String(), ModelID: model.ID.String(), System: false},
	}, nil)

	mockView.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.View{ID: uuid.New(), Title: "Default", BaseID: model.BaseID.String(), ModelID: model.ID.String()}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", model.ID.String()).Return(model, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", model.ID.String()).Return([]tenant.Column{}, nil)
	mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	resp, err := svc.CreateTableWithDefaults(context.Background(), dto.CreateTableRequest{BaseID: model.BaseID.String(), WorkspaceID: "ws", Title: "Table", CreatedBy: "user"}, "schema")

	assert.NoError(t, err)
	assert.Equal(t, model.ID, resp.Model.ID)

	// Verify that the System field is properly set based on column definitions
	// Find the Title column (should have System: true)
	var titleColumn *dto.ColumnInsertion
	for i := range insertedColumns {
		if insertedColumns[i].Title == "Title" {
			titleColumn = &insertedColumns[i]
			break
		}
	}

	assert.NotNil(t, titleColumn, "Title column should be inserted")
	assert.True(t, titleColumn.System, "Title column should have System: true")

	// Verify other system columns have System: true
	for _, col := range insertedColumns {
		if col.Title == "Id" || col.Title == "Created Time" || col.Title == "Created By" || col.Title == "Title" {
			assert.True(t, col.System, "Column %s should have System: true", col.Title)
		}
	}
}

func TestCreateTableWithDefaultsImport_Success(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

	model := tenant.Model{ID: uuid.New(), BaseID: uuid.New(), WorkspaceID: uuid.New(), Alias: "tbl", CreatedBy: "user", Title: "Table"}
	mockModel.On("Create", mock.Anything, mock.Anything, "schema").Return(model, nil)
	mockTable.On("CreateTable", mock.Anything).Return(nil)
	mockColumn.On("BulkInsert", mock.Anything, "schema").Return([]tenant.Column{{ID: uuid.New(), BaseID: model.BaseID.String(), ModelID: model.ID.String()}}, nil)
	mockView.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.View{ID: uuid.New(), Title: "Default", BaseID: model.BaseID.String(), ModelID: model.ID.String()}, nil)

	mockModel.On("GetModelByID", mock.Anything, "schema", model.ID.String()).Return(model, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", model.ID.String()).Return([]tenant.Column{}, nil)
	mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	importSvc, ok := svc.(interface {
		CreateTableWithDefaultsImport(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error)
	})
	assert.True(t, ok)

	resp, err := importSvc.CreateTableWithDefaultsImport(context.Background(), dto.CreateTableRequest{BaseID: model.BaseID.String(), WorkspaceID: "ws", Title: "Table", CreatedBy: "user"}, "schema")

	assert.NoError(t, err)
	assert.Equal(t, model.ID, resp.Model.ID)
}

func TestUpdateTable(t *testing.T) {
	_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

	updated := tenant.Model{ID: uuid.New(), Title: "Updated"}
	mockModel.On("Update", mock.Anything, "schema", "id", mock.Anything).Return(updated, nil)

	resp, err := svc.UpdateTable(context.Background(), "id", dto.UpdateTableRequest{UpdatedBy: "user"}, "schema")

	assert.NoError(t, err)
	assert.Equal(t, updated.ID, resp.Model.ID)
}

func TestGetTableByID(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

	model := tenant.Model{ID: uuid.New(), Alias: "tbl"}
	mockModel.On("GetModelByID", mock.Anything, "schema", "id").Return(model, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", "id").Return([]tenant.Column{}, nil)
	mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)
	mockView.On("GetViewsByModelID", mock.Anything, "schema", "id").Return([]tenant.View{}, nil)

	resp, err := svc.GetTableByID(context.Background(), "id", "schema")

	assert.NoError(t, err)
	assert.Equal(t, model.ID, resp.Model.ID)
}

func TestGetRecordsWithLookups_Normalize(t *testing.T) {
	_, mockTable, _, _, _, _, _, _, svc := setupTableManagementService()

	mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).
		Return([]map[string]interface{}{{"get_table_data_with_relation": []interface{}{map[string]interface{}{"a": 1}}}}, nil)

	getRecords, ok := svc.(interface {
		GetRecordsWithLookups(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error)
	})
	assert.True(t, ok)

	records, err := getRecords.GetRecordsWithLookups(context.Background(), "schema", "tbl", []dto.ColumnResponse{})

	assert.NoError(t, err)
	assert.Len(t, records.Records, 1)
}

func TestGetRecordsWithLookups_Empty(t *testing.T) {
	_, mockTable, _, _, _, _, _, _, svc := setupTableManagementService()

	mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).
		Return([]map[string]interface{}{}, nil)

	getRecords, ok := svc.(interface {
		GetRecordsWithLookups(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error)
	})
	assert.True(t, ok)

	records, err := getRecords.GetRecordsWithLookups(context.Background(), "schema", "tbl", []dto.ColumnResponse{})

	assert.NoError(t, err)
	assert.Nil(t, records.Records)
}

func TestGetAllTablesAndModels(t *testing.T) {
	_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

	model := tenant.Model{ID: uuid.New(), BaseID: uuid.New(), WorkspaceID: uuid.New(), Title: "T"}
	mockModel.On("GetAllModels", mock.Anything, "schema").Return([]tenant.Model{model}, nil)

	tables, err := svc.GetAllTables(context.Background(), "schema")

	assert.NoError(t, err)
	assert.Len(t, tables, 1)
	assert.Equal(t, model.ID, tables[0].Model.ID)
}

func TestGetModelByBaseAndWorkspace(t *testing.T) {
	_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

	model := tenant.Model{ID: uuid.New(), BaseID: uuid.New(), WorkspaceID: uuid.New(), Title: "T"}
	mockModel.On("GetModelByBaseID", mock.Anything, "schema", "base").Return([]tenant.Model{model}, nil)
	mockModel.On("GetModelByWorkspaceID", mock.Anything, "schema", "ws").Return([]tenant.Model{model}, nil)

	byBase, err := svc.GetModelByBaseID(context.Background(), "schema", "base")
	assert.NoError(t, err)
	assert.Len(t, byBase, 1)

	byWS, err := svc.GetModelByWorkspaceID(context.Background(), "schema", "ws")
	assert.NoError(t, err)
	assert.Len(t, byWS, 1)
}

func TestDeleteTable(t *testing.T) {
	t.Run("model not found", func(t *testing.T) {
		_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

		mockModel.On("GetModelByID", mock.Anything, "schema", "id").Return(tenant.Model{}, errors.New("fail"))

		err := svc.DeleteTable(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.TableNotFound)
	})

	t.Run("success", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

		model := tenant.Model{ID: uuid.New(), Alias: "tbl"}
		mockModel.On("GetModelByID", mock.Anything, "schema", "id").Return(model, nil)
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", "id").Return([]tenant.Column{}, nil)
		mockView.On("GetViewsByModelID", mock.Anything, "schema", "id").Return([]tenant.View{}, nil)
		mockModel.On("DeleteModel", mock.Anything, "schema", "id").Return(nil)
		mockTable.On("DropTable", mock.Anything, mock.Anything).Return(nil)

		err := svc.DeleteTable(context.Background(), "schema", "id")

		assert.NoError(t, err)
	})
}

func TestColumnAndViewWrappers(t *testing.T) {
	_, _, _, _, mockColumn, mockView, _, _, svc := setupTableManagementService()

	modelID := uuid.New().String()
	baseID := uuid.New().String()
	col := tenant.Column{ID: uuid.New(), ColumnName: "c", ModelID: modelID, BaseID: baseID}
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
	mockColumn.On("GetAllColumns", mock.Anything, "schema").Return([]tenant.Column{col}, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", "mid").Return([]tenant.Column{col}, nil)

	view := tenant.View{ID: uuid.New(), Title: "v", BaseID: uuid.New().String(), ModelID: uuid.New().String()}
	mockView.On("GetViewByID", mock.Anything, "schema", "vid").Return(view, nil)
	mockView.On("GetAllViews", mock.Anything, "schema").Return([]tenant.View{view}, nil)
	mockView.On("GetViewsByModelID", mock.Anything, "schema", "mid").Return([]tenant.View{view}, nil)

	_, err := svc.GetColumnById(context.Background(), "schema", "cid")
	assert.NoError(t, err)
	_, err = svc.GetAllColumns(context.Background(), "schema")
	assert.NoError(t, err)
	_, err = svc.GetColumnsByModelID(context.Background(), "schema", "mid")
	assert.NoError(t, err)

	_, err = svc.GetViewByID(context.Background(), "schema", "vid")
	assert.NoError(t, err)
	_, err = svc.GetAllViews(context.Background(), "schema")
	assert.NoError(t, err)
	_, err = svc.GetViewsByModelID(context.Background(), "schema", "mid")
	assert.NoError(t, err)
}

func TestCreateViewAndUpdateDeleteView(t *testing.T) {
	_, _, _, _, _, mockView, _, _, svc := setupTableManagementService()

	t.Run("create view struct error", func(t *testing.T) {
		mockView.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.View{}, nil).Once()

		_, err := svc.CreateView(context.Background(), "schema", dto.CreateViewRequest{ModelID: uuid.New(), BaseID: uuid.New(), Title: "Title", Description: "", Type: "grid", Meta: &map[string]interface{}{}, CreatedBy: "u"})

		assert.ErrorIs(t, err, app_errors.ErrStructToStruct)
	})

	t.Run("update/delete", func(t *testing.T) {
		view := tenant.View{ID: uuid.New(), Title: "v", BaseID: uuid.New().String(), ModelID: uuid.New().String()}
		mockView.On("GetViewByID", mock.Anything, "schema", "vid").Return(view, nil)
		mockView.On("UpdateView", mock.Anything, "schema", "vid", mock.Anything).Return(view, nil)
		mockView.On("DeleteView", mock.Anything, "schema", "vid").Return(nil)

		_, err := svc.UpdateView(context.Background(), "schema", "vid", dto.ViewUpdate{})
		assert.NoError(t, err)

		err = svc.DeleteView(context.Background(), "schema", "vid")
		assert.NoError(t, err)
	})
}

func TestTableManagement_MetadataErrorBranches(t *testing.T) {
	t.Run("update table error", func(t *testing.T) {
		_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

		mockModel.On("Update", mock.Anything, "schema", "id", mock.Anything).Return(tenant.Model{}, errors.New("update failed"))

		_, err := svc.UpdateTable(context.Background(), "id", dto.UpdateTableRequest{}, "schema")
		assert.Error(t, err)
	})

	t.Run("get all tables error", func(t *testing.T) {
		_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

		mockModel.On("GetAllModels", mock.Anything, "schema").Return([]tenant.Model{}, errors.New("list failed"))

		_, err := svc.GetAllTables(context.Background(), "schema")
		assert.Error(t, err)
	})

	t.Run("get models by base error", func(t *testing.T) {
		_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

		mockModel.On("GetModelByBaseID", mock.Anything, "schema", "base").Return(nil, errors.New("base failed"))

		_, err := svc.GetModelByBaseID(context.Background(), "schema", "base")
		assert.Error(t, err)
	})

	t.Run("get models by workspace error", func(t *testing.T) {
		_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

		mockModel.On("GetModelByWorkspaceID", mock.Anything, "schema", "workspace").Return([]tenant.Model{}, errors.New("workspace failed"))

		_, err := svc.GetModelByWorkspaceID(context.Background(), "schema", "workspace")
		assert.Error(t, err)
	})

	t.Run("column and view read errors", func(t *testing.T) {
		_, _, _, _, mockColumn, mockView, _, _, svc := setupTableManagementService()

		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(tenant.Column{}, errors.New("column failed"))
		mockColumn.On("GetAllColumns", mock.Anything, "schema").Return([]tenant.Column{}, errors.New("columns failed"))
		mockView.On("GetViewByID", mock.Anything, "schema", "vid").Return(tenant.View{}, errors.New("view failed"))
		mockView.On("GetAllViews", mock.Anything, "schema").Return([]tenant.View{}, errors.New("views failed"))
		mockView.On("GetViewsByModelID", mock.Anything, "schema", "mid").Return([]tenant.View{}, errors.New("model views failed"))

		_, err := svc.GetColumnById(context.Background(), "schema", "cid")
		assert.Error(t, err)
		_, err = svc.GetAllColumns(context.Background(), "schema")
		assert.Error(t, err)
		_, err = svc.GetViewByID(context.Background(), "schema", "vid")
		assert.Error(t, err)
		_, err = svc.GetAllViews(context.Background(), "schema")
		assert.Error(t, err)
		_, err = svc.GetViewsByModelID(context.Background(), "schema", "mid")
		assert.Error(t, err)
	})

	t.Run("update and delete view errors", func(t *testing.T) {
		_, _, _, _, _, mockView, _, _, svc := setupTableManagementService()

		view := tenant.View{ID: uuid.New(), Title: "v", BaseID: uuid.New().String(), ModelID: uuid.New().String()}
		mockView.On("GetViewByID", mock.Anything, "schema", "vid").Return(view, nil)
		mockView.On("UpdateView", mock.Anything, "schema", "vid", mock.Anything).Return(tenant.View{}, errors.New("update view failed"))
		mockView.On("DeleteView", mock.Anything, "schema", "vid").Return(errors.New("delete view failed"))

		_, err := svc.UpdateView(context.Background(), "schema", "vid", dto.ViewUpdate{})
		assert.Error(t, err)
		err = svc.DeleteView(context.Background(), "schema", "vid")
		assert.Error(t, err)
	})
}

// Additional tests from consolidated files

func TestAddColumn_SimpleAndInvalid(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

	t.Run("simple", func(t *testing.T) {
		modelID := uuid.New()
		baseID := uuid.New()
		col := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID.String(), ColumnName: "col", Title: "Title", UIDT: "text", DT: helpers.StringPtr("TEXT")}
		mockColumn.On("Create", mock.Anything, mock.Anything, "schema").Return(col, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{ID: modelID, Alias: "tbl"}, nil)
		mockTable.On("AddColumn", mock.Anything, mock.Anything).Return(nil)

		resp, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{ModelID: modelID, BaseID: baseID, Title: "Title", UIDT: "text", Description: "", Meta: nil, CreatedBy: "u"})

		assert.NoError(t, err)
		assert.Equal(t, modelID, resp.ModelID)
	})

	t.Run("invalid uidt", func(t *testing.T) {
		_, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{ModelID: uuid.New(), BaseID: uuid.New(), Title: "Bad", UIDT: "bad"})
		assert.ErrorIs(t, err, app_errors.InvalidUIDT)
	})
}

func TestAddColumn_LinksAndLookup(t *testing.T) {
	t.Run("links invalid meta", func(t *testing.T) {
		_, _, _, _, _, _, _, _, svc := setupTableManagementService()

		_, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{ModelID: uuid.New(), BaseID: uuid.New(), Title: "Rel", UIDT: "links", Meta: nil})

		assert.ErrorIs(t, err, app_errors.InvalidColumnMetaForLinkType)
	})

	t.Run("links success", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, _, mockRel, _, svc := setupTableManagementService()

		sourceModelID := uuid.New()
		targetModelID := uuid.New()
		baseID := uuid.New()

		meta := map[string]interface{}{
			"relation": map[string]interface{}{
				"with": targetModelID.String(),
				"type": "one-to-one",
			},
		}

		sourceCol := tenant.Column{ID: uuid.New(), ModelID: sourceModelID.String(), BaseID: baseID.String(), ColumnName: "src", UIDT: "links", DT: helpers.StringPtr("INT")}
		targetCol := tenant.Column{ID: uuid.New(), ModelID: targetModelID.String(), BaseID: baseID.String(), ColumnName: "tgt", UIDT: "links", DT: helpers.StringPtr("INT")}

		mockColumn.On("Create", mock.Anything, mock.MatchedBy(func(req dto.ColumnInsertion) bool { return req.ModelID == sourceModelID }), "schema").Return(sourceCol, nil)
		mockColumn.On("Create", mock.Anything, mock.MatchedBy(func(req dto.ColumnInsertion) bool { return req.ModelID == targetModelID }), "schema").Return(targetCol, nil)
		mockColumn.On("GetMaxOrderIndexOfColumn", mock.Anything, "schema", targetModelID.String()).Return(float64(1), nil)

		mockModel.On("GetModelByID", mock.Anything, "schema", sourceModelID.String()).Return(tenant.Model{ID: sourceModelID, Alias: "src", Title: "Src"}, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", targetModelID.String()).Return(tenant.Model{ID: targetModelID, Alias: "tgt", Title: "Tgt"}, nil)

		mockTable.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
		mockRel.On("Create", mock.Anything, mock.Anything, "schema").Return(tenant.Relation{}, nil)

		addReq := dto.AddColumnRequest{ModelID: sourceModelID, BaseID: baseID, Title: "Rel", UIDT: "links", Meta: meta, CreatedBy: "u"}
		resp, err := svc.AddColumn(context.Background(), "schema", addReq)

		assert.NoError(t, err)
		assert.Equal(t, sourceModelID, resp.ModelID)
	})

	t.Run("lookup invalid meta", func(t *testing.T) {
		_, _, _, _, _, _, _, _, svc := setupTableManagementService()

		_, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{ModelID: uuid.New(), BaseID: uuid.New(), Title: "Lookup", UIDT: "lookup", Meta: nil})

		assert.ErrorIs(t, err, app_errors.InvalidColumnMetaForLookupType)
	})

	t.Run("lookup success", func(t *testing.T) {
		_, _, _, mockModel, mockColumn, _, mockRel, _, svc := setupTableManagementService()

		lookupColumnID := uuid.New().String()
		relationID := uuid.New().String()
		modelID := uuid.New()
		baseID := uuid.New().String()
		lookupModelID := uuid.New().String()

		lookupColumn := tenant.Column{ID: uuid.New(), ModelID: lookupModelID, BaseID: uuid.New().String(), ColumnName: "src_col", UIDT: "text"}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", lookupColumnID).Return(lookupColumn, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", lookupModelID).Return(tenant.Model{ID: uuid.MustParse(lookupModelID), Alias: "lk"}, nil)

		createdColumn := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID, ColumnName: "lk_src_col", UIDT: "lookup"}
		mockColumn.On("Create", mock.Anything, mock.Anything, "schema").Return(createdColumn, nil)

		rel := tenant.Relation{ID: uuid.MustParse(relationID), SourceModelID: modelID.String()}
		mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(rel, nil)
		mockRel.On("UpdateRelation", mock.Anything, relationID, mock.Anything, "schema").Return(tenant.Relation{}, nil)

		meta := map[string]interface{}{"lookup_column_id": lookupColumnID, "relation_id": relationID}
		resp, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{ModelID: modelID, BaseID: uuid.MustParse(baseID), Title: "L", UIDT: "lookup", Meta: meta})

		assert.NoError(t, err)
		assert.Equal(t, modelID, resp.ModelID)
	})
}

func TestUpdateColumn_Variants(t *testing.T) {
	t.Run("system not allowed", func(t *testing.T) {
		_, _, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

		col := tenant.Column{ID: uuid.New(), ModelID: uuid.New().String(), BaseID: uuid.New().String(), ColumnName: "sys", UIDT: "text", DT: helpers.StringPtr("TEXT"), System: true}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", col.ModelID).Return(tenant.Model{Alias: "tbl"}, nil).Maybe()

		_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{})

		assert.ErrorIs(t, err, app_errors.UpdateNotAllowed)
	})

	t.Run("system with title allowed", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

		modelID := uuid.New()
		colID := uuid.New()
		col := tenant.Column{ID: colID, ModelID: modelID.String(), BaseID: uuid.New().String(), ColumnName: "title", Title: "Title", UIDT: "text", DT: helpers.StringPtr("TEXT"), System: true}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", colID.String()).Return(col, nil)

		newTitle := "Name"
		newUIType := "number"
		newDT := "INTEGER"
		updatedCol := tenant.Column{ID: colID, ModelID: modelID.String(), BaseID: col.BaseID, ColumnName: "title", Title: newTitle, UIDT: newUIType, DT: helpers.StringPtr(newDT), System: true}
		mockColumn.On("UpdateColumn", mock.Anything, "schema", colID.String(), mock.Anything).Return(updatedCol, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{Alias: "tbl"}, nil)
		mockTable.On("AlterTableColumn", mock.Anything).Return(nil)
		mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

		updateReq := dto.ColumnUpdate{Title: &newTitle, UIDT: &newUIType}
		resp, err := svc.UpdateColumn(context.Background(), "schema", colID.String(), updateReq)

		assert.NoError(t, err)
		assert.Equal(t, newTitle, resp.Title)
		assert.Equal(t, newUIType, resp.UIDT)
	})

	t.Run("title column type update allowed", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

		modelID := uuid.New()
		colID := uuid.New()
		// Title column with System: false should allow full updates including type changes
		col := tenant.Column{ID: colID, ModelID: modelID.String(), BaseID: uuid.New().String(), ColumnName: "title", Title: "Title", UIDT: "text", DT: helpers.StringPtr("TEXT"), System: false}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", colID.String()).Return(col, nil)

		newTitle := "Name"
		newUIType := "number"
		newDT := "INTEGER"
		updatedCol := tenant.Column{ID: colID, ModelID: modelID.String(), BaseID: col.BaseID, ColumnName: "title", Title: newTitle, UIDT: newUIType, DT: helpers.StringPtr(newDT), System: false}
		mockColumn.On("UpdateColumn", mock.Anything, "schema", colID.String(), mock.Anything).Return(updatedCol, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID.String()).Return(tenant.Model{Alias: "tbl"}, nil)
		mockTable.On("AlterTableColumn", mock.Anything).Return(nil)
		mockTable.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

		updateReq := dto.ColumnUpdate{Title: &newTitle, UIDT: &newUIType}
		resp, err := svc.UpdateColumn(context.Background(), "schema", colID.String(), updateReq)

		assert.NoError(t, err)
		assert.Equal(t, newTitle, resp.Title)
		assert.Equal(t, newUIType, resp.UIDT)
	})

	t.Run("lookup update", func(t *testing.T) {
		_, _, _, mockModel, mockColumn, _, mockRel, _, svc := setupTableManagementService()

		modelID := uuid.New()
		baseID := uuid.New().String()
		relationID := uuid.New().String()
		lookupID := uuid.New().String()
		newLookupID := uuid.New().String()

		col := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID, ColumnName: "lk", UIDT: "lookup", Meta: map[string]interface{}{"lookup_column_id": lookupID, "relation_id": relationID}}
		lookupCol := tenant.Column{ID: uuid.New(), ModelID: uuid.New().String(), BaseID: uuid.New().String(), ColumnName: "src"}
		updatedCol := tenant.Column{ID: col.ID, ModelID: col.ModelID, BaseID: baseID, ColumnName: "lk", UIDT: "lookup", Meta: map[string]interface{}{"lookup_column_id": newLookupID, "relation_id": relationID}}
		newLookupCol := tenant.Column{ID: uuid.New(), ModelID: uuid.New().String(), BaseID: uuid.New().String(), ColumnName: "new"}

		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
		mockColumn.On("GetColumnByID", mock.Anything, "schema", lookupID).Return(lookupCol, nil)
		mockColumn.On("UpdateColumn", mock.Anything, "schema", col.ID.String(), mock.Anything).Return(updatedCol, nil)
		mockColumn.On("GetColumnByID", mock.Anything, "schema", newLookupID).Return(newLookupCol, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", newLookupCol.ModelID).Return(tenant.Model{Alias: "lk"}, nil)

		rel := tenant.Relation{ID: uuid.MustParse(relationID), SourceModelID: modelID.String()}
		mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(rel, nil)
		mockRel.On("UpdateRelation", mock.Anything, relationID, mock.Anything, "schema").Return(tenant.Relation{}, nil)

		updateMeta := map[string]interface{}{"lookup_column_id": newLookupID, "relation_id": relationID}
		_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: &updateMeta})

		assert.NoError(t, err)
	})

	t.Run("datatype change error triggers revert", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

		modelID := uuid.New().String()
		col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "c", UIDT: "text", DT: helpers.StringPtr("TEXT")}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)

		updatedCol := tenant.Column{ID: col.ID, ModelID: modelID, BaseID: col.BaseID, ColumnName: "c", UIDT: "number", DT: helpers.StringPtr("INTEGER")}
		mockColumn.On("UpdateColumn", mock.Anything, "schema", "cid", mock.Anything).Return(updatedCol, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(modelID)}, nil)

		mockTable.On("GetByFunction", mock.Anything, "public.convert_column_type", mock.Anything).Return(nil, errors.New("fail"))
		mockColumn.On("UpdateColumn", mock.Anything, "schema", "cid", mock.MatchedBy(func(req dto.ColumnUpdate) bool {
			return req.UIDT != nil && *req.UIDT == "text"
		})).Return(updatedCol, nil)

		uidt := "number"
		_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{UIDT: &uidt})

		assert.Error(t, err)
	})

	t.Run("datatype change success", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

		modelID := uuid.New().String()
		col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "c", UIDT: "text", DT: helpers.StringPtr("TEXT")}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)

		updatedCol := tenant.Column{ID: col.ID, ModelID: modelID, BaseID: col.BaseID, ColumnName: "c", UIDT: "number", DT: helpers.StringPtr("INTEGER")}
		mockColumn.On("UpdateColumn", mock.Anything, "schema", "cid", mock.Anything).Return(updatedCol, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(modelID)}, nil)

		mockTable.On("GetByFunction", mock.Anything, "public.convert_column_type", mock.Anything).Return([]map[string]interface{}{}, nil)

		uidt := "number"
		resp, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{UIDT: &uidt})

		assert.NoError(t, err)
		assert.Equal(t, updatedCol.ID, resp.ID)
	})
}

func TestDeleteColumn_Variants(t *testing.T) {
	t.Run("delete not allowed", func(t *testing.T) {
		_, _, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

		col := tenant.Column{ID: uuid.New(), ModelID: uuid.New().String(), BaseID: uuid.New().String(), ColumnName: "sys", UIDT: "text", System: true}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", col.ModelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(col.ModelID)}, nil).Maybe()

		err := svc.DeleteColumn(context.Background(), "schema", "cid")

		assert.ErrorIs(t, err, app_errors.DeleteNotAllowed)
	})

	t.Run("lookup delete", func(t *testing.T) {
		_, _, _, mockModel, mockColumn, _, mockRel, _, svc := setupTableManagementService()

		relationID := uuid.New().String()
		lookupID := uuid.New().String()
		modelID := uuid.New().String()
		col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "lk", UIDT: "lookup", Meta: map[string]interface{}{"lookup_column_id": lookupID, "relation_id": relationID}}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
		mockColumn.On("DeleteColumn", mock.Anything, "schema", col.ID.String()).Return(nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(modelID)}, nil)

		lookupCol := tenant.Column{ID: uuid.New(), ModelID: uuid.New().String(), BaseID: uuid.New().String(), ColumnName: "src"}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", lookupID).Return(lookupCol, nil)

		rel := tenant.Relation{ID: uuid.MustParse(relationID), SourceModelID: modelID}
		mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(rel, nil)
		mockRel.On("UpdateRelation", mock.Anything, relationID, mock.Anything, "schema").Return(tenant.Relation{}, nil)

		err := svc.DeleteColumn(context.Background(), "schema", "cid")

		assert.NoError(t, err)
	})

	t.Run("links delete", func(t *testing.T) {
		_, mockTable, _, mockModel, mockColumn, _, mockRel, _, svc := setupTableManagementService()

		relationID := uuid.New().String()
		sourceColID := uuid.New().String()
		targetColID := uuid.New().String()
		sourceModelID := uuid.New().String()
		targetModelID := uuid.New().String()

		col := tenant.Column{ID: uuid.MustParse(sourceColID), ModelID: sourceModelID, BaseID: uuid.New().String(), ColumnName: "src", UIDT: "links", Meta: map[string]interface{}{"relation_id": relationID, "entity_role": "source"}}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)

		rel := tenant.Relation{ID: uuid.MustParse(relationID), SourceColumnID: sourceColID, TargetColumnID: targetColID, SourceModelID: sourceModelID, TargetModelID: targetModelID, RelationType: "one-to-one"}
		mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(rel, nil)

		mockColumn.On("DeleteColumn", mock.Anything, "schema", sourceColID).Return(nil)
		mockColumn.On("DeleteColumn", mock.Anything, "schema", targetColID).Return(nil)
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", sourceModelID).Return([]tenant.Column{}, nil)
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", targetModelID).Return([]tenant.Column{}, nil)

		mockColumn.On("GetColumnByID", mock.Anything, "schema", targetColID).Return(tenant.Column{ID: uuid.MustParse(targetColID), ModelID: targetModelID, BaseID: uuid.New().String(), ColumnName: "tgt"}, nil)

		mockModel.On("GetModelByID", mock.Anything, "schema", sourceModelID).Return(tenant.Model{Alias: "src", ID: uuid.MustParse(sourceModelID)}, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", targetModelID).Return(tenant.Model{Alias: "tgt", ID: uuid.MustParse(targetModelID)}, nil)

		mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)

		err := svc.DeleteColumn(context.Background(), "schema", "cid")

		assert.NoError(t, err)
	})
}

func TestReorderColumn(t *testing.T) {
	_, _, _, _, mockColumn, _, _, _, svc := setupTableManagementService()

	order1 := 1.0
	order2 := 2.0
	sourceID := uuid.New()
	targetID := uuid.New()
	mockColumn.On("GetColumnByID", mock.Anything, "schema", sourceID.String()).Return(tenant.Column{ID: sourceID, ModelID: uuid.New().String(), BaseID: uuid.New().String(), OrderIndex: &order1}, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", targetID.String()).Return(tenant.Column{ID: targetID, ModelID: uuid.New().String(), BaseID: uuid.New().String(), OrderIndex: &order2}, nil)
	mockColumn.On("UpdateColumn", mock.Anything, "schema", sourceID.String(), mock.Anything).Return(tenant.Column{ID: sourceID, ModelID: uuid.New().String(), BaseID: uuid.New().String(), OrderIndex: &order2}, nil)
	mockColumn.On("UpdateColumn", mock.Anything, "schema", targetID.String(), mock.Anything).Return(tenant.Column{ID: targetID, ModelID: uuid.New().String(), BaseID: uuid.New().String(), OrderIndex: &order1}, nil)

	resp, err := svc.ReorderColumn(context.Background(), "schema", dto.ReorderColumnRequest{SourceColumnID: sourceID, TargetColumnID: targetID})

	assert.NoError(t, err)
	assert.Len(t, resp, 2)
}

func TestRowsAndLinks(t *testing.T) {
	t.Run("create and insert row data", func(t *testing.T) {
		stubTable := &StubTableService{}
		stubBulk := &StubBulkService{}
		mockModel := &MockModelService{}
		mockColumn := &MockColumnService{}
		mockView := &MockViewService{}
		mockRel := &MockRelationshipService{}
		mockAsset := &MockAssetManagementService{}

		modelID := uuid.New().String()
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl"}, nil)

		stubTable.CreateRecordFn = func(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"id": 1}, nil
		}
		stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"id": id}, nil
		}

		svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

		_, err := svc.CreateRow(context.Background(), "schema", dto.CreateRowRequest{ModelID: modelID, CreatedBy: "u"})
		assert.NoError(t, err)

		col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "arr", UIDT: "text", DT: helpers.StringPtr("INT[]")}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", "col").Return(col, nil)

		val := interface{}(5)
		_, err = svc.InsertRowData(context.Background(), "schema", dto.InsertRowDataRequest{ModelID: modelID, ColumnId: "col", RowId: 1, Value: &val, UpdatedBy: "u"})
		assert.NoError(t, err)
	})

	t.Run("update raw data for links", func(t *testing.T) {
		stubTable := &StubTableService{}
		stubBulk := &StubBulkService{}
		mockModel := &MockModelService{}
		mockColumn := &MockColumnService{}
		mockView := &MockViewService{}
		mockRel := &MockRelationshipService{}
		mockAsset := &MockAssetManagementService{}

		sourceModelID := uuid.New().String()
		targetModelID := uuid.New().String()
		relationID := uuid.New().String()
		columnID := uuid.New().String()
		targetColumnID := uuid.New().String()

		meta := map[string]interface{}{
			"relation_id": relationID,
			"entity_role": "source",
			"relation":    map[string]interface{}{"with": targetModelID, "type": "one-to-one"},
		}
		sourceCol := tenant.Column{ID: uuid.MustParse(columnID), ModelID: sourceModelID, BaseID: uuid.New().String(), ColumnName: "src_col", UIDT: "links", Meta: meta}
		targetCol := tenant.Column{ID: uuid.MustParse(targetColumnID), ModelID: targetModelID, BaseID: uuid.New().String(), ColumnName: "tgt_col", UIDT: "links"}

		mockColumn.On("GetColumnByID", mock.Anything, "schema", columnID).Return(sourceCol, nil)
		mockColumn.On("GetColumnByID", mock.Anything, "schema", targetColumnID).Return(targetCol, nil)

		mockModel.On("GetModelByID", mock.Anything, "schema", sourceModelID).Return(tenant.Model{Alias: "src"}, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", targetModelID).Return(tenant.Model{Alias: "tgt"}, nil)

		mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(tenant.Relation{ID: uuid.MustParse(relationID), SourceModelID: sourceModelID, TargetModelID: targetModelID, SourceColumnID: columnID, TargetColumnID: targetColumnID, RelationType: "one-to-one"}, nil)

		stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
			if len(params.Filters) > 0 {
				switch params.Filters[0].Column {
				case "id":
					return []map[string]interface{}{{"id": int64(1), "src_col": int64(2), "tgt_col": int64(2)}}, nil
				case "src_col", "tgt_col":
					return []map[string]interface{}{{"id": int64(99)}}, nil
				}
			}
			return []map[string]interface{}{}, nil
		}
		stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"id": id}, nil
		}

		svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

		_, err := svc.UpdateRawDataForLinks(context.Background(), "schema", dto.UpdateRowDataLinksRequest{
			ModelID:     sourceModelID,
			ColumnId:    columnID,
			SourceRowId: 1,
			TargetRowId: 2,
			Action:      "link",
			UpdatedBy:   "u",
		})
		assert.NoError(t, err)
	})
}

func TestAttachmentsAndBulkDelete(t *testing.T) {
	t.Run("add and remove attachments", func(t *testing.T) {
		stubTable := &StubTableService{}
		stubBulk := &StubBulkService{}
		mockModel := &MockModelService{}
		mockColumn := &MockColumnService{}
		mockView := &MockViewService{}
		mockRel := &MockRelationshipService{}
		mockAsset := &MockAssetManagementService{}

		modelID := uuid.New().String()
		colID := "col"
		col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "attachments", UIDT: "attachment"}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", colID).Return(col, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl"}, nil)

		assetID := uuid.New().String()
		mockAsset.On("Upload", mock.Anything, mock.Anything, "schema").Return([]tenant.Assets{{ID: uuid.MustParse(assetID)}}, nil)

		stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
			return []map[string]interface{}{{"id": 1, "attachments": []map[string]interface{}{{"id": assetID, "name": "a"}}}}, nil
		}
		stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"id": id}, nil
		}

		svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

		_, err := svc.AddAttachment(context.Background(), "schema", dto.AddAttachmentRequest{ModelID: modelID, ColumnId: colID, RowId: 1}, nil)
		assert.NoError(t, err)

		_, err = svc.RemoveAttachments(context.Background(), "schema", dto.RemoveAttachmentsRequest{ModelID: modelID, ColumnId: colID, RowId: 1, Attachments: []string{assetID}})
		assert.NoError(t, err)
	})

	t.Run("update attachment with []interface{} payload", func(t *testing.T) {
		stubTable := &StubTableService{}
		stubBulk := &StubBulkService{}
		mockModel := &MockModelService{}
		mockColumn := &MockColumnService{}
		mockView := &MockViewService{}
		mockRel := &MockRelationshipService{}
		mockAsset := &MockAssetManagementService{}

		modelID := uuid.New().String()
		colID := "col"
		assetID := uuid.New()

		col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "attachments", UIDT: "attachment"}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", colID).Return(col, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl"}, nil)

		mockAsset.On("UpdateAsset", mock.Anything, assetID.String(), mock.Anything, "schema").Return(tenant.Assets{
			ID:    assetID,
			Title: "updated-file",
			Url:   "https://cdn.example/new",
		}, nil)

		stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
			return []map[string]interface{}{
				{
					"id": 1,
					"attachments": []interface{}{
						map[string]interface{}{
							"id":    assetID.String(),
							"title": "old-file",
							"url":   "https://cdn.example/old",
						},
						map[string]interface{}{
							"id":    uuid.New().String(),
							"title": "keep-file",
							"url":   "https://cdn.example/keep",
						},
					},
				},
			}, nil
		}
		stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"id": id, "attachments": data["attachments"]}, nil
		}

		svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)
		updatedTitle := "updated-file"

		resp, err := svc.UpdateAttachment(context.Background(), "schema", dto.UpdateAttachmentRequest{
			ModelID:  modelID,
			ColumnId: colID,
			RowId:    1,
			AssetId:  assetID.String(),
			Content: dto.AssetUpdate{
				Title: &updatedTitle,
			},
		})
		assert.NoError(t, err)
		attachments, ok := resp.Record["attachments"].([]map[string]interface{})
		assert.True(t, ok)
		assert.Len(t, attachments, 2)

		var updated map[string]interface{}
		for _, a := range attachments {
			if id, ok := a["id"].(uuid.UUID); ok && id == assetID {
				updated = a
				break
			}
			if id, ok := a["id"].(string); ok && id == assetID.String() {
				updated = a
				break
			}
		}
		assert.NotNil(t, updated)
		assert.Equal(t, "updated-file", updated["title"])
	})

	t.Run("update attachment returns database error when row update fails", func(t *testing.T) {
		stubTable := &StubTableService{}
		stubBulk := &StubBulkService{}
		mockModel := &MockModelService{}
		mockColumn := &MockColumnService{}
		mockView := &MockViewService{}
		mockRel := &MockRelationshipService{}
		mockAsset := &MockAssetManagementService{}

		modelID := uuid.New().String()
		colID := "col"
		assetID := uuid.New()

		col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "attachments", UIDT: "attachment"}
		mockColumn.On("GetColumnByID", mock.Anything, "schema", colID).Return(col, nil)
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl"}, nil)
		mockAsset.On("UpdateAsset", mock.Anything, assetID.String(), mock.Anything, "schema").Return(tenant.Assets{ID: assetID, Title: "updated-file"}, nil)

		stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
			return []map[string]interface{}{
				{"id": 1, "attachments": []map[string]interface{}{{"id": assetID.String(), "title": "old-file"}}},
			}, nil
		}
		stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
			return nil, errors.New("update failed")
		}

		svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)
		updatedTitle := "updated-file"

		_, err := svc.UpdateAttachment(context.Background(), "schema", dto.UpdateAttachmentRequest{
			ModelID:  modelID,
			ColumnId: colID,
			RowId:    1,
			AssetId:  assetID.String(),
			Content:  dto.AssetUpdate{Title: &updatedTitle},
		})
		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("bulk delete rows", func(t *testing.T) {
		stubTable := &StubTableService{}
		stubBulk := &StubBulkService{}
		mockModel := &MockModelService{}
		mockColumn := &MockColumnService{}
		mockView := &MockViewService{}
		mockRel := &MockRelationshipService{}
		mockAsset := &MockAssetManagementService{}

		modelID := uuid.New().String()
		mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(modelID)}, nil)
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{}, nil)

		stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
			if len(params.Filters) > 0 && params.Filters[0].Value == 1 {
				return []map[string]interface{}{{"id": int64(1)}}, nil
			}
			return []map[string]interface{}{}, nil
		}
		stubBulk.BulkDeleteFn = func(tableName string, ids []interface{}, idColumn string) (int64, error) {
			return int64(len(ids)), nil
		}

		svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

		count, err := svc.BulkDeleteRows(context.Background(), "schema", dto.BulkDeleteRowsRequest{ModelID: modelID, RowIds: []int{1, 2}})
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestGetRecordsWithLookups_Relations(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	relationID := uuid.New().String()
	lookupRelID := relationID
	otherModelID := uuid.New().String()

	columnsData := []dto.ColumnResponse{
		{UIDT: "lookup", Meta: map[string]interface{}{"relation_id": lookupRelID}},
		{UIDT: "links", ColumnName: "link_col", Meta: map[string]interface{}{"relation_id": relationID, "entity_role": "source"}},
	}

	mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(tenant.Relation{RelationType: "one-to-one", SourceLookupColumns: []string{"name"}, TargetModelID: otherModelID}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", otherModelID).Return(tenant.Model{Alias: "target"}, nil)

	stubTable.GetByFunctionFn = func(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"get_table_data_with_relation": []map[string]interface{}{{"id": 1}}}}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	getRecords, ok := svc.(interface {
		GetRecordsWithLookups(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error)
	})
	assert.True(t, ok)

	records, err := getRecords.GetRecordsWithLookups(context.Background(), "schema", "tbl", columnsData)

	assert.NoError(t, err)
	assert.Len(t, records.Records, 1)
}

func TestHelperBehavior_Misc(t *testing.T) {
	t.Run("get database type invalid driver", func(t *testing.T) {
		// use real service with invalid driver to hit InvalidDriver via AddColumn
		mockTable := &MockTableService{}
		mockBulk := &MockBulkService{}
		mockModel := &MockModelService{}
		mockColumn := &MockColumnService{}
		mockView := &MockViewService{}
		mockRel := &MockRelationshipService{}
		mockAsset := &MockAssetManagementService{}
		db := &pkg.DatabaseService{TableService: mockTable, BulkService: mockBulk}
		svc := services.NewTableManagementService("unknown", db, mockModel, mockColumn, mockView, mockRel, mockAsset)

		_, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{ModelID: uuid.New(), BaseID: uuid.New(), Title: "t", UIDT: "text"})
		assert.ErrorIs(t, err, app_errors.InvalidDriver)
	})

	t.Run("normalize records missing key", func(t *testing.T) {
		stubTable := &StubTableService{}
		stubBulk := &StubBulkService{}
		mockModel := &MockModelService{}
		mockColumn := &MockColumnService{}
		mockView := &MockViewService{}
		mockRel := &MockRelationshipService{}
		mockAsset := &MockAssetManagementService{}

		stubTable.GetByFunctionFn = func(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error) {
			return []map[string]interface{}{{"other": "x"}}, nil
		}

		svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

		getRecords, ok := svc.(interface {
			GetRecordsWithLookups(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error)
		})
		assert.True(t, ok)

		records, err := getRecords.GetRecordsWithLookups(context.Background(), "schema", "tbl", []dto.ColumnResponse{})
		assert.NoError(t, err)
		assert.Empty(t, records.Records)
	})
}

func TestUpdateRawDataForLinks_HasManyVariants(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	sourceModelID := uuid.New().String()
	targetModelID := uuid.New().String()
	relationID := uuid.New().String()
	columnID := uuid.New().String()
	targetColumnID := uuid.New().String()

	meta := map[string]interface{}{
		"relation_id": relationID,
		"entity_role": "source",
		"relation":    map[string]interface{}{"with": targetModelID, "type": "has-many"},
	}
	sourceCol := tenant.Column{ID: uuid.MustParse(columnID), ModelID: sourceModelID, BaseID: uuid.New().String(), ColumnName: "src_col", UIDT: "links", Meta: meta}
	targetCol := tenant.Column{ID: uuid.MustParse(targetColumnID), ModelID: targetModelID, BaseID: uuid.New().String(), ColumnName: "tgt_col", UIDT: "links"}

	mockColumn.On("GetColumnByID", mock.Anything, "schema", columnID).Return(sourceCol, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", targetColumnID).Return(targetCol, nil)

	mockModel.On("GetModelByID", mock.Anything, "schema", sourceModelID).Return(tenant.Model{Alias: "src"}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", targetModelID).Return(tenant.Model{Alias: "tgt"}, nil)

	mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(tenant.Relation{ID: uuid.MustParse(relationID), SourceModelID: sourceModelID, TargetModelID: targetModelID, SourceColumnID: columnID, TargetColumnID: targetColumnID, RelationType: "has-many"}, nil)

	stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
		if len(params.Filters) == 0 {
			return []map[string]interface{}{}, nil
		}
		filter := params.Filters[0]
		if filter.Operator == "any" {
			return []map[string]interface{}{}, nil
		}
		if filter.Column == "id" {
			if id, ok := filter.Value.(int); ok {
				switch id {
				case 1:
					return []map[string]interface{}{{"id": int64(1), "src_col": nil, "tgt_col": int64(2)}}, nil
				case 2:
					return []map[string]interface{}{{"id": int64(2), "src_col": []int64{1}, "tgt_col": int64(2)}}, nil
				case 3:
					return []map[string]interface{}{{"id": int64(3), "src_col": []string{"3"}, "tgt_col": int64(2)}}, nil
				case 4:
					return []map[string]interface{}{{"id": int64(4), "src_col": int64(4), "tgt_col": int64(2)}}, nil
				case 5:
					return []map[string]interface{}{{"id": int64(5), "src_col": 5, "tgt_col": int64(2)}}, nil
				case 6:
					return []map[string]interface{}{{"id": int64(6), "src_col": map[string]interface{}{"x": 1}, "tgt_col": int64(2)}}, nil
				}
			}
		}
		return []map[string]interface{}{}, nil
	}

	stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": id}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	for i := 1; i <= 6; i++ {
		_, err := svc.UpdateRawDataForLinks(context.Background(), "schema", dto.UpdateRowDataLinksRequest{
			ModelID:     sourceModelID,
			ColumnId:    columnID,
			SourceRowId: i,
			TargetRowId: 2,
			Action:      "link",
			UpdatedBy:   "u",
		})
		assert.NoError(t, err)
	}
}

func TestUpdateRawDataForLinks_HasManyExisting(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	sourceModelID := uuid.New().String()
	targetModelID := uuid.New().String()
	relationID := uuid.New().String()
	columnID := uuid.New().String()
	targetColumnID := uuid.New().String()

	meta := map[string]interface{}{
		"relation_id": relationID,
		"entity_role": "source",
		"relation":    map[string]interface{}{"with": targetModelID, "type": "has-many"},
	}
	sourceCol := tenant.Column{ID: uuid.MustParse(columnID), ModelID: sourceModelID, BaseID: uuid.New().String(), ColumnName: "src_col", UIDT: "links", Meta: meta}
	targetCol := tenant.Column{ID: uuid.MustParse(targetColumnID), ModelID: targetModelID, BaseID: uuid.New().String(), ColumnName: "tgt_col", UIDT: "links"}

	mockColumn.On("GetColumnByID", mock.Anything, "schema", columnID).Return(sourceCol, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", targetColumnID).Return(targetCol, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", sourceModelID).Return(tenant.Model{Alias: "src"}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", targetModelID).Return(tenant.Model{Alias: "tgt"}, nil)

	mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(tenant.Relation{ID: uuid.MustParse(relationID), SourceModelID: sourceModelID, TargetModelID: targetModelID, SourceColumnID: columnID, TargetColumnID: targetColumnID, RelationType: "has-many"}, nil)

	stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
		if len(params.Filters) == 0 {
			return []map[string]interface{}{}, nil
		}
		filter := params.Filters[0]
		if filter.Operator == "any" {
			return []map[string]interface{}{{"id": int64(9), "src_col": []string{"2"}}}, nil
		}
		if filter.Column == "id" {
			return []map[string]interface{}{{"id": int64(1), "src_col": []string{"2", "3"}, "tgt_col": int64(2)}}, nil
		}
		return []map[string]interface{}{}, nil
	}
	stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": id}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	_, err := svc.UpdateRawDataForLinks(context.Background(), "schema", dto.UpdateRowDataLinksRequest{
		ModelID:     sourceModelID,
		ColumnId:    columnID,
		SourceRowId: 1,
		TargetRowId: 2,
		Action:      "link",
		UpdatedBy:   "u",
	})
	assert.NoError(t, err)
}

func TestDeleteRow_WithLinks(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	sourceModelID := uuid.New().String()
	targetModelID := uuid.New().String()
	relationID := uuid.New().String()
	columnID := uuid.New().String()
	targetColumnID := uuid.New().String()

	linkMeta := map[string]interface{}{
		"relation_id": relationID,
		"entity_role": "source",
		"relation":    map[string]interface{}{"with": targetModelID, "type": "one-to-one"},
	}
	linkCol := tenant.Column{ID: uuid.MustParse(columnID), ModelID: sourceModelID, BaseID: uuid.New().String(), ColumnName: "src_col", UIDT: "links", Meta: linkMeta}
	targetCol := tenant.Column{ID: uuid.MustParse(targetColumnID), ModelID: targetModelID, BaseID: uuid.New().String(), ColumnName: "tgt_col", UIDT: "links"}

	mockModel.On("GetModelByID", mock.Anything, "schema", sourceModelID).Return(tenant.Model{Alias: "src", ID: uuid.MustParse(sourceModelID)}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", targetModelID).Return(tenant.Model{Alias: "tgt"}, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", sourceModelID).Return([]tenant.Column{linkCol}, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", targetColumnID).Return(targetCol, nil)

	mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(tenant.Relation{ID: uuid.MustParse(relationID), SourceModelID: sourceModelID, TargetModelID: targetModelID, SourceColumnID: columnID, TargetColumnID: targetColumnID, RelationType: "one-to-one"}, nil)

	stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
		if len(params.Filters) > 0 && params.Filters[0].Column == "id" {
			return []map[string]interface{}{{"id": int64(1), "src_col": int64(2)}}, nil
		}
		return []map[string]interface{}{}, nil
	}
	stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": id}, nil
	}
	stubTable.DeleteRecordFn = func(tableName string, id interface{}) error { return nil }

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	err := svc.DeleteRow(context.Background(), "schema", dto.DeleteRowDataRequest{ModelID: sourceModelID, RowId: 1})
	assert.NoError(t, err)
}

func TestDeleteTable_WithColumnsAndViews(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

	modelID := uuid.New().String()
	model := tenant.Model{ID: uuid.MustParse(modelID), Alias: "tbl"}
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)

	col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "c", UIDT: "text"}
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{col}, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", col.ID.String()).Return(nil)

	view := tenant.View{ID: uuid.New(), ModelID: modelID}
	mockView.On("GetViewsByModelID", mock.Anything, "schema", modelID).Return([]tenant.View{view}, nil)
	mockView.On("DeleteView", mock.Anything, "schema", view.ID.String()).Return(nil)

	mockModel.On("DeleteModel", mock.Anything, "schema", modelID).Return(nil)
	mockTable.On("DropTable", mock.Anything, mock.Anything).Return(nil)

	err := svc.DeleteTable(context.Background(), "schema", modelID)
	assert.NoError(t, err)
}

func TestDeleteColumn_NonLookup(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

	modelID := uuid.New().String()
	order := 1.0
	col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "c", UIDT: "text", DT: helpers.StringPtr("TEXT"), OrderIndex: &order}
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", "cid").Return(nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(modelID)}, nil)
	mockTable.On("GetByFunction", mock.Anything, "public.reorder_columns_after_delete", mock.Anything).Return([]map[string]interface{}{}, nil)
	mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)

	err := svc.DeleteColumn(context.Background(), "schema", "cid")
	assert.NoError(t, err)
}

func TestCreateRowsAndGetAllRecords(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	modelID := uuid.New().String()
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(modelID)}, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{}, nil)

	stubTable.CreateRecordFn = func(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": 1}, nil
	}
	stubBulk.BulkInsertFn = func(tableName string, records []map[string]interface{}) ([]map[string]interface{}, error) {
		return records, nil
	}
	stubTable.GetByFunctionFn = func(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"get_table_data_with_relation": []map[string]interface{}{{"id": 1}}}}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	_, err := svc.CreateRowWithRecords(context.Background(), "schema", "tbl", map[string]interface{}{"id": 1})
	assert.NoError(t, err)

	_, err = svc.CreateRowsWithRecordsBulk(context.Background(), "schema", "tbl", []map[string]interface{}{{"id": 1}, {"id": 2}})
	assert.NoError(t, err)

	_, err = svc.GetAllRecords(context.Background(), "schema", modelID)
	assert.NoError(t, err)
}

func TestDeleteTable_DropError(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

	modelID := uuid.New().String()
	model := tenant.Model{ID: uuid.MustParse(modelID), Alias: "tbl"}
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{}, nil)
	mockView.On("GetViewsByModelID", mock.Anything, "schema", modelID).Return([]tenant.View{}, nil)
	mockModel.On("DeleteModel", mock.Anything, "schema", modelID).Return(nil)
	mockTable.On("DropTable", mock.Anything, mock.Anything).Return(errors.New("fail"))

	err := svc.DeleteTable(context.Background(), "schema", modelID)
	assert.Error(t, err)
}

func TestAddColumn_InvalidMetaCases(t *testing.T) {
	_, _, _, _, _, _, _, _, svc := setupTableManagementService()

	modelID := uuid.New()
	baseID := uuid.New()

	cases := []struct {
		name string
		meta map[string]interface{}
		uidt string
	}{
		{"links missing relation", map[string]interface{}{}, "links"},
		{"links bad with", map[string]interface{}{"relation": map[string]interface{}{"with": 123, "type": "one-to-one"}}, "links"},
		{"links bad uuid", map[string]interface{}{"relation": map[string]interface{}{"with": "bad", "type": "one-to-one"}}, "links"},
		{"links bad type", map[string]interface{}{"relation": map[string]interface{}{"with": uuid.New().String(), "type": "invalid"}}, "links"},
		{"lookup missing ids", map[string]interface{}{}, "lookup"},
		{"lookup bad uuid", map[string]interface{}{"lookup_column_id": "bad", "relation_id": "bad"}, "lookup"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.AddColumn(context.Background(), "schema", dto.AddColumnRequest{ModelID: modelID, BaseID: baseID, Title: "X", UIDT: tc.uidt, Meta: tc.meta})
			assert.Error(t, err)
		})
	}
}

func TestGetRecordsWithLookups_TargetRole(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	relationID := uuid.New().String()
	lookupRelID := relationID
	sourceModelID := uuid.New().String()

	columnsData := []dto.ColumnResponse{
		{UIDT: "lookup", Meta: map[string]interface{}{"relation_id": lookupRelID}},
		{UIDT: "links", ColumnName: "link_col", Meta: map[string]interface{}{"relation_id": relationID, "entity_role": "target"}},
	}

	mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(tenant.Relation{RelationType: "one-to-one", TargetLookupColumns: []string{"title"}, SourceModelID: sourceModelID}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", sourceModelID).Return(tenant.Model{Alias: "source"}, nil)

	stubTable.GetByFunctionFn = func(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"get_table_data_with_relation": []map[string]interface{}{{"id": 1}}}}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	getRecords, ok := svc.(interface {
		GetRecordsWithLookups(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error)
	})
	assert.True(t, ok)

	records, err := getRecords.GetRecordsWithLookups(context.Background(), "schema", "tbl", columnsData)
	assert.NoError(t, err)
	assert.Len(t, records.Records, 1)
}

func TestInsertRowData_SystemColumn(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	modelID := uuid.New().String()
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl"}, nil)

	col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "created_by", UIDT: "text", System: true}
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "col").Return(col, nil)

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	val := interface{}("x")
	_, err := svc.InsertRowData(context.Background(), "schema", dto.InsertRowDataRequest{ModelID: modelID, ColumnId: "col", RowId: 1, Value: &val, UpdatedBy: "u"})
	assert.ErrorIs(t, err, app_errors.UpdateNotAllowed)
}

func TestDeleteTable_WithLinkColumn(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, mockView, mockRel, _, svc := setupTableManagementService()

	modelID := uuid.New().String()
	targetModelID := uuid.New().String()
	relationID := uuid.New().String()
	sourceColID := uuid.New().String()
	targetColID := uuid.New().String()

	model := tenant.Model{ID: uuid.MustParse(modelID), Alias: "tbl"}
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)

	linkCol := tenant.Column{ID: uuid.MustParse(sourceColID), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "link", UIDT: "links", Meta: map[string]interface{}{"relation_id": relationID, "entity_role": "source"}}
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{linkCol}, nil)

	mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(tenant.Relation{ID: uuid.MustParse(relationID), SourceColumnID: sourceColID, TargetColumnID: targetColID, SourceModelID: modelID, TargetModelID: targetModelID, RelationType: "one-to-one"}, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", sourceColID).Return(nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", targetColID).Return(tenant.Column{ID: uuid.MustParse(targetColID), ModelID: targetModelID, BaseID: uuid.New().String(), ColumnName: "tgt"}, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", targetColID).Return(nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{}, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", targetModelID).Return([]tenant.Column{}, nil)

	mockModel.On("GetModelByID", mock.Anything, "schema", targetModelID).Return(tenant.Model{Alias: "tgt", ID: uuid.MustParse(targetModelID)}, nil)
	mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)

	mockView.On("GetViewsByModelID", mock.Anything, "schema", modelID).Return([]tenant.View{}, nil)
	mockModel.On("DeleteModel", mock.Anything, "schema", modelID).Return(nil)
	mockTable.On("DropTable", mock.Anything, mock.Anything).Return(nil)

	err := svc.DeleteTable(context.Background(), "schema", modelID)
	assert.NoError(t, err)
}

func TestConvertToInt64Array_Variants(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	sourceModelID := uuid.New().String()
	targetModelID := uuid.New().String()
	relationID := uuid.New().String()
	columnID := uuid.New().String()
	targetColumnID := uuid.New().String()

	meta := map[string]interface{}{
		"relation_id": relationID,
		"entity_role": "source",
		"relation":    map[string]interface{}{"with": targetModelID, "type": "has-many"},
	}
	sourceCol := tenant.Column{ID: uuid.MustParse(columnID), ModelID: sourceModelID, BaseID: uuid.New().String(), ColumnName: "src_col", UIDT: "links", Meta: meta}
	targetCol := tenant.Column{ID: uuid.MustParse(targetColumnID), ModelID: targetModelID, BaseID: uuid.New().String(), ColumnName: "tgt_col", UIDT: "links"}

	mockColumn.On("GetColumnByID", mock.Anything, "schema", columnID).Return(sourceCol, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", targetColumnID).Return(targetCol, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", sourceModelID).Return(tenant.Model{Alias: "src"}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", targetModelID).Return(tenant.Model{Alias: "tgt"}, nil)
	mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(tenant.Relation{ID: uuid.MustParse(relationID), SourceModelID: sourceModelID, TargetModelID: targetModelID, SourceColumnID: columnID, TargetColumnID: targetColumnID, RelationType: "has-many"}, nil)

	stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
		if len(params.Filters) > 0 && params.Filters[0].Column == "id" {
			return []map[string]interface{}{{"id": int64(1), "src_col": []int{1, 2}, "tgt_col": int64(2)}}, nil
		}
		return []map[string]interface{}{}, nil
	}
	stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": id}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	_, err := svc.UpdateRawDataForLinks(context.Background(), "schema", dto.UpdateRowDataLinksRequest{
		ModelID:     sourceModelID,
		ColumnId:    columnID,
		SourceRowId: 1,
		TargetRowId: 2,
		Action:      "unlink",
		UpdatedBy:   "u",
	})
	assert.NoError(t, err)
}

func TestAllowInsert_TitleSystemColumn(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	modelID := uuid.New().String()
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl"}, nil)

	col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "title", UIDT: "text", System: true}
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "col").Return(col, nil)

	stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": id}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	val := interface{}("ok")
	_, err := svc.InsertRowData(context.Background(), "schema", dto.InsertRowDataRequest{ModelID: modelID, ColumnId: "col", RowId: 1, Value: &val, UpdatedBy: "u"})
	assert.NoError(t, err)
}

func TestCheckAttachmentType_Variants(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	modelID := uuid.New().String()
	colID := "col"
	col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.New().String(), ColumnName: "attachments", UIDT: "attachment"}
	mockColumn.On("GetColumnByID", mock.Anything, "schema", colID).Return(col, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl"}, nil)

	stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{
			"id":          1,
			"attachments": []interface{}{map[string]interface{}{"id": "a"}, "skip"},
		}}, nil
	}
	stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": id}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	_, err := svc.RemoveAttachments(context.Background(), "schema", dto.RemoveAttachmentsRequest{ModelID: modelID, ColumnId: colID, RowId: 1, Attachments: []string{"a"}})
	assert.NoError(t, err)
}

func TestUpdateColumnForLookup_TargetBranch(t *testing.T) {
	_, _, _, mockModel, mockColumn, _, mockRel, _, svc := setupTableManagementService()

	modelID := uuid.New()
	baseID := uuid.New().String()
	relationID := uuid.New().String()
	lookupID := uuid.New().String()

	col := tenant.Column{ID: uuid.New(), ModelID: modelID.String(), BaseID: baseID, ColumnName: "lk", UIDT: "lookup", Meta: map[string]interface{}{"lookup_column_id": lookupID, "relation_id": relationID}}
	lookupCol := tenant.Column{ID: uuid.New(), ModelID: uuid.New().String(), BaseID: uuid.New().String(), ColumnName: "src"}
	updatedCol := tenant.Column{ID: col.ID, ModelID: col.ModelID, BaseID: baseID, ColumnName: "lk", UIDT: "lookup", Meta: map[string]interface{}{"lookup_column_id": lookupID, "relation_id": relationID}}

	mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", lookupID).Return(lookupCol, nil)
	mockColumn.On("UpdateColumn", mock.Anything, "schema", col.ID.String(), mock.Anything).Return(updatedCol, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", lookupCol.ModelID).Return(tenant.Model{Alias: "lk"}, nil)

	rel := tenant.Relation{ID: uuid.MustParse(relationID), TargetModelID: modelID.String(), TargetLookupColumns: []string{"src", "other"}}
	mockRel.On("GetRelationByID", mock.Anything, relationID, "schema").Return(rel, nil)
	mockRel.On("UpdateRelation", mock.Anything, relationID, mock.Anything, "schema").Return(tenant.Relation{}, nil)

	updateMeta := map[string]interface{}{"lookup_column_id": lookupID, "relation_id": relationID}
	_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{Meta: &updateMeta})
	assert.NoError(t, err)
}

func TestUpdateColumnForLink(t *testing.T) {
	_, _, _, _, mockColumn, _, _, _, svc := setupTableManagementService()

	modelID := uuid.New()
	baseID := uuid.New().String()
	columnID := uuid.New()

	newTitle := "Updated Link Title"
	newDescription := "Updated Link Description"
	updatedBy := "test-user"

	// Create a link column
	col := tenant.Column{
		ID:         columnID,
		ModelID:    modelID.String(),
		BaseID:     baseID,
		ColumnName: "link_col",
		UIDT:       "link",
		Title:      "Original Title",
		Meta:       map[string]interface{}{"relation_id": uuid.New().String()},
	}

	// Expected updated column with only title, description, and metadata fields changed
	updatedCol := tenant.Column{
		ID:          columnID,
		ModelID:     modelID.String(),
		BaseID:      baseID,
		ColumnName:  "link_col",
		UIDT:        "link",
		Title:       newTitle,
		Description: &newDescription,
		Meta:        col.Meta, // Meta should remain unchanged for link updates
	}

	mockColumn.On("GetColumnByID", mock.Anything, "schema", columnID.String()).Return(col, nil)
	mockColumn.On("UpdateColumn", mock.Anything, "schema", col.ID.String(), mock.MatchedBy(func(req dto.ColumnUpdate) bool {
		// Verify that only title, description, updatedBy, and updatedAt are being updated
		return req.Title != nil && *req.Title == newTitle &&
			req.Description != nil && *req.Description == newDescription &&
			req.UpdatedBy == updatedBy &&
			!req.UpdatedAt.IsZero() &&
			req.Meta == nil && // Meta should not be updated
			req.UIDT == nil && // UIDT should not be updated
			req.DT == nil // DT should not be updated
	})).Return(updatedCol, nil)

	updateReq := dto.ColumnUpdate{
		Title:       helpers.StringPtr(newTitle),
		Description: helpers.StringPtr(newDescription),
		UpdatedBy:   updatedBy,
		Meta:        &map[string]interface{}{"some": "data"}, // This should be ignored for link columns
	}

	resp, err := svc.UpdateColumn(context.Background(), "schema", columnID.String(), updateReq)

	assert.NoError(t, err)
	assert.Equal(t, newTitle, resp.Title)
	assert.Equal(t, newDescription, resp.Description)
	mockColumn.AssertExpectations(t)
}

func TestGetTableByID_ErrorPaths(t *testing.T) {
	_, _, _, mockModel, mockColumn, mockView, _, _, svc := setupTableManagementService()

	mockModel.On("GetModelByID", mock.Anything, "schema", "id").Return(tenant.Model{}, errors.New("fail"))
	_, err := svc.GetTableByID(context.Background(), "id", "schema")
	assert.Error(t, err)

	model := tenant.Model{ID: uuid.New(), Alias: "tbl"}
	mockModel.On("GetModelByID", mock.Anything, "schema", "id2").Return(model, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", "id2").Return([]tenant.Column{}, errors.New("fail"))
	_, err = svc.GetTableByID(context.Background(), "id2", "schema")
	assert.Error(t, err)

	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", "id3").Return([]tenant.Column{}, nil)
	mockView.On("GetViewsByModelID", mock.Anything, "schema", "id3").Return([]tenant.View{}, errors.New("fail"))
	mockModel.On("GetModelByID", mock.Anything, "schema", "id3").Return(model, nil)
	_, err = svc.GetTableByID(context.Background(), "id3", "schema")
	assert.Error(t, err)
}

// Tests for BulkUpdateColumns
func TestBulkUpdateColumns_EmptyUpdates(t *testing.T) {
	_, _, _, _, _, _, _, _, svc := setupTableManagementService()

	err := svc.BulkUpdateColumns(context.Background(), "schema", "modelID", "columnID", []dto.UpdateColumnsRequest{})

	assert.NoError(t, err)
}

func TestBulkUpdateColumns_GetModelError(t *testing.T) {
	_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

	mockModel.On("GetModelByID", mock.Anything, "schema", "modelID").Return(tenant.Model{}, errors.New("model not found"))

	err := svc.BulkUpdateColumns(context.Background(), "schema", "modelID", "columnID", []dto.UpdateColumnsRequest{{Id: "row1", Value: "val"}})

	assert.Error(t, err)
}

func TestBulkUpdateColumns_GetColumnError(t *testing.T) {
	_, _, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

	model := tenant.Model{ID: uuid.New(), Alias: "tbl"}
	mockModel.On("GetModelByID", mock.Anything, "schema", "modelID").Return(model, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "columnID").Return(tenant.Column{}, errors.New("column not found"))

	err := svc.BulkUpdateColumns(context.Background(), "schema", "modelID", "columnID", []dto.UpdateColumnsRequest{{Id: "row1", Value: "val"}})

	assert.Error(t, err)
}

// Tests for ResetColumnValues
func TestResetColumnValues_GetModelError(t *testing.T) {
	_, _, _, mockModel, _, _, _, _, svc := setupTableManagementService()

	mockModel.On("GetModelByID", mock.Anything, "schema", "modelID").Return(tenant.Model{}, errors.New("model not found"))

	err := svc.ResetColumnValues(context.Background(), "schema", "modelID", "columnID")

	assert.Error(t, err)
}

func TestResetColumnValues_GetColumnError(t *testing.T) {
	_, _, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

	model := tenant.Model{ID: uuid.New(), Alias: "tbl"}
	mockModel.On("GetModelByID", mock.Anything, "schema", "modelID").Return(model, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "columnID").Return(tenant.Column{}, errors.New("column not found"))

	err := svc.ResetColumnValues(context.Background(), "schema", "modelID", "columnID")

	assert.Error(t, err)
}

func TestCreateRowsWithValues_EmptyRows(t *testing.T) {
	_, _, _, _, _, _, _, _, svc := setupTableManagementService()

	rows, err := svc.(interface {
		CreateRowsWithValues(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error)
	}).CreateRowsWithValues(context.Background(), "schema", "modelID", []map[string]interface{}{}, "user", "user")

	assert.NoError(t, err)
	assert.Len(t, rows, 0)
}

func TestCreateRowsWithValues_Success(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	modelID := uuid.New().String()
	baseID := uuid.New().String()
	colTextID := uuid.New().String()
	colArrayID := uuid.New().String()
	model := tenant.Model{ID: uuid.MustParse(modelID), Alias: "tbl"}

	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", colTextID).
		Return(tenant.Column{ID: uuid.MustParse(colTextID), ModelID: modelID, BaseID: baseID, ColumnName: "text_col", UIDT: "text", DT: helpers.StringPtr("TEXT")}, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", colArrayID).
		Return(tenant.Column{ID: uuid.MustParse(colArrayID), ModelID: modelID, BaseID: baseID, ColumnName: "array_col", UIDT: "text", DT: helpers.StringPtr("TEXT[]")}, nil)

	nextID := 0
	stubTable.CreateRecordFn = func(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
		nextID++
		assert.Equal(t, "\"schema\".\"tbl\"", tableName)
		assert.Equal(t, "creator", data["created_by"])
		return map[string]interface{}{"id": nextID}, nil
	}
	stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
		assert.Equal(t, "\"schema\".\"tbl\"", tableName)
		assert.Equal(t, "editor", data["last_modified_by"])
		return map[string]interface{}{"id": id, "data": data}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	rows, err := svc.(interface {
		CreateRowsWithValues(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error)
	}).CreateRowsWithValues(context.Background(), "schema", modelID, []map[string]interface{}{
		{colTextID: "alpha", colArrayID: "tag"},
		{},
	}, "creator", "editor")

	assert.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, 2, nextID)
	mockModel.AssertExpectations(t)
	mockColumn.AssertExpectations(t)
}

func TestCreateRowsWithValues_CreateRowError(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	modelID := uuid.New().String()
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{}, errors.New("model failed"))

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	rows, err := svc.(interface {
		CreateRowsWithValues(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error)
	}).CreateRowsWithValues(context.Background(), "schema", modelID, []map[string]interface{}{{"col": "value"}}, "creator", "editor")

	assert.Error(t, err)
	assert.Nil(t, rows)
}

func TestCreateRowsWithValues_InsertRowDataError(t *testing.T) {
	stubTable := &StubTableService{}
	stubBulk := &StubBulkService{}
	mockModel := &MockModelService{}
	mockColumn := &MockColumnService{}
	mockView := &MockViewService{}
	mockRel := &MockRelationshipService{}
	mockAsset := &MockAssetManagementService{}

	modelID := uuid.New().String()
	columnID := uuid.New().String()
	model := tenant.Model{ID: uuid.MustParse(modelID), Alias: "tbl"}

	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(model, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", columnID).Return(tenant.Column{}, errors.New("column failed"))
	stubTable.CreateRecordFn = func(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": 1}, nil
	}

	svc := setupTableManagementServiceWithStubs(stubTable, stubBulk, mockModel, mockColumn, mockView, mockRel, mockAsset)

	rows, err := svc.(interface {
		CreateRowsWithValues(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error)
	}).CreateRowsWithValues(context.Background(), "schema", modelID, []map[string]interface{}{{columnID: "value"}}, "creator", "editor")

	assert.Error(t, err)
	assert.Nil(t, rows)
}

func TestExtractCreatedRowID_MissingID(t *testing.T) {
	record := map[string]interface{}{"name": "test"}

	rowID, err := services.ExtractCreatedRowID(record)

	assert.Error(t, err)
	assert.Equal(t, 0, rowID)
}

func TestExtractCreatedRowID_IntType(t *testing.T) {
	record := map[string]interface{}{"id": 42}

	rowID, err := services.ExtractCreatedRowID(record)

	assert.NoError(t, err)
	assert.Equal(t, 42, rowID)
}

func TestExtractCreatedRowID_Int32Type(t *testing.T) {
	record := map[string]interface{}{"id": int32(123)}

	rowID, err := services.ExtractCreatedRowID(record)

	assert.NoError(t, err)
	assert.Equal(t, 123, rowID)
}

func TestExtractCreatedRowID_Int64Type(t *testing.T) {
	record := map[string]interface{}{"id": int64(456)}

	rowID, err := services.ExtractCreatedRowID(record)

	assert.NoError(t, err)
	assert.Equal(t, 456, rowID)
}

func TestExtractCreatedRowID_Float32Type(t *testing.T) {
	record := map[string]interface{}{"id": float32(789.0)}

	rowID, err := services.ExtractCreatedRowID(record)

	assert.NoError(t, err)
	assert.Equal(t, 789, rowID)
}

func TestExtractCreatedRowID_StringType(t *testing.T) {
	record := map[string]interface{}{"id": "555"}

	rowID, err := services.ExtractCreatedRowID(record)

	assert.NoError(t, err)
	assert.Equal(t, 555, rowID)
}

func TestExtractCreatedRowID_StringTypeInvalid(t *testing.T) {
	record := map[string]interface{}{"id": "invalid"}

	rowID, err := services.ExtractCreatedRowID(record)

	assert.Error(t, err)
	assert.Equal(t, 0, rowID)
}

func TestExtractCreatedRowID_UnsupportedType(t *testing.T) {
	record := map[string]interface{}{"id": true}

	rowID, err := services.ExtractCreatedRowID(record)

	assert.Error(t, err)
	assert.Equal(t, 0, rowID)
}

func TestHandleLinkedColumnDeletion_NoRelation(t *testing.T) {
	_, _, _, _, _, _, _, _, svc := setupTableManagementService()

	col := dto.ColumnResponse{
		ID:      uuid.New(),
		Meta:    map[string]interface{}{},
		ModelID: uuid.New(),
	}

	columnData := dto.ColumnResponse{
		ID:      uuid.New(),
		Meta:    map[string]interface{}{},
		ModelID: uuid.New(),
	}

	// This should handle gracefully when no relation exists
	svc.(interface {
		HandleLinkedColumnDeletion(ctx context.Context, schemaName string, col dto.ColumnResponse, columnData dto.ColumnResponse)
	}).HandleLinkedColumnDeletion(context.Background(), "schema", col, columnData)

	// No assertions needed - should just not panic
}

func TestHandleLinkedColumnDeletion_WithRelation(t *testing.T) {
	_, _, _, _, mockColumn, _, _, _, svc := setupTableManagementService()

	linkedModelID := uuid.New().String()
	columnDataID := uuid.New()

	col := dto.ColumnResponse{
		ID:      uuid.New(),
		Meta:    map[string]interface{}{"relation": map[string]interface{}{"with": linkedModelID}},
		ModelID: uuid.New(),
	}

	columnData := dto.ColumnResponse{
		ID:      columnDataID,
		Meta:    map[string]interface{}{},
		ModelID: uuid.New(),
	}

	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", linkedModelID).
		Return([]tenant.Column{}, nil)

	// This should handle gracefully
	svc.(interface {
		HandleLinkedColumnDeletion(ctx context.Context, schemaName string, col dto.ColumnResponse, columnData dto.ColumnResponse)
	}).HandleLinkedColumnDeletion(context.Background(), "schema", col, columnData)

	mockColumn.AssertExpectations(t)
}

func TestHandleLinkedColumnDeletion_WithLookupColumn(t *testing.T) {
	_, _, _, _, mockColumn, _, _, _, svc := setupTableManagementService()

	linkedModelID := uuid.New().String()
	columnDataID := uuid.New().String()
	lookupColumnID := uuid.New().String()

	col := dto.ColumnResponse{
		ID:      uuid.New(),
		Meta:    map[string]interface{}{"relation": map[string]interface{}{"with": linkedModelID}},
		ModelID: uuid.New(),
	}

	lookupCol := tenant.Column{
		ID:         uuid.MustParse(lookupColumnID),
		ColumnName: "lookup_col",
		UIDT:       "lookup",
		Meta: map[string]interface{}{
			"lookup_column_id": columnDataID,
		},
	}

	columnData := dto.ColumnResponse{
		ID:      uuid.MustParse(columnDataID),
		Meta:    map[string]interface{}{},
		ModelID: uuid.New(),
	}

	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", linkedModelID).
		Return([]tenant.Column{lookupCol}, nil)

	svc.(interface {
		HandleLinkedColumnDeletion(ctx context.Context, schemaName string, col dto.ColumnResponse, columnData dto.ColumnResponse)
	}).HandleLinkedColumnDeletion(context.Background(), "schema", col, columnData)

	mockColumn.AssertCalled(t, "GetColumnByModelID", mock.Anything, "schema", linkedModelID)
}

func TestDeleteLookupColumnAndReorder_Success(t *testing.T) {
	_, mockTable, _, _, mockColumn, _, mockRel, _, svc := setupTableManagementService()

	modelID := uuid.New()
	relationID := uuid.New()
	lookupSourceColumnID := uuid.New()
	linkedColumnID := uuid.New()
	orderIndex := 3.0

	linkedCol := dto.ColumnResponse{
		ID:         linkedColumnID,
		ModelID:    modelID,
		ColumnName: "lookup_title",
		OrderIndex: &orderIndex,
		Meta: map[string]interface{}{
			"lookup_column_id": lookupSourceColumnID.String(),
			"relation_id":      relationID.String(),
		},
	}
	sourceLookupCol := tenant.Column{ID: lookupSourceColumnID, ColumnName: "title"}
	relation := tenant.Relation{
		ID:                  relationID,
		SourceModelID:       modelID.String(),
		SourceLookupColumns: []string{"title", "other"},
	}

	mockColumn.On("GetColumnByID", mock.Anything, "schema", lookupSourceColumnID.String()).Return(sourceLookupCol, nil)
	mockRel.On("GetRelationByID", mock.Anything, relationID.String(), "schema").Return(relation, nil)
	mockRel.On("UpdateRelation", mock.Anything, relationID.String(), mock.MatchedBy(func(req dto.RelationUpdate) bool {
		columns, ok := req.SourceLookupColumns.([]string)
		return ok && len(columns) == 1 && columns[0] == "other"
	}), "schema").Return(relation, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", linkedColumnID.String()).Return(nil)
	mockTable.On("GetByFunction", mock.Anything, "public.reorder_columns_after_delete", mock.MatchedBy(func(args map[string]interface{}) bool {
		return args["p_schema_name"] == "schema" && args["p_model_id"] == modelID.String() && args["p_order_index"] == orderIndex
	})).Return([]map[string]interface{}{}, nil)

	svc.(interface {
		DeleteLookupColumnAndReorder(ctx context.Context, schemaName string, linkedCol dto.ColumnResponse)
	}).DeleteLookupColumnAndReorder(context.Background(), "schema", linkedCol)

	mockColumn.AssertExpectations(t)
	mockRel.AssertExpectations(t)
	mockTable.AssertExpectations(t)
}
