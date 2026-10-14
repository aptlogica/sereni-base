// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Postgres names are at most 63 bytes: "uq_" + alias + "_" + 8 hex characters
const maxConstraintAliasLength = 50

type tableConstraintService struct {
	repo          *pkg.DatabaseService
	modelService  interfaces.ModelService
	columnService interfaces.ColumnService
}

func NewTableConstraintService(repo *pkg.DatabaseService, modelService interfaces.ModelService, columnService interfaces.ColumnService) interfaces.TableConstraintService {
	return &tableConstraintService{repo: repo, modelService: modelService, columnService: columnService}
}

// List returns the constraints of one type on a table, newest first
func (s *tableConstraintService) List(ctx context.Context, schemaName, modelID, constraintType string) ([]dto.TableConstraintResponse, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}
	constraints, err := s.fetch(schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "model_id", Operator: "eq", Value: modelID},
			{Column: "type", Operator: "eq", Value: constraintType},
		},
		OrderBy: []string{"created_time DESC"},
	})
	if err != nil {
		return nil, err
	}
	if len(constraints) == 0 {
		return []dto.TableConstraintResponse{}, nil
	}

	columns, err := s.columnsByID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}
	installed, err := s.installedConstraints(tableIdentifier(schemaName, model.Alias))
	if err != nil {
		return nil, err
	}

	out := make([]dto.TableConstraintResponse, 0, len(constraints))
	for _, c := range constraints {
		status := dto.ConstraintStatusOK
		if !installed[c.ConstraintName] {
			status = dto.ConstraintStatusMissing
		}
		out = append(out, toConstraintResponse(c, columns, status))
	}
	return out, nil
}

// Check looks for records that already repeat values in the columns; nothing is changed
func (s *tableConstraintService) Check(ctx context.Context, schemaName string, req dto.CheckTableConstraintRequest) (dto.ConstraintCheckResult, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.ConstraintCheckResult{}, err
	}
	keyColumns, _, err := s.resolveColumns(ctx, schemaName, req.ModelID, req.ColumnIDs)
	if err != nil {
		return dto.ConstraintCheckResult{}, err
	}
	found, err := s.hasDuplicates(tableIdentifier(schemaName, model.Alias), keyColumns)
	if err != nil {
		return dto.ConstraintCheckResult{}, err
	}
	if found {
		return dto.ConstraintCheckResult{Valid: false, Message: duplicateMessage(keyColumns)}, nil
	}
	return dto.ConstraintCheckResult{Valid: true}, nil
}

// Create adds the unique key in Postgres, then stores it; if storing fails the key is dropped again
func (s *tableConstraintService) Create(ctx context.Context, schemaName string, req dto.CreateTableConstraintRequest) (dto.TableConstraintResponse, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	keyColumns, columns, err := s.resolveColumns(ctx, schemaName, req.ModelID, req.ColumnIDs)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	if err := s.ensureNoSameKey(schemaName, req.ModelID, "", req.ColumnIDs); err != nil {
		return dto.TableConstraintResponse{}, err
	}

	table := tableIdentifier(schemaName, model.Alias)
	if err := s.ensureNoDuplicates(table, keyColumns); err != nil {
		return dto.TableConstraintResponse{}, err
	}

	id := uuid.New()
	name := constraintName(model.Alias, id)
	if err := s.exec(duplicateMessage(keyColumns), addUniqueSQL(table, name, keyColumns)); err != nil {
		return dto.TableConstraintResponse{}, err
	}

	saved, err := s.repo.TableService.CreateRecord(tenant.TableConstraint{}.TableName(schemaName), map[string]interface{}{
		"id":               id.String(),
		"model_id":         req.ModelID,
		"type":             req.Type,
		"title":            req.Title,
		"constraint_name":  name,
		"column_ids":       mustJSON(req.ColumnIDs),
		"created_by":       req.CreatedBy,
		"last_modified_by": req.CreatedBy,
	})
	if err != nil {
		if _, dropErr := s.repo.DB.Exec(dropConstraintSQL(table, name)); dropErr != nil {
			logger.Get().Error().Err(dropErr).Str("constraint", name).Msg("failed to drop unique key after save failed")
		}
		return dto.TableConstraintResponse{}, app_errors.LogDatabaseError(err, "failed to create table constraint")
	}

	var out tenant.TableConstraint
	if err := helpers.MapToStruct(saved, &out); err != nil {
		return dto.TableConstraintResponse{}, app_errors.ErrMapToStruct
	}
	return toConstraintResponse(out, columns, dto.ConstraintStatusOK), nil
}

