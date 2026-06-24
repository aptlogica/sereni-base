package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type bulkCapableTableService struct {
	*mocks.MockTableManagementService
	createRowsWithValues func(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error)
}

// minimalSvc provides ValidateColumnsAllowed but does NOT implement CaseNormalization,
// so it can be used to exercise the NotImplemented path in handlers.
type minimalSvc struct{}

func (m *minimalSvc) ValidateColumnsAllowed(ctx context.Context, schemaName string, modelID string, columns []string) error {
	return nil
}

func (s *bulkCapableTableService) CreateRowsWithValues(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error) {
	return s.createRowsWithValues(ctx, schemaName, modelID, rowsInput, createdBy, updatedBy)
}

func TestNewTableHandler(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)
	assert.NotNil(t, handler)
}

func TestTableHandler_CreateTable_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/table", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test_schema")
	c.Set("user_id", "user123")

	handler.CreateTable(c)

	assert.NotEqual(t, 200, w.Code)
}

// Test for ImportTable - currently at 0% coverage
func TestTableHandler_ImportTable_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Initialize config for test
	orig := config.AppConfig
	config.AppConfig = &config.Config{Import: config.ImportConfig{MaxSize: 2097152}}
	defer func() { config.AppConfig = orig }()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "data").
		Return(dto.ImportTableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_NoFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orig := config.AppConfig
	config.AppConfig = &config.Config{Import: config.ImportConfig{MaxSize: 2097152}}
	defer func() { config.AppConfig = orig }()

	handler := handlers.NewTableHandler(nil, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("baseId", uuid.New().String())
	writer.WriteField("tableName", "Test Table")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.ImportTableWithConfig(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTableHandler_ImportTable_NoBaseId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orig := config.AppConfig
	config.AppConfig = &config.Config{Import: config.ImportConfig{MaxSize: 2097152}}
	defer func() { config.AppConfig = orig }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	// Expect ImportWithConfig to be called but return error for missing base_id
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(dto.ImportTableResponse{}, errors.New("base_id is required")).AnyTimes()
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("title", "Test Table")
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)

	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_InvalidConfigJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orig := config.AppConfig
	config.AppConfig = &config.Config{Import: config.ImportConfig{MaxSize: 2097152}}
	defer func() { config.AppConfig = orig }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	// invalid JSON
	writer.WriteField("config", "{invalid")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_PrimaryColumnNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orig := config.AppConfig
	config.AppConfig = &config.Config{Import: config.ImportConfig{MaxSize: 2097152}}
	defer func() { config.AppConfig = orig }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[{"column_name":"other"}]}`)
	writer.WriteField("primary_column", "missing")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_FileTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// temporarily set import max size to 1 byte
	orig := config.AppConfig
	config.AppConfig = &config.Config{Import: config.ImportConfig{MaxSize: 1}}
	defer func() { config.AppConfig = orig }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	// ensure content > 1 byte
	io.WriteString(part, strings.Repeat("a", 10))
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_PrimaryColumnFound_OrderIndexParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)

	// Expect ImportWithConfig and inspect the request
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			if req.Config.PrimaryColumn == nil {
				return dto.ImportTableResponse{}, errors.New("primary not set")
			}
			// order index should be parsed to float64
			if req.OrderIndex != 2.0 && req.OrderIndex != 2.5 {
				return dto.ImportTableResponse{}, errors.New("order index parse failed")
			}
			// tableTitle should have extension removed
			if tableTitle == "myfile.csv" {
				return dto.ImportTableResponse{}, errors.New("title not trimmed")
			}
			return dto.ImportTableResponse{}, nil
		})

	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	// Test with integer order_index
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "myfile.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[{"column_name":"id"}]}`)
	writer.WriteField("primary_column", "id")
	writer.WriteField("order_index", "2")
	writer.WriteField("title", "myfile.csv")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Test with float order_index
	ctrl.Finish()
	ctrl = gomock.NewController(t)
	mockTableService = mocks.NewMockTableManagementService(ctrl)
	mockImportService = mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			if req.Config.PrimaryColumn == nil {
				return dto.ImportTableResponse{}, errors.New("primary not set")
			}
			if req.OrderIndex != 2.5 {
				return dto.ImportTableResponse{}, errors.New("order index parse failed")
			}
			if tableTitle != "myfile" {
				return dto.ImportTableResponse{}, errors.New("title not trimmed")
			}
			return dto.ImportTableResponse{}, nil
		})

	handler = handlers.NewTableHandler(mockTableService, mockImportService)

	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	part, _ = writer.CreateFormFile("file", "myfile.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[{"column_name":"id"}]}`)
	writer.WriteField("primary_column", "id")
	writer.WriteField("order_index", "2.5")
	writer.WriteField("title", "myfile.csv")
	writer.Close()

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_NoConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	// NOTE: no config field provided
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_RejectsNonCSVFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := handlers.NewTableHandler(mocks.NewMockTableManagementService(ctrl), mocks.NewMockImportService(ctrl))

	tests := []string{"data.txt", "data.exe", "data.php", "data.rtf"}
	for _, fileName := range tests {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", fileName)
		io.WriteString(part, "not,csv")
		writer.WriteField("base_id", uuid.New().String())
		writer.WriteField("config", `{"columns":[]}`)
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/import", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		c.Set("schema", "test")
		c.Set("user_id", "user123")

		handler.ImportTableWithConfig(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Only CSV files are allowed")
	}
}

// ============ Additional Import API Test Cases ============

func TestTableHandler_ImportTable_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(dto.ImportTableResponse{}, errors.New("database error"))
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_MissingSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), "", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(dto.ImportTableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	// NOTE: No schema set in context
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(dto.ImportTableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	// NOTE: No user_id set in context

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_EmptyTitle_UsesFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "my_data").
		Return(dto.ImportTableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "my_data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	// NOTE: No title provided
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_WithDescription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "data").
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			// tableTitle should be computed from filename (extension stripped), not from form fields
			if tableTitle != "data" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should be 'data', got " + tableTitle)
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.WriteField("description", "Table description")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_TitleNotUsedFromFormField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "myfile").
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			// Verify tableTitle comes from filename, not from form field
			if tableTitle == "CustomTitle" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should not use form field value")
			}
			if tableTitle != "myfile" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should be 'myfile' from filename")
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "myfile.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.WriteField("title", "CustomTitle")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_WithWorkspaceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	workspaceID := uuid.New().String()
	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			if req.WorkspaceID != workspaceID {
				return dto.ImportTableResponse{}, errors.New("workspace_id not set correctly")
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("workspace_id", workspaceID)
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_WithMultipleColumns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "data").
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			if len(req.Config.Columns) != 3 {
				return dto.ImportTableResponse{}, errors.New("columns count not correct")
			}
			if tableTitle != "data" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should be 'data' from filename")
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[{"column_name":"id","column_type":"text"},{"column_name":"name","column_type":"text"},{"column_name":"email","column_type":"email"}]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_SpecialCharactersInTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "data").
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			// tableTitle is computed from filename (extension stripped), not from form payload
			if tableTitle != "data" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should be 'data' from filename, not form field")
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.WriteField("title", "Table@#$%^&*()")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_IntegerOrderIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "data").
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			if req.OrderIndex != 5.0 {
				return dto.ImportTableResponse{}, errors.New("order index not set correctly")
			}
			if tableTitle != "data" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should be 'data' from filename")
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.WriteField("order_index", "5")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_InvalidOrderIndexFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "data").
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			// Invalid format should default to 0
			if req.OrderIndex != 0.0 {
				return dto.ImportTableResponse{}, errors.New("order index should default to 0 for invalid format")
			}
			if tableTitle != "data" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should be 'data' from filename")
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.WriteField("order_index", "invalid_number")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_MultipleDotsInFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "archive.backup").
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			// Only the last extension (.csv) should be stripped
			if tableTitle != "archive.backup" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should be 'archive.backup', got " + tableTitle)
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "archive.backup.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_LargeValidFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Set import max size to 10MB to allow test file
	orig := config.AppConfig
	config.AppConfig = &config.Config{Import: config.ImportConfig{MaxSize: 10485760}}
	defer func() { config.AppConfig = orig }()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "data").
		Return(dto.ImportTableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	// Create content that's just under max size
	content := strings.Repeat("a,b,c\n", 100000)
	io.WriteString(part, content)
	writer.WriteField("base_id", uuid.New().String())
	writer.WriteField("config", `{"columns":[]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_ImportTable_AllFieldsProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseID := uuid.New().String()
	workspaceID := uuid.New().String()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockImportService := mocks.NewMockImportService(ctrl)
	mockImportService.EXPECT().ImportWithConfig(gomock.Any(), "test_schema", gomock.Any(), gomock.Any(), "data").
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
			if req.BaseID != baseID {
				return dto.ImportTableResponse{}, errors.New("base_id not set correctly")
			}
			if req.WorkspaceID != workspaceID {
				return dto.ImportTableResponse{}, errors.New("workspace_id not set correctly")
			}
			// tableTitle is computed from filename (extension stripped), not from form fields
			if tableTitle != "data" {
				return dto.ImportTableResponse{}, errors.New("tableTitle should be 'data' from filename, got " + tableTitle)
			}
			if req.OrderIndex != 3.5 {
				return dto.ImportTableResponse{}, errors.New("order_index not set correctly")
			}
			if req.CreatedBy != "user123" {
				return dto.ImportTableResponse{}, errors.New("created_by not set correctly")
			}
			return dto.ImportTableResponse{}, nil
		})
	handler := handlers.NewTableHandler(mockTableService, mockImportService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "data.csv")
	io.WriteString(part, "col1,col2\nval1,val2")
	writer.WriteField("base_id", baseID)
	writer.WriteField("workspace_id", workspaceID)
	writer.WriteField("title", "Complete Import")
	writer.WriteField("description", "Detailed description")
	writer.WriteField("order_index", "3.5")
	writer.WriteField("config", `{"columns":[{"column_name":"col1"},{"column_name":"col2"}]}`)
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test_schema")
	c.Set("user_id", "user123")

	handler.ImportTableWithConfig(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_CreateTable_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().CreateTableWithDefaults(gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.CreateTableRequest{
		BaseID:      uuid.New().String(),
		WorkspaceID: "w1",
		Title:       "Table",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/table", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateTable(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_CreateTable_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.CreateTableRequest{Title: "Table"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateTable(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_CreateTable_TitleTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewTableHandler(nil, nil)

	longTitle := string(bytes.Repeat([]byte("a"), 300))
	body, _ := json.Marshal(dto.CreateTableRequest{
		BaseID:      uuid.New().String(),
		WorkspaceID: uuid.New().String(),
		Title:       longTitle,
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateTable(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_CreateTable_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().CreateTableWithDefaults(gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, errors.New("create failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.CreateTableRequest{
		BaseID:      uuid.New().String(),
		WorkspaceID: uuid.New().String(),
		Title:       "Table",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/tables", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateTable(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_UpdateTable_InvalidID(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.UpdateTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateTable_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateTable(gomock.Any(), gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	title := "Updated"
	body, _ := json.Marshal(dto.UpdateTableRequest{Title: &title})
	tableID := uuid.New().String()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/"+tableID, bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateTable(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateTable_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateTable(gomock.Any(), gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, errors.New("update failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	title := "Updated"
	body, _ := json.Marshal(dto.UpdateTableRequest{Title: &title})
	tableID := uuid.New().String()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/"+tableID, bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateTable_TitleTooLong(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	title := string(bytes.Repeat([]byte("a"), 260))
	body, _ := json.Marshal(dto.UpdateTableRequest{Title: &title})
	tableID := uuid.New().String()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/"+tableID, bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: tableID}}

	handler.UpdateTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetTableByID_InvalidID(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/table/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetTableByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetTableByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetTableByID(gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	tableID := uuid.New().String()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/table/"+tableID, nil)
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")

	handler.GetTableByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetTableByID_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetTableByID(gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, errors.New("not found"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	tableID := uuid.New().String()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/table/"+tableID, nil)
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")

	handler.GetTableByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetAllTables_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetAllTables(gomock.Any(), "test").Return([]dto.TableResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tables", nil)
	c.Set("schema", "test")

	handler.GetAllTables(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetAllTables_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetAllTables(gomock.Any(), "test").Return(nil, errors.New("fetch failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tables", nil)
	c.Set("schema", "test")

	handler.GetAllTables(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddColumn_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetTableByID(gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, nil)
	mockTableService.EXPECT().AddColumn(gomock.Any(), "test", gomock.Any()).Return(dto.ColumnResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.AddColumnRequest{
		ModelID: uuid.New(),
		BaseID:  uuid.New(),
		Title:   "Column",
		UIDT:    "text",
		DT:      "text",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.AddColumn(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddColumn_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AddColumn(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddColumn_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetTableByID(gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, nil)
	mockTableService.EXPECT().AddColumn(gomock.Any(), "test", gomock.Any()).Return(dto.ColumnResponse{}, errors.New("add failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.AddColumnRequest{
		ModelID: uuid.New(),
		BaseID:  uuid.New(),
		Title:   "Column",
		UIDT:    "text",
		DT:      "text",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.AddColumn(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddColumn_TableNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetTableByID(gomock.Any(), gomock.Any(), "test").Return(dto.TableResponse{}, app_errors.TableNotFound)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.AddColumnRequest{
		ModelID: uuid.New(),
		BaseID:  uuid.New(),
		Title:   "Column",
		UIDT:    "text",
		DT:      "text",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.AddColumn(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetColumnById_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetColumnById(gomock.Any(), "test", "c1").Return(dto.ColumnResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/columns/c1", nil)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	c.Set("schema", "test")

	handler.GetColumnById(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetAllColumns_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetAllColumns(gomock.Any(), "test").Return([]dto.ColumnResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/columns", nil)
	c.Set("schema", "test")

	handler.GetAllColumns(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetAllColumns_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetAllColumns(gomock.Any(), "test").Return(nil, errors.New("fetch failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/columns", nil)
	c.Set("schema", "test")

	handler.GetAllColumns(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetColumnsByTable_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetColumnsByModelID(gomock.Any(), "test", "m1").Return([]dto.ColumnResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tables/m1/columns", nil)
	c.Params = gin.Params{{Key: "id", Value: "m1"}}
	c.Set("schema", "test")

	handler.GetColumnsByTable(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateView_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().CreateView(gomock.Any(), "test", gomock.Any()).Return(dto.ViewResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.CreateViewRequest{
		ModelID: uuid.New(),
		BaseID:  uuid.New(),
		Title:   "View",
		Type:    "grid",
		Meta:    &map[string]interface{}{},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/views", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateView(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_CreateView_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/views", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateView_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().CreateView(gomock.Any(), "test", gomock.Any()).Return(dto.ViewResponse{}, errors.New("create failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.CreateViewRequest{
		ModelID: uuid.New(),
		BaseID:  uuid.New(),
		Title:   "View",
		Type:    "grid",
		Meta:    &map[string]interface{}{},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/views", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetViewByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetViewByID(gomock.Any(), "test", "v1").Return(dto.ViewResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/views/v1", nil)
	c.Params = gin.Params{{Key: "id", Value: "v1"}}
	c.Set("schema", "test")

	handler.GetViewByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetAllViews_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetAllViews(gomock.Any(), "test").Return([]dto.ViewResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/views", nil)
	c.Set("schema", "test")

	handler.GetAllViews(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetAllViews_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetAllViews(gomock.Any(), "test").Return(nil, errors.New("fetch failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/views", nil)
	c.Set("schema", "test")

	handler.GetAllViews(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetViewsByModelID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetViewsByModelID(gomock.Any(), "test", "m1").Return([]dto.ViewResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tables/m1/views", nil)
	c.Params = gin.Params{{Key: "id", Value: "m1"}}
	c.Set("schema", "test")

	handler.GetViewsByModelID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateView_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateView(gomock.Any(), "test", "v1", gomock.Any()).Return(dto.ViewResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.ViewUpdate{Title: ptrString("Updated")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/views/v1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "v1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateView(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateView_TitleTooLong(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	title := string(bytes.Repeat([]byte("a"), 260))
	body, _ := json.Marshal(dto.ViewUpdate{Title: &title})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/views/v1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "v1"}}

	handler.UpdateView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteView_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().DeleteView(gomock.Any(), "test", "v1").Return(nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/views/v1", nil)
	c.Params = gin.Params{{Key: "id", Value: "v1"}}
	c.Set("schema", "test")

	handler.DeleteView(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateColumn_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateColumn(gomock.Any(), "test", "c1", gomock.Any()).Return(dto.ColumnResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.ColumnUpdate{Title: ptrString("Updated")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/columns/c1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateColumn(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteColumn_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().DeleteColumn(gomock.Any(), "test", "c1").Return(nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/columns/c1", nil)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	c.Set("schema", "test")

	handler.DeleteColumn(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateRow_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().CreateRow(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.CreateRowRequest{ModelID: uuid.New().String()})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateRow(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_CreateRow_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateRow_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().CreateRow(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, errors.New("create failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.CreateRowRequest{ModelID: uuid.New().String()})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateRow_ValidationError(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.CreateRowOrBulkInsertRequest{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateRow(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_CreateRow_BulkSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	modelID := uuid.New().String()
	mockTableService := mocks.NewMockTableManagementService(ctrl)
	handler := handlers.NewTableHandler(&bulkCapableTableService{
		MockTableManagementService: mockTableService,
		createRowsWithValues: func(ctx context.Context, schemaName string, gotModelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error) {
			assert.Equal(t, "test", schemaName)
			assert.Equal(t, modelID, gotModelID)
			assert.Len(t, rowsInput, 2)
			assert.Equal(t, "user123", createdBy)
			assert.Equal(t, "user123", updatedBy)
			return []dto.RecordResponse{{Record: map[string]interface{}{"id": 1}}, {Record: map[string]interface{}{"id": 2}}}, nil
		},
	}, nil)

	body, _ := json.Marshal(dto.CreateRowOrBulkInsertRequest{
		ModelID: modelID,
		Rows: []map[string]interface{}{
			{"name": "alpha"},
			{"name": "beta"},
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateRow(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_CreateRow_BulkUnsupportedService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := handlers.NewTableHandler(mocks.NewMockTableManagementService(ctrl), nil)

	body, _ := json.Marshal(dto.CreateRowOrBulkInsertRequest{
		ModelID: uuid.New().String(),
		Rows:    []map[string]interface{}{{"name": "alpha"}},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateRow_BulkServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	handler := handlers.NewTableHandler(&bulkCapableTableService{
		MockTableManagementService: mockTableService,
		createRowsWithValues: func(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error) {
			return nil, errors.New("bulk failed")
		},
	}, nil)

	body, _ := json.Marshal(dto.CreateRowOrBulkInsertRequest{
		ModelID: uuid.New().String(),
		Rows:    []map[string]interface{}{{"name": "alpha"}},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateRow_SuccessWithMultipleValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	modelID := uuid.New().String()
	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().
		InsertRowData(gomock.Any(), "test", gomock.Any()).
		DoAndReturn(func(ctx context.Context, schemaName string, req dto.InsertRowDataRequest) (dto.RecordResponse, error) {
			assert.Equal(t, modelID, req.ModelID)
			assert.Equal(t, 7, req.RowId)
			assert.Equal(t, "user123", req.UpdatedBy)
			return dto.RecordResponse{Record: map[string]interface{}{"id": req.RowId, "column_id": req.ColumnId}}, nil
		}).
		Times(2)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.UpdateRowRequest{
		ModelID: modelID,
		RowId:   7,
		Values: map[string]interface{}{
			uuid.New().String(): "value",
			uuid.New().String(): nil,
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PATCH", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateRow(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_UpdateRow_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().InsertRowData(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, errors.New("insert failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.UpdateRowRequest{
		ModelID: uuid.New().String(),
		RowId:   7,
		Values:  map[string]interface{}{uuid.New().String(): "value"},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PATCH", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.UpdateRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_InsertRowDataForLinks_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateRawDataForLinks(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.UpdateRowDataLinksRequest{
		ModelID:     uuid.New().String(),
		ColumnId:    uuid.New().String(),
		SourceRowId: 1,
		TargetRowId: 2,
		Action:      "link",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/links", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.InsertRowDataForLinks(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_InsertRowDataForLinks_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/links", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.InsertRowDataForLinks(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_InsertRowDataForLinks_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateRawDataForLinks(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, errors.New("update failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.UpdateRowDataLinksRequest{
		ModelID:     uuid.New().String(),
		ColumnId:    uuid.New().String(),
		SourceRowId: 1,
		TargetRowId: 2,
		Action:      "link",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/links", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.InsertRowDataForLinks(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddAttachment_NoMultipart(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments", nil)

	handler.AddAttachment(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddAttachment_NoFiles(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.AddAttachment(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddAttachment_MissingModelID(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("files", "file.txt")
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("column_id", uuid.New().String())
	_ = writer.WriteField("row_id", "1")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.AddAttachment(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddAttachment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().AddAttachment(gomock.Any(), "test", gomock.Any(), gomock.Any()).Return(dto.RecordResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("files", "file.txt")
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("model_id", uuid.New().String())
	_ = writer.WriteField("column_id", uuid.New().String())
	_ = writer.WriteField("row_id", "1")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.AddAttachment(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_UpdateAttachment_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments/update", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateAttachment(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateAttachment_ValidationError(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.UpdateAttachmentRequest{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments/update", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateAttachment(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateAttachment_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateAttachment(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, errors.New("update failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	title := "updated-title"
	body, _ := json.Marshal(dto.UpdateAttachmentRequest{
		ModelID:  uuid.New().String(),
		ColumnId: uuid.New().String(),
		RowId:    1,
		AssetId:  uuid.New().String(),
		Content:  dto.AssetUpdate{Title: &title},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments/update", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.UpdateAttachment(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateAttachment_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateAttachment(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	title := "updated-title"
	body, _ := json.Marshal(dto.UpdateAttachmentRequest{
		ModelID:  uuid.New().String(),
		ColumnId: uuid.New().String(),
		RowId:    1,
		AssetId:  uuid.New().String(),
		Content:  dto.AssetUpdate{Title: &title},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments/update", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.UpdateAttachment(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_RemoveAttachments_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().RemoveAttachments(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.RemoveAttachmentsRequest{
		ModelID:     uuid.New().String(),
		ColumnId:    uuid.New().String(),
		RowId:       1,
		Attachments: []string{"a1"},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments/remove", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.RemoveAttachments(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetAllRecords_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetAllRecords(gomock.Any(), "test", "m1").Return(dto.RecordsResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tables/m1/records", nil)
	c.Params = gin.Params{{Key: "id", Value: "m1"}}
	c.Set("schema", "test")

	handler.GetAllRecords(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_InsertRowData_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().InsertRowData(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	val := interface{}("value")
	body, _ := json.Marshal(dto.InsertRowDataRequest{
		ModelID:  uuid.New().String(),
		ColumnId: uuid.New().String(),
		RowId:    1,
		Value:    &val,
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/data", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.InsertRowData(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTableHandler_InsertRowData_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/data", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.InsertRowData(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_InsertRowData_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().InsertRowData(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, errors.New("insert failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	val := interface{}("value")
	body, _ := json.Marshal(dto.InsertRowDataRequest{
		ModelID:  uuid.New().String(),
		ColumnId: uuid.New().String(),
		RowId:    1,
		Value:    &val,
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/data", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.InsertRowData(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteRow_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().DeleteRow(gomock.Any(), "test", gomock.Any()).Return(nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.DeleteRowDataRequest{
		ModelID: uuid.New().String(),
		RowId:   1,
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.DeleteRow(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteRow_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/rows", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.DeleteRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteRow_ValidationError(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.DeleteRowDataRequest{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.DeleteRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteTable_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().DeleteTable(gomock.Any(), "test", gomock.Any()).Return(nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	tableID := uuid.New().String()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/tables/"+tableID, nil)
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")

	handler.DeleteTable(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteTable_InvalidID(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/tables/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.DeleteTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteTable_BadUUID(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/tables/bad", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad"}}

	handler.DeleteTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_ReorderColumn_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().ReorderColumn(gomock.Any(), "test", gomock.Any()).Return([]dto.ColumnResponse{}, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.ReorderColumnRequest{
		SourceColumnID: uuid.New(),
		TargetColumnID: uuid.New(),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/reorder", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.ReorderColumn(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_ReorderColumn_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/reorder", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ReorderColumn(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_ReorderColumn_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().ReorderColumn(gomock.Any(), "test", gomock.Any()).Return(nil, errors.New("reorder failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.ReorderColumnRequest{
		SourceColumnID: uuid.New(),
		TargetColumnID: uuid.New(),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/reorder", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.ReorderColumn(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_BulkDeleteRows_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().BulkDeleteRows(gomock.Any(), "test", gomock.Any()).Return(2, nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.BulkDeleteRowsRequest{
		ModelID: uuid.New().String(),
		RowIds:  []int{1, 2},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/bulk-delete", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.BulkDeleteRows(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_BulkDeleteRows_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/bulk-delete", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BulkDeleteRows(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_BulkDeleteRows_ValidationError(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.BulkDeleteRowsRequest{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/bulk-delete", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BulkDeleteRows(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_RemoveAttachments_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments/remove", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RemoveAttachments(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_RemoveAttachments_ValidationError(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.RemoveAttachmentsRequest{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments/remove", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RemoveAttachments(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// Tests for BulkUpdateColumns
func TestTableHandler_BulkUpdateColumns_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().BulkUpdateColumns(gomock.Any(), "test", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.BulkUpdateColumnsRequest{
		ModelID:  uuid.New().String(),
		ColumnID: uuid.New().String(),
		Updates: []dto.UpdateColumnsRequest{
			{Id: "1", Value: "value1"},
			{Id: "2", Value: "value2"},
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/bulk-update", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.BulkUpdateColumns(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_BulkUpdateColumns_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/bulk-update", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BulkUpdateColumns(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_BulkUpdateColumns_ValidationError(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.BulkUpdateColumnsRequest{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/bulk-update", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BulkUpdateColumns(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_BulkUpdateColumns_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().BulkUpdateColumns(gomock.Any(), "test", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("bulk update failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.BulkUpdateColumnsRequest{
		ModelID:  uuid.New().String(),
		ColumnID: uuid.New().String(),
		Updates: []dto.UpdateColumnsRequest{
			{Id: "1", Value: "value1"},
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/bulk-update", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.BulkUpdateColumns(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// Tests for ResetColumnValues
func TestTableHandler_ResetColumnValues_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().ResetColumnValues(gomock.Any(), "test", gomock.Any(), gomock.Any()).
		Return(nil)
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.ResetColumnValuesRequest{
		ModelID:  uuid.New().String(),
		ColumnId: uuid.New().String(),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/reset", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.ResetColumnValues(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTableHandler_ResetColumnValues_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/reset", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ResetColumnValues(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_ResetColumnValues_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().ResetColumnValues(gomock.Any(), "test", gomock.Any(), gomock.Any()).
		Return(errors.New("reset failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.ResetColumnValuesRequest{
		ModelID:  uuid.New().String(),
		ColumnId: uuid.New().String(),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/columns/reset", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.ResetColumnValues(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// Additional edge case tests for better coverage
func TestTableHandler_CreateTable_EmptyTitleAfterTrim(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.CreateTableRequest{
		BaseID:      uuid.New().String(),
		WorkspaceID: uuid.New().String(),
		Title:       "   ", // Only whitespace
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/table", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateTable(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestTableHandler_UpdateTable_EmptyTitleAfterTrim(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	emptyTitle := "   "
	body, _ := json.Marshal(dto.UpdateTableRequest{Title: &emptyTitle})
	tableID := uuid.New().String()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/"+tableID, bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateTable_InvalidID_BadUUID(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.UpdateTableRequest{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/invalid", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}

	handler.UpdateTable(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTableHandler_UpdateTable_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateTable(gomock.Any(), gomock.Any(), gomock.Any(), "test").
		Return(dto.TableResponse{}, app_errors.TableNotFound)
	handler := handlers.NewTableHandler(mockTableService, nil)

	title := "Updated"
	body, _ := json.Marshal(dto.UpdateTableRequest{Title: &title})
	tableID := uuid.New().String()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/"+tableID, bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateTable(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTableHandler_UpdateTable_IgnoresClientLastModifiedTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewTableHandler(nil, nil)

	body := []byte(`{"title":"Updated","last_modified_time":"string"}`)
	tableID := uuid.New().String()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/"+tableID, bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateTable_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	tableID := uuid.New().String()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/table/"+tableID, bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: tableID}}

	handler.UpdateTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetTableByID_BadUUID(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/table/bad-id", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}

	handler.GetTableByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetColumnById_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetColumnById(gomock.Any(), "test", "c1").Return(dto.ColumnResponse{}, errors.New("not found"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/columns/c1", nil)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	c.Set("schema", "test")

	handler.GetColumnById(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateView_EmptyTitleAfterTrim(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.CreateViewRequest{
		ModelID: uuid.New(),
		BaseID:  uuid.New(),
		Title:   "   ",
		Type:    "grid",
		Meta:    &map[string]interface{}{},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/views", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.CreateView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateView_TitleTooLong(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	longTitle := string(bytes.Repeat([]byte("a"), 300))
	body, _ := json.Marshal(dto.CreateViewRequest{
		ModelID: uuid.New(),
		BaseID:  uuid.New(),
		Title:   longTitle,
		Type:    "grid",
		Meta:    &map[string]interface{}{},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/views", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.CreateView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_CreateView_InvalidMetaGalleryView(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.CreateViewRequest{
		ModelID: uuid.New(),
		BaseID:  uuid.New(),
		Title:   "Gallery View",
		Type:    "gallery",
		Meta:    &map[string]interface{}{},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/views", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.CreateView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetViewByID_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetViewByID(gomock.Any(), "test", "v1").Return(dto.ViewResponse{}, errors.New("not found"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/views/v1", nil)
	c.Params = gin.Params{{Key: "id", Value: "v1"}}
	c.Set("schema", "test")

	handler.GetViewByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetViewsByModelID_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetViewsByModelID(gomock.Any(), "test", "m1").Return(nil, errors.New("fetch failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tables/m1/views", nil)
	c.Params = gin.Params{{Key: "id", Value: "m1"}}
	c.Set("schema", "test")

	handler.GetViewsByModelID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateView_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/views/v1", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "v1"}}

	handler.UpdateView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateView_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateView(gomock.Any(), "test", "v1", gomock.Any()).Return(dto.ViewResponse{}, errors.New("update failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.ViewUpdate{Title: ptrString("Updated")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/views/v1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "v1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteView_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().DeleteView(gomock.Any(), "test", "v1").Return(errors.New("delete failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/views/v1", nil)
	c.Params = gin.Params{{Key: "id", Value: "v1"}}
	c.Set("schema", "test")

	handler.DeleteView(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateColumn_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/columns/c1", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "c1"}}

	handler.UpdateColumn(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateColumn_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().UpdateColumn(gomock.Any(), "test", "c1", gomock.Any()).Return(dto.ColumnResponse{}, errors.New("update failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.ColumnUpdate{Title: ptrString("Updated")})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/columns/c1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateColumn(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteColumn_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().DeleteColumn(gomock.Any(), "test", "c1").Return(errors.New("delete failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/columns/c1", nil)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	c.Set("schema", "test")

	handler.DeleteColumn(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_GetAllRecords_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().GetAllRecords(gomock.Any(), "test", "m1").Return(dto.RecordsResponse{}, errors.New("fetch failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/tables/m1/records", nil)
	c.Params = gin.Params{{Key: "id", Value: "m1"}}
	c.Set("schema", "test")

	handler.GetAllRecords(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateRow_InvalidJSON(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/rows", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_UpdateRow_ValidationError(t *testing.T) {
	handler := handlers.NewTableHandler(nil, nil)

	body, _ := json.Marshal(dto.UpdateRowRequest{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteRow_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().DeleteRow(gomock.Any(), "test", gomock.Any()).Return(errors.New("delete failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.DeleteRowDataRequest{
		ModelID: uuid.New().String(),
		RowId:   1,
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/rows", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.DeleteRow(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_DeleteTable_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().DeleteTable(gomock.Any(), "test", gomock.Any()).Return(errors.New("delete failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	tableID := uuid.New().String()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/tables/"+tableID, nil)
	c.Params = gin.Params{{Key: "id", Value: tableID}}
	c.Set("schema", "test")

	handler.DeleteTable(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_RemoveAttachments_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().RemoveAttachments(gomock.Any(), "test", gomock.Any()).Return(dto.RecordResponse{}, errors.New("remove failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.RemoveAttachmentsRequest{
		ModelID:     uuid.New().String(),
		ColumnId:    uuid.New().String(),
		RowId:       1,
		Attachments: []string{"a1"},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments/remove", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.RemoveAttachments(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_AddAttachment_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().AddAttachment(gomock.Any(), "test", gomock.Any(), gomock.Any()).Return(dto.RecordResponse{}, errors.New("add failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("files", "file.txt")
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("model_id", uuid.New().String())
	_ = writer.WriteField("column_id", uuid.New().String())
	_ = writer.WriteField("row_id", "1")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/attachments", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.AddAttachment(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestTableHandler_BulkDeleteRows_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTableService := mocks.NewMockTableManagementService(ctrl)
	mockTableService.EXPECT().BulkDeleteRows(gomock.Any(), "test", gomock.Any()).Return(0, errors.New("bulk delete failed"))
	handler := handlers.NewTableHandler(mockTableService, nil)

	body, _ := json.Marshal(dto.BulkDeleteRowsRequest{
		ModelID: uuid.New().String(),
		RowIds:  []int{1, 2},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/rows/bulk-delete", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.BulkDeleteRows(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func ptrString(s string) *string {
	return &s
}

func postJSONContext(body interface{}) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	payload, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	return w, c
}

func enhancementColumnIDs() (modelID, colID, colID2 string) {
	return uuid.New().String(), uuid.New().String(), uuid.New().String()
}

func TestTableHandler_TrimWhitespace(t *testing.T) {
	modelID, colID, _ := enhancementColumnIDs()
	validReq := dto.TrimWhitespaceRequest{ModelID: modelID, Columns: []string{colID}, TrimMode: "trim_both"}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.TrimWhitespace(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.TrimWhitespace(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(app_errors.UpdateNotAllowed)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.TrimWhitespace(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().TrimWhitespace(gomock.Any(), "test", validReq).Return(dto.TrimWhitespaceResponse{}, errors.New("trim failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.TrimWhitespace(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	trimModes := []string{"trim_both", "trim_leading", "trim_trailing", "collapse_spaces"}
	for _, mode := range trimModes {
		mode := mode
		t.Run("success "+mode, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req := validReq
			req.TrimMode = mode
			mockSvc := mocks.NewMockTableManagementService(ctrl)
			mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
			mockSvc.EXPECT().TrimWhitespace(gomock.Any(), "test", req).Return(dto.TrimWhitespaceResponse{TotalUpdated: 1}, nil)
			handler := handlers.NewTableHandler(mockSvc, nil)
			w, c := postJSONContext(req)
			handler.TrimWhitespace(c)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestTableHandler_FindReplace(t *testing.T) {
	modelID, colID, _ := enhancementColumnIDs()
	validReq := dto.FindReplaceRequest{
		ModelID: modelID, Columns: []string{colID},
		FindValue: "a", ReplaceValue: "b", MatchType: "ignore_case",
	}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.FindReplace(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.FindReplace(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(app_errors.ColumnNotFound)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.FindReplace(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().FindReplace(gomock.Any(), "test", validReq).Return(dto.FindReplaceResponse{}, errors.New("replace failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.FindReplace(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	for _, matchType := range []string{"match_case", "ignore_case", "match_entire_value"} {
		matchType := matchType
		t.Run("success "+matchType, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req := validReq
			req.MatchType = matchType
			mockSvc := mocks.NewMockTableManagementService(ctrl)
			mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
			mockSvc.EXPECT().FindReplace(gomock.Any(), "test", req).Return(dto.FindReplaceResponse{TotalMatched: 1}, nil)
			handler := handlers.NewTableHandler(mockSvc, nil)
			w, c := postJSONContext(req)
			handler.FindReplace(c)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestTableHandler_CaseNormalization(t *testing.T) {
	modelID, colID, _ := enhancementColumnIDs()
	validReq := dto.CaseNormalizationRequest{
		ModelID: modelID, Columns: []string{colID}, CaseFormat: "lowercase",
	}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.CaseNormalization(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.CaseNormalization(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(app_errors.UpdateNotAllowed)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.CaseNormalization(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().CaseNormalization(gomock.Any(), "test", validReq).Return(dto.CaseNormalizationResponse{}, errors.New("normalize failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.CaseNormalization(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	for _, caseFormat := range []string{"lowercase", "uppercase", "title_case", "sentence_case"} {
		caseFormat := caseFormat
		t.Run("success "+caseFormat, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req := validReq
			req.CaseFormat = caseFormat
			mockSvc := mocks.NewMockTableManagementService(ctrl)
			mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
			mockSvc.EXPECT().CaseNormalization(gomock.Any(), "test", req).Return(dto.CaseNormalizationResponse{TotalUpdated: 1}, nil)
			handler := handlers.NewTableHandler(mockSvc, nil)
			w, c := postJSONContext(req)
			handler.CaseNormalization(c)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestTableHandler_RemoveSpecialCharacters(t *testing.T) {
	modelID, colID, _ := enhancementColumnIDs()
	validReq := dto.RemoveSpecialCharactersRequest{
		ModelID: modelID, Columns: []string{colID}, SpecialCharactersType: "symbols",
	}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.RemoveSpecialCharacters(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.RemoveSpecialCharacters(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(app_errors.ColumnNotFound)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.RemoveSpecialCharacters(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().RemoveSpecialCharacters(gomock.Any(), "test", validReq).Return(dto.RemoveSpecialCharactersResponse{}, errors.New("remove failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.RemoveSpecialCharacters(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	for _, charType := range []string{"symbols", "currency_symbols", "brackets", "punctuation", "custom"} {
		charType := charType
		t.Run("success "+charType, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req := validReq
			req.SpecialCharactersType = charType
			if charType == "custom" {
				req.CustomCharacter = []string{"#"}
			}
			mockSvc := mocks.NewMockTableManagementService(ctrl)
			mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
			mockSvc.EXPECT().RemoveSpecialCharacters(gomock.Any(), "test", req).Return(dto.RemoveSpecialCharactersResponse{TotalMatched: 1}, nil)
			handler := handlers.NewTableHandler(mockSvc, nil)
			w, c := postJSONContext(req)
			handler.RemoveSpecialCharacters(c)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestTableHandler_ColumnSplit(t *testing.T) {
	modelUUID := uuid.New()
	colUUID := uuid.New()
	validReq := dto.ColumnSplitRequest{
		ModelID: modelUUID, ColumnID: colUUID,
		SplitBy: dto.SplitByRequest{Type: "separator", Config: map[string]interface{}{"separator": ","}},
		Where:   "end",
	}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.ColumnSplit(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.ColumnSplit(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate column error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnAllowedForSplit(gomock.Any(), "test", modelUUID.String(), colUUID.String()).Return(app_errors.SplitNotPossible)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.ColumnSplit(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnAllowedForSplit(gomock.Any(), "test", modelUUID.String(), colUUID.String()).Return(nil)
		mockSvc.EXPECT().ColumnSplit(gomock.Any(), "test", validReq).Return(dto.ColumnSplitResponse{}, errors.New("split failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.ColumnSplit(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnAllowedForSplit(gomock.Any(), "test", modelUUID.String(), colUUID.String()).Return(nil)
		mockSvc.EXPECT().ColumnSplit(gomock.Any(), "test", validReq).Return(dto.ColumnSplitResponse{Message: "ok", CreatedColumns: []string{"c1"}}, nil)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.ColumnSplit(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTableHandler_RemoveFormatting(t *testing.T) {
	modelID, colID, _ := enhancementColumnIDs()
	validReq := dto.RemoveFormattingRequest{
		ModelID: modelID, Columns: []string{colID}, Formatting: "currency",
	}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.RemoveFormatting(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.RemoveFormatting(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(app_errors.UpdateNotAllowed)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.RemoveFormatting(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().RemoveFormatting(gomock.Any(), "test", validReq).Return(dto.RemoveFormattingResponse{}, errors.New("format failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.RemoveFormatting(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	for _, formatting := range []string{"currency", "percentage", "separator", "phone", "date", "custom"} {
		formatting := formatting
		t.Run("success "+formatting, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req := validReq
			req.Formatting = formatting
			if formatting == "custom" {
				req.CustomPattern = []string{"-"}
			}
			mockSvc := mocks.NewMockTableManagementService(ctrl)
			mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
			mockSvc.EXPECT().RemoveFormatting(gomock.Any(), "test", req).Return(dto.RemoveFormattingResponse{UpdatedRecords: 1}, nil)
			handler := handlers.NewTableHandler(mockSvc, nil)
			w, c := postJSONContext(req)
			handler.RemoveFormatting(c)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestTableHandler_RemoveDuplicates(t *testing.T) {
	modelID, colID, _ := enhancementColumnIDs()
	validReq := dto.RemoveDuplicatesRequest{
		ModelID: modelID, Columns: []string{colID},
		Duplicate: "remove_row", KeepRule: "keep_first",
	}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.RemoveDuplicates(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.RemoveDuplicates(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(app_errors.ColumnNotFound)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.RemoveDuplicates(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().RemoveDuplicates(gomock.Any(), "test", validReq).Return(dto.RemoveDuplicatesResponse{}, errors.New("dedupe failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.RemoveDuplicates(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().RemoveDuplicates(gomock.Any(), "test", validReq).Return(dto.RemoveDuplicatesResponse{TotalRowsAffected: 2}, nil)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.RemoveDuplicates(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTableHandler_MergeColumns(t *testing.T) {
	modelID, colID, colID2 := enhancementColumnIDs()
	validReq := dto.MergeColumnsRequest{
		ModelID: modelID, Columns: []string{colID, colID2}, MergeFormat: "space",
	}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.MergeColumns(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.MergeColumns(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID, colID2}).Return(app_errors.UpdateNotAllowed)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.MergeColumns(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID, colID2}).Return(nil)
		mockSvc.EXPECT().MergeColumns(gomock.Any(), "test", validReq).Return(dto.MergeColumnsResponse{}, errors.New("merge failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.MergeColumns(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID, colID2}).Return(nil)
		mockSvc.EXPECT().MergeColumns(gomock.Any(), "test", validReq).Return(dto.MergeColumnsResponse{TotalUpdated: 1}, nil)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.MergeColumns(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTableHandler_ExtractSubstring(t *testing.T) {
	modelID, colID, _ := enhancementColumnIDs()
	validReq := dto.ExtractSubstringRequest{
		ModelID: modelID, ColumnId: colID,
		ExtractionMethod: "extraction_type", ExtractionType: "email",
	}

	t.Run("invalid JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.ExtractSubstring(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.ExtractSubstring(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(app_errors.ColumnNotFound)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.ExtractSubstring(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().ExtractSubstring(gomock.Any(), "test", validReq).Return(dto.ExtractSubstringResponse{}, errors.New("extract failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.ExtractSubstring(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().ExtractSubstring(gomock.Any(), "test", validReq).Return(dto.ExtractSubstringResponse{UpdatedRecords: 1}, nil)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.ExtractSubstring(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
