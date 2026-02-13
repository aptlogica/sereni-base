package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"mime/multipart"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	antivirusProviderInterface "serenibase/internal/providers/antivirus/interfaces"
	"serenibase/internal/providers/logger"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
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

	if err := s.scanFile(ctx, file, lg); err != nil {
		return dto.ImportTableResponse{}, err
	}

	if err := s.ensureBase(ctx, schemaName, &req, lg); err != nil {
		return dto.ImportTableResponse{}, err
	}

	headers, dataRows, err := s.parseCSV(file, lg)
	if err != nil {
		return dto.ImportTableResponse{}, err
	}

	columnTypes := s.inferColumnTypes(headers, dataRows)
	lg.Info().Strs("headers", headers).Strs("columnTypes", columnTypes).Msg("Column types inferred")

	titleColumnName := ""
	if len(headers) > 0 && headers[0] != "" {
		titleColumnName = headers[0]
	}

	tableResp, err := s.createTable(ctx, schemaName, req, lg)
	if err != nil {
		return dto.ImportTableResponse{}, err
	}

	titleColumnID := ""
	for _, col := range tableResp.Columns {
		if col.Title == "Title" {
			titleColumnID = col.ID.String()
			break
		}
	}

	if titleColumnID != "" {
		dt := s.getDatabaseType(columnTypes[0])
		updateColReq := dto.ColumnUpdate{
			Title: &titleColumnName,
			UIDT:  &columnTypes[0],
			DT:    &dt,
		}
		if _, err := s.tableService.UpdateColumn(ctx, schemaName, titleColumnID, updateColReq); err != nil {
			return dto.ImportTableResponse{}, err
		}
	}

	columnMap, err := s.addColumns(ctx, schemaName, req, headers, columnTypes, tableResp, lg)
	if err != nil {
		return dto.ImportTableResponse{}, err
	}

	newRecords := s.buildRecords(dataRows, columnTypes, columnMap, req, lg)
	lg.Info().Int("recordsCreated", len(newRecords)).Msg("Records prepared for insertion")

	if err := s.insertBatches(ctx, schemaName, tableResp, newRecords, lg); err != nil {
		return dto.ImportTableResponse{}, err
	}

	finalTableResp, err := s.refreshTable(ctx, schemaName, tableResp, lg)
	if err != nil {
		return dto.ImportTableResponse{TableResponse: tableResp}, nil
	}

	return dto.ImportTableResponse{TableResponse: finalTableResp}, nil
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

func (s *importService) ensureBase(ctx context.Context, schemaName string, req *dto.CreateTableRequest, lg *zerolog.Logger) error {
	if req.BaseID != "" {
		return nil
	}

	if req.WorkspaceID == "" {
		lg.Error().Msg("workspace_id is required when base_id is not provided")
		return fmt.Errorf("workspace_id is required when base_id is not provided")
	}

	baseName := req.Title + "_base"
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
	dataRows := records[1:]
	return headers, dataRows, nil
}

func (s *importService) createTable(ctx context.Context, schemaName string, req dto.CreateTableRequest, lg *zerolog.Logger) (dto.TableResponse, error) {
	lg.Info().Str("tableName", req.Title).Str("schemaName", schemaName).Msg("Creating table with defaults")
	tableResp, err := s.tableService.CreateTableWithDefaults(ctx, req, schemaName)
	if err != nil {
		fmt.Println("Error creating table:", err)
		lg.Error().Stack().Err(err).Str("tableName", req.Title).Str("schemaName", schemaName).Msg("Failed to create table with defaults")
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
			fmt.Println("Error adding column:", err)
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

func (s *importService) insertBatches(ctx context.Context, schemaName string, tableResp dto.TableResponse, newRecords []map[string]interface{}, lg *zerolog.Logger) error {
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
		if _, err := s.tableService.CreateRowsWithRecordsBulk(ctx, schemaName, tableResp.Model.Alias, batch); err != nil {
			lg.Error().Stack().Err(err).Int("batchNumber", batchNum).Int("batchSize", len(batch)).Msg("Failed to insert batch")
			return err
		}
		lg.Debug().Int("batchNumber", batchNum).Msg("Batch inserted successfully")
	}

	lg.Info().Msg("All batches inserted successfully")
	return nil
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
	isNumber := true
	isDecimal := true
	isBool := true
	isDate := true
	isEmail := true
	isURL := true
	isPhone := true
	isJSON := true

	hasData := false
	totalLength := 0
	count := 0

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
		count++

		// Check Number (integer or decimal)
		if isNumber || isDecimal {
			isNumber, isDecimal = s.checkNumericTypes(val, isNumber, isDecimal)
		}

		// Check Bool
		if isBool {
			isBool = s.checkBoolType(val)
		}

		// Check Date (Simple check)
		if isDate {
			isDate = s.checkDateType(val)
		}

		// Check Email
		if isEmail {
			isEmail = s.checkEmailType(val)
		}

		// Check URL
		if isURL {
			isURL = s.checkURLType(val)
		}

		// Check Phone
		if isPhone {
			isPhone = s.checkPhoneType(val)
		}

		// Check JSON
		if isJSON {
			isJSON = s.checkJSONType(val)
		}
	}

	if !hasData {
		return "text"
	}

	return s.determineTypeFromFlags(isBool, isNumber, isDecimal, isDate, isEmail, isURL, isPhone, isJSON, totalLength, count)
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
	formats := []string{"2006-01-02", "02-01-2006", "2006/01/02", "02/01/2006"}
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

func (s *importService) determineTypeFromFlags(isBool, isNumber, isDecimal, isDate, isEmail, isURL, isPhone, isJSON bool, totalLength, count int) string {
	avgLength := 0
	if count > 0 {
		avgLength = totalLength / count
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
	if isEmail {
		return "email"
	}
	if isURL {
		return "url"
	}
	if isPhone {
		return "phoneNumber"
	}
	if isJSON {
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
		// Return string for date, as DB usually handles it or expects string
		return val
	case "email", "url", "phoneNumber", "json":
		// These are stored as text
		return val
	}
	return val
}
