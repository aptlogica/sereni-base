// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers/validators"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

type TableConstraintHandler struct {
	constraintService interfaces.TableConstraintService
}

func NewTableConstraintHandler(constraintService interfaces.TableConstraintService) *TableConstraintHandler {
	return &TableConstraintHandler{constraintService: constraintService}
}

// validateConstraintTitle sends the error and returns false when the title is empty or too long
func validateConstraintTitle(c *gin.Context, title string) bool {
	if title == "" {
		response.SendError(c, responseConst.TableConstraintError.TitleRequired)
		return false
	}
	if errCode, ok := validators.ValidateMaxNameOrTitleLength(title, responseConst.TableConstraintError.TitleTooLong); ok {
		response.SendError(c, errCode)
		return false
	}
	return true
}

func schemaFromContext(c *gin.Context) string {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)
	return schemaName
}

func userFromContext(c *gin.Context) string {
	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(string)
	return userID
}

// @Summary      List the unique keys of a table
// @Tags         Table Constraints
// @Produce      json
// @Param        model_id  query     string  true   "Table (model) ID"
// @Param        type      query     string  false  "Constraint type (default unique)"
// @Success      200       {array}   dto.TableConstraintResponse
// @Failure      400       {object}  models.ErrorResponse  "Bad Request — model_id missing"
// @Security     BearerAuth
// @Router       /table-constraint [get]
func (h *TableConstraintHandler) GetTableConstraints(c *gin.Context) {
	modelID := strings.TrimSpace(c.Query("model_id"))
	if modelID == "" {
		response.SendError(c, responseConst.TableError.ModelIDRequired)
		return
	}
	constraintType := strings.TrimSpace(c.DefaultQuery("type", tenant.ConstraintTypeUnique))

	constraints, err := h.constraintService.List(c, schemaFromContext(c), modelID, constraintType)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableConstraintSuccess.ConstraintsFetched, constraints)
}

// @Summary      Check records against a unique key
// @Description  Reports records that already repeat values in the columns. Nothing is changed.
// @Tags         Table Constraints
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CheckTableConstraintRequest  true  "Table and columns"
// @Success      200      {object}  dto.ConstraintCheckResult
// @Failure      400      {object}  models.ErrorResponse  "Bad Request"
// @Security     BearerAuth
// @Router       /table-constraint/check [post]
func (h *TableConstraintHandler) CheckTableConstraint(c *gin.Context) {
	var req dto.CheckTableConstraintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	result, err := h.constraintService.Check(c, schemaFromContext(c), req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableConstraintSuccess.ConstraintChecked, result)
}

// @Summary      Add a unique key to a table
// @Description  Adds a UNIQUE constraint on one or more columns. Fails when records already repeat values.
// @Tags         Table Constraints
// @Accept       json
// @Produce      json
// @Param        request  body      dto.CreateTableConstraintRequest  true  "Unique key"
// @Success      201      {object}  dto.TableConstraintResponse
// @Failure      400      {object}  models.ErrorResponse  "Bad Request"
// @Failure      409      {object}  models.ErrorResponse  "Duplicate records or same key exists"
// @Security     BearerAuth
// @Router       /table-constraint/create [post]
func (h *TableConstraintHandler) CreateTableConstraint(c *gin.Context) {
	var req dto.CreateTableConstraintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if !validateConstraintTitle(c, req.Title) {
		return
	}
	req.CreatedBy = userFromContext(c)

	constraint, err := h.constraintService.Create(c, schemaFromContext(c), req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableConstraintSuccess.ConstraintCreated, constraint)
}

// @Summary      Update a unique key
// @Description  Saves the name and message. When the columns or options change, the key is replaced in Postgres.
// @Tags         Table Constraints
// @Accept       json
// @Produce      json
// @Param        id       path      string                            true  "Constraint ID"
// @Param        request  body      dto.UpdateTableConstraintRequest  true  "Unique key"
// @Success      200      {object}  dto.TableConstraintResponse
// @Failure      400      {object}  models.ErrorResponse  "Bad Request"
// @Security     BearerAuth
// @Router       /table-constraint/{id} [put]
func (h *TableConstraintHandler) UpdateTableConstraint(c *gin.Context) {
	var req dto.UpdateTableConstraintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if !validateConstraintTitle(c, req.Title) {
		return
	}
	req.UpdatedBy = userFromContext(c)

	constraint, err := h.constraintService.Update(c, schemaFromContext(c), c.Param("id"), req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableConstraintSuccess.ConstraintUpdated, constraint)
}

// @Summary      Delete a unique key
// @Description  Drops the constraint from the table; records may repeat the values again.
// @Tags         Table Constraints
// @Produce      json
// @Param        id   path      string  true  "Constraint ID"
// @Success      200  {object}  models.SuccessResponse
// @Security     BearerAuth
// @Router       /table-constraint/{id} [delete]
func (h *TableConstraintHandler) DeleteTableConstraint(c *gin.Context) {
	if err := h.constraintService.Delete(c, schemaFromContext(c), c.Param("id")); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableConstraintSuccess.ConstraintDeleted, nil)
}
