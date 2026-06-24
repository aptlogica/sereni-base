package table_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime/multipart"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	antivirusInterfaces "github.com/aptlogica/sereni-base/internal/providers/antivirus/interfaces"
	svcInterfaces "github.com/aptlogica/sereni-base/internal/services/interfaces"
	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

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
func (m *MockTableManagementService) UpdateAttachment(ctx context.Context, schemaName string, req dto.UpdateAttachmentRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (m *MockTableManagementService) BulkDeleteRows(ctx context.Context, schemaName string, req dto.BulkDeleteRowsRequest) (int, error) {
	return 0, nil
}
func (m *MockTableManagementService) RemoveAttachments(ctx context.Context, schemaName string, req dto.RemoveAttachmentsRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (m *MockTableManagementService) BulkUpdateColumns(ctx context.Context, schemaName string, modelID string, columnID string, updates []dto.UpdateColumnsRequest) error {
	args := m.Called(ctx, schemaName, modelID, columnID, updates)
	return args.Error(0)
}
func (m *MockTableManagementService) ResetColumnValues(ctx context.Context, schemaName string, modelID string, columnID string) error {
	args := m.Called(ctx, schemaName, modelID, columnID)
	return args.Error(0)
}

// ValidateColumnsAllowed validates multiple columns are allowed for operations like split
func (m *MockTableManagementService) ValidateColumnsAllowed(ctx context.Context, schemaName string, modelID string, columnIDs []string) error {
	args := m.Called(ctx, schemaName, modelID, columnIDs)
	return args.Error(0)
}

// ValidateColumnAllowedForSplit validates a single column is allowed for splitting
func (m *MockTableManagementService) ValidateColumnAllowedForSplit(ctx context.Context, schemaName string, modelID string, columnID string) error {
	args := m.Called(ctx, schemaName, modelID, columnID)
	return args.Error(0)
}

func (m *MockTableManagementService) ColumnSplit(ctx context.Context, schemaName string, req dto.ColumnSplitRequest) (dto.ColumnSplitResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.ColumnSplitResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.ColumnSplitResponse), args.Error(1)
}

func (m *MockTableManagementService) ExtractSubstring(ctx context.Context, schemaName string, req dto.ExtractSubstringRequest) (dto.ExtractSubstringResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.ExtractSubstringResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.ExtractSubstringResponse), args.Error(1)
}

func (m *MockTableManagementService) FindReplace(ctx context.Context, schemaName string, req dto.FindReplaceRequest) (dto.FindReplaceResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.FindReplaceResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.FindReplaceResponse), args.Error(1)
}

func (m *MockTableManagementService) CaseNormalization(ctx context.Context, schemaName string, req dto.CaseNormalizationRequest) (dto.CaseNormalizationResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.CaseNormalizationResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.CaseNormalizationResponse), args.Error(1)
}

func (m *MockTableManagementService) MergeColumns(ctx context.Context, schemaName string, req dto.MergeColumnsRequest) (dto.MergeColumnsResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.MergeColumnsResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.MergeColumnsResponse), args.Error(1)
}

func (m *MockTableManagementService) TrimWhitespace(ctx context.Context, schemaName string, req dto.TrimWhitespaceRequest) (dto.TrimWhitespaceResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.TrimWhitespaceResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.TrimWhitespaceResponse), args.Error(1)
}

func (m *MockTableManagementService) RemoveSpecialCharacters(ctx context.Context, schemaName string, req dto.RemoveSpecialCharactersRequest) (dto.RemoveSpecialCharactersResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.RemoveSpecialCharactersResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.RemoveSpecialCharactersResponse), args.Error(1)
}

func (m *MockTableManagementService) RemoveFormatting(ctx context.Context, schemaName string, req dto.RemoveFormattingRequest) (dto.RemoveFormattingResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.RemoveFormattingResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.RemoveFormattingResponse), args.Error(1)
}

