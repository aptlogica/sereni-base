package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"serenibase/internal/dto"
	"serenibase/internal/handlers"
	"serenibase/internal/models/tenant"
	"serenibase/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewWorkspaceHandler(t *testing.T) {
	handler := handlers.NewWorkspaceHandler(nil, nil)
	assert.NotNil(t, handler)
}

func TestWorkspaceHandler_CreateWorkspace_InvalidJSON(t *testing.T) {
	handler := handlers.NewWorkspaceHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/workspace", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test_schema")
	c.Set("user_id", "user123")

	handler.CreateWorkspace(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestWorkspaceHandler_UpdateWorkspace_InvalidJSON(t *testing.T) {
	handler := handlers.NewWorkspaceHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/workspace/123", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "123"}}
	c.Set("schema", "test_schema")
	c.Set("user_id", "user123")

	handler.UpdateWorkspace(c)

	assert.NotEqual(t, 200, w.Code)
}

func TestWorkspaceHandler_CreateWorkspace_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().Create(gomock.Any(), gomock.Any(), "test", "user123").Return(dto.WorkspaceResponse{}, nil)
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	body, _ := json.Marshal(dto.CreateWorkspaceRequest{Title: "Workspace"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/workspaces", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.CreateWorkspace(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_GetWorkspaceByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().GetByID(gomock.Any(), "test", "w1").Return(tenant.Workspace{}, nil)
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/w1", nil)
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")

	handler.GetWorkspaceByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_GetAllWorkspaces_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().GetAll(gomock.Any(), "test").Return([]tenant.Workspace{}, nil)
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces", nil)
	c.Set("schema", "test")

	handler.GetAllWorkspaces(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_GetAllWorkspaces_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().GetAll(gomock.Any(), "test").Return(nil, errors.New("fetch failed"))
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces", nil)
	c.Set("schema", "test")

	handler.GetAllWorkspaces(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_UpdateWorkspace_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().Update(gomock.Any(), "test", "w1", gomock.Any(), "user123").Return(tenant.Workspace{}, nil)
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	title := "Updated"
	body, _ := json.Marshal(dto.WorkspaceUpdate{Title: &title})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/workspaces/w1", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.UpdateWorkspace(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_DeleteWorkspace_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().Delete(gomock.Any(), "test", "w1").Return(nil)
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/workspaces/w1", nil)
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")

	handler.DeleteWorkspace(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_DeleteWorkspace_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().Delete(gomock.Any(), "test", "w1").Return(errors.New("delete failed"))
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/workspaces/w1", nil)
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")

	handler.DeleteWorkspace(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_GetTablesByWorkspaceId_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().GetTablesByWorkspaceId(gomock.Any(), "test", "w1").Return([]dto.TableResponse{}, nil)
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/w1/tables", nil)
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")

	handler.GetTablesByWorkspaceId(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_GetBasesByWorkspaceId_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorkspace := mocks.NewMockWorkspaceManagementService(ctrl)
	mockWorkspace.EXPECT().GetAllBasesByWorkspaceId(gomock.Any(), "test", "w1", "Admin", "user123").Return([]dto.BaseResponse{}, nil)
	handler := handlers.NewWorkspaceHandler(mockWorkspace, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/w1/bases", nil)
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")
	c.Set("roles", "Admin")
	c.Set("user_id", "user123")

	handler.GetBasesByWorkspaceId(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_BulkAddMembers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuth := mocks.NewMockAuthManagementService(ctrl)
	mockAuth.EXPECT().BulkAddMembers(gomock.Any(), "test", gomock.Any(), "user123").Return(dto.BulkAddMembersResponse{}, nil)
	handler := handlers.NewWorkspaceHandler(nil, mockAuth)

	body, _ := json.Marshal(dto.BulkAddMembersRequest{
		Members: []dto.BulkMemberRequest{
			{UserID: "u1", Memberships: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "Admin"}}},
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/workspaces/bulk-add", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.BulkAddMembers(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_BulkAddMembers_InvalidJSON(t *testing.T) {
	handler := handlers.NewWorkspaceHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/workspaces/bulk-add", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BulkAddMembers(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_BulkAddBaseMembers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuth := mocks.NewMockAuthManagementService(ctrl)
	mockAuth.EXPECT().BulkAddBaseMembers(gomock.Any(), "test", "b1", gomock.Any(), "user123").Return(dto.BulkAddMembersResponse{}, nil)
	handler := handlers.NewWorkspaceHandler(nil, mockAuth)

	body, _ := json.Marshal(dto.BulkAddBaseMembersRequest{
		Members: []dto.BulkMemberRequest{
			{UserID: "u1", Memberships: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "Admin"}}},
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bases/b1/bulk-add", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.BulkAddBaseMembers(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_BulkAddBaseMembers_InvalidJSON(t *testing.T) {
	handler := handlers.NewWorkspaceHandler(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bases/b1/bulk-add", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "b1"}}

	handler.BulkAddBaseMembers(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
