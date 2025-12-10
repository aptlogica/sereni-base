package handlers

import (
	"fmt"
	"serenibase/internal/dto"
	"serenibase/internal/handlers/validators"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type TableHandler struct {
	tableManagementService interfaces.TableManagementService
	importService          interfaces.ImportService
}

func NewTableHandler(tableManagementService interfaces.TableManagementService, importService interfaces.ImportService) *TableHandler {
	return &TableHandler{
		tableManagementService: tableManagementService,
		importService:          importService,
	}
}

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

func (h *TableHandler) UpdateTable(c *gin.Context) {

	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	var req dto.UpdateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	table, err := h.tableManagementService.UpdateTable(c, id, req, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableUpdated, table)
}

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

	// Explicitly initialize pageSize, pageNumber to 0 first
	pageStr, pageSizeStr := c.Query("page"), c.Query("page_size")
	pageSize, pageNumber := 0, 0

	if len(pageStr) > 0 {
		pn, err := strconv.Atoi(pageStr)
		if err != nil {
			response.SendError(c, responseConst.TableError.PageRequired)
			return
		}
		if pn < 1 {
			response.SendError(c, responseConst.TableError.PageInvalid)
			return
		}
		pageNumber = pn
	}

	if len(pageSizeStr) > 0 {
		ps, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			response.SendError(c, responseConst.TableError.LimitRequired)
			return
		}
		if ps < 1 {
			response.SendError(c, responseConst.TableError.LimitInvalid)
			return
		}
		pageSize = ps
	}
	table, err := h.tableManagementService.GetTableByID(c, id, schemaName, pageSize, pageNumber)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableFetched, table)
}

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

func (h *TableHandler) UpdateView(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	var req dto.ViewUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
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

func (h *TableHandler) CreateRow(c *gin.Context) {
	var req dto.CreateRowRequest
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

	record, err := h.tableManagementService.CreateRow(c, schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RecordCreated, record)
}

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

	response.SendSuccess(c, responseConst.TableSuccess.RowDataInserted, record)
}

func (h *TableHandler) GetAllRecords(c *gin.Context) {
	id := c.Param("id")

	schemaName, _ := c.Get("schema")
	schemaNameStr := schemaName.(string)

	pageStr, pageSizeStr := c.Query("page"), c.Query("page_size")
	hasPagination := pageStr != "" || pageSizeStr != ""

	if hasPagination {
		var req dto.PaginationRequest
		req.ModelID = id
		pageNumber, err := strconv.Atoi(pageStr)
		if err != nil {
			response.SendError(c, responseConst.TableError.PageInvalid)
			return
		}
		req.PageNumber = pageNumber
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			response.SendError(c, responseConst.TableError.LimitRequired)
			return
		}
		req.PageSize = pageSize

		if req.PageNumber < 1 {
			response.SendError(c, responseConst.TableError.PageInvalid)
			return
		}
		if req.PageSize < 1 {
			response.SendError(c, responseConst.TableError.LimitInvalid)
			return
		}

		records, err := h.tableManagementService.GetTableDataPagination(c, req, schemaNameStr)
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}
		response.SendSuccess(c, responseConst.TableSuccess.RecordsFetched, records)
		return
	}

	records, err := h.tableManagementService.GetAllRecords(c, schemaNameStr, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.RecordsFetched, records)
}

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

func (h *TableHandler) GetTableDataPagination(c *gin.Context) {
	var req dto.PaginationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.PaginationRequestValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	tableResponse, err := h.tableManagementService.GetTableDataPagination(c, req, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableFetched, tableResponse)
}

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

func (h *TableHandler) ImportTable(c *gin.Context) {
	// Expect multipart form with both file and fields
	file, err := c.FormFile("file")
	if err != nil {
		response.SendError(c, responseConst.AssetError.FilesNotFound)
		return
	}

	var req dto.CreateTableRequest
	req.BaseID = c.PostForm("base_id")
	req.WorkspaceID = c.PostForm("workspace_id")
	req.Title = c.PostForm("title")
	req.Description = c.PostForm("description")

	orderIndexStr := c.PostForm("order_index")
	if orderIndexStr != "" {
		if orderIndexInt, err := strconv.Atoi(orderIndexStr); err == nil {
			req.OrderIndex = float64(orderIndexInt)
		} else if orderIndexFloat, err := strconv.ParseFloat(orderIndexStr, 64); err == nil {
			req.OrderIndex = orderIndexFloat
		}
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}

	table, err := h.importService.Import(c, schemaName, req, file)
	if err != nil {
		fmt.Println("err===>>>", err)
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TableSuccess.TableCreated, table)
}
