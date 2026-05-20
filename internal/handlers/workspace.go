// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers/validators"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type WorkspaceHandler struct {
	workspaceManagementService interfaces.WorkspaceManagementService
	authManagementService      interfaces.AuthManagementService
}

func NewWorkspaceHandler(workspaceManagementService interfaces.WorkspaceManagementService, authManagementService interfaces.AuthManagementService) *WorkspaceHandler {
	return &WorkspaceHandler{
		workspaceManagementService: workspaceManagementService,
		authManagementService:      authManagementService,
	}
}

// isWorkspaceNotFound checks if error is a workspace not found error
func isWorkspaceNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "workspace not found")
}

// @Summary      Create a workspace
// @Description  Persists a new workspace entity for the tenant, storing the title and metadata provided by the requester.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.CreateWorkspaceRequest  true  "Workspace payload"
// @Success      201      {object}  dto.WorkspaceResponse      "Workspace created"
// @Failure      400      {object}  models.ErrorResponse        "Bad Request — missing title or workspace data"
// @Failure      401      {object}  models.ErrorResponse        "Unauthorized — invalid credentials"
// @Failure      403      {object}  models.ErrorResponse        "Forbidden — insufficient privileges"
// @Failure      500      {object}  models.ErrorResponse        "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/create [post]
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

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.SendError(c, responseConst.WorkspaceError.NameRequired)
		return
	}

	if errCode, ok := validators.ValidateMaxNameOrTitleLength(req.Title, responseConst.WorkspaceError.NameTooLong); ok {
		response.SendError(c, errCode)
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

// @Summary      Get workspace by ID
// @Description  Returns the workspace record that matches the provided ID with all tenant metadata.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string            true  "Workspace ID"
// @Success      200  {object}  models.SuccessResponse  "Workspace retrieved successfully (data contains workspace fields)"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid ID"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized — invalid token"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden — not allowed to read this workspace"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — workspace missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id} [get]
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

// @Summary      List all workspaces
// @Description  Returns every workspace visible to the tenant including metadata.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200  {object}  models.SuccessResponse  "Workspaces retrieved successfully (data contains workspace list)"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized — invalid token"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden — insufficient privileges"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/ [get]
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

// @Summary      Update workspace metadata
// @Description  Applies the supplied fields to the workspace record. Empty fields are ignored.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string               true  "Workspace ID"
// @Param        request  body      dto.WorkspaceUpdate   true  "Fields to patch"
// @Success      200      {object}  models.SuccessResponse  "Workspace updated successfully (data contains updated workspace)"
// @Failure      400      {object}  models.ErrorResponse  "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse  "Forbidden — insufficient privileges"
// @Failure      404      {object}  models.ErrorResponse  "Not Found — workspace missing"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id} [put]
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

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			response.SendError(c, responseConst.WorkspaceError.NameRequired)
			return
		}
		req.Title = &title
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

// @Summary      Delete a workspace
// @Description  Removes the workspace and its tables, returning success once cleanup finishes.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Workspace ID"
// @Success      200  {object}  models.SuccessResponse  "Workspace deleted"
// @Failure      400  {object}  models.ErrorResponse    "Bad Request — invalid workspace id"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized — invalid token"
// @Failure      403  {object}  models.ErrorResponse    "Forbidden — not allowed to delete"
// @Failure      404  {object}  models.ErrorResponse    "Not Found — workspace missing"
// @Failure      500  {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id} [delete]
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

