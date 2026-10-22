// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
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

	"github.com/lib/pq"
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

// ensureTypeFitsKey fails when the column's type changes and a constraint can't take it: a check
// constraint using the column (its condition was written for the old type), or a unique key using
// it when the new type can't be part of a key
func (s tableManagementService) ensureTypeFitsKey(schemaName string, column dto.ColumnResponse, req dto.ColumnUpdate) error {
	if req.UIDT == nil || *req.UIDT == "" || strings.EqualFold(*req.UIDT, column.UIDT) {
		return nil
	}
	constraints, err := s.tableConstraints(schemaName, dbModels.QueryFilter{Column: "model_id", Operator: "eq", Value: column.ModelID.String()})
	if err != nil {
		return err
	}
	keyBlocked := tenant.KeyColumnBlockReason(tenant.Column{ColumnName: column.ColumnName, UIDT: *req.UIDT}) != ""
	for _, c := range constraints {
		if !slices.Contains(c.ColumnIDs, column.ID.String()) {
			continue
		}
		if c.Type == tenant.ConstraintTypeCheck {
			return app_errors.WithMessage(app_errors.ColumnUsedInConstraint,
				fmt.Sprintf("%s is used by the check constraint %q. Delete it before changing the field type.", column.Title, c.Title))
		}
		if keyBlocked {
			return app_errors.WithMessage(app_errors.ColumnUsedInConstraint,
				fmt.Sprintf("%s is part of the unique key %q. Remove it from the key before changing its type.", column.Title, c.Title))
		}
	}
	return nil
}

// recordWriteError turns a duplicate rejected by a unique key into a message naming the values;
// other errors are logged and reported as database errors
func (s tableManagementService) recordWriteError(ctx context.Context, schemaName string, err error, msg string) error {
	if name, ok := app_errors.CheckViolation(err); ok {
		return s.checkViolationError(schemaName, name)
	}
	name, detail, ok := app_errors.UniqueViolation(err)
	if !ok {
		return app_errors.LogDatabaseError(err, msg)
	}
	logger.Get().Info().Str("constraint", name).Msg("record rejected by unique key")
	return app_errors.WithMessage(app_errors.UniqueKeyViolation, s.uniqueViolationMessage(ctx, schemaName, name, detail))
}

// uniqueViolationMessage says that the unique key doesn't allow duplicates, naming the key and its
// columns by title, e.g. Duplicate value not allowed. The unique key "Unique code" does not allow
// two records with the same Department and Emp Code.
func (s tableManagementService) uniqueViolationMessage(ctx context.Context, schemaName, constraintName, detail string) string {
	const prefix = "Duplicate value not allowed. "
	key := s.constraintByName(schemaName, constraintName)
	columns := s.duplicateColumnTitles(ctx, schemaName, key, detail)
	switch {
	case key != nil && len(columns) > 0:
		return fmt.Sprintf("%sThe unique key %q does not allow two records with the same %s.", prefix, key.Title, joinTitles(columns))
	case key != nil:
		return fmt.Sprintf("%sThe unique key %q does not allow duplicate values.", prefix, key.Title)
	case len(columns) > 0:
		return fmt.Sprintf("%sTwo records can't have the same %s.", prefix, joinTitles(columns))
	}
	return prefix + "This record repeats the values of a unique key."
}

// checkViolationError says which check constraint rejected the record and its condition, e.g.
// Value not allowed. The check constraint "Age 18–100" failed: age BETWEEN 18 AND 100
func (s tableManagementService) checkViolationError(schemaName, constraintName string) error {
	logger.Get().Info().Str("constraint", constraintName).Msg("record rejected by check constraint")
	check := s.constraintByName(schemaName, constraintName)
	if check == nil {
		return app_errors.WithMessage(app_errors.CheckConstraintViolation,
			fmt.Sprintf("Value not allowed by the check constraint %q.", constraintName))
	}
	message := fmt.Sprintf("Value not allowed. The check constraint %q failed", check.Title)
	if condition := s.checkConditionText(schemaName, constraintName); condition != "" {
		message += ": " + condition
	} else {
		message += "."
	}
	return app_errors.WithMessage(app_errors.CheckConstraintViolation, message)
}

