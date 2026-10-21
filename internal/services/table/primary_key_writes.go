// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"
	"github.com/lib/pq"
)

// keyDetailRegexp reads Postgres' unique violation detail: Key (code, dept)=(HR, IN) already exists.
var keyDetailRegexp = regexp.MustCompile(`^Key \((.+?)\)=\((.*)\) (?:already exists|is duplicated)\.?$`)

// keyField is a primary key column with its field title
type keyField struct {
	Column string
	Title  string
}

// primaryKeyColumns returns the primary key column names of a table, in key order
func (s tableManagementService) primaryKeyColumns(ctx context.Context, schemaName, alias string) ([]string, error) {
	rows, err := s.repo.TableService.GetByFunction(ctx, fmt.Sprintf("%s.%s", constant.MasterDatabase, keyColumnsFunctionName),
		map[string]interface{}{"p_table": fmt.Sprintf(SchemaTableFormat, schemaName, alias)})
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to read primary key")
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return textArray(rows[0][keyColumnsFunctionName]), nil
}

// keyFields returns the key columns of a table with their titles; titles fall back to column names
func (s tableManagementService) keyFields(ctx context.Context, schemaName string, model tenant.Model, keyCols []string) []keyField {
	titles := map[string]string{}
	if columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, model.ID.String()); err == nil {
		for _, col := range columns {
			titles[col.ColumnName] = col.Title
		}
	}
	fields := make([]keyField, len(keyCols))
	for i, c := range keyCols {
		fields[i] = keyField{Column: c, Title: titles[c]}
		if fields[i].Title == "" {
			fields[i].Title = c
		}
	}
	return fields
}

