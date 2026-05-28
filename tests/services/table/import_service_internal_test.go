package table_test

import (
	"context"
	"errors"
	"math"
	"mime/multipart"
	"strconv"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	antivirusProviderInterface "github.com/aptlogica/sereni-base/internal/providers/antivirus/interfaces"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	svcInterfaces "github.com/aptlogica/sereni-base/internal/services/interfaces"
	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type importServiceTestAPI interface {
	InferColumnTypes(headers []string, rows [][]string) []string
	InferType(rows [][]string, colIndex int) string
	CheckPhoneType(val string) bool
	CheckDateType(val string) bool
	CheckJSONType(val string) bool
	GetDatabaseType(uidt string) string
	ConvertValue(val string, typeName string) interface{}
	ConvertDateToISO(val string) string
	FindUniqueName(proposedName string, existingNames []string, maxLength int) string
	GetUniqueTableName(ctx context.Context, schemaName string, baseID string, proposedName string, lg *zerolog.Logger) (string, error)
	GetUniqueBaseName(ctx context.Context, schemaName string, workspaceID string, proposedName string, lg *zerolog.Logger) (string, error)
	EnsureBase(ctx context.Context, schemaName string, req *dto.CreateTableRequest, lg *zerolog.Logger, tableTitle string) error
	EnsureBaseWithConfig(ctx context.Context, schemaName string, req *dto.ImportWithConfigRequest, lg *zerolog.Logger, tableTitle string) error
	CleanupBaseIfNeeded(ctx context.Context, schemaName string, baseID string, createdBase bool, lg *zerolog.Logger)
	CleanData(rows [][]string, settings dto.ImportSettings) [][]string
	IsRowEmpty(row []string) bool
	RemoveEmptyRows(rows [][]string) [][]string
	RemoveDuplicateRecords(rows [][]string) [][]string
	CountEmptyRows(rows [][]string) int
	CountDuplicateRecords(rows [][]string) int
	IdentifyEmptyRowsWithLineNumbers(rows [][]string) map[int][]string
	IdentifyDuplicateRowsWithLineNumbers(rows [][]string) map[int][]string
	EscapeCSVCell(cell string) string
	EscapeRowForCSV(row []string) []string
	SortedLineNumbers(m map[int][]string) []int
	BuildErrorTypeSummary(errorMessages []string) string
	BuildAllValidationErrorsBlock(errorMessages []string) string
	BuildEmptyRowsHumanSection(emptyRowsWithLineNumbers map[int][]string) string
	BuildDuplicateRowsHumanSection(duplicateRowsWithLineNumbers map[int][]string) string
	BuildRawCSVSection(headers []string, errorRows [][]string, emptyRowsWithLineNumbers map[int][]string, duplicateRowsWithLineNumbers map[int][]string) string
	SaveErrorRows(headers []string, errorRows [][]string, errorMessages []string, emptyRowsWithLineNumbers map[int][]string, duplicateRowsWithLineNumbers map[int][]string, lg *zerolog.Logger) (string, error)
	GetDefaultValue(cfg *dto.ColumnConfig) string
	ValidateNumberField(cellVal string, columnName string, meta map[string]interface{}) []string
	ValidateDecimalField(cellVal string, columnName string, meta map[string]interface{}) []string
	ValidateBooleanField(cellVal string, columnName string) []string
	ValidateEmailField(cellVal string, columnName string) []string
	ValidateJSONField(cellVal string, columnName string) []string
	ValidateTextField(cellVal string, columnName string, fieldType string, meta map[string]interface{}) []string
	BuildRecordsWithConfigAndErrors(params dto.BuildRecordsWithConfigAndErrorsParams, lg *zerolog.Logger) ([]map[string]interface{}, [][]string, []string)
	ProcessTitleCell(header string, cfg dto.ColumnConfig, cellVal string) (interface{}, []string)
	ProcessDataCell(header string, cfg dto.ColumnConfig, colResp dto.ColumnResponse, cellVal string) (string, interface{}, []string, bool)
	ApplyNonPrimary(record map[string]interface{}, header string, cfg dto.ColumnConfig, configExists bool, params dto.BuildRecordsWithConfigAndErrorsParams, i int, cellVal string) ([]string, bool)
	ValidateColumnConfig(columnConfigs []dto.ColumnConfig, primary *dto.ColumnConfig, headers []string, lg *zerolog.Logger) error
	PrepareImportData(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string, lg *zerolog.Logger) ([]string, [][]string, *dto.ImportStatistics, string, error)
	CreateTableForImport(ctx context.Context, schemaName string, createTableReq dto.CreateTableRequest, lg *zerolog.Logger, createdBase bool, baseID string) (dto.TableResponse, func(), error)
}

func newImportServiceForTest(t *testing.T, tableService svcInterfaces.TableManagementService, baseManagementService svcInterfaces.BaseManagementService, antivirusProvider antivirusProviderInterface.Provider) importServiceTestAPI {
	t.Helper()
	svc := services.NewImportService(tableService, baseManagementService, antivirusProvider)
	helperSvc, ok := svc.(importServiceTestAPI)
	require.True(t, ok)
	return helperSvc
}

type importTableServiceStub struct {
	getModelByBaseID func(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error)
	deleteTable      func(ctx context.Context, schemaName string, modelID string) error
}

type importTableServiceWithCreate struct {
	importTableServiceStub
	createTable func(ctx context.Context, req dto.CreateTableRequest, schemaName string) (dto.TableResponse, error)
}

func (s importTableServiceWithCreate) CreateTableWithDefaults(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
	if s.createTable != nil {
		return s.createTable(ctx, tableData, schemaName)
	}
	return dto.TableResponse{}, nil
}

func (s importTableServiceStub) CreateTableWithDefaults(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
	return dto.TableResponse{}, nil
}
func (s importTableServiceStub) UpdateTable(ctx context.Context, id string, tableData dto.UpdateTableRequest, schemaName string) (dto.TableResponse, error) {
	return dto.TableResponse{}, nil
}
func (s importTableServiceStub) GetTableByID(ctx context.Context, id string, schemaName string) (dto.TableResponse, error) {
	return dto.TableResponse{}, nil
}
func (s importTableServiceStub) GetAllTables(ctx context.Context, schemaName string) ([]dto.TableResponse, error) {
	return nil, nil
}
func (s importTableServiceStub) GetModelByBaseID(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
	if s.getModelByBaseID != nil {
		return s.getModelByBaseID(ctx, schemaName, baseID)
	}
	return nil, nil
}
func (s importTableServiceStub) GetModelByWorkspaceID(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
	return nil, nil
}
func (s importTableServiceStub) DeleteTable(ctx context.Context, schemaName string, modelID string) error {
	if s.deleteTable != nil {
		return s.deleteTable(ctx, schemaName, modelID)
	}
	return nil
}
func (s importTableServiceStub) AddColumn(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error) {
	return dto.ColumnResponse{}, nil
}
func (s importTableServiceStub) GetColumnById(ctx context.Context, schemaName string, id string) (dto.ColumnResponse, error) {
	return dto.ColumnResponse{}, nil
}
func (s importTableServiceStub) GetAllColumns(ctx context.Context, schemaName string) ([]dto.ColumnResponse, error) {
	return nil, nil
}
func (s importTableServiceStub) GetColumnsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ColumnResponse, error) {
	return nil, nil
}
func (s importTableServiceStub) UpdateColumn(ctx context.Context, schemaName string, id string, req dto.ColumnUpdate) (dto.ColumnResponse, error) {
	return dto.ColumnResponse{}, nil
}
func (s importTableServiceStub) DeleteColumn(ctx context.Context, schemaName string, id string) error {
	return nil
}
func (s importTableServiceStub) ReorderColumn(ctx context.Context, schemaName string, req dto.ReorderColumnRequest) ([]dto.ColumnResponse, error) {
	return nil, nil
}
func (s importTableServiceStub) CreateView(ctx context.Context, schemaName string, viewData dto.CreateViewRequest) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (s importTableServiceStub) GetViewByID(ctx context.Context, schemaName string, id string) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (s importTableServiceStub) GetAllViews(ctx context.Context, schemaName string) ([]dto.ViewResponse, error) {
	return nil, nil
}
func (s importTableServiceStub) GetViewsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ViewResponse, error) {
	return nil, nil
}
func (s importTableServiceStub) UpdateView(ctx context.Context, schemaName string, id string, req dto.ViewUpdate) (dto.ViewResponse, error) {
	return dto.ViewResponse{}, nil
}
func (s importTableServiceStub) DeleteView(ctx context.Context, schemaName string, id string) error {
	return nil
}
func (s importTableServiceStub) CreateRow(ctx context.Context, schemaName string, req dto.CreateRowRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s importTableServiceStub) CreateRowWithRecords(ctx context.Context, schemaName string, modelAlias string, record map[string]interface{}) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s importTableServiceStub) CreateRowsWithRecordsBulk(ctx context.Context, schemaName string, modelAlias string, records []map[string]interface{}) ([]dto.RecordResponse, error) {
	return nil, nil
}
func (s importTableServiceStub) GetAllRecords(ctx context.Context, schemaName string, modelID string) (dto.RecordsResponse, error) {
	return dto.RecordsResponse{}, nil
}
func (s importTableServiceStub) InsertRowData(ctx context.Context, schemaName string, req dto.InsertRowDataRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s importTableServiceStub) DeleteRow(ctx context.Context, schemaName string, req dto.DeleteRowDataRequest) error {
	return nil
}
func (s importTableServiceStub) UpdateRawDataForLinks(ctx context.Context, schemaName string, req dto.UpdateRowDataLinksRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s importTableServiceStub) AddAttachment(ctx context.Context, schemaName string, req dto.AddAttachmentRequest, files []*multipart.FileHeader) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s importTableServiceStub) UpdateAttachment(ctx context.Context, schemaName string, req dto.UpdateAttachmentRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s importTableServiceStub) BulkDeleteRows(ctx context.Context, schemaName string, req dto.BulkDeleteRowsRequest) (int, error) {
	return 0, nil
}
func (s importTableServiceStub) RemoveAttachments(ctx context.Context, schemaName string, req dto.RemoveAttachmentsRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}
func (s importTableServiceStub) BulkUpdateColumns(ctx context.Context, schemaName string, modelID string, columnID string, updates []dto.UpdateColumnsRequest) error {
	return nil
}
func (s importTableServiceStub) ResetColumnValues(ctx context.Context, schemaName string, modelID string, columnID string) error {
	return nil
}

