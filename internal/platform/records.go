// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"unicode"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const (
	// maxBulkRecords is the most records one create, update or delete request may hold
	maxBulkRecords = 100
	maxBodyBytes   = 10 << 20
)

// ---- Output ----

type fieldValue struct {
	key   string
	value interface{}
}

// orderedFields marshals as a JSON object in the table's field order
type orderedFields []fieldValue

func (o orderedFields) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, fv := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := json.Marshal(fv.key)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(fv.value)
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(value)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

type recordOut struct {
	ID        string        `json:"id"`
	Fields    orderedFields `json:"fields"`
	CreatedAt interface{}   `json:"created_at"`
	UpdatedAt interface{}   `json:"updated_at"`
}

type deletedOut struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// fieldKeys returns the key of each field in "fields": its name, or its ID with ?keys=id. Fields that share a
// name with another field of the table are keyed by ID, so no value is hidden.
func fieldKeys(fields []*field, byID bool) map[*field]string {
	count := map[string]int{}
	for _, f := range fields {
		count[f.Name]++
	}
	keys := make(map[*field]string, len(fields))
	for _, f := range fields {
		if byID || count[f.Name] > 1 {
			keys[f] = f.PublicID()
		} else {
			keys[f] = f.Name
		}
	}
	return keys
}

func rowID(row map[string]interface{}) int64 {
	if n, ok := row["id"].(json.Number); ok {
		id, _ := n.Int64()
		return id
	}
	return 0
}

// buildRecords turns stored rows into API records with the given fields. A link field becomes a list of
// objects, one per linked record: its id and the values of the lookups that go through the link. Lookups are
// not keys of their own: a selected lookup brings its link field, holding only the selected lookups.
func (a *API) buildRecords(ctx context.Context, tc *tableContext, rows []map[string]interface{}, fields []*field, keysByID bool) ([]recordOut, error) {
	keys := fieldKeys(tc.fields, keysByID)

	// The fields to return, in order, with the lookups each link field carries
	selected := map[*field]bool{}
	for _, f := range fields {
		selected[f] = true
	}
	var returned []*field
	carried := map[*field][]*field{}
	added := map[*field]bool{}
	for _, f := range fields {
		link := f
		if f.Type == typeLookup {
			link = fieldByID(tc.fields, f.LookupLinkColumnID)
			if link == nil || link.Type != typeLinks {
				continue
			}
		}
		if added[link] {
			continue
		}
		added[link] = true
		returned = append(returned, link)
		if link.Type != typeLinks {
			continue
		}
		// A selected link field carries all its lookups; otherwise only the selected ones
		for _, lf := range tc.fields {
			if lf.Type == typeLookup && strings.EqualFold(lf.LookupLinkColumnID, link.ID) && (selected[link] || selected[lf]) {
				carried[link] = append(carried[link], lf)
			}
		}
	}

	linkIDs := map[*field][][]int64{}
	lookups := map[*field][]linkLookup{}
	for _, f := range returned {
		if f.Type != typeLinks {
			continue
		}
		cells := make([]interface{}, len(rows))
		for i, row := range rows {
			cells[i] = row[f.ColumnName]
		}
		ids, err := a.resolveLinks(ctx, tc.ws, f, cells)
		if err != nil {
			return nil, err
		}
		linkIDs[f] = ids
		values, err := a.linkLookups(ctx, tc.ws, f, carried[f], ids, keysByID)
		if err != nil {
			return nil, err
		}
		lookups[f] = values
	}

	out := make([]recordOut, len(rows))
	for i, row := range rows {
		values := make(orderedFields, 0, len(returned))
		for _, f := range returned {
			var v interface{}
			if f.Type == typeLinks {
				list := make([]orderedFields, 0, len(linkIDs[f][i]))
				for _, id := range linkIDs[f][i] {
					obj := orderedFields{{key: refID, value: EncodeRecordID(id)}}
					for _, lk := range lookups[f] {
						obj = append(obj, fieldValue{key: lk.key, value: lk.values[id]})
					}
					list = append(list, obj)
				}
				v = list
			} else {
				v = apiValue(f, row[f.ColumnName])
			}
			values = append(values, fieldValue{key: keys[f], value: v})
		}
		out[i] = recordOut{
			ID:        EncodeRecordID(rowID(row)),
			Fields:    values,
			CreatedAt: apiTime(row["created_time"]),
			UpdatedAt: apiTime(row["last_modified_time"]),
		}
	}
	return out, nil
}

