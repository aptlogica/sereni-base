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
// key equal to another record's, naming the field or the record that already holds the key. Other fields, and
// tables keyed by id alone, are not checked.
func (s tableManagementService) checkKeyValues(ctx context.Context, schemaName string, model tenant.Model, rowID int, values map[string]interface{}) error {
	keyCols, err := s.primaryKeyColumns(ctx, schemaName, model.Alias)
	if err != nil || slices.Equal(keyCols, []string{constant.PrimaryKeyIDColumn}) {
		return err
	}
	changed := false
	for _, c := range keyCols {
		if _, ok := values[c]; ok {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	fields := s.keyFields(ctx, schemaName, model, keyCols)

	for _, c := range keyCols {
		if v, ok := values[c]; ok && isEmptyKeyValue(v) {
			return keyEmptyError(fieldTitle(fields, c), fields)
		}
	}
	if s.repo.DB == nil {
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
	args = append(args, rowID)
	query := fmt.Sprintf(
		`SELECT o.id::bigint, COALESCE(o.title::text, ''), jsonb_build_object(%s) FROM %s o JOIN %s cur ON cur.id = $%d WHERE o.id <> $%d AND %s LIMIT 1`,
		strings.Join(pairs, ", "), table, table, len(args), len(args), strings.Join(conditions, " AND "))

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
		return app_errors.LogDatabaseError(err, msg)
	}
	if pqErr.Code != "23505" && pqErr.Code != "23502" {
		return app_errors.LogDatabaseError(err, msg)
	}

	model, modelErr := s.modelByAlias(schemaName, alias)
	keyCols, _ := s.primaryKeyColumns(ctx, schemaName, alias)
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
	match := keyDetailRegexp.FindStringSubmatch(pqErr.Detail)
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

func keyAPIError(code responseConst.ResponseCode, status int, message string, details map[string]interface{}) error {
	raw, _ := json.Marshal(details)
	return &app_errors.APIError{Code: string(code), Message: message, Details: string(raw), StatusCode: status}
}
