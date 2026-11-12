// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

const (
	// idColumn is the system ID column, the primary key unless the table's key was changed
	idColumn = "id"
	// byKeySegment is the record path that finds a record by its named key fields: records/by-key?key[SKU]=A1
	byKeySegment = "by-key"
)

var keyParamPattern = regexp.MustCompile(`^key\[([^\[\]]+)\]$`)

// recordKey is what finds a table's records: its primary key columns and their fields, and whether that is the
// default key (id)
type recordKey struct {
	tc     *tableContext
	cols   []string
	fields []*field // the field of each key column; nil when a column has none
	idKey  bool
}

// recordKeyOf reads the table's primary key, once per request
func (a *API) recordKeyOf(ctx context.Context, tc *tableContext) (*recordKey, error) {
	if tc.key != nil {
		return tc.key, nil
	}
	cols, err := a.primaryKeyColumns(ctx, tc)
	if err != nil {
		return nil, err
	}
	k := &recordKey{tc: tc, cols: cols, fields: make([]*field, len(cols)),
		idKey: len(cols) == 0 || (len(cols) == 1 && cols[0] == idColumn)}
	for i, col := range cols {
		for _, f := range tc.fields {
			if f.ColumnName == col {
				k.fields[i] = f
				break
			}
		}
	}
	tc.key = k
	return k, nil
}

// titles are the key fields' names, in key order
func (k *recordKey) titles() []string {
	out := make([]string, len(k.cols))
	for i, col := range k.cols {
		out[i] = col
		if k.fields[i] != nil {
			out[i] = k.fields[i].Name
		}
	}
	return out
}

func (k *recordKey) title() string { return strings.Join(k.titles(), " + ") }

// pathHint tells how to address a record of the table on the record path
func (k *recordKey) pathHint() string {
	titles := k.titles()
	params := make([]string, len(titles))
	for i, t := range titles {
		params[i] = fmt.Sprintf("key[%s]=…", t)
	}
	byKey := "records/" + byKeySegment + "?" + strings.Join(params, "&")
	if len(titles) == 1 {
		return fmt.Sprintf("records/<%s> or %s", titles[0], byKey)
	}
	return byKey
}