// checkConditionText is the condition of a check constraint as the user wrote it (kept in the
// constraint's comment), else as Postgres has it; "" when it can't be read
func (s tableManagementService) checkConditionText(schemaName, constraintName string) string {
	if s.repo.DB == nil {
		return ""
	}
	var comment, definition string
	err := s.repo.DB.QueryRow(`SELECT coalesce(obj_description(oid, 'pg_constraint'), ''), pg_get_constraintdef(oid)
		FROM pg_constraint WHERE conname = $1 AND contype = 'c' AND connamespace = $2::regnamespace LIMIT 1`,
		constraintName, pq.QuoteIdentifier(schemaName)).Scan(&comment, &definition)
	if err != nil {
		return ""
	}
	var label struct {
		Condition string `json:"condition"`
	}
	if json.Unmarshal([]byte(comment), &label) == nil && label.Condition != "" {
		return label.Condition
	}
	return strings.TrimSuffix(strings.TrimPrefix(strings.TrimSuffix(definition, " NOT VALID"), "CHECK ("), ")")
}

// constraintByName returns the unique key with the Postgres name, or nil when it can't be read
func (s tableManagementService) constraintByName(schemaName, constraintName string) *tenant.TableConstraint {
	constraints, err := s.tableConstraints(schemaName, dbModels.QueryFilter{Column: "constraint_name", Operator: "eq", Value: constraintName})
	if err != nil || len(constraints) == 0 {
		return nil
	}
	return &constraints[0]
}

// duplicateColumnTitles returns the titles of the key's columns, in key order. For keys added
// outside the app, the columns come from Postgres' detail, e.g. Key (badge)=(B7) already exists.
func (s tableManagementService) duplicateColumnTitles(ctx context.Context, schemaName string, key *tenant.TableConstraint, detail string) []string {
	if key == nil {
		return detailColumnNames(detail)
	}
	byID := map[string]string{}
	if columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, key.ModelID); err == nil {
		for _, c := range columns {
			byID[c.ID.String()] = c.Title
		}
	}
	titles := make([]string, 0, len(key.ColumnIDs))
	for _, id := range key.ColumnIDs {
		if title, ok := byID[id]; ok {
			titles = append(titles, title)
		}
	}
	return titles
}

// detailColumnNames reads the column names from Postgres' detail of a duplicate; nil when it can't be read
func detailColumnNames(detail string) []string {
	match := duplicateDetailRegexp.FindStringSubmatch(detail)
	if match == nil {
		return nil
	}
	names := strings.Split(match[1], ", ")
	for i, name := range names {
		names[i] = strings.Trim(name, `"`)
	}
	return names
}

// joinTitles lists titles as "Email", or "Title and Number", or "A, B and C"
func joinTitles(titles []string) string {
	if len(titles) == 1 {
		return titles[0]
	}
	return strings.Join(titles[:len(titles)-1], ", ") + " and " + titles[len(titles)-1]
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

// ensureDefaultFitsKey fails when the update sets a default on a column whose unique key would
// then have a default on all its columns. Removing or leaving the default is always allowed.
func (s tableManagementService) ensureDefaultFitsKey(ctx context.Context, schemaName string, column dto.ColumnResponse, req dto.ColumnUpdate) error {
	if req.Meta == nil || !tenant.HasDefaultValue(tenant.Column{Meta: *req.Meta}) {
		return nil
	}
	constraints, err := s.tableConstraints(schemaName, dbModels.QueryFilter{Column: "model_id", Operator: "eq", Value: column.ModelID.String()})
	if err != nil {
		return err
	}
	columnID := column.ID.String()
	keys := make([]tenant.TableConstraint, 0, len(constraints))
	for _, c := range constraints {
		if slices.Contains(c.ColumnIDs, columnID) {
			keys = append(keys, c)
		}
	}
	if len(keys) == 0 {
		return nil
	}

	all, err := s.columnsService.GetColumnByModelID(ctx, schemaName, column.ModelID.String())
	if err != nil {
		return err
	}
	byID := make(map[string]tenant.Column, len(all))
	for _, c := range all {
		byID[c.ID.String()] = c
	}
	for _, key := range keys {
		if othersAllHaveDefault(key.ColumnIDs, columnID, byID) {
			return app_errors.WithMessage(app_errors.UniqueKeyDefaultConflict, tenant.UniqueDefaultConflictMessage)
		}
	}
	return nil
}

// othersAllHaveDefault reports whether every column of the key except skipID has a default value
func othersAllHaveDefault(columnIDs []string, skipID string, columns map[string]tenant.Column) bool {
	for _, id := range columnIDs {
		if id != skipID && !tenant.HasDefaultValue(columns[id]) {
			return false
		}
	}
	return true
}
