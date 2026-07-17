// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers/validators"
	_ "github.com/aptlogica/sereni-base/internal/models"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BaseHandler struct {
	baseManagementService interfaces.BaseManagementService
	importService         interfaces.ImportService
}

func NewBaseHandler(
	baseManagementService interfaces.BaseManagementService,
	importService interfaces.ImportService,
) *BaseHandler {
	fmt.Println("Initializing BaseHandler with services:", baseManagementService, importService)
	return &BaseHandler{
		baseManagementService: baseManagementService,
		importService:         importService,
	}
}

func validateBaseID(c *gin.Context, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		response.SendError(c, responseConst.BaseError.IdRequired)
		return false
	}

	if _, err := uuid.Parse(id); err != nil {
		response.SendError(c, responseConst.BaseError.IdInvalid)
		return false
	}

	return true
}

// @Summary      Create a new base
// @Description  Persists a base associated with a workspace and optional description, returning the stored base data. Optional image dimensions must not exceed 800x400 pixels.
// @Tags         Admin Base
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request     body      dto.CreateBaseRequest  true   "Base payload"
// @Param        workspace_id formData string                true   "Workspace ID ownership"
// @Param        image       formData  file                false  "Optional base image (max 800x400 pixels)"
// @Success      201         {object}  dto.BaseResponse     "Base created"
// @Failure      400         {object}  models.ErrorResponse  "Bad Request — missing title, workspace, or invalid image dimensions"
// @Failure      401         {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403         {object}  models.ErrorResponse  "Forbidden — not allowed to create base"
// @Failure      500         {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/create [post]
func (h *BaseHandler) CreateBase(c *gin.Context) {
	// Get form values
	title := c.PostForm("title")
	description := c.PostForm("description")
	workspaceID := c.PostForm("workspace_id")

	title = strings.TrimSpace(title)
	if title == "" {
		response.SendError(c, responseConst.BaseError.NameRequired)
		return
	}
	if errCode, ok := validators.ValidateMaxNameOrTitleLength(title, responseConst.BaseError.NameTooLong); ok {
		response.SendError(c, errCode)
		return
	}

	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		response.SendError(c, responseConst.WorkspaceError.IdRequired)
		return
	}

	// Get optional image file
	file, _ := c.FormFile("image")

	// Validate file size if image is provided
	if file != nil {
		maxSize := int64(config.AppConfig.Asset.MaxSize)
		if file.Size > maxSize {
			response.SendError(c, responseConst.BaseError.ImageTooLarge)
			return
		}
	}

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

// @Summary      Get base details
// @Description  Fetches the base identified by ID including visibility, config, and metadata.
// @Tags         Admin Base
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string         true  "Base ID"
// @Success      200  {object}  dto.BaseResponse  "Base retrieved successfully"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid base id"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden — missing access"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — base missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id} [get]
func (h *BaseHandler) GetBaseByID(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
		return
	}

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

// @Summary      Update base metadata
// @Description  Applies the form fields and optional uploaded image to update a base record. Image dimensions must not exceed 800x400 pixels.
// @Tags         Admin Base
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id           path      string          true  "Base ID"
// @Param        request      body      dto.BaseUpdate   true  "Fields to update"
// @Param        image        formData  file            false "New base image (max 800x400 pixels)"
// @Param        remove_image formData  string          false "Pass true to drop the existing image"
// @Success      200          {object}  dto.BaseResponse  "Updated base data"
// @Failure      400          {object}  models.ErrorResponse  "Bad Request — invalid payload or invalid image dimensions"
// @Failure      401          {object}  models.ErrorResponse  "Unauthorized — invalid token"
// @Failure      403          {object}  models.ErrorResponse  "Forbidden — insufficient privileges"
// @Failure      404          {object}  models.ErrorResponse  "Not Found — base missing"
// @Failure      500          {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id} [put]
func (h *BaseHandler) UpdateBase(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
		return
	}

	req, fileHeader, removeImage := h.parseUpdateBaseForm(c)
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			response.SendError(c, responseConst.BaseError.NameRequired)
			return
		}
		if errCode, ok := validators.ValidateMaxNameOrTitleLength(title, responseConst.BaseError.NameTooLong); ok {
			response.SendError(c, errCode)
			return
		}
		req.Title = &title
	}

	// Validate file size if image is provided
	if fileHeader != nil {
		maxSize := int64(config.AppConfig.Asset.MaxSize)
		if fileHeader.Size > maxSize {
			response.SendError(c, responseConst.BaseError.ImageTooLarge)
			return
		}
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	// Ensure we return a proper 404 when the base does not exist.
	if _, err := h.baseManagementService.GetBaseByID(c.Request.Context(), schemaName, id); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	req.UpdatedBy = userId

	updatedBase, err := h.baseManagementService.UpdateBase(c.Request.Context(), schemaName, id, req, userId, fileHeader, removeImage)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base updated successfully", updatedBase)
}