// linkLookup is one lookup in a link field's objects: its key and its value in each linked record
type linkLookup struct {
	key    string
	values map[int64]interface{}
}

// lookupTargets returns the linked table and the looked-up fields of the lookups, in order and once each.
// Lookups whose linked table or field no longer exists, or isn't stored, are left out.
func (a *API) lookupTargets(ctx context.Context, ws workspaceRef, link *field, lookups []*field) (*tableInfo, []*field, error) {
	if len(lookups) == 0 || link.LinkTargetModelID == "" {
		return nil, nil, nil
	}
	target, err := a.tableByID(ctx, ws, link.LinkTargetModelID)
	if err != nil || target == nil {
		return nil, nil, err
	}
	targetFields, err := a.fields(ctx, ws, target)
	if err != nil {
		return nil, nil, err
	}
	var out []*field
	seen := map[*field]bool{}
	for _, lf := range lookups {
		tf := fieldByID(targetFields, lf.LookupTargetColumnID)
		if tf == nil || !tf.physical() || seen[tf] {
			continue
		}
		seen[tf] = true
		out = append(out, tf)
	}
	return target, out, nil
}

// lookupKeys returns the key of each looked-up field in a link object: its name, or its ID with ?keys=id,
// when the name is "id" or when two of the fields share it
func lookupKeys(targets []*field, keysByID bool) map[*field]string {
	keys := fieldKeys(targets, keysByID)
	for f, key := range keys {
		if key == refID {
			keys[f] = f.PublicID()
		}
	}
	return keys
}

