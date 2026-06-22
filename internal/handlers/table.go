// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers/validators"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	_ "github.com/aptlogica/sereni-base/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type TableHandler struct {
	tableManagementService interfaces.TableManagementService
	importService          interfaces.ImportService
}

func isCSVImportFile(file *multipart.FileHeader) bool {
	if file == nil {
		return false
	}
	return strings.EqualFold(filepath.Ext(file.Filename), ".csv")
}

func NewTableHandler(tableManagementService interfaces.TableManagementService, importService interfaces.ImportService) *TableHandler {
	return &TableHandler{
		tableManagementService: tableManagementService,
		importService:          importService,
	}
}

// @Summary      Create a table
// @Description  Creates a new table (model) within the assigned base and workspace and returns the metadata needed to render it.
// @Tags         Admin Table Column Row
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.CreateTableRequest  true  "Table creation payload"
// @Success      201      {object}  dto.TableResponse       "Returns table metadata with columns and views"
// @Failure      400      {object}  models.ErrorResponse    "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse    "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse    "Forbidden — user lacks privileges"
// @Failure      422      {object}  models.ErrorResponse    "Unprocessable Entity — validation error"
// @Failure      500      {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/create [post]
func (h *TableHandler) CreateTable(c *gin.Context) {
	var req dto.CreateTableRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.CreateTableValidationErrors(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.SendError(c, responseConst.TableError.TitleRequired)
		return
	}
	if errCode, ok := validators.ValidateMaxNameOrTitleLength(req.Title, responseConst.TableError.ViewTitleTooLong); ok {
		response.SendError(c, errCode)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}

	table, err := h.tableManagementService.CreateTableWithDefaults(c, req, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableCreated, table)
}

// @Summary      Update a table
// @Description  Applies the provided table metadata updates (title, description, meta) for the specified table ID.
// @Tags         Admin Table Column Row
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string               true  "Table ID"
// @Param        request  body      dto.UpdateTableRequest  true  "Fields to patch"
// @Success      200      {object}  dto.TableResponse       "Updated table metadata"
// @Failure      400      {object}  models.ErrorResponse    "Bad Request — invalid ID or payload"
// @Failure      401      {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse    "Forbidden"
// @Failure      404      {object}  models.ErrorResponse    "Not Found — table missing"
// @Failure      500      {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/{id} [patch]
func (h *TableHandler) UpdateTable(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.TableError.ModelIDRequired)
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		response.SendError(c, responseConst.TableError.ModelIDInvalid)
		return
	}

	var req dto.UpdateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			response.SendError(c, responseConst.TableError.TitleRequired)
			return
		}
		if errCode, ok := validators.ValidateMaxNameOrTitleLength(title, responseConst.TableError.TitleTooLong); ok {
			response.SendError(c, errCode)
			return
		}
		req.Title = &title
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}

	table, err := h.tableManagementService.UpdateTable(c, id, req, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableUpdated, table)
}

// @Summary      Get table by ID
// @Description  Retrieves the complete table metadata for the provided model ID, including columns and views.
// @Tags         Admin Table Column Row
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string            true  "Table ID"
// @Success      200  {object}  dto.TableResponse  "Table retrieved successfully"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid ID"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — table missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/{id} [get]
func (h *TableHandler) GetTableByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	table, err := h.tableManagementService.GetTableByID(c, id, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableFetched, table)
}

// @Summary      List all tables
// @Description  Returns every table model defined in the tenant schema for discovery.
// @Tags         Admin Table Column Row
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200  {array}   dto.TableResponse  "All tables retrieved"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/ [get]
func (h *TableHandler) GetAllTables(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	tables, err := h.tableManagementService.GetAllTables(c, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableFetched, tables)
}