func (m *MockTableManagementService) RemoveDuplicates(ctx context.Context, schemaName string, req dto.RemoveDuplicatesRequest) (dto.RemoveDuplicatesResponse, error) {
	args := m.Called(ctx, schemaName, req)
	if args.Get(0) == nil {
		return dto.RemoveDuplicatesResponse{}, args.Error(1)
	}
	return args.Get(0).(dto.RemoveDuplicatesResponse), args.Error(1)
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

// importAdapter adapts the current tests (which call svc.Import with a CreateTableRequest)
// to the service's new ImportWithConfig signature.
type importAdapter struct {
	svc svcInterfaces.ImportService
}

func (a importAdapter) Import(ctx context.Context, schema string, create dto.CreateTableRequest, file *multipart.FileHeader) (dto.ImportTableResponse, error) {
	req := dto.ImportWithConfigRequest{
		BaseID:      create.BaseID,
		WorkspaceID: create.WorkspaceID,
		Title:       create.Title,
		Description: create.Description,
		OrderIndex:  create.OrderIndex,
		CreatedBy:   create.CreatedBy,
	}

	// If no explicit column configuration provided (old tests), try to infer columns from CSV headers
	if file != nil {
		f, err := file.Open()
		if err != nil {
			return dto.ImportTableResponse{}, err
		}
		defer f.Close()
		reader := csv.NewReader(f)
		records, err := reader.ReadAll()
		if err == nil && len(records) > 0 {
			headers := records[0]
			dataRows := records[1:]
			cols := make([]dto.ColumnConfig, 0, len(headers))
			for i, h := range headers {
				if i == 0 {
					h = strings.TrimPrefix(h, "\ufeff")
				}
				if strings.TrimSpace(h) == "" {
					continue
				}
				uidt := inferTypeFromRows(dataRows, i)
				cols = append(cols, dto.ColumnConfig{ColumnName: h, Title: h, UIDT: uidt})
			}
			// Default primary to first header
			var primary *dto.ColumnConfig
			if len(cols) > 0 {
				primary = &cols[0]
			}
			req.Config = dto.ImportConfig{Columns: cols, PrimaryColumn: primary}
		}
	}

	// Ensure Config is at least empty but non-nil
	if req.Config.Columns == nil {
		req.Config = dto.ImportConfig{Columns: []dto.ColumnConfig{}}
	}

	return a.svc.ImportWithConfig(ctx, schema, req, file, create.Title)
}

// Type inference helpers (lightweight copy of importService heuristics used in production)
type typeFlags struct {
	isNumber, isDecimal, isBool, isDate, isEmail, isURL, isPhone, isJSON bool
	hasData                                                              bool
	totalLength, count                                                   int
}

func collectTypeFlags(rows [][]string, colIndex int) typeFlags {
	flags := typeFlags{
		isNumber:  true,
		isDecimal: true,
		isBool:    true,
		isDate:    true,
		isEmail:   true,
		isURL:     true,
		isPhone:   true,
		isJSON:    true,
	}
	for _, row := range rows {
		if colIndex >= len(row) {
			continue
		}
		val := row[colIndex]
		if val == "" {
			continue
		}
		flags.hasData = true
		flags.totalLength += len(val)
		flags.count++
		updateTypeFlags(&flags, val)
	}
	return flags
}

func updateTypeFlags(flags *typeFlags, val string) {
	if flags.isNumber || flags.isDecimal {
		flags.isNumber, flags.isDecimal = checkNumericTypes(val, flags.isNumber, flags.isDecimal)
	}
	if flags.isBool {
		flags.isBool = checkBoolType(val)
	}
	if flags.isDate {
		flags.isDate = checkDateType(val)
	}
	if flags.isEmail {
		flags.isEmail = checkEmailType(val)
	}
	if flags.isURL {
		flags.isURL = checkURLType(val)
	}
	if flags.isPhone {
		flags.isPhone = checkPhoneType(val)
	}
	if flags.isJSON {
		flags.isJSON = checkJSONType(val)
	}
}

func checkNumericTypes(val string, isNumber, isDecimal bool) (bool, bool) {
	if v, err := strconv.ParseInt(val, 10, 64); err != nil {
		isNumber = false
	} else {
		if v > math.MaxInt32 || v < math.MinInt32 {
			isNumber = false
		}
	}
	if _, err := strconv.ParseFloat(val, 64); err != nil {
		isDecimal = false
	}
	return isNumber, isDecimal
}

func checkBoolType(val string) bool {
	lower := strings.ToLower(val)
	return lower == "true" || lower == "false" || lower == "0" || lower == "1" || lower == "yes" || lower == "no"
}

func checkDateType(val string) bool {
	formats := []string{"2006-01-02", "02-01-2006", "2006/01/02", "02/01/2006"}
	for _, f := range formats {
		if _, err := time.Parse(f, val); err == nil {
			return true
		}
	}
	return false
}

func checkEmailType(val string) bool {
	return strings.Contains(val, "@") && strings.Contains(val, ".")
}

func checkURLType(val string) bool {
	return strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://")
}

func checkPhoneType(val string) bool {
	// very loose phone check: digits and optional punctuation
	for _, r := range val {
		if !(r >= '0' && r <= '9') && r != '+' && r != '-' && r != ' ' && r != '(' && r != ')' {
			return false
		}
	}
	return true
}

func checkJSONType(val string) bool {
	var js interface{}
	return json.Unmarshal([]byte(val), &js) == nil
}

func inferTypeFromRows(rows [][]string, colIndex int) string {
	flags := collectTypeFlags(rows, colIndex)
	if !flags.hasData {
		return "text"
	}
	avgLength := 0
	if flags.count > 0 {
		avgLength = flags.totalLength / flags.count
	}
	if flags.isNumber {
		return "number"
	}
	if flags.isDecimal {
		return "decimal"
	}
	if flags.isBool {
		return "boolean"
	}
	if flags.isDate {
		return "date"
	}
	if flags.isEmail {
		return "email"
	}
	if flags.isURL {
		return "url"
	}
	if flags.isPhone {
		return "phoneNumber"
	}
	if flags.isJSON {
		return "json"
	}
	if avgLength > 255 {
		return "longText"
	}
	return "text"
}

func TestImport_ScanFileOpenError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}
	mockAV := &MockAntivirusProvider{}

	svcReal := services.NewImportService(mockTable, mockBase, mockAV)
	svc := importAdapter{svc: svcReal}

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

	svcReal := services.NewImportService(mockTable, mockBase, mockAV)
	svc := importAdapter{svc: svcReal}
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

	svcReal := services.NewImportService(mockTable, mockBase, mockAV)
	svc := importAdapter{svc: svcReal}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.Error(t, err)
}