// linkLookups returns the lookups a link field carries, with their values in the linked records
func (a *API) linkLookups(ctx context.Context, ws workspaceRef, link *field, lookups []*field, linkIDs [][]int64, keysByID bool) ([]linkLookup, error) {
	target, targets, err := a.lookupTargets(ctx, ws, link, lookups)
	if err != nil || len(targets) == 0 {
		return nil, err
	}
	var ids []int64
	seen := map[int64]bool{}
	for _, list := range linkIDs {
		for _, id := range list {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	keys := lookupKeys(targets, keysByID)
	out := make([]linkLookup, len(targets))
	for i, tf := range targets {
		out[i] = linkLookup{key: keys[tf], values: map[int64]interface{}{}}
		if len(ids) == 0 {
			continue
		}
		values, err := a.lookupValues(ctx, target, tf, ids)
		if err != nil {
			return nil, err
		}
		out[i].values = values
	}
	return out, nil
}

// resolveLinks returns, for each link cell, the IDs of the linked records. Cells hold the linked records'
// primary key objects: {"id": 12} on most tables, the key columns on tables with a custom primary key.
func (a *API) resolveLinks(ctx context.Context, ws workspaceRef, f *field, cells []interface{}) ([][]int64, error) {
	out := make([][]int64, len(cells))
	var target *tableInfo
	if f.LinkTargetModelID != "" {
		t, err := a.tableByID(ctx, ws, f.LinkTargetModelID)
		if err != nil {
			return nil, err
		}
		target = t
	}

	// Key objects other than {"id": n} are looked up in the linked table, once per distinct key
	var lookup []string
	index := map[string]int{}
	for _, cell := range cells {
		list, _ := cell.([]interface{})
		for _, item := range list {
			obj, ok := item.(map[string]interface{})
			if !ok || isIDKey(obj) {
				continue
			}
			key, _ := json.Marshal(obj)
			if _, seen := index[string(key)]; !seen {
				index[string(key)] = len(lookup)
				lookup = append(lookup, string(key))
			}
		}
	}
	found := make([]int64, len(lookup))
	if len(lookup) > 0 && target != nil {
		rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
			`SELECT k.ord, x.id FROM jsonb_array_elements($1::jsonb) WITH ORDINALITY AS k(obj, ord) JOIN %s AS x ON to_jsonb(x) @> k.obj`,
			a.recordsTable(target)), "["+strings.Join(lookup, ",")+"]")
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var ord, id int64
			if err := rows.Scan(&ord, &id); err != nil {
				rows.Close()
				return nil, err
			}
			found[ord-1] = id
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	for i, cell := range cells {
		ids := []int64{}
		list, _ := cell.([]interface{})
		for _, item := range list {
			obj, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if isIDKey(obj) {
				if n, ok := obj["id"].(json.Number); ok {
					if id, err := n.Int64(); err == nil {
						ids = append(ids, id)
					}
				}
				continue
			}
			key, _ := json.Marshal(obj)
			if id := found[index[string(key)]]; id > 0 {
				ids = append(ids, id)
			}
		}
		out[i] = ids
	}
	return out, nil
}

func isIDKey(obj map[string]interface{}) bool {
	_, ok := obj["id"]
	return ok && len(obj) == 1
}

// lookupValues returns the looked-up field's value in each of the linked records
func (a *API) lookupValues(ctx context.Context, target *tableInfo, tf *field, ids []int64) (map[int64]interface{}, error) {
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(`SELECT x.id, (to_jsonb(x) -> $1)::text FROM %s AS x WHERE x.id = ANY($2)`,
		a.recordsTable(target)), tf.ColumnName, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := map[int64]interface{}{}
	for rows.Next() {
		var id int64
		var raw sql.NullString
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		var v interface{}
		if raw.Valid {
			v, _ = decodeJSON([]byte(raw.String))
		}
		values[id] = apiValue(tf, v)
	}
	return values, rows.Err()
}

// ---- Reads ----

func decodeJSON(b []byte) (interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v interface{}
	err := dec.Decode(&v)
	return v, err
}

// queryRows runs a query that returns row_to_json text per row
func (a *API) queryRows(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		v, err := decodeJSON([]byte(raw))
		if err != nil {
			return nil, err
		}
		row, _ := v.(map[string]interface{})
		out = append(out, row)
	}
	return out, rows.Err()
}

func (a *API) linkTargetTable(ctx context.Context, ws workspaceRef) func(*field) string {
	return func(f *field) string {
		if f.LinkTargetModelID == "" {
			return ""
		}
		t, err := a.tableByID(ctx, ws, f.LinkTargetModelID)
		if err != nil || t == nil {
			return ""
		}
		return a.recordsTable(t)
	}
}

// GET /records
func (a *API) listRecords(c *gin.Context) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	opts, err := parseListOptions(c.Request.URL.Query(), tc.fields, a.linkTargetTable(c, tc.ws))
	if err != nil {
		respondError(c, err)
		return
	}
	query, args := listSQL(a.recordsTable(tc.table), opts)
	rows, err := a.queryRows(c, query, args...)
	if err != nil {
		respondError(c, err)
		return
	}
	next := ""
	if len(rows) > opts.limit {
		rows = rows[:opts.limit]
		next = encodeCursor(opts.offset+opts.limit, opts.hash)
	}
	out, err := a.buildRecords(c, tc, rows, opts.fields, opts.keysByID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondList(c, out, next, opts.limit)
}

func keysByID(c *gin.Context) (bool, error) {
	switch c.Query("keys") {
	case "", "name":
		return false, nil
	case "id":
		return true, nil
	}
	return false, badRequest("keys must be id or name.")
}