// @Summary      Add a column to a model
// @Description  Creates a new column within the specified model and returns the column definition.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.AddColumnRequest  true  "Column creation payload"
// @Success      200      {object}  dto.ColumnResponse    "Column created"
// @Failure      400      {object}  models.ErrorResponse  "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse  "Forbidden"
// @Failure      422      {object}  models.ErrorResponse  "Unprocessable Entity — validation failed"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/create [post]
func (h *TableHandler) AddColumn(c *gin.Context) {
	var req dto.AddColumnRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.AddColumnValidator(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if _, err := h.tableManagementService.GetTableByID(c, req.ModelID.String(), schemaName); err != nil {
		if errors.Is(err, app_errors.TableNotFound) {
			response.SendError(c, responseConst.TableError.TableNotFound)
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}

	column, err := h.tableManagementService.AddColumn(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnAdded, column)
}

// @Summary      Get column by ID
// @Description  Returns a single column definition so clients can inspect its type and metadata.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string          true  "Column ID"
// @Success      200  {object}  dto.ColumnResponse  "Column retrieved successfully"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid column ID"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — column missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/{id} [get]
func (h *TableHandler) GetColumnById(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	column, err := h.tableManagementService.GetColumnById(c, schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnFetched, column)
}

// @Summary      List all columns
// @Description  Returns every column defined in the tenant schema.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200  {array}   dto.ColumnResponse  "Columns retrieved"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/ [get]
func (h *TableHandler) GetAllColumns(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	columns, err := h.tableManagementService.GetAllColumns(c, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnFetched, columns)
}

// @Summary      Get columns for a specific table
// @Description  Returns the columns belonging to the table with the provided ID.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string            true  "Table ID"
// @Success      200  {array}   dto.ColumnResponse  "Columns listed"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — table missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/{id}/columns [get]
func (h *TableHandler) GetColumnsByTable(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	table, err := h.tableManagementService.GetColumnsByModelID(c, schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnFetched, table)
}

// @Summary      Create a view
// @Description  Persists a new view configuration tied to a model and returns the created view metadata.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.CreateViewRequest  true  "View creation payload"
// @Success      200      {object}  dto.ViewResponse        "View created"
// @Failure      400      {object}  models.ErrorResponse    "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse    "Forbidden"
// @Failure      500      {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /view/create [post]
func (h *TableHandler) CreateView(c *gin.Context) {
	var req dto.CreateViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.CreateViewValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.SendError(c, responseConst.TableError.TitleRequired)
		return
	}
	if errCode, ok := validators.ValidateMaxNameOrTitleLength(req.Title, responseConst.TableError.TitleTooLong); ok {
		response.SendError(c, errCode)
		return
	}

	if errCode, ok := validators.ValidateCreateViewMeta(req); ok {
		response.SendError(c, errCode)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}

	view, err := h.tableManagementService.CreateView(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ViewCreated, view)
}

// @Summary      Get view by ID
// @Description  Retrieves view metadata for the provided view identifier.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string          true  "View ID"
// @Success      200  {object}  dto.ViewResponse  "View retrieved"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid id"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — view missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /view/{id} [get]
func (h *TableHandler) GetViewByID(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	view, err := h.tableManagementService.GetViewByID(c, schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ViewFetched, view)
}

// @Summary      List all views
// @Description  Returns every saved view definition available in the tenant schema.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200  {array}   dto.ViewResponse  "Views retrieved"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /view/ [get]
func (h *TableHandler) GetAllViews(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	views, err := h.tableManagementService.GetAllViews(c, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ViewFetched, views)
}

// @Summary      Get views for a table
// @Description  Returns view definitions linked to the specific model ID.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string            true  "Table ID"
// @Success      200  {array}   dto.ViewResponse  "Views returned"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — table missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/{id}/views [get]
func (h *TableHandler) GetViewsByModelID(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	views, err := h.tableManagementService.GetViewsByModelID(c, schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ViewFetched, views)
}

// @Summary      Update a view
// @Description  Updates the requested view configuration while treating missing fields as no-ops.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string        true  "View ID"
// @Param        request  body      dto.ViewUpdate  true  "Fields to update"
// @Success      200      {object}  dto.ViewResponse  "Updated view returned"
// @Failure      400      {object}  models.ErrorResponse  "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse  "Forbidden"
// @Failure      404      {object}  models.ErrorResponse  "Not Found — view missing"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /view/{id} [patch]
func (h *TableHandler) UpdateView(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	var req dto.ViewUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			response.SendError(c, responseConst.TableError.TitleRequired)
			return
		}
		if errCode, ok := validators.ValidateMaxNameOrTitleLength(title, responseConst.TableError.TitleTooLong); ok {
			response.SendError(c, errCode)
			return
		}
		req.Title = &title
	}

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}

	view, err := h.tableManagementService.UpdateView(c, schemaName, id, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ViewUpdated, view)
}

// @Summary      Delete a view
// @Description  Removes the view definition and all associated filters/metadata.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "View ID"
// @Success      200  {object}  models.SuccessResponse  "View deleted"
// @Failure      400  {object}  models.ErrorResponse    "Bad Request — invalid view id"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse    "Forbidden"
// @Failure      404  {object}  models.ErrorResponse    "Not Found — view missing"
// @Failure      500  {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /view/{id} [delete]
func (h *TableHandler) DeleteView(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	err := h.tableManagementService.DeleteView(c, schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ViewDeleted, nil)
}