// objectHint is the key object for a bulk request: {"SKU": …, "Region": …}
func (k *recordKey) objectHint() string {
	titles := k.titles()
	parts := make([]string, len(titles))
	for i, t := range titles {
		parts[i] = fmt.Sprintf("%q: …", t)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// keyValues is the record's key for responses, {"SKU": "A1", "Region": "EU"}; nil with the default key
func (k *recordKey) keyValues(row map[string]interface{}, keys map[*field]string) orderedFields {
	if k == nil || k.idKey {
		return nil
	}
	out := make(orderedFields, len(k.cols))
	for i, col := range k.cols {
		name, v := col, row[col]
		if col == idColumn {
			v = EncodeRecordID(rowID(row))
		}
		if f := k.fields[i]; f != nil {
			name, v = f.Name, apiValue(f, row[col])
			if key, ok := keys[f]; ok {
				name = key
			}
		}
		out[i] = fieldValue{key: name, value: v}
	}
	return out
}

// recordRef is how a request names one record: a record ID or single key value (ref), or the key fields'
// values (key: field name or ID to value)
type recordRef struct {
	ref string
	key map[string]interface{}
}

// label names the record in errors and delete results: the ID or key value sent, or the key object as JSON
func (r recordRef) label() string {
	if r.key == nil {
		return r.ref
	}
	b, _ := json.Marshal(r.key)
	return string(b)
}

// recordFromPath reads the {record} path segment. With the default key that is the record ID (rec_12) or its
// number; with a key of one other field, that field's value; records/by-key?key[SKU]=A1&key[Region]=EU names
// each key field. A record ID is refused when the key is other fields.
func (a *API) recordFromPath(c *gin.Context, tc *tableContext) (int64, error) {
	k, err := a.recordKeyOf(c, tc)
	if err != nil {
		return 0, err
	}
	ref := recordRef{ref: c.Param("record")}
	if ref.ref == byKeySegment {
		key, apiErr := keyFromQuery(c.Request.URL.Query(), k)
		if apiErr != nil {
			return 0, apiErr
		}
		ref = recordRef{key: key}
	}
	id, apiErr, err := a.findRecord(c, k, ref, "path")
	if err != nil {
		return 0, err
	}
	if apiErr != nil {
		return 0, apiErr
	}
	return id, nil
}

// pathRef is how the record path named the record, for messages and delete results
func pathRef(c *gin.Context) recordRef {
	if c.Param("record") != byKeySegment {
		return recordRef{ref: c.Param("record")}
	}
	key := map[string]interface{}{}
	for name, values := range c.Request.URL.Query() {
		if m := keyParamPattern.FindStringSubmatch(name); m != nil && len(values) > 0 {
			key[m[1]] = values[0]
		}
	}
	return recordRef{key: key}
}

// keyFromQuery reads the key[Field]=value parameters of records/by-key
func keyFromQuery(query url.Values, k *recordKey) (map[string]interface{}, *Error) {
	key := map[string]interface{}{}
	for name, values := range query {
		m := keyParamPattern.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		if len(values) != 1 {
			return nil, badRequest(fmt.Sprintf("%s is given more than once.", name))
		}
		key[m[1]] = values[0]
	}
	if len(key) == 0 {
		if k.idKey {
			return nil, badRequest("The primary key of this table is the record ID: use records/rec_….")
		}
		return nil, badRequest(fmt.Sprintf("Name the key fields: %s.", k.pathHint()))
	}
	return key, nil
}

// recordRefs finds the records of a bulk request. Records that don't exist give one not_found error naming
// each; the same record twice is a bad request. where names the request part ("records", "ids", "keys").
func (a *API) recordRefs(ctx context.Context, tc *tableContext, refs []recordRef, where string) ([]int64, error) {
	k, err := a.recordKeyOf(ctx, tc)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(refs))
	seen := map[int64]int{}
	var missing []ErrorDetail
	for i, ref := range refs {
		id, apiErr, err := a.findRecord(ctx, k, ref, where)
		if err != nil {
			return nil, err
		}
		if apiErr != nil {
			if apiErr.Code == CodeNotFound {
				index := i
				missing = append(missing, ErrorDetail{Index: &index, ID: ref.label(), Issue: "no record with this ID or key"})
				continue
			}
			return nil, NewError(apiErr.Code, fmt.Sprintf("%s[%d]: %s", where, i, apiErr.Message))
		}
		if j, dup := seen[id]; dup {
			return nil, badRequest(fmt.Sprintf("%s: positions %d and %d are the same record (%s).", where, j, i, ref.label()))
		}
		seen[id] = i
		ids[i] = id
	}
	if len(missing) > 0 {
		return nil, NewError(CodeNotFound, "Some records don't exist. Nothing was changed.", missing...)
	}
	return ids, nil
}

// findRecord finds one record. apiErr is the caller's problem (not found, a record ID where the key is
// needed, a wrong key object); err is a failure. where is "path", or the bulk request part for the hints.
func (a *API) findRecord(ctx context.Context, k *recordKey, ref recordRef, where string) (int64, *Error, error) {
	hint := k.pathHint()
	switch where {
	case "records":
		hint = fmt.Sprintf(`"key": %s instead of "id"`, k.objectHint())
	case "ids":
		hint = fmt.Sprintf(`"keys": [%s] instead of "ids"`, k.objectHint())
	}
	if ref.key != nil {
		if k.idKey {
			return 0, badRequest("The primary key of this table is the record ID: use a record ID (rec_…), not key fields."), nil
		}
		values, apiErr := keyObjectValues(k, ref.key)
		if apiErr != nil {
			return 0, apiErr, nil
		}
		return a.findByValues(ctx, k, values)
	}
	if id, ok := DecodeRecordID(ref.ref); ok {
		if !k.idKey {
			return 0, badRequest(fmt.Sprintf("%q is a record ID, but the primary key of this table is %s. Use %s.", ref.ref, k.title(), hint)), nil
		}
		return id, nil, nil
	}
	switch {
	case ref.ref == "" || len(k.cols) == 0:
		return 0, notFound(fmt.Sprintf("No record with ID %q.", ref.ref)), nil
	case len(k.cols) > 1:
		return 0, badRequest(fmt.Sprintf("The primary key of this table is %s, so name each key field: use %s.", k.title(), hint)), nil
	}
	return a.findByValues(ctx, k, []string{ref.ref})
}

// keyObjectValues reads a key object, {"SKU": "A1", "Region": "EU"} with field names or IDs, as the key
// columns' values in key order
func keyObjectValues(k *recordKey, key map[string]interface{}) ([]string, *Error) {
	values := make([]string, len(k.cols))
	given := make([]bool, len(k.cols))
	refs := make([]string, 0, len(key))
	for ref := range key {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		pos, name, apiErr := k.keyPosition(ref)
		if apiErr != nil {
			return nil, apiErr
		}
		if given[pos] {
			return nil, badRequest(fmt.Sprintf("%s is given more than once.", name))
		}
		given[pos] = true
		switch v := key[ref].(type) {
		case string:
			// The record ID in a key may be given as rec_12 or 12
			if k.cols[pos] == idColumn {
				if id, ok := DecodeRecordID(v); ok {
					v = fmt.Sprint(id)
				}
			}
			values[pos] = v
		case json.Number:
			values[pos] = v.String()
		case float64, int, int64, bool:
			values[pos] = fmt.Sprint(v)
		case nil:
			return nil, badRequest(fmt.Sprintf("%s can't be empty: it is part of the primary key (%s).", name, k.title()))
		default:
			return nil, badRequest(fmt.Sprintf("%s must be a single value.", name))
		}
	}
	var missing []string
	titles := k.titles()
	for i, ok := range given {
		if !ok {
			missing = append(missing, titles[i])
		}
	}
	switch len(missing) {
	case 0:
		return values, nil
	case 1:
		return nil, badRequest(fmt.Sprintf("The primary key is %s; %s is missing.", k.title(), missing[0]))
	}
	return nil, badRequest(fmt.Sprintf("The primary key is %s; %s are missing.", k.title(), strings.Join(missing, " and ")))
}

// keyPosition finds which key column a key object entry names: a field by name or ID, or "id", the record ID,
// when it is part of the key
func (k *recordKey) keyPosition(ref string) (int, string, *Error) {
	if ref == refID {
		for i, col := range k.cols {
			if col == idColumn {
				return i, refID, nil
			}
		}
		return 0, "", badRequest(fmt.Sprintf("id is not part of the primary key (%s).", k.title()))
	}
	f, err := resolveField(k.tc.fields, ref)
	if err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) {
			return 0, "", badRequest(apiErr.Message)
		}
		return 0, "", badRequest(err.Error())
	}
	for i, kf := range k.fields {
		if kf == f {
			return i, f.Name, nil
		}
	}
	return 0, "", badRequest(fmt.Sprintf("%s is not part of the primary key (%s).", f.Name, k.title()))
}

