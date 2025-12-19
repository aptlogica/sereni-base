package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"mime/multipart"
	"serenibase/internal/dto"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type importService struct {
	tableService interfaces.TableManagementService
}

func NewImportService(tableService interfaces.TableManagementService) interfaces.ImportService {
	return &importService{
		tableService: tableService,
	}
}

func (s *importService) Import(ctx context.Context, schemaName string, req dto.CreateTableRequest, file *multipart.FileHeader) (dto.ImportTableResponse, error) {
	// 1. Open file
	f, err := file.Open()
	if err != nil {
		return dto.ImportTableResponse{}, err
	}
	defer f.Close()

	// 2. Parse CSV
	// TODO: Handle other extensions based on file.Filename
	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return dto.ImportTableResponse{}, err
	}

	if len(records) < 1 {
		return dto.ImportTableResponse{}, fmt.Errorf("empty csv file")
	}

	headers := records[0]
	dataRows := records[1:]

	// 3. Infer Types
	columnTypes := s.inferColumnTypes(headers, dataRows)

	// Store first CSV column header to use as the title column name
	titleColumnName := ""
	if len(headers) > 0 && headers[0] != "" {
		titleColumnName = headers[0]
	}

	// 4. Create Table
	// Keep the original req.Title for the model title, don't override with CSV column name
	tableResp, err := s.tableService.CreateTableWithDefaults(ctx, req, schemaName)
	if err != nil {
		return dto.ImportTableResponse{}, err
	}

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
			updateColReq := dto.ColumnUpdate{
				Title: &titleColumnName,
			}
			_, err := s.tableService.UpdateColumn(ctx, schemaName, titleColumnID, updateColReq)
			if err != nil {
				return dto.ImportTableResponse{}, err
			}
		}
	}

	// 5. Add Columns
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
			return dto.ImportTableResponse{}, err
		}
		columnMap[i] = colResp
	}

	// 6. Insert Data
	newRecords := []map[string]interface{}{}
	for _, row := range dataRows {
		// Compose a record (map) with all header-column values for this row
		record := map[string]interface{}{
			"id":                 uuid.New().String(),
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

	batchSize := 50
	for i := 0; i < len(newRecords); i += batchSize {
		end := i + batchSize
		if end > len(newRecords) {
			end = len(newRecords)
		}
		batch := newRecords[i:end]
		_, err := s.tableService.CreateRowsWithRecordsBulk(ctx, schemaName, tableResp.Model.Alias, batch)
		if err != nil {
			return dto.ImportTableResponse{}, err
		}
	}

	// Refresh table response to include new columns and records
	finalTableResp, err := s.tableService.GetTableByID(ctx, tableResp.Model.ID.String(), schemaName, 0, 0)
	if err != nil {
		return dto.ImportTableResponse{TableResponse: tableResp}, nil
	}

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