// Update saves the title; when the columns change, the Postgres
// constraint is replaced in one transaction, so a failure keeps the old one
func (s *tableConstraintService) Update(ctx context.Context, schemaName, id string, req dto.UpdateTableConstraintRequest) (dto.TableConstraintResponse, error) {
	existing, err := s.getByID(schemaName, id)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	model, err := s.modelService.GetModelByID(ctx, schemaName, existing.ModelID)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	keyColumns, columns, err := s.resolveColumns(ctx, schemaName, existing.ModelID, req.ColumnIDs)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}

	table := tableIdentifier(schemaName, model.Alias)
	keyChanged := !slices.Equal(existing.ColumnIDs, req.ColumnIDs)
	if keyChanged {
		if err := s.ensureNoSameKey(schemaName, existing.ModelID, id, req.ColumnIDs); err != nil {
			return dto.TableConstraintResponse{}, err
		}
		if err := s.ensureNoDuplicates(table, keyColumns); err != nil {
			return dto.TableConstraintResponse{}, err
		}
		if err := s.exec(duplicateMessage(keyColumns), dropConstraintSQL(table, existing.ConstraintName), addUniqueSQL(table, existing.ConstraintName, keyColumns)); err != nil {
			return dto.TableConstraintResponse{}, err
		}
	}

	updated, err := s.repo.TableService.UpdateRecord(tenant.TableConstraint{}.TableName(schemaName), id, map[string]interface{}{
		"title":              req.Title,
		"column_ids":         mustJSON(req.ColumnIDs),
		"last_modified_by":   req.UpdatedBy,
		"last_modified_time": time.Now(),
	})
	if err != nil {
		return dto.TableConstraintResponse{}, app_errors.LogDatabaseError(err, "failed to update table constraint")
	}

	var out tenant.TableConstraint
	if err := helpers.MapToStruct(updated, &out); err != nil {
		return dto.TableConstraintResponse{}, app_errors.ErrMapToStruct
	}
	return toConstraintResponse(out, columns, dto.ConstraintStatusOK), nil
}

// Delete drops the constraint in Postgres and removes it
func (s *tableConstraintService) Delete(ctx context.Context, schemaName, id string) error {
	existing, err := s.getByID(schemaName, id)
	if err != nil {
		return err
	}
	model, err := s.modelService.GetModelByID(ctx, schemaName, existing.ModelID)
	if err != nil {
		return err
	}
	if err := s.exec("", dropConstraintSQL(tableIdentifier(schemaName, model.Alias), existing.ConstraintName)); err != nil {
		return err
	}
	if err := s.repo.TableService.DeleteRecord(tenant.TableConstraint{}.TableName(schemaName), id); err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete table constraint")
	}
	return nil
}

// resolveColumns returns the key's columns in order, and all columns of the table by ID.
// Every column must belong to the table, appear once and be of a type a key can use.
// There's no limit on how many: Postgres enforces its own (32 columns per index).
func (s *tableConstraintService) resolveColumns(ctx context.Context, schemaName, modelID string, columnIDs []string) ([]tenant.Column, map[string]tenant.Column, error) {
	columns, err := s.columnsByID(ctx, schemaName, modelID)
	if err != nil {
		return nil, nil, err
	}

	keyColumns := make([]tenant.Column, 0, len(columnIDs))
	seen := map[string]bool{}
	for _, id := range columnIDs {
		if seen[id] {
			return nil, nil, app_errors.ConstraintRepeatedColumn
		}
		seen[id] = true
		column, ok := columns[id]
		if !ok {
			return nil, nil, app_errors.WithMessage(app_errors.ConstraintColumnNotFound, fmt.Sprintf("Column %s does not exist in this table", id))
		}
		if reason := tenant.KeyColumnBlockReason(column); reason != "" {
			return nil, nil, app_errors.WithMessage(app_errors.ConstraintColumnNotAllowed, fmt.Sprintf("%s can't be part of a key: %s", column.Title, reason))
		}
		keyColumns = append(keyColumns, column)
	}
	return keyColumns, columns, nil
}

// columnsByID returns the table's columns that are not deleted
func (s *tableConstraintService) columnsByID(ctx context.Context, schemaName, modelID string) (map[string]tenant.Column, error) {
	columns, err := s.columnService.GetColumnByModelID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]tenant.Column, len(columns))
	for _, c := range columns {
		if !c.Deleted {
			byID[c.ID.String()] = c
		}
	}
	return byID, nil
}

// ensureNoSameKey fails when another unique key of the table has the same columns, in any order
func (s *tableConstraintService) ensureNoSameKey(schemaName, modelID, exceptID string, columnIDs []string) error {
	others, err := s.fetch(schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{{Column: "model_id", Operator: "eq", Value: modelID}},
	})
	if err != nil {
		return err
	}
	for _, other := range others {
		if other.ID.String() != exceptID && sameColumnSet(other.ColumnIDs, columnIDs) {
			return app_errors.WithMessage(app_errors.ConstraintAlreadyExists, fmt.Sprintf("%q already covers these columns", other.Title))
		}
	}
	return nil
}

// ensureNoDuplicates fails when existing records already repeat values in the columns
func (s *tableConstraintService) ensureNoDuplicates(table string, columns []tenant.Column) error {
	found, err := s.hasDuplicates(table, columns)
	if err != nil {
		return err
	}
	if found {
		return app_errors.WithMessage(app_errors.ConstraintDuplicateValues, duplicateMessage(columns))
	}
	return nil
}

// hasDuplicates reports whether two records share the same values in the columns. Rows with an
// empty value in any column never conflict, so they are skipped.
func (s *tableConstraintService) hasDuplicates(table string, columns []tenant.Column) (bool, error) {
	names := make([]string, len(columns))
	notNull := make([]string, len(columns))
	for i, c := range columns {
		names[i] = pq.QuoteIdentifier(c.ColumnName)
		notNull[i] = names[i] + " IS NOT NULL"
	}
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s WHERE %s GROUP BY %s HAVING count(*) > 1)",
		table, strings.Join(notNull, " AND "), strings.Join(names, ", "))
	var found bool
	if err := s.repo.DB.QueryRow(query).Scan(&found); err != nil {
		return false, app_errors.WithMessage(app_errors.ConstraintFailed, app_errors.PostgresMessage(err))
	}
	return found, nil
}

// duplicateMessage names the columns whose values repeat, e.g.
// Duplicate data in column "EMAIL". Remove the duplicates and try again.
func duplicateMessage(columns []tenant.Column) string {
	titles := make([]string, len(columns))
	for i, c := range columns {
		titles[i] = fmt.Sprintf("%q", c.Title)
	}
	if len(titles) == 1 {
		return fmt.Sprintf("Duplicate data in column %s. Remove the duplicates and try again.", titles[0])
	}
	return fmt.Sprintf("Duplicate data in columns %s: some records have the same combination of values. Remove the duplicates and try again.",
		strings.Join(titles, " + "))
}

// installedConstraints returns the names of the unique and primary keys Postgres has on the table
func (s *tableConstraintService) installedConstraints(table string) (map[string]bool, error) {
	rows, err := s.repo.DB.Query(`SELECT conname FROM pg_constraint WHERE conrelid = $1::regclass AND contype IN ('u', 'p')`, table)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to read table constraints")
	}
	defer rows.Close()

	names := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, app_errors.LogDatabaseError(err, "failed to read table constraints")
		}
		names[name] = true
	}
	return names, rows.Err()
}

// exec runs the statements in one transaction. A duplicate found by Postgres (records added
// since the check) is reported with onDuplicate; other failures carry Postgres' message.
func (s *tableConstraintService) exec(onDuplicate string, statements ...string) error {
	tx, err := s.repo.DB.Begin()
	if err != nil {
		return app_errors.WithMessage(app_errors.ConstraintFailed, app_errors.PostgresMessage(err))
	}
	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			logger.Get().Error().Err(err).Str("query", stmt).Msg("failed to apply table constraint")
			if _, _, duplicate := app_errors.UniqueViolation(err); duplicate {
				return app_errors.WithMessage(app_errors.ConstraintDuplicateValues, onDuplicate)
			}
			return app_errors.WithMessage(app_errors.ConstraintFailed, app_errors.PostgresMessage(err))
		}
	}
	if err := tx.Commit(); err != nil {
		return app_errors.WithMessage(app_errors.ConstraintFailed, app_errors.PostgresMessage(err))
	}
	return nil
}

func (s *tableConstraintService) getByID(schemaName, id string) (tenant.TableConstraint, error) {
	limit := 1
	constraints, err := s.fetch(schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{{Column: "id", Operator: "eq", Value: id}},
		Limit:   &limit,
	})
	if err != nil {
		return tenant.TableConstraint{}, err
	}
	if len(constraints) == 0 {
		return tenant.TableConstraint{}, app_errors.ConstraintNotFound
	}
	return constraints[0], nil
}

func (s *tableConstraintService) fetch(schemaName string, params dbModels.QueryParams) ([]tenant.TableConstraint, error) {
	rows, err := s.repo.TableService.GetTableData(tenant.TableConstraint{}.TableName(schemaName), params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch table constraints")
	}
	out := make([]tenant.TableConstraint, 0, len(rows))
	for _, row := range rows {
		var c tenant.TableConstraint
		if err := helpers.MapToStruct(row, &c); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		out = append(out, c)
	}
	return out, nil
}

func tableIdentifier(schemaName, alias string) string {
	return pq.QuoteIdentifier(schemaName) + "." + pq.QuoteIdentifier(alias)
}

// constraintName is a unique Postgres name for a new key, e.g. uq_employee_3f9a2c1b
func constraintName(alias string, id uuid.UUID) string {
	if len(alias) > maxConstraintAliasLength {
		alias = alias[:maxConstraintAliasLength]
	}
	return "uq_" + alias + "_" + strings.ReplaceAll(id.String(), "-", "")[:8]
}

func addUniqueSQL(table, name string, columns []tenant.Column) string {
	names := make([]string, len(columns))
	for i, c := range columns {
		names[i] = pq.QuoteIdentifier(c.ColumnName)
	}
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s)", table, pq.QuoteIdentifier(name), strings.Join(names, ", "))
}

func dropConstraintSQL(table, name string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s", table, pq.QuoteIdentifier(name))
}

// sameColumnSet reports whether both keys have the same columns, in any order
func sameColumnSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA, sortedB := slices.Clone(a), slices.Clone(b)
	slices.Sort(sortedA)
	slices.Sort(sortedB)
	return slices.Equal(sortedA, sortedB)
}

func toConstraintResponse(c tenant.TableConstraint, columns map[string]tenant.Column, status string) dto.TableConstraintResponse {
	resolved := make([]dto.ConstraintColumn, 0, len(c.ColumnIDs))
	for _, id := range c.ColumnIDs {
		title := "Deleted column"
		if column, ok := columns[id]; ok {
			title = column.Title
		}
		resolved = append(resolved, dto.ConstraintColumn{ID: id, Title: title})
	}
	return dto.TableConstraintResponse{
		ID:             c.ID.String(),
		Title:          c.Title,
		ConstraintName: c.ConstraintName,
		Columns:        resolved,
		Status:         status,
	}
}

// mustJSON encodes a value for a jsonb column; the values passed here always encode
func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
