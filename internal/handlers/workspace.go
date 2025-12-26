package handlers

import (
	"serenibase/internal/dto"
	"serenibase/internal/handlers/validators"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type WorkspaceHandler struct {
	workspaceManagementService interfaces.WorkspaceManagementService
}

func NewWorkspaceHandler(workspaceManagementService interfaces.WorkspaceManagementService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaceManagementService: workspaceManagementService}
}

func (h *WorkspaceHandler) CreateWorkspace(c *gin.Context) {
	var req dto.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.WorkspaceCreationValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	workspace, err := h.workspaceManagementService.Create(c.Request.Context(), req, schemaName, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, "Workspace created successfully", workspace)
}

func (h *WorkspaceHandler) GetWorkspaceByID(c *gin.Context) {
	id := c.Param("id") // directly fetch from URI like /workspaces/:id

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	workspace, err := h.workspaceManagementService.GetByID(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Workspace retrieved successfully", workspace)
}

func (h *WorkspaceHandler) GetAllWorkspaces(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	workspaces, err := h.workspaceManagementService.GetAll(c.Request.Context(), schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Workspaces retrieved successfully", workspaces)
}

func (h *WorkspaceHandler) UpdateWorkspace(c *gin.Context) {
	var req dto.WorkspaceUpdate

	id := c.Param("id") // directly fetch from URI like /workspaces/:id

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.WorkspaceUpdateValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	workspace, err := h.workspaceManagementService.Update(c.Request.Context(), schemaName, id, req, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Workspace updated successfully", workspace)
}

func (h *WorkspaceHandler) DeleteWorkspace(c *gin.Context) {
	id := c.Param("id") // directly fetch from URI like /workspaces/:id

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	if err := h.workspaceManagementService.Delete(c.Request.Context(), schemaName, id); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Workspace deleted successfully", nil)
}

func (h *WorkspaceHandler) GetTablesByWorkspaceId(c *gin.Context) {
	workspaceID := c.Param("id") // expects route like /workspaces/:id/tables

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	tables, err := h.workspaceManagementService.GetTablesByWorkspaceId(c.Request.Context(), schemaName, workspaceID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Tables retrieved successfully", tables)
}

func (h *WorkspaceHandler) GetBasesByWorkspaceId(c *gin.Context) {
	workspaceID := c.Param("id") // expects route like /workspaces/:id/bases

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.(string)

	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(string)

	var bases interface{}
	var err error

	bases, err = h.workspaceManagementService.GetAllBasesByWorkspaceId(c.Request.Context(), schemaName, workspaceID, roles, userID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Bases retrieved successfully", bases)
}
