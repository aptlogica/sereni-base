// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aptlogica/sereni-base/internal/dto"
)

// Row filters of an export. A filter is the list of conditions a table view's filter builder makes, and it works the
// same way: the same operators for the same field types, and conditions joined from first to last with AND or OR
// (no grouping). The rows are read first, then the ones that do not match are dropped, before they get a _rid, so
// the rest of the export (links, files, row counts) only sees the rows that stay.

const (
	maxFilterConditions = 20
	maxFilterValueLen   = 1000

	opIsEqual       = "is equal"
	opIsNotEqual    = "is not equal"
	opContains      = "contains"
	opNotContain    = "does not contain"
	opIsEmpty       = "is empty"
	opIsNotEmpty    = "is not empty"
	opGreater       = "greater than"
	opLess          = "less than"
	opGreaterEqual  = "greater than or equal"
	opLessEqual     = "less than or equal"
	opBefore        = "before"
	opAfter         = "after"
	opIsChecked     = "is checked"
	opIsNotChecked  = "is not checked"
	opContainsAny   = "contains any of"
	opNotContainAny = "does not contain any of"
)

var (
	stringOps = []string{opIsEqual, opIsNotEqual, opContains, opNotContain, opIsEmpty, opIsNotEmpty}
	numberOps = []string{opIsEqual, opIsNotEqual, opLess, opLessEqual, opGreater, opGreaterEqual, opIsEmpty, opIsNotEmpty}
	dateOps   = []string{opIsEqual, opIsNotEqual, opBefore, opAfter, opIsEmpty, opIsNotEmpty}
	choiceOps = []string{opIsEqual, opIsNotEqual, opContainsAny, opNotContainAny, opIsEmpty, opIsNotEmpty}
	plainOps  = []string{opIsEqual, opIsNotEqual, opIsEmpty, opIsNotEmpty}

	// filterOperators are the operators of each field type, as the filter builder of a table view offers them
	filterOperators = map[string][]string{
		"text": stringOps, "email": stringOps, "url": stringOps, "uuid": stringOps, "longText": stringOps,
		"number": numberOps, "decimal": numberOps, "currency": numberOps, "percent": numberOps, "year": numberOps,
		"rating": numberOps, "duration": numberOps,
		"date": dateOps, "time": dateOps,
		"select": choiceOps, "multiSelect": choiceOps,
		"boolean": {opIsChecked, opIsNotChecked},
	}

	// notFilterable are the field types a row filter cannot use, the same as in a table view's filter builder
	notFilterable = map[string]bool{
		"links": true, "lookup": true, "datetime": true, "attachment": true, "json": true, "id": true,
		"createdTime": true, "lastModifiedTime": true, "createdBy": true, "lastModifiedBy": true,
	}
)

// isFilterableField reports whether a row filter can use the field
func isFilterableField(c dto.ColumnResponse) bool {
	return !boolOf(c.System) && !notFilterable[c.UIDT]
}

// operatorsOf returns the operators a field type allows
func operatorsOf(uidt string) []string {
	if ops, ok := filterOperators[uidt]; ok {
		return ops
	}
	return plainOps
}

// normalizeOperator makes an operator comparable: lower case, and the spelling "does not contains any of" that a
// table view's filter builder writes is read as "does not contain any of"
func normalizeOperator(op string) string {
	op = strings.ToLower(strings.TrimSpace(op))
	if op == "does not contains any of" {
		return opNotContainAny
	}
	return op
}

func operatorRequiresValue(op string) bool {
	switch op {
	case opIsEmpty, opIsNotEmpty, opIsChecked, opIsNotChecked:
		return false
	}
	return true
}