// @Summary      Update a column
// @Description  Updates column metadata (title, meta) for the provided column ID.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string          true  "Column ID"
// @Param        request  body      dto.ColumnUpdate  true  "Column fields to update"
// @Success      200      {object}  dto.ColumnResponse "Updated column returned"
// @Failure      400      {object}  models.ErrorResponse  "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse  "Forbidden"
// @Failure      404      {object}  models.ErrorResponse  "Not Found — column missing"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/{id} [patch]
func (h *TableHandler) UpdateColumn(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	var req dto.ColumnUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}

	column, err := h.tableManagementService.UpdateColumn(c, schemaName, id, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnUpdated, column)
}

// @Summary      Delete a column
// @Description  Removes the column definition permanently from the model.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Column ID"
// @Success      200  {object}  models.SuccessResponse  "Column deleted"
// @Failure      400  {object}  models.ErrorResponse    "Bad Request — invalid column id"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse    "Forbidden"
// @Failure      404  {object}  models.ErrorResponse    "Not Found — column missing"
// @Failure      500  {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/{id} [delete]
func (h *TableHandler) DeleteColumn(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	err := h.tableManagementService.DeleteColumn(c, schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnDeleted, nil)
}

// @Summary      Create row(s)
// @Description  Creates a single row when only model_id is provided, or creates rows with values when rows[] is provided.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.CreateRowOrBulkInsertRequest  true  "Row create or bulk insert payload"
// @Success      200      {object}  models.SuccessResponse  "Single row record or bulk insert summary in data"
// @Failure      400      {object}  models.ErrorResponse   "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse   "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse   "Forbidden"
// @Failure      422      {object}  models.ErrorResponse   "Unprocessable Entity — invalid data"
// @Failure      500      {object}  models.ErrorResponse   "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/create [post]
func (h *TableHandler) CreateRow(c *gin.Context) {
	var req dto.CreateRowOrBulkInsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.CreateRowRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}
	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}

	if len(req.Rows) > 0 {
		bulkInsertService, ok := h.tableManagementService.(interface {
			CreateRowsWithValues(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error)
		})
		if !ok {
			response.CheckAndSendError(c, fmt.Errorf("table service does not support CreateRowsWithValues"))
			return
		}

		rows, err := bulkInsertService.CreateRowsWithValues(c, schemaName, req.ModelID, req.Rows, req.CreatedBy, req.UpdatedBy)
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}

		response.SendSuccess(c, responseConst.TableSuccess.RowDataInserted, gin.H{
			"inserted_count": len(rows),
			"rows":           rows,
		})
		return
	}

	record, err := h.tableManagementService.CreateRow(c, schemaName, dto.CreateRowRequest{
		ModelID:   req.ModelID,
		CreatedBy: req.CreatedBy,
	})
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RecordCreated, record)
}

// @Summary      Update a row
// @Description  Partially patches a single row by applying provided column values in one API call.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.UpdateRowRequest  true  "Model ID, row ID, and direct column-value map"
// @Success      200      {object}  dto.RecordResponse    "Updated row returned"
// @Failure      400      {object}  models.ErrorResponse  "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse  "Forbidden"
// @Failure      422      {object}  models.ErrorResponse  "Unprocessable Entity — invalid value"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/update [patch]
func (h *TableHandler) UpdateRow(c *gin.Context) {
	var req dto.UpdateRowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.UpdateRowRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}

	updatedRow := dto.RecordResponse{}
	for columnID, rawValue := range req.Values {
		var valuePtr *interface{}
		if rawValue != nil {
			value := rawValue
			valuePtr = &value
		}

		record, err := h.tableManagementService.InsertRowData(c, schemaName, dto.InsertRowDataRequest{
			ModelID:   req.ModelID,
			ColumnId:  columnID,
			RowId:     req.RowId,
			Value:     valuePtr,
			UpdatedBy: req.UpdatedBy,
		})
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}
		updatedRow = record
	}

	response.SendSuccess(c, responseConst.TableSuccess.RowDataInserted, updatedRow)
}

// @Summary      Update link data for rows
// @Description  Updates relationship columns to link rows together and returns the updated row metadata.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.UpdateRowDataLinksRequest  true  "Link target/source definitions"
// @Success      200      {object}  dto.RecordResponse              "Row link updated"
// @Failure      400      {object}  models.ErrorResponse            "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse            "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse            "Forbidden"
// @Failure      404      {object}  models.ErrorResponse            "Not Found — columns or rows missing"
// @Failure      500      {object}  models.ErrorResponse            "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/data/relation [post]
func (h *TableHandler) InsertRowDataForLinks(c *gin.Context) {
	var req dto.UpdateRowDataLinksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.UpdateRowDataLinksRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}

	record, err := h.tableManagementService.UpdateRawDataForLinks(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RecordCreated, record)
}