// recordFromPath reads the {record} path segment
func recordFromPath(c *gin.Context) (int64, error) {
	id, ok := DecodeRecordID(c.Param("record"))
	if !ok {
		return 0, notFound(fmt.Sprintf("No record with ID %q.", c.Param("record")))
	}
	return id, nil
}

func (a *API) readRecord(ctx context.Context, tc *tableContext, id int64) (map[string]interface{}, error) {
	rows, err := a.queryRows(ctx, fmt.Sprintf(`SELECT row_to_json(t)::text FROM %s AS t WHERE t.id = $1`, a.recordsTable(tc.table)), id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, notFound(fmt.Sprintf("No record with ID %q.", EncodeRecordID(id)))
	}
	return rows[0], nil
}

// GET /records/{id}
func (a *API) getRecord(c *gin.Context) {
	byID, err := keysByID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	id, err := recordFromPath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	row, err := a.readRecord(c, tc, id)
	if err != nil {
		respondError(c, err)
		return
	}
	a.respondRecords(c, tc, []map[string]interface{}{row}, true, byID, http.StatusOK)
}

// respondRecords writes one record, or the list when single is false
func (a *API) respondRecords(c *gin.Context, tc *tableContext, rows []map[string]interface{}, single, byID bool, status int) {
	out, err := a.buildRecords(c, tc, rows, tc.fields, byID)
	if err != nil {
		respondError(c, err)
		return
	}
	if single {
		respond(c, status, out[0])
		return
	}
	respond(c, status, out)
}

// ---- Writes ----

// readBody reads the request body as a JSON object, keeping numbers exact
func readBody(c *gin.Context) (map[string]interface{}, error) {
	raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes))
	if err != nil {
		return nil, NewError(CodePayloadTooLarge, "The request body is too large.")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var body map[string]interface{}
	if err := dec.Decode(&body); err != nil || body == nil || dec.More() {
		return nil, badRequest("The request body must be a JSON object.")
	}
	return body, nil
}

// onlyKeys rejects body keys other than the allowed ones, so typos surface
func onlyKeys(obj map[string]interface{}, where string, allowed ...string) error {
	for k := range obj {
		ok := false
		for _, a := range allowed {
			if k == a {
				ok = true
			}
		}
		if !ok {
			return badRequest(fmt.Sprintf("%s: unknown key %q. Allowed: %s.", where, k, strings.Join(allowed, ", ")))
		}
	}
	return nil
}

func objectOf(v interface{}, where string) (map[string]interface{}, error) {
	if v == nil {
		return map[string]interface{}{}, nil
	}
	obj, ok := v.(map[string]interface{})
	if !ok {
		return nil, badRequest(where + " must be a JSON object.")
	}
	return obj, nil
}

// recordsArray reads body["records"]: 1 to 100 objects
func recordsArray(body map[string]interface{}) ([]map[string]interface{}, error) {
	list, ok := body["records"].([]interface{})
	if !ok || len(list) == 0 {
		return nil, badRequest("records must be a non-empty array.")
	}
	if len(list) > maxBulkRecords {
		return nil, NewError(CodePayloadTooLarge, fmt.Sprintf("At most %d records per request; this one has %d.", maxBulkRecords, len(list)))
	}
	out := make([]map[string]interface{}, len(list))
	for i, item := range list {
		obj, err := objectOf(item, fmt.Sprintf("records[%d]", i))
		if err != nil || item == nil {
			return nil, badRequest(fmt.Sprintf("records[%d] must be a JSON object.", i))
		}
		out[i] = obj
	}
	return out, nil
}

type writeInput struct {
	values map[string]interface{} // by column name
	links  map[*field][]int64
}