// findByValues finds the record whose key columns hold values
func (a *API) findByValues(ctx context.Context, k *recordKey, values []string) (int64, *Error, error) {
	ids, err := a.idsByKey(ctx, k.tc, k.cols, values)
	if err != nil {
		return 0, nil, err
	}
	if len(ids) == 0 {
		titles := k.titles()
		parts := make([]string, len(values))
		for i := range values {
			parts[i] = fmt.Sprintf("%s %q", titles[i], values[i])
		}
		return 0, notFound(fmt.Sprintf("No record with %s.", strings.Join(parts, " and "))), nil
	}
	return ids[0], nil, nil
}

// primaryKeyColumns returns the table's primary key columns in key order
func (a *API) primaryKeyColumns(ctx context.Context, tc *tableContext) ([]string, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT a.attname::text
		FROM pg_index i
		CROSS JOIN LATERAL unnest(i.indkey) WITH ORDINALITY AS k(attnum, pos)
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = k.attnum
		WHERE i.indrelid = $1::regclass AND i.indisprimary
		ORDER BY k.pos`, a.recordsTable(tc.table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

// idsByKey returns the IDs of the records whose key columns hold values. A value the column can't hold
// ("abc" for a number key) matches nothing.
func (a *API) idsByKey(ctx context.Context, tc *tableContext, cols, values []string) ([]int64, error) {
	conds := make([]string, len(cols))
	args := make([]interface{}, len(cols))
	for i, col := range cols {
		conds[i] = fmt.Sprintf("t.%s = $%d", pq.QuoteIdentifier(col), i+1)
		args[i] = values[i]
	}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(`SELECT t.id FROM %s AS t WHERE %s LIMIT 2`,
		a.recordsTable(tc.table), strings.Join(conds, " AND ")), args...)
	if err != nil {
		var pqErr *pq.Error
		// Class 22 (data exception): the value isn't of the column's type
		if errors.As(err, &pqErr) && pqErr.Code.Class() == "22" {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