// filterOptions returns the choices of a select or multi-select field
func filterOptions(c dto.ColumnResponse) []string {
	var out []string
	if list, ok := c.Meta["options"].([]interface{}); ok {
		for _, item := range list {
			switch x := item.(type) {
			case string:
				out = append(out, x)
			case map[string]interface{}:
				if option, ok := x["option"].(string); ok {
					out = append(out, option)
				}
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Matching
// ---------------------------------------------------------------------------

// filterRecords keeps the records that match the filter, in order.
func filterRecords(records []map[string]interface{}, columns []dto.ColumnResponse, f dto.ExportTableFilter) []map[string]interface{} {
	if len(f.Conditions) == 0 {
		return records
	}
	byID := map[string]dto.ColumnResponse{}
	for _, c := range columns {
		byID[c.ID.String()] = c
	}
	out := make([]map[string]interface{}, 0, len(records))
	for _, rec := range records {
		if recordMatches(rec, byID, f.Conditions) {
			out = append(out, rec)
		}
	}
	return out
}

// recordMatches combines the conditions from first to last: the first one starts the result, each next one is joined
// to it with its own logic (AND unless it says OR).
func recordMatches(rec map[string]interface{}, columns map[string]dto.ColumnResponse, conditions []dto.ExportFilterCondition) bool {
	result := true
	for i, cond := range conditions {
		col, ok := columns[cond.Column]
		matches := true // a condition on a field that is gone does not filter (the request is validated before)
		if ok {
			matches = conditionMatches(rec[col.ColumnName], col.UIDT, normalizeOperator(cond.Operator), cond.Value)
		}
		switch {
		case i == 0:
			result = matches
		case strings.EqualFold(cond.Logic, "OR"):
			result = result || matches
		default:
			result = result && matches
		}
	}
	return result
}

func conditionMatches(raw interface{}, uidt, op, val string) bool {
	if op == opIsEmpty {
		return isEmptyValue(raw)
	}
	if op == opIsNotEmpty {
		return !isEmptyValue(raw)
	}
	switch uidt {
	case "multiSelect":
		have, want := multiValues(raw), multiValues(val)
		switch op {
		case opIsEqual:
			return sameSet(have, want)
		case opIsNotEqual:
			return !sameSet(have, want)
		case opContainsAny:
			return anyIn(have, want)
		case opNotContainAny:
			return !anyIn(have, want)
		}
		return true
	case "select":
		s := textValue(raw)
		switch op {
		case opIsEqual:
			return s == val
		case opIsNotEqual:
			return s != val
		case opContainsAny:
			return anyIn([]string{s}, multiValues(val))
		case opNotContainAny:
			return !anyIn([]string{s}, multiValues(val))
		}
		return true
	case "boolean":
		checked := isTrue(raw)
		switch op {
		case opIsChecked:
			return checked
		case opIsNotChecked:
			return !checked
		}
		return true
	case "number", "decimal", "currency", "percent", "year", "rating", "duration":
		n, okN := toFloat(raw)
		t, okT := toFloat(val)
		if !okN || !okT {
			return false
		}
		switch op {
		case opIsEqual:
			return n == t
		case opIsNotEqual:
			return n != t
		case opGreater:
			return n > t
		case opLess:
			return n < t
		case opGreaterEqual:
			return n >= t
		case opLessEqual:
			return n <= t
		}
		return true
	case "date":
		d, okD := toTime(raw)
		t, okT := toTime(val)
		if !okD || !okT {
			return false
		}
		switch op {
		case opIsEqual:
			return d.Equal(t)
		case opIsNotEqual:
			return !d.Equal(t)
		case opBefore:
			return d.Before(t)
		case opAfter:
			return d.After(t)
		}
		return true
	case "time":
		// a time of day is written as HH:MM[:SS], so the text order is the time order
		s, t := textValue(raw), val
		switch op {
		case opIsEqual:
			return s == t
		case opIsNotEqual:
			return s != t
		case opBefore:
			return s < t
		case opAfter:
			return s > t
		}
		return true
	default:
		s := textValue(raw)
		switch op {
		case opIsEqual:
			return s == val
		case opIsNotEqual:
			return s != val
		case opContains:
			return strings.Contains(strings.ToLower(s), strings.ToLower(val))
		case opNotContain:
			return !strings.Contains(strings.ToLower(s), strings.ToLower(val))
		}
		return true
	}
}

func isEmptyValue(v interface{}) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(x) == ""
	case []interface{}:
		return len(x) == 0
	case []string:
		return len(x) == 0
	}
	return false
}

// textValue is a value as text; a missing value is empty
func textValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64, float32, json.Number:
		return cellKey(x)
	}
	return fmt.Sprint(v)
}

func isTrue(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case nil:
		return false
	}
	s := strings.ToLower(textValue(v))
	return s == "true" || s == "1"
}

