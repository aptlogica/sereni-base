package handlers

import (
	"serenibase/internal/dto"
	"serenibase/internal/handlers/validators"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type BaseHandler struct {
	baseManagementService interfaces.BaseManagementService
}

func NewBaseHandler(baseManagementService interfaces.BaseManagementService) *BaseHandler {
	return &BaseHandler{baseManagementService: baseManagementService}
}

func (h *BaseHandler) CreateBase(c *gin.Context) {
	// Get form values
	title := c.PostForm("title")
	description := c.PostForm("description")
	workspaceID := c.PostForm("workspace_id")

	if title == "" || workspaceID == "" {
		response.SendError(c, "title and workspace_id are required")
		return
	}

	// Get optional image file
	file, _ := c.FormFile("image")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	req := dto.CreateBaseRequest{
		Title:       title,
		Description: &description,
		WorkspaceID: workspaceID,
		CreatedBy:   userId,
	}

	base, err := h.baseManagementService.CreateBaseWithImage(c.Request.Context(), req, schemaName, userId, file)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base created successfully", base)
}

func (h *BaseHandler) GetBaseByID(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	base, err := h.baseManagementService.GetBaseByID(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base retrieved successfully", base)
}

func (h *BaseHandler) UpdateBase(c *gin.Context) {
	var req dto.BaseUpdate

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.BaseUpdateValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	updatedBase, err := h.baseManagementService.UpdateBase(c.Request.Context(), schemaName, id, req, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base updated successfully", updatedBase)
}

func (h *BaseHandler) DeleteBase(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	if err := h.baseManagementService.DeleteBase(c.Request.Context(), schemaName, id); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base deleted successfully", nil)
}

func (h *BaseHandler) GetTablesByBaseId(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	tables, err := h.baseManagementService.GetTablesByBaseId(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "tables retrieved successfully", tables)
}

func (h *BaseHandler) AddBaseImage(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, "invalid base id")
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		response.SendError(c, "invalid image file")
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	updatedBase, err := h.baseManagementService.AddBaseImage(c.Request.Context(), schemaName, id, file, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base image added successfully", updatedBase)
}

func (h *BaseHandler) RemoveBaseImage(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, "invalid base id")
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	updatedBase, err := h.baseManagementService.RemoveBaseImage(c.Request.Context(), schemaName, id, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base image removed successfully", updatedBase)
}
