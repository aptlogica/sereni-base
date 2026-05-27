// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"mime/multipart"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	antivirusProviderInterface "github.com/aptlogica/sereni-base/internal/providers/antivirus/interfaces"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/rs/zerolog"
)

const (
	isoDateFormat             = "2006-01-02"
	ddmmyyyyDateFormat        = "02-01-2006"
	yyyymmddSlashFormat       = "2006/01/02"
	ddmmyyyySlashFormat       = "02/01/2006"
	errWorkspaceIDRequiredMsg = "workspace_id is required when base_id is not provided"
	descAutoCreatedBase       = "Auto-created base for table import"
	errFailedCreateBase       = "Failed to create base for import"
	colNameFmt                = "\"%s\""
)

type importService struct {
	tableService          interfaces.TableManagementService
	baseManagementService interfaces.BaseManagementService
	antivirusProvider     antivirusProviderInterface.Provider
}

func NewImportService(tableService interfaces.TableManagementService, baseManagementService interfaces.BaseManagementService, antivirusProvider antivirusProviderInterface.Provider) interfaces.ImportService {
	return &importService{
		tableService:          tableService,
		baseManagementService: baseManagementService,
		antivirusProvider:     antivirusProvider,
	}
}

// ImportWithConfig imports a table with user-provided column configuration and data cleaning settings
func (s *importService) ImportWithConfig(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string) (dto.ImportTableResponse, error) {
	lg := logger.Get()

	if err := s.ScanFile(ctx, file, lg); err != nil {
		return dto.ImportTableResponse{}, err
	}
	// Track whether we auto-created a base/table so we can clean up on partial failures
	createdBase := false
	if req.BaseID == "" {
		if err := s.EnsureBaseWithConfig(ctx, schemaName, &req, lg, tableTitle); err != nil {
			return dto.ImportTableResponse{}, err
		}
		createdBase = true
	}
	headers, dataRows, stats, uniqueTableTitle, err := s.PrepareImportData(ctx, schemaName, req, file, tableTitle, lg)
	if err != nil {
		return dto.ImportTableResponse{}, err
	}

	// Create table (and obtain cleanup function)
	createTableReq := dto.CreateTableRequest{
		BaseID:      req.BaseID,
		WorkspaceID: req.WorkspaceID,
		Title:       uniqueTableTitle,
		Description: req.Description,
		OrderIndex:  req.OrderIndex,
		CreatedBy:   req.CreatedBy,
	}

	tableResp, cleanup, err := s.CreateTableForImport(ctx, schemaName, createTableReq, lg, createdBase, req.BaseID)
	if err != nil {
		return dto.ImportTableResponse{}, err
	}

	// Add columns with config
	columnMap, err := s.AddColumnsWithConfig(dto.AddColumnsWithConfigParams{
		Ctx:           ctx,
		SchemaName:    schemaName,
		Req:           createTableReq,
		Headers:       headers,
		ColumnConfigs: req.Config.Columns,
		Primary:       req.Config.PrimaryColumn,
		TableResp:     tableResp,
	}, lg)
	if err != nil {
		cleanup()
		return dto.ImportTableResponse{}, err
	}

	// Build records using config column types with error tracking
	newRecords, errorRows, errorMessages := s.BuildRecordsWithConfigAndErrors(dto.BuildRecordsWithConfigAndErrorsParams{
		DataRows:      dataRows,
		ColumnConfigs: req.Config.Columns,
		Primary:       req.Config.PrimaryColumn,
		ColumnMap:     columnMap,
		Req:           createTableReq,
		Headers:       headers,
		Settings:      req.Config.Settings,
	}, lg)
	lg.Info().Int("recordsCreated", len(newRecords)).Int("errorRows", len(errorRows)).Msg("Records prepared for insertion with config and error tracking")

	// Always identify empty and duplicate rows for logging (regardless of settings)
	emptyRowsWithLineNumbers := s.IdentifyEmptyRowsWithLineNumbers(dataRows)
	duplicateRowsWithLineNumbers := s.IdentifyDuplicateRowsWithLineNumbers(dataRows)

	// Always generate error report with import details
	var errorRowsFileContent string
	errorRowsFileContent, err = s.SaveErrorRows(headers, errorRows, errorMessages, emptyRowsWithLineNumbers, duplicateRowsWithLineNumbers, lg)
	if err != nil {
		lg.Warn().Err(err).Msg("Failed to generate error report, continuing with import")
		// Don't fail the import if we can't generate error report, just log it
	}

	stats.TotalRows = len(newRecords)
	stats.TotalColumns = len(req.Config.Columns)
	stats.ErrorRows = len(errorRows)
	stats.ErrorRowsFileContent = errorRowsFileContent

	return s.FinalizeImport(finalizeImportOptions{
		Ctx:           ctx,
		SchemaName:    schemaName,
		TableResp:     tableResp,
		NewRecords:    newRecords,
		Headers:       headers,
		Stats:         stats,
		ErrorRows:     errorRows,
		ErrorMessages: errorMessages,
		LG:            lg,
	})
}

// FinalizeImport handles batch insertion, aggregates DB errors into statistics and
// refreshes the table before returning a final response. It mirrors the original
// inline logic to preserve behavior exactly.
// finalizeImportOptions packages arguments for FinalizeImport helper
type finalizeImportOptions struct {
	Ctx           context.Context
	SchemaName    string
	TableResp     dto.TableResponse
	NewRecords    []map[string]interface{}
	Headers       []string
	Stats         *dto.ImportStatistics
	ErrorRows     [][]string
	ErrorMessages []string
	LG            *zerolog.Logger
}

// FinalizeImport handles batch insertion, aggregates DB errors into statistics and
// refreshes the table before returning a final response. It mirrors the original
// inline logic to preserve behavior exactly.
func (s *importService) FinalizeImport(opts finalizeImportOptions) (dto.ImportTableResponse, error) {
	// Insert batches with database error handling - skip failed rows and continue
	dbErrorRows, dbErrorMessages := s.InsertBatchesWithErrorHandling(opts.Ctx, opts.SchemaName, opts.TableResp, opts.NewRecords, opts.Headers, opts.LG)

	// Add database errors to statistics and error rows
	if len(dbErrorRows) > 0 {
		opts.Stats.ErrorRows += len(dbErrorRows)

		// Append database errors to error report
		if opts.Stats.ErrorRowsFileContent != "" {
			opts.Stats.ErrorRowsFileContent += "\n\n" +
				strings.Repeat("=", 100) + "\n" +
				"DATABASE ERRORS (Failed to insert into database)\n" +
				strings.Repeat("=", 100) + "\n\n"
		}

		for _, dbErrMsg := range dbErrorMessages {
			opts.Stats.ErrorRowsFileContent += dbErrMsg + "\n\n"
		}

		opts.LG.Warn().Int("dbErrorRowCount", len(dbErrorRows)).Msg("Some rows failed due to database errors - import continued with remaining rows")
	}

	finalTableResp, err := s.RefreshTable(opts.Ctx, opts.SchemaName, opts.TableResp, opts.LG)
	if err != nil {
		return dto.ImportTableResponse{ImportStats: opts.Stats, TableModelViewResponse: dto.TableModelViewResponse{
			Model: opts.TableResp.Model,
			Views: opts.TableResp.Views,
		}}, nil
	}

	return dto.ImportTableResponse{ImportStats: opts.Stats, TableModelViewResponse: dto.TableModelViewResponse{
		Model: finalTableResp.Model,
		Views: finalTableResp.Views,
	}}, nil
}

func (s *importService) ScanFile(ctx context.Context, file *multipart.FileHeader, lg *zerolog.Logger) error {
	if s.antivirusProvider == nil {
		return nil
	}

	f, err := file.Open()
	if err != nil {
		lg.Error().Stack().Err(err).Str("file", file.Filename).Msg("Failed to open CSV file for antivirus scan")
		return err
	}
	scanResult, scanErr := s.antivirusProvider.ScanReader(ctx, file.Filename, f)
	f.Close()
	if scanErr != nil {
		lg.Info().Str("scanErr: ", scanErr.Error())
		lg.Error().Stack().Err(scanErr).Str("file", file.Filename).Str("threat", scanResult.Threat).Msg("Antivirus scan detected threat")
		return fmt.Errorf("file '%s' is infected or contains malicious content", file.Filename)
	}

	lg.Info().Str("file", file.Filename).Msg("Antivirus scan passed")
	return nil
}

func (s *importService) EnsureBase(ctx context.Context, schemaName string, req *dto.CreateTableRequest, lg *zerolog.Logger, tableTitle string) error {
	if req.BaseID != "" {
		return nil
	}

	if req.WorkspaceID == "" {
		lg.Error().Msg(errWorkspaceIDRequiredMsg)
		return fmt.Errorf("%s", errWorkspaceIDRequiredMsg)
	}

	baseName := tableTitle + "_base"
	lg.Info().Str("baseName", baseName).Str("workspaceID", req.WorkspaceID).Msg("Creating new base for import")

	createBaseReq := dto.CreateBaseRequest{
		Title:       baseName,
		Description: helpers.StringPtr(descAutoCreatedBase),
		WorkspaceID: req.WorkspaceID,
		CreatedBy:   req.CreatedBy,
	}

	newBase, err := s.baseManagementService.CreateBaseWithoutTable(ctx, createBaseReq, schemaName, req.CreatedBy)
	if err != nil {
		lg.Error().Stack().Err(err).Str("baseName", baseName).Msg(errFailedCreateBase)
		return fmt.Errorf("failed to create base: %w", err)
	}

	req.BaseID = newBase.ID.String()
	lg.Info().Str("baseID", req.BaseID).Str("baseName", baseName).Msg("Base created successfully for import")
	return nil
}

