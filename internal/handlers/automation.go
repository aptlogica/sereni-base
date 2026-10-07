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
	"github.com/aptlogica/sereni-base/internal/handlers/validators"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

type AutomationHandler struct {
	automationService interfaces.AutomationService
}

func NewAutomationHandler(automationService interfaces.AutomationService) *AutomationHandler {
	return &AutomationHandler{automationService: automationService}
}

// validateAutomationTitle sends the error and returns false when the title is empty or too long
func validateAutomationTitle(c *gin.Context, title string) bool {
	if title == "" {
		response.SendError(c, responseConst.AutomationError.TitleRequired)
		return false
	}
	if errCode, ok := validators.ValidateMaxNameOrTitleLength(title, responseConst.AutomationError.TitleTooLong); ok {
		response.SendError(c, errCode)
		return false
	}
	return true
}

// sendAutomationError returns the Postgres message when a query failed, so the user can fix it
func sendAutomationError(c *gin.Context, err error) {
	if errors.Is(err, app_errors.TriggerFailed) {
		message := strings.TrimPrefix(err.Error(), app_errors.TriggerFailed.Error()+": ")
		response.SendErrorWithMessage(c, responseConst.AutomationError.TriggerFailed, strings.TrimPrefix(message, "pq: "))
		return
	}
	response.CheckAndSendError(c, err)
}

// @Summary      Create a trigger or webhook
// @Description  Stores a trigger or webhook of a table. For type "trigger" the CREATE TRIGGER query in context is run on the table.
// @Tags         Automations
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreateAutomationRequest  true  "Automation payload"
// @Success      201      {object}  tenant.Automation
// @Failure      400      {object}  models.ErrorResponse  "Bad Request"
// @Security     BearerAuth
// @Router       /automation/create [post]
func (h *AutomationHandler) CreateAutomation(c *gin.Context) {
	var req dto.CreateAutomationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if !validateAutomationTitle(c, req.Title) {
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIDVal, _ := c.Get("user_id")
	req.CreatedBy, _ = userIDVal.(string)

	automation, err := h.automationService.Create(c, schemaName, req)
	if err != nil {
		sendAutomationError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AutomationSuccess.AutomationCreated, automation)
}

// @Summary      List the triggers, webhooks and functions of a table
// @Tags         Automations
// @Produce      json
// @Param        model_id  query     string  true  "Table (model) ID"
// @Success      200       {array}   tenant.Automation
// @Failure      400       {object}  models.ErrorResponse  "Bad Request — model_id missing"
// @Security     BearerAuth
// @Router       /automation [get]
func (h *AutomationHandler) GetAutomations(c *gin.Context) {
	modelID := strings.TrimSpace(c.Query("model_id"))
	if modelID == "" {
		response.SendError(c, responseConst.TableError.ModelIDRequired)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	automations, err := h.automationService.List(c, schemaName, modelID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AutomationSuccess.AutomationsFetched, automations)
}

// @Summary      Update a trigger, webhook or function
// @Description  Saves the title and query (the query format is checked). Postgres is not changed until the entry is run.
// @Tags         Automations
// @Accept       json
// @Produce      json
// @Param        id       path      string                       true  "Automation ID"
// @Param        request  body      dto.UpdateAutomationRequest  true  "New title and query"
// @Success      200      {object}  tenant.Automation
// @Failure      400      {object}  models.ErrorResponse  "Bad Request"
// @Security     BearerAuth
// @Router       /automation/{id} [put]
func (h *AutomationHandler) UpdateAutomation(c *gin.Context) {
	var req dto.UpdateAutomationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if !validateAutomationTitle(c, req.Title) {
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIDVal, _ := c.Get("user_id")
	req.UpdatedBy, _ = userIDVal.(string)

	automation, err := h.automationService.Update(c, schemaName, c.Param("id"), req)
	if err != nil {
		sendAutomationError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AutomationSuccess.AutomationUpdated, automation)
}

// @Summary      Apply a trigger or function in Postgres
// @Description  Applies the saved query in Postgres, replacing the trigger or function applied before, in one transaction.
// @Tags         Automations
// @Produce      json
// @Param        id   path      string  true  "Automation ID"
// @Success      200  {object}  tenant.Automation
// @Failure      400  {object}  models.ErrorResponse  "Bad Request"
// @Security     BearerAuth
// @Router       /automation/{id}/run [post]
func (h *AutomationHandler) RunAutomation(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	automation, err := h.automationService.Run(c, schemaName, c.Param("id"))
	if err != nil {
		sendAutomationError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AutomationSuccess.AutomationRun, automation)
}

// @Summary      Delete a trigger or webhook
// @Description  Deletes the automation. For a trigger it also drops the trigger from the table.
// @Tags         Automations
// @Produce      json
// @Param        id   path      string  true  "Automation ID"
// @Success      200  {object}  models.SuccessResponse
// @Security     BearerAuth
// @Router       /automation/{id} [delete]
func (h *AutomationHandler) DeleteAutomation(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	if err := h.automationService.Delete(c, schemaName, c.Param("id")); err != nil {
		sendAutomationError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AutomationSuccess.AutomationDeleted, nil)
}
