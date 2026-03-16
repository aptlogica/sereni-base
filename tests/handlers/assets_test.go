package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	// Initialize minimal config for tests
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{
			Asset: config.AssetConfig{
				MaxSize: 5242880, // 5MB
			},
		}
	}
}

func TestNewAssetsHandler(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)
	assert.NotNil(t, handler)
}

func TestAssetsHandler_Upload_NoMultipartForm(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/upload", nil)
	c.Set("schema", "test_schema")

	handler.Upload(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestAssetsHandler_UploadImage_NoMultipartForm(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/upload-image", nil)
	c.Set("schema", "test_schema")

	handler.UploadImage(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestAssetsHandler_GetBulkAssets_InvalidJSON(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bulk", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test_schema")

	handler.GetBulkAssets(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestAssetsHandler_GetBulkAssets_EmptyIDs(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)

	req := map[string][]string{"ids": {}}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bulk", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test_schema")

	handler.GetBulkAssets(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestAssetsHandler_UpdateAssetByID_EmptyID(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/assets/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	c.Set("schema", "test_schema")

	handler.UpdateAssetByID(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestAssetsHandler_UpdateAssetByID_InvalidUUID(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/assets/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	c.Set("schema", "test_schema")

	handler.UpdateAssetByID(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestAssetsHandler_DeleteAssetByID_EmptyID(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/assets/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	c.Set("schema", "test_schema")

	handler.DeleteAssetByID(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestAssetsHandler_DeleteAssetByID_InvalidUUID(t *testing.T) {
	handler := handlers.NewAssetsHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/assets/invalid-uuid", nil)
	c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
	c.Set("schema", "test_schema")

	handler.DeleteAssetByID(c)

	assert.NotEqual(t, 200, w.Code)
}

// Success path tests for better coverage
func TestAssetsHandler_UploadImage_SuccessPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAssetManagementService(ctrl)
	mockService.EXPECT().UploadImage(gomock.Any(), gomock.Any(), gomock.Any()).Return([]tenant.Assets{}, nil)
	handler := handlers.NewAssetsHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// Create file with proper image content type
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="files"; filename="test.png"`)
	h.Set("Content-Type", "image/png")
	part, _ := writer.CreatePart(h)
	part.Write([]byte("fake image data"))
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/upload-image", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.UploadImage(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAssetsHandler_Upload_NoFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAssetsHandler(nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.Upload(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAssetsHandler_Upload_FileTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	origMax := config.AppConfig.Asset.MaxSize
	config.AppConfig.Asset.MaxSize = 1
	defer func() { config.AppConfig.Asset.MaxSize = origMax }()

	handler := handlers.NewAssetsHandler(mocks.NewMockAssetManagementService(ctrl))

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("files", "big.txt")
	_, _ = io.WriteString(part, "too big")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.Upload(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAssetsHandler_Upload_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAssetManagementService(ctrl)
	mockService.EXPECT().Upload(gomock.Any(), gomock.Any(), "test").Return([]tenant.Assets{}, nil)
	handler := handlers.NewAssetsHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("files", "ok.txt")
	_, _ = io.WriteString(part, "ok")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.Upload(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAssetsHandler_UploadImage_TooManyFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAssetsHandler(nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_, _ = writer.CreateFormFile("files", "a.png")
	_, _ = writer.CreateFormFile("files", "b.png")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/upload-image", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.UploadImage(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAssetsHandler_UploadImage_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAssetsHandler(nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="files"; filename="file.txt"`)
	h.Set("Content-Type", "text/plain")
	part, _ := writer.CreatePart(h)
	_, _ = part.Write([]byte("not image"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/upload-image", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")

	handler.UploadImage(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAssetsHandler_GetBulkAssets_SuccessPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAssetManagementService(ctrl)
	mockService.EXPECT().GetBulkAssets(gomock.Any(), gomock.Any(), gomock.Any()).Return([]tenant.Assets{}, nil)
	handler := handlers.NewAssetsHandler(mockService)

	body, _ := json.Marshal(dto.BulkAssetRequest{IDs: []string{uuid.New().String()}})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bulk", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.GetBulkAssets(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAssetsHandler_UpdateAssetByID_SuccessPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAssetManagementService(ctrl)
	mockService.EXPECT().UpdateAsset(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tenant.Assets{}, nil)
	handler := handlers.NewAssetsHandler(mockService)

	title := "test"
	body, _ := json.Marshal(dto.AssetUpdate{Title: &title})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assetID := uuid.New().String()
	c.Request = httptest.NewRequest("PUT", "/assets/"+assetID, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: assetID}}
	c.Set("schema", "test")

	handler.UpdateAssetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAssetsHandler_DeleteAssetByID_SuccessPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAssetManagementService(ctrl)
	mockService.EXPECT().DeleteAsset(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	handler := handlers.NewAssetsHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assetID := uuid.New().String()
	c.Request = httptest.NewRequest("DELETE", "/assets/"+assetID, nil)
	c.Params = gin.Params{{Key: "id", Value: assetID}}
	c.Set("schema", "test")

	handler.DeleteAssetByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
}
