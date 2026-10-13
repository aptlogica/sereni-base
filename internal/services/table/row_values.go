// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
)

// UpdateRowValues saves several fields of one record (values by column ID) in one statement. Every value is
// checked first, including the record's primary key as a whole, so the record is either fully updated or left
// as it was.
func (s tableManagementService) UpdateRowValues(
	ctx context.Context,
	schemaName string,
	modelID string,
	rowID int,
	values map[string]interface{},
	updatedBy string,
) (dto.RecordResponse, error) {
	if len(values) == 0 {
		return dto.RecordResponse{}, nil
	}

	// Column IDs in a fixed order, so errors and statements don't depend on map order
	columnIDs := make([]string, 0, len(values))
	for id := range values {
		columnIDs = append(columnIDs, id)
	}
	sort.Strings(columnIDs)

	byName := make(map[string]interface{}, len(values))
	for _, columnID := range columnIDs {
		columnData, err := s.GetColumnById(ctx, schemaName, columnID)
		if err != nil {
			return dto.RecordResponse{}, err
		}
		// Link values must go through the link endpoint so both sides stay in sync; lookups are read-only.
		if columnData.UIDT == uidtLinks || columnData.UIDT == uidtLookup {
			return dto.RecordResponse{}, app_errors.LinkColumnNotWritable
		}
		if !s.allowInsert(columnData) {
			return dto.RecordResponse{}, app_errors.UpdateNotAllowed
		}
		byName[columnData.ColumnName] = cellValue(columnData, values[columnID])
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	// Key fields: name the field when it is left empty and the record that already holds the key
	if err := s.checkKeyValues(ctx, schemaName, model, rowID, byName); err != nil {
		return dto.RecordResponse{}, err
	}

	data := make(map[string]interface{}, len(byName)+2)
	for name, value := range byName {
		data[fmt.Sprintf(QuotedColumnFormat, name)] = value
	}
	data["last_modified_by"] = updatedBy
	data["last_modified_time"] = time.Now().UTC()

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	record, err := s.repo.TableService.UpdateRecord(tableName, rowID, data)
	if err != nil {
		return dto.RecordResponse{}, s.keyWriteError(ctx, schemaName, model.Alias, err, "failed to update record for column")
	}
	return dto.RecordResponse{Record: record}, nil
}

// cellValue prepares a value for its column: array columns always get a slice
func cellValue(column dto.ColumnResponse, value interface{}) interface{} {
	if value == nil || column.DT == "" || !strings.HasSuffix(column.DT, "[]") {
		return value
	}
	if _, ok := value.([]interface{}); ok {
		return value
	}
	return []interface{}{value}
}

// removeCreatedRows deletes rows created by a request that then failed, so no half-filled record is left behind
func (s tableManagementService) removeCreatedRows(tableName string, rowIDs []int) {
	for _, id := range rowIDs {
		if err := s.repo.TableService.DeleteRecord(tableName, id); err != nil {
			logger.Get().Error().Err(err).Str("table", tableName).Int("row_id", id).Msg("failed to remove row after a failed create")
		}
	}
}
