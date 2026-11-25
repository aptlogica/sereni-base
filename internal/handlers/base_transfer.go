// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	csvformat "github.com/aptlogica/sereni-base/internal/services/transfer/adapter/csv"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/jsonfmt"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/sbase"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/xlsx"
	"github.com/aptlogica/sereni-base/internal/services/transfer/formats"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// maxBaseImportBytes caps an uploaded base file, and the JSON read out of a zip
const maxBaseImportBytes = 100 << 20

// BaseTransferHandler exports and imports bases. The core service builds and creates bases as a model; an adapter
// (internal/services/transfer/adapter) turns the model into a file and back, so the handler writes no file format.
type BaseTransferHandler struct {
	transferService interfaces.BaseTransferService
	// jobs runs the exports with data in the background; without it they are built during the request
	jobs interfaces.BaseTransferJobService
	// adapters are the file formats
	adapters *adapter.Registry
}

func NewBaseTransferHandler(transferService interfaces.BaseTransferService) *BaseTransferHandler {
	return &BaseTransferHandler{transferService: transferService, adapters: formats.Registry()}
}

// NewBaseTransferHandlerWithJobs is NewBaseTransferHandler with background exports: an export with data
// then returns a job to follow instead of the zip.
func NewBaseTransferHandlerWithJobs(transferService interfaces.BaseTransferService, jobs interfaces.BaseTransferJobService) *BaseTransferHandler {
	return &BaseTransferHandler{transferService: transferService, jobs: jobs, adapters: formats.Registry()}
}

// format returns the adapter of the SereniBase file format
func (h *BaseTransferHandler) format() adapter.Adapter {
	a, _ := h.adapters.Get(sbase.Name)
	return a
}

// AutomationsAllowedKey is the context key under which the router says whether the user may read the trigger
// functions and triggers of the base. Without it (or when false) an export leaves them out.
const AutomationsAllowedKey = "base_transfer_automations_allowed"

// ExportDependencies lists the tables of a base and which table needs which, for the export dialog's warnings.
// GET /base/:id/export-dependencies
// Triggers and functions are only looked at for a user who has the automations permission.
func (h *BaseTransferHandler) ExportDependencies(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
		return
	}
	schemaName, _ := c.Get("schema")
	schema, _ := schemaName.(string)
	result, err := h.transferService.ExportDependencies(c.Request.Context(), schema, id, c.GetBool(AutomationsAllowedKey))
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, "export dependencies loaded", result)
}