func TestImport_EnsureBaseErrors(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}

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

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{Title: "T", WorkspaceID: "ws", CreatedBy: "user"}, file)

	assert.Error(t, err)
	assert.Equal(t, newBaseID.String(), captured.BaseID)
}

func TestImport_ParseCSVError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}

	file := makeFileHeader(t, "data.csv", "")
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)
	assert.Error(t, err)
}

func TestImport_ParseCSVOpenError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	badHeader := &multipart.FileHeader{Filename: "missing.csv"}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, badHeader)

	assert.Error(t, err)
}

func TestImport_ParseCSVReadError(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
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

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
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

	// In the current ImportWithConfig flow a rows insertion may be attempted;
	// ensure mock handles it to keep behavior stable for this test.
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Return([]dto.RecordResponse{}, errors.New("insert fail"))

	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T"}, file)

	assert.NoError(t, err)
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

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
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

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
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

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
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

	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotNil(t, result.ImportStats)
	assert.Greater(t, result.ImportStats.ErrorRows, 0)
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

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.Equal(t, resp.Model.ID, result.Model.ID)
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

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.Equal(t, resp.Model.ID, result.Model.ID)
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

// TestImport_UpdateTypeFlags tests the updateTypeFlags helper method for type detection
func TestImport_UpdateTypeFlags_NumericType(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Number\nA,123\nB,456\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		// Verify numeric column is detected
		if columnData.Title == "Number" {
			assert.Equal(t, "INTEGER", columnData.DT)
		}
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").
		Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
}

// TestImport_UpdateTypeFlags_BooleanType tests updateTypeFlags with boolean values
func TestImport_UpdateTypeFlags_BooleanType(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,IsActive\nA,true\nB,false\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		// Verify boolean column is detected
		if columnData.Title == "IsActive" {
			assert.Equal(t, "BOOLEAN", columnData.DT)
		}
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").
		Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
}

// TestImport_UpdateTypeFlags_DateType tests updateTypeFlags with date values
func TestImport_UpdateTypeFlags_DateType(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,CreatedDate\nA,2006-01-02\nB,2007-01-02\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		// Verify date column is detected
		if columnData.Title == "CreatedDate" {
			assert.Equal(t, "DATE", columnData.DT)
		}
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").
		Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
}

// TestImport_UpdateTypeFlags_MixedTypes tests updateTypeFlags with values that don't match any specific type
func TestImport_UpdateTypeFlags_MixedTypes(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Mixed\nA,hello\nB,world\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		// Verify text column is detected for non-matching types
		if columnData.Title == "Mixed" {
			assert.Equal(t, "TEXT", columnData.DT)
		}
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").
		Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
}

// TestImport_UpdateTypeFlags_EmailType tests updateTypeFlags with email values
func TestImport_UpdateTypeFlags_EmailType(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Email\nA,user@example.com\nB,admin@example.com\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		// Verify email column is detected
		if columnData.Title == "Email" {
			assert.Equal(t, "TEXT", columnData.DT)
		}
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).
		Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").
		Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	_, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
}