// @Summary      Attach files to a row
// @Description  Uploads attachments for the given model/column/row triplet and returns the row record with references.
// @Tags         Admin Table Column
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        model_id  formData  string  true  "Model ID"
// @Param        column_id formData  string  true  "Column ID"
// @Param        row_id    formData  int     true  "Row index"
// @Param        files     formData  file    true  "Attachment files"
// @Success      200       {object}  dto.RecordResponse  "Row updated with attachment references"
// @Failure      400       {object}  models.ErrorResponse  "Bad Request — missing ids or files"
// @Failure      401       {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403       {object}  models.ErrorResponse  "Forbidden"
// @Failure      415       {object}  models.ErrorResponse  "Unsupported Media Type — upload blocked"
// @Failure      500       {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/attachment/add [post]
func (h *TableHandler) AddAttachment(c *gin.Context) {
	// Parse multipart form (with a reasonable memory limit)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32 MB
		response.SendError(c, responseConst.AssetError.MultipartFormNotFound)
		return
	}

	form := c.Request.MultipartForm
	files := form.File["files"]
	if len(files) == 0 {
		response.SendError(c, responseConst.AssetError.FilesNotFound)
		return
	}

	attatchementReq := c.Request.PostForm

	// Bind form fields into struct (from form values) using attatchementReq
	var req dto.AddAttachmentRequest
	modelID := attatchementReq.Get("model_id")
	if modelID == "" {
		response.SendError(c, responseConst.TableError.ModelIDInvalid)
		return
	}
	req.ModelID = modelID

	columnId := attatchementReq.Get("column_id")
	if columnId == "" {
		response.SendError(c, responseConst.TableError.ColumnIdInvalid)
		return
	}
	req.ColumnId = columnId

	rowIdStr := attatchementReq.Get("row_id")
	rowId, err := strconv.Atoi(rowIdStr)
	if err != nil {
		response.SendError(c, responseConst.TableError.RowIdInvalid)
		return
	}
	req.RowId = rowId

	// Get schema name from context
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	// Call service to add attachment
	record, err := h.tableManagementService.AddAttachment(c, schemaName, req, files)
	if err != nil {
		response.SendErrorWithMessage(c, responseConst.AssetError.VirusDetected, err.Error())
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RowDataInserted, record)
}

// @Summary      Update attachment metadata
// @Description  Updates attachment metadata (such as filename, description, etc.) for a given attachment reference in a row.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.UpdateAttachmentRequest  true  "Attachment update payload"
// @Success      200      {object}  dto.RecordResponse           "Row updated with new attachment metadata"
// @Failure      400      {object}  models.ErrorResponse         "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse         "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse         "Forbidden"
// @Failure      404      {object}  models.ErrorResponse         "Not Found — attachment missing"
// @Failure      500      {object}  models.ErrorResponse         "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/attachment/update [post]
func (h *TableHandler) UpdateAttachment(c *gin.Context) {
	var req dto.UpdateAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.UpdateAttachmentRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	record, err := h.tableManagementService.UpdateAttachment(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RowDataInserted, record)
}

// @Summary      Remove attachments from a row
// @Description  Deletes the requested attachment references for a row and returns the updated record metadata.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.RemoveAttachmentsRequest  true  "Rows and attachments to remove"
// @Success      200      {object}  dto.RecordResponse            "Row returned without attachments"
// @Failure      400      {object}  models.ErrorResponse          "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse          "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse          "Forbidden"
// @Failure      404      {object}  models.ErrorResponse          "Not Found — attachment missing"
// @Failure      500      {object}  models.ErrorResponse          "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/attachment/remove [post]
func (h *TableHandler) RemoveAttachments(c *gin.Context) {
	var req dto.RemoveAttachmentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.RemoveAttachmentsRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	record, err := h.tableManagementService.RemoveAttachments(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RowDataRemoved, record)
}

// @Summary      Get all records for a table
// @Description  Retrieves every record stored under the specified model ID.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string              true  "Table ID"
// @Success      200  {object}  dto.RecordsResponse  "Records returned"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — table missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/{id}/records [get]
func (h *TableHandler) GetAllRecords(c *gin.Context) {
	id := c.Param("id")

	schemaName, _ := c.Get("schema")
	schemaNameStr := schemaName.(string)

	records, err := h.tableManagementService.GetAllRecords(c, schemaNameStr, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RecordsFetched, records)
}

