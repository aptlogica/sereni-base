// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/lib/pq"
)

// Check constraints take a structured condition, never SQL text. It is the data API's filter tree:
//
//	{"all": [ {"column": "Email", "op": "contains", "value": "@"}, {"any": [ ... ]} ]}
//
// compileCondition turns it into the SQL inside CHECK (...). Column names come only from the table's own
// columns and are quoted; values become escaped literals; operators come from a fixed list.
const (
	maxConditionDepth      = 5
	maxConditionConditions = 50
)

// conditionOps are the operators a condition can use, and whether they take a value
var conditionOps = map[string]bool{
	"eq": true, "neq": true, "gt": true, "gte": true, "lt": true, "lte": true,
	"in": true, "contains": true, "starts_with": true, "ends_with": true,
	"is_null": false, "is_not_null": false, "not_empty": false,
}

var textOps = map[string]bool{"contains": true, "starts_with": true, "ends_with": true, "not_empty": true}

var plainIdentifier = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

type conditionCompiler struct {
	metas []*columnMeta
	used  []*columnMeta
	count int
}

// compileCondition returns the SQL condition and the columns it uses. Problems carry the path of the
// condition that is wrong, e.g. where.all[1].op.
func compileCondition(where map[string]interface{}, metas []*columnMeta) (string, []*columnMeta, error) {
	cc := &conditionCompiler{metas: metas}
	sql, err := cc.group(where, "where", 1)
	if err != nil {
		return "", nil, err
	}
	return sql, cc.used, nil
}

func (cc *conditionCompiler) fail(path, issue string) error {
	return NewError(CodeValidationFailed, "The check condition is not valid.", ErrorDetail{Field: path, Issue: issue})
}

func (cc *conditionCompiler) group(node map[string]interface{}, path string, depth int) (string, error) {
	if depth > maxConditionDepth {
		return "", cc.fail(path, fmt.Sprintf("groups can be nested at most %d deep", maxConditionDepth))
	}
	var key string
	for _, k := range []string{"all", "any", "none"} {
		if _, ok := node[k]; ok {
			if key != "" {
				return "", cc.fail(path, "a group has one of all, any or none")
			}
			key = k
		}
	}
	if key == "" || len(node) != 1 {
		return "", cc.fail(path, "must be an object with exactly one of all, any or none")
	}
	list, ok := node[key].([]interface{})
	if !ok || len(list) == 0 {
		return "", cc.fail(path+"."+key, "must be a non-empty array")
	}
	parts := make([]string, 0, len(list))
	for i, item := range list {
		itemPath := fmt.Sprintf("%s.%s[%d]", path, key, i)
		obj, ok := item.(map[string]interface{})
		if !ok {
			return "", cc.fail(itemPath, "must be an object")
		}
		var sql string
		var err error
		if _, isCond := obj["column"]; isCond {
			sql, err = cc.condition(obj, itemPath)
		} else {
			sql, err = cc.group(obj, itemPath, depth+1)
		}
		if err != nil {
			return "", err
		}
		parts = append(parts, sql)
	}
	switch key {
	case "all":
		return "(" + strings.Join(parts, " AND ") + ")", nil
	case "any":
		return "(" + strings.Join(parts, " OR ") + ")", nil
	}
	return "NOT (" + strings.Join(parts, " OR ") + ")", nil
}

