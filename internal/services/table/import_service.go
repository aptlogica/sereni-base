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
	isoDateFormat = "2006-01-02"
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

	if err := s.scanFile(ctx, file, lg); err != nil {
		return dto.ImportTableResponse{}, err
	}
	// Track whether we auto-created a base/table so we can clean up on partial failures
	createdBase := false
	if req.BaseID == "" {
		if err := s.ensureBaseWithConfig(ctx, schemaName, &req, lg, tableTitle); err != nil {
			return dto.ImportTableResponse{}, err
		}
		createdBase = true
	}
	headers, dataRows, err := s.parseCSV(file, lg)
	if err != nil {
		return dto.ImportTableResponse{}, err
	}

	// Initialize statistics
	stats := &dto.ImportStatistics{}

	// Validate column config
	if err := s.validateColumnConfig(req.Config.Columns, req.Config.PrimaryColumn, headers, lg); err != nil {
		return dto.ImportTableResponse{}, err
	}

	// Step 1: Apply TrimSpaces setting (if enabled)
	if req.Config.Settings.TrimSpaces {
		dataRows = s.cleanData(dataRows, req.Config.Settings)
		lg.Info().Bool("trimSpaces", true).Msg("Trim spaces applied")
	} else {
		lg.Info().Bool("trimSpaces", false).Msg("Trim spaces skipped")
	}

	// Step 2: Count and optionally remove empty rows (after trimming for accurate detection)
	emptyRowCount := s.countEmptyRows(dataRows)
	stats.EmptyRows = emptyRowCount
	if req.Config.Settings.RemoveEmptyRows {
		dataRows = s.removeEmptyRows(dataRows)
		stats.EmptyRowsSkipped = emptyRowCount
		lg.Info().Bool("removeEmptyRows", true).Int("emptyRowsDetected", emptyRowCount).Int("emptyRowsRemoved", emptyRowCount).Msg("Empty rows removed")
	} else {
		stats.EmptyRowsSkipped = 0
		lg.Info().Bool("removeEmptyRows", false).Int("emptyRowsDetected", emptyRowCount).Msg("Empty rows detected but not removed")
	}

	// Step 3: Count and optionally remove duplicates
	duplicateRowCount := s.countDuplicateRecords(dataRows)
	stats.DuplicateRows = duplicateRowCount
	if req.Config.Settings.RemoveDuplicateRecords {
		dataRows = s.removeDuplicateRecords(dataRows)
		stats.DuplicatesRemoved = duplicateRowCount
		lg.Info().Bool("removeDuplicates", true).Int("duplicatesDetected", duplicateRowCount).Int("duplicatesRemoved", duplicateRowCount).Int("recordsAfterDedup", len(dataRows)).Msg("Duplicate records removed")
	} else {
		stats.DuplicatesRemoved = 0
		lg.Info().Bool("removeDuplicates", false).Int("duplicatesDetected", duplicateRowCount).Msg("Duplicate records detected but not removed")
	}

	// Always log and save cleaned data to JSON file (regardless of settings)
	s.logCleanedDataWithSettings(headers, dataRows, stats, req.Config.Settings, lg)

	// Check for duplicate table names and get unique name if needed
	uniqueTableTitle := tableTitle
	uniqueTableTitle, err = s.getUniqueTableName(ctx, schemaName, req.BaseID, tableTitle, lg)
	if err != nil {
		lg.Error().Stack().Err(err).Str("tableName", tableTitle).Msg("Failed to check for duplicate table names")
		// Continue with original name if check fails
	}

	// Create table
	createTableReq := dto.CreateTableRequest{
		BaseID:      req.BaseID,
		WorkspaceID: req.WorkspaceID,
		Title:       uniqueTableTitle,
		Description: req.Description,
		OrderIndex:  req.OrderIndex,
		CreatedBy:   req.CreatedBy,
	}

	tableResp, err := s.createTable(ctx, schemaName, createTableReq, lg, uniqueTableTitle)
	if err != nil {
		if createdBase {
			if delErr := s.baseManagementService.DeleteBase(ctx, schemaName, req.BaseID); delErr != nil {
				lg.Error().Stack().Err(delErr).Str("baseID", req.BaseID).Msg("Failed to cleanup auto-created base after table creation failure")
			}
		}
		return dto.ImportTableResponse{}, err
	}

	createdTable := true

	cleanup := func() {
		if createdTable {
			if delErr := s.tableService.DeleteTable(ctx, schemaName, tableResp.Model.ID.String()); delErr != nil {
				lg.Error().Stack().Err(delErr).Str("tableID", tableResp.Model.ID.String()).Msg("Failed to cleanup created table after import error")
			}
		}
		if createdBase {
			if delErr := s.baseManagementService.DeleteBase(ctx, schemaName, req.BaseID); delErr != nil {
				lg.Error().Stack().Err(delErr).Str("baseID", req.BaseID).Msg("Failed to cleanup auto-created base after import error")
			}
		}
	}

	// Add columns with config
	columnMap, err := s.addColumnsWithConfig(ctx, schemaName, createTableReq, headers, req.Config.Columns, req.Config.PrimaryColumn, tableResp, lg)
	if err != nil {
		cleanup()
		return dto.ImportTableResponse{}, err
	}

	// Build records using config column types with error tracking
	newRecords, errorRows, errorMessages := s.buildRecordsWithConfigAndErrors(dataRows, req.Config.Columns, req.Config.PrimaryColumn, columnMap, createTableReq, headers, req.Config.Settings, lg)
	lg.Info().Int("recordsCreated", len(newRecords)).Int("errorRows", len(errorRows)).Msg("Records prepared for insertion with config and error tracking")

	// Always identify empty and duplicate rows for logging (regardless of settings)
	emptyRowsWithLineNumbers := s.identifyEmptyRowsWithLineNumbers(dataRows)
	duplicateRowsWithLineNumbers := s.identifyDuplicateRowsWithLineNumbers(dataRows)

	// Always generate error report with import details
	var errorRowsFileContent string
	errorRowsFileContent, err = s.saveErrorRows(headers, errorRows, errorMessages, emptyRowsWithLineNumbers, duplicateRowsWithLineNumbers, lg)
	if err != nil {
		lg.Warn().Err(err).Msg("Failed to generate error report, continuing with import")
		// Don't fail the import if we can't generate error report, just log it
	}

	stats.TotalRows = len(newRecords)
	stats.TotalColumns = len(req.Config.Columns)
	stats.ErrorRows = len(errorRows)
	stats.ErrorRowsFileContent = errorRowsFileContent

	// if err := s.insertBatches(ctx, schemaName, tableResp, newRecords, lg); err != nil {
	// 	cleanup()
	// 	return dto.ImportTableResponse{}, err
	// }
	// Insert batches with database error handling - skip failed rows and continue
	dbErrorRows, dbErrorMessages := s.insertBatchesWithErrorHandling(ctx, schemaName, tableResp, newRecords, headers, lg)

	// Add database errors to statistics and error rows
	if len(dbErrorRows) > 0 {
		stats.ErrorRows += len(dbErrorRows)

		// Append database errors to error report
		if stats.ErrorRowsFileContent != "" {
			stats.ErrorRowsFileContent += "\n\n" +
				strings.Repeat("=", 100) + "\n" +
				"DATABASE ERRORS (Failed to insert into database)\n" +
				strings.Repeat("=", 100) + "\n\n"
		}

		for _, dbErrMsg := range dbErrorMessages {
			stats.ErrorRowsFileContent += dbErrMsg + "\n\n"
		}

		lg.Warn().Int("dbErrorRowCount", len(dbErrorRows)).Msg("Some rows failed due to database errors - import continued with remaining rows")
	}

	finalTableResp, err := s.refreshTable(ctx, schemaName, tableResp, lg)
	if err != nil {
		return dto.ImportTableResponse{ImportStats: stats, TableModelViewResponse: dto.TableModelViewResponse{
			Model: tableResp.Model,
			Views: tableResp.Views,
		}}, nil
	}

	// Log final statistics
	// lg.Info().
	// 	Int("totalRows", stats.TotalRows).
	// 	Int("totalColumns", stats.TotalColumns).
	// 	Int("errorRows", stats.ErrorRows).
	// 	Int("emptyRows", stats.EmptyRows).
	// 	Int("duplicateRows", stats.DuplicateRows).
	// 	Int("emptyRowsSkipped", stats.EmptyRowsSkipped).
	// 	Int("duplicatesRemoved", stats.DuplicatesRemoved).
	// 	Str("errorRowsFileContent", stats.ErrorRowsFileContent).
	// 	Msg("Import completed with statistics")

	return dto.ImportTableResponse{ImportStats: stats, TableModelViewResponse: dto.TableModelViewResponse{
		Model: finalTableResp.Model,
		Views: finalTableResp.Views,
	}}, nil
}