// ============ VALIDATION ERROR TESTS (Exercise validation functions) ============

// Test import with number validation errors
func TestImport_ValidationErrors_InvalidNumber(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Age\nJohn,abc\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	// Should succeed but with errors
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with email validation
func TestImport_ValidationErrors_InvalidEmail(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Email\nJohn,invalidemail\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with boolean validation
func TestImport_ValidationErrors_InvalidBoolean(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Active\nJohn,maybe\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with JSON validation
func TestImport_ValidationErrors_InvalidJSON(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Metadata\nJohn,{invalid}\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with decimal validation
func TestImport_ValidationErrors_InvalidDecimal(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Price\nItem,notanumber\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with multiple rows to exercise error aggregation
func TestImport_MultipleValidationErrors(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	csvData := "Title,Age,Email\nJohn,abc,invalid\nJane,25,user@example.com\nBob,notnum,noemail\n"
	file := makeFileHeader(t, "data.csv", csvData)
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with CSV escaping (comma, quote, newline in data)
func TestImport_CSVEscaping_CommaInValue(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Description\n\"Full Name, Inc\",\"Good, affordable\"\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with very large numbers
func TestImport_LargeNumbers(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Amount\nTransaction,9999999999999999\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with special characters and unicode
func TestImport_SpecialCharactersAndUnicode(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,Description\nProduct,中文 Special @#$%\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with mixed boolean formats
func TestImport_MixedBooleanFormats(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	csvData := "Title,Flag1,Flag2,Flag3\nItem,true,yes,0\n"
	file := makeFileHeader(t, "data.csv", csvData)
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with empty values
func TestImport_EmptyAndNullValues(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	file := makeFileHeader(t, "data.csv", "Title,OptionalField\nItem,\n")
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

// Test import with decimal numbers vs integers
func TestImport_DecimalVsIntegerNumbers(t *testing.T) {
	mockTable := &MockTableManagementService{}
	mockBase := &MockBaseManagementService{}

	csvData := "Title,IntField,DecimalField\nItem,42,3.14\n"
	file := makeFileHeader(t, "data.csv", csvData)
	resp := baseTableResponse()

	mockTable.On("CreateTableWithDefaults", mock.Anything, mock.Anything, "schema").Return(resp, nil)
	mockTable.On("UpdateColumn", mock.Anything, "schema", mock.Anything, mock.Anything).Return(dto.ColumnResponse{}, nil)
	mockTable.AddColumnFn = func(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
		return dto.ColumnResponse{ID: uuid.New(), ColumnName: helpers.ToSnakeCase(columnData.Title)}, nil
	}
	mockTable.On("CreateRowsWithRecordsBulk", mock.Anything, "schema", resp.Model.Alias, mock.Anything).Return([]dto.RecordResponse{}, nil)
	mockTable.On("GetTableByID", mock.Anything, resp.Model.ID.String(), "schema").Return(resp, nil)

	svcReal := services.NewImportService(mockTable, mockBase, nil)
	svc := importAdapter{svc: svcReal}
	result, err := svc.Import(context.Background(), "schema", dto.CreateTableRequest{BaseID: "base", Title: "T", CreatedBy: "user"}, file)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}