// @Summary      Get tables scoped to a workspace
// @Description  Returns the table definitions owned by the workspace so front-end can render the schema.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string            true  "Workspace ID"
// @Success      200  {array}   dto.TableResponse  "Tables listed"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid workspace ID"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized — invalid token"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden — insufficient access"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — workspace missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id}/tables [get]
func (h *WorkspaceHandler) GetTablesByWorkspaceId(c *gin.Context) {
	workspaceID := c.Param("id") // expects route like /workspaces/:id/tables

	// Validate workspace ID is not empty
	if workspaceID == "" {
		response.SendError(c, responseConst.WorkspaceError.IdRequired)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	// Validate workspace exists before fetching tables
	_, err := h.workspaceManagementService.GetByID(c.Request.Context(), schemaName, workspaceID)
	if err != nil {
		// Return 404 if workspace not found
		if isWorkspaceNotFound(err) {
			response.SendError(c, responseConst.WorkspaceError.ErrNotFound)
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	// Get all tables for the workspace
	tables, err := h.workspaceManagementService.GetTablesByWorkspaceId(c.Request.Context(), schemaName, workspaceID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableFetched, tables)
}

// @Summary      List bases for a workspace
// @Description  Returns every base the workspace owns, including access-level meta for the requesting user.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string            true  "Workspace ID"
// @Success      200  {array}   dto.BaseResponse   "Bases retrieved"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid workspace ID"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized — invalid authentication"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden — missing access level"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — workspace missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id}/bases [get]
func (h *WorkspaceHandler) GetBasesByWorkspaceId(c *gin.Context) {
	workspaceID := c.Param("id") // expects route like /workspaces/:id/bases

	// Validate workspace ID is not empty
	if workspaceID == "" {
		response.SendError(c, responseConst.WorkspaceError.IdRequired)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.(string)

	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(string)

	// Validate workspace exists before fetching bases
	_, err := h.workspaceManagementService.GetByID(c.Request.Context(), schemaName, workspaceID)
	if err != nil {
		// Return 404 if workspace not found
		if isWorkspaceNotFound(err) {
			response.SendError(c, responseConst.WorkspaceError.ErrNotFound)
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	// Get all bases for the workspace
	bases, err := h.workspaceManagementService.GetAllBasesByWorkspaceId(c.Request.Context(), schemaName, workspaceID, roles, userID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.BaseSuccess.BasesFetched, bases)
}

// BulkAddMembers adds multiple members to workspace with their memberships
// @Summary      Bulk add members to workspace
// @Description  Creates multiple workspace access records using the memberships described in the request.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string                  true  "Target workspace ID"
// @Param        request  body      dto.BulkAddMembersRequest  true  "Array of member and membership definitions"
// @Success      200      {object}  dto.BulkAddMembersResponse  "Bulk member addition results"
// @Failure      400      {object}  models.ErrorResponse        "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse        "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse        "Forbidden — not allowed to bulk add"
// @Failure      500      {object}  models.ErrorResponse        "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id}/bulk-add-members [post]
func (h *WorkspaceHandler) BulkAddMembers(c *gin.Context) {
	var req dto.BulkAddMembersRequest

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

	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(string)

	result, err := h.authManagementService.BulkAddMembers(c.Request.Context(), schemaName, req, userID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Members added to workspace", result)
}

// BulkAddBaseMembers adds multiple members to bases
// @Summary      Bulk add members to a base
// @Description  Adds multiple users to the provided base according to their requested base roles.
// @Tags         Admin Workspace
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string                       true  "Base ID"
// @Param        request  body      dto.BulkAddBaseMembersRequest  true  "Bulk membership payload"
// @Success      200      {object}  dto.BulkAddMembersResponse     "Bulk additions reported"
// @Failure      400      {object}  models.ErrorResponse           "Bad Request — missing fields"
// @Failure      401      {object}  models.ErrorResponse           "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse           "Forbidden — insufficient privileges"
// @Failure      500      {object}  models.ErrorResponse           "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id}/bulk-add-members [post]
func (h *WorkspaceHandler) BulkAddBaseMembers(c *gin.Context) {
	baseID := c.Param("id")
	var req dto.BulkAddBaseMembersRequest

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

	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(string)

	result, err := h.authManagementService.BulkAddBaseMembers(c.Request.Context(), schemaName, baseID, req, userID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Members added to base", result)
}
