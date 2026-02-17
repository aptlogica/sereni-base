package handlers_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"serenibase/internal/dto"
	"serenibase/internal/handlers"
	"serenibase/internal/models/tenant"
	"serenibase/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewBaseHandler(t *testing.T) {
	handler := handlers.NewBaseHandler(nil)
	assert.NotNil(t, handler)
}

func TestBaseHandler_CreateBase_MissingTitle(t *testing.T) {
	handler := handlers.NewBaseHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/base", nil)
	c.Request.Header.Set("Content-Type", "multipart/form-data")
	c.Set("schema", "test_schema")
	c.Set("user_id", "user123")

	handler.CreateBase(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestBaseHandler_CreateBase_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().CreateBaseWithImage(gomock.Any(), gomock.Any(), "test", "user123", gomock.Any()).Return(tenant.Base{}, nil)
	handler := handlers.NewBaseHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("title", "Base")
	_ = writer.WriteField("workspace_id", "w1")
	_ = writer.WriteField("description", "desc")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bases", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateBase(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaseHandler_GetBaseByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().GetBaseByID(gomock.Any(), "test", "b1").Return(tenant.Base{}, nil)
	handler := handlers.NewBaseHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/bases/b1", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")

	handler.GetBaseByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaseHandler_GetBaseByID_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().GetBaseByID(gomock.Any(), "test", "b1").Return(tenant.Base{}, errors.New("not found"))
	handler := handlers.NewBaseHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/bases/b1", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")

	handler.GetBaseByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_UpdateBase_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().UpdateBase(gomock.Any(), "test", "b1", gomock.Any(), "user123", gomock.Any(), gomock.Any()).Return(tenant.Base{}, nil)
	handler := handlers.NewBaseHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader("title=New+Title&description=desc")
	c.Request = httptest.NewRequest("PUT", "/bases/b1", body)
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateBase(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaseHandler_UpdateBase_WithImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().UpdateBase(gomock.Any(), "test", "b1", gomock.Any(), "user123", gomock.Any(), gomock.Any()).Return(tenant.Base{}, nil)
	handler := handlers.NewBaseHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("title", "New Title")
	part, _ := writer.CreateFormFile("image", "img.png")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/bases/b1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateBase(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaseHandler_UpdateBase_RemoveImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().UpdateBase(gomock.Any(), "test", "b1", gomock.Any(), "user123", gomock.Any(), gomock.Any()).Return(tenant.Base{}, nil)
	handler := handlers.NewBaseHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("title", "New Title")
	_ = writer.WriteField("remove_image", "true")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/bases/b1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateBase(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaseHandler_UpdateBase_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().UpdateBase(gomock.Any(), "test", "b1", gomock.Any(), "user123", gomock.Any(), gomock.Any()).Return(tenant.Base{}, errors.New("update failed"))
	handler := handlers.NewBaseHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("title", "New Title")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/bases/b1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateBase(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_UpdateBase_AddImageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().UpdateBase(gomock.Any(), "test", "b1", gomock.Any(), "user123", gomock.Any(), gomock.Any()).Return(tenant.Base{}, errors.New("add image failed"))
	handler := handlers.NewBaseHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("title", "New Title")
	part, _ := writer.CreateFormFile("image", "img.png")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/bases/b1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateBase(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_UpdateBase_RemoveImageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().UpdateBase(gomock.Any(), "test", "b1", gomock.Any(), "user123", gomock.Any(), gomock.Any()).Return(tenant.Base{}, errors.New("remove image failed"))
	handler := handlers.NewBaseHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("title", "New Title")
	_ = writer.WriteField("remove_image", "true")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/bases/b1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateBase(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_DeleteBase_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().DeleteBase(gomock.Any(), "test", "b1").Return(nil)
	handler := handlers.NewBaseHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/bases/b1", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")

	handler.DeleteBase(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaseHandler_DeleteBase_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().DeleteBase(gomock.Any(), "test", "b1").Return(errors.New("delete failed"))
	handler := handlers.NewBaseHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/bases/b1", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")

	handler.DeleteBase(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_GetTablesByBaseId_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().GetTablesByBaseId(gomock.Any(), "test", "b1").Return([]dto.TableResponse{}, nil)
	handler := handlers.NewBaseHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/bases/b1/tables", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")

	handler.GetTablesByBaseId(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaseHandler_GetTablesByBaseId_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().GetTablesByBaseId(gomock.Any(), "test", "b1").Return(nil, errors.New("fetch failed"))
	handler := handlers.NewBaseHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/bases/b1/tables", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")

	handler.GetTablesByBaseId(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_AddBaseImage_MissingID(t *testing.T) {
	handler := handlers.NewBaseHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bases//image", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.AddBaseImage(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_AddBaseImage_MissingFile(t *testing.T) {
	handler := handlers.NewBaseHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bases/b1/image", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}

	handler.AddBaseImage(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_AddBaseImage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().AddBaseImage(gomock.Any(), "test", "b1", gomock.Any(), "user123").Return(tenant.Base{}, nil)
	handler := handlers.NewBaseHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "img.png")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bases/b1/image", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.AddBaseImage(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBaseHandler_RemoveBaseImage_MissingID(t *testing.T) {
	handler := handlers.NewBaseHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/bases//image", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.RemoveBaseImage(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBaseHandler_RemoveBaseImage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockBaseManagementService(ctrl)
	mockService.EXPECT().RemoveBaseImage(gomock.Any(), "test", "b1", "user123").Return(tenant.Base{}, nil)
	handler := handlers.NewBaseHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/bases/b1/image", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.RemoveBaseImage(c)
	assert.Equal(t, http.StatusOK, w.Code)
}