func (s *importService) ParseCSV(file *multipart.FileHeader, lg *zerolog.Logger) ([]string, [][]string, error) {
	f, err := file.Open()
	if err != nil {
		lg.Error().Stack().Err(err).Str("file", file.Filename).Msg("Failed to open CSV file")
		return nil, nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		lg.Error().Stack().Err(err).Str("file", file.Filename).Msg("Failed to parse CSV file")
		return nil, nil, err
	}

	if len(records) < 1 {
		errMsg := "empty csv file"
		lg.Error().Str("file", file.Filename).Msg(errMsg)
		return nil, nil, fmt.Errorf("%s", errMsg)
	}

	lg.Info().Str("file", file.Filename).Int("rows", len(records)).Msg("CSV file parsed successfully")

	headers := records[0]
	// Strip BOM (Byte Order Mark) from the first header if present
	if len(headers) > 0 && len(headers[0]) > 0 {
		headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
		lg.Debug().Str("firstHeader", headers[0]).Msg("Stripped BOM from first CSV header if present")
	}
	dataRows := records[1:]
	return headers, dataRows, nil
}

func (s *importService) CreateTable(ctx context.Context, schemaName string, req dto.CreateTableRequest, lg *zerolog.Logger, tableTitle string) (dto.TableResponse, error) {
	lg.Info().Str("tableName", tableTitle).Str("schemaName", schemaName).Msg("Creating table with defaults")
	tableResp, err := s.tableService.CreateTableWithDefaults(ctx, req, schemaName)
	if err != nil {
		lg.Error().Stack().Err(err).Str("tableName", tableTitle).Str("schemaName", schemaName).Msg("Failed to create table with defaults")
		return dto.TableResponse{}, err
	}
	lg.Info().Str("tableID", tableResp.Model.ID.String()).Msg("Table created successfully")
	return tableResp, nil
}

// CreateTableForImport wraps table creation and returns a cleanup function that will
// remove created resources in case of later failures. It mirrors original behavior
// to ensure semantics remain identical.
func (s *importService) CreateTableForImport(ctx context.Context, schemaName string, createTableReq dto.CreateTableRequest, lg *zerolog.Logger, createdBase bool, baseID string) (dto.TableResponse, func(), error) {
	tableResp, err := s.CreateTable(ctx, schemaName, createTableReq, lg, createTableReq.Title)
	if err != nil {
		s.CleanupBaseIfNeeded(ctx, schemaName, baseID, createdBase, lg)
		return dto.TableResponse{}, nil, err
	}

	createdTable := true

	cleanup := func() {
		if createdTable {
			if delErr := s.tableService.DeleteTable(ctx, schemaName, tableResp.Model.ID.String()); delErr != nil {
				lg.Error().Stack().Err(delErr).Str("tableID", tableResp.Model.ID.String()).Msg("Failed to cleanup created table after import error")
			}
		}
		if createdBase {
			if delErr := s.baseManagementService.DeleteBase(ctx, schemaName, baseID); delErr != nil {
				lg.Error().Stack().Err(delErr).Str("baseID", baseID).Msg("Failed to cleanup auto-created base after import error")
			}
		}
	}

	return tableResp, cleanup, nil
}

// CleanupBaseIfNeeded deletes the auto-created base if it was created and logs errors
func (s *importService) CleanupBaseIfNeeded(ctx context.Context, schemaName string, baseID string, createdBase bool, lg *zerolog.Logger) {
	if createdBase {
		if delErr := s.baseManagementService.DeleteBase(ctx, schemaName, baseID); delErr != nil {
			lg.Error().Stack().Err(delErr).Str("baseID", baseID).Msg("Failed to cleanup auto-created base after table creation failure")
		}
	}
}

func (s *importService) InsertBatchesWithErrorHandling(ctx context.Context, schemaName string, tableResp dto.TableResponse, newRecords []map[string]interface{}, headers []string, lg *zerolog.Logger) ([][]string, []string) {
	batchSize := 50
	totalBatches := (len(newRecords) + batchSize - 1) / batchSize
	failedRows := [][]string{}
	errorMessages := []string{}
	lg.Info().Int("batchSize", batchSize).Int("totalBatches", totalBatches).Msg("Starting batch insertion with database error handling")

	for i := 0; i < len(newRecords); i += batchSize {
		end := i + batchSize
		if end > len(newRecords) {
			end = len(newRecords)
		}
		batch := newRecords[i:end]
		batchNum := (i / batchSize) + 1
		startRowIndex := i

		lg.Info().Int("batchNumber", batchNum).Int("batchSize", len(batch)).Msg("Inserting batch")
		if _, err := s.tableService.CreateRowsWithRecordsBulk(ctx, schemaName, tableResp.Model.Alias, batch); err != nil {
			lg.Warn().Stack().Err(err).Int("batchNumber", batchNum).Int("batchSize", len(batch)).Msg("Batch insertion failed - testing rows individually to identify problematic rows")

			// Try inserting each row individually to find which ones actually fail
			newFailedRows, newErrorMessages := s.TestInsertRowsIndividually(batchInsertOptions{
				Ctx:           ctx,
				SchemaName:    schemaName,
				TableResp:     tableResp,
				Batch:         batch,
				StartRowIndex: startRowIndex,
				Headers:       headers,
				BatchNum:      batchNum,
				FailedOffset:  len(failedRows),
				LG:            lg,
			})
			if len(newFailedRows) > 0 {
				failedRows = append(failedRows, newFailedRows...)
			}
			if len(newErrorMessages) > 0 {
				errorMessages = append(errorMessages, newErrorMessages...)
			}
			// Continue to next batch after processing individual rows
			continue
		}
		lg.Debug().Int("batchNumber", batchNum).Msg("Batch inserted successfully")
	}

	if len(failedRows) > 0 {
		lg.Warn().Int("failedRowCount", len(failedRows)).Int("totalRecords", len(newRecords)).Msg("Batch insertion completed with some database errors - import continued successfully")
	} else {
		lg.Info().Int("totalBatches", totalBatches).Int("totalRecords", len(newRecords)).Msg("All batches inserted successfully")
	}

	return failedRows, errorMessages
}

// batchInsertOptions packages parameters for per-row retry helper to keep parameter
// counts low and the callsite readable.
type batchInsertOptions struct {
	Ctx           context.Context
	SchemaName    string
	TableResp     dto.TableResponse
	Batch         []map[string]interface{}
	StartRowIndex int
	Headers       []string
	BatchNum      int
	FailedOffset  int
	LG            *zerolog.Logger
}

// TestInsertRowsIndividually tries to insert each row in the provided batch independently
// and returns failed row placeholders and corresponding error messages. `FailedOffset`
// should be the count of previously collected failed rows so that error numbering stays
// consistent with the original implementation.
func (s *importService) TestInsertRowsIndividually(opts batchInsertOptions) ([][]string, []string) {
	newFailedRows := [][]string{}
	newErrorMessages := []string{}

	for j := 0; j < len(opts.Batch); j++ {
		rowIndex := opts.StartRowIndex + j
		lineNumber := rowIndex + 2 // Line numbers start from 2 (row 1 is header)
		singleRowBatch := []map[string]interface{}{opts.Batch[j]}

		if _, err := s.tableService.CreateRowsWithRecordsBulk(opts.Ctx, opts.SchemaName, opts.TableResp.Model.Alias, singleRowBatch); err != nil {
			// This specific row failed
			opts.LG.Error().Stack().Err(err).Int("batchNumber", opts.BatchNum).Int("lineNumber", lineNumber).Int("rowIndex", rowIndex).Msg("Row failed - skipping")

			// Create error message with specific row line number and consistent numbering
			failureMsg := fmt.Sprintf("[Database Error %d] Batch %d, Row Line %d (CSV Line %d)\n", opts.FailedOffset+len(newFailedRows)+1, opts.BatchNum, lineNumber, lineNumber)
			failureMsg += fmt.Sprintf("Error: %v\n", err)
			failureMsg += fmt.Sprintf("Record: %v\n", opts.Batch[j])

			newErrorMessages = append(newErrorMessages, failureMsg)

			// Create a representation of the failed row
			failedRowData := make([]string, len(opts.Headers))
			for k := range failedRowData {
				failedRowData[k] = "[FAILED_TO_INSERT]"
			}
			newFailedRows = append(newFailedRows, failedRowData)
		} else {
			opts.LG.Debug().Int("batchNumber", opts.BatchNum).Int("lineNumber", lineNumber).Msg("Row inserted successfully")
		}
	}

	return newFailedRows, newErrorMessages
}

func (s *importService) RefreshTable(ctx context.Context, schemaName string, tableResp dto.TableResponse, lg *zerolog.Logger) (dto.TableResponse, error) {
	lg.Info().Str("tableID", tableResp.Model.ID.String()).Msg("Refreshing table response")
	finalTableResp, err := s.tableService.GetTableByID(ctx, tableResp.Model.ID.String(), schemaName)
	if err != nil {
		lg.Warn().Stack().Err(err).Str("tableID", tableResp.Model.ID.String()).Msg("Failed to refresh table response, returning cached response")
		return tableResp, err
	}

	lg.Info().Str("tableID", finalTableResp.Model.ID.String()).Int("columns", len(finalTableResp.Columns)).Int("records", len(finalTableResp.Records)).Msg("Import completed successfully")
	return finalTableResp, nil
}

