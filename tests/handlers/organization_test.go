package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewOrganizationHandler(t *testing.T) {
	handler := handlers.NewOrganizationHandler(nil)
	assert.NotNil(t, handler)
}

func TestOrganizationHandler_CreateOrganization_InvalidJSON(t *testing.T) {
	handler := handlers.NewOrganizationHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/organization", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test_schema")
	c.Set("user_id", "user123")

	handler.CreateOrganization(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestOrganizationHandler_UpdateOrganization_InvalidJSON(t *testing.T) {
	handler := handlers.NewOrganizationHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/organization/123", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "123"}}
	c.Set("schema", "test_schema")

	handler.UpdateOrganization(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestOrganizationHandler_CreateOrganization_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().CreateOrganization(gomock.Any(), "test", gomock.Any()).Return(tenant.Organization{}, nil)
	handler := handlers.NewOrganizationHandler(mockService)

	body, _ := json.Marshal(dto.CreateOrganizationRequest{Name: "Org", Email: "org@example.com"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/organization", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateOrganization(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_GetOrganizationByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().GetOrganizationByID(gomock.Any(), "test", "org1").Return(tenant.Organization{}, nil)
	handler := handlers.NewOrganizationHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/organization/org1", nil)
	c.Params = gin.Params{{Key: "id", Value: "org1"}}
	c.Set("schema", "test")

	handler.GetOrganizationByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_GetAllOrganizations_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().GetOrganization(gomock.Any(), "test").Return(tenant.Organization{}, nil)
	handler := handlers.NewOrganizationHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/organization", nil)
	c.Set("schema", "test")

	handler.GetAllOrganizations(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_UpdateOrganization_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().UpdateOrganization(gomock.Any(), "test", "org1", gomock.Any()).Return(tenant.Organization{}, nil)
	handler := handlers.NewOrganizationHandler(mockService)

	name := "Updated"
	body, _ := json.Marshal(dto.UpdateOrganizationRequest{Name: &name})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/organization/org1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "org1"}}
	c.Set("schema", "test")

	handler.UpdateOrganization(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_UpdateOrganization_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().UpdateOrganization(gomock.Any(), "test", "org1", gomock.Any()).Return(tenant.Organization{}, errors.New("update failed"))
	handler := handlers.NewOrganizationHandler(mockService)

	name := "Updated"
	body, _ := json.Marshal(dto.UpdateOrganizationRequest{Name: &name})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/organization/org1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "org1"}}
	c.Set("schema", "test")

	handler.UpdateOrganization(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_DeleteOrganization_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().DeleteOrganization(gomock.Any(), "test", "org1").Return(nil)
	handler := handlers.NewOrganizationHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/organization/org1", nil)
	c.Params = gin.Params{{Key: "id", Value: "org1"}}
	c.Set("schema", "test")

	handler.DeleteOrganization(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_GetOrganizationByEmail_MissingEmail(t *testing.T) {
	handler := handlers.NewOrganizationHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/organization/email", nil)

	handler.GetOrganizationByEmail(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_GetOrganizationByEmail_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().GetOrganizationByEmail(gomock.Any(), "test", "test@example.com").Return(tenant.Organization{}, nil)
	handler := handlers.NewOrganizationHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/organization/email?email=test@example.com", nil)
	c.Set("schema", "test")

	handler.GetOrganizationByEmail(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Additional tests for error cases and improved coverage
func TestOrganizationHandler_CreateOrganization_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().CreateOrganization(gomock.Any(), "test", gomock.Any()).Return(tenant.Organization{}, errors.New("creation failed"))
	handler := handlers.NewOrganizationHandler(mockService)

	body, _ := json.Marshal(dto.CreateOrganizationRequest{Name: "Org", Email: "org@example.com"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/organization", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateOrganization(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_GetOrganizationByID_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().GetOrganizationByID(gomock.Any(), "test", "org1").Return(tenant.Organization{}, errors.New("not found"))
	handler := handlers.NewOrganizationHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/organization/org1", nil)
	c.Params = gin.Params{{Key: "id", Value: "org1"}}
	c.Set("schema", "test")

	handler.GetOrganizationByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_GetAllOrganizations_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().GetOrganization(gomock.Any(), "test").Return(tenant.Organization{}, errors.New("service error"))
	handler := handlers.NewOrganizationHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/organization", nil)
	c.Set("schema", "test")

	handler.GetAllOrganizations(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_DeleteOrganization_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().DeleteOrganization(gomock.Any(), "test", "org1").Return(errors.New("delete failed"))
	handler := handlers.NewOrganizationHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/organization/org1", nil)
	c.Params = gin.Params{{Key: "id", Value: "org1"}}
	c.Set("schema", "test")

	handler.DeleteOrganization(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestOrganizationHandler_GetOrganizationByEmail_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockOrganizationService(ctrl)
	mockService.EXPECT().GetOrganizationByEmail(gomock.Any(), "test", "test@example.com").Return(tenant.Organization{}, errors.New("service error"))
	handler := handlers.NewOrganizationHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/organization/email?email=test@example.com", nil)
	c.Set("schema", "test")

	handler.GetOrganizationByEmail(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