// checkKeyValues rejects saving key fields (new values by column name) that are empty or that make the record's
// key equal to another record's, naming the fields or the record that already holds the key. rowID 0 checks a
// new record, which must have every key field. Other fields, and tables keyed by id alone, are not checked.
func (s tableManagementService) checkKeyValues(ctx context.Context, schemaName string, model tenant.Model, rowID int, values map[string]interface{}) error {
	keyCols, err := s.primaryKeyColumns(ctx, schemaName, model.Alias)
	if err != nil || len(keyCols) == 0 || slices.Equal(keyCols, []string{constant.PrimaryKeyIDColumn}) {
		return err
	}
	insert := rowID <= 0
	changed := insert
	for _, c := range keyCols {
		if _, ok := values[c]; ok {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	fields := s.keyFields(ctx, schemaName, model, keyCols)

	// A new record needs every key field except id, which the database assigns; an update only the ones it changes
	var empty []string
	for _, c := range keyCols {
		v, given := values[c]
		if c == constant.PrimaryKeyIDColumn && !given {
			continue
		}
		if (given || insert) && isEmptyKeyValue(v) {
			empty = append(empty, fieldTitle(fields, c))
		}
	}
	if len(empty) > 0 {
		return keyMissingError(empty, fields)
	}
	return s.checkKeyClash(schemaName, model, rowID, keyCols, fields, values)
}

// checkKeyClash rejects a record whose key equals another record's, naming the record that already holds it
func (s tableManagementService) checkKeyClash(schemaName string, model tenant.Model, rowID int, keyCols []string, fields []keyField, values map[string]interface{}) error {
	insert := rowID <= 0
	// A new record's id is new, so a key that includes id can't be used by another record
	if s.repo.DB == nil || (insert && slices.Contains(keyCols, constant.PrimaryKeyIDColumn)) {
		return nil
	}

	// Another record with the same key: the new values where given, this record's values elsewhere
	table := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	conditions := make([]string, 0, len(keyCols))
	pairs := make([]string, 0, len(keyCols))
	args := []interface{}{}
	for _, c := range keyCols {
		col := quoteIdentifier(c)
		if v, ok := values[c]; ok {
			args = append(args, v)
			conditions = append(conditions, fmt.Sprintf("o.%s = $%d", col, len(args)))
		} else {
			conditions = append(conditions, fmt.Sprintf("o.%s = cur.%s", col, col))
		}
		pairs = append(pairs, fmt.Sprintf("'%s', o.%s", strings.ReplaceAll(c, "'", "''"), col))
	}
	var query string
	if insert {
		query = fmt.Sprintf(`SELECT o.id::bigint, COALESCE(o.title::text, ''), jsonb_build_object(%s) FROM %s o WHERE %s LIMIT 1`,
			strings.Join(pairs, ", "), table, strings.Join(conditions, " AND "))
	} else {
		args = append(args, rowID)
		query = fmt.Sprintf(
			`SELECT o.id::bigint, COALESCE(o.title::text, ''), jsonb_build_object(%s) FROM %s o JOIN %s cur ON cur.id = $%d WHERE o.id <> $%d AND %s LIMIT 1`,
			strings.Join(pairs, ", "), table, table, len(args), len(args), strings.Join(conditions, " AND "))
	}

	var otherID int64
	var otherTitle string
	var rawValues []byte
	if err := s.repo.DB.QueryRow(query, args...).Scan(&otherID, &otherTitle, &rawValues); err != nil {
		// No clash, or a value Postgres can't compare (e.g. text in a number field): the write reports it
		return nil
	}
	var found map[string]interface{}
	_ = json.Unmarshal(rawValues, &found)
	return keyDuplicateError(fields, found, &dto.PrimaryKeyRecord{ID: otherID, Title: otherTitle})
}

// keyWriteError turns a Postgres key violation from a write to a user table into an exact error; other errors
// become the usual database error
func (s tableManagementService) keyWriteError(ctx context.Context, schemaName, alias string, err error, msg string) error {
	var pqErr *pq.Error
	var violation *app_errors.KeyViolation
	switch {
	case errors.As(err, &violation):
		pqErr = violation.Postgres
	case !errors.As(err, &pqErr):
		var apiErr *app_errors.APIError
		if errors.As(err, &apiErr) {
			return err
		}
		// Layers that keep only the text of a duplicate: the constraint name tells the unique key
		return s.recordWriteError(ctx, schemaName, err, msg)
	}
	if _, ok := app_errors.CheckViolation(pqErr); ok {
		return s.checkViolationError(schemaName, pqErr.Constraint)
	}
	if pqErr.Code != "23505" && pqErr.Code != "23502" {
		return app_errors.LogDatabaseError(err, msg)
	}

	keyCols, _ := s.primaryKeyColumns(ctx, schemaName, alias)
	// A duplicate rejected by a unique key, not by the primary key, gets the unique key's message
	if pqErr.Code == "23505" && s.isUniqueKeyViolation(schemaName, pqErr, keyCols) {
		return s.recordWriteError(ctx, schemaName, pqErr, msg)
	}
	// An empty field outside the primary key is not a key error
	if pqErr.Code == "23502" && pqErr.Column != "" && !slices.Contains(keyCols, pqErr.Column) {
		return app_errors.LogDatabaseError(err, msg)
	}

	model, modelErr := s.modelByAlias(schemaName, alias)
	var fields []keyField
	if modelErr == nil {
		fields = s.keyFields(ctx, schemaName, model, keyCols)
	} else {
		for _, c := range keyCols {
			fields = append(fields, keyField{Column: c, Title: c})
		}
	}

	if pqErr.Code == "23502" {
		return keyEmptyError(fieldTitle(fields, pqErr.Column), fields)
	}
	return keyDuplicateFromDetail(fields, pqErr.Detail)
}

// keyDuplicateFromDetail builds the duplicate key error from the detail Postgres gives, e.g. Key (a, b)=(1, 2) already exists
func keyDuplicateFromDetail(fields []keyField, detail string) error {
	match := keyDetailRegexp.FindStringSubmatch(detail)
	if match == nil {
		return keyDuplicateError(fields, nil, nil)
	}
	names := strings.Split(match[1], ", ")
	rawValues := splitKeyValues(match[2], len(names))
	values := map[string]interface{}{}
	for i, name := range names {
		name = strings.Trim(name, `"`)
		if rawValues != nil {
			values[name] = rawValues[i]
		}
	}
	if rawValues == nil {
		values = nil
	}
	return keyDuplicateError(fields, values, nil)
}

// isUniqueKeyViolation tells a duplicate rejected by a unique key from one rejected by the primary key: the
// constraint is a unique key of the table, or Postgres' detail names columns other than the key columns
func (s tableManagementService) isUniqueKeyViolation(schemaName string, pqErr *pq.Error, keyCols []string) bool {
	if pqErr.Constraint != "" && s.constraintByName(schemaName, pqErr.Constraint) != nil {
		return true
	}
	match := keyDetailRegexp.FindStringSubmatch(pqErr.Detail)
	if match == nil || len(keyCols) == 0 {
		return false
	}
	names := strings.Split(match[1], ", ")
	for i, name := range names {
		names[i] = strings.Trim(name, `"`)
	}
	slices.Sort(names)
	sortedKey := slices.Sorted(slices.Values(keyCols))
	return !slices.Equal(names, sortedKey)
}

func (s tableManagementService) modelByAlias(schemaName, alias string) (tenant.Model, error) {
	rows, err := s.repo.TableService.GetTableData(tenant.Model{}.TableName(schemaName), dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{{Column: "alias", Operator: "eq", Value: alias}},
	})
	if err != nil || len(rows) == 0 {
		return tenant.Model{}, app_errors.TableNotFound
	}
	var model tenant.Model
	if err := helpers.MapToStruct(rows[0], &model); err != nil {
		return tenant.Model{}, err
	}
	return model, nil
}

// splitQualifiedTable reads "schema"."table" back into its parts
func splitQualifiedTable(qualified string) (string, string) {
	parts := strings.SplitN(qualified, `"."`, 2)
	if len(parts) != 2 {
		return constant.MasterDatabase, strings.Trim(qualified, `"`)
	}
	return strings.Trim(parts[0], `"`), strings.Trim(parts[1], `"`)
}

func isEmptyKeyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	return false
}