// @Summary      Insert or update row cell
// @Description  Sets the value for a specific column cell and returns the updated row record.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.InsertRowDataRequest  true  "Column and row data payload"
// @Success      200      {object}  dto.RecordResponse        "Updated record returned"
// @Failure      400      {object}  models.ErrorResponse      "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse      "Forbidden"
// @Failure      422      {object}  models.ErrorResponse      "Unprocessable Entity — invalid value"
// @Failure      500      {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/data/insert [post]
func (h *TableHandler) InsertRowData(c *gin.Context) {
	var req dto.InsertRowDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.InsertRowDataRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}

	record, err := h.tableManagementService.InsertRowData(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RowDataInserted, record)
}

// @Summary      Delete a row
// @Description  Removes the row described in the payload from the model.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.DeleteRowDataRequest  true  "Model ID and row ID to delete"
// @Success      200      {object}  models.SuccessResponse     "Row removed"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — row missing"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/remove [post]
func (h *TableHandler) DeleteRow(c *gin.Context) {
	var req dto.DeleteRowDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.DeleteRowDataRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	if err := h.tableManagementService.DeleteRow(c, schemaName, req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RowDeleted, nil)
}

// @Summary      Delete a table
// @Description  Deletes the table metadata and related records for the specified ID.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Table ID"
// @Success      200  {object}  models.SuccessResponse  "Table deleted"
// @Failure      400  {object}  models.ErrorResponse    "Bad Request — invalid ID"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse    "Forbidden"
// @Failure      404  {object}  models.ErrorResponse    "Not Found — table missing"
// @Failure      500  {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/{id} [delete]
func (h *TableHandler) DeleteTable(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	if err := h.tableManagementService.DeleteTable(c, schemaName, id); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableDeleted, nil)
}

// @Summary      Reorder columns
// @Description  Moves the column order by taking a source and target column and returning the refreshed ordering.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.ReorderColumnRequest  true  "Source and target column IDs"
// @Success      200      {array}   dto.ColumnResponse       "Columns re-ordered"
// @Failure      400      {object}  models.ErrorResponse    "Bad Request — invalid columns"
// @Failure      401      {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse    "Forbidden"
// @Failure      500      {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/reorder [post]
func (h *TableHandler) ReorderColumn(c *gin.Context) {
	var req dto.ReorderColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ReorderColumnRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	updatedColumns, err := h.tableManagementService.ReorderColumn(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnReordered, updatedColumns)
}

// @Summary      Import a table via file with configuration
// @Description  Imports table schema/data from an uploaded file with user-provided column configuration and data cleaning settings.
// @Tags         Admin Table Column
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        file         formData  file    true   "CSV file to import"
// @Param        base_id      formData  string  true   "Base ID to associate"
// @Param        workspace_id formData  string  true   "Workspace ID"
// @Param        title        formData  string  true   "Table name"
// @Param        description  formData  string  false  "Table description"
// @Param        order_index  formData  string  false  "Numeric order index"
// @Param        config       formData  string  true   "JSON config with settings and column configurations"
// @Success      200          {object}  dto.TableResponse  "Table imported with custom config"
// @Failure      400          {object}  models.ErrorResponse  "Bad Request — missing required fields"
// @Failure      401          {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403          {object}  models.ErrorResponse  "Forbidden"
// @Failure      422          {object}  models.ErrorResponse  "Unprocessable Entity — invalid config or file"
// @Failure      500          {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /table/import [post]
func (h *TableHandler) ImportTableWithConfig(c *gin.Context) {
	// Get and validate file
	file, err := c.FormFile("file")
	if err != nil {
		response.SendError(c, responseConst.AssetError.FilesNotFound)
		return
	}

	if !isCSVImportFile(file) {
		response.SendErrorWithMessage(c, responseConst.AssetError.InvalidFileFormat, "Only CSV files are allowed")
		return
	}

	// Enforce import file size limit
	maxImportSize := int64(config.AppConfig.Import.MaxSize)
	if maxImportSize <= 0 {
		maxImportSize = 2097152 // 2MB fallback
	}
	if file.Size > maxImportSize {
		response.SendError(c, responseConst.AssetError.FileTooLargeError)
		return
	}

	// Parse JSON config from form
	configJSON := c.PostForm("config")
	if configJSON == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	importConfig, err := parseImportConfig(configJSON)
	if err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	// Map the provided primary column name to its matching column config.
	primaryName := c.PostForm("primary_column")
	if primaryName != "" {
		if err := setPrimaryColumn(&importConfig, primaryName); err != nil {
			response.SendErrorWithMessage(c, responseConst.Error.InvalidPayload, "primary_column not found in config columns")
			return
		}
	}

	// Build import request from context values and config
	req := buildImportRequestFromContext(c, importConfig)

	// Compute tableTitle from uploaded filename (strip extension).
	tableTitle := file.Filename
	if lastDot := strings.LastIndex(tableTitle, "."); lastDot != -1 {
		tableTitle = tableTitle[:lastDot]
	}

	// Get schema from context
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	// Call import service with config
	// Pass computed tableTitle to the import service (title/description not accepted from form)
	tableResp, err := h.importService.ImportWithConfig(c.Request.Context(), schemaName, req, file, tableTitle)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableCreated, tableResp)
}