func (s *importService) InferColumnTypes(headers []string, rows [][]string) []string {
	types := make([]string, len(headers))
	for i := range headers {
		types[i] = s.InferType(rows, i)
	}
	return types
}

func (s *importService) InferType(rows [][]string, colIndex int) string {
	flags := s.CollectTypeFlags(rows, colIndex)
	if !flags.hasData {
		return "text"
	}
	return s.DetermineTypeFromFlags(flags)
}

type typeFlags struct {
	isNumber, isDecimal, isBool, isDate, isEmail, isURL, isPhone, isJSON bool
	hasData                                                              bool
	totalLength, count                                                   int
}

func (s *importService) CollectTypeFlags(rows [][]string, colIndex int) typeFlags {
	flags := typeFlags{
		isNumber:  true,
		isDecimal: true,
		isBool:    true,
		isDate:    true,
		isEmail:   true,
		isURL:     true,
		isPhone:   true,
		isJSON:    true,
	}
	for _, row := range rows {
		if colIndex >= len(row) {
			continue
		}
		val := row[colIndex]
		if val == "" {
			continue
		}
		flags.hasData = true
		flags.totalLength += len(val)
		flags.count++
		s.UpdateTypeFlags(&flags, val)
	}
	return flags
}

// UpdateTypeFlags updates the type flags based on a single value
func (s *importService) UpdateTypeFlags(flags *typeFlags, val string) {
	if flags.isNumber || flags.isDecimal {
		flags.isNumber, flags.isDecimal = s.CheckNumericTypes(val, flags.isNumber, flags.isDecimal)
	}
	if flags.isBool {
		flags.isBool = s.CheckBoolType(val)
	}
	if flags.isDate {
		flags.isDate = s.CheckDateType(val)
	}
	if flags.isEmail {
		flags.isEmail = s.CheckEmailType(val)
	}
	if flags.isURL {
		flags.isURL = s.CheckURLType(val)
	}
	if flags.isPhone {
		flags.isPhone = s.CheckPhoneType(val)
	}
	if flags.isJSON {
		flags.isJSON = s.CheckJSONType(val)
	}
}

func (s *importService) CheckNumericTypes(val string, isNumber, isDecimal bool) (bool, bool) {
	if v, err := strconv.ParseInt(val, 10, 64); err != nil {
		isNumber = false
	} else {
		if v > math.MaxInt32 || v < math.MinInt32 {
			isNumber = false
		}
	}
	if _, err := strconv.ParseFloat(val, 64); err != nil {
		isDecimal = false
	}
	return isNumber, isDecimal
}

func (s *importService) CheckBoolType(val string) bool {
	lower := strings.ToLower(val)
	return lower == "true" || lower == "false" || lower == "0" || lower == "1" || lower == "yes" || lower == "no"
}

func (s *importService) CheckDateType(val string) bool {
	formats := []string{isoDateFormat, ddmmyyyyDateFormat, yyyymmddSlashFormat, ddmmyyyySlashFormat}
	for _, f := range formats {
		if _, err := time.Parse(f, val); err == nil {
			return true
		}
	}
	return false
}

func (s *importService) CheckEmailType(val string) bool {
	return strings.Contains(val, "@") && strings.Contains(val, ".")
}

func (s *importService) CheckURLType(val string) bool {
	return strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://")
}

func (s *importService) CheckPhoneType(val string) bool {
	// Simple check: contains only digits, spaces, dashes, parentheses, plus
	for _, r := range val {
		if !((r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '(' || r == ')' || r == '+') {
			return false
		}
	}
	return len(val) > 0
}

func (s *importService) CheckJSONType(val string) bool {
	var js interface{}
	return json.Unmarshal([]byte(val), &js) == nil
}

func (s *importService) DetermineTypeFromFlags(flags typeFlags) string {
	avgLength := 0
	if flags.count > 0 {
		avgLength = flags.totalLength / flags.count
	}
	if flags.isBool {
		return "boolean"
	}
	if flags.isNumber {
		return "number"
	}
	if flags.isDecimal {
		return "decimal"
	}
	if flags.isDate {
		return "date"
	}
	if flags.isEmail {
		return "email"
	}
	if flags.isURL {
		return "url"
	}
	if flags.isPhone {
		return "phoneNumber"
	}
	if flags.isJSON {
		return "json"
	}
	if avgLength > 255 {
		return "longText"
	}
	return "text"
}

func (s *importService) GetDatabaseType(uidt string) string {
	if mapping, exists := constant.UITypeMappings[uidt]; exists {
		return mapping.Postgres
	}
	// Default to TEXT
	return "TEXT"
}

func (s *importService) ConvertValue(val string, typeName string) interface{} {
	switch typeName {
	case "number":
		if v, err := strconv.ParseInt(val, 10, 64); err == nil {
			return v
		}
		// fallback to float if somehow an int wasn't parsed, but float would work
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			return v
		}
	case "decimal":
		if v, err := strconv.ParseFloat(val, 64); err == nil {
			return v
		}
	case "boolean":
		lower := strings.ToLower(val)
		if lower == "true" || lower == "1" || lower == "yes" {
			return true
		}
		return false
	case "date":
		// Convert date string to PostgreSQL-compatible format (YYYY-MM-DD)
		return s.ConvertDateToISO(val)
	case "email", "url", "phoneNumber", "json":
		// These are stored as text
		return val
	}
	return val
}

// ConvertDateToISO converts date strings from various formats to ISO format (YYYY-MM-DD)
func (s *importService) ConvertDateToISO(val string) string {
	formats := []string{
		isoDateFormat,
		ddmmyyyyDateFormat,
		yyyymmddSlashFormat,
		ddmmyyyySlashFormat,
	}

	for _, format := range formats {
		if parsedTime, err := time.Parse(format, val); err == nil {
			// Convert to ISO format (YYYY-MM-DD)
			return parsedTime.Format(isoDateFormat)
		}
	}
	return val
}

// FindUniqueName finds a unique name from a list of existing names by appending numbers if needed
// Enforces the specified character limit by truncating the base name
func (s *importService) FindUniqueName(proposedName string, existingNames []string, maxLength int) string {
	// Build a map of existing names for O(1) lookup
	existingNamesMap := make(map[string]bool)
	for _, name := range existingNames {
		existingNamesMap[name] = true
	}

	// Truncate to max length if necessary
	cleanName := proposedName
	if len(cleanName) > maxLength {
		cleanName = cleanName[:maxLength]
	}

	// Check if the clean name already exists
	if !existingNamesMap[cleanName] {
		return cleanName
	}

	// Name exists, find a unique one by appending numbers
	counter := 1
	for {
		counterStr := fmt.Sprintf(" %d", counter)
		availableLength := maxLength - len(counterStr)
		if availableLength < 1 {
			// Shouldn't happen with reasonable max lengths, return fallback
			return cleanName
		}

		// Truncate base name if necessary to fit the counter
		baseName := cleanName
		if len(baseName) > availableLength {
			baseName = baseName[:availableLength]
		}

		uniqueName := baseName + counterStr

		// Ensure we don't exceed max length
		if len(uniqueName) > maxLength {
			uniqueName = uniqueName[:maxLength]
		}

		if !existingNamesMap[uniqueName] {
			return uniqueName
		}

		counter++

		// Safety check to prevent infinite loops
		if counter > 1000 {
			return cleanName
		}
	}
}

// GetUniqueTableName checks if a table with the given name exists in the schema
// If it does, appends a number (1, 2, 3, etc.) to make it unique
// Enforces 50 character limit for table names
func (s *importService) GetUniqueTableName(ctx context.Context, schemaName string, baseID string, proposedName string, lg *zerolog.Logger) (string, error) {
	const maxTableNameLength = 50 // UI display limit for table names

	allTables, err := s.tableService.GetModelByBaseID(ctx, schemaName, baseID)
	if err != nil {
		lg.Error().Stack().Err(err).Str("baseID", baseID).Msg("Failed to fetch existing tables")
		return proposedName, err
	}

	// Extract table titles into a slice
	existingTableNames := make([]string, len(allTables))
	for i, table := range allTables {
		existingTableNames[i] = table.Model.Title
	}

	// Find unique name using helper
	uniqueName := s.FindUniqueName(proposedName, existingTableNames, maxTableNameLength)

	if uniqueName == proposedName {
		return uniqueName, nil
	}

	if len(proposedName) > maxTableNameLength {
		lg.Info().Str("originalName", proposedName).Str("truncatedName", uniqueName).Int("limit", maxTableNameLength).Msg("Table name exceeds 50 character limit, truncating")
	} else {
		lg.Info().Str("originalName", proposedName).Str("uniqueName", uniqueName).Int("length", len(uniqueName)).Msg("Table name already exists, using unique name with 50 char limit")
	}

	return uniqueName, nil
}