// ExportBase exports a base.
// GET /base/:id/export?include_data=true
// Options:
//   - include_data: boolean (default false). When true, exports records.
//   - include_attachments: boolean (default matches include_data). When true, packs attachment files into the zip.
//   - include_system_columns: boolean (default false). Writes id, created / modified time and user of every row; needs include_data.
//   - include_automations: boolean (default true for users with automations permission, false otherwise). Exports triggers and functions.
//   - tables: comma-separated list or multiple query parameters of table IDs, titles, or keys to export.
//   - format: "sbase" (default), "csv", "xlsx" or "json". A csv is a zip with one file per table (and attachments/ with the files);
//     an xlsx is one workbook with one sheet per table (and, with attachments, a zip of the workbook and attachments/).
//     In both, links and lookups are written as text, and triggers and system columns are not exported.
//   - delimiter: only for csv: "comma" (default), "semicolon", "tab" or "pipe". Anything else is 400; so is a delimiter with xlsx.
//
// An export is a zip, .sbase.zip (base.sbase.json plus the attachment files and cover image when asked), except an xlsx
// export, which is a single .xlsx file unless it carries attachment files (then .xlsx.zip).
// Without include_data it is returned directly. With it, it can take a while, so it runs
// as a background job: the response is 202 with the job (poll GET /export-job/:id, then download it from
// GET /export-job/:id/download); the finished zip is also sent to the user's email. A user can have two exports
// active at once; asking again for a base that is already being exported returns that job ("reused": true), and a
// third one at the same time is refused with 429.
// Triggers and trigger functions are only exported for a user who has the automations permission
// (the router sets AutomationsAllowedKey); for a user who does not, they are left out.
func (h *BaseTransferHandler) ExportBase(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
		return
	}
	includeData, err := queryBool(c, "include_data", false)
	if err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	includeAttachments := includeData
	if c.Query("include_attachments") != "" {
		includeAttachments, err = queryBool(c, "include_attachments", includeData)
		if err != nil {
			response.SendError(c, responseConst.Error.InvalidPayload)
			return
		}
	} else if c.Query("attachments") != "" {
		includeAttachments, err = queryBool(c, "attachments", includeData)
		if err != nil {
			response.SendError(c, responseConst.Error.InvalidPayload)
			return
		}
	}

	includeSystemColumns, err := queryBool(c, "include_system_columns", false)
	if err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	automationsAllowed := c.GetBool(AutomationsAllowedKey)
	includeAutomations := automationsAllowed

	if c.Query("include_automations") != "" {
		incAuto, err := queryBool(c, "include_automations", automationsAllowed)
		if err != nil {
			response.SendError(c, responseConst.Error.InvalidPayload)
			return
		}
		includeAutomations = automationsAllowed && incAuto
	} else if c.Query("automations") != "" {
		incAuto, err := queryBool(c, "automations", automationsAllowed)
		if err != nil {
			response.SendError(c, responseConst.Error.InvalidPayload)
			return
		}
		includeAutomations = automationsAllowed && incAuto
	} else if c.Query("include_triggers") != "" || c.Query("include_functions") != "" {
		incTrig, _ := queryBool(c, "include_triggers", automationsAllowed)
		incFn, _ := queryBool(c, "include_functions", automationsAllowed)
		includeAutomations = automationsAllowed && (incTrig || incFn)
	}

	opts := dto.ExportBaseOptions{
		IncludeData:        includeData,
		IncludeAttachments: includeAttachments,
		IncludeAutomations: includeAutomations,
		// source columns are values of the rows, so they need the data
		IncludeSystemColumns: includeSystemColumns && includeData,
		Tables:               queryTables(c),
	}
	if !applyExportFormat(c, &opts, c.Query("format"), c.Query("delimiter")) {
		return
	}
	h.runExport(c, id, opts)
}

// applyExportFormat sets the file format of an export (sbase by default, or csv) and, for csv, the delimiter. An
// unknown format or delimiter is a 400. A csv cannot hold triggers, system columns or links, so those are switched
// off (links become text).
func applyExportFormat(c *gin.Context, opts *dto.ExportBaseOptions, format, delimiter string) bool {
	format = strings.ToLower(strings.TrimSpace(format))
	delimiter = strings.ToLower(strings.TrimSpace(delimiter))
	switch format {
	case "", sbase.Name:
		opts.Format = sbase.Name
		return true
	case jsonfmt.Name:
		// a json export is a zip with one file per table, as a csv is: no delimiter, no triggers or system columns,
		// and links become text. The attachment files are in attachments/ when they are asked for.
		if delimiter != "" {
			response.SendError(c, responseConst.Error.InvalidPayload)
			return false
		}
		opts.Format = jsonfmt.Name
		opts.LinksAsText = true
		opts.IncludeAutomations = false
		opts.IncludeSystemColumns = false
		return true
	case xlsx.Name:
		// a workbook has no delimiter and carries no triggers or system columns; links become text. The attachment
		// files travel next to it in a zip when they are asked for (include_attachments, on with the data).
		if delimiter != "" {
			response.SendError(c, responseConst.Error.InvalidPayload)
			return false
		}
		opts.Format = xlsx.Name
		opts.LinksAsText = true
		opts.IncludeAutomations = false
		opts.IncludeSystemColumns = false
		return true
	case csvformat.Name:
		opts.Format = csvformat.Name
	default:
		response.SendError(c, responseConst.Error.InvalidPayload)
		return false
	}
	opts.Delimiter = ','
	if delimiter != "" {
		d, ok := csvformat.Delimiters[delimiter]
		if !ok {
			response.SendError(c, responseConst.Error.InvalidPayload)
			return false
		}
		opts.Delimiter = d
	}
	opts.LinksAsText = true
	opts.IncludeAutomations = false
	opts.IncludeSystemColumns = false
	return true
}