// parseImportConfig parses the import config JSON into a dto.ImportConfig
func parseImportConfig(configJSON string) (dto.ImportConfig, error) {
	var importConfig dto.ImportConfig
	if err := json.Unmarshal([]byte(configJSON), &importConfig); err != nil {
		return importConfig, err
	}
	return importConfig, nil
}

// setPrimaryColumn finds the named column in importConfig and sets PrimaryColumn
func setPrimaryColumn(importConfig *dto.ImportConfig, primaryName string) error {
	for _, col := range importConfig.Columns {
		if col.ColumnName == primaryName {
			copyCol := col
			importConfig.PrimaryColumn = &copyCol
			return nil
		}
	}
	return errors.New("primary_column not found in config columns")
}

// buildImportRequestFromContext builds dto.ImportWithConfigRequest from the gin context and parsed config
func buildImportRequestFromContext(c *gin.Context, importConfig dto.ImportConfig) dto.ImportWithConfigRequest {
	var req dto.ImportWithConfigRequest
	req.BaseID = c.PostForm("base_id")
	req.WorkspaceID = c.PostForm("workspace_id")
	req.Config = importConfig

	orderIndexStr := c.PostForm("order_index")
	if orderIndexStr != "" {
		if orderIndexInt, err := strconv.Atoi(orderIndexStr); err == nil {
			req.OrderIndex = float64(orderIndexInt)
		} else if orderIndexFloat, err := strconv.ParseFloat(orderIndexStr, 64); err == nil {
			req.OrderIndex = orderIndexFloat
		}
	}

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)
	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}

	return req
}

// @Summary      Bulk delete rows
// @Description  Deletes multiple rows for the same model and returns the total count along with a summary message.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.BulkDeleteRowsRequest  true  "Model ID and row IDs to delete"
// @Success      200      {object}  models.SuccessResponse      "Deleted rows summary (data contains deleted_count/message)"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /row/bulk-remove [post]
func (h *TableHandler) BulkDeleteRows(c *gin.Context) {
	var req dto.BulkDeleteRowsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.BulkDeleteRowsRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)
	deletedCount, err := h.tableManagementService.BulkDeleteRows(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, responseConst.TableSuccess.RowDeleted, gin.H{
		"deleted_count": deletedCount,
		"message":       fmt.Sprintf("Successfully deleted %d rows", deletedCount),
	})
}

// @Summary      Bulk update columns
// @Description  Updates multiple columns with the provided metadata and returns success status.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.BulkUpdateColumnsRequest  true  "Model ID and array of column updates"
// @Success      200      {object}  models.SuccessResponse      "Bulk update completed successfully"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/bulk-update [post]
func (h *TableHandler) BulkUpdateColumns(c *gin.Context) {
	var req dto.BulkUpdateColumnsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.BulkUpdateColumnsRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	err := h.tableManagementService.BulkUpdateColumns(c, schemaName, req.ModelID, req.ColumnID, req.Updates)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnUpdated, gin.H{
		"message": "Columns updated successfully",
		"count":   len(req.Updates),
	})
}

// @Summary      Reset column values
// @Description  Sets all values in a specified column to NULL across all rows.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.ResetColumnValuesRequest  true  "Model ID and column ID to reset"
// @Success      200      {object}  models.SuccessResponse        "Column values reset successfully"
// @Failure      400      {object}  models.ErrorResponse         "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse         "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse         "Forbidden"
// @Failure      404      {object}  models.ErrorResponse         "Not Found — column missing"
// @Failure      500      {object}  models.ErrorResponse         "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/reset [post]
func (h *TableHandler) ResetColumnValues(c *gin.Context) {
	var req dto.ResetColumnValuesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ResetColumnValuesRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	err := h.tableManagementService.ResetColumnValues(c, schemaName, req.ModelID, req.ColumnId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnUpdated, gin.H{
		"message":  "Column values reset to NULL successfully",
		"columnId": req.ColumnId,
	})
}



