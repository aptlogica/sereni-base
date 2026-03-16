package base_test

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/base"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBaseService is a mock implementation of BaseService interface
type MockBaseService struct {
	mock.Mock
}

func (m *MockBaseService) CreateBase(ctx context.Context, schemaName string) (tenant.Base, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).(tenant.Base), args.Error(1)
}

func (m *MockBaseService) BaseInsertion(ctx context.Context, req dto.BaseInsertion, schemaName string) (tenant.Base, error) {
	args := m.Called(ctx, req, schemaName)
	return args.Get(0).(tenant.Base), args.Error(1)
}

func (m *MockBaseService) GetBaseByID(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
	args := m.Called(ctx, schemaName, id)
	return args.Get(0).(tenant.Base), args.Error(1)
}

func (m *MockBaseService) GetAllBases(ctx context.Context, schemaName string) ([]tenant.Base, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).([]tenant.Base), args.Error(1)
}

func (m *MockBaseService) UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate) (tenant.Base, error) {
	args := m.Called(ctx, schemaName, id, req)
	return args.Get(0).(tenant.Base), args.Error(1)
}

func (m *MockBaseService) DeleteBase(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}

func (m *MockBaseService) GetBasesByWorkspace(ctx context.Context, schemaName, workspaceID string) ([]tenant.Base, error) {
	args := m.Called(ctx, schemaName, workspaceID)
	return args.Get(0).([]tenant.Base), args.Error(1)
}

func (m *MockBaseService) GetBulkbases(ctx context.Context, schemaName string, ids []string) ([]tenant.Base, error) {
	args := m.Called(ctx, schemaName, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Base), args.Error(1)
}

// MockTableManagementService is a mock implementation of TableManagementService interface
type MockTableManagementService struct {
	mock.Mock
}

func (m *MockTableManagementService) CreateTableWithDefaults(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
	args := m.Called(ctx, tableData, schemaName)
	return args.Get(0).(dto.TableResponse), args.Error(1)
}

func (m *MockTableManagementService) UpdateTable(ctx context.Context, id string, tableData dto.UpdateTableRequest, schemaName string) (dto.TableResponse, error) {
	args := m.Called(ctx, id, tableData, schemaName)
	return args.Get(0).(dto.TableResponse), args.Error(1)
}

func (m *MockTableManagementService) GetTableByID(ctx context.Context, id string, schemaName string) (dto.TableResponse, error) {
	args := m.Called(ctx, id, schemaName)
	return args.Get(0).(dto.TableResponse), args.Error(1)
}

func (m *MockTableManagementService) GetAllTables(ctx context.Context, schemaName string) ([]dto.TableResponse, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).([]dto.TableResponse), args.Error(1)
}

func (m *MockTableManagementService) GetModelByBaseID(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
	args := m.Called(ctx, schemaName, baseID)
	return args.Get(0).([]dto.TableResponse), args.Error(1)
}

func (m *MockTableManagementService) GetModelByWorkspaceID(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
	args := m.Called(ctx, schemaName, workspaceID)
	return args.Get(0).([]dto.TableResponse), args.Error(1)
}

func (m *MockTableManagementService) DeleteTable(ctx context.Context, schemaName string, modelID string) error {
	args := m.Called(ctx, schemaName, modelID)
	return args.Error(0)
}

func (m *MockTableManagementService) AddColumn(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
	args := m.Called(ctx, schemaName, columnData)
	return args.Get(0).(dto.ColumnResponse), args.Error(1)
}

func (m *MockTableManagementService) GetColumnById(ctx context.Context, schemaName string, id string) (dto.ColumnResponse, error) {
	args := m.Called(ctx, schemaName, id)
	return args.Get(0).(dto.ColumnResponse), args.Error(1)
}

func (m *MockTableManagementService) GetAllColumns(ctx context.Context, schemaName string) ([]dto.ColumnResponse, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).([]dto.ColumnResponse), args.Error(1)
}

func (m *MockTableManagementService) GetColumnsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ColumnResponse, error) {
	args := m.Called(ctx, schemaName, modelID)
	return args.Get(0).([]dto.ColumnResponse), args.Error(1)
}