func fieldTitle(fields []keyField, column string) string {
	for _, f := range fields {
		if f.Column == column {
			return f.Title
		}
	}
	return column
}

func keyTitles(fields []keyField) string {
	titles := make([]string, len(fields))
	for i, f := range fields {
		titles[i] = f.Title
	}
	return strings.Join(titles, " + ")
}

// splitKeyValues splits "HR, IN" into one value per key column; nil when a value itself contains ", "
func splitKeyValues(raw string, count int) []string {
	if count == 1 {
		return []string{raw}
	}
	parts := strings.Split(raw, ", ")
	if len(parts) != count {
		return nil
	}
	return parts
}

func formatWriteValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "(empty)"
	case string:
		if _, err := strconv.ParseFloat(val, 64); err == nil {
			return val
		}
		return strconv.Quote(val)
	default:
		return fmt.Sprint(val)
	}
}

// keyMissingError names every empty key field: "Code and Year can't be empty: they are part of the primary key
// (Code + Year)."
func keyMissingError(titles []string, fields []keyField) error {
	if len(titles) == 1 {
		return keyEmptyError(titles[0], fields)
	}
	list := strings.Join(titles[:len(titles)-1], ", ") + " and " + titles[len(titles)-1]
	message := fmt.Sprintf("%s can't be empty: they are part of the primary key (%s).", list, keyTitles(fields))
	return keyAPIError(responseConst.PrimaryKeyError.EmptyValue, http.StatusBadRequest, message, map[string]interface{}{"fields": titles})
}

// keyEmptyError: "Code can't be empty: it is part of the primary key (Code + Dept)."
func keyEmptyError(title string, fields []keyField) error {
	message := fmt.Sprintf("%s can't be empty: it is part of the primary key.", title)
	if len(fields) > 1 {
		message = fmt.Sprintf("%s can't be empty: it is part of the primary key (%s).", title, keyTitles(fields))
	}
	return keyAPIError(responseConst.PrimaryKeyError.EmptyValue, http.StatusBadRequest, message, map[string]interface{}{"field": title})
}

// keyDuplicateError: "The combination Code = "HR", Dept = "IN" is already used by record #4 (Acme). Each record
// needs a unique primary key (Code + Dept)."
func keyDuplicateError(fields []keyField, values map[string]interface{}, other *dto.PrimaryKeyRecord) error {
	subject := keyValuesSubject(fields, values)
	holder := "another record"
	if other != nil {
		holder = fmt.Sprintf("record #%d", other.ID)
		if other.Title != "" {
			holder += fmt.Sprintf(" (%s)", other.Title)
		}
	}
	message := fmt.Sprintf("%s is already used by %s. Each record needs a unique primary key", subject, holder)
	if len(fields) > 1 {
		message += fmt.Sprintf(" (%s)", keyTitles(fields))
	}
	details := map[string]interface{}{"fields": keyTitles(fields), "values": values}
	if other != nil {
		details["record_id"] = other.ID
	}
	return keyAPIError(responseConst.PrimaryKeyError.DuplicateValue, http.StatusConflict, message+".", details)
}

// keyValuesSubject names the key values of an error: a single field, or the combination of several
func keyValuesSubject(fields []keyField, values map[string]interface{}) string {
	subject := "This primary key value"
	if values != nil {
		parts := make([]string, 0, len(fields))
		for _, f := range fields {
			if v, ok := values[f.Column]; ok {
				parts = append(parts, fmt.Sprintf("%s = %s", f.Title, formatWriteValue(v)))
			}
		}
		if len(parts) > 0 {
			subject = strings.Join(parts, ", ")
			if len(parts) > 1 {
				subject = "The combination " + subject
			}
		}
	}
	return subject
}

func keyAPIError(code responseConst.ResponseCode, status int, message string, details map[string]interface{}) error {
	raw, _ := json.Marshal(details)
	return &app_errors.APIError{Code: string(code), Message: message, Details: string(raw), StatusCode: status}
}