// convertFields validates a record's "fields" and converts them for storage. index is the record's position
// in a bulk request, or -1. Link fields are only accepted when allowLinks is set (single-record writes).
func convertFields(fields []*field, input map[string]interface{}, index int, allowLinks bool) (writeInput, []ErrorDetail) {
	in := writeInput{values: map[string]interface{}{}, links: map[*field][]int64{}}
	var details []ErrorDetail
	issue := func(key, msg string) {
		d := ErrorDetail{Field: key, Issue: msg}
		if index >= 0 {
			i := index
			d.Index = &i
		}
		details = append(details, d)
	}

	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	seen := map[*field]bool{}
	for _, key := range keys {
		f, err := resolveField(fields, key)
		if err != nil {
			var apiErr *Error
			if errors.As(err, &apiErr) && apiErr.Code == CodeAmbiguousName {
				issue(key, "several fields have this name; use the field ID")
			} else {
				issue(key, "unknown field")
			}
			continue
		}
		if seen[f] {
			issue(key, "is given twice (by name and by ID)")
			continue
		}
		seen[f] = true

		if f.Type == typeLinks {
			if !allowLinks {
				issue(key, "link fields can only be set when writing one record")
				continue
			}
			ids, msg := linkInput(f, input[key])
			if msg != "" {
				issue(key, msg)
				continue
			}
			in.links[f] = ids
			continue
		}
		value, msg := dbValue(f, input[key])
		if msg != "" {
			issue(key, msg)
			continue
		}
		in.values[f.ColumnName] = value
	}
	return in, details
}