func (m *MockTableManagementService) UpdateColumn(ctx context.Context, schemaName string, id string, req dto.ColumnUpdate) (dto.ColumnResponse, error) {
	args := m.Called(ctx, schemaName, id, req)
	return args.Get(0).(dto.ColumnResponse), args.Error(1)
}

func (m *MockTableManagementService) DeleteColumn(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}

func (m *MockTableManagementService) ReorderColumn(ctx context.Context, schemaName string, req dto.ReorderColumnRequest) ([]dto.ColumnResponse, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).([]dto.ColumnResponse), args.Error(1)
}

func (m *MockTableManagementService) CreateView(ctx context.Context, schemaName string, viewData dto.CreateViewRequest) (dto.ViewResponse, error) {
	args := m.Called(ctx, schemaName, viewData)
	return args.Get(0).(dto.ViewResponse), args.Error(1)
}

func (m *MockTableManagementService) GetViewByID(ctx context.Context, schemaName string, id string) (dto.ViewResponse, error) {
	args := m.Called(ctx, schemaName, id)
	return args.Get(0).(dto.ViewResponse), args.Error(1)
}

func (m *MockTableManagementService) GetAllViews(ctx context.Context, schemaName string) ([]dto.ViewResponse, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).([]dto.ViewResponse), args.Error(1)
}

func (m *MockTableManagementService) GetViewsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ViewResponse, error) {
	args := m.Called(ctx, schemaName, modelID)
	return args.Get(0).([]dto.ViewResponse), args.Error(1)
}

func (m *MockTableManagementService) UpdateView(ctx context.Context, schemaName string, id string, req dto.ViewUpdate) (dto.ViewResponse, error) {
	args := m.Called(ctx, schemaName, id, req)
	return args.Get(0).(dto.ViewResponse), args.Error(1)
}

func (m *MockTableManagementService) DeleteView(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}

func (m *MockTableManagementService) CreateRow(ctx context.Context, schemaName string, req dto.CreateRowRequest) (dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(dto.RecordResponse), args.Error(1)
}

func (m *MockTableManagementService) CreateRowWithRecords(ctx context.Context, schemaName string, modelAlias string, record map[string]interface{}) (dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, modelAlias, record)
	return args.Get(0).(dto.RecordResponse), args.Error(1)
}

func (m *MockTableManagementService) CreateRowsWithRecordsBulk(ctx context.Context, schemaName string, modelAlias string, records []map[string]interface{}) ([]dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, modelAlias, records)
	return args.Get(0).([]dto.RecordResponse), args.Error(1)
}

func (m *MockTableManagementService) GetAllRecords(ctx context.Context, schemaName string, modelID string) (dto.RecordsResponse, error) {
	args := m.Called(ctx, schemaName, modelID)
	return args.Get(0).(dto.RecordsResponse), args.Error(1)
}

func (m *MockTableManagementService) InsertRowData(ctx context.Context, schemaName string, req dto.InsertRowDataRequest) (dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(dto.RecordResponse), args.Error(1)
}

func (m *MockTableManagementService) DeleteRow(ctx context.Context, schemaName string, req dto.DeleteRowDataRequest) error {
	args := m.Called(ctx, schemaName, req)
	return args.Error(0)
}

func (m *MockTableManagementService) UpdateRawDataForLinks(ctx context.Context, schemaName string, req dto.UpdateRowDataLinksRequest) (dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(dto.RecordResponse), args.Error(1)
}

func (m *MockTableManagementService) AddAttachment(ctx context.Context, schemaName string, req dto.AddAttachmentRequest, files []*multipart.FileHeader) (dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, req, files)
	return args.Get(0).(dto.RecordResponse), args.Error(1)
}

func (m *MockTableManagementService) UpdateAttachment(ctx context.Context, schemaName string, req dto.UpdateAttachmentRequest) (dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(dto.RecordResponse), args.Error(1)
}

func (m *MockTableManagementService) BulkDeleteRows(ctx context.Context, schemaName string, req dto.BulkDeleteRowsRequest) (int, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Int(0), args.Error(1)
}

