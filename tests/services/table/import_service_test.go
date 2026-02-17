package table_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	antivirusInterfaces "serenibase/internal/providers/antivirus/interfaces"
	services "serenibase/internal/services/table"
	"serenibase/internal/utils/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTableManagementService implements interfaces.TableManagementService
// Only methods used by Import are mocked; others return zero values.
type MockTableManagementService struct {
	mock.Mock
	AddColumnFn func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error)
}

func (m *MockTableManagementService) CreateTableWithDefaults(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
	args := m.Called(ctx, tableData, schemaName)
	return args.Get(0).(dto.TableResponse), args.Error(1)
}

func (m *MockTableManagementService) UpdateColumn(ctx context.Context, schemaName string, id string, req dto.ColumnUpdate) (dto.ColumnResponse, error) {
	args := m.Called(ctx, schemaName, id, req)
	return args.Get(0).(dto.ColumnResponse), args.Error(1)
}

func (m *MockTableManagementService) AddColumn(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
	if m.AddColumnFn != nil {
		return m.AddColumnFn(ctx, schemaName, columnData)
	}
	args := m.Called(ctx, schemaName, columnData)
	return args.Get(0).(dto.ColumnResponse), args.Error(1)
}

func (m *MockTableManagementService) CreateRowsWithRecordsBulk(ctx context.Context, schemaName string, modelAlias string, records []map[string]interface{}) ([]dto.RecordResponse, error) {
	args := m.Called(ctx, schemaName, modelAlias, records)
	return args.Get(0).([]dto.RecordResponse), args.Error(1)
}

func (m *MockTableManagementService) GetTableByID(ctx context.Context, id string, schemaName string) (dto.TableResponse, error) {
	args := m.Called(ctx, id, schemaName)
	return args.Get(0).(dto.TableResponse), args.Error(1)
}

