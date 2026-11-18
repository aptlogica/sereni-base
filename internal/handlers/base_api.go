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

type BaseAPIHandler struct {
	baseAPIService interfaces.BaseAPIService
}

func NewBaseAPIHandler(baseAPIService interfaces.BaseAPIService) *BaseAPIHandler {
	return &BaseAPIHandler{baseAPIService: baseAPIService}
}

// @Summary      Get the base API status
// @Description  Returns whether the base API is on and, once it has been turned on, the base's permanent public ID and API base path. A base that never turned it on is returned with enabled false.
// @Tags         Admin Base
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Base ID"
// @Success      200  {object}  dto.BaseAPIResponse
// @Security     BearerAuth
// @Router       /base/{id}/api [get]
func (h *BaseAPIHandler) GetBaseAPI(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	out, err := h.baseAPIService.Get(c, schemaName, c.Param("id"))
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, responseConst.BaseAPISuccess.Fetched, out)
}

// @Summary      Turn on the base API
// @Description  Gives the base its permanent public ID (bs_ + 10 characters) the first time, and turns the API on. Turning it on again keeps the same ID. Owner and co-owner only.
// @Tags         Admin Base
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Base ID"
// @Success      200  {object}  dto.BaseAPIResponse
// @Failure      404  {object}  models.ErrorResponse  "Base not found"
// @Failure      500  {object}  models.ErrorResponse  "No unused public ID could be generated"
// @Security     BearerAuth
// @Router       /base/{id}/api/enable [post]
func (h *BaseAPIHandler) EnableBaseAPI(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	out, err := h.baseAPIService.Enable(c, schemaName, c.Param("id"))
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, responseConst.BaseAPISuccess.Enabled, out)
}