// @Summary      Trim whitespace in selected columns
// @Description  Cleans whitespace for selected columns by fetching records directly from the database, skipping NULL/non-string values, and updating only changed cells.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.TrimWhitespaceRequest  true  "Model ID, target column IDs, trim mode, and optional extra-space cleanup"
// @Success      200      {object}  models.SuccessResponse     "Whitespace cleanup summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — model/column missing"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/trim-whitespace [post]
func (h *TableHandler) TrimWhitespace(c *gin.Context) {
	var req dto.TrimWhitespaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.TrimWhitespaceRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)
	result, err := h.tableManagementService.TrimWhitespace(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	var successCode responseConst.ResponseCode
	switch req.TrimMode {
	case "trim_both":
		successCode = responseConst.TableSuccess.TrimWhitespaceBoth
	case "trim_leading":
		successCode = responseConst.TableSuccess.TrimWhitespaceLeading
	case "trim_trailing":
		successCode = responseConst.TableSuccess.TrimWhitespaceTrailing
	case "collapse_spaces":
		successCode = responseConst.TableSuccess.TrimWhitespaceCollapse
	default:
		successCode = responseConst.TableSuccess.TrimWhitespaceBoth
	}
	response.SendSuccess(c, successCode, result)
}

// @Summary      Find and replace values in selected columns
// @Description  Finds occurrences of `find_value` in the selected columns and replaces them with `replace_value`. The backend fetches target records directly from the DB, skips NULL/non-string values, and updates only changed cells.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.FindReplaceRequest  true  "Model ID, target column IDs, find value, replace value, and matching preference"
// @Success      200      {object}  models.SuccessResponse     "Find & Replace summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — model/column missing"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/find-replace [post]
func (h *TableHandler) FindReplace(c *gin.Context) {
	var req dto.FindReplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.FindReplaceRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	result, err := h.tableManagementService.FindReplace(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	var successCode responseConst.ResponseCode
	switch req.MatchType {
	case "match_case":
		successCode = responseConst.TableSuccess.FindReplaceMatchCase
	case "ignore_case":
		successCode = responseConst.TableSuccess.FindReplaceIgnoreCase
	case "match_entire_value":
		successCode = responseConst.TableSuccess.FindReplaceEntireValue
	default:
		successCode = responseConst.TableSuccess.FindReplace
	}

	response.SendSuccess(c, successCode, result)
}

// @Summary      Normalize case in selected columns
// @Description  Normalizes text case for selected columns by fetching records directly from the database, skipping NULL/non-string values, and updating only changed cells.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.CaseNormalizationRequest  true  "Model ID, target column IDs, and desired case format"
// @Success      200      {object}  models.SuccessResponse     "Normalization summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — model/column missing"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/case-normalize [post]
func (h *TableHandler) CaseNormalization(c *gin.Context) {
	var req dto.CaseNormalizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.CaseNormalizationRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	if svcWithCase, ok := h.tableManagementService.(interface {
		CaseNormalization(ctx context.Context, schemaName string, req dto.CaseNormalizationRequest) (dto.CaseNormalizationResponse, error)
	}); ok {
		result, err := svcWithCase.CaseNormalization(c, schemaName, req)
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}

		var successCode responseConst.ResponseCode
		switch req.CaseFormat {
		case "lowercase":
			successCode = responseConst.TableSuccess.CaseNormalizationLowercase
		case "uppercase":
			successCode = responseConst.TableSuccess.CaseNormalizationUppercase
		case "title_case":
			successCode = responseConst.TableSuccess.CaseNormalizationTitleCase
		case "sentence_case":
			successCode = responseConst.TableSuccess.CaseNormalizationSentenceCase
		default:
			successCode = responseConst.TableSuccess.ColumnUpdated
		}

		response.SendSuccess(c, successCode, result)
		return
	}

	response.SendError(c, responseConst.Error.NotImplemented)
}