func (m *MockTableManagementService) RemoveAttachments(ctx context.Context, schemaName string, req dto.RemoveAttachmentsRequest) (dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(dto.RecordResponse), args.Error(1)
}

// MockModelService is a mock implementation of ModelService interface
type MockModelService struct {
	mock.Mock
}

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

func (m *MockModelService) GetModelByBaseID(ctx context.Context, schemaName string, baseID string) ([]tenant.Model, error) {
	args := m.Called(ctx, schemaName, baseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Model), args.Error(1)
}

func (m *MockModelService) GetModelByWorkspaceID(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Model, error) {
	args := m.Called(ctx, schemaName, workspaceID)
	return args.Get(0).([]tenant.Model), args.Error(1)
}

func (m *MockModelService) DeleteModel(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}

// MockAssetManagementService is a mock implementation of AssetManagementService interface
type MockAssetManagementService struct {
	mock.Mock
}

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

func setupBaseManagementService() (*pkg.DatabaseService, *MockTableService, *MockBaseService, *MockTableManagementService, *MockModelService, *MockAssetManagementService, interfaces.BaseManagementService) {
	mockTable := &MockTableService{}
	mockBase := &MockBaseService{}
	mockTableManagement := &MockTableManagementService{}
	mockModel := &MockModelService{}
	mockAsset := &MockAssetManagementService{}

	db := &pkg.DatabaseService{TableService: mockTable}

	service := services.NewBaseManagementService(db, mockBase, mockTableManagement, mockModel, mockAsset)

	return db, mockTable, mockBase, mockTableManagement, mockModel, mockAsset, service
}

func TestNewBaseManagementService(t *testing.T) {
	_, _, _, _, _, _, service := setupBaseManagementService()
	assert.NotNil(t, service)
}