// linkInput reads a link field's value: an array of record IDs, or of objects with an id as the API returns
// them (other keys are ignored); null clears the field
func linkInput(f *field, v interface{}) ([]int64, string) {
	const invalid = "must be an array of record IDs like rec_12, or of objects with an id"
	if v == nil {
		return []int64{}, ""
	}
	list, ok := v.([]interface{})
	if !ok {
		return nil, invalid
	}
	ids := []int64{}
	seen := map[int64]bool{}
	for _, item := range list {
		if obj, isObject := item.(map[string]interface{}); isObject {
			item = obj[refID]
		}
		s, _ := item.(string)
		id, ok := DecodeRecordID(s)
		if !ok {
			return nil, invalid
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if !f.LinkMulti && len(ids) > 1 {
		return nil, "links to one record only"
	}
	return ids, ""
}

// checkLinkTargets reports link values that point to records that don't exist
func (a *API) checkLinkTargets(ctx context.Context, tc *tableContext, links map[*field][]int64) ([]ErrorDetail, error) {
	var details []ErrorDetail
	for _, f := range tc.fields {
		ids, ok := links[f]
		if !ok || len(ids) == 0 {
			continue
		}
		target, err := a.tableByID(ctx, tc.ws, f.LinkTargetModelID)
		if err != nil {
			return nil, err
		}
		if target == nil {
			details = append(details, ErrorDetail{Field: f.Name, Issue: "the linked table no longer exists"})
			continue
		}
		rows, err := a.db.QueryContext(ctx, fmt.Sprintf(`SELECT id FROM %s WHERE id = ANY($1)`, a.recordsTable(target)), pq.Array(ids))
		if err != nil {
			return nil, err
		}
		exists := map[int64]bool{}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			exists[id] = true
		}
		rows.Close()
		for _, id := range ids {
			if !exists[id] {
				details = append(details, ErrorDetail{Field: f.Name, ID: EncodeRecordID(id), Issue: "no record with this ID in the linked table"})
			}
		}
	}
	return details, nil
}

// applyLinks makes each given link field hold exactly the given records, linking and unlinking one pair at
// a time so both sides stay in sync. It runs after the record is saved, outside its transaction.
func (a *API) applyLinks(ctx context.Context, tc *tableContext, row map[string]interface{}, links map[*field][]int64) error {
	id := rowID(row)
	for _, f := range tc.fields {
		want, ok := links[f]
		if !ok {
			continue
		}
		current, err := a.resolveLinks(ctx, tc.ws, f, []interface{}{row[f.ColumnName]})
		if err != nil {
			return err
		}
		have := map[int64]bool{}
		for _, t := range current[0] {
			have[t] = true
		}
		keep := map[int64]bool{}
		for _, t := range want {
			keep[t] = true
		}
		change := func(target int64, action string) error {
			_, err := a.records.UpdateRawDataForLinks(ctx, a.schema, dto.UpdateRowDataLinksRequest{
				ModelID: tc.table.ID, ColumnId: f.ID, SourceRowId: int(id), TargetRowId: int(target), Action: action,
			})
			if err != nil {
				return NewError(CodeValidationFailed,
					"The record was saved, but its link fields could not be updated.",
					ErrorDetail{Field: f.Name, ID: EncodeRecordID(target), Issue: userMessage(err)})
			}
			return nil
		}
		for _, t := range current[0] {
			if !keep[t] {
				if err := change(t, "unlink"); err != nil {
					return err
				}
			}
		}
		for _, t := range want {
			if !have[t] {
				if err := change(t, "link"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func modelOf(t *tableInfo) (tenant.Model, error) {
	id, err := uuid.Parse(t.ID)
	if err != nil {
		return tenant.Model{}, fmt.Errorf("table %s has an invalid ID: %w", t.ID, err)
	}
	baseID, _ := uuid.Parse(t.BaseID)
	return tenant.Model{ID: id, BaseID: baseID, Title: t.Title, Alias: t.Alias}, nil
}

// writeError turns a write service error into the API error; recordIDs gives the ID of each record in the
// request, for not-found details
func writeError(err error, recordIDs func(int) string) error {
	var recErr *app_errors.RecordWriteError
	if !errors.As(err, &recErr) {
		return err
	}
	index := recErr.Index
	if errors.Is(recErr.Err, app_errors.ErrRecordNotFound) {
		return NewError(CodeNotFound, "Some records don't exist. Nothing was changed.",
			ErrorDetail{Index: &index, ID: recordIDs(index), Issue: "no record with this ID"})
	}
	if errors.Is(recErr.Err, app_errors.DatabaseError) {
		return recErr.Err
	}
	return NewError(CodeValidationFailed, "Some records can't be saved. Nothing was changed.",
		ErrorDetail{Index: &index, Issue: userMessage(recErr.Err)})
}

// userMessage is the message the app shows for a write error
func userMessage(err error) string {
	var detailed *app_errors.DetailedError
	if errors.As(err, &detailed) && detailed.Message != "" {
		return detailed.Message
	}
	var apiErr *app_errors.APIError
	if errors.As(err, &apiErr) && apiErr.Message != "" {
		return apiErr.Message
	}
	msg := err.Error()
	if msg == "" {
		return msg
	}
	r := []rune(msg)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func validationFailed(details []ErrorDetail) error {
	return NewError(CodeValidationFailed, "Some fields have invalid values. Nothing was saved.", details...)
}

// POST /records: {"fields": {...}} creates one record; {"records": [{"fields": {...}}]} creates many
func (a *API) createRecords(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	_, many := body["records"]
	var inputs []map[string]interface{}
	if many {
		if err = onlyKeys(body, "body", "records"); err == nil {
			inputs, err = recordsArray(body)
		}
	} else {
		if err = onlyKeys(body, "body", "fields"); err == nil {
			var obj map[string]interface{}
			obj, err = objectOf(body["fields"], "fields")
			inputs = []map[string]interface{}{{"fields": obj}}
		}
	}
	if err == nil && many {
		for i, in := range inputs {
			if err = onlyKeys(in, fmt.Sprintf("records[%d]", i), "fields"); err != nil {
				break
			}
		}
	}
	if err != nil {
		respondError(c, err)
		return
	}

	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	writes := make([]writeInput, len(inputs))
	values := make([]map[string]interface{}, len(inputs))
	var details []ErrorDetail
	for i, in := range inputs {
		fields, err := objectOf(in["fields"], fmt.Sprintf("records[%d].fields", i))
		if err != nil {
			respondError(c, err)
			return
		}
		index := i
		if !many {
			index = -1
		}
		w, d := convertFields(tc.fields, fields, index, !many)
		writes[i], values[i] = w, w.values
		details = append(details, d...)
	}
	if !many && len(details) == 0 {
		d, err := a.checkLinkTargets(c, tc, writes[0].links)
		if err != nil {
			respondError(c, err)
			return
		}
		details = append(details, d...)
	}
	if len(details) > 0 {
		respondError(c, validationFailed(details))
		return
	}

	model, err := modelOf(tc.table)
	if err != nil {
		respondError(c, err)
		return
	}
	rows, err := a.records.CreateRecordsForAPI(c, a.schema, model, values)
	if err != nil {
		respondError(c, writeError(err, func(int) string { return "" }))
		return
	}
	if !many && len(writes[0].links) > 0 {
		if err := a.applyLinks(c, tc, rows[0], writes[0].links); err != nil {
			respondError(c, err)
			return
		}
		if rows[0], err = a.readRecord(c, tc, rowID(rows[0])); err != nil {
			respondError(c, err)
			return
		}
	}
	a.respondRecords(c, tc, rows, !many, c.Query("keys") == "id", http.StatusCreated)
}

// PATCH /records/{id}: {"fields": {...}} changes only the fields sent
func (a *API) updateRecord(c *gin.Context) {
	body, err := readBody(c)
	if err == nil {
		err = onlyKeys(body, "body", "fields")
	}
	var fields map[string]interface{}
	if err == nil {
		fields, err = objectOf(body["fields"], "fields")
	}
	if err != nil {
		respondError(c, err)
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	id, err := recordFromPath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	w, details := convertFields(tc.fields, fields, -1, true)
	if len(details) == 0 {
		d, err := a.checkLinkTargets(c, tc, w.links)
		if err != nil {
			respondError(c, err)
			return
		}
		details = d
	}
	if len(details) > 0 {
		respondError(c, validationFailed(details))
		return
	}

	model, err := modelOf(tc.table)
	if err != nil {
		respondError(c, err)
		return
	}
	rows, err := a.records.UpdateRecordsForAPI(c, a.schema, model, []dto.APIRecordUpdate{{RowID: id, Values: w.values}})
	if err != nil {
		var recErr *app_errors.RecordWriteError
		if errors.As(err, &recErr) && errors.Is(recErr.Err, app_errors.ErrRecordNotFound) {
			respondError(c, notFound(fmt.Sprintf("No record with ID %q.", EncodeRecordID(id))))
			return
		}
		respondError(c, writeError(err, func(int) string { return EncodeRecordID(id) }))
		return
	}
	if len(w.links) > 0 {
		if err := a.applyLinks(c, tc, rows[0], w.links); err != nil {
			respondError(c, err)
			return
		}
		if rows[0], err = a.readRecord(c, tc, id); err != nil {
			respondError(c, err)
			return
		}
	}
	a.respondRecords(c, tc, rows, true, c.Query("keys") == "id", http.StatusOK)
}

// recordIDs reads a list of record IDs from a bulk request; the same ID twice is a bad request
func recordIDs(raw []string, where string) ([]int64, error) {
	ids := make([]int64, len(raw))
	seen := map[int64]int{}
	for i, s := range raw {
		id, ok := DecodeRecordID(s)
		if !ok {
			return nil, badRequest(fmt.Sprintf("%s[%d]: %q is not a record ID like rec_12.", where, i, s))
		}
		if j, dup := seen[id]; dup {
			return nil, badRequest(fmt.Sprintf("%s: %s appears twice (positions %d and %d).", where, s, j, i))
		}
		seen[id] = i
		ids[i] = id
	}
	return ids, nil
}

// PATCH /records: {"records": [{"id": "rec_1", "fields": {...}}]}
func (a *API) updateRecords(c *gin.Context) {
	body, err := readBody(c)
	if err == nil {
		err = onlyKeys(body, "body", "records")
	}
	var inputs []map[string]interface{}
	if err == nil {
		inputs, err = recordsArray(body)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	rawIDs := make([]string, len(inputs))
	for i, in := range inputs {
		if err := onlyKeys(in, fmt.Sprintf("records[%d]", i), "id", "fields"); err != nil {
			respondError(c, err)
			return
		}
		rawIDs[i], _ = in["id"].(string)
	}
	ids, err := recordIDs(rawIDs, "records.id")
	if err != nil {
		respondError(c, err)
		return
	}

	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	updates := make([]dto.APIRecordUpdate, len(inputs))
	var details []ErrorDetail
	for i, in := range inputs {
		fields, err := objectOf(in["fields"], fmt.Sprintf("records[%d].fields", i))
		if err != nil {
			respondError(c, err)
			return
		}
		w, d := convertFields(tc.fields, fields, i, false)
		updates[i] = dto.APIRecordUpdate{RowID: ids[i], Values: w.values}
		details = append(details, d...)
	}
	if len(details) > 0 {
		respondError(c, validationFailed(details))
		return
	}

	model, err := modelOf(tc.table)
	if err != nil {
		respondError(c, err)
		return
	}
	rows, err := a.records.UpdateRecordsForAPI(c, a.schema, model, updates)
	if err != nil {
		respondError(c, writeError(err, func(i int) string { return rawIDs[i] }))
		return
	}
	a.respondRecords(c, tc, rows, false, c.Query("keys") == "id", http.StatusOK)
}

// deleteIDs deletes the records, or none of them when some don't exist
func (a *API) deleteIDs(c *gin.Context, ids []int64) ([]deletedOut, error) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		return nil, err
	}
	model, err := modelOf(tc.table)
	if err != nil {
		return nil, err
	}
	missing, err := a.records.DeleteRecordsForAPI(c, a.schema, model, ids)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		details := make([]ErrorDetail, len(missing))
		for i, id := range missing {
			details[i] = ErrorDetail{ID: EncodeRecordID(id), Issue: "no record with this ID"}
		}
		return nil, NewError(CodeNotFound, "Some records don't exist. Nothing was deleted.", details...)
	}
	out := make([]deletedOut, len(ids))
	for i, id := range ids {
		out[i] = deletedOut{ID: EncodeRecordID(id), Deleted: true}
	}
	return out, nil
}

// DELETE /records/{id}
func (a *API) deleteRecord(c *gin.Context) {
	id, err := recordFromPath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	out, err := a.deleteIDs(c, []int64{id})
	if err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) && apiErr.Code == CodeNotFound && len(apiErr.Details) > 0 {
			err = notFound(fmt.Sprintf("No record with ID %q.", EncodeRecordID(id)))
		}
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, out[0])
}

// POST /records/delete: {"ids": ["rec_1", "rec_2"]}
func (a *API) deleteRecords(c *gin.Context) {
	body, err := readBody(c)
	if err == nil {
		err = onlyKeys(body, "body", "ids")
	}
	if err != nil {
		respondError(c, err)
		return
	}
	list, ok := body["ids"].([]interface{})
	if !ok || len(list) == 0 {
		respondError(c, badRequest("ids must be a non-empty array of record IDs."))
		return
	}
	if len(list) > maxBulkRecords {
		respondError(c, NewError(CodePayloadTooLarge, fmt.Sprintf("At most %d records per request; this one has %d.", maxBulkRecords, len(list))))
		return
	}
	raw := make([]string, len(list))
	for i, item := range list {
		raw[i], _ = item.(string)
	}
	ids, err := recordIDs(raw, "ids")
	if err != nil {
		respondError(c, err)
		return
	}
	out, err := a.deleteIDs(c, ids)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, out)
}