// @Summary      Delete a base
// @Description  Removes the base record permanently from the schema.
// @Tags         Admin Base
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Base ID"
// @Success      200  {object}  models.SuccessResponse  "Base deleted successfully"
// @Failure      400  {object}  models.ErrorResponse    "Bad Request — invalid base id"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse    "Forbidden — not allowed to delete base"
// @Failure      404  {object}  models.ErrorResponse    "Not Found — base missing"
// @Failure      500  {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id} [delete]
func (h *BaseHandler) DeleteBase(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	if err := h.baseManagementService.DeleteBase(c.Request.Context(), schemaName, id); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base deleted successfully", nil)
}

// @Summary      Get tables inside a base
// @Description  Lists the tables that were created under the base identified by the path ID.
// @Tags         Admin Base
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string            true  "Base ID"
// @Success      200  {array}   dto.TableResponse  "Tables retrieved"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden — no access to this base"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — base missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id}/tables [get]
func (h *BaseHandler) GetTablesByBaseId(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	// Ensure missing base returns a 404 as documented.
	if _, err := h.baseManagementService.GetBaseByID(c.Request.Context(), schemaName, id); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	tables, err := h.baseManagementService.GetTablesByBaseId(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "tables retrieved successfully", tables)
}

// @Summary      Upload a base image
// @Description  Attaches an image file to the base metadata and returns the updated base record. Image dimensions must not exceed 800x400 pixels.
// @Tags         Admin Base
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id    path      string  true  "Base ID"
// @Param        image formData  file    true  "Image to upload (max 800x400 pixels)"
// @Success      200   {object}  dto.BaseResponse  "Image stored and base returned"
// @Failure      400   {object}  models.ErrorResponse  "Bad Request — invalid id, missing image, or invalid dimensions"
// @Failure      401   {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403   {object}  models.ErrorResponse  "Forbidden — no permission to edit base image"
// @Failure      404   {object}  models.ErrorResponse  "Not Found — base missing"
// @Failure      500   {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id}/image [post]
func (h *BaseHandler) AddBaseImage(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	// Validate file size
	maxSize := int64(config.AppConfig.Asset.MaxSize)
	if file.Size > maxSize {
		response.SendError(c, responseConst.BaseError.ImageTooLarge)
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

// @Summary      Remove base image
// @Description  Deletes the stored image for a base and returns the updated base metadata.
// @Tags         Admin Base
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Base ID"
// @Success      200  {object}  dto.BaseResponse  "Image removed and base returned"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid id"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden — insufficient privileges"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — base missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id}/image [delete]
func (h *BaseHandler) RemoveBaseImage(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
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

func (h *BaseHandler) PreviewAiBase(c *gin.Context) {
	var body struct {
		Prompt string `json:"prompt" binding:"required"`
	}

	// Accept JSON or form; only prompt is required
	if err := c.ShouldBind(&body); err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	req := dto.CreateTableRequest{
		Prompt:    body.Prompt,
		CreatedBy: userId,
	}

	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}

	// fmt.Println("PreviewAiTable", req.Prompt)

	aiSchema, err := h.importService.FetchAiBaseSchema(c, req.Prompt)
	fmt.Println("aiSchema------>", aiSchema)
	if err != nil {
		fmt.Println("err===>>>", err)
		response.CheckAndSendError(c, err)
		return
	}

	// Return raw AI schema so frontend can preview/edit without meta block
	resp := response.StandardResponse{
		Success: true,
		Message: "Base fetched successfully",
		Data:    aiSchema,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *BaseHandler) ApplyAiBase(c *gin.Context) {
	var body struct {
		dto.AiBaseResponse
		WorkspaceID string `json:"workspace_id"`
		SampleData  bool   `json:"sample_data"`
		Row         int    `json:"row"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	aiBaseResp := body.AiBaseResponse
	// fmt.Println("base response----->", aiBaseResp)

	if aiBaseResp.BaseName == "" || body.WorkspaceID == "" || len(aiBaseResp.Tables) == 0 {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	baseReq := dto.CreateBaseRequest{
		Title:       aiBaseResp.BaseName,
		WorkspaceID: body.WorkspaceID,
		CreatedBy:   userId,
	}
	if baseReq.CreatedBy == "" {
		baseReq.CreatedBy = userId
	}

	base, err := h.baseManagementService.CreateBase(c.Request.Context(), baseReq, schemaName, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	applyReq := dto.CreateTableRequest{
		BaseID:      base.ID.String(),
		WorkspaceID: body.WorkspaceID,
		CreatedBy:   userId,
	}
	if applyReq.CreatedBy == "" {
		applyReq.CreatedBy = userId
	}

	importBaseResp, err := h.importService.ApplyAiBaseSchema(c, schemaName, applyReq, aiBaseResp, body.SampleData, body.Row)
	if err != nil {
		if delErr := h.baseManagementService.DeleteBase(c.Request.Context(), schemaName, base.ID.String()); delErr != nil {
			fmt.Println("failed to cleanup AI base after apply error:", delErr)
		}
		response.CheckAndSendError(c, err)
		return
	}

	resp := response.StandardResponse{
		Success: true,
		Message: responseConst.BaseSuccessCodes[responseConst.BaseSuccess.BaseCreated].Message,
		Data:    importBaseResp,
	}
	c.JSON(http.StatusCreated, resp)
}