func TestBaseManagementService_CreateBase(t *testing.T) {
	t.Run("success with default table", func(t *testing.T) {
		_, _, mockBase, mockTableManagement, _, _, service := setupBaseManagementService()

		wsID := uuid.New()
		inserted := tenant.Base{ID: uuid.New(), WorkspaceID: wsID.String(), Title: "Base"}
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(inserted, nil)
		mockTableManagement.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(dto.TableResponse{}, nil)

		result, err := service.CreateBase(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base", CreatedBy: "user"}, "schema", "user")

		assert.NoError(t, err)
		assert.Equal(t, inserted.ID, result.ID)
		mockBase.AssertExpectations(t)
		mockTableManagement.AssertExpectations(t)
	})

	t.Run("base insertion error", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		wsID := uuid.New()
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(tenant.Base{}, errors.New("insert failed"))

		_, err := service.CreateBase(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base"}, "schema", "user")

		assert.Error(t, err)
	})

	t.Run("table creation error", func(t *testing.T) {
		_, _, mockBase, mockTableManagement, _, _, service := setupBaseManagementService()

		wsID := uuid.New()
		inserted := tenant.Base{ID: uuid.New(), WorkspaceID: wsID.String(), Title: "Base"}
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(inserted, nil)
		mockTableManagement.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(dto.TableResponse{}, errors.New("table creation failed"))

		_, err := service.CreateBase(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base"}, "schema", "user")

		assert.Error(t, err)
	})

	t.Run("invalid workspace id", func(t *testing.T) {
		_, _, _, _, _, _, service := setupBaseManagementService()

		_, err := service.CreateBase(context.Background(), dto.CreateBaseRequest{WorkspaceID: "invalid-uuid", Title: "Base"}, "schema", "user")

		assert.Error(t, err)
		assert.Equal(t, app_errors.InvalidPayload, err)
	})
}

func TestBaseManagementService_CreateBaseWithoutTable(t *testing.T) {
	t.Run("success without default table", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		wsID := uuid.New()
		inserted := tenant.Base{ID: uuid.New(), WorkspaceID: wsID.String(), Title: "Base"}
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(inserted, nil)

		result, err := service.CreateBaseWithoutTable(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base", CreatedBy: "user"}, "schema", "user")

		assert.NoError(t, err)
		assert.Equal(t, inserted.ID, result.ID)
		mockBase.AssertExpectations(t)
	})

	t.Run("base insertion error", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		wsID := uuid.New()
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(tenant.Base{}, errors.New("insert failed"))

		_, err := service.CreateBaseWithoutTable(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base"}, "schema", "user")

		assert.Error(t, err)
	})

	t.Run("invalid workspace id", func(t *testing.T) {
		_, _, _, _, _, _, service := setupBaseManagementService()

		_, err := service.CreateBaseWithoutTable(context.Background(), dto.CreateBaseRequest{WorkspaceID: "invalid-uuid", Title: "Base"}, "schema", "user")

		assert.Error(t, err)
		assert.Equal(t, app_errors.InvalidPayload, err)
	})
}

func TestCreateBaseWithImage(t *testing.T) {
	t.Run("no file", func(t *testing.T) {
		_, _, mockBase, mockTableManagement, _, _, service := setupBaseManagementService()

		wsID := uuid.New()
		inserted := tenant.Base{ID: uuid.New(), WorkspaceID: wsID.String(), Title: "Base"}
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(inserted, nil)
		mockTableManagement.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(dto.TableResponse{}, nil)

		result, err := service.CreateBaseWithImage(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base"}, "schema", "user", nil)

		assert.NoError(t, err)
		assert.Equal(t, inserted.ID, result.ID)
	})

	t.Run("invalid extension", func(t *testing.T) {
		_, _, mockBase, mockTableManagement, _, _, service := setupBaseManagementService()

		wsID := uuid.New()
		inserted := tenant.Base{ID: uuid.New(), WorkspaceID: wsID.String(), Title: "Base"}
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(inserted, nil)
		mockTableManagement.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(dto.TableResponse{}, nil)

		fh := &multipart.FileHeader{Filename: "image.gif"}
		result, err := service.CreateBaseWithImage(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base"}, "schema", "user", fh)

		assert.NoError(t, err)
		assert.Equal(t, inserted.ID, result.ID)
	})

	t.Run("upload error", func(t *testing.T) {
		_, _, mockBase, mockTableManagement, _, mockAsset, service := setupBaseManagementService()

		wsID := uuid.New()
		inserted := tenant.Base{ID: uuid.New(), WorkspaceID: wsID.String(), Title: "Base"}
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(inserted, nil)
		mockTableManagement.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(dto.TableResponse{}, nil)
		mockAsset.On("Upload", mock.Anything, mock.Anything, "schema").Return(nil, errors.New("upload failed"))

		fh := &multipart.FileHeader{Filename: "image.png"}
		result, err := service.CreateBaseWithImage(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base"}, "schema", "user", fh)

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, result.ID)
	})

	t.Run("update error", func(t *testing.T) {
		_, _, mockBase, mockTableManagement, _, mockAsset, service := setupBaseManagementService()

		wsID := uuid.New()
		inserted := tenant.Base{ID: uuid.New(), WorkspaceID: wsID.String(), Title: "Base"}
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(inserted, nil)
		mockTableManagement.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(dto.TableResponse{}, nil)
		mockAsset.On("Upload", mock.Anything, mock.Anything, "schema").Return([]tenant.Assets{{Url: "http://img"}}, nil)
		mockBase.On("UpdateBase", mock.Anything, "schema", inserted.ID.String(), mock.Anything).
			Return(tenant.Base{}, errors.New("update fail"))

		fh := &multipart.FileHeader{Filename: "image.png"}
		result, err := service.CreateBaseWithImage(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base"}, "schema", "user", fh)

		assert.NoError(t, err)
		assert.Equal(t, inserted.ID, result.ID)
	})

	t.Run("success", func(t *testing.T) {
		_, _, mockBase, mockTableManagement, _, mockAsset, service := setupBaseManagementService()

		wsID := uuid.New()
		inserted := tenant.Base{ID: uuid.New(), WorkspaceID: wsID.String(), Title: "Base"}
		updated := tenant.Base{ID: inserted.ID, WorkspaceID: wsID.String(), Title: "Base", Image: "http://img"}
		mockBase.On("BaseInsertion", mock.Anything, mock.Anything, "schema").Return(inserted, nil)
		mockTableManagement.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(dto.TableResponse{}, nil)
		mockAsset.On("Upload", mock.Anything, mock.Anything, "schema").Return([]tenant.Assets{{Url: "http://img"}}, nil)
		mockBase.On("UpdateBase", mock.Anything, "schema", inserted.ID.String(), mock.Anything).
			Return(updated, nil)

		fh := &multipart.FileHeader{Filename: "image.png"}
		result, err := service.CreateBaseWithImage(context.Background(), dto.CreateBaseRequest{WorkspaceID: wsID.String(), Title: "Base"}, "schema", "user", fh)

		assert.NoError(t, err)
		assert.Equal(t, "http://img", result.Image)
	})
}

func TestGetBaseByID_Proxy(t *testing.T) {
	_, _, mockBase, _, _, _, service := setupBaseManagementService()

	base := tenant.Base{ID: uuid.New(), Title: "Base"}
	mockBase.On("GetBaseByID", mock.Anything, "schema", base.ID.String()).Return(base, nil)

	result, err := service.GetBaseByID(context.Background(), "schema", base.ID.String())

	assert.NoError(t, err)
	assert.Equal(t, base.ID, result.ID)
	mockBase.AssertExpectations(t)
}

func TestGetAllBasesWithAccess(t *testing.T) {
	t.Run("wildcard", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		bases := []tenant.Base{{ID: uuid.New(), Title: "Base"}}
		mockBase.On("GetBasesByWorkspace", mock.Anything, "schema", "ws").Return(bases, nil)

		member := &tenant.WorkspaceMember{WorkspaceID: "ws", BasesIds: "*"}
		result, err := service.GetAllBasesWithAccess(context.Background(), "schema", member)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockBase.AssertExpectations(t)
	})

	t.Run("explicit ids", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		bases := []tenant.Base{{ID: uuid.New(), Title: "Base"}}
		mockBase.On("GetBulkbases", mock.Anything, "schema", []string{"id1", "id2"}).Return(bases, nil)

		member := &tenant.WorkspaceMember{WorkspaceID: "ws", BasesIds: "id1, id2"}
		result, err := service.GetAllBasesWithAccess(context.Background(), "schema", member)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		mockBase.AssertExpectations(t)
	})
}

func TestUpdateBase_ProxyAndDefaultUser(t *testing.T) {
	_, _, mockBase, _, _, _, service := setupBaseManagementService()

	base := tenant.Base{ID: uuid.New(), Title: "Base"}
	mockBase.On("UpdateBase", mock.Anything, "schema", base.ID.String(), mock.MatchedBy(func(req dto.BaseUpdate) bool {
		return req.UpdatedBy == "user"
	})).Return(base, nil)

	result, err := service.UpdateBase(context.Background(), "schema", base.ID.String(), dto.BaseUpdate{}, "user", nil, "")

	assert.NoError(t, err)
	assert.Equal(t, base.ID, result.ID)
	mockBase.AssertExpectations(t)
}

func TestDeleteBase_Proxy(t *testing.T) {
	_, _, mockBase, _, _, _, service := setupBaseManagementService()

	mockBase.On("DeleteBase", mock.Anything, "schema", "id").Return(nil)

	err := service.DeleteBase(context.Background(), "schema", "id")

	assert.NoError(t, err)
	mockBase.AssertExpectations(t)
}

func TestGetTablesByBaseId(t *testing.T) {
	t.Run("model service error", func(t *testing.T) {
		_, _, _, _, mockModel, _, service := setupBaseManagementService()

		mockModel.On("GetModelByBaseID", mock.Anything, "schema", "base").Return(nil, errors.New("fail"))

		_, err := service.GetTablesByBaseId(context.Background(), "schema", "base")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		_, _, _, _, mockModel, _, service := setupBaseManagementService()

		model := tenant.Model{ID: uuid.New(), BaseID: uuid.New(), Title: "Model", CreatedBy: "user", UpdatedBy: "user", CreatedAt: time.Now(), UpdatedAt: time.Now()}
		mockModel.On("GetModelByBaseID", mock.Anything, "schema", "base").Return([]tenant.Model{model}, nil)

		results, err := service.GetTablesByBaseId(context.Background(), "schema", "base")

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, model.ID, results[0].Model.ID)
	})
}

func TestGetBasesByWorkspace_Proxy(t *testing.T) {
	_, _, mockBase, _, _, _, service := setupBaseManagementService()

	bases := []tenant.Base{{ID: uuid.New(), Title: "Base"}}
	mockBase.On("GetBasesByWorkspace", mock.Anything, "schema", "ws").Return(bases, nil)

	result, err := service.GetBasesByWorkspace(context.Background(), "schema", "ws")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockBase.AssertExpectations(t)
}

func TestAddBaseImage(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, errors.New("fail"))

		_, err := service.AddBaseImage(context.Background(), "schema", "base", &multipart.FileHeader{Filename: "image.png"}, "user")

		assert.Error(t, err)
	})

	t.Run("delete existing image error", func(t *testing.T) {
		_, _, mockBase, _, _, mockAsset, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{Image: "http://old"}, nil)
		mockAsset.On("GetAssetByURL", mock.Anything, "schema", "http://old").Return(tenant.Assets{Url: "http://old", ID: uuid.New()}, nil)
		mockAsset.On("DeleteAsset", mock.Anything, mock.Anything, "schema").Return(errors.New("delete failed"))

		_, err := service.AddBaseImage(context.Background(), "schema", "base", &multipart.FileHeader{Filename: "image.png"}, "user")

		assert.Error(t, err)
	})

	t.Run("nil file", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, nil)

		_, err := service.AddBaseImage(context.Background(), "schema", "base", nil, "user")

		assert.ErrorIs(t, err, app_errors.InvalidPayload)
	})

	t.Run("invalid extension", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, nil)

		_, err := service.AddBaseImage(context.Background(), "schema", "base", &multipart.FileHeader{Filename: "image.bmp"}, "user")

		assert.ErrorIs(t, err, app_errors.InvalidPayload)
	})

	t.Run("upload error", func(t *testing.T) {
		_, _, mockBase, _, _, mockAsset, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, nil)
		mockAsset.On("Upload", mock.Anything, mock.Anything, "schema").Return(nil, errors.New("upload failed"))

		_, err := service.AddBaseImage(context.Background(), "schema", "base", &multipart.FileHeader{Filename: "image.png"}, "user")

		assert.Error(t, err)
	})

	t.Run("get asset error ignored", func(t *testing.T) {
		_, _, mockBase, _, _, mockAsset, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{Image: "http://old"}, nil)
		mockAsset.On("GetAssetByURL", mock.Anything, "schema", "http://old").Return(tenant.Assets{}, errors.New("not found"))
		mockAsset.On("Upload", mock.Anything, mock.Anything, "schema").Return(nil, errors.New("upload failed"))

		_, err := service.AddBaseImage(context.Background(), "schema", "base", &multipart.FileHeader{Filename: "image.png"}, "user")

		assert.Error(t, err)
	})

	t.Run("update error", func(t *testing.T) {
		_, _, mockBase, _, _, mockAsset, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, nil)
		mockAsset.On("Upload", mock.Anything, mock.Anything, "schema").Return([]tenant.Assets{{Url: "http://img"}}, nil)
		mockBase.On("UpdateBase", mock.Anything, "schema", "base", mock.Anything).Return(tenant.Base{}, errors.New("fail"))

		_, err := service.AddBaseImage(context.Background(), "schema", "base", &multipart.FileHeader{Filename: "image.png"}, "user")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		_, _, mockBase, _, _, mockAsset, service := setupBaseManagementService()

		updated := tenant.Base{ID: uuid.New(), Image: "http://img"}
		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, nil)
		mockAsset.On("Upload", mock.Anything, mock.Anything, "schema").Return([]tenant.Assets{{Url: "http://img"}}, nil)
		mockBase.On("UpdateBase", mock.Anything, "schema", "base", mock.Anything).Return(updated, nil)

		result, err := service.AddBaseImage(context.Background(), "schema", "base", &multipart.FileHeader{Filename: "image.png"}, "user")

		assert.NoError(t, err)
		assert.Equal(t, "http://img", result.Image)
	})
}