// ExportBaseWithBody exports a base with the options in a JSON body, which can also hold row filters.
// POST /base/:id/export
// Body: {"include_data": true, "include_attachments": true, "include_automations": false, "include_system_columns": false, "tables": ["<table>"],
// "filters": {"<table>": {"conditions": [{"column": "<field id>", "operator": "is equal", "value": "Active"},
// {"column": "<field id>", "operator": "greater than", "value": "100", "logic": "AND"}]}}}
// The options mean the same as in the query of GET. A filter keeps only the matching rows of its table, so it needs
// include_data; a table that has a filter must be part of the export. A filter that is not valid is refused with
// 400 and the list of what is wrong. The response is the same as for GET: a zip, or 202 with a job.
func (h *BaseTransferHandler) ExportBaseWithBody(c *gin.Context) {
	id := c.Param("id")
	if !validateBaseID(c, id) {
		return
	}
	var req dto.ExportBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}
	includeData := req.IncludeData != nil && *req.IncludeData
	includeAttachments := includeData
	if req.IncludeAttachments != nil {
		includeAttachments = *req.IncludeAttachments
	}
	automationsAllowed := c.GetBool(AutomationsAllowedKey)
	includeAutomations := automationsAllowed
	if req.IncludeAutomations != nil {
		includeAutomations = automationsAllowed && *req.IncludeAutomations
	}
	opts := dto.ExportBaseOptions{
		IncludeData:          includeData,
		IncludeAttachments:   includeAttachments,
		IncludeAutomations:   includeAutomations,
		IncludeSystemColumns: includeData && req.IncludeSystemColumns != nil && *req.IncludeSystemColumns,
		Tables:               cleanTableNames(req.Tables),
		Filters:              req.Filters,
	}
	if !applyExportFormat(c, &opts, req.Format, req.Delimiter) {
		return
	}

	if len(opts.Filters) > 0 {
		if !includeData {
			response.SendErrorWithData(c, responseConst.Error.ValidationFailed, "the export filters are not valid",
				[]string{"row filters need include_data to be true"})
			return
		}
		schemaName, _ := c.Get("schema")
		schema, _ := schemaName.(string)
		err := h.transferService.ValidateExportFilters(c.Request.Context(), schema, id, opts.Tables, opts.Filters)
		var invalid *dto.ExportFilterError
		if errors.As(err, &invalid) {
			response.SendErrorWithData(c, responseConst.Error.ValidationFailed, "the export filters are not valid", invalid.Problems)
			return
		}
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}
	}
	h.runExport(c, id, opts)
}

// runExport exports a base with the options: as a background job when it has data, otherwise as a direct download.
func (h *BaseTransferHandler) runExport(c *gin.Context, id string, exportOpts dto.ExportBaseOptions) {
	includeData, includeAttachments := exportOpts.IncludeData, exportOpts.IncludeAttachments

	schemaName, _ := c.Get("schema")
	schema, _ := schemaName.(string)

	if includeData && h.jobs != nil {
		userID, _ := c.Get("user_id")
		user, _ := userID.(string)
		job, err := h.jobs.StartExport(c.Request.Context(), schema, id, user, exportOpts)
		if errors.Is(err, app_errors.TransferLimitReached) {
			response.SendError(c, responseConst.BaseError.TransferLimitReached)
			return
		}
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}
		response.SendSuccess(c, responseConst.BaseSuccess.ExportStarted, job)
		return
	}

	file, err := h.transferService.ExportBase(c.Request.Context(), schema, id, exportOpts)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	// the core service read the base into a model; the adapter writes it as a file
	format := h.adapterFor(exportOpts.Format)
	opts := adapter.ExportOptions{IncludeData: includeData, IncludeFiles: includeAttachments, OpenFile: h.transferService.OpenFile, Delimiter: exportOpts.Delimiter}
	filename, contentType := format.Describe(opts, file.Base.Title)
	h.sendArchive(c, format, opts, file, filename, contentType)
}

