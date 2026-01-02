package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"mime/multipart"
	"serenibase/internal/dto"
	antivirusProviderInterface "serenibase/internal/providers/antivirus/interfaces"
	"serenibase/internal/providers/logger"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"strconv"
	"strings"
	"time"
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

func (s *importService) Import(ctx context.Context, schemaName string, req dto.CreateTableRequest, file *multipart.FileHeader) (dto.ImportTableResponse, error) {
	lg := logger.Get()

	// 1. Scan file with antivirus before processing
	if s.antivirusProvider != nil {
		f, err := file.Open()
		if err != nil {
			lg.Error().Stack().Err(err).Str("file", file.Filename).Msg("Failed to open CSV file for antivirus scan")
			return dto.ImportTableResponse{}, err
		}
		scanResult, scanErr := s.antivirusProvider.ScanReader(ctx, file.Filename, f)
		f.Close()
		if scanErr != nil {
			lg.Error().Stack().Err(scanErr).Str("file", file.Filename).Str("threat", scanResult.Threat).Msg("Antivirus scan detected threat")
			return dto.ImportTableResponse{}, fmt.Errorf("file '%s' is infected or contains malicious content", file.Filename)
		}
		lg.Info().Str("file", file.Filename).Msg("Antivirus scan passed")
	}

	// 2. Create base if base_id is not provided
	if req.BaseID == "" {
		if req.WorkspaceID == "" {
			lg.Error().Msg("workspace_id is required when base_id is not provided")
			return dto.ImportTableResponse{}, fmt.Errorf("workspace_id is required when base_id is not provided")
		}

		baseName := req.Title + "_base"
		lg.Info().Str("baseName", baseName).Str("workspaceID", req.WorkspaceID).Msg("Creating new base for import")

		createBaseReq := dto.CreateBaseRequest{
			Title:       baseName,
			Description: helpers.StringPtr("Auto-created base for table import"),
			WorkspaceID: req.WorkspaceID,
			CreatedBy:   req.CreatedBy,
		}

		newBase, err := s.baseManagementService.CreateBase(ctx, createBaseReq, schemaName, req.CreatedBy)
		if err != nil {
			lg.Error().Stack().Err(err).Str("baseName", baseName).Msg("Failed to create base for import")
			return dto.ImportTableResponse{}, fmt.Errorf("failed to create base: %w", err)
		}

		req.BaseID = newBase.ID.String()
		lg.Info().Str("baseID", req.BaseID).Str("baseName", baseName).Msg("Base created successfully for import")
	}

	// 3. Open file for processing
	f, err := file.Open()
	if err != nil {
		lg.Error().Stack().Err(err).Str("file", file.Filename).Msg("Failed to open CSV file")
		return dto.ImportTableResponse{}, err
	}
	defer f.Close()

	// 2. Parse CSV
	// TODO: Handle other extensions based on file.Filename
	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		lg.Error().Stack().Err(err).Str("file", file.Filename).Msg("Failed to parse CSV file")
		return dto.ImportTableResponse{}, err
	}

	if len(records) < 1 {
		errMsg := "empty csv file"
		lg.Error().Str("file", file.Filename).Msg(errMsg)
		return dto.ImportTableResponse{}, fmt.Errorf("%s", errMsg)
	}

	lg.Info().Str("file", file.Filename).Int("rows", len(records)).Msg("CSV file parsed successfully")

	headers := records[0]
	dataRows := records[1:]

	// 3. Infer Types
	columnTypes := s.inferColumnTypes(headers, dataRows)
	lg.Info().Strs("headers", headers).Strs("columnTypes", columnTypes).Msg("Column types inferred")

	// Store first CSV column header to use as the title column name
	titleColumnName := ""
	if len(headers) > 0 && headers[0] != "" {
		titleColumnName = headers[0]
	}

	// 4. Create Table
	// Keep the original req.Title for the model title, don't override with CSV column name
	lg.Info().Str("tableName", req.Title).Str("schemaName", schemaName).Msg("Creating table with defaults")
	tableResp, err := s.tableService.CreateTableWithDefaults(ctx, req, schemaName)
	if err != nil {
		lg.Error().Stack().Err(err).Str("tableName", req.Title).Str("schemaName", schemaName).Msg("Failed to create table with defaults")
		return dto.ImportTableResponse{}, err
	}
	lg.Info().Str("tableID", tableResp.Model.ID.String()).Msg("Table created successfully")

	// Update the Title column with the first CSV header name
	if titleColumnName != "" {
		titleColumnID := ""
		// Find the Title column ID from the default columns
		for _, col := range tableResp.Columns {
			if col.Title == "Title" {
				titleColumnID = col.ID.String()
				break
			}
		}

		// Update the Title column with the first CSV header name
		if titleColumnID != "" {
			lg.Info().Str("columnID", titleColumnID).Str("newTitle", titleColumnName).Msg("Updating title column name")
			updateColReq := dto.ColumnUpdate{
				Title: &titleColumnName,
			}
			_, err := s.tableService.UpdateColumn(ctx, schemaName, titleColumnID, updateColReq)
			if err != nil {
				lg.Error().Stack().Err(err).Str("columnID", titleColumnID).Str("newTitle", titleColumnName).Msg("Failed to update title column")
				return dto.ImportTableResponse{}, err
			}
			lg.Info().Str("columnID", titleColumnID).Msg("Title column updated successfully")
		}
	}

	// 5. Add Columns
	lg.Info().Int("columnCount", len(headers)-1).Msg("Starting to add columns")
	columnMap := make(map[int]dto.ColumnResponse) // Index -> Column
	systemFieldAdded := false
	for i, header := range headers {
		// Skip first column as it's used for the title field
		if i == 0 {
			continue
		}

		colType := columnTypes[i]
		if colType == "text" && !systemFieldAdded {
			systemFieldAdded = true
		}

		// Skip empty headers
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
			DT:          colType,
			OrderIndex:  helpers.Float64Ptr(float64(i + 6)), // Start after system columns (0-5)
			Virtual:     helpers.BoolPtr(false),
			System:      helpers.BoolPtr(false),
			CreatedBy:   req.CreatedBy,
		}
		colResp, err := s.tableService.AddColumn(ctx, schemaName, addColReq)
		if err != nil {
			lg.Error().Stack().Err(err).Str("columnTitle", header).Str("columnType", colType).Msg("Failed to add column")
			return dto.ImportTableResponse{}, err
		}
		columnMap[i] = colResp
		lg.Debug().Str("columnTitle", header).Str("columnType", colType).Msg("Column added successfully")
	}
	lg.Info().Int("columnsAdded", len(columnMap)).Msg("All columns added")

	// 6. Insert Data
	lg.Info().Int("recordCount", len(dataRows)).Msg("Starting to insert records")
	newRecords := []map[string]interface{}{}
	for _, row := range dataRows {
		// Compose a record (map) with all header-column values for this row
		// Note: Do not set 'id' field - let the database auto-generate it (bigint auto-increment)
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

			// Map first column to title field
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
	lg.Info().Int("recordsCreated", len(newRecords)).Msg("Records prepared for insertion")

	batchSize := 50
	totalBatches := (len(newRecords) + batchSize - 1) / batchSize
	lg.Info().Int("batchSize", batchSize).Int("totalBatches", totalBatches).Msg("Starting batch insertion")

	for i := 0; i < len(newRecords); i += batchSize {
		end := i + batchSize
		if end > len(newRecords) {
			end = len(newRecords)
		}
		batch := newRecords[i:end]
		batchNum := (i / batchSize) + 1

		lg.Info().Int("batchNumber", batchNum).Int("batchSize", len(batch)).Msg("Inserting batch")
		_, err := s.tableService.CreateRowsWithRecordsBulk(ctx, schemaName, tableResp.Model.Alias, batch)
		if err != nil {
			lg.Error().Stack().Err(err).Int("batchNumber", batchNum).Int("batchSize", len(batch)).Msg("Failed to insert batch")
			return dto.ImportTableResponse{}, err
		}
		lg.Debug().Int("batchNumber", batchNum).Msg("Batch inserted successfully")
	}
	lg.Info().Msg("All batches inserted successfully")

	// Refresh table response to include new columns and records
	lg.Info().Str("tableID", tableResp.Model.ID.String()).Msg("Refreshing table response")
	finalTableResp, err := s.tableService.GetTableByID(ctx, tableResp.Model.ID.String(), schemaName, 0, 0)
	if err != nil {
		lg.Warn().Stack().Err(err).Str("tableID", tableResp.Model.ID.String()).Msg("Failed to refresh table response, returning cached response")
		return dto.ImportTableResponse{TableResponse: tableResp}, nil
	}

	lg.Info().Str("tableID", finalTableResp.Model.ID.String()).Int("columns", len(finalTableResp.Columns)).Int("records", len(finalTableResp.Records)).Msg("Import completed successfully")
	return dto.ImportTableResponse{TableResponse: finalTableResp}, nil
}