func toFloat(v interface{}) (float64, bool) {
	if v == nil {
		return 0, false
	}
	s := strings.TrimSpace(textValue(v))
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

var dateLayouts = []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"}

func toTime(v interface{}) (time.Time, bool) {
	if v == nil {
		return time.Time{}, false
	}
	s := strings.TrimSpace(textValue(v))
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// multiValues reads a list of choices: a list, a JSON list in text, or a single value; a comma separates choices in
// plain text. An item written as {"option": "A"} is the choice A.
func multiValues(v interface{}) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if obj, ok := item.(map[string]interface{}); ok {
				if option, ok := obj["option"]; ok {
					item = option
				}
			}
			out = append(out, textValue(item))
		}
		return out
	case []string:
		return x
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		if strings.HasPrefix(s, "[") {
			var list []interface{}
			if json.Unmarshal([]byte(s), &list) == nil {
				return multiValues(list)
			}
		}
		parts := strings.Split(s, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return []string{textValue(v)}
}

func sameSet(a, b []string) bool {
	sa, sb := map[string]struct{}{}, map[string]struct{}{}
	for _, x := range a {
		sa[x] = struct{}{}
	}
	for _, x := range b {
		sb[x] = struct{}{}
	}
	if len(sa) != len(sb) {
		return false
	}
	for x := range sa {
		if _, ok := sb[x]; !ok {
			return false
		}
	}
	return true
}

func anyIn(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// matchesTable reports whether a table is named by id, title or alias (case does not matter)
func matchesTable(r dto.TableResponse, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name != "" && (name == strings.ToLower(r.Model.ID.String()) || name == strings.ToLower(r.Model.Title) ||
		name == strings.ToLower(r.Model.Alias) || strings.TrimPrefix(name, "t_") == strings.ToLower(r.Model.Title))
}

// resolveFilters maps the filters, which name tables by id, title or alias, to the id of each table.
func resolveFilters(responses []dto.TableResponse, filters map[string]dto.ExportTableFilter) map[string]dto.ExportTableFilter {
	out := map[string]dto.ExportTableFilter{}
	for name, f := range filters {
		for _, r := range responses {
			if matchesTable(r, name) && len(f.Conditions) > 0 {
				out[r.Model.ID.String()] = f
				break
			}
		}
	}
	return out
}

// ValidateExportFilters checks the row filters of an export before it starts, so a mistake is refused at once with
// words for the user instead of failing a background job. It lists every problem it finds.
func (s *baseTransferService) ValidateExportFilters(ctx context.Context, schemaName, baseID string, tables []string, filters map[string]dto.ExportTableFilter) error {
	if len(filters) == 0 {
		return nil
	}
	responses, err := s.tableService.GetModelByBaseID(ctx, schemaName, baseID)
	if err != nil {
		return err
	}
	selected := responses
	if len(tables) > 0 {
		selected = filterTableResponses(responses, tables)
	}

	var problems []string
	for name, f := range filters {
		var table *dto.TableResponse
		for i := range responses {
			if matchesTable(responses[i], name) {
				table = &responses[i]
				break
			}
		}
		if table == nil {
			problems = append(problems, fmt.Sprintf("the filter names a table that does not exist: %q", name))
			continue
		}
		inExport := false
		for _, r := range selected {
			if r.Model.ID == table.Model.ID {
				inExport = true
			}
		}
		if !inExport {
			problems = append(problems, fmt.Sprintf("the table %q has a filter but is not part of the export", table.Model.Title))
			continue
		}
		cols, err := s.tableService.GetColumnsByModelID(ctx, schemaName, table.Model.ID.String())
		if err != nil {
			return err
		}
		live := make([]dto.ColumnResponse, 0, len(cols))
		for _, c := range cols {
			if !boolOf(c.Deleted) {
				live = append(live, c)
			}
		}
		problems = append(problems, validateTableFilter(table.Model.Title, live, f)...)
	}
	if len(problems) > 0 {
		return &dto.ExportFilterError{Problems: problems}
	}
	return nil
}

// validateTableFilter lists what is wrong with the conditions of one table's filter.
func validateTableFilter(title string, columns []dto.ColumnResponse, f dto.ExportTableFilter) []string {
	var problems []string
	if len(f.Conditions) > maxFilterConditions {
		return []string{fmt.Sprintf("the filter of %s has more than %d conditions", title, maxFilterConditions)}
	}
	byID := map[string]dto.ColumnResponse{}
	for _, c := range columns {
		byID[c.ID.String()] = c
	}
	for i, cond := range f.Conditions {
		where := fmt.Sprintf("filter of %s, condition %d", title, i+1)
		col, ok := byID[cond.Column]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: the field %q does not exist", where, cond.Column))
			continue
		}
		if !isFilterableField(col) {
			problems = append(problems, fmt.Sprintf("%s: the field %s (%s) cannot be filtered", where, col.Title, col.UIDT))
			continue
		}
		op := normalizeOperator(cond.Operator)
		allowed := false
		for _, a := range operatorsOf(col.UIDT) {
			allowed = allowed || a == op
		}
		if !allowed {
			problems = append(problems, fmt.Sprintf("%s: %q cannot be used on the field %s (%s)", where, cond.Operator, col.Title, col.UIDT))
			continue
		}
		if i > 0 && cond.Logic != "" && !strings.EqualFold(cond.Logic, "AND") && !strings.EqualFold(cond.Logic, "OR") {
			problems = append(problems, fmt.Sprintf("%s: the logic must be AND or OR", where))
		}
		if !operatorRequiresValue(op) {
			continue
		}
		val := strings.TrimSpace(cond.Value)
		switch {
		case val == "":
			problems = append(problems, fmt.Sprintf("%s: a value is needed for %q", where, cond.Operator))
		case len(cond.Value) > maxFilterValueLen:
			problems = append(problems, fmt.Sprintf("%s: the value is too long", where))
		default:
			switch col.UIDT {
			case "number", "decimal", "currency", "percent", "year", "rating", "duration":
				if _, ok := toFloat(val); !ok {
					problems = append(problems, fmt.Sprintf("%s: %q is not a number", where, cond.Value))
				}
			case "date":
				if _, ok := toTime(val); !ok {
					problems = append(problems, fmt.Sprintf("%s: %q is not a date (use YYYY-MM-DD)", where, cond.Value))
				}
			}
		}
	}
	return problems
}