func TestRemoveBaseImage(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, errors.New("fail"))

		_, err := service.RemoveBaseImage(context.Background(), "schema", "base", "user")

		assert.Error(t, err)
	})

	t.Run("delete existing image success", func(t *testing.T) {
		_, _, mockBase, _, _, mockAsset, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{Image: "http://old"}, nil)
		mockAsset.On("GetAssetByURL", mock.Anything, "schema", "http://old").Return(tenant.Assets{Url: "http://old", ID: uuid.New()}, nil)
		mockAsset.On("DeleteAsset", mock.Anything, mock.Anything, "schema").Return(nil)
		mockBase.On("UpdateBase", mock.Anything, "schema", "base", mock.Anything).Return(tenant.Base{ID: uuid.New()}, nil)

		_, err := service.RemoveBaseImage(context.Background(), "schema", "base", "user")

		assert.NoError(t, err)
	})

	t.Run("update error", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, nil)
		mockBase.On("UpdateBase", mock.Anything, "schema", "base", mock.Anything).Return(tenant.Base{}, errors.New("fail"))

		_, err := service.RemoveBaseImage(context.Background(), "schema", "base", "user")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		_, _, mockBase, _, _, _, service := setupBaseManagementService()

		updated := tenant.Base{ID: uuid.New(), Image: ""}
		mockBase.On("GetBaseByID", mock.Anything, "schema", "base").Return(tenant.Base{}, nil)
		mockBase.On("UpdateBase", mock.Anything, "schema", "base", mock.Anything).Return(updated, nil)

		result, err := service.RemoveBaseImage(context.Background(), "schema", "base", "user")

		assert.NoError(t, err)
		assert.Equal(t, "", result.Image)
	})
}