// GetUniqueBaseName checks if a base with the given name exists in the workspace
// If it does, appends a number (1, 2, 3, etc.) to make it unique
// Enforces 50 character limit for base names
func (s *importService) GetUniqueBaseName(ctx context.Context, schemaName string, workspaceID string, proposedName string, lg *zerolog.Logger) (string, error) {
	allBases, err := s.baseManagementService.GetBasesByWorkspace(ctx, schemaName, workspaceID)
	if err != nil {
		lg.Error().Stack().Err(err).Str("workspaceID", workspaceID).Msg("Failed to fetch existing bases")
		return proposedName, err
	}

	// Remove "_base" suffix if present to get the clean base name
	cleanBaseName := strings.TrimSuffix(proposedName, "_base")

	// Extract base titles into a slice
	existingBaseNames := make([]string, len(allBases))
	for i, base := range allBases {
		existingBaseNames[i] = base.Title
	}

	const maxNameLength = 50

	// Find unique name using helper
	uniqueName := s.FindUniqueName(cleanBaseName, existingBaseNames, maxNameLength)

	if uniqueName == cleanBaseName && len(proposedName) > maxNameLength {
		lg.Warn().Str("baseName", cleanBaseName).Int("length", len(cleanBaseName)).Int("maxLength", maxNameLength).Msg("Base name exceeds 50 character limit, truncating")
	} else if uniqueName != cleanBaseName {
		lg.Info().Str("originalName", proposedName).Str("uniqueName", uniqueName).Int("length", len(uniqueName)).Msg("Base name already exists, using unique name with 50 char limit")
	}

	return uniqueName, nil
}

// EnsureBaseWithConfig ensures a base exists for config-based import
func (s *importService) EnsureBaseWithConfig(ctx context.Context, schemaName string, req *dto.ImportWithConfigRequest, lg *zerolog.Logger, tableTitle string) error {
	if req.BaseID != "" {
		return nil
	}

	if req.WorkspaceID == "" {
		lg.Error().Msg(errWorkspaceIDRequiredMsg)
		return fmt.Errorf("%s", errWorkspaceIDRequiredMsg)
	}

	baseName := tableTitle

	// Check for duplicate base names and get unique name if needed (with 50 char limit)
	uniqueBaseName, err := s.GetUniqueBaseName(ctx, schemaName, req.WorkspaceID, baseName, lg)
	if err != nil {
		lg.Error().Stack().Err(err).Str("baseName", baseName).Msg("Failed to check for duplicate base names")
		// If check fails, at least truncate to 50 characters to avoid database errors
		if len(baseName) > 50 {
			uniqueBaseName = baseName[:50]
			lg.Warn().Str("originalBaseName", baseName).Str("truncatedBaseName", uniqueBaseName).Msg("Truncating base name to 50 characters after duplicate check failure")
		} else {
			uniqueBaseName = baseName
		}
	}

	lg.Info().Str("baseName", uniqueBaseName).Str("workspaceID", req.WorkspaceID).Msg("Creating new base for import")

	createBaseReq := dto.CreateBaseRequest{
		Title:       uniqueBaseName,
		Description: helpers.StringPtr(descAutoCreatedBase),
		WorkspaceID: req.WorkspaceID,
		CreatedBy:   req.CreatedBy,
	}

	newBase, err := s.baseManagementService.CreateBaseWithoutTable(ctx, createBaseReq, schemaName, req.CreatedBy)
	if err != nil {
		lg.Error().Stack().Err(err).Str("baseName", uniqueBaseName).Msg(errFailedCreateBase)
		return fmt.Errorf("failed to create base: %w", err)
	}

	req.BaseID = newBase.ID.String()
	lg.Info().Str("baseID", req.BaseID).Str("baseName", uniqueBaseName).Msg("Base created successfully for import")
	return nil
}

// CleanData applies data cleaning transformations (trim, remove extra spaces)
func (s *importService) CleanData(rows [][]string, settings dto.ImportSettings) [][]string {
	cleanedRows := make([][]string, len(rows))

	for i, row := range rows {
		cleanedRow := make([]string, len(row))
		for j, cell := range row {
			cleanedCell := cell

			// Trim spaces if enabled
			if settings.TrimSpaces {
				cleanedCell = strings.TrimSpace(cleanedCell)
			}

			// Remove extra spaces if enabled
			if settings.RemoveEmptyRows {
				// Replace multiple spaces with single space
				cleanedCell = strings.Join(strings.Fields(cleanedCell), " ")
			}

			cleanedRow[j] = cleanedCell
		}
		cleanedRows[i] = cleanedRow
	}

	return cleanedRows
}

// AddColumnsWithConfig adds columns to the table using user-provided config
func (s *importService) AddColumnsWithConfig(params dto.AddColumnsWithConfigParams, lg *zerolog.Logger) (map[int]dto.ColumnResponse, error) {
	lg.Info().Int("columnCount", len(params.ColumnConfigs)).Msg("Starting to add columns with config")
	columnMap := make(map[int]dto.ColumnResponse)

	// Build a map of source names to configs for quick lookup
	configMap := make(map[string]dto.ColumnConfig)
	for _, cfg := range params.ColumnConfigs {
		configMap[cfg.ColumnName] = cfg
	}

	for i, header := range params.Headers {
		colResp, added, err := s.AddColumnForHeader(params, header, i, configMap, lg)
		if err != nil {
			return nil, err
		}
		if added {
			columnMap[i] = colResp
			lg.Debug().Str("columnTitle", colResp.Title).Str("columnType", colResp.UIDT).Msg("Column added with config")
		}
	}

	lg.Info().Int("columnsAdded", len(columnMap)).Msg("All columns added with config")
	return columnMap, nil
}

// UpdateTitleColumnWithConfig updates the existing Title column with provided config metadata
func (s *importService) UpdateTitleColumnWithConfig(params dto.AddColumnsWithConfigParams, cfg dto.ColumnConfig, header string, lg *zerolog.Logger) error {
	// find the existing Title column id
	titleColID := ""
	for _, c := range params.TableResp.Columns {
		if c.Title == "Title" {
			titleColID = c.ID.String()
			break
		}
	}
	if titleColID == "" {
		lg.Warn().Msg("Title column not found to update with config")
		return nil
	}

	// prepare update with meta/type/title
	colType := cfg.UIDT
	if colType == "" {
		colType = "text"
	}
	colDT := s.GetDatabaseType(colType)

	meta := map[string]interface{}{}
	if cfg.Meta != nil {
		meta = cfg.Meta
	}

	metaPtr := meta
	titleStr := cfg.Title
	if titleStr == "" {
		titleStr = header
	}
	updateReq := dto.ColumnUpdate{
		Title: &titleStr,
		UIDT:  &colType,
		DT:    &colDT,
		Meta:  &metaPtr,
	}
	if _, err := s.tableService.UpdateColumn(params.Ctx, params.SchemaName, titleColID, updateReq); err != nil {
		lg.Error().Stack().Err(err).Str("column", titleColID).Msg("Failed to update Title column with config meta")
		return err
	}
	return nil
}

// CreateColumnFromConfig constructs AddColumnRequest and calls tableService.AddColumn
func (s *importService) CreateColumnFromConfig(params dto.AddColumnsWithConfigParams, cfg dto.ColumnConfig, header string, i int, lg *zerolog.Logger) (dto.ColumnResponse, error) {
	colType := cfg.UIDT
	if colType == "" {
		colType = "text"
	}

	colTitle := cfg.Title
	if colTitle == "" {
		colTitle = header
	}

	colDT := s.GetDatabaseType(colType)

	meta := map[string]interface{}{}
	if cfg.Meta != nil {
		meta = cfg.Meta
	}

	addColReq := dto.AddColumnRequest{
		ModelID:     params.TableResp.Model.ID,
		BaseID:      params.TableResp.Model.BaseID,
		Title:       colTitle,
		Description: "",
		Meta:        meta,
		UIDT:        colType,
		DT:          colDT,
		OrderIndex:  helpers.Float64Ptr(float64(i + 6)),
		Virtual:     helpers.BoolPtr(false),
		System:      helpers.BoolPtr(false),
		CreatedBy:   params.Req.CreatedBy,
	}

	colResp, err := s.tableService.AddColumn(params.Ctx, params.SchemaName, addColReq)
	if err != nil {
		lg.Error().Stack().Err(err).Str("columnTitle", colTitle).Str("columnType", colType).Msg("Failed to add column with config")
		return dto.ColumnResponse{}, err
	}
	return colResp, nil
}

// AddColumnForHeader processes a single header: update title column or create new column
func (s *importService) AddColumnForHeader(params dto.AddColumnsWithConfigParams, header string, i int, configMap map[string]dto.ColumnConfig, lg *zerolog.Logger) (dto.ColumnResponse, bool, error) {
	if header == "" {
		return dto.ColumnResponse{}, false, nil
	}

	cfg, exists := configMap[header]
	primaryMatch := false
	if params.Primary != nil && params.Primary.ColumnName == header {
		primaryMatch = true
		cfg = *params.Primary
		exists = true
	}

	if !exists {
		lg.Debug().Str("columnName", header).Msg("Skipping column not in config")
		return dto.ColumnResponse{}, false, nil
	}

	if primaryMatch {
		if err := s.UpdateTitleColumnWithConfig(params, cfg, header, lg); err != nil {
			return dto.ColumnResponse{}, false, err
		}
		return dto.ColumnResponse{}, false, nil
	}

	colResp, err := s.CreateColumnFromConfig(params, cfg, header, i, lg)
	if err != nil {
		return dto.ColumnResponse{}, false, err
	}
	return colResp, true, nil
}