// Unused methods
func (m *MockTableManagementService) UpdateTable(ctx context.Context, id string, tableData dto.UpdateTableRequest, schemaName string) (dto.TableResponse, error) {
	return dto.TableResponse{}, nil
}
func (m *MockTableManagementService) GetAllTables(ctx context.Context, schemaName string) ([]dto.TableResponse, error) {
	return nil, nil
}
func (m *MockTableManagementService) GetModelByBaseID(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
	return nil, nil
}
func (m *MockTableManagementService) GetModelByWorkspaceID(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
	return nil, nil
}
func (m *MockTableManagementService) DeleteTable(ctx context.Context, schemaName string, modelID string) error {
	return nil
}
func (m *MockTableManagementService) GetColumnById(ctx context.Context, schemaName string, id string) (dto.ColumnResponse, error) {
	return dto.ColumnResponse{}, nil
}
func (m *MockTableManagementService) GetAllColumns(ctx context.Context, schemaName string) ([]dto.ColumnResponse, error) {
	return nil, nil
}
func (m *MockTableManagementService) GetColumnsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ColumnResponse, error) {
	return nil, nil
}
func (m *MockTableManagementService) DeleteColumn(ctx context.Context, schemaName string, id string) error {
	return nil
}
func (m *MockTableManagementService) ReorderColumn(ctx context.Context, schemaName string, req dto.ReorderColumnRequest) ([]dto.ColumnResponse, error) {
	return nil, nil
}
func (m *MockTableManagementService) CreateView(ctx context.Context, schemaName string, viewData dto.CreateViewRequest) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (m *MockTableManagementService) GetViewByID(ctx context.Context, schemaName string, id string) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (m *MockTableManagementService) GetAllViews(ctx context.Context, schemaName string) ([]dto.ViewResponse, error) {
	return nil, nil
}
func (m *MockTableManagementService) GetViewsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ViewResponse, error) {
	return nil, nil
}
func (m *MockTableManagementService) UpdateView(ctx context.Context, schemaName string, id string, req dto.ViewUpdate) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (m *MockTableManagementService) DeleteView(ctx context.Context, schemaName string, id string) error {
	return nil
}
func (m *MockTableManagementService) CreateRow(ctx context.Context, schemaName string, req dto.CreateRowRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (m *MockTableManagementService) CreateRowWithRecords(ctx context.Context, schemaName string, modelAlias string, record map[string]interface{}) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (m *MockTableManagementService) GetAllRecords(ctx context.Context, schemaName string, modelID string) (dto.RecordsResponse, error) {
	return dto.RecordsResponse{}, nil
}
func (m *MockTableManagementService) InsertRowData(ctx context.Context, schemaName string, req dto.InsertRowDataRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (m *MockTableManagementService) DeleteRow(ctx context.Context, schemaName string, req dto.DeleteRowDataRequest) error {
	return nil
}
func (m *MockTableManagementService) UpdateRawDataForLinks(ctx context.Context, schemaName string, req dto.UpdateRowDataLinksRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (m *MockTableManagementService) AddAttachment(ctx context.Context, schemaName string, req dto.AddAttachmentRequest, files []*multipart.FileHeader) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (m *MockTableManagementService) BulkDeleteRows(ctx context.Context, schemaName string, req dto.BulkDeleteRowsRequest) (int, error) {
	return 0, nil
}
func (m *MockTableManagementService) RemoveAttachments(ctx context.Context, schemaName string, req dto.RemoveAttachmentsRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}

// MockBaseManagementService implements interfaces.BaseManagementService
// Only CreateBase is used in Import tests.
type MockBaseManagementService struct {
	mock.Mock
}

func (m *MockBaseManagementService) CreateBase(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
	args := m.Called(ctx, req, schemaName, userId)
	return args.Get(0).(tenant.Base), args.Error(1)
}

func (m *MockBaseManagementService) CreateBaseWithoutTable(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
	args := m.Called(ctx, req, schemaName, userId)
	return args.Get(0).(tenant.Base), args.Error(1)
}

// Unused methods
func (m *MockBaseManagementService) CreateBaseWithImage(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string, fileHeader *multipart.FileHeader) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (m *MockBaseManagementService) GetBaseByID(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (m *MockBaseManagementService) GetAllBasesWithAccess(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error) {
	return nil, nil
}
func (m *MockBaseManagementService) UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate, userId string, fileHeader *multipart.FileHeader, removeImage string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (m *MockBaseManagementService) DeleteBase(ctx context.Context, schemaName string, id string) error {
	return nil
}
func (m *MockBaseManagementService) GetTablesByBaseId(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
	return nil, nil
}
func (m *MockBaseManagementService) GetBasesByWorkspace(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
	return nil, nil
}
func (m *MockBaseManagementService) AddBaseImage(ctx context.Context, schema string, baseID string, fileHeader *multipart.FileHeader, userId string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (m *MockBaseManagementService) RemoveBaseImage(ctx context.Context, schema string, baseID string, userId string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (m *MockBaseManagementService) RemoveUserFromBase(ctx context.Context, schemaName string, baseID string, userID string) error {
	return nil
}

// MockAntivirusProvider
// Only ScanReader is used in tests.
type MockAntivirusProvider struct {
	mock.Mock
}

func (m *MockAntivirusProvider) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAntivirusProvider) ScanReader(ctx context.Context, fileName string, r io.Reader) (antivirusInterfaces.ScanResult, error) {
	args := m.Called(ctx, fileName, r)
	return args.Get(0).(antivirusInterfaces.ScanResult), args.Error(1)
}

// helper to create multipart.FileHeader
func makeFileHeader(t *testing.T, filename string, content string) *multipart.FileHeader {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", filename)
	assert.NoError(t, err)
	_, err = fw.Write([]byte(content))
	assert.NoError(t, err)
	assert.NoError(t, w.Close())

	r := multipart.NewReader(&b, w.Boundary())
	form, err := r.ReadForm(int64(len(content)) + 1024)
	assert.NoError(t, err)
	return form.File["file"][0]
}

func baseTableResponse() dto.TableResponse {
	modelID := uuid.New()
	baseID := uuid.New()
	titleColID := uuid.New()
	return dto.TableResponse{
		Model: dto.ModelResponse{
			ID:        modelID,
			BaseID:    baseID,
			Alias:     "table_alias",
			CreatedBy: "user",
		},
		Columns: []dto.ColumnResponse{{
			ID:          titleColID,
			Title:       "Title",
			ColumnName:  "title",
			ModelID:     modelID,
			BaseID:      baseID,
			Description: "",
		}},
	}
}

func TestImport_ScanFileOpenError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}
	mockAV := &MockAntivirusProvider{}

	svc := services.NewImportService(mockTable, mockBase, mockAV)

	badHeader := &multipart.FileHeader{Filename: "missing.csv"}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, badHeader)

	assert.Error(t, err)
}

func TestImport_AntivirusThreat(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}
	mockAV := &MockAntivirusProvider{}

	file := makeFileHeader(t, "data.csv", "Title\n")

	mockAV.On("ScanReader", mock.Anything, "data.csv", mock.Anything).
		Return(antivirusInterfaces.ScanResult{Clean: false, Threat: "virus"}, errors.New("infected"))

	svc := services.NewImportService(mockTable, mockBase, mockAV)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_AntivirusPass(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}
	mockAV := &MockAntivirusProvider{}

	file := makeFileHeader(t, "data.csv", "Title\nA\n")
	resp := baseTableResponse()

	mockAV.On("ScanReader", mock.Anything, "data.csv", mock.Anything).
		Return(antivirusInterfaces.ScanResult{Clean: true}, nil)
	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").
		Return(resp, errors.New("stop"))

	svc := services.NewImportService(mockTable, mockBase, mockAV)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_EnsureBaseErrors(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	svc := services.NewImportService(mockTable, mockBase, nil)

	file := makeFileHeader(t, "data.csv", "Title\n")
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{Title: "T"}, file)
	assert.Error(t, err)

	mockBase.On("CreateBaseWithoutTable", mock.Anything, mock.Anything, "schema", "user").
		Return(tenant.Base{}, errors.New("fail"))
	_, err = svc.Import(context.Background(), "schema", dto.CreateTableRequest{Title: "T", WorkspaceID: "ws", CreatedBy: "user"}, file)
	assert.Error(t, err)
}

func TestImport_EnsureBaseSuccess(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title\nA\n")
	newBaseID := uuid.New()
	mockBase.On("CreateBaseWithoutTable", mock.Anything, mock.Anything, "schema", "user").
		Return(tenant.Base{ID: newBaseID}, nil)

	var captured dto.CreateTableRequest
	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").
		Run(func(args mock.Arguments) { captured = args.Get(1).(dto.CreateTableRequest) }).
		Return(dto.TableResponse{}, errors.New("stop"))

	svc := services.NewImportService(mockTable, mockBase, nil)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{Title: "T", WorkspaceID: "ws", CreatedBy: "user"}, file)

	assert.Error(t, err)
	assert.Equal(t, newBaseID.String(), captured.BaseID)
}

