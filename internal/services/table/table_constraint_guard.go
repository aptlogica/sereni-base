// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
)

// duplicateDetailRegexp reads Postgres' detail of a duplicate: Key (a, b)=(1, x) already exists.
var duplicateDetailRegexp = regexp.MustCompile(`^Key \((.+)\)=\((.*)\) already exists\.?$`)

// removeKeysUsingColumn deletes the entries of the unique keys that used a deleted column.
// Postgres already dropped those constraints with the column (DROP COLUMN ... CASCADE), so only
// the entries are left. The column is gone either way, so failures are logged, not returned.
func (s tableManagementService) removeKeysUsingColumn(schemaName string, column dto.ColumnResponse) {
	constraints, err := s.tableConstraints(schemaName, dbModels.QueryFilter{Column: "model_id", Operator: "eq", Value: column.ModelID.String()})
	if err != nil {
		logger.Get().Error().Err(err).Str("column", column.ID.String()).Msg("failed to read unique keys of deleted column")
		return
	}
	tableName := tenant.TableConstraint{}.TableName(schemaName)
	for _, c := range constraints {
		if !slices.Contains(c.ColumnIDs, column.ID.String()) {
			continue
		}
		if err := s.repo.TableService.DeleteRecord(tableName, c.ID.String()); err != nil {
			logger.Get().Error().Err(err).Str("constraint", c.ConstraintName).Msg("failed to remove unique key of deleted column")
		}
	}
}

// ensureColumnNotInKey fails when a unique key of the table uses the column
func (s tableManagementService) ensureColumnNotInKey(schemaName string, column dto.ColumnResponse, action string) error {
	constraints, err := s.tableConstraints(schemaName, dbModels.QueryFilter{Column: "model_id", Operator: "eq", Value: column.ModelID.String()})
	if err != nil {
		return err
	}
	for _, c := range constraints {
		if slices.Contains(c.ColumnIDs, column.ID.String()) {
			return app_errors.WithMessage(app_errors.ColumnUsedInConstraint,
				fmt.Sprintf("%s is part of the unique key %q. Remove it from the key before %s.", column.Title, c.Title, action))
		}
	}
	return nil
}

// ensureTypeFitsKey fails when the column is in a unique key and its new type can't be part of one
func (s tableManagementService) ensureTypeFitsKey(schemaName string, column dto.ColumnResponse, req dto.ColumnUpdate) error {
	if req.UIDT == nil || *req.UIDT == "" || strings.EqualFold(*req.UIDT, column.UIDT) {
		return nil
	}
	changed := tenant.Column{ColumnName: column.ColumnName, UIDT: *req.UIDT}
	if tenant.KeyColumnBlockReason(changed) == "" {
		return nil
	}
	return s.ensureColumnNotInKey(schemaName, column, "changing its type")
}

// recordWriteError turns a duplicate rejected by a unique key into a message naming the values;
// other errors are logged and reported as database errors
func (s tableManagementService) recordWriteError(ctx context.Context, schemaName string, err error, msg string) error {
	name, detail, ok := app_errors.UniqueViolation(err)
	if !ok {
		return app_errors.LogDatabaseError(err, msg)
	}
	logger.Get().Info().Str("constraint", name).Msg("record rejected by unique key")
	return app_errors.WithMessage(app_errors.UniqueKeyViolation, s.uniqueViolationMessage(ctx, schemaName, name, detail))
}

// uniqueViolationMessage names the duplicate values by column title,
// e.g. "A record with Department = 4 and Emp Code = E01 already exists"
func (s tableManagementService) uniqueViolationMessage(ctx context.Context, schemaName, constraintName, detail string) string {
	key := s.constraintByName(schemaName, constraintName)
	// Column names in the detail are shown by their titles
	titles := map[string]string{}
	if key != nil {
		titles = s.columnTitles(ctx, schemaName, key.ModelID)
	}
	if values, ok := duplicateValues(detail, titles); ok {
		return fmt.Sprintf("A record with %s already exists", values)
	}
	if key != nil {
		return fmt.Sprintf("A record with the same values already exists (unique key %q)", key.Title)
	}
	return "A record with the same values already exists"
}

// constraintByName returns the unique key with the Postgres name, or nil when it can't be read
func (s tableManagementService) constraintByName(schemaName, constraintName string) *tenant.TableConstraint {
	constraints, err := s.tableConstraints(schemaName, dbModels.QueryFilter{Column: "constraint_name", Operator: "eq", Value: constraintName})
	if err != nil || len(constraints) == 0 {
		return nil
	}
	return &constraints[0]
}

// columnTitles maps the column names of a table to their titles; empty when they can't be read
func (s tableManagementService) columnTitles(ctx context.Context, schemaName, modelID string) map[string]string {
	titles := map[string]string{}
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, modelID)
	if err != nil {
		return titles
	}
	for _, c := range columns {
		titles[c.ColumnName] = c.Title
	}
	return titles
}

// duplicateValues reads Postgres' detail of a duplicate as "Department = 4 and Emp Code = E01",
// naming columns by title when known; false when the detail can't be read
func duplicateValues(detail string, titles map[string]string) (string, bool) {
	match := duplicateDetailRegexp.FindStringSubmatch(detail)
	if match == nil {
		return "", false
	}
	names, values := strings.Split(match[1], ", "), strings.Split(match[2], ", ")
	if len(names) != len(values) {
		return "", false
	}
	parts := make([]string, len(names))
	for i, name := range names {
		name = strings.Trim(name, `"`)
		if title, ok := titles[name]; ok {
			name = title
		}
		parts[i] = fmt.Sprintf("%s = %s", name, values[i])
	}
	return strings.Join(parts, " and "), true
}

func (s tableManagementService) tableConstraints(schemaName string, filter dbModels.QueryFilter) ([]tenant.TableConstraint, error) {
	rows, err := s.repo.TableService.GetTableData(tenant.TableConstraint{}.TableName(schemaName), dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{filter},
	})
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to read unique keys")
	}
	constraints := make([]tenant.TableConstraint, 0, len(rows))
	for _, row := range rows {
		var c tenant.TableConstraint
		if err := helpers.MapToStruct(row, &c); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		constraints = append(constraints, c)
	}
	return constraints, nil
}