type importBaseServiceStub struct {
	createBaseWithoutTable func(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userID string) (tenant.Base, error)
	getBasesByWorkspace    func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error)
	deleteBase             func(ctx context.Context, schemaName string, id string) error
}

func (s importBaseServiceStub) CreateBase(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userID string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s importBaseServiceStub) CreateBaseWithoutTable(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userID string) (tenant.Base, error) {
	if s.createBaseWithoutTable != nil {
		return s.createBaseWithoutTable(ctx, req, schemaName, userID)
	}
	return tenant.Base{}, nil
}
func (s importBaseServiceStub) CreateBaseWithImage(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userID string, fileHeader *multipart.FileHeader) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s importBaseServiceStub) GetBaseByID(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s importBaseServiceStub) GetAllBasesWithAccess(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error) {
	return nil, nil
}
func (s importBaseServiceStub) UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate, userID string, fileHeader *multipart.FileHeader, removeImage string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s importBaseServiceStub) DeleteBase(ctx context.Context, schemaName string, id string) error {
	if s.deleteBase != nil {
		return s.deleteBase(ctx, schemaName, id)
	}
	return nil
}
func (s importBaseServiceStub) GetTablesByBaseId(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
	return nil, nil
}
func (s importBaseServiceStub) GetBasesByWorkspace(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
	if s.getBasesByWorkspace != nil {
		return s.getBasesByWorkspace(ctx, schemaName, workspaceID)
	}
	return nil, nil
}
func (s importBaseServiceStub) AddBaseImage(ctx context.Context, schema string, baseID string, fileHeader *multipart.FileHeader, userID string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s importBaseServiceStub) RemoveBaseImage(ctx context.Context, schema string, baseID string, userID string) (tenant.Base, error) {
	return tenant.Base{}, nil
}
func (s importBaseServiceStub) RemoveUserFromBase(ctx context.Context, schemaName string, baseID string, userID string) error {
	return nil
}