// exportAdapter returns the adapter of the format a request asks for; an empty name is the SereniBase format
func (h *BaseTransferHandler) adapterFor(name string) adapter.Adapter {
	if a, ok := h.adapters.Get(name); ok && name != "" {
		return a
	}
	return h.format()
}

// cleanTableNames trims the names of the tables of an export and drops blanks and duplicates
func cleanTableNames(names []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// sendArchive downloads the export that holds files (a zip). It is built in a temporary file first, so a failure
// while reading an attachment still produces a clean error response.
func (h *BaseTransferHandler) sendArchive(c *gin.Context, format adapter.Adapter, opts adapter.ExportOptions, file *dto.BaseExportFile, filename, contentType string) {
	tmp, err := os.CreateTemp("", "sbase-export-*.zip")
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if err := format.Export(c.Request.Context(), file, opts, tmp); err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	info, err := tmp.Stat()
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.DataFromReader(http.StatusOK, info.Size(), contentType, tmp, nil)
}

// importFileProblem says why the name of an uploaded file is refused for a format, or "" when it is fine. An sbase or
// csv export is a zip, and that is the only file an import of them takes; an xlsx export is a workbook (.xlsx), or a
// zip of the workbook and its attachment files; a json import also takes one .json file, which is one table.
func importFileProblem(format, filename string) string {
	lowerName := strings.ToLower(filename)
	if format == xlsx.Name {
		if !strings.HasSuffix(lowerName, xlsx.Extension) && !strings.HasSuffix(lowerName, ".zip") {
			return "only .xlsx or .zip files are supported"
		}
	} else if format == jsonfmt.Name {
		if !strings.HasSuffix(lowerName, jsonfmt.Extension) && !strings.HasSuffix(lowerName, ".zip") {
			return "only .zip or .json files are supported"
		}
	} else if !strings.HasSuffix(lowerName, ".zip") {
		return "only .zip files are supported"
	}
	return ""
}

// importFlags reads the include_* options of an import
func importFlags(c *gin.Context) (data, attachments, automations, systemColumns bool, err error) {
	if data, err = formBool(c, "include_data", true); err != nil {
		return
	}
	if attachments, err = formBool(c, "include_attachments", true); err != nil {
		return
	}
	if automations, err = ImportAutomationsRequested(c); err != nil {
		return
	}
	if systemColumns, err = formBool(c, "include_system_columns", false); err != nil {
		return
	}
	// source columns are values of the rows, so they need the data
	systemColumns = systemColumns && data
	return
}

// ImportBase creates a new base in a workspace from an uploaded .sbase.zip file, from a zip of CSV files, or from an
// .xlsx workbook.
// POST /workspace/:id/import-base  (multipart: file, title, format, include_data, include_attachments,
// include_system_columns, include_automations; include_automations may also be a URL parameter)
// format is "sbase" (default) or "csv": a zip with one .csv file per table (the file an export with
// format=csv writes), whose first row holds the field titles. The delimiter of each file is found from its header,
// the types are inferred from the values, and the paths of files in attachments/ become attachment fields. The new
// base is named after the file unless title is given. Triggers and system columns are not imported from a CSV.
// With format=xlsx the file is what an export with format=xlsx writes: an .xlsx workbook, or a .zip of one workbook and
// an attachments/ folder. One table is made per visible sheet with data, named after the sheet, the first row holding
// the field titles; the types are inferred from the values, a column of paths of files in attachments/ becomes an
// attachment field, and a formula is read as the value it last calculated. Any other file is refused with 400
// "only .xlsx or .zip files are supported". Triggers and system columns are not imported from a workbook.
// A broken file is refused at once (400). Otherwise the import runs as a background job: the response is 202 with the
// job (poll GET /import-job/:id until it is completed or failed).
// A user can have two imports active at once; a third is refused with 429.
func (h *BaseTransferHandler) ImportBase(c *gin.Context) {
	workspaceID := strings.TrimSpace(c.Param("id"))
	if workspaceID == "" {
		response.SendError(c, responseConst.WorkspaceError.IdRequired)
		return
	}
	upload, err := c.FormFile("file")
	if err != nil {
		response.SendError(c, responseConst.AssetError.FilesNotFound)
		return
	}
	// the format of the upload: a SereniBase export (default), a zip of CSV files, or an Excel workbook
	importFormat := strings.ToLower(strings.TrimSpace(c.PostForm("format")))
	switch importFormat {
	case "", sbase.Name, csvformat.Name, xlsx.Name, jsonfmt.Name:
	default:
		response.SendErrorWithMessage(c, responseConst.Error.InvalidPayload, "format must be sbase, csv, xlsx or json")
		return
	}
	if problem := importFileProblem(importFormat, upload.Filename); problem != "" {
		response.SendErrorWithMessage(c, responseConst.Error.InvalidPayload, problem)
		return
	}
	if upload.Size > maxBaseImportBytes {
		response.SendError(c, responseConst.AssetError.FileTooLargeError)
		return
	}
	includeData, includeAttachments, includeAutomations, includeSystemColumns, err := importFlags(c)
	if err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	// A CSV or a workbook has no triggers and no system columns, so those are not imported from it, and a workbook
	// has no files.
	switch importFormat {
	case "", sbase.Name:
		importFormat = sbase.Name
	case csvformat.Name, xlsx.Name, jsonfmt.Name:
		includeAutomations = false
		includeSystemColumns = false
	}

	// the adapter reads the file into a model; it stays open until the import is done, because the files of a zip
	// are read from it while the base is created
	upfile, err := upload.Open()
	if err != nil {
		response.SendErrorWithMessage(c, responseConst.Error.InvalidPayload, "the file could not be read")
		return
	}
	defer upfile.Close()
	read, err := h.adapterFor(importFormat).Import(c.Request.Context(), upfile, upload.Size, adapter.ImportOptions{
		Filename: upload.Filename,
		MaxBytes: maxBaseImportBytes,
	})
	if err != nil {
		response.SendErrorWithMessage(c, responseConst.Error.InvalidPayload, err.Error())
		return
	}

	schemaName, _ := c.Get("schema")
	schema, _ := schemaName.(string)
	userID, _ := c.Get("user_id")
	user, _ := userID.(string)

	title := strings.TrimSpace(c.PostForm("title"))

	// An import can take longer than the server's write timeout, so it runs as a background job: the file is kept and
	// the response is 202 with the job to follow (GET /import-job/:id). The file was read above only to refuse a broken
	// one at once; the job reads it again.
	if h.jobs != nil {
		if _, err := upfile.Seek(0, io.SeekStart); err != nil {
			response.SendErrorWithMessage(c, responseConst.Error.InvalidPayload, "the file could not be read")
			return
		}
		job, err := h.jobs.StartImport(c.Request.Context(), schema, workspaceID, user, dto.ImportUpload{File: upfile, Filename: upload.Filename, Size: upload.Size}, dto.ImportBaseOptions{
			Title:                title,
			IncludeData:          includeData,
			IncludeAttachments:   includeAttachments,
			IncludeAutomations:   includeAutomations,
			IncludeSystemColumns: includeSystemColumns,
			Format:               importFormat,
			SkipFailedRows:       importFormat != sbase.Name,
		})
		if errors.Is(err, app_errors.TransferLimitReached) {
			response.SendError(c, responseConst.BaseError.TransferLimitReached)
			return
		}
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}
		response.SendSuccess(c, responseConst.BaseSuccess.ImportStarted, job)
		return
	}

	result, err := h.transferService.ImportBase(c.Request.Context(), schema, workspaceID, user, read.File, dto.ImportBaseOptions{
		Title:                title,
		IncludeData:          includeData,
		Attachments:          read.Attachments,
		IncludeAttachments:   includeAttachments,
		IncludeAutomations:   includeAutomations,
		IncludeSystemColumns: includeSystemColumns,
		Format:               importFormat,
		SkipFailedRows:       importFormat != sbase.Name,
		Source:               read.Source,
		FileName:             upload.Filename,
	})
	var invalid *dto.ImportValidationError
	if errors.As(err, &invalid) {
		response.SendErrorWithData(c, responseConst.Error.ValidationFailed, "the base file is not valid", invalid.Problems)
		return
	}
	if errors.Is(err, app_errors.BaseNameExists) {
		response.SendError(c, responseConst.BaseError.BaseAlreadyExists)
		return
	}
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, "base imported successfully", result)
}