func (s *importService) inferColumnTypes(headers []string, rows [][]string) []string {
	types := make([]string, len(headers))
	for i := range headers {
		types[i] = s.inferType(rows, i)
	}
	return types
}

func (s *importService) inferType(rows [][]string, colIndex int) string {
	isNumber := true
	isDecimal := true
	isBool := true
	isDate := true
	isLongText := false

	hasData := false
	totalLength := 0

	for _, row := range rows {
		if colIndex >= len(row) {
			continue
		}
		val := row[colIndex]
		if val == "" {
			continue
		}
		hasData = true
		totalLength += len(val)

		// Check Number (integer or decimal)
		if isNumber || isDecimal {
			if _, err := strconv.ParseInt(val, 10, 64); err != nil {
				isNumber = false
			}
			if _, err := strconv.ParseFloat(val, 64); err != nil {
				isDecimal = false
			}
		}

		// Check Bool
		if isBool {
			lower := strings.ToLower(val)
			if lower != "true" && lower != "false" && lower != "0" && lower != "1" && lower != "yes" && lower != "no" {
				isBool = false
			}
		}

		// Check Date (Simple check)
		if isDate {
			formats := []string{"2006-01-02", "02-01-2006", "2006/01/02", "02/01/2006"}
			parsed := false
			for _, f := range formats {
				if _, err := time.Parse(f, val); err == nil {
					parsed = true
					break
				}
			}
			if !parsed {
				isDate = false
			}
		}

		// Check if long text (avg length > 255)
		if len(val) > 255 {
			isLongText = true
		}
	}

	if !hasData {
		return "text"
	}

	if isBool {
		return "boolean"
	}
	if isNumber {
		return "number"
	}
	if isDecimal {
		return "decimal"
	}
	if isDate {
		return "date"
	}
	if isLongText {
		return "longText"
	}
	return "text"
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
		// Return string for date, as DB usually handles it or expects string
		return val
	}
	return val
}