func TestImport_ParseCSVError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	svc := services.NewImportService(mockTable, mockBase, nil)

	file := makeFileHeader(t, "data.csv", "")
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)
	assert.Error(t, err)
}

func TestImport_ParseCSVOpenError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	svc := services.NewImportService(mockTable, mockBase, nil)
	badHeader := &multipart.FileHeader{Filename: "missing.csv"}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, badHeader)

	assert.Error(t, err)
}

func TestImport_ParseCSVReadError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	svc := services.NewImportService(mockTable, mockBase, nil)
	file := makeFileHeader(t, "data.csv", "Title\n\"bad")
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_CreateTableError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title\nA\n")
	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").
		Return(dto.TableResponse{}, errors.New("fail"))

	svc := services.NewImportService(mockTable, mockBase, nil)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_UpdateTitleColumnNoTitleName(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", ",ColA\n,1\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	// Mock UpdateColumn in case the empty header still triggers the update
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil).Maybe()
	mockTable.On("AddColumn", mock.Anything, "schema", mock.Anything).
		Return(dto.ColumnResponse{}, errors.New("add fail"))

	svc := services.NewImportService(mockTable, mockBase, nil)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_UpdateTitleColumnNoTitleFound(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Name,ColA\nA,1\n")
	resp := baseTableResponse()
	resp.Columns = []dto.ColumnResponse{{Title: "NotTitle", ID: uuid.New()}}

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("AddColumn", mock.Anything, "schema", mock.Anything).
		Return(dto.ColumnResponse{}, errors.New("add fail"))

	svc := services.NewImportService(mockTable, mockBase, nil)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_UpdateTitleColumnError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Name\nA\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, errors.New("fail"))

	svc := services.NewImportService(mockTable, mockBase, nil)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_AddColumnsError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,ColA\nA,1\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.On("AddColumn", mock.Anything, "schema", mock.Anything).
		Return(dto.ColumnResponse{}, errors.New("add fail"))

	svc := services.NewImportService(mockTable, mockBase, nil)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_InsertBatchError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,ColA\nA,1\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.On("AddColumn", mock.Anything, "schema", mock.Anything).
		Return(dto.ColumnResponse{ColumnName: "cola", ID: uuid.New()}, nil)
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Return([]dto.RecordResponse{}, errors.New("insert fail"))

	svc := services.NewImportService(mockTable, mockBase, nil)
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.Error(t, err)
}

func TestImport_RefreshTableError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,ColA\nA,1\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.On("AddColumn", mock.Anything, "schema", mock.Anything).
		Return(dto.ColumnResponse{ColumnName: "cola", ID: uuid.New()}, nil)
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").
		Return(dto.TableResponse{}, errors.New("refresh fail"))

	svc := services.NewImportService(mockTable, mockBase, nil)
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.Equal(t, resp.Model.ID, result.TableResponse.Model.ID)
}

func TestImport_SuccessAndTypes(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	longText := strings.Repeat("a", 300)
	csv := strings.Join([]string{
		"Title,BoolCol,IntCol,DecCol,DateCol,LongTextCol,EmptyCol,TextCol,BigIntCol,,BoolFalseCol",
		"Task,yes,123,12.5,2006-01-02," + longText + ",,hello,2147483648,skipme,no",
		"Task2,true,-5,3.14,02-01-2006," + longText + ",,world,9223372036854775807,skipme,false",
	}, "\n")

	file := makeFileHeader(t, "data.csv", csv)
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}

	var captured []map[string]interface{}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Run(func(args mock.Arguments) { captured = args.Get(3).([]map[string]interface{}) }).
		Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").
		Return(resp, nil)

	svc := services.NewImportService(mockTable, mockBase, nil)
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.Equal(t, resp.Model.ID, result.TableResponse.Model.ID)
	assert.Len(t, captured, 2)

	row := captured[0]
	assert.Equal(t, "Task", row["title"])
	assert.Equal(t, true, row["\"bool_col\""])
	assert.Equal(t, int64(123), row["\"int_col\""])
	assert.Equal(t, 12.5, row["\"dec_col\""])
	assert.Equal(t, "2006-01-02", row["\"date_col\""])
	assert.Equal(t, longText, row["\"long_text_col\""])
	assert.Equal(t, "hello", row["\"text_col\""])
	assert.Equal(t, 2147483648.0, row["\"big_int_col\""])
	assert.Equal(t, false, row["\"bool_false_col\""])
}