func TestRemoveUserFromBase(t *testing.T) {
	t.Run("get data error", func(t *testing.T) {
		_, mockTable, _, _, _, _, service := setupBaseManagementService()

		mockTable.On("GetTableData", "\"schema\".access_members", mock.Anything).
			Return(nil, errors.New("db error"))

		err := service.RemoveUserFromBase(context.Background(), "schema", "base", "user")

		assert.Error(t, err)
	})

	t.Run("no records", func(t *testing.T) {
		_, mockTable, _, _, _, _, service := setupBaseManagementService()

		mockTable.On("GetTableData", "\"schema\".access_members", mock.Anything).
			Return([]map[string]interface{}{}, nil)

		err := service.RemoveUserFromBase(context.Background(), "schema", "base", "user")

		assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
	})

	t.Run("success", func(t *testing.T) {
		_, mockTable, _, _, _, _, service := setupBaseManagementService()

		mockTable.On("GetTableData", "\"schema\".access_members", mock.Anything).
			Return([]map[string]interface{}{{"id": "member-id"}}, nil)
		mockTable.On("DeleteRecord", "\"schema\".access_members", "member-id").Return(nil)

		err := service.RemoveUserFromBase(context.Background(), "schema", "base", "user")

		assert.NoError(t, err)
	})

	t.Run("delete error", func(t *testing.T) {
		_, mockTable, _, _, _, _, service := setupBaseManagementService()

		mockTable.On("GetTableData", "\"schema\".access_members", mock.Anything).
			Return([]map[string]interface{}{{"id": "member-id"}}, nil)
		mockTable.On("DeleteRecord", "\"schema\".access_members", "member-id").Return(errors.New("delete failed"))

		err := service.RemoveUserFromBase(context.Background(), "schema", "base", "user")

		assert.Error(t, err)
	})
}

// Make sure compile-time interface compliance is checked
var _ interfaces.BaseManagementService = services.NewBaseManagementService(&pkg.DatabaseService{}, &MockBaseService{}, &MockTableManagementService{}, &MockModelService{}, &MockAssetManagementService{})

// Ensure MockTableService satisfies the table interface used by DatabaseService
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
