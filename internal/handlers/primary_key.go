// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"errors"
	"strings"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

type PrimaryKeyHandler struct {
	primaryKeyService interfaces.PrimaryKeyService
}

func NewPrimaryKeyHandler(primaryKeyService interfaces.PrimaryKeyService) *PrimaryKeyHandler {
	return &PrimaryKeyHandler{primaryKeyService: primaryKeyService}
}

// sendPrimaryKeyError returns the offending records when existing data blocks the change, and the exact
// reason (which field, which link fields, the Postgres detail) for the other failures
func sendPrimaryKeyError(c *gin.Context, err error) {
	var violation *app_errors.PrimaryKeyViolation
	if errors.As(err, &violation) {
		response.SendErrorWithData(c, responseConst.PrimaryKeyError.DataInvalid, violation.Message, violation.Details)
		return
	}
	// These errors carry the exact reason after "<sentinel>: "
	for _, known := range []struct {
		err  error
		code responseConst.ResponseCode
	}{
		{app_errors.PrimaryKeyUpdateFailed, responseConst.PrimaryKeyError.UpdateFailed},
		{app_errors.PrimaryKeyColumnInvalid, responseConst.PrimaryKeyError.ColumnInvalid},
		{app_errors.PrimaryKeyLocked, responseConst.PrimaryKeyError.Locked},
	} {
		if !errors.Is(err, known.err) {
			continue
		}
		if message, ok := strings.CutPrefix(err.Error(), known.err.Error()+": "); ok {
			response.SendErrorWithMessage(c, known.code, strings.TrimPrefix(message, "pq: "))
			return
		}
		response.SendError(c, known.code)
		return
	}
	response.CheckAndSendError(c, err)
}

// @Summary      Get the primary key of a table
// @Description  Returns the key fields in key order, every field with whether it can be a key field, and whether link fields lock the key.
// @Tags         Admin Table Column Row
// @Produce      json
// @Param        id   path      string  true  "Table (model) ID"
// @Success      200  {object}  dto.PrimaryKeyResponse
// @Failure      404  {object}  models.ErrorResponse  "Table not found"
// @Security     BearerAuth
// @Router       /table/{id}/primary-key [get]
func (h *PrimaryKeyHandler) GetPrimaryKey(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	out, err := h.primaryKeyService.Get(c, schemaName, c.Param("id"))
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, responseConst.PrimaryKeySuccess.Fetched, out)
}

// @Summary      Change the primary key of a table
// @Description  Sets the key fields (one or more, in order). Existing records must have every key field filled in and unique combined values; otherwise PKY_1004 is returned with the offending records in data. Not allowed while the table has link fields.
// @Tags         Admin Table Column Row
// @Accept       json
// @Produce      json
// @Param        id       path      string                       true  "Table (model) ID"
// @Param        request  body      dto.UpdatePrimaryKeyRequest  true  "Key field IDs in key order"
// @Success      200      {object}  dto.PrimaryKeyResponse
// @Failure      400      {object}  models.ErrorResponse  "No or invalid key fields"
// @Failure      409      {object}  models.ErrorResponse  "Locked by link fields, or existing records do not satisfy the key"
// @Security     BearerAuth
// @Router       /table/{id}/primary-key [put]
func (h *PrimaryKeyHandler) UpdatePrimaryKey(c *gin.Context) {
	var req dto.UpdatePrimaryKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	if len(req.ColumnIDs) == 0 {
		response.SendError(c, responseConst.PrimaryKeyError.Required)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	out, err := h.primaryKeyService.Update(c, schemaName, c.Param("id"), req)
	if err != nil {
		sendPrimaryKeyError(c, err)
		return
	}
	response.SendSuccess(c, responseConst.PrimaryKeySuccess.Updated, out)
}
