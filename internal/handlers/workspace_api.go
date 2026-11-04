// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

type WorkspaceAPIHandler struct {
	workspaceAPIService interfaces.WorkspaceAPIService
}

func NewWorkspaceAPIHandler(workspaceAPIService interfaces.WorkspaceAPIService) *WorkspaceAPIHandler {
	return &WorkspaceAPIHandler{workspaceAPIService: workspaceAPIService}
}

// @Summary      Get the workspace API status
// @Description  Returns whether the workspace API is on and, once it has been turned on, the workspace's permanent public ID and API base path. A workspace that never turned it on is returned with enabled false.
// @Tags         Admin Workspace
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Workspace ID"
// @Success      200  {object}  dto.WorkspaceAPIResponse
// @Security     BearerAuth
// @Router       /workspace/{id}/api [get]
func (h *WorkspaceAPIHandler) GetWorkspaceAPI(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	out, err := h.workspaceAPIService.Get(c, schemaName, c.Param("id"))
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, responseConst.WorkspaceAPISuccess.Fetched, out)
}

// @Summary      Turn on the workspace API
// @Description  Gives the workspace its permanent public ID (ws_ + 10 characters) the first time, and turns the API on. Turning it on again keeps the same ID. Owner and co-owner only.
// @Tags         Admin Workspace
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Workspace ID"
// @Success      200  {object}  dto.WorkspaceAPIResponse
// @Failure      404  {object}  models.ErrorResponse  "Workspace not found"
// @Failure      500  {object}  models.ErrorResponse  "No unused public ID could be generated"
// @Security     BearerAuth
// @Router       /workspace/{id}/api/enable [post]
func (h *WorkspaceAPIHandler) EnableWorkspaceAPI(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	out, err := h.workspaceAPIService.Enable(c, schemaName, c.Param("id"))
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, responseConst.WorkspaceAPISuccess.Enabled, out)
}