// @Summary      Remove special characters from selected columns
// @Description  Removes special characters from selected columns by fetching records directly from the database, skipping NULL/non-string values, and updating only changed cells.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.RemoveSpecialCharactersRequest  true  "Model ID, target column IDs, remove type, and optional custom character"
// @Success      200      {object}  models.SuccessResponse     "Special character removal summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — model/column missing"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/remove-special-characters [post]
func (h *TableHandler) RemoveSpecialCharacters(c *gin.Context) {
	var req dto.RemoveSpecialCharactersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.RemoveSpecialCharactersRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	result, err := h.tableManagementService.RemoveSpecialCharacters(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	var successCode responseConst.ResponseCode
	switch req.SpecialCharactersType {
	case "symbols":
		successCode = responseConst.TableSuccess.RemoveSpecialCharactersSymbols
	case "currency_symbols":
		successCode = responseConst.TableSuccess.RemoveSpecialCharactersCurrency
	case "brackets":
		successCode = responseConst.TableSuccess.RemoveSpecialCharactersBrackets
	case "punctuation":
		successCode = responseConst.TableSuccess.RemoveSpecialCharactersPunctuation
	case "custom":
		successCode = responseConst.TableSuccess.RemoveSpecialCharactersCustom
	default:
		successCode = responseConst.TableSuccess.RemoveSpecialCharacters
	}

	response.SendSuccess(c, successCode, result)
}

// @Summary      Split a column into multiple columns
// @Description  Splits the selected column using separator, fixed-length, or regex rules, creates new columns, and optionally removes the original column.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.ColumnSplitRequest  true  "Model ID, column ID, split strategy, keep original flag, and placement"
// @Success      200      {object}  dto.ColumnSplitResponse  "Split summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse     "Bad Request â€” invalid payload"
// @Failure      401      {object}  models.ErrorResponse     "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse     "Forbidden"
// @Failure      404      {object}  models.ErrorResponse     "Not Found â€” model/column missing"
// @Failure      500      {object}  models.ErrorResponse     "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/split [post]
func (h *TableHandler) ColumnSplit(c *gin.Context) {
	var req dto.ColumnSplitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ColumnSplitRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	result, err := h.tableManagementService.ColumnSplit(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ColumnSplit, result)
}

// @Summary      Remove formatting from selected columns
// @Description  Removes formatting from selected columns by fetching records directly from the database, skipping NULL values, updating only changed cells, and preserving column-level updates only.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.RemoveFormattingRequest  true  "Model ID, target column IDs, formatting type, and optional custom pattern"
// @Success      200      {object}  models.SuccessResponse      "Formatting removal summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse        "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse        "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse        "Forbidden"
// @Failure      404      {object}  models.ErrorResponse        "Not Found — model/column missing"
// @Failure      500      {object}  models.ErrorResponse        "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/remove-formatting [post]
func (h *TableHandler) RemoveFormatting(c *gin.Context) {
	var req dto.RemoveFormattingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.RemoveFormattingRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	result, err := h.tableManagementService.RemoveFormatting(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, removeFormattingSuccessCode(req.Formatting), result)
}

func removeFormattingSuccessCode(formatting string) responseConst.ResponseCode {
	switch formatting {
	case "currency":
		return responseConst.TableSuccess.RemoveFormattingCurrency
	case "percentage":
		return responseConst.TableSuccess.RemoveFormattingPercentage
	case "separator":
		return responseConst.TableSuccess.RemoveFormattingSeparator
	case "phone":
		return responseConst.TableSuccess.RemoveFormattingPhone
	case "date":
		return responseConst.TableSuccess.RemoveFormattingDate
	case "custom":
		return responseConst.TableSuccess.RemoveFormattingCustom
	default:
		return responseConst.TableSuccess.RemoveFormatting
	}
}

// @Summary      Remove duplicate rows or clear duplicate values in selected columns
// @Description  Deduplicates rows using database-side grouping and batching. Supports removing duplicate rows or keeping duplicate rows while clearing duplicate values in selected columns. Rows where all selected columns are empty/null are excluded from duplicate detection.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.RemoveDuplicatesRequest  true  "Model ID, target column IDs, duplicate handling, and keep rule"
// @Success      200      {object}  models.SuccessResponse     "Deduplication summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — model/column missing"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/remove-duplicates [post]
func (h *TableHandler) RemoveDuplicates(c *gin.Context) {
	var req dto.RemoveDuplicatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.RemoveDuplicatesRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	result, err := h.tableManagementService.RemoveDuplicates(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RemoveDuplicates, result)
}

// @Summary      Merge selected columns into a new column
// @Description  Merges values from selected columns into a new generated column using the requested separator. Skips NULL/empty values and updates only changed cells.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.MergeColumnsRequest  true  "Model ID, target column IDs, merge format, and options"
// @Success      200      {object}  models.SuccessResponse     "Merge summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — model/column missing"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/merge-columns [post]
func (h *TableHandler) MergeColumns(c *gin.Context) {
	var req dto.MergeColumnsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.MergeColumnsRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	result, err := h.tableManagementService.MergeColumns(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.MergeColumns, result)
}

// @Summary      Extract substring from a column
// @Description  Extracts substring from a single selected column between two delimiters and creates a new generated column populated with extracted values.
// @Tags         Admin Table Column
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.ExtractSubstringRequest  true  "Model ID, column_id (column ID only), extraction_method, extraction_type (when using extraction_type), start_after and end_before (when using between_characters), keep_original_column, add_at_end"
// @Success      200      {object}  models.SuccessResponse     "Extraction summary returned in success.data"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse       "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse       "Forbidden"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — model/column missing"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Security     BearerAuth
// @Router       /column/extract-substring [post]
func (h *TableHandler) ExtractSubstring(c *gin.Context) {
	var req dto.ExtractSubstringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ExtractSubstringRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	result, err := h.tableManagementService.ExtractSubstring(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.ExtractSubstring, result)
}