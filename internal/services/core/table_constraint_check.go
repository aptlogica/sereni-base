// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Check constraints: the user writes the SQL condition, Postgres enforces it with CHECK (...).
// The table_constraints row keeps the title and the columns the condition uses; the category and
// the condition as typed are kept in Postgres, as a comment on the constraint.

const (
	maxConditionLength   = 2000
	defaultCheckCategory = "custom"
	// Name of the constraint Test adds in a transaction that is always rolled back
	testCheckName = "ck_sereni_test"
	// Postgres' own messages are shown after this
	postgresPrefix    = "PostgreSQL: "
	unbalancedMessage = "Unbalanced parentheses or quotes."
)

// checkCategories are the categories the settings popup offers. A category is only a label: any
// condition can be saved under any category.
var checkCategories = map[string]bool{
	"number_date": true, "text": true, "compare": true, defaultCheckCategory: true,
}

// Adding a CHECK scans the table under a lock, so a slow condition or a busy table gives up early
var checkLimits = []string{"SET LOCAL statement_timeout = '15s'", "SET LOCAL lock_timeout = '5s'"}

// A row would start failing as time passes, so a CHECK can't use the current date or time
var currentTimeRegexp = regexp.MustCompile(`(?i)\b(now\s*\(|(current_date|current_time|current_timestamp|localtime|localtimestamp|clock_timestamp|statement_timestamp|transaction_timestamp)\b)`)

// escapeStringRegexp finds E'...' strings, where a backslash can hide the closing quote
var escapeStringRegexp = regexp.MustCompile(`(?i)(^|[^a-z0-9_$])e'`)

// installedCheck is a CHECK constraint as Postgres has it
type installedCheck struct {
	definition string
	input      dto.CheckInput
}

// checkInput validates the category and the condition; an empty category is custom
func checkInput(in dto.CheckInput) (dto.CheckInput, error) {
	category := strings.TrimSpace(in.Category)
	if category == "" {
		category = defaultCheckCategory
	}
	if !checkCategories[category] {
		return dto.CheckInput{}, app_errors.WithMessage(app_errors.CheckCategoryUnknown, fmt.Sprintf("Unknown category %q", category))
	}
	condition, err := checkCondition(in.Condition)
	if err != nil {
		return dto.CheckInput{}, err
	}
	return dto.CheckInput{Category: category, Condition: condition}, nil
}

// checkCondition makes sure the condition stays inside CHECK (...): one expression, no statement
// separators or comments, balanced parentheses and closed quotes. Postgres checks the rest.
func checkCondition(text string) (string, error) {
	condition := strings.TrimSpace(text)
	message := ""
	switch {
	case condition == "":
		message = "Condition is required."
	case len(condition) > maxConditionLength:
		message = fmt.Sprintf("The condition is too long (at most %d characters).", maxConditionLength)
	case strings.Contains(condition, `\'`), escapeStringRegexp.MatchString(condition):
		message = `Escape strings (E'...' or \') aren't allowed.`
	default:
		var code []byte
		if code, message = scanCondition(condition); message == "" && currentTimeRegexp.Match(code) {
			message = "A check can't depend on the current date or time: a record would start failing as time passes. Use a trigger."
		}
	}
	if message != "" {
		return "", app_errors.WithMessage(app_errors.CheckConditionInvalid, message)
	}
	return condition, nil
}

// scanCondition returns the condition with quoted text blanked out, or why it isn't allowed
func scanCondition(condition string) ([]byte, string) {
	code := make([]byte, 0, len(condition))
	depth := 0
	for i := 0; i < len(condition); i++ {
		ch := condition[i]
		if ch == '\'' || ch == '"' {
			end := closingQuote(condition, i)
			if end < 0 {
				return nil, unbalancedMessage
			}
			code = append(code, ' ')
			i = end
			continue
		}
		message, delta := scanChar(condition, i)
		if message != "" {
			return nil, message
		}
		depth += delta
		if depth < 0 {
			return nil, unbalancedMessage
		}
		code = append(code, ch)
	}
	if depth != 0 {
		return nil, unbalancedMessage
	}
	return code, ""
}

// scanChar checks the unquoted character at index i: the reason it isn't allowed, and how it changes the parentheses depth
func scanChar(condition string, i int) (string, int) {
	ch := condition[i]
	next := byte(0)
	if i+1 < len(condition) {
		next = condition[i+1]
	}
	switch {
	case ch == ';', ch == '-' && next == '-', ch == '/' && next == '*':
		return `The condition can't contain ";" or comments.`, 0
	case ch == '$':
		return "Dollar quoting isn't allowed.", 0
	case ch == '(':
		return "", 1
	case ch == ')':
		return "", -1
	}
	return "", 0
}

// closingQuote returns the index of the quote that closes the one at start (” and "" are escapes), or -1
func closingQuote(text string, start int) int {
	quote := text[start]
	for i := start + 1; i < len(text); i++ {
		if text[i] != quote {
			continue
		}
		if i+1 < len(text) && text[i+1] == quote {
			i++
			continue
		}
		return i
	}
	return -1
}

// testCheck asks Postgres whether the condition can be a CHECK on the table, and counts the records
// that break it, in a transaction that is always rolled back
func (s *tableConstraintService) testCheck(table, condition string) (dto.ConstraintCheckResult, error) {
	tx, err := s.repo.DB.Begin()
	if err != nil {
		return dto.ConstraintCheckResult{}, app_errors.WithMessage(app_errors.ConstraintFailed, app_errors.PostgresMessage(err))
	}
	defer func() { _ = tx.Rollback() }()

	// NOT VALID: Postgres checks the condition without scanning the records
	for _, stmt := range append(checkLimits, addCheckSQL(table, testCheckName, condition)+" NOT VALID") {
		if _, err := tx.Exec(stmt); err != nil {
			return dto.ConstraintCheckResult{}, app_errors.WithMessage(app_errors.ConstraintFailed, postgresPrefix+app_errors.PostgresMessage(err))
		}
	}
	var count int
	// IS FALSE, like Postgres: a record where the condition is NULL passes
	if err := tx.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s WHERE (%s) IS FALSE", table, condition)).Scan(&count); err != nil {
		return dto.ConstraintCheckResult{}, app_errors.WithMessage(app_errors.ConstraintFailed, postgresPrefix+app_errors.PostgresMessage(err))
	}
	if count > 0 {
		return dto.ConstraintCheckResult{Valid: false, Message: checkViolationMessage(count), Violations: count}, nil
	}
	return dto.ConstraintCheckResult{Valid: true}, nil
}

// applyCheck adds the CHECK constraint (replacing the old one with the same name when replace is set)
// and labels it with the category and condition, in one transaction, so a failure keeps what was
// there. It returns the IDs of the columns the condition uses.
func (s *tableConstraintService) applyCheck(table, name string, in dto.CheckInput, replace bool, columns map[string]tenant.Column) ([]string, error) {
	tx, err := s.repo.DB.Begin()
	if err != nil {
		return nil, app_errors.WithMessage(app_errors.ConstraintFailed, app_errors.PostgresMessage(err))
	}
	defer func() { _ = tx.Rollback() }()

	statements := append([]string{}, checkLimits...)
	if replace {
		statements = append(statements, dropConstraintSQL(table, name))
	}
	statements = append(statements, addCheckSQL(table, name, in.Condition), checkCommentSQL(table, name, in))
	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			logger.Get().Error().Err(err).Str("query", stmt).Msg("failed to apply check constraint")
			if _, broken := app_errors.CheckViolation(err); broken {
				// Release the table before counting the records in another transaction
				_ = tx.Rollback()
				return nil, s.existingRowsError(table, in.Condition)
			}
			return nil, app_errors.WithMessage(app_errors.ConstraintFailed, postgresPrefix+app_errors.PostgresMessage(err))
		}
	}
	columnIDs, err := checkColumnIDs(tx, table, name, columns)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, app_errors.WithMessage(app_errors.ConstraintFailed, app_errors.PostgresMessage(err))
	}
	return columnIDs, nil
}

// existingRowsError says how many records break the condition
func (s *tableConstraintService) existingRowsError(table, condition string) error {
	message := "Some records break this condition. Fix them or change the condition."
	if result, err := s.testCheck(table, condition); err == nil && result.Violations > 0 {
		message = result.Message
	}
	return app_errors.WithMessage(app_errors.CheckExistingRows, message)
}

// checkColumnIDs returns the IDs of the columns the constraint uses, as Postgres recorded them
func checkColumnIDs(tx *sql.Tx, table, name string, columns map[string]tenant.Column) ([]string, error) {
	rows, err := tx.Query(`SELECT a.attname FROM pg_constraint c
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
		WHERE c.conrelid = $1::regclass AND c.conname = $2 ORDER BY a.attnum`, table, name)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to read check constraint columns")
	}
	defer rows.Close()

	byName := make(map[string]string, len(columns))
	for id, c := range columns {
		byName[c.ColumnName] = id
	}
	ids := []string{}
	for rows.Next() {
		var attname string
		if err := rows.Scan(&attname); err != nil {
			return nil, app_errors.LogDatabaseError(err, "failed to read check constraint columns")
		}
		// System columns (created_time, …) have no field to show
		if id, ok := byName[attname]; ok {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// installedChecks returns the CHECK constraints Postgres has on the table, by name
func (s *tableConstraintService) installedChecks(table string) (map[string]installedCheck, error) {
	rows, err := s.repo.DB.Query(`SELECT conname, pg_get_constraintdef(oid), coalesce(obj_description(oid, 'pg_constraint'), '')
		FROM pg_constraint WHERE conrelid = $1::regclass AND contype = 'c'`, table)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to read check constraints")
	}
	defer rows.Close()

	checks := map[string]installedCheck{}
	for rows.Next() {
		var name, definition, comment string
		if err := rows.Scan(&name, &definition, &comment); err != nil {
			return nil, app_errors.LogDatabaseError(err, "failed to read check constraints")
		}
		var in dto.CheckInput
		// Constraints added outside the app have no label: show Postgres' own condition
		if json.Unmarshal([]byte(comment), &in) != nil || in.Condition == "" {
			in = dto.CheckInput{Category: defaultCheckCategory, Condition: conditionOf(definition)}
		}
		checks[name] = installedCheck{definition: definition, input: in}
	}
	return checks, rows.Err()
}

// listChecks builds the responses for the check constraints of a table
func (s *tableConstraintService) listChecks(table string, constraints []tenant.TableConstraint, columns map[string]tenant.Column) ([]dto.TableConstraintResponse, error) {
	installed, err := s.installedChecks(table)
	if err != nil {
		return nil, err
	}
	out := make([]dto.TableConstraintResponse, 0, len(constraints))
	for _, c := range constraints {
		check, ok := installed[c.ConstraintName]
		status := dto.ConstraintStatusOK
		if !ok {
			status = dto.ConstraintStatusMissing
		}
		resp := toConstraintResponse(c, columns, status)
		resp.CheckInput = check.input
		resp.TableName = table
		resp.Definition = check.definition
		out = append(out, resp)
	}
	return out, nil
}

// createCheck adds a CHECK constraint, then stores it; if storing fails the constraint is dropped again
func (s *tableConstraintService) createCheck(ctx context.Context, schemaName string, req dto.CreateTableConstraintRequest) (dto.TableConstraintResponse, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	in, err := checkInput(req.CheckInput)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	columns, err := s.columnsByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}

	// The popup may send the ID it used for the name in its SQL preview
	id, err := uuid.Parse(req.ID)
	if err != nil {
		id = uuid.New()
	}
	table := tableIdentifier(schemaName, model.Alias)
	name := checkConstraintName(model.Alias, id)
	columnIDs, err := s.applyCheck(table, name, in, false, columns)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}

	saved, err := s.repo.TableService.CreateRecord(tenant.TableConstraint{}.TableName(schemaName), map[string]interface{}{
		"id":               id.String(),
		"model_id":         req.ModelID,
		"type":             tenant.ConstraintTypeCheck,
		"title":            req.Title,
		"constraint_name":  name,
		"column_ids":       mustJSON(columnIDs),
		"created_by":       req.CreatedBy,
		"last_modified_by": req.CreatedBy,
	})
	if err != nil {
		if _, dropErr := s.repo.DB.Exec(dropConstraintSQL(table, name)); dropErr != nil {
			logger.Get().Error().Err(dropErr).Str("constraint", name).Msg("failed to drop check constraint after save failed")
		}
		return dto.TableConstraintResponse{}, app_errors.LogDatabaseError(err, "failed to create table constraint")
	}
	return checkResponse(saved, columns, table, in)
}

// updateCheck saves the title. A changed condition replaces the constraint in one transaction, so a
// failure keeps the old one; a constraint missing from Postgres is added again. A changed category
// only relabels it.
func (s *tableConstraintService) updateCheck(ctx context.Context, schemaName string, existing tenant.TableConstraint, req dto.UpdateTableConstraintRequest) (dto.TableConstraintResponse, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, existing.ModelID)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	in, err := checkInput(req.CheckInput)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	columns, err := s.columnsByID(ctx, schemaName, existing.ModelID)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}
	table := tableIdentifier(schemaName, model.Alias)
	installed, err := s.installedChecks(table)
	if err != nil {
		return dto.TableConstraintResponse{}, err
	}

	columnIDs := existing.ColumnIDs
	current, ok := installed[existing.ConstraintName]
	switch {
	case !ok || current.input.Condition != in.Condition:
		if columnIDs, err = s.applyCheck(table, existing.ConstraintName, in, true, columns); err != nil {
			return dto.TableConstraintResponse{}, err
		}
	case current.input.Category != in.Category:
		if _, err := s.repo.DB.Exec(checkCommentSQL(table, existing.ConstraintName, in)); err != nil {
			return dto.TableConstraintResponse{}, err
		}
	}

	updated, err := s.repo.TableService.UpdateRecord(tenant.TableConstraint{}.TableName(schemaName), existing.ID.String(), map[string]interface{}{
		"title":              req.Title,
		"column_ids":         mustJSON(columnIDs),
		"last_modified_by":   req.UpdatedBy,
		"last_modified_time": time.Now(),
	})
	if err != nil {
		return dto.TableConstraintResponse{}, app_errors.LogDatabaseError(err, "failed to update table constraint")
	}
	return checkResponse(updated, columns, table, in)
}

func checkResponse(row map[string]interface{}, columns map[string]tenant.Column, table string, in dto.CheckInput) (dto.TableConstraintResponse, error) {
	var out tenant.TableConstraint
	if err := helpers.MapToStruct(row, &out); err != nil {
		return dto.TableConstraintResponse{}, app_errors.ErrMapToStruct
	}
	resp := toConstraintResponse(out, columns, dto.ConstraintStatusOK)
	resp.CheckInput = in
	resp.TableName = table
	return resp, nil
}

// checkConstraintName is a unique Postgres name for a new check constraint, e.g. ck_employee_3f9a2c1b
func checkConstraintName(alias string, id uuid.UUID) string {
	return "ck" + strings.TrimPrefix(constraintName(alias, id), "uq")
}

// addCheckSQL is the statement the popup shows as its SQL preview
func addCheckSQL(table, name, condition string) string {
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s)", table, pq.QuoteIdentifier(name), condition)
}

// checkCommentSQL labels the constraint with its category and the condition as the user wrote it
func checkCommentSQL(table, name string, in dto.CheckInput) string {
	return fmt.Sprintf("COMMENT ON CONSTRAINT %s ON %s IS %s", pq.QuoteIdentifier(name), table, pq.QuoteLiteral(mustJSON(in)))
}

// conditionOf reads the condition from Postgres' definition, e.g. CHECK ((age > 0)) NOT VALID
func conditionOf(definition string) string {
	condition := strings.TrimSuffix(strings.TrimSpace(definition), " NOT VALID")
	condition = strings.TrimPrefix(condition, "CHECK (")
	return strings.TrimSuffix(condition, ")")
}

// checkViolationMessage says how many records break a condition
func checkViolationMessage(count int) string {
	if count == 1 {
		return "1 record breaks this condition. Fix it or change the condition."
	}
	return fmt.Sprintf("%d records break this condition. Fix them or change the condition.", count)
}
