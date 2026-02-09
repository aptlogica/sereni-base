package handlers

import (
	"mime/multipart"
	"serenibase/internal/dto"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"

	"github.com/gin-gonic/gin"
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

func (h *BaseHandler) parseUpdateBaseForm(c *gin.Context) (dto.BaseUpdate, *multipart.FileHeader, string) {
	req := dto.BaseUpdate{}

	// Get optional title from form
	if title := c.PostForm("title"); title != "" {
		req.Title = &title
	}

	// Get optional description from form
	if description := c.PostForm("description"); description != "" {
		req.Description = &description
	}

	// Get optional status from form
	if status := c.PostForm("status"); status != "" {
		req.Status = &status
	}

	// Get optional visibility from form
	if visibility := c.PostForm("visibility"); visibility != "" {
		req.Visibility = &visibility
	}

	// Get optional type from form
	if baseType := c.PostForm("type"); baseType != "" {
		req.Type = &baseType
	}

	// Handle image file upload if provided
	var fileHeader *multipart.FileHeader
	if fh, err := c.FormFile("image"); err == nil && fh != nil {
		fileHeader = fh
	}

	// Check if remove image is requested
	removeImage := c.PostForm("remove_image")

	return req, fileHeader, removeImage
}

func (h *BaseHandler) UpdateBase(c *gin.Context) {
	id := c.Param("id")

	req, fileHeader, removeImage := h.parseUpdateBaseForm(c)

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	req.UpdatedBy = userId

	updatedBase, err := h.baseManagementService.UpdateBase(c.Request.Context(), schemaName, id, req, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	// Handle image: add if file provided, else remove if requested
	if fileHeader != nil {
		var imgErr error
		updatedBase, imgErr = h.baseManagementService.AddBaseImage(c.Request.Context(), schemaName, id, fileHeader, userId)
		if imgErr != nil {
			response.CheckAndSendError(c, imgErr)
			return
		}
	} else if removeImage == "true" {
		var remErr error
		updatedBase, remErr = h.baseManagementService.RemoveBaseImage(c.Request.Context(), schemaName, id, userId)
		if remErr != nil {
			response.CheckAndSendError(c, remErr)
			return
		}
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
