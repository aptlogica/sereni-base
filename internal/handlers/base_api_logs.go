// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"context"
	"errors"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/providers/logquery"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

type BaseAPILogsHandler struct {
	logService interfaces.BaseAPILogService
}

func NewBaseAPILogsHandler(logService interfaces.BaseAPILogService) *BaseAPILogsHandler {
	return &BaseAPILogsHandler{logService: logService}
}

// @Summary      List the API logs of a base
// @Description  Returns the requests made to the base's public API (/api/v1/bs_...), newest first. The answer is the one of the log service, as it is: {success, code, message, data:{query, count, entries, next_cursor}}. A base whose API was never turned on has no logs and gets an empty list. Logs appear a couple of seconds after the request. Any member who can open the base can read them.
// @Tags         Admin Base
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id          path   string  true   "Base ID"
// @Param        level       query  string  false  "debug, info, warn, error or fatal"
// @Param        status      query  int     false  "Exact HTTP status, e.g. 500"
// @Param        method      query  string  false  "HTTP method, e.g. GET"
// @Param        request_id  query  string  false  "One request"
// @Param        table       query  string  false  "Table of the request"
// @Param        error_code  query  string  false  "Error code, e.g. internal_error"
// @Param        search      query  string  false  "Text inside the log line"
// @Param        start       query  string  false  "Start time, RFC3339 (default: 24 hours ago)"
// @Param        end         query  string  false  "End time, RFC3339 (default: now)"
// @Param        limit       query  int     false  "Entries per page, 1 to 500 (default 100)"
// @Param        direction   query  string  false  "backward (newest first, default) or forward"
// @Param        cursor      query  string  false  "next_cursor of the previous page; send the same filters"
// @Success      200  {object}  map[string]interface{}  "Answer of the log service"
// @Failure      400  {object}  models.ErrorResponse  "Bad filter, or a window longer than the retention period"
// @Failure      502  {object}  models.ErrorResponse  "The log service failed"
// @Failure      503  {object}  models.ErrorResponse  "LOG_SERVICE_URL is not set"
// @Failure      504  {object}  models.ErrorResponse  "The log service timed out"
// @Security     BearerAuth
// @Router       /base/{id}/api/logs [get]
func (h *BaseAPILogsHandler) ListLogs(c *gin.Context) {
	h.serve(c, h.logService.List)
}

// @Summary      Count the API logs of a base
// @Description  Number of requests made to the base's public API for the same filters as the list (no limit, direction or cursor). The answer is the one of the log service, as it is: {success, code, message, data:{query, count, start, end}}.
// @Tags         Admin Base
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id          path   string  true   "Base ID"
// @Param        level       query  string  false  "debug, info, warn, error or fatal"
// @Param        status      query  int     false  "Exact HTTP status, e.g. 500"
// @Param        method      query  string  false  "HTTP method, e.g. GET"
// @Param        request_id  query  string  false  "One request"
// @Param        table       query  string  false  "Table of the request"
// @Param        error_code  query  string  false  "Error code, e.g. internal_error"
// @Param        search      query  string  false  "Text inside the log line"
// @Param        start       query  string  false  "Start time, RFC3339 (default: 24 hours ago)"
// @Param        end         query  string  false  "End time, RFC3339 (default: now)"
// @Success      200  {object}  map[string]interface{}  "Answer of the log service"
// @Failure      400  {object}  models.ErrorResponse  "Bad filter, or a window longer than the retention period"
// @Failure      502  {object}  models.ErrorResponse  "The log service failed"
// @Failure      503  {object}  models.ErrorResponse  "LOG_SERVICE_URL is not set"
// @Failure      504  {object}  models.ErrorResponse  "The log service timed out"
// @Security     BearerAuth
// @Router       /base/{id}/api/logs/count [get]
func (h *BaseAPILogsHandler) CountLogs(c *gin.Context) {
	h.serve(c, h.logService.Count)
}

// serve reads the filters, calls the service and writes the status and body of the log service as they are
func (h *BaseAPILogsHandler) serve(c *gin.Context, call func(ctx context.Context, schemaName, baseID string, req dto.BaseAPILogsRequest) (interfaces.BaseAPILogResult, error)) {
	var req dto.BaseAPILogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.SendErrorWithMessage(c, responseConst.BaseAPIError.LogFilterInvalid, "invalid query parameters")
		return
	}
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	res, err := call(c.Request.Context(), schemaName, c.Param("id"), req)
	if err != nil {
		sendLogError(c, err)
		return
	}
	c.Data(res.Status, "application/json; charset=utf-8", res.Body)
}

// sendLogError answers an error of the log read: our own format, never the one of the log service
func sendLogError(c *gin.Context, err error) {
	var bad *interfaces.InvalidLogFilterError
	switch {
	case errors.As(err, &bad):
		response.SendErrorWithMessage(c, responseConst.BaseAPIError.LogFilterInvalid, bad.Message)
	case errors.Is(err, logquery.ErrNotConfigured):
		response.SendError(c, responseConst.BaseAPIError.LogsNotConfigured)
	case errors.Is(err, logquery.ErrTimeout):
		logger.Get().Error().Err(err).Msg("base API logs: log service timed out")
		response.SendError(c, responseConst.BaseAPIError.LogsTimeout)
	case errors.Is(err, logquery.ErrUnavailable):
		logger.Get().Error().Err(err).Msg("base API logs: log service failed")
		response.SendError(c, responseConst.BaseAPIError.LogsUnavailable)
	default:
		response.CheckAndSendError(c, err)
	}
}