func (s *importService) scanFile(ctx context.Context, file *multipart.FileHeader, lg *zerolog.Logger) error {
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

func (s *importService) ensureBase(ctx context.Context, schemaName string, req *dto.CreateTableRequest, lg *zerolog.Logger, tableTitle string) error {
	if req.BaseID != "" {
		return nil
	}

	if req.WorkspaceID == "" {
		lg.Error().Msg("workspace_id is required when base_id is not provided")
		return fmt.Errorf("workspace_id is required when base_id is not provided")
	}

	baseName := tableTitle + "_base"
	lg.Info().Str("baseName", baseName).Str("workspaceID", req.WorkspaceID).Msg("Creating new base for import")

	createBaseReq := dto.CreateBaseRequest{
		Title:       baseName,
		Description: helpers.StringPtr("Auto-created base for table import"),
		WorkspaceID: req.WorkspaceID,
		CreatedBy:   req.CreatedBy,
	}

	newBase, err := s.baseManagementService.CreateBaseWithoutTable(ctx, createBaseReq, schemaName, req.CreatedBy)
	if err != nil {
		lg.Error().Stack().Err(err).Str("baseName", baseName).Msg("Failed to create base for import")
		return fmt.Errorf("failed to create base: %w", err)
	}

	req.BaseID = newBase.ID.String()
	lg.Info().Str("baseID", req.BaseID).Str("baseName", baseName).Msg("Base created successfully for import")
	return nil
}

func (s *importService) parseCSV(file *multipart.FileHeader, lg *zerolog.Logger) ([]string, [][]string, error) {
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

func (s *importService) createTable(ctx context.Context, schemaName string, req dto.CreateTableRequest, lg *zerolog.Logger, tableTitle string) (dto.TableResponse, error) {
	lg.Info().Str("tableName", tableTitle).Str("schemaName", schemaName).Msg("Creating table with defaults")
	tableResp, err := s.tableService.CreateTableWithDefaults(ctx, req, schemaName)
	if err != nil {
		lg.Error().Stack().Err(err).Str("tableName", tableTitle).Str("schemaName", schemaName).Msg("Failed to create table with defaults")
		return dto.TableResponse{}, err
	}
	lg.Info().Str("tableID", tableResp.Model.ID.String()).Msg("Table created successfully")
	return tableResp, nil
}

func (s *importService) updateTitleColumn(ctx context.Context, schemaName string, lg *zerolog.Logger, titleColumnName string, columnType string, tableResp *dto.TableResponse) error {
	if titleColumnName == "" {
		return nil
	}

	titleColumnID := ""
	for _, col := range tableResp.Columns {
		if col.Title == "Title" {
			titleColumnID = col.ID.String()
			break
		}
	}

	if titleColumnID == "" {
		return nil
	}

	lg.Info().Str("columnID", titleColumnID).Str("newTitle", titleColumnName).Str("columnType", columnType).Msg("Updating title column name and type")

	// Get the database type for the inferred column type
	dt := s.getDatabaseType(columnType)

	updateColReq := dto.ColumnUpdate{
		Title: &titleColumnName,
		UIDT:  &columnType,
		DT:    &dt,
	}
	if _, err := s.tableService.UpdateColumn(ctx, schemaName, titleColumnID, updateColReq); err != nil {
		lg.Error().Stack().Err(err).Str("columnID", titleColumnID).Str("newTitle", titleColumnName).Msg("Failed to update title column")
		return err
	}

	lg.Info().Str("columnID", titleColumnID).Msg("Title column updated successfully")
	return nil
}

func (s *importService) addColumns(ctx context.Context, schemaName string, req dto.CreateTableRequest, headers []string, columnTypes []string, tableResp dto.TableResponse, lg *zerolog.Logger) (map[int]dto.ColumnResponse, error) {
	lg.Info().Int("columnCount", len(headers)-1).Msg("Starting to add columns")
	columnMap := make(map[int]dto.ColumnResponse)
	systemFieldAdded := false

	for i, header := range headers {
		if i == 0 {
			continue
		}

		colType := columnTypes[i]
		if colType == "text" && !systemFieldAdded {
			systemFieldAdded = true
		}

		if header == "" {
			continue
		}

		addColReq := dto.AddColumnRequest{
			ModelID:     tableResp.Model.ID,
			BaseID:      tableResp.Model.BaseID,
			Title:       header,
			Description: "",
			Meta:        map[string]interface{}{},
			UIDT:        colType,
			DT:          s.getDatabaseType(colType),
			OrderIndex:  helpers.Float64Ptr(float64(i + 6)),
			Virtual:     helpers.BoolPtr(false),
			System:      helpers.BoolPtr(false),
			CreatedBy:   req.CreatedBy,
		}
		colResp, err := s.tableService.AddColumn(ctx, schemaName, addColReq)
		if err != nil {
			lg.Error().Stack().Err(err).Str("columnTitle", header).Str("columnType", colType).Msg("Failed to add column")
			return nil, err
		}
		columnMap[i] = colResp
		lg.Debug().Str("columnTitle", header).Str("columnType", colType).Msg("Column added successfully")
	}

	lg.Info().Int("columnsAdded", len(columnMap)).Msg("All columns added")
	return columnMap, nil
}

func (s *importService) buildRecords(dataRows [][]string, columnTypes []string, columnMap map[int]dto.ColumnResponse, req dto.CreateTableRequest, lg *zerolog.Logger) []map[string]interface{} {
	lg.Info().Int("recordCount", len(dataRows)).Msg("Starting to insert records")

	newRecords := []map[string]interface{}{}
	for _, row := range dataRows {
		record := map[string]interface{}{
			"created_by":         req.CreatedBy,
			"last_modified_by":   req.CreatedBy,
			"created_time":       time.Now().UTC(),
			"last_modified_time": time.Now().UTC(),
		}

		for i, cellVal := range row {
			if cellVal == "" {
				continue
			}

			if i == 0 {
				val := s.convertValue(cellVal, columnTypes[i])
				record["title"] = val
				continue
			}

			colResp, exists := columnMap[i]
			if !exists {
				continue
			}
			val := s.convertValue(cellVal, columnTypes[i])
			record[fmt.Sprintf("\"%s\"", colResp.ColumnName)] = val
		}
		newRecords = append(newRecords, record)
	}

	return newRecords
}

func (s *importService) insertBatchesWithErrorHandling(ctx context.Context, schemaName string, tableResp dto.TableResponse, newRecords []map[string]interface{}, headers []string, lg *zerolog.Logger) ([][]string, []string) {
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
			for j := 0; j < len(batch); j++ {
				rowIndex := startRowIndex + j
				lineNumber := rowIndex + 2 // Line numbers start from 2 (row 1 is header)
				singleRowBatch := []map[string]interface{}{batch[j]}

				if _, err := s.tableService.CreateRowsWithRecordsBulk(ctx, schemaName, tableResp.Model.Alias, singleRowBatch); err != nil {
					// This specific row failed
					lg.Error().Stack().Err(err).Int("batchNumber", batchNum).Int("lineNumber", lineNumber).Int("rowIndex", rowIndex).Msg("Row failed - skipping")

					// Create error message with specific row line number
					failureMsg := fmt.Sprintf("[Database Error %d] Batch %d, Row Line %d (CSV Line %d)\n", len(failedRows)+1, batchNum, lineNumber, lineNumber)
					failureMsg += fmt.Sprintf("Error: %v\n", err)
					failureMsg += fmt.Sprintf("Record: %v\n", batch[j])

					errorMessages = append(errorMessages, failureMsg)

					// Create a representation of the failed row
					failedRowData := make([]string, len(headers))
					for k := range failedRowData {
						failedRowData[k] = "[FAILED_TO_INSERT]"
					}
					failedRows = append(failedRows, failedRowData)
				} else {
					lg.Debug().Int("batchNumber", batchNum).Int("lineNumber", lineNumber).Msg("Row inserted successfully")
				}
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

func (s *importService) refreshTable(ctx context.Context, schemaName string, tableResp dto.TableResponse, lg *zerolog.Logger) (dto.TableResponse, error) {
	lg.Info().Str("tableID", tableResp.Model.ID.String()).Msg("Refreshing table response")
	finalTableResp, err := s.tableService.GetTableByID(ctx, tableResp.Model.ID.String(), schemaName)
	if err != nil {
		lg.Warn().Stack().Err(err).Str("tableID", tableResp.Model.ID.String()).Msg("Failed to refresh table response, returning cached response")
		return tableResp, err
	}

	lg.Info().Str("tableID", finalTableResp.Model.ID.String()).Int("columns", len(finalTableResp.Columns)).Int("records", len(finalTableResp.Records)).Msg("Import completed successfully")
	return finalTableResp, nil
}

func (s *importService) inferColumnTypes(headers []string, rows [][]string) []string {
	types := make([]string, len(headers))
	for i := range headers {
		types[i] = s.inferType(rows, i)
	}
	return types
}

func (s *importService) inferType(rows [][]string, colIndex int) string {
	flags := s.collectTypeFlags(rows, colIndex)
	if !flags.hasData {
		return "text"
	}
	return s.determineTypeFromFlags(flags)
}

type typeFlags struct {
	isNumber, isDecimal, isBool, isDate, isEmail, isURL, isPhone, isJSON bool
	hasData                                                              bool
	totalLength, count                                                   int
}

func (s *importService) collectTypeFlags(rows [][]string, colIndex int) typeFlags {
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
		s.updateTypeFlags(&flags, val)
	}
	return flags
}

// updateTypeFlags updates the type flags based on a single value
func (s *importService) updateTypeFlags(flags *typeFlags, val string) {
	if flags.isNumber || flags.isDecimal {
		flags.isNumber, flags.isDecimal = s.checkNumericTypes(val, flags.isNumber, flags.isDecimal)
	}
	if flags.isBool {
		flags.isBool = s.checkBoolType(val)
	}
	if flags.isDate {
		flags.isDate = s.checkDateType(val)
	}
	if flags.isEmail {
		flags.isEmail = s.checkEmailType(val)
	}
	if flags.isURL {
		flags.isURL = s.checkURLType(val)
	}
	if flags.isPhone {
		flags.isPhone = s.checkPhoneType(val)
	}
	if flags.isJSON {
		flags.isJSON = s.checkJSONType(val)
	}
}

func (s *importService) checkNumericTypes(val string, isNumber, isDecimal bool) (bool, bool) {
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

func (s *importService) checkBoolType(val string) bool {
	lower := strings.ToLower(val)
	return lower == "true" || lower == "false" || lower == "0" || lower == "1" || lower == "yes" || lower == "no"
}

func (s *importService) checkDateType(val string) bool {
	formats := []string{isoDateFormat, "02-01-2006", "2006/01/02", "02/01/2006"}
	for _, f := range formats {
		if _, err := time.Parse(f, val); err == nil {
			return true
		}
	}
	return false
}

func (s *importService) checkEmailType(val string) bool {
	return strings.Contains(val, "@") && strings.Contains(val, ".")
}

func (s *importService) checkURLType(val string) bool {
	return strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://")
}

func (s *importService) checkPhoneType(val string) bool {
	// Simple check: contains only digits, spaces, dashes, parentheses, plus
	for _, r := range val {
		if !((r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '(' || r == ')' || r == '+') {
			return false
		}
	}
	return len(val) > 0
}

func (s *importService) checkJSONType(val string) bool {
	var js interface{}
	return json.Unmarshal([]byte(val), &js) == nil
}

func (s *importService) determineTypeFromFlags(flags typeFlags) string {
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

func (s *importService) getDatabaseType(uidt string) string {
	if mapping, exists := constant.UITypeMappings[uidt]; exists {
		return mapping.Postgres
	}
	// Default to TEXT
	return "TEXT"
}

func (s *importService) convertValue(val string, typeName string) interface{} {
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
		return s.convertDateToISO(val)
	case "email", "url", "phoneNumber", "json":
		// These are stored as text
		return val
	}
	return val
}

// convertDateToISO converts date strings from various formats to ISO format (YYYY-MM-DD)
func (s *importService) convertDateToISO(val string) string {
	formats := []string{
		isoDateFormat, // Already ISO format
		"02-01-2006",  // DD-MM-YYYY
		"2006/01/02",  // YYYY/MM/DD
		"02/01/2006",  // DD/MM/YYYY
	}

	for _, format := range formats {
		if parsedTime, err := time.Parse(format, val); err == nil {
			// Convert to ISO format (YYYY-MM-DD)
			return parsedTime.Format(isoDateFormat)
		}
	}
	return val
}

// findUniqueName finds a unique name from a list of existing names by appending numbers if needed
// Enforces the specified character limit by truncating the base name
func (s *importService) findUniqueName(proposedName string, existingNames []string, maxLength int) string {
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

// getUniqueTableName checks if a table with the given name exists in the schema
// If it does, appends a number (1, 2, 3, etc.) to make it unique
// Enforces 50 character limit for table names
func (s *importService) getUniqueTableName(ctx context.Context, schemaName string, baseID string, proposedName string, lg *zerolog.Logger) (string, error) {
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
	uniqueName := s.findUniqueName(proposedName, existingTableNames, maxTableNameLength)

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

// getUniqueBaseName checks if a base with the given name exists in the workspace
// If it does, appends a number (1, 2, 3, etc.) to make it unique
// Enforces 50 character limit for base names
func (s *importService) getUniqueBaseName(ctx context.Context, schemaName string, workspaceID string, proposedName string, lg *zerolog.Logger) (string, error) {
	allBases, err := s.baseManagementService.GetBasesByWorkspace(ctx, schemaName, workspaceID)
	if err != nil {
		lg.Error().Stack().Err(err).Str("workspaceID", workspaceID).Msg("Failed to fetch existing bases")
		return proposedName, err
	}

	// Remove "_base" suffix if present to get the clean base name
	cleanBaseName := proposedName
	if strings.HasSuffix(cleanBaseName, "_base") {
		cleanBaseName = strings.TrimSuffix(cleanBaseName, "_base")
	}

	// Extract base titles into a slice
	existingBaseNames := make([]string, len(allBases))
	for i, base := range allBases {
		existingBaseNames[i] = base.Title
	}

	const maxNameLength = 50

	// Find unique name using helper
	uniqueName := s.findUniqueName(cleanBaseName, existingBaseNames, maxNameLength)

	if uniqueName == cleanBaseName && len(proposedName) > maxNameLength {
		lg.Warn().Str("baseName", cleanBaseName).Int("length", len(cleanBaseName)).Int("maxLength", maxNameLength).Msg("Base name exceeds 50 character limit, truncating")
	} else if uniqueName != cleanBaseName {
		lg.Info().Str("originalName", proposedName).Str("uniqueName", uniqueName).Int("length", len(uniqueName)).Msg("Base name already exists, using unique name with 50 char limit")
	}

	return uniqueName, nil
}

// ensureBaseWithConfig ensures a base exists for config-based import
func (s *importService) ensureBaseWithConfig(ctx context.Context, schemaName string, req *dto.ImportWithConfigRequest, lg *zerolog.Logger, tableTitle string) error {
	if req.BaseID != "" {
		return nil
	}

	if req.WorkspaceID == "" {
		lg.Error().Msg("workspace_id is required when base_id is not provided")
		return fmt.Errorf("workspace_id is required when base_id is not provided")
	}

	baseName := tableTitle

	// Check for duplicate base names and get unique name if needed (with 50 char limit)
	uniqueBaseName, err := s.getUniqueBaseName(ctx, schemaName, req.WorkspaceID, baseName, lg)
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
		Description: helpers.StringPtr("Auto-created base for table import"),
		WorkspaceID: req.WorkspaceID,
		CreatedBy:   req.CreatedBy,
	}

	newBase, err := s.baseManagementService.CreateBaseWithoutTable(ctx, createBaseReq, schemaName, req.CreatedBy)
	if err != nil {
		lg.Error().Stack().Err(err).Str("baseName", uniqueBaseName).Msg("Failed to create base for import")
		return fmt.Errorf("failed to create base: %w", err)
	}

	req.BaseID = newBase.ID.String()
	lg.Info().Str("baseID", req.BaseID).Str("baseName", uniqueBaseName).Msg("Base created successfully for import")
	return nil
}

// cleanData applies data cleaning transformations (trim, remove extra spaces)
func (s *importService) cleanData(rows [][]string, settings dto.ImportSettings) [][]string {
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

// removeDuplicateRecords removes duplicate rows while preserving order
func (s *importService) removeDuplicateRecords(rows [][]string) [][]string {
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

// addColumnsWithConfig adds columns to the table using user-provided config
func (s *importService) addColumnsWithConfig(ctx context.Context, schemaName string, req dto.CreateTableRequest, headers []string, columnConfigs []dto.ColumnConfig, primary *dto.ColumnConfig, tableResp dto.TableResponse, lg *zerolog.Logger) (map[int]dto.ColumnResponse, error) {
	lg.Info().Int("columnCount", len(columnConfigs)).Msg("Starting to add columns with config")
	columnMap := make(map[int]dto.ColumnResponse)

	// Build a map of source names to configs for quick lookup
	configMap := make(map[string]dto.ColumnConfig)
	for _, cfg := range columnConfigs {
		configMap[cfg.ColumnName] = cfg
	}

	for i, header := range headers {
		if header == "" {
			continue
		}

		// Use column config for the header, overriding with primary column config when matched.
		cfg, exists := configMap[header]
		primaryMatch := false
		if primary != nil && primary.ColumnName == header {
			primaryMatch = true
			// Use primary config as the effective cfg (overrides any entry)
			cfg = *primary
			exists = true
		}

		if !exists {
			// Skip columns not in config (frontend controls what config is sent)
			lg.Debug().Str("columnName", header).Msg("Skipping column not in config")
			continue
		}

		// only for the configured primary column as the default Title column.
		isTitleColumn := primaryMatch
		if isTitleColumn {
			// find the existing Title column id
			titleColID := ""
			for _, c := range tableResp.Columns {
				if c.Title == "Title" {
					titleColID = c.ID.String()
					break
				}
			}
			if titleColID == "" {
				lg.Warn().Msg("Title column not found to update with config")
			} else {
				// prepare update with meta/type/title
				colType := cfg.UIDT
				if colType == "" {
					colType = "text"
				}
				colDT := s.getDatabaseType(colType)

				meta := map[string]interface{}{}
				if cfg.Meta != nil {
					meta = cfg.Meta
				}

				// ColumnUpdate expects *map[string]interface{} for Meta
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
				if _, err := s.tableService.UpdateColumn(ctx, schemaName, titleColID, updateReq); err != nil {
					lg.Error().Stack().Err(err).Str("column", titleColID).Msg("Failed to update Title column with config meta")
					return nil, err
				}
			}
			continue
		}

		// Use provided field type or default to text
		colType := cfg.UIDT
		if colType == "" {
			colType = "text"
		}

		// Use provided title or default to source name
		colTitle := cfg.Title
		if colTitle == "" {
			colTitle = header
		}

		// Compute DT from UIDT (database type auto-generated from UI type)
		colDT := s.getDatabaseType(colType)

		// Use provided meta if present
		meta := map[string]interface{}{}
		if cfg.Meta != nil {
			meta = cfg.Meta
		}

		// Default values are expected to come from meta["default_value"]
		addColReq := dto.AddColumnRequest{
			ModelID:     tableResp.Model.ID,
			BaseID:      tableResp.Model.BaseID,
			Title:       colTitle,
			Description: "",
			Meta:        meta,
			UIDT:        colType,
			DT:          colDT,
			OrderIndex:  helpers.Float64Ptr(float64(i + 6)),
			Virtual:     helpers.BoolPtr(false),
			System:      helpers.BoolPtr(false),
			CreatedBy:   req.CreatedBy,
		}

		colResp, err := s.tableService.AddColumn(ctx, schemaName, addColReq)
		if err != nil {
			lg.Error().Stack().Err(err).Str("columnTitle", colTitle).Str("columnType", colType).Msg("Failed to add column with config")
			return nil, err
		}

		columnMap[i] = colResp
		lg.Debug().Str("columnTitle", colTitle).Str("columnType", colType).Msg("Column added with config")
	}

	lg.Info().Int("columnsAdded", len(columnMap)).Msg("All columns added with config")
	return columnMap, nil
}

// buildRecordsWithConfig builds records using column configuration
func (s *importService) buildRecordsWithConfig(dataRows [][]string, columnConfigs []dto.ColumnConfig, primary *dto.ColumnConfig, columnMap map[int]dto.ColumnResponse, req dto.CreateTableRequest, headers []string, lg *zerolog.Logger) []map[string]interface{} {
	lg.Info().Int("recordCount", len(dataRows)).Msg("Starting to insert records with config")

	// Build config map for quick lookup
	configMap := make(map[string]dto.ColumnConfig)
	for _, cfg := range columnConfigs {
		configMap[cfg.ColumnName] = cfg
	}

	newRecords := []map[string]interface{}{}

	for _, row := range dataRows {
		record := map[string]interface{}{
			"created_by":         req.CreatedBy,
			"last_modified_by":   req.CreatedBy,
			"created_time":       time.Now().UTC(),
			"last_modified_time": time.Now().UTC(),
		}

		for i, cellVal := range row {
			if i >= len(headers) {
				break
			}

			header := headers[i]
			cfg, configExists := configMap[header]

			// Determine if this header is the designated primary/title column.
			primaryMatch := false
			if primary != nil && primary.ColumnName == header {
				primaryMatch = true
				cfg = *primary
				configExists = true
			}

			// Handle title column (only if primary explicitly matches this header)
			if primaryMatch {
				// title column: prefer default from meta if present
				var defaultVal string
				if cfg.Meta != nil {
					if dv, ok := cfg.Meta["default_value"]; ok {
						if sVal, ok2 := dv.(string); ok2 {
							defaultVal = sVal
						}
					}
				}

				if cellVal == "" && defaultVal != "" {
					record["title"] = s.convertValue(defaultVal, cfg.UIDT)
				} else if cellVal != "" {
					record["title"] = s.convertValue(cellVal, cfg.UIDT)
				}
				continue
			}

			// Skip if column not in config (frontend controls what config is sent)
			if !configExists {
				continue
			}

			// Skip if no column map entry (shouldn't happen)
			colResp, colExists := columnMap[i]
			if !colExists {
				continue
			}

			// Use cell value or default value
			var val interface{}
			// Determine default value: prefer meta["default_value"] then cfg.DefaultValue
			var defaultVal string
			if cfg.Meta != nil {
				if dv, ok := cfg.Meta["default_value"]; ok {
					if sVal, ok2 := dv.(string); ok2 {
						defaultVal = sVal
					}
				}
			}

			if cellVal == "" && defaultVal != "" {
				val = s.convertValue(defaultVal, cfg.UIDT)
			} else if cellVal != "" {
				val = s.convertValue(cellVal, cfg.UIDT)
			} else {
				continue // Skip empty cells with no default
			}

			record[fmt.Sprintf("\"%s\"", colResp.ColumnName)] = val
		}

		newRecords = append(newRecords, record)
	}

	return newRecords
}

// saveErrorRows writes error rows, empty rows, and duplicate rows to a text log file
func (s *importService) saveErrorRows(headers []string, errorRows [][]string, errorMessages []string, emptyRowsWithLineNumbers map[int][]string, duplicateRowsWithLineNumbers map[int][]string, lg *zerolog.Logger) (string, error) {
	// Create tmp directory if it doesn't exist
	tmpDir := "./internal/tmp"
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			lg.Warn().Err(err).Str("dir", tmpDir).Msg("Failed to create tmp directory for error rows, continuing without file save")
			// Don't fail - continue with content in memory
		}
	} else if err != nil {
		lg.Warn().Err(err).Str("dir", tmpDir).Msg("Error checking tmp directory, continuing without file save")
		// Continue anyway
	} else {
		// Directory exists, try to set proper permissions
		if err := os.Chmod(tmpDir, 0755); err != nil {
			lg.Warn().Err(err).Str("dir", tmpDir).Msg("Failed to set directory permissions, continuing anyway")
		}
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	errorFile := fmt.Sprintf("%s/import_error_rows_%s.txt", tmpDir, timestamp)

	// Build the content
	var content strings.Builder
	content.WriteString("Import Issues Report\n")
	content.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(strings.Repeat("=", 100))
	content.WriteString("\n\n")

	// Write summary
	content.WriteString("SUMMARY:\n")
	content.WriteString(fmt.Sprintf("Total Error Rows: %d\n", len(errorRows)))
	content.WriteString(fmt.Sprintf("Total Empty Rows: %d\n", len(emptyRowsWithLineNumbers)))
	content.WriteString(fmt.Sprintf("Total Duplicate Rows: %d\n", len(duplicateRowsWithLineNumbers)))

	if len(errorRows) == 0 && len(emptyRowsWithLineNumbers) == 0 && len(duplicateRowsWithLineNumbers) == 0 {
		content.WriteString("\nStatus: ✓ No issues detected - All rows are valid\n")
	}
	content.WriteString(strings.Repeat("-", 100))
	content.WriteString("\n\n")

	// Write headers
	content.WriteString("CSV Headers:\n")
	content.WriteString(strings.Join(headers, " | "))
	content.WriteString("\n\n")

	// Write ALL ERRORS section FIRST - comprehensive error details
	if len(errorRows) > 0 {
		content.WriteString(strings.Repeat("=", 100))
		content.WriteString("\n")
		content.WriteString("ALL VALIDATION ERRORS (Detailed Error Analysis)\n")
		content.WriteString(strings.Repeat("=", 100))
		content.WriteString("\n\n")

		for i, errorMsg := range errorMessages {
			content.WriteString(fmt.Sprintf("[Error Set %d]\n", i+1))
			content.WriteString(fmt.Sprintf("%s\n", errorMsg))
			content.WriteString("\n")
		}
		content.WriteString(strings.Repeat("-", 100))
		content.WriteString("\n\n")

		// Write ERROR TYPE SUMMARY - Only CSV validation errors
		content.WriteString(strings.Repeat("=", 100))
		content.WriteString("\n")
		content.WriteString("CSV VALIDATION ERROR TYPES\n")
		content.WriteString(strings.Repeat("=", 100))
		content.WriteString("\n\n")

		errorTypeCount := make(map[string]int)
		for _, errMsg := range errorMessages {
			// Type format errors
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
			content.WriteString(fmt.Sprintf("  • %s: %d errors\n", errType, errorTypeCount[errType]))
		}
		content.WriteString("\n")
		content.WriteString(strings.Repeat("-", 100))
		content.WriteString("\n\n")
	}

	// Write EMPTY ROWS section
	if len(emptyRowsWithLineNumbers) > 0 {
		content.WriteString(strings.Repeat("=", 100))
		content.WriteString("\n")
		content.WriteString("EMPTY ROWS (All cells are empty)\n")
		content.WriteString(strings.Repeat("=", 100))
		content.WriteString("\n\n")

		lineNumbers := make([]int, 0, len(emptyRowsWithLineNumbers))
		for lineNum := range emptyRowsWithLineNumbers {
			lineNumbers = append(lineNumbers, lineNum)
		}
		sort.Ints(lineNumbers)

		for idx, lineNum := range lineNumbers {
			row := emptyRowsWithLineNumbers[lineNum]
			content.WriteString(fmt.Sprintf("[Empty Row %d] Line %d in CSV file\n", idx+1, lineNum))
			content.WriteString(fmt.Sprintf("Row Data: %v\n", row))
			content.WriteString("\n")
		}
	}

	// Write DUPLICATE ROWS section
	if len(duplicateRowsWithLineNumbers) > 0 {
		content.WriteString(strings.Repeat("=", 100))
		content.WriteString("\n")
		content.WriteString("DUPLICATE ROWS (Identical rows found)\n")
		content.WriteString(strings.Repeat("=", 100))
		content.WriteString("\n\n")

		lineNumbers := make([]int, 0, len(duplicateRowsWithLineNumbers))
		for lineNum := range duplicateRowsWithLineNumbers {
			lineNumbers = append(lineNumbers, lineNum)
		}
		sort.Ints(lineNumbers)

		for idx, lineNum := range lineNumbers {
			row := duplicateRowsWithLineNumbers[lineNum]
			content.WriteString(fmt.Sprintf("[Duplicate Row %d] Line %d in CSV file\n", idx+1, lineNum))
			content.WriteString(fmt.Sprintf("Row Data: %v\n", row))
			content.WriteString("\n")
		}
	}

	// Write CSV format section
	content.WriteString(strings.Repeat("=", 100))
	content.WriteString("\n")
	content.WriteString("RAW DATA (CSV Format)\n")
	content.WriteString(strings.Repeat("=", 100))
	content.WriteString("\n\n")

	// Headers
	content.WriteString(strings.Join(headers, ","))
	content.WriteString("\n")

	// Error rows in CSV format
	if len(errorRows) > 0 {
		content.WriteString("# ERROR ROWS:\n")
		for _, row := range errorRows {
			escapedRow := make([]string, len(row))
			for j, cell := range row {
				if strings.Contains(cell, ",") || strings.Contains(cell, "\"") || strings.Contains(cell, "\n") {
					escapedRow[j] = "\"" + strings.ReplaceAll(cell, "\"", "\"\"") + "\""
				} else {
					escapedRow[j] = cell
				}
			}
			content.WriteString(strings.Join(escapedRow, ","))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Empty rows in CSV format
	if len(emptyRowsWithLineNumbers) > 0 {
		content.WriteString("# EMPTY ROWS:\n")
		lineNumbers := make([]int, 0, len(emptyRowsWithLineNumbers))
		for lineNum := range emptyRowsWithLineNumbers {
			lineNumbers = append(lineNumbers, lineNum)
		}
		sort.Ints(lineNumbers)

		for _, lineNum := range lineNumbers {
			row := emptyRowsWithLineNumbers[lineNum]
			escapedRow := make([]string, len(row))
			for j, cell := range row {
				if strings.Contains(cell, ",") || strings.Contains(cell, "\"") || strings.Contains(cell, "\n") {
					escapedRow[j] = "\"" + strings.ReplaceAll(cell, "\"", "\"\"") + "\""
				} else {
					escapedRow[j] = cell
				}
			}
			content.WriteString(fmt.Sprintf("# Line %d: ", lineNum))
			content.WriteString(strings.Join(escapedRow, ","))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Duplicate rows in CSV format
	if len(duplicateRowsWithLineNumbers) > 0 {
		content.WriteString("# DUPLICATE ROWS:\n")
		lineNumbers := make([]int, 0, len(duplicateRowsWithLineNumbers))
		for lineNum := range duplicateRowsWithLineNumbers {
			lineNumbers = append(lineNumbers, lineNum)
		}
		sort.Ints(lineNumbers)

		for _, lineNum := range lineNumbers {
			row := duplicateRowsWithLineNumbers[lineNum]
			escapedRow := make([]string, len(row))
			for j, cell := range row {
				if strings.Contains(cell, ",") || strings.Contains(cell, "\"") || strings.Contains(cell, "\n") {
					escapedRow[j] = "\"" + strings.ReplaceAll(cell, "\"", "\"\"") + "\""
				} else {
					escapedRow[j] = cell
				}
			}
			content.WriteString(fmt.Sprintf("# Line %d: ", lineNum))
			content.WriteString(strings.Join(escapedRow, ","))
			content.WriteString("\n")
		}
	}

	// Write to file
	if err := os.WriteFile(errorFile, []byte(content.String()), 0644); err != nil {
		lg.Warn().Err(err).Str("file", errorFile).Msg("Failed to save error rows to file, but returning content anyway")
		// Don't fail - return the content even if file write fails
	} else {
		lg.Info().Str("file", errorFile).Int("errorRowCount", len(errorRows)).Int("emptyRowCount", len(emptyRowsWithLineNumbers)).Int("duplicateRowCount", len(duplicateRowsWithLineNumbers)).Msg("Import log file generated successfully")
	}

	// Return file content directly instead of path
	return content.String(), nil
}

// getDefaultValue extracts default value from column config metadata
func (s *importService) getDefaultValue(cfg *dto.ColumnConfig) string {
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

// validateFieldValue performs comprehensive validation on a cell value for all UIDT types
func (s *importService) validateFieldValue(cellVal string, columnName string, fieldType string, meta map[string]interface{}) []string {
	var errors []string

	// Skip validation for empty values
	if cellVal == "" {
		return errors
	}

	// Type-specific validation
	switch fieldType {
	case "number":
		// For integer field, reject decimal values
		if strings.Contains(cellVal, ".") {
			errors = append(errors, fmt.Sprintf("Column '%s' [number]: Invalid number format '%s' - integer field cannot contain decimal point", columnName, cellVal))
		} else {
			// Try parsing as int64
			if intVal, err := strconv.ParseInt(cellVal, 10, 64); err != nil {
				errors = append(errors, fmt.Sprintf("Column '%s' [number]: Invalid number format '%s' - cannot parse as integer", columnName, cellVal))
			} else {
				// Check if value is within int32 range (PostgreSQL integer type)
				if intVal > math.MaxInt32 || intVal < math.MinInt32 {
					errors = append(errors, fmt.Sprintf("Column '%s' [number]: Value %s is out of range for integer type (must be between %d and %d)", columnName, cellVal, math.MinInt32, math.MaxInt32))
				}
			}
		}
		// Check min/max bounds from meta
		if meta != nil {
			if minVal, ok := meta["min"]; ok {
				if minFloat, ok2 := minVal.(float64); ok2 {
					if v, err := strconv.ParseFloat(cellVal, 64); err == nil && v < minFloat {
						errors = append(errors, fmt.Sprintf("Column '%s' [number]: Value %s is less than minimum %v", columnName, cellVal, minFloat))
					}
				}
			}
			if maxVal, ok := meta["max"]; ok {
				if maxFloat, ok2 := maxVal.(float64); ok2 {
					if v, err := strconv.ParseFloat(cellVal, 64); err == nil && v > maxFloat {
						errors = append(errors, fmt.Sprintf("Column '%s' [number]: Value %s exceeds maximum %v", columnName, cellVal, maxFloat))
					}
				}
			}
		}

	case "decimal":
		if _, err := strconv.ParseFloat(cellVal, 64); err != nil {
			errors = append(errors, fmt.Sprintf("Column '%s' [decimal]: Invalid decimal format '%s' - must be a valid floating point number", columnName, cellVal))
		}
		// Check min/max bounds
		if meta != nil {
			if minVal, ok := meta["min"]; ok {
				if minFloat, ok2 := minVal.(float64); ok2 {
					if v, err := strconv.ParseFloat(cellVal, 64); err == nil && v < minFloat {
						errors = append(errors, fmt.Sprintf("Column '%s' [decimal]: Value %s is less than minimum %v", columnName, cellVal, minFloat))
					}
				}
			}
			if maxVal, ok := meta["max"]; ok {
				if maxFloat, ok2 := maxVal.(float64); ok2 {
					if v, err := strconv.ParseFloat(cellVal, 64); err == nil && v > maxFloat {
						errors = append(errors, fmt.Sprintf("Column '%s' [decimal]: Value %s exceeds maximum %v", columnName, cellVal, maxFloat))
					}
				}
			}
		}

	case "boolean":
		lower := strings.ToLower(cellVal)
		if lower != "true" && lower != "false" && lower != "0" && lower != "1" && lower != "yes" && lower != "no" {
			errors = append(errors, fmt.Sprintf("Column '%s' [boolean]: Invalid boolean value '%s' - must be one of: true, false, 0, 1, yes, no", columnName, cellVal))
		}

	case "email":
		atIndex := strings.LastIndex(cellVal, "@")
		if atIndex == -1 || atIndex == 0 || atIndex == len(cellVal)-1 {
			errors = append(errors, fmt.Sprintf("Column '%s' [email]: Invalid email format '%s' - must contain @ with local and domain parts", columnName, cellVal))
		} else {
			domain := cellVal[atIndex+1:]
			if !strings.Contains(domain, ".") {
				errors = append(errors, fmt.Sprintf("Column '%s' [email]: Invalid email format '%s' - domain must contain a dot (.)", columnName, cellVal))
			}
		}

	case "json":
		var js interface{}
		if err := json.Unmarshal([]byte(cellVal), &js); err != nil {
			errors = append(errors, fmt.Sprintf("Column '%s' [json]: Invalid JSON format '%s' - %v", columnName, cellVal, err))
		}

	case "text", "longText":
		// Check max length from meta
		if meta != nil {
			if maxLen, ok := meta["max_length"]; ok {
				if maxLenInt, ok2 := maxLen.(float64); ok2 {
					if len(cellVal) > int(maxLenInt) {
						errors = append(errors, fmt.Sprintf("Column '%s' [%s]: Text length %d exceeds maximum length of %d characters", columnName, fieldType, len(cellVal), int(maxLenInt)))
					}
				}
			}
		}
	}

	return errors
}

// buildRecordsWithConfigAndErrors builds records using column configuration and tracks comprehensive error rows
func (s *importService) buildRecordsWithConfigAndErrors(dataRows [][]string, columnConfigs []dto.ColumnConfig, primary *dto.ColumnConfig, columnMap map[int]dto.ColumnResponse, req dto.CreateTableRequest, headers []string, settings dto.ImportSettings, lg *zerolog.Logger) ([]map[string]interface{}, [][]string, []string) {
	lg.Info().Int("recordCount", len(dataRows)).Msg("Starting to insert records with comprehensive error tracking")

	// Build config map for quick lookup
	configMap := make(map[string]dto.ColumnConfig)
	for _, cfg := range columnConfigs {
		configMap[cfg.ColumnName] = cfg
	}

	newRecords := []map[string]interface{}{}
	errorRows := [][]string{}
	errorMessages := []string{}

	for rowIdx, row := range dataRows {
		record := map[string]interface{}{
			"created_by":         req.CreatedBy,
			"last_modified_by":   req.CreatedBy,
			"created_time":       time.Now().UTC(),
			"last_modified_time": time.Now().UTC(),
		}

		rowErrors := []string{} // Collect all errors for this row
		recordValid := true

		for i, cellVal := range row {
			if i >= len(headers) {
				break
			}

			header := headers[i]
			cfg, configExists := configMap[header]

			// Determine if this header is the designated primary/title column
			primaryMatch := false
			if primary != nil && primary.ColumnName == header {
				primaryMatch = true
				cfg = *primary
				configExists = true
			}

			// Handle title column (only if primary explicitly matches this header)
			if primaryMatch {
				defaultVal := s.getDefaultValue(&cfg)

				// If cell is empty, use default if available
				if cellVal == "" && defaultVal != "" {
					valToValidate := defaultVal

					// Run validation checks on default value
					validationErrors := s.validateFieldValue(valToValidate, header, cfg.UIDT, cfg.Meta)
					if len(validationErrors) > 0 {
						recordValid = false
						rowErrors = append(rowErrors, validationErrors...)
					} else {
						record["title"] = s.convertValue(defaultVal, cfg.UIDT)
					}
				} else if cellVal != "" {
					// Validate the cell value
					validationErrors := s.validateFieldValue(cellVal, header, cfg.UIDT, cfg.Meta)
					if len(validationErrors) > 0 {
						recordValid = false
						rowErrors = append(rowErrors, validationErrors...)
					} else {
						record["title"] = s.convertValue(cellVal, cfg.UIDT)
					}
				}
				// If both cellVal and defaultVal are empty, leave title as is (allow empty rows)
				continue
			}

			// Skip if column not in config
			if !configExists {
				continue
			}

			// Skip if no column map entry
			colResp, colExists := columnMap[i]
			if !colExists {
				continue
			}

			defaultVal := s.getDefaultValue(&cfg)

			// Handle empty cells
			if cellVal == "" && defaultVal == "" {
				continue // Skip empty cells with no default
			}

			// Use cell value or default
			valToValidate := cellVal
			if cellVal == "" && defaultVal != "" {
				valToValidate = defaultVal
			}

			// Run validation checks before conversion
			validationErrors := s.validateFieldValue(valToValidate, header, cfg.UIDT, cfg.Meta)
			if len(validationErrors) > 0 {
				recordValid = false
				rowErrors = append(rowErrors, validationErrors...)
				continue // Skip this field if validation fails
			}

			// Convert value
			val := s.convertValue(valToValidate, cfg.UIDT)
			record[fmt.Sprintf("\"%s\"", colResp.ColumnName)] = val
		}

		// Check for empty title and allow it
		if _, hasTitle := record["title"]; !hasTitle {
			// Don't mark as invalid, just log it - allow empty rows to be inserted
		}

		if recordValid && len(rowErrors) == 0 {
			newRecords = append(newRecords, record)
		} else {
			// Add error row with all error messages
			errorRows = append(errorRows, row)
			// Show ALL ERRORS FIRST for this row
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

// validateColumnConfig validates that all column configs have required fields and valid source names
func (s *importService) validateColumnConfig(columnConfigs []dto.ColumnConfig, primary *dto.ColumnConfig, headers []string, lg *zerolog.Logger) error {
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

	// Validate each column config
	for i, cfg := range columnConfigs {
		// ColumnName is required
		if cfg.ColumnName == "" {
			lg.Error().Int("columnIndex", i).Msg("ColumnName is required for column config")
			return fmt.Errorf("column %d: column_name is required", i)
		}

		// ColumnName must match a CSV header
		if !validHeaders[cfg.ColumnName] {
			lg.Error().Str("columnName", cfg.ColumnName).Msg("ColumnName does not match any CSV column header")
			return fmt.Errorf("column %d: column_name '%s' not found in CSV headers", i, cfg.ColumnName)
		}

		// Primary column is required and must match a CSV header
		if primary == nil {
			lg.Error().Msg("primary_column is required in import config")
			return fmt.Errorf("primary_column is required in import config")
		}
		if primary.ColumnName == "" {
			lg.Error().Msg("PrimaryColumn provided but ColumnName is empty")
			return fmt.Errorf("primary_column: column_name is required")
		}
		valid := false
		for _, h := range headers {
			if h == primary.ColumnName {
				valid = true
				break
			}
		}
		if !valid {
			lg.Error().Str("primaryColumn", primary.ColumnName).Msg("PrimaryColumn does not match any CSV header")
			return fmt.Errorf("primary_column '%s' not found in CSV headers", primary.ColumnName)
		}

		// Title is the display name (optional)
		if cfg.Title == "" {
			lg.Warn().Str("columnName", cfg.ColumnName).Msg("Title is empty for column config, will use column name")
		}

		// UIDT (field type) should not be empty
		if cfg.UIDT == "" {
			lg.Warn().Str("columnName", cfg.ColumnName).Msg("UIDT is empty for column config, defaulting to text")
		}
	}

	lg.Info().Int("columnCount", len(columnConfigs)).Msg("Column config validation passed")
	return nil
}

// isRowEmpty checks if all cells in a row are empty (after trimming)
func (s *importService) isRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// removeEmptyRows removes rows where all cells are empty
func (s *importService) removeEmptyRows(rows [][]string) [][]string {
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

// countEmptyRows counts rows where all cells are empty without removing them
func (s *importService) countEmptyRows(rows [][]string) int {
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

// countDuplicateRecords counts duplicate rows without removing them
func (s *importService) countDuplicateRecords(rows [][]string) int {
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

// identifyEmptyRowsWithLineNumbers returns empty rows with their line numbers in the CSV
func (s *importService) identifyEmptyRowsWithLineNumbers(rows [][]string) map[int][]string {
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

// identifyDuplicateRowsWithLineNumbers returns duplicate rows with their line numbers in the CSV
func (s *importService) identifyDuplicateRowsWithLineNumbers(rows [][]string) map[int][]string {
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

// logCleanedDataWithSettings logs and saves cleaned data to a temporary JSON file before insertion
// This is called EVERY time regardless of which settings are enabled
func (s *importService) logCleanedDataWithSettings(headers []string, dataRows [][]string, stats *dto.ImportStatistics, settings dto.ImportSettings, lg *zerolog.Logger) {
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