func TestImportServiceTypeInferenceHelpers(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	longText := strings.Repeat("x", 260)
	rows := [][]string{
		{"123", "12.5", "true", "2024-05-27", "user@example.com", "https://example.com", "+1 (555) 111-2222", `{"a":1}`, longText, "", "plain"},
		{"456", "14.75", "false", "27-05-2024", "admin@example.com", "http://example.com", "555-2222", `["x"]`, longText, "", "words"},
		{"", "", "", "", "", "", "", "", "", "", ""},
	}

	got := svc.InferColumnTypes([]string{"n", "d", "b", "date", "email", "url", "phone", "json", "long", "empty", "text"}, rows)
	assert.Equal(t, []string{"number", "decimal", "boolean", "date", "email", "url", "phoneNumber", "json", "longText", "text", "text"}, got)
	assert.False(t, svc.CheckPhoneType("555-abc"))
	assert.False(t, svc.CheckDateType("not-date"))
	assert.False(t, svc.CheckJSONType("{bad"))
	assert.Equal(t, "TEXT", svc.GetDatabaseType("unknown-type"))
	assert.Equal(t, int64(42), svc.ConvertValue("42", "number"))
	assert.Equal(t, 42.5, svc.ConvertValue("42.5", "number"))
	assert.Equal(t, "not-number", svc.ConvertValue("not-number", "number"))
	assert.Equal(t, "not-date", svc.ConvertDateToISO("not-date"))
}