// SaveErrorRows writes error rows, empty rows, and duplicate rows to a text log file
func (s *importService) SaveErrorRows(headers []string, errorRows [][]string, errorMessages []string, emptyRowsWithLineNumbers map[int][]string, duplicateRowsWithLineNumbers map[int][]string, lg *zerolog.Logger) (string, error) {
	// Build the content using helper functions
	var content strings.Builder
	content.WriteString("Import Issues Report\n")
	content.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(strings.Repeat("=", 100))
	content.WriteString("\n\n")

	// Summary
	content.WriteString("SUMMARY:\n")
	content.WriteString(fmt.Sprintf("Total Error Rows: %d\n", len(errorRows)))
	content.WriteString(fmt.Sprintf("Total Empty Rows: %d\n", len(emptyRowsWithLineNumbers)))
	content.WriteString(fmt.Sprintf("Total Duplicate Rows: %d\n", len(duplicateRowsWithLineNumbers)))

	if len(errorRows) == 0 && len(emptyRowsWithLineNumbers) == 0 && len(duplicateRowsWithLineNumbers) == 0 {
		content.WriteString("\nStatus: ✓ No issues detected - All rows are valid\n")
	}
	content.WriteString(strings.Repeat("-", 100))
	content.WriteString("\n\nCSV Headers:\n")
	content.WriteString(strings.Join(headers, " | "))
	content.WriteString("\n\n")

	// Use helper functions for section building
	if len(errorMessages) > 0 {
		content.WriteString(s.BuildErrorTypeSummary(errorMessages))
		content.WriteString(s.BuildAllValidationErrorsBlock(errorMessages))
	}
	if len(emptyRowsWithLineNumbers) > 0 {
		content.WriteString(s.BuildEmptyRowsHumanSection(emptyRowsWithLineNumbers))
	}
	if len(duplicateRowsWithLineNumbers) > 0 {
		content.WriteString(s.BuildDuplicateRowsHumanSection(duplicateRowsWithLineNumbers))
	}

	// Raw CSV data section
	content.WriteString(s.BuildRawCSVSection(headers, errorRows, emptyRowsWithLineNumbers, duplicateRowsWithLineNumbers))

	// Ensure tmp directory exists
	tmpDir := "./internal/tmp"
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			lg.Warn().Err(err).Str("dir", tmpDir).Msg("Failed to create tmp directory, continuing without file save")
		}
	}

	// Write to file
	errorFile := fmt.Sprintf("%s/import_error_rows_%s.txt", tmpDir, time.Now().Format("20060102_150405"))
	if err := os.WriteFile(errorFile, []byte(content.String()), 0644); err != nil {
		lg.Warn().Err(err).Str("file", errorFile).Msg("Failed to save error rows to file, but returning content anyway")
	} else {
		lg.Info().Str("file", errorFile).Int("errorRowCount", len(errorRows)).Int("emptyRowCount", len(emptyRowsWithLineNumbers)).Int("duplicateRowCount", len(duplicateRowsWithLineNumbers)).Msg("Import log file generated successfully")
	}

	return content.String(), nil
}

// EscapeCSVCell returns a CSV-escaped cell (quotes doubled and wrapped) when needed
func (s *importService) EscapeCSVCell(cell string) string {
	if strings.Contains(cell, ",") || strings.Contains(cell, "\"") || strings.Contains(cell, "\n") {
		return "\"" + strings.ReplaceAll(cell, "\"", "\"\"") + "\""
	}
	return cell
}

// EscapeRowForCSV returns a slice of escaped cells for a row
func (s *importService) EscapeRowForCSV(row []string) []string {
	escaped := make([]string, len(row))
	for i, cell := range row {
		escaped[i] = s.EscapeCSVCell(cell)
	}
	return escaped
}

// SortedLineNumbers returns sorted keys from a map[int][]string
func (s *importService) SortedLineNumbers(m map[int][]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// BuildErrorTypeSummary builds the CSV validation error types summary block
func (s *importService) BuildErrorTypeSummary(errorMessages []string) string {
	var b strings.Builder
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n")
	b.WriteString("CSV VALIDATION ERROR TYPES\n")
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n\n")

	errorTypeCount := make(map[string]int)
	for _, errMsg := range errorMessages {
		if strings.Contains(errMsg, "Invalid number format") {
			errorTypeCount["Invalid Number Format"]++
		} else if strings.Contains(errMsg, "Invalid decimal format") {
			errorTypeCount["Invalid Decimal Format"]++
		} else if strings.Contains(errMsg, "Invalid boolean value") {
			errorTypeCount["Invalid Boolean Format"]++
		} else if strings.Contains(errMsg, "Invalid email format") {
			errorTypeCount["Invalid Email Format"]++
		} else if strings.Contains(errMsg, "Invalid JSON format") {
			errorTypeCount["Invalid JSON Format"]++
		} else if strings.Contains(errMsg, "Text length") {
			errorTypeCount["Field Length Violation"]++
		} else if strings.Contains(errMsg, "exceeds maximum") || strings.Contains(errMsg, "is less than minimum") {
			errorTypeCount["Value Out of Range"]++
		} else if strings.Contains(errMsg, "out of range for") {
			errorTypeCount["Value Out of Range"]++
		}
	}

	// Sort error types for consistent output
	errorTypes := make([]string, 0, len(errorTypeCount))
	for errType := range errorTypeCount {
		errorTypes = append(errorTypes, errType)
	}
	sort.Strings(errorTypes)

	for _, errType := range errorTypes {
		b.WriteString(fmt.Sprintf("  • %s: %d errors\n", errType, errorTypeCount[errType]))
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 100))
	b.WriteString("\n\n")
	return b.String()
}

// BuildAllValidationErrorsBlock builds the detailed ALL VALIDATION ERRORS block
func (s *importService) BuildAllValidationErrorsBlock(errorMessages []string) string {
	var b strings.Builder
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n")
	b.WriteString("ALL VALIDATION ERRORS (Detailed Error Analysis)\n")
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n\n")

	for i, errorMsg := range errorMessages {
		b.WriteString(fmt.Sprintf("[Error Set %d]\n", i+1))
		b.WriteString(fmt.Sprintf("%s\n", errorMsg))
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat("-", 100))
	b.WriteString("\n\n")
	return b.String()
}

// BuildEmptyRowsHumanSection builds the human-readable EMPTY ROWS section
func (s *importService) BuildEmptyRowsHumanSection(emptyRowsWithLineNumbers map[int][]string) string {
	var b strings.Builder
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n")
	b.WriteString("EMPTY ROWS (All cells are empty)\n")
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n\n")

	lineNumbers := s.SortedLineNumbers(emptyRowsWithLineNumbers)
	for idx, lineNum := range lineNumbers {
		row := emptyRowsWithLineNumbers[lineNum]
		b.WriteString(fmt.Sprintf("[Empty Row %d] Line %d in CSV file\n", idx+1, lineNum))
		b.WriteString(fmt.Sprintf("Row Data: %v\n", row))
		b.WriteString("\n")
	}
	return b.String()
}

// BuildDuplicateRowsHumanSection builds the human-readable DUPLICATE ROWS section
func (s *importService) BuildDuplicateRowsHumanSection(duplicateRowsWithLineNumbers map[int][]string) string {
	var b strings.Builder
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n")
	b.WriteString("DUPLICATE ROWS (Identical rows found)\n")
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n\n")

	lineNumbers := s.SortedLineNumbers(duplicateRowsWithLineNumbers)
	for idx, lineNum := range lineNumbers {
		row := duplicateRowsWithLineNumbers[lineNum]
		b.WriteString(fmt.Sprintf("[Duplicate Row %d] Line %d in CSV file\n", idx+1, lineNum))
		b.WriteString(fmt.Sprintf("Row Data: %v\n", row))
		b.WriteString("\n")
	}
	return b.String()
}

// BuildRawCSVSection builds the RAW DATA (CSV Format) block including headers and CSV rows
func (s *importService) BuildRawCSVSection(headers []string, errorRows [][]string, emptyRowsWithLineNumbers map[int][]string, duplicateRowsWithLineNumbers map[int][]string) string {
	var b strings.Builder
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n")
	b.WriteString("RAW DATA (CSV Format)\n")
	b.WriteString(strings.Repeat("=", 100))
	b.WriteString("\n\n")

	// Headers
	b.WriteString(strings.Join(headers, ","))
	b.WriteString("\n")

	// Add row sections
	s.AppendErrorRowsToCSV(&b, errorRows)
	s.AppendEmptyRowsToCSV(&b, emptyRowsWithLineNumbers)
	s.AppendDuplicateRowsToCSV(&b, duplicateRowsWithLineNumbers)

	return b.String()
}