func (cc *conditionCompiler) condition(node map[string]interface{}, path string) (string, error) {
	cc.count++
	if cc.count > maxConditionConditions {
		return "", cc.fail(path, fmt.Sprintf("at most %d conditions", maxConditionConditions))
	}
	for k := range node {
		if k != "column" && k != "op" && k != "value" {
			return "", cc.fail(path, fmt.Sprintf("unknown key %q; allowed: column, op, value", k))
		}
	}
	ref, _ := node["column"].(string)
	meta, err := pickMeta(cc.metas, ref)
	if err != nil {
		return "", cc.fail(path+".column", err.Error())
	}
	if meta.ColumnName == "" || linkOrLookup(meta.UIDT) {
		return "", cc.fail(path+".column", "links and lookups have no value of their own to check")
	}
	op, _ := node["op"].(string)
	takesValue, known := conditionOps[op]
	if !known {
		return "", cc.fail(path+".op", "unknown operator; allowed: eq, neq, gt, gte, lt, lte, in, contains, starts_with, ends_with, is_null, is_not_null, not_empty")
	}
	if textOps[op] && !isTextType(meta) {
		return "", cc.fail(path+".op", fmt.Sprintf("%q works on text columns only", op))
	}
	cc.use(meta)
	col := columnSQL(meta.ColumnName)
	if !takesValue {
		if _, given := node["value"]; given {
			return "", cc.fail(path+".value", fmt.Sprintf("%q takes no value", op))
		}
		switch op {
		case "is_null":
			return col + " IS NULL", nil
		case "is_not_null":
			return col + " IS NOT NULL", nil
		}
		return "(" + col + " IS NOT NULL AND btrim(" + col + ") <> '')", nil // not_empty
	}
	value, given := node["value"]
	if !given {
		return "", cc.fail(path+".value", "is required")
	}
	switch op {
	case "in":
		list, ok := value.([]interface{})
		if !ok || len(list) == 0 || len(list) > 100 {
			return "", cc.fail(path+".value", "must be an array of 1 to 100 values")
		}
		lits := make([]string, len(list))
		for i, v := range list {
			if lits[i], err = literal(v); err != nil {
				return "", cc.fail(fmt.Sprintf("%s.value[%d]", path, i), err.Error())
			}
		}
		return col + " IN (" + strings.Join(lits, ", ") + ")", nil
	case "contains", "starts_with", "ends_with":
		s, ok := value.(string)
		if !ok {
			return "", cc.fail(path+".value", "must be a string")
		}
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
		pattern := map[string]string{"contains": "%" + esc + "%", "starts_with": esc + "%", "ends_with": "%" + esc}[op]
		lit, err := literal(pattern)
		if err != nil {
			return "", cc.fail(path+".value", err.Error())
		}
		return col + " LIKE " + lit, nil
	}
	lit, err := literal(value)
	if err != nil {
		return "", cc.fail(path+".value", err.Error())
	}
	sqlOp := map[string]string{"eq": "=", "neq": "<>", "gt": ">", "gte": ">=", "lt": "<", "lte": "<="}[op]
	return col + " " + sqlOp + " " + lit, nil
}

func (cc *conditionCompiler) use(m *columnMeta) {
	for _, u := range cc.used {
		if u.ID == m.ID {
			return
		}
	}
	cc.used = append(cc.used, m)
}

func isTextType(m *columnMeta) bool {
	dt := strings.ToUpper(m.DT)
	return strings.HasPrefix(dt, "TEXT") || strings.HasPrefix(dt, "VARCHAR") || strings.HasPrefix(dt, "CHAR")
}

// columnSQL writes a physical column name; plain names stay unquoted, as the app writes them
func columnSQL(name string) string {
	if plainIdentifier.MatchString(name) {
		return name
	}
	return pq.QuoteIdentifier(name)
}

// literal writes a JSON value as an SQL literal: strings are single-quoted with quotes doubled, numbers are
// checked to be finite, booleans are true or false
func literal(v interface{}) (string, error) {
	switch x := v.(type) {
	case string:
		if strings.ContainsRune(x, 0) {
			return "", fmt.Errorf("text can't contain a NUL character")
		}
		return "'" + strings.ReplaceAll(x, "'", "''") + "'", nil
	case bool:
		if x {
			return "TRUE", nil
		}
		return "FALSE", nil
	case json.Number:
		f, err := x.Float64()
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return "", fmt.Errorf("must be a finite number")
		}
		return x.String(), nil
	case float64:
		if math.IsInf(x, 0) || math.IsNaN(x) {
			return "", fmt.Errorf("must be a finite number")
		}
		return fmt.Sprintf("%v", x), nil
	}
	return "", fmt.Errorf("must be a string, number or true/false")
}