// ImportAutomationsRequested reports whether an import request asks for the trigger functions and triggers of the file.
// The option can be a form field or a URL parameter (the form field wins). The router's permission check and the handler
// both read it through this function, so the check always covers exactly what the handler then does.
func ImportAutomationsRequested(c *gin.Context) (bool, error) {
	raw := c.PostForm("include_automations")
	if strings.TrimSpace(raw) == "" {
		raw = c.Query("include_automations")
	}
	return parseBool(raw, false)
}

func queryBool(c *gin.Context, name string, fallback bool) (bool, error) {
	return parseBool(c.Query(name), fallback)
}

func queryTables(c *gin.Context) []string {
	var raw []string
	for _, s := range append(c.QueryArray("tables"), c.QueryArray("table_ids")...) {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				raw = append(raw, part)
			}
		}
	}
	var result []string
	seen := map[string]bool{}
	for _, t := range raw {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

func formBool(c *gin.Context, name string, fallback bool) (bool, error) {
	return parseBool(c.PostForm(name), fallback)
}

func parseBool(raw string, fallback bool) (bool, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	return strconv.ParseBool(strings.TrimSpace(raw))
}

// GetExportJob returns the state of a background export of the current user.
// GET /export-job/:id
func (h *BaseTransferHandler) GetExportJob(c *gin.Context) {
	h.getJob(c, "export", responseConst.BaseSuccess.ExportStatus)
}

// GetImportJob returns the state of a background import of the current user. When it is completed, base_id is the
// new base and result says what was created; when it failed, error says why (and result lists the problems of a file
// that is not valid).
// GET /import-job/:id
func (h *BaseTransferHandler) GetImportJob(c *gin.Context) {
	h.getJob(c, "import", responseConst.BaseSuccess.ImportStatus)
}

// RemoveExportJob deletes a finished export (and its file) from the jobs list.
// DELETE /export-job/:id
func (h *BaseTransferHandler) RemoveExportJob(c *gin.Context) {
	h.removeJob(c, "export", responseConst.BaseSuccess.ExportStatus)
}

// RemoveImportJob deletes a finished import (and its log) from the jobs list.
// DELETE /import-job/:id
func (h *BaseTransferHandler) RemoveImportJob(c *gin.Context) {
	h.removeJob(c, "import", responseConst.BaseSuccess.ImportStatus)
}

func (h *BaseTransferHandler) removeJob(c *gin.Context, jobType string, success responseConst.ResponseCode) {
	id, user, schema, ok := h.transferJobRequest(c)
	if !ok {
		return
	}
	job, err := h.jobs.Get(c.Request.Context(), schema, id, user)
	if err == nil && job.Type != jobType {
		err = app_errors.TransferJobNotFound
	}
	if err == nil {
		err = h.jobs.Remove(c.Request.Context(), schema, id, user)
	}
	if err != nil {
		h.sendTransferJobError(c, err)
		return
	}
	response.SendSuccess(c, success, job)
}

// ListWorkspaceJobs returns the current user's exports and imports of a workspace with their status and progress, newest
// first, so the app can show them all in one place. Other users' jobs are never listed.
// GET /workspace/:id/transfer-jobs
func (h *BaseTransferHandler) ListWorkspaceJobs(c *gin.Context) {
	if h.jobs == nil {
		response.SendSuccess(c, responseConst.BaseSuccess.ExportStatus, []dto.TransferJobResponse{})
		return
	}
	workspaceID := strings.TrimSpace(c.Param("id"))
	if workspaceID == "" {
		response.SendError(c, responseConst.WorkspaceError.IdRequired)
		return
	}
	userID, _ := c.Get("user_id")
	user, _ := userID.(string)
	schemaName, _ := c.Get("schema")
	schema, _ := schemaName.(string)
	jobs, err := h.jobs.ListWorkspaceJobs(c.Request.Context(), schema, workspaceID, user)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, responseConst.BaseSuccess.ExportStatus, jobs)
}