// AppendErrorRowsToCSV appends error rows to CSV output
func (s *importService) AppendErrorRowsToCSV(b *strings.Builder, errorRows [][]string) {
	if len(errorRows) == 0 {
		return
	}
	b.WriteString("# ERROR ROWS:\n")
	for _, row := range errorRows {
		b.WriteString(strings.Join(s.EscapeRowForCSV(row), ","))
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

// AppendEmptyRowsToCSV appends empty rows to CSV output
func (s *importService) AppendEmptyRowsToCSV(b *strings.Builder, emptyRows map[int][]string) {
	if len(emptyRows) == 0 {
		return
	}
	b.WriteString("# EMPTY ROWS:\n")
	for _, lineNum := range s.SortedLineNumbers(emptyRows) {
		row := emptyRows[lineNum]
		b.WriteString(fmt.Sprintf("# Line %d: %s\n", lineNum, strings.Join(s.EscapeRowForCSV(row), ",")))
	}
	b.WriteString("\n")
}

// AppendDuplicateRowsToCSV appends duplicate rows to CSV output
func (s *importService) AppendDuplicateRowsToCSV(b *strings.Builder, duplicateRows map[int][]string) {
	if len(duplicateRows) == 0 {
		return
	}
	b.WriteString("# DUPLICATE ROWS:\n")
	for _, lineNum := range s.SortedLineNumbers(duplicateRows) {
		row := duplicateRows[lineNum]
		b.WriteString(fmt.Sprintf("# Line %d: %s\n", lineNum, strings.Join(s.EscapeRowForCSV(row), ",")))
	}
}

// GetDefaultValue extracts default value from column config metadata
func (s *importService) GetDefaultValue(cfg *dto.ColumnConfig) string {
	if cfg == nil || cfg.Meta == nil {
		return ""
	}
	if dv, ok := cfg.Meta["default_value"]; ok {
		if sVal, ok2 := dv.(string); ok2 {
			return sVal
		}
	}
	return ""
}

// ValidateNumberField validates integer number fields including range and meta bounds
func (s *importService) ValidateNumberField(cellVal string, columnName string, meta map[string]interface{}) []string {
	var errors []string

	// For integer field, reject decimal values
	if strings.Contains(cellVal, ".") {
		errors = append(errors, fmt.Sprintf("Column '%s' [number]: Invalid number format '%s' - integer field cannot contain decimal point", columnName, cellVal))
		return errors
	}

	if intVal, err := strconv.ParseInt(cellVal, 10, 64); err != nil {
		errors = append(errors, fmt.Sprintf("Column '%s' [number]: Invalid number format '%s' - cannot parse as integer", columnName, cellVal))
	} else {
		if intVal > math.MaxInt32 || intVal < math.MinInt32 {
			errors = append(errors, fmt.Sprintf("Column '%s' [number]: Value %s is out of range for integer type (must be between %d and %d)", columnName, cellVal, math.MinInt32, math.MaxInt32))
		}
	}

	// Check numeric bounds using generic helper
	if boundErrors := s.CheckNumericBounds(cellVal, columnName, "number", meta); len(boundErrors) > 0 {
		errors = append(errors, boundErrors...)
	}

	return errors
}

// ValidateDecimalField validates floating point decimal fields and meta bounds
func (s *importService) ValidateDecimalField(cellVal string, columnName string, meta map[string]interface{}) []string {
	var errors []string
	if _, err := strconv.ParseFloat(cellVal, 64); err != nil {
		errors = append(errors, fmt.Sprintf("Column '%s' [decimal]: Invalid decimal format '%s' - must be a valid floating point number", columnName, cellVal))
		return errors
	}
	// Check numeric bounds using generic helper
	if boundErrors := s.CheckNumericBounds(cellVal, columnName, "decimal", meta); len(boundErrors) > 0 {
		errors = append(errors, boundErrors...)
	}

	return errors
}

// CheckNumericBounds checks meta min/max bounds for numeric values and returns any errors
func (s *importService) CheckNumericBounds(cellVal string, columnName string, fieldType string, meta map[string]interface{}) []string {
	if meta == nil {
		return nil
	}

	numVal, err := strconv.ParseFloat(cellVal, 64)
	if err != nil {
		return nil // Already validated in caller
	}

	var errors []string
	if errs := s.CheckMinBound(numVal, cellVal, columnName, fieldType, meta); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := s.CheckMaxBound(numVal, cellVal, columnName, fieldType, meta); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	return errors
}

// CheckMinBound checks minimum bound for numeric values
func (s *importService) CheckMinBound(numVal float64, cellVal string, columnName string, fieldType string, meta map[string]interface{}) []string {
	if minVal, ok := meta["min"]; ok {
		if minFloat, ok2 := minVal.(float64); ok2 && numVal < minFloat {
			return []string{fmt.Sprintf("Column '%s' [%s]: Value %s is less than minimum %v", columnName, fieldType, cellVal, minFloat)}
		}
	}
	return nil
}

// CheckMaxBound checks maximum bound for numeric values
func (s *importService) CheckMaxBound(numVal float64, cellVal string, columnName string, fieldType string, meta map[string]interface{}) []string {
	if maxVal, ok := meta["max"]; ok {
		if maxFloat, ok2 := maxVal.(float64); ok2 && numVal > maxFloat {
			return []string{fmt.Sprintf("Column '%s' [%s]: Value %s exceeds maximum %v", columnName, fieldType, cellVal, maxFloat)}
		}
	}
	return nil
}

// ValidateBooleanField validates boolean-like values
func (s *importService) ValidateBooleanField(cellVal string, columnName string) []string {
	lower := strings.ToLower(cellVal)
	if lower != "true" && lower != "false" && lower != "0" && lower != "1" && lower != "yes" && lower != "no" {
		return []string{fmt.Sprintf("Column '%s' [boolean]: Invalid boolean value '%s' - must be one of: true, false, 0, 1, yes, no", columnName, cellVal)}
	}
	return nil
}

// ValidateEmailField performs basic email format checks
func (s *importService) ValidateEmailField(cellVal string, columnName string) []string {
	atIndex := strings.LastIndex(cellVal, "@")
	if atIndex == -1 || atIndex == 0 || atIndex == len(cellVal)-1 {
		return []string{fmt.Sprintf("Column '%s' [email]: Invalid email format '%s' - must contain @ with local and domain parts", columnName, cellVal)}
	}
	domain := cellVal[atIndex+1:]
	if !strings.Contains(domain, ".") {
		return []string{fmt.Sprintf("Column '%s' [email]: Invalid email format '%s' - domain must contain a dot (.)", columnName, cellVal)}
	}
	return nil
}

// ValidateJSONField checks if the value is valid JSON
func (s *importService) ValidateJSONField(cellVal string, columnName string) []string {
	var js interface{}
	if err := json.Unmarshal([]byte(cellVal), &js); err != nil {
		return []string{fmt.Sprintf("Column '%s' [json]: Invalid JSON format '%s' - %v", columnName, cellVal, err)}
	}
	return nil
}

// ValidateTextField validates text/longText fields against max_length meta
func (s *importService) ValidateTextField(cellVal string, columnName string, fieldType string, meta map[string]interface{}) []string {
	if meta != nil {
		if maxLen, ok := meta["max_length"]; ok {
			if maxLenInt, ok2 := maxLen.(float64); ok2 {
				if len(cellVal) > int(maxLenInt) {
					return []string{fmt.Sprintf("Column '%s' [%s]: Text length %d exceeds maximum length of %d characters", columnName, fieldType, len(cellVal), int(maxLenInt))}
				}
			}
		}
	}
	return nil
}

// BuildRecordsWithConfigAndErrors builds records using column configuration and tracks comprehensive error rows
func (s *importService) BuildRecordsWithConfigAndErrors(params dto.BuildRecordsWithConfigAndErrorsParams, lg *zerolog.Logger) ([]map[string]interface{}, [][]string, []string) {
	lg.Info().Int("recordCount", len(params.DataRows)).Msg("Starting to insert records with comprehensive error tracking")

	// Build config map for quick lookup
	configMap := make(map[string]dto.ColumnConfig)
	for _, cfg := range params.ColumnConfigs {
		configMap[cfg.ColumnName] = cfg
	}

	var newRecords []map[string]interface{}
	var errorRows [][]string
	var errorMessages []string

	for rowIdx, row := range params.DataRows {
		record, rowErrors, valid := s.BuildRecordFromRow(rowIdx, row, params, configMap, lg)

		if valid && len(rowErrors) == 0 {
			newRecords = append(newRecords, record)
		} else {
			errorRows = append(errorRows, row)
			errorMsg := fmt.Sprintf("Row %d Errors:\n%s\nRow Data: %v", rowIdx+2, strings.Join(rowErrors, "\n"), row)
			errorMessages = append(errorMessages, errorMsg)
			for _, errMsg := range rowErrors {
				lg.Warn().Int("rowNumber", rowIdx+2).Str("error", errMsg).Msg("Validation error")
			}
		}
	}

	lg.Info().Int("validRecords", len(newRecords)).Int("errorRows", len(errorRows)).Int("totalErrors", len(errorMessages)).Msg("Record processing completed with comprehensive error tracking")
	return newRecords, errorRows, errorMessages
}

// BuildRecordFromRow constructs a record from a CSV row based on provided params and returns any validation errors
func (s *importService) BuildRecordFromRow(rowIdx int, row []string, params dto.BuildRecordsWithConfigAndErrorsParams, configMap map[string]dto.ColumnConfig, lg *zerolog.Logger) (map[string]interface{}, []string, bool) {
	record := map[string]interface{}{
		"created_by":         params.Req.CreatedBy,
		"last_modified_by":   params.Req.CreatedBy,
		"created_time":       time.Now().UTC(),
		"last_modified_time": time.Now().UTC(),
	}
	rowErrors, recordValid := s.PopulateRecordFromCells(record, row, params, configMap, lg)
	return record, rowErrors, recordValid
}

// PopulateRecordFromCells iterates cells in a row and applies validations/conversions
func (s *importService) PopulateRecordFromCells(record map[string]interface{}, row []string, params dto.BuildRecordsWithConfigAndErrorsParams, configMap map[string]dto.ColumnConfig, lg *zerolog.Logger) ([]string, bool) {
	var rowErrors []string
	recordValid := true

	for i, cellVal := range row {
		if i >= len(params.Headers) {
			break
		}

		errs := s.ProcessCellByType(record, i, cellVal, params, configMap)
		if len(errs) > 0 {
			rowErrors = append(rowErrors, errs...)
			recordValid = false
		}
	}

	return rowErrors, recordValid
}

// ProcessCellByType processes a cell based on whether it's primary or non-primary
func (s *importService) ProcessCellByType(record map[string]interface{}, colIdx int, cellVal string, params dto.BuildRecordsWithConfigAndErrorsParams, configMap map[string]dto.ColumnConfig) []string {
	header := params.Headers[colIdx]
	cfg, configExists := configMap[header]

	// Handle primary column
	if params.Primary != nil && params.Primary.ColumnName == header {
		cfg = *params.Primary
		return s.ApplyPrimaryCell(record, header, cfg, cellVal)
	}

	// Handle non-primary column
	errs, applied := s.ApplyNonPrimary(record, header, cfg, configExists, params, colIdx, cellVal)
	if len(errs) > 0 || applied {
		return errs
	}
	return nil
}

// ApplyNonPrimary handles non-primary column processing for a single cell
func (s *importService) ApplyNonPrimary(record map[string]interface{}, header string, cfg dto.ColumnConfig, configExists bool, params dto.BuildRecordsWithConfigAndErrorsParams, i int, cellVal string) ([]string, bool) {
	if !configExists {
		return nil, false
	}

	colResp, colExists := params.ColumnMap[i]
	if !colExists {
		return nil, false
	}

	return s.HandleDataCellForRow(record, header, cfg, colResp, cellVal)
}

// ApplyPrimaryCell validates and applies the primary/title cell to the record
func (s *importService) ApplyPrimaryCell(record map[string]interface{}, header string, cfg dto.ColumnConfig, cellVal string) []string {
	if val, errs := s.ProcessTitleCell(header, cfg, cellVal); len(errs) > 0 {
		return errs
	} else if val != nil {
		record["title"] = val
	}
	return nil
}

// HandleDataCellForRow validates a non-primary data cell and applies it to the record if valid
func (s *importService) HandleDataCellForRow(record map[string]interface{}, header string, cfg dto.ColumnConfig, colResp dto.ColumnResponse, cellVal string) ([]string, bool) {
	if key, val, errs, ok := s.ProcessDataCell(header, cfg, colResp, cellVal); len(errs) > 0 {
		return errs, false
	} else if ok {
		record[key] = val
		return nil, true
	}
	return nil, false
}

// ProcessTitleCell handles validation and conversion for the primary/title column
func (s *importService) ProcessTitleCell(header string, cfg dto.ColumnConfig, cellVal string) (interface{}, []string) {
	defaultVal := s.GetDefaultValue(&cfg)

	// Determine which value to use
	var primaryValue string
	if cellVal != "" {
		primaryValue = cellVal
	} else if defaultVal != "" {
		primaryValue = defaultVal
	}

	// If both cell and default are empty, title remains unset
	if primaryValue == "" {
		return nil, nil
	}

	// Validate primary column type conversion
	conversionErr := s.ValidateConversion(primaryValue, cfg.UIDT)
	if conversionErr != "" {
		return nil, []string{"Primary column: " + conversionErr}
	}

	return s.ConvertValue(primaryValue, cfg.UIDT), nil
}

// ProcessDataCell handles validation and conversion for non-primary data cells
func (s *importService) ProcessDataCell(header string, cfg dto.ColumnConfig, colResp dto.ColumnResponse, cellVal string) (string, interface{}, []string, bool) {
	defaultVal := s.GetDefaultValue(&cfg)

	// Determine which value to use
	var valueToConvert string
	if cellVal == "" && defaultVal != "" {
		valueToConvert = defaultVal
	} else if cellVal != "" {
		valueToConvert = cellVal
	} else {
		return "", nil, nil, false // Skip empty cells with no default
	}

	// Validate type conversion before applying
	conversionErr := s.ValidateConversion(valueToConvert, cfg.UIDT)
	if conversionErr != "" {
		return "", nil, []string{conversionErr}, false
	}

	val := s.ConvertValue(valueToConvert, cfg.UIDT)
	key := fmt.Sprintf(colNameFmt, colResp.ColumnName)
	return key, val, nil, true
}

// ValidateColumnConfig validates that all column configs have required fields and valid source names
func (s *importService) ValidateColumnConfig(columnConfigs []dto.ColumnConfig, primary *dto.ColumnConfig, headers []string, lg *zerolog.Logger) error {
	if len(columnConfigs) == 0 {
		lg.Error().Msg("No columns provided in config")
		return fmt.Errorf("at least one column must be configured")
	}

	// Create a set of valid header names for quick lookup
	validHeaders := make(map[string]bool)
	for _, header := range headers {
		if header != "" {
			validHeaders[header] = true
		}
	}

	if err := s.ValidatePrimaryConfig(primary, headers, lg); err != nil {
		return err
	}

	if err := s.ValidateEachColumnConfig(columnConfigs, validHeaders, lg); err != nil {
		return err
	}

	lg.Info().Int("columnCount", len(columnConfigs)).Msg("Column config validation passed")
	return nil
}

// ValidatePrimaryConfig checks the primary column configuration
func (s *importService) ValidatePrimaryConfig(primary *dto.ColumnConfig, headers []string, lg *zerolog.Logger) error {
	if primary == nil {
		lg.Error().Msg("primary_column is required in import config")
		return fmt.Errorf("primary_column is required in import config")
	}
	if primary.ColumnName == "" {
		lg.Error().Msg("PrimaryColumn provided but ColumnName is empty")
		return fmt.Errorf("primary_column: column_name is required")
	}

	primaryValid := false
	for _, h := range headers {
		if h == primary.ColumnName {
			primaryValid = true
			break
		}
	}
	if !primaryValid {
		lg.Error().Str("primaryColumn", primary.ColumnName).Msg("PrimaryColumn does not match any CSV header")
		return fmt.Errorf("primary_column '%s' not found in CSV headers", primary.ColumnName)
	}
	return nil
}

// ValidateEachColumnConfig iterates and checks each column config entry
func (s *importService) ValidateEachColumnConfig(columnConfigs []dto.ColumnConfig, validHeaders map[string]bool, lg *zerolog.Logger) error {
	for i, cfg := range columnConfigs {
		if cfg.ColumnName == "" {
			lg.Error().Int("columnIndex", i).Msg("ColumnName is required for column config")
			return fmt.Errorf("column %d: column_name is required", i)
		}
		if !validHeaders[cfg.ColumnName] {
			lg.Error().Str("columnName", cfg.ColumnName).Msg("ColumnName does not match any CSV column header")
			return fmt.Errorf("column %d: column_name '%s' not found in CSV headers", i, cfg.ColumnName)
		}
		if cfg.Title == "" {
			lg.Warn().Str("columnName", cfg.ColumnName).Msg("Title is empty for column config, will use column name")
		}
		if cfg.UIDT == "" {
			lg.Warn().Str("columnName", cfg.ColumnName).Msg("UIDT is empty for column config, defaulting to text")
		}
	}
	return nil
}

// IsRowEmpty checks if all cells in a row are empty (after trimming)
func (s *importService) IsRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// RemoveEmptyRows removes rows where all cells are empty
func (s *importService) RemoveEmptyRows(rows [][]string) [][]string {
	var nonEmptyRows [][]string
	for _, row := range rows {
		allEmpty := true
		for _, cell := range row {
			if cell != "" { // do not trim, only exact empty string
				allEmpty = false
				break
			}
		}
		if !allEmpty {
			nonEmptyRows = append(nonEmptyRows, row)
		}
	}
	return nonEmptyRows
}

// RemoveDuplicateRecords removes duplicate rows while preserving order
func (s *importService) RemoveDuplicateRecords(rows [][]string) [][]string {
	seen := make(map[string]bool)
	var uniqueRows [][]string

	for _, row := range rows {
		// Create a key from the row by joining all cells
		key := strings.Join(row, "|")

		if !seen[key] {
			seen[key] = true
			uniqueRows = append(uniqueRows, row)
		}
	}

	return uniqueRows
}

// CountEmptyRows counts rows where all cells are empty without removing them
func (s *importService) CountEmptyRows(rows [][]string) int {
	count := 0
	for _, row := range rows {
		allEmpty := true
		for _, cell := range row {
			if cell != "" { // do not trim, only exact empty string
				allEmpty = false
				break
			}
		}
		if allEmpty {
			count++
		}
	}
	return count
}

// CountDuplicateRecords counts duplicate rows without removing them
func (s *importService) CountDuplicateRecords(rows [][]string) int {
	seen := make(map[string]bool)
	duplicateCount := 0

	for _, row := range rows {
		// Create a key from the row by joining all cells
		key := strings.Join(row, "|")

		if !seen[key] {
			seen[key] = true
		} else {
			duplicateCount++
		}
	}

	return duplicateCount
}

// IdentifyEmptyRowsWithLineNumbers returns empty rows with their line numbers in the CSV
func (s *importService) IdentifyEmptyRowsWithLineNumbers(rows [][]string) map[int][]string {
	emptyRows := make(map[int][]string)
	for idx, row := range rows {
		allEmpty := true
		for _, cell := range row {
			if cell != "" { // do not trim, only exact empty string
				allEmpty = false
				break
			}
		}
		if allEmpty {
			// Line number is idx + 2 (1 for header, 1 for 1-based indexing)
			emptyRows[idx+2] = row
		}
	}
	return emptyRows
}

// IdentifyDuplicateRowsWithLineNumbers returns duplicate rows with their line numbers in the CSV
func (s *importService) IdentifyDuplicateRowsWithLineNumbers(rows [][]string) map[int][]string {
	seen := make(map[string]int) // key -> first occurrence line number
	duplicates := make(map[int][]string)

	for idx, row := range rows {
		key := strings.Join(row, "|")
		lineNum := idx + 2 // Line number is idx + 2 (1 for header, 1 for 1-based indexing)

		if _, exists := seen[key]; exists {
			// This is a duplicate of the row at firstOccurrence
			duplicates[lineNum] = row
		} else {
			// First occurrence of this row
			seen[key] = lineNum
		}
	}

	return duplicates
}

// LogCleanedDataWithSettings logs and saves cleaned data to a temporary JSON file before insertion
// This is called EVERY time regardless of which settings are enabled
func (s *importService) LogCleanedDataWithSettings(headers []string, dataRows [][]string, stats *dto.ImportStatistics, settings dto.ImportSettings, lg *zerolog.Logger) {
	// Create a structured representation of the cleaned data
	cleanedData := map[string]interface{}{
		"headers": headers,
		"rows":    dataRows,
		"statistics": map[string]interface{}{
			"total_rows":         stats.TotalRows,
			"total_columns":      stats.TotalColumns,
			"empty_rows":         stats.EmptyRows,
			"duplicate_rows":     stats.DuplicateRows,
			"rows_after_cleanup": len(dataRows),
			"empty_rows_skipped": stats.EmptyRowsSkipped,
			"duplicates_removed": stats.DuplicatesRemoved,
		},
		"settings_applied": map[string]interface{}{
			"trim_spaces":              settings.TrimSpaces,
			"remove_empty_rows":        settings.RemoveEmptyRows,
			"remove_duplicate_records": settings.RemoveDuplicateRecords,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(cleanedData, "", "  ")
	if err != nil {
		lg.Warn().Err(err).Msg("Failed to marshal cleaned data to JSON")
		return
	}

	// Save to application tmp directory (host-visible)
	tmpDir := "./internal/tmp"
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			lg.Warn().Err(err).Str("dir", tmpDir).Msg("Failed to create tmp directory for cleaned data")
		}
	}
	tmpFile := fmt.Sprintf("%s/import_cleaned_data_%s.json", tmpDir, time.Now().Format("20060102_150405"))
	if err := os.WriteFile(tmpFile, jsonData, 0644); err != nil {
		lg.Warn().Err(err).Str("file", tmpFile).Msg("Failed to save cleaned data to tmp directory")
	} else {
		lg.Info().Str("file", tmpFile).Msg("Cleaned data saved to application tmp directory")
	}

	// Log a summary of the cleaned data
	lg.Info().
		Int("headerCount", len(headers)).
		Int("rowCount", len(dataRows)).
		Int("totalOriginalRows", stats.TotalRows).
		Int("emptyRowsSkipped", stats.EmptyRowsSkipped).
		Int("duplicatesRemoved", stats.DuplicatesRemoved).
		Bool("trimSpacesApplied", settings.TrimSpaces).
		Bool("removeEmptyRowsApplied", settings.RemoveEmptyRows).
		Bool("removeDuplicatesApplied", settings.RemoveDuplicateRecords).
		Str("headers", fmt.Sprintf("%v", headers)).
		Msg("Cleaned data summary (full data saved to temp file)")

	// Log first few rows as sample
	sampleSize := 3
	if len(dataRows) < sampleSize {
		sampleSize = len(dataRows)
	}
	for i := 0; i < sampleSize; i++ {
		lg.Debug().
			Int("rowIndex", i).
			Str("rowData", fmt.Sprintf("%v", dataRows[i])).
			Msg("Sample cleaned row")
	}
}

// PrepareImportData parses CSV, validates column config, applies cleaning settings,
// logs cleaned data and determines a unique table title. Returns headers, cleaned
// data rows, statistics and the unique table title (or original on error).
func (s *importService) PrepareImportData(ctx context.Context, schemaName string, req dto.ImportWithConfigRequest, file *multipart.FileHeader, tableTitle string, lg *zerolog.Logger) ([]string, [][]string, *dto.ImportStatistics, string, error) {
	headers, dataRows, err := s.ParseCSV(file, lg)
	if err != nil {
		return nil, nil, nil, "", err
	}

	// Initialize statistics
	stats := &dto.ImportStatistics{}

	// Validate column config
	if err := s.ValidateColumnConfig(req.Config.Columns, req.Config.PrimaryColumn, headers, lg); err != nil {
		return nil, nil, stats, "", err
	}

	// Step 1: Apply TrimSpaces setting (if enabled)
	if req.Config.Settings.TrimSpaces {
		dataRows = s.CleanData(dataRows, req.Config.Settings)
		lg.Info().Bool("trimSpaces", true).Msg("Trim spaces applied")
	} else {
		lg.Info().Bool("trimSpaces", false).Msg("Trim spaces skipped")
	}

	// Step 2: Count and optionally remove empty rows (after trimming for accurate detection)
	emptyRowCount := s.CountEmptyRows(dataRows)
	stats.EmptyRows = emptyRowCount
	if req.Config.Settings.RemoveEmptyRows {
		dataRows = s.RemoveEmptyRows(dataRows)
		stats.EmptyRowsSkipped = emptyRowCount
		lg.Info().Bool("RemoveEmptyRows", true).Int("emptyRowsDetected", emptyRowCount).Int("emptyRowsRemoved", emptyRowCount).Msg("Empty rows removed")
	} else {
		stats.EmptyRowsSkipped = 0
		lg.Info().Bool("RemoveEmptyRows", false).Int("emptyRowsDetected", emptyRowCount).Msg("Empty rows detected but not removed")
	}

	// Step 3: Count and optionally remove duplicates
	duplicateRowCount := s.CountDuplicateRecords(dataRows)
	stats.DuplicateRows = duplicateRowCount
	if req.Config.Settings.RemoveDuplicateRecords {
		dataRows = s.RemoveDuplicateRecords(dataRows)
		stats.DuplicatesRemoved = duplicateRowCount
		lg.Info().Bool("removeDuplicates", true).Int("duplicatesDetected", duplicateRowCount).Int("duplicatesRemoved", duplicateRowCount).Int("recordsAfterDedup", len(dataRows)).Msg("Duplicate records removed")
	} else {
		stats.DuplicatesRemoved = 0
		lg.Info().Bool("removeDuplicates", false).Int("duplicatesDetected", duplicateRowCount).Msg("Duplicate records detected but not removed")
	}

	// Always log and save cleaned data to JSON file (regardless of settings)
	s.LogCleanedDataWithSettings(headers, dataRows, stats, req.Config.Settings, lg)

	// Check for duplicate table names and get unique name if needed
	uniqueTableTitle := tableTitle
	uniqueTableTitle, err = s.GetUniqueTableName(ctx, schemaName, req.BaseID, tableTitle, lg)
	if err != nil {
		lg.Error().Stack().Err(err).Str("tableName", tableTitle).Msg("Failed to check for duplicate table names")
		// Continue with original name if check fails
		uniqueTableTitle = tableTitle
	}

	return headers, dataRows, stats, uniqueTableTitle, nil
}

// ValidateConversion checks if a value can be successfully converted to the target type
// Returns an error message if conversion would fail, or empty string if valid
func (s *importService) ValidateConversion(val string, typeName string) string {
	if val == "" {
		return "" // Empty values are allowed (will use defaults or be skipped)
	}

	switch typeName {
	case "number":
		return s.ValidateNumberConversion(val)
	case "decimal":
		return s.ValidateDecimalConversion(val)
	case "boolean":
		return s.ValidateBooleanConversion(val)
	case "date":
		return s.ValidateDateConversion(val)
	case "email":
		return s.ValidateEmailConversion(val)
	}
	return "" // Other types pass through
}

// ValidateNumberConversion checks if value is a valid number
func (s *importService) ValidateNumberConversion(val string) string {
	if _, err := strconv.ParseInt(val, 10, 64); err == nil {
		return "" // Valid integer
	}
	if _, err := strconv.ParseFloat(val, 64); err == nil {
		return "" // Valid float
	}
	return fmt.Sprintf("Column type 'number' expects numeric value, got '%s'", val)
}

// ValidateDecimalConversion checks if value is a valid decimal
func (s *importService) ValidateDecimalConversion(val string) string {
	if _, err := strconv.ParseFloat(val, 64); err != nil {
		return fmt.Sprintf("Column type 'decimal' expects numeric value, got '%s'", val)
	}
	return ""
}

// ValidateBooleanConversion checks if value is a valid boolean
func (s *importService) ValidateBooleanConversion(val string) string {
	lower := strings.ToLower(val)
	validBools := []string{"true", "false", "1", "0", "yes", "no"}
	for _, b := range validBools {
		if lower == b {
			return "" // Valid boolean
		}
	}
	return fmt.Sprintf("Column type 'boolean' expects true/false/yes/no/1/0, got '%s'", val)
}

// ValidateDateConversion checks if value is a valid date
func (s *importService) ValidateDateConversion(val string) string {
	formats := []string{
		isoDateFormat,
		ddmmyyyyDateFormat,
		yyyymmddSlashFormat,
		ddmmyyyySlashFormat,
	}
	for _, format := range formats {
		if _, err := time.Parse(format, val); err == nil {
			return "" // Valid date format
		}
	}
	return fmt.Sprintf("Column type 'date' cannot parse '%s' (expected YYYY-MM-DD, DD-MM-YYYY, or similar)", val)
}

// ValidateEmailConversion checks if value is a valid email
func (s *importService) ValidateEmailConversion(val string) string {
	if !strings.Contains(val, "@") {
		return fmt.Sprintf("Column type 'email' expects valid email format, got '%s'", val)
	}
	return ""
}