func TestImportServiceValidationHelpers(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	bounds := map[string]interface{}{"min": float64(10), "max": float64(20)}

	assert.Empty(t, svc.ValidateNumberField("12", "Age", bounds))
	assert.NotEmpty(t, svc.ValidateNumberField("12.3", "Age", nil))
	assert.NotEmpty(t, svc.ValidateNumberField("abc", "Age", nil))
	assert.NotEmpty(t, svc.ValidateNumberField(strconvFormatInt64(math.MaxInt32+1), "Age", nil))
	assert.NotEmpty(t, svc.ValidateNumberField("9", "Age", bounds))
	assert.NotEmpty(t, svc.ValidateNumberField("21", "Age", bounds))

	assert.Empty(t, svc.ValidateDecimalField("12.5", "Price", bounds))
	assert.NotEmpty(t, svc.ValidateDecimalField("bad", "Price", nil))
	assert.NotEmpty(t, svc.ValidateDecimalField("9.5", "Price", bounds))
	assert.NotEmpty(t, svc.ValidateDecimalField("20.5", "Price", bounds))

	assert.Empty(t, svc.ValidateBooleanField("yes", "Active"))
	assert.NotEmpty(t, svc.ValidateBooleanField("maybe", "Active"))
	assert.Empty(t, svc.ValidateEmailField("a@b.com", "Email"))
	assert.NotEmpty(t, svc.ValidateEmailField("@b.com", "Email"))
	assert.NotEmpty(t, svc.ValidateEmailField("a@b", "Email"))
	assert.Empty(t, svc.ValidateJSONField(`{"ok":true}`, "Meta"))
	assert.NotEmpty(t, svc.ValidateJSONField(`{bad`, "Meta"))
	assert.Empty(t, svc.ValidateTextField("abc", "Name", "text", map[string]interface{}{"max_length": float64(3)}))
	assert.NotEmpty(t, svc.ValidateTextField("abcd", "Name", "text", map[string]interface{}{"max_length": float64(3)}))
	assert.Equal(t, "", svc.GetDefaultValue(nil))
	assert.Equal(t, "", svc.GetDefaultValue(&dto.ColumnConfig{Meta: map[string]interface{}{"default_value": 12}}))
	assert.Equal(t, "fallback", svc.GetDefaultValue(&dto.ColumnConfig{Meta: map[string]interface{}{"default_value": "fallback"}}))
}

func strconvFormatInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}

func TestImportServiceErrorReportHelpers(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	headers := []string{"Name", "Age"}
	errorRows := [][]string{{`A, "quoted"`, "bad\nvalue"}}
	emptyRows := map[int][]string{4: {"", ""}, 2: {" ", ""}}
	duplicateRows := map[int][]string{5: {"Alice", "10"}}
	messages := []string{
		"Invalid number format",
		"Invalid decimal format",
		"Invalid boolean value",
		"Invalid email format",
		"Invalid JSON format",
		"Text length exceeds maximum",
		"Value 1 is less than minimum 2",
		"value out of range for integer type",
	}

	assert.Equal(t, `"a,b"`, svc.EscapeCSVCell("a,b"))
	assert.Equal(t, `"a""b"`, svc.EscapeCSVCell(`a"b`))
	assert.Equal(t, []string{"plain", `"a,b"`}, svc.EscapeRowForCSV([]string{"plain", "a,b"}))
	assert.Equal(t, []int{2, 4}, svc.SortedLineNumbers(emptyRows))
	assert.Contains(t, svc.BuildErrorTypeSummary(messages), "Invalid Number Format")
	assert.Contains(t, svc.BuildAllValidationErrorsBlock(messages), "[Error Set 1]")
	assert.Contains(t, svc.BuildEmptyRowsHumanSection(emptyRows), "Line 2")
	assert.Contains(t, svc.BuildDuplicateRowsHumanSection(duplicateRows), "Duplicate Row")

	report, err := svc.SaveErrorRows(headers, errorRows, messages, emptyRows, duplicateRows, logger.Get())
	require.NoError(t, err)
	assert.Contains(t, report, "CSV VALIDATION ERROR TYPES")
	assert.Contains(t, report, "# ERROR ROWS:")
	assert.Contains(t, report, "# EMPTY ROWS:")
	assert.Contains(t, report, "# DUPLICATE ROWS:")
}

func TestImportServiceCleaningAndRowHelpers(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	rows := [][]string{
		{"  Alice   Smith  ", " 10 "},
		{"", ""},
		{"Alice", "10"},
		{"Alice", "10"},
		{"  ", "\t"},
	}

	cleaned := svc.CleanData(rows, dto.ImportSettings{TrimSpaces: true, RemoveEmptyRows: true})
	assert.Equal(t, "Alice Smith", cleaned[0][0])
	assert.True(t, svc.IsRowEmpty([]string{" ", "\t"}))
	assert.False(t, svc.IsRowEmpty([]string{"x", ""}))
	assert.Equal(t, [][]string{{"x"}}, svc.RemoveEmptyRows([][]string{{""}, {"x"}}))
	assert.Equal(t, [][]string{{"a"}, {"b"}}, svc.RemoveDuplicateRecords([][]string{{"a"}, {"a"}, {"b"}}))
}