// getJob answers with a job of the given type. A job of the other type looks like a job that does not exist.
func (h *BaseTransferHandler) getJob(c *gin.Context, jobType string, success responseConst.ResponseCode) {
	id, user, schema, ok := h.transferJobRequest(c)
	if !ok {
		return
	}
	job, err := h.jobs.Get(c.Request.Context(), schema, id, user)
	if err == nil && job.Type != jobType {
		err = app_errors.TransferJobNotFound
	}
	if err != nil {
		h.sendTransferJobError(c, err)
		return
	}
	response.SendSuccess(c, success, job)
}

// importLogUnsafe is what a file name cannot hold
var importLogUnsafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// importLogFileName is the name of the error log of an import: after the new base when it is known
func importLogFileName(baseTitle string) string {
	name := strings.Trim(importLogUnsafe.ReplaceAllString(baseTitle, "_"), "_")
	if name == "" {
		return "import_error_rows_report.txt"
	}
	return name + "_import_error_rows_report.txt"
}

// GetImportJobLog downloads the error log of an import of the current user as a text file: the rows that were left
// out and the warnings of a finished import, or what went wrong in a failed one. A job that has no log answers 404.
// GET /import-job/:id/log
func (h *BaseTransferHandler) GetImportJobLog(c *gin.Context) {
	id, user, schema, ok := h.transferJobRequest(c)
	if !ok {
		return
	}
	job, err := h.jobs.Get(c.Request.Context(), schema, id, user)
	if err == nil && job.Type != "import" {
		err = app_errors.TransferJobNotFound
	}
	if err != nil {
		h.sendTransferJobError(c, err)
		return
	}
	var result struct {
		ErrorLog string `json:"error_log"`
	}
	if len(job.Result) == 0 || json.Unmarshal(job.Result, &result) != nil || strings.TrimSpace(result.ErrorLog) == "" {
		response.SendError(c, responseConst.BaseError.TransferJobNotFound)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, importLogFileName(job.BaseTitle)))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(result.ErrorLog))
}

