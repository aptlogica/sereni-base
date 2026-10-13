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
	"github.com/aptlogica/sereni-base/internal/models/tenant"
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

	byName, err := s.rowValuesByName(ctx, schemaName, values)
	if err != nil {
		return dto.RecordResponse{}, err
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

// UpsertRowValues creates a record with values (column ID -> value) when rowID is 0, otherwise updates that
// record. Used by POST /row/data/insert.
func (s tableManagementService) UpsertRowValues(
	ctx context.Context,
	schemaName string,
	modelID string,
	rowID int,
	values map[string]interface{},
	user string,
) (dto.RecordResponse, error) {
	if rowID > 0 {
		return s.UpdateRowValues(ctx, schemaName, modelID, rowID, values, user)
	}
	model, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}
	return s.insertRowValues(ctx, schemaName, model, values, user, user)
}

// rowValuesByName checks that every column (by ID) can be written and returns the values by column name
func (s tableManagementService) rowValuesByName(ctx context.Context, schemaName string, values map[string]interface{}) (map[string]interface{}, error) {
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
			return nil, err
		}
		// Link values must go through the link endpoint so both sides stay in sync; lookups are read-only.
		if columnData.UIDT == uidtLinks || columnData.UIDT == uidtLookup {
			return nil, app_errors.LinkColumnNotWritable
		}
		if !s.allowInsert(columnData) {
			return nil, app_errors.UpdateNotAllowed
		}
		byName[columnData.ColumnName] = cellValue(columnData, values[columnID])
	}
	return byName, nil
}

// insertRowValues creates one record with its values (by column ID) in one statement. On a table with a custom
// primary key every key field must be given, and the key must not be used by another record.
func (s tableManagementService) insertRowValues(
	ctx context.Context,
	schemaName string,
	model tenant.Model,
	values map[string]interface{},
	createdBy string,
	updatedBy string,
) (dto.RecordResponse, error) {
	byName, err := s.rowValuesByName(ctx, schemaName, values)
	if err != nil {
		return dto.RecordResponse{}, err
	}
	if err := s.checkKeyValues(ctx, schemaName, model, 0, byName); err != nil {
		return dto.RecordResponse{}, err
	}

	now := time.Now().UTC()
	data := make(map[string]interface{}, len(byName)+4)
	for name, value := range byName {
		data[fmt.Sprintf(QuotedColumnFormat, name)] = value
	}
	data["created_by"] = createdBy
	data["last_modified_by"] = updatedBy
	data["created_time"] = now
	data["last_modified_time"] = now

	record, err := s.repo.TableService.CreateRecord(fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias), data)
	if err != nil {
		return dto.RecordResponse{}, s.keyWriteError(ctx, schemaName, model.Alias, err, "failed to create row with values")
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