func TestImportServiceUniqueNameAndBaseHelpers(t *testing.T) {
	lg := logger.Get()
	ctx := context.Background()
	baseID := uuid.New()
	createdBaseID := uuid.New()
	deleteCalls := 0

	svc := newImportServiceForTest(t,
		importTableServiceStub{
			getModelByBaseID: func(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
				return []dto.TableResponse{{Model: dto.ModelResponse{Title: "Existing"}}}, nil
			},
		},
		importBaseServiceStub{
			getBasesByWorkspace: func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
				return []tenant.Base{{Title: "Table"}}, nil
			},
			createBaseWithoutTable: func(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userID string) (tenant.Base, error) {
				assert.Contains(t, []string{"Imported_base", "Table 1"}, req.Title)
				return tenant.Base{ID: createdBaseID}, nil
			},
			deleteBase: func(ctx context.Context, schemaName string, id string) error {
				deleteCalls++
				return errors.New("delete failed")
			},
		},
		nil,
	)

	assert.Equal(t, "New", svc.FindUniqueName("New", []string{"Existing"}, 50))
	assert.Equal(t, "Existing 1", svc.FindUniqueName("Existing", []string{"Existing"}, 50))
	assert.Equal(t, "A", svc.FindUniqueName("A", []string{"A"}, 1))
	assert.Len(t, svc.FindUniqueName(strings.Repeat("x", 60), nil, 50), 50)

	tableName, err := svc.GetUniqueTableName(ctx, "schema", "base", "Existing", lg)
	require.NoError(t, err)
	assert.Equal(t, "Existing 1", tableName)

	baseName, err := svc.GetUniqueBaseName(ctx, "schema", "workspace", "Table_base", lg)
	require.NoError(t, err)
	assert.Equal(t, "Table 1", baseName)

	createReq := dto.CreateTableRequest{WorkspaceID: "workspace", CreatedBy: "user"}
	require.NoError(t, svc.EnsureBase(ctx, "schema", &createReq, lg, "Imported"))
	assert.Equal(t, createdBaseID.String(), createReq.BaseID)

	configReq := dto.ImportWithConfigRequest{WorkspaceID: "workspace", CreatedBy: "user"}
	require.NoError(t, svc.EnsureBaseWithConfig(ctx, "schema", &configReq, lg, "Table"))
	assert.Equal(t, createdBaseID.String(), configReq.BaseID)

	svc.CleanupBaseIfNeeded(ctx, "schema", baseID.String(), true, lg)
	assert.Equal(t, 1, deleteCalls)
}

func TestImportServiceUniqueNameErrorBranches(t *testing.T) {
	lg := logger.Get()
	ctx := context.Background()
	createErr := errors.New("create failed")

	svc := newImportServiceForTest(t,
		importTableServiceStub{
			getModelByBaseID: func(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
				return nil, errors.New("tables failed")
			},
		},
		importBaseServiceStub{
			getBasesByWorkspace: func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
				return nil, errors.New("bases failed")
			},
			createBaseWithoutTable: func(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userID string) (tenant.Base, error) {
				return tenant.Base{}, createErr
			},
		},
		nil,
	)

	_, err := svc.GetUniqueTableName(ctx, "schema", "base", "Name", lg)
	assert.Error(t, err)
	baseName, err := svc.GetUniqueBaseName(ctx, "schema", "workspace", strings.Repeat("x", 60), lg)
	assert.Error(t, err)
	assert.Equal(t, strings.Repeat("x", 60), baseName)

	tableReq := dto.CreateTableRequest{}
	assert.Error(t, svc.EnsureBase(ctx, "schema", &tableReq, lg, "Table"))
	tableReq.WorkspaceID = "workspace"
	assert.ErrorIs(t, svc.EnsureBase(ctx, "schema", &tableReq, lg, "Table"), createErr)

	configReq := dto.ImportWithConfigRequest{WorkspaceID: "workspace", CreatedBy: "user"}
	err = svc.EnsureBaseWithConfig(ctx, "schema", &configReq, lg, strings.Repeat("x", 60))
	assert.ErrorIs(t, err, createErr)
}