// DownloadExportJob downloads the zip of a completed export of the current user.
// GET /export-job/:id/download
func (h *BaseTransferHandler) DownloadExportJob(c *gin.Context) {
	id, user, schema, ok := h.transferJobRequest(c)
	if !ok {
		return
	}
	f, job, err := h.jobs.Open(c.Request.Context(), schema, id, user)
	if err != nil {
		h.sendTransferJobError(c, err)
		return
	}
	defer f.Close()

	// A large zip can take longer to send than the server's write timeout; this response is exempt from it
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{})

	name := job.FileName
	if name == "" {
		name = "base.sbase.zip"
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	contentType := "application/zip"
	if strings.HasSuffix(strings.ToLower(name), xlsx.Extension) {
		contentType = xlsx.ContentType
	}
	c.DataFromReader(http.StatusOK, job.SizeBytes, contentType, f, nil)
}

// transferJobRequest reads the job id and the user from a request; it answers the request itself when they are not usable.
func (h *BaseTransferHandler) transferJobRequest(c *gin.Context) (id, user, schema string, ok bool) {
	if h.jobs == nil {
		response.SendError(c, responseConst.BaseError.TransferJobNotFound)
		return "", "", "", false
	}
	id = strings.TrimSpace(c.Param("id"))
	if _, err := uuid.Parse(id); err != nil {
		response.SendError(c, responseConst.BaseError.TransferJobNotFound)
		return "", "", "", false
	}
	userID, _ := c.Get("user_id")
	user, _ = userID.(string)
	schemaName, _ := c.Get("schema")
	schema, _ = schemaName.(string)
	return id, user, schema, true
}

func (h *BaseTransferHandler) sendTransferJobError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, app_errors.TransferJobNotFound):
		response.SendError(c, responseConst.BaseError.TransferJobNotFound)
	case errors.Is(err, app_errors.TransferJobNotReady):
		response.SendError(c, responseConst.BaseError.TransferJobNotReady)
	case errors.Is(err, app_errors.TransferJobExpired):
		response.SendError(c, responseConst.BaseError.TransferJobGone)
	default:
		response.CheckAndSendError(c, err)
	}
}
