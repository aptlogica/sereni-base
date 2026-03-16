package handlers_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewUserHandler(t *testing.T) {
	handler := handlers.NewUserHandler(nil)
	assert.NotNil(t, handler)
}

func TestUserHandler_GetUserProfileByID_EmptyID(t *testing.T) {
	handler := handlers.NewUserHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/user/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}
	c.Set("schema", "test_schema")

	handler.GetUserProfileByID(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestUserHandler_GetUserProfileByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().GetUserProfileByID(gomock.Any(), "test", "u1").Return(dto.UserResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/users/u1", nil)
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.GetUserProfileByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetUserProfileByID_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().GetUserProfileByID(gomock.Any(), "test", "u1").Return(dto.UserResponse{}, errors.New("not found"))
	handler := handlers.NewUserHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/users/u1", nil)
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.GetUserProfileByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestUserHandler_UpdateUserProfile_NoAvatar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().UpdateUserProfile(gomock.Any(), "test", "u1", gomock.Any()).Return(dto.UserResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("first_name", "Test")
	_ = writer.WriteField("last_name", "User")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/users/u1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.UpdateUserProfile(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_UpdateUserProfile_AllFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().UpdateUserProfile(gomock.Any(), "test", "u1", gomock.Any()).Return(dto.UserResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("first_name", "Test")
	_ = writer.WriteField("last_name", "User")
	_ = writer.WriteField("display_name", "TU")
	_ = writer.WriteField("dob", "2000-01-01")
	_ = writer.WriteField("country", "US")
	_ = writer.WriteField("timezone", "UTC")
	_ = writer.WriteField("locale", "en-US")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/users/u1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.UpdateUserProfile(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_UpdateUserProfile_WithAvatar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().AddAvatar(gomock.Any(), "test", "u1", gomock.Any()).Return(dto.UserResponse{}, nil)
	mockService.EXPECT().UpdateUserProfile(gomock.Any(), "test", "u1", gomock.Any()).Return(dto.UserResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("first_name", "Test")
	part, _ := writer.CreateFormFile("avatar", "avatar.png")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/users/u1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.UpdateUserProfile(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_AddAvatar_MissingFile(t *testing.T) {
	handler := handlers.NewUserHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/u1/avatar", nil)
	c.Params = gin.Params{{Key: "id", Value: "u1"}}

	handler.AddAvatar(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestUserHandler_AddAvatar_MissingID(t *testing.T) {
	handler := handlers.NewUserHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users//avatar", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.AddAvatar(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestUserHandler_AddAvatar_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().AddAvatar(gomock.Any(), "test", "u1", gomock.Any()).Return(dto.UserResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "avatar.png")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/u1/avatar", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.AddAvatar(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_RemoveAvatar_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().RemoveAvatar(gomock.Any(), "test", "u1").Return(dto.UserResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/users/u1/avatar", nil)
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.RemoveAvatar(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_RemoveAvatar_MissingID(t *testing.T) {
	handler := handlers.NewUserHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/users//avatar", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.RemoveAvatar(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetWorkspaces_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().GetWorkspaces(gomock.Any(), "test", "user123", "Admin").Return([]dto.UserWorkspaceResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/users/workspaces", nil)
	c.Set("schema", "test")
	c.Set("user_id", "user123")
	c.Set("roles", "Admin")

	handler.GetWorkspaces(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetUserAccessDetails_Unauthorized(t *testing.T) {
	handler := handlers.NewUserHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/users/access?user_id=u1", nil)
	c.Set("schema", "test")
	c.Set("roles", "Member")

	handler.GetUserAccessDetails(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetUserAccessDetails_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	mockService.EXPECT().GetUserAccessDetails(gomock.Any(), "test", "u1", "Admin", "w1").Return(dto.UserAccessDetailsResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/users/access?user_id=u1&workspace_id=w1", nil)
	c.Set("schema", "test")
	c.Set("roles", "Admin")

	handler.GetUserAccessDetails(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetUserRolesAndAccess_MissingID(t *testing.T) {
	handler := handlers.NewUserHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/users//roles", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.GetUserRolesAndAccess(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetUserRolesAndAccess_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockUserManagementService(ctrl)
	scope := "w1"
	mockService.EXPECT().GetUserRolesAndAccess(gomock.Any(), "test", "u1", &scope).Return([]dto.UserRolesAccessResponse{}, nil)
	handler := handlers.NewUserHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/users/u1/roles?scope_id=w1", nil)
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.GetUserRolesAndAccess(c)
	assert.Equal(t, http.StatusOK, w.Code)
}