func TestImportServiceBuildRecordsWithConfigBranches(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	lg := logger.Get()
	titleDefault := "Untitled"
	numberDefault := "18"
	params := dto.BuildRecordsWithConfigAndErrorsParams{
		Headers: []string{"Title", "Age", "Ignored", "MissingMap"},
		Req:     dto.CreateTableRequest{CreatedBy: "user"},
		Primary: &dto.ColumnConfig{
			ColumnName: "Title",
			UIDT:       "text",
			Meta:       map[string]interface{}{"default_value": titleDefault},
		},
		ColumnConfigs: []dto.ColumnConfig{
			{ColumnName: "Title", UIDT: "text", Meta: map[string]interface{}{"default_value": titleDefault}},
			{ColumnName: "Age", UIDT: "number", Meta: map[string]interface{}{"default_value": numberDefault}},
			{ColumnName: "MissingMap", UIDT: "text"},
		},
		ColumnMap: map[int]dto.ColumnResponse{
			1: {ColumnName: "age"},
		},
		DataRows: [][]string{
			{"Alice", "42", "skip", "no column map", "extra cell"},
			{"", "", "skip", ""},
			{"Bob", "bad", "skip", ""},
			{"", "20", "skip", ""},
		},
	}

	records, errorRows, errorMessages := svc.BuildRecordsWithConfigAndErrors(params, lg)
	require.Len(t, records, 3)
	require.Len(t, errorRows, 1)
	require.Len(t, errorMessages, 1)
	assert.Equal(t, "Alice", records[0]["title"])
	assert.Equal(t, int64(42), records[0]["\"age\""])
	assert.Equal(t, titleDefault, records[1]["title"])
	assert.Equal(t, int64(18), records[1]["\"age\""])
	assert.Contains(t, errorMessages[0], "Column type 'number'")

	_, errs := svc.ProcessTitleCell("Title", dto.ColumnConfig{UIDT: "number"}, "not-number")
	assert.NotEmpty(t, errs)

	key, val, errs, ok := svc.ProcessDataCell("Age", dto.ColumnConfig{UIDT: "number"}, dto.ColumnResponse{ColumnName: "age"}, "")
	assert.False(t, ok)
	assert.Empty(t, key)
	assert.Nil(t, val)
	assert.Empty(t, errs)

	errs, applied := svc.ApplyNonPrimary(map[string]interface{}{}, "Unknown", dto.ColumnConfig{}, false, params, 1, "x")
	assert.False(t, applied)
	assert.Empty(t, errs)
}

func TestImportServiceConfigValidationBranches(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	lg := logger.Get()
	headers := []string{"Title", "Age"}

	assert.Error(t, svc.ValidateColumnConfig(nil, nil, headers, lg))
	assert.Error(t, svc.ValidateColumnConfig([]dto.ColumnConfig{{ColumnName: "Title"}}, nil, headers, lg))
	assert.Error(t, svc.ValidateColumnConfig([]dto.ColumnConfig{{ColumnName: "Title"}}, &dto.ColumnConfig{}, headers, lg))
	assert.Error(t, svc.ValidateColumnConfig([]dto.ColumnConfig{{ColumnName: "Title"}}, &dto.ColumnConfig{ColumnName: "Missing"}, headers, lg))
	assert.Error(t, svc.ValidateColumnConfig([]dto.ColumnConfig{{}}, &dto.ColumnConfig{ColumnName: "Title"}, headers, lg))
	assert.Error(t, svc.ValidateColumnConfig([]dto.ColumnConfig{{ColumnName: "Missing"}}, &dto.ColumnConfig{ColumnName: "Title"}, headers, lg))
	assert.NoError(t, svc.ValidateColumnConfig(
		[]dto.ColumnConfig{{ColumnName: "Title"}, {ColumnName: "Age", Title: "Age", UIDT: "number"}},
		&dto.ColumnConfig{ColumnName: "Title"},
		headers,
		lg,
	))
}

