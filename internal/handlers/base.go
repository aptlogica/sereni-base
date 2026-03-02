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

// @Summary      Create a new base
// @Description  Persists a base associated with a workspace and optional description, returning the stored base data.
// @Tags         Admin Base
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request     body      dto.CreateBaseRequest  true   "Base payload"
// @Param        workspace_id formData string                true   "Workspace ID ownership"
// @Param        image       formData  file                false  "Optional base image"
// @Success      201         {object}  dto.BaseResponse     "Base created"
// @Failure      400         {object}  models.ErrorResponse  "Bad Request — missing title or workspace"
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
// @Description  Applies the form fields and optional uploaded image to update a base record.
// @Tags         Admin Base
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id           path      string          true  "Base ID"
// @Param        request      body      dto.BaseUpdate   true  "Fields to update"
// @Param        image        formData  file            false "New base image"
// @Param        remove_image formData  string          false "Pass true to drop the existing image"
// @Success      200          {object}  dto.BaseResponse  "Updated base data"
// @Failure      400          {object}  models.ErrorResponse  "Bad Request — invalid payload"
// @Failure      401          {object}  models.ErrorResponse  "Unauthorized — invalid token"
// @Failure      403          {object}  models.ErrorResponse  "Forbidden — insufficient privileges"
// @Failure      404          {object}  models.ErrorResponse  "Not Found — base missing"
// @Failure      500          {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id} [put]
func (h *BaseHandler) UpdateBase(c *gin.Context) {
	id := c.Param("id")

	req, fileHeader, removeImage := h.parseUpdateBaseForm(c)

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

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

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	tables, err := h.baseManagementService.GetTablesByBaseId(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "tables retrieved successfully", tables)
}

// @Summary      Upload a base image
// @Description  Attaches an image file to the base metadata and returns the updated base record.
// @Tags         Admin Base
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id    path      string  true  "Base ID"
// @Param        image formData  file    true  "Image to upload"
// @Success      200   {object}  dto.BaseResponse  "Image stored and base returned"
// @Failure      400   {object}  models.ErrorResponse  "Bad Request — invalid id or missing image"
// @Failure      401   {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403   {object}  models.ErrorResponse  "Forbidden — no permission to edit base image"
// @Failure      404   {object}  models.ErrorResponse  "Not Found — base missing"
// @Failure      500   {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id}/image [post]
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