func TestImportServicePrepareImportDataBranches(t *testing.T) {
	ctx := context.Background()
	lg := logger.Get()
	file := makeFileHeader(t, "data.csv", "Title,Age\n Alice ,10\n,\n Alice ,10\n")
	req := dto.ImportWithConfigRequest{
		BaseID:    "base",
		CreatedBy: "user",
		Config: dto.ImportConfig{
			PrimaryColumn: &dto.ColumnConfig{ColumnName: "Title", UIDT: "text"},
			Columns: []dto.ColumnConfig{
				{ColumnName: "Title", UIDT: "text"},
				{ColumnName: "Age", UIDT: "number"},
			},
			Settings: dto.ImportSettings{
				TrimSpaces:             true,
				RemoveEmptyRows:        true,
				RemoveDuplicateRecords: true,
			},
		},
	}
	svc := newImportServiceForTest(t,
		importTableServiceStub{
			getModelByBaseID: func(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
				return []dto.TableResponse{{Model: dto.ModelResponse{Title: "Table"}}}, nil
			},
		},
		nil,
		nil,
	)

	headers, rows, stats, uniqueTitle, err := svc.PrepareImportData(ctx, "schema", req, file, "Table", lg)
	require.NoError(t, err)
	assert.Equal(t, []string{"Title", "Age"}, headers)
	assert.Equal(t, [][]string{{"Alice", "10"}}, rows)
	assert.Equal(t, "Table 1", uniqueTitle)
	assert.Equal(t, 1, stats.EmptyRowsSkipped)
	assert.Equal(t, 1, stats.DuplicatesRemoved)

	errSvc := newImportServiceForTest(t,
		importTableServiceStub{
			getModelByBaseID: func(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
				return nil, errors.New("name lookup failed")
			},
		},
		nil,
		nil,
	)
	_, _, _, fallbackTitle, err := errSvc.PrepareImportData(ctx, "schema", req, file, "Table", lg)
	require.NoError(t, err)
	assert.Equal(t, "Table", fallbackTitle)
}

func TestImportServiceCreateTableForImportCleanup(t *testing.T) {
	ctx := context.Background()
	lg := logger.Get()
	modelID := uuid.New()
	deleteTableCalls := 0
	deleteBaseCalls := 0

	svc := newImportServiceForTest(t,
		importTableServiceWithCreate{
			importTableServiceStub: importTableServiceStub{
				deleteTable: func(ctx context.Context, schemaName string, modelID string) error {
					deleteTableCalls++
					return errors.New("delete table failed")
				},
			},
			createTable: func(ctx context.Context, req dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
				return dto.TableResponse{Model: dto.ModelResponse{ID: modelID, Alias: "alias"}}, nil
			},
		},
		importBaseServiceStub{
			deleteBase: func(ctx context.Context, schemaName string, id string) error {
				deleteBaseCalls++
				return errors.New("delete base failed")
			},
		},
		nil,
	)

	_, cleanup, err := svc.CreateTableForImport(ctx, "schema", dto.CreateTableRequest{Title: "Table"}, lg, true, "base")
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	cleanup()
	assert.Equal(t, 1, deleteTableCalls)
	assert.Equal(t, 1, deleteBaseCalls)

	createErrSvc := newImportServiceForTest(t,
		importTableServiceWithCreate{
			createTable: func(ctx context.Context, req dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
				return dto.TableResponse{}, errors.New("create table failed")
			},
		},
		importBaseServiceStub{
			deleteBase: func(ctx context.Context, schemaName string, id string) error {
				deleteBaseCalls++
				return nil
			},
		},
		nil,
	)
	_, cleanup, err = createErrSvc.CreateTableForImport(ctx, "schema", dto.CreateTableRequest{Title: "Table"}, lg, true, "base")
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Equal(t, 2, deleteBaseCalls)
}
