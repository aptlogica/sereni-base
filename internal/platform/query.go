// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

// Names that mean the system values in select, sort and filter; a user field with one of these names must
// be referenced by its ID in query parameters
const (
	refID        = "id"
	refCreatedAt = "created_at"
	refUpdatedAt = "updated_at"
)

var systemSortColumns = map[string]string{refID: "id", refCreatedAt: "created_time", refUpdatedAt: "last_modified_time"}

// sqlArgs numbers the parameters of a statement as they are added
type sqlArgs struct{ values []interface{} }

func (a *sqlArgs) add(v interface{}) string {
	a.values = append(a.values, v)
	return fmt.Sprintf("$%d", len(a.values))
}

type filterTerm struct {
	key   string // canonical form, for the cursor hash
	build func(args *sqlArgs) string
}

type sortTerm struct {
	key  string
	expr string
	desc bool
}

// listOptions are the query parameters of GET /records
type listOptions struct {
	limit    int
	offset   int
	fields   []*field // fields to return; all fields when select is not given
	filters  []filterTerm
	sorts    []sortTerm
	keysByID bool
	hash     string // filter and sort, so a cursor can't be reused with different ones
}

var filterParamPattern = regexp.MustCompile(`^filter\[([^\[\]]+)\](?:\[([a-z_]+)\])?$`)

// parseListOptions reads the list query parameters. Problems are bad_request errors.
func parseListOptions(query url.Values, fields []*field, linkTarget func(*field) string) (*listOptions, error) {
	opts := &listOptions{limit: defaultLimit, fields: fields}

	if raw := query.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxLimit {
			return nil, badRequest(fmt.Sprintf("limit must be a whole number from 1 to %d.", maxLimit))
		}
		opts.limit = n
	}

	switch query.Get("keys") {
	case "", "name":
	case "id":
		opts.keysByID = true
	default:
		return nil, badRequest("keys must be id or name.")
	}

	if raw, ok := query["select"]; ok {
		selected := []*field{}
		seen := map[*field]bool{}
		for _, ref := range splitList(strings.Join(raw, ",")) {
			if ref == refID || ref == refCreatedAt || ref == refUpdatedAt {
				continue // always returned
			}
			f, err := resolveField(fields, ref)
			if err != nil {
				return nil, asBadRequest(err, "select")
			}
			if !seen[f] {
				seen[f] = true
				selected = append(selected, f)
			}
		}
		opts.fields = selected
	}

	// Filters, in a fixed order so the cursor hash doesn't depend on parameter order
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !strings.HasPrefix(k, "filter") {
			continue
		}
		m := filterParamPattern.FindStringSubmatch(k)
		if m == nil {
			return nil, badRequest(fmt.Sprintf("%q is not a valid filter. Use filter[Field]=value or filter[Field][operator]=value.", k))
		}
		op := m[2]
		if op == "" {
			op = "eq"
		}
		for _, value := range query[k] {
			term, err := buildFilter(fields, m[1], op, value, linkTarget)
			if err != nil {
				return nil, err
			}
			opts.filters = append(opts.filters, term)
		}
	}

	if raw := query.Get("sort"); raw != "" {
		for _, ref := range splitList(raw) {
			desc := strings.HasPrefix(ref, "-")
			ref = strings.TrimPrefix(ref, "-")
			term, err := buildSort(fields, ref, desc)
			if err != nil {
				return nil, err
			}
			opts.sorts = append(opts.sorts, term)
		}
	}
	// id is always the last tie-breaker, so pages never overlap or skip records
	if !hasIDSort(opts.sorts) {
		opts.sorts = append(opts.sorts, sortTerm{key: refID, expr: "t.id"})
	}

	opts.hash = optionsHash(opts)
	if raw := query.Get("cursor"); raw != "" {
		offset, err := decodeCursor(raw, opts.hash)
		if err != nil {
			return nil, err
		}
		opts.offset = offset
	}
	return opts, nil
}

func hasIDSort(sorts []sortTerm) bool {
	for _, s := range sorts {
		if s.key == refID {
			return true
		}
	}
	return false
}

// asBadRequest turns a field lookup error in a query parameter into bad_request, keeping ambiguous_name
func asBadRequest(err error, param string) error {
	if apiErr, ok := err.(*Error); ok && apiErr.Code == CodeNotFound {
		return badRequest(fmt.Sprintf("%s: %s", param, apiErr.Message))
	}
	return err
}

// ---- Sort ----

var unsortableTypes = map[string]bool{typeMultiSelect: true, typeLinks: true, typeLookup: true, typeAttachment: true, typeJSON: true}

func buildSort(fields []*field, ref string, desc bool) (sortTerm, error) {
	if col, ok := systemSortColumns[ref]; ok {
		return sortTerm{key: ref, expr: "t." + pq.QuoteIdentifier(col), desc: desc}, nil
	}
	f, err := resolveField(fields, ref)
	if err != nil {
		return sortTerm{}, asBadRequest(err, "sort")
	}
	if unsortableTypes[f.Type] {
		return sortTerm{}, badRequest(fmt.Sprintf("sort: records can't be sorted by %q (%s field).", f.Name, f.APIType()))
	}
	return sortTerm{key: f.ID, expr: columnExpr(f), desc: desc}, nil
}

// columnExpr is the SQL value of a field for comparing and sorting. Duration is stored as text minutes, so it
// is compared as a number.
func columnExpr(f *field) string {
	col := "t." + pq.QuoteIdentifier(f.ColumnName)
	if f.Type == typeDuration {
		return fmt.Sprintf(`(CASE WHEN %[1]s ~ '^\s*-?[0-9]+(\.[0-9]+)?\s*$' THEN %[1]s::numeric END)`, col)
	}
	return col
}

// ---- Filter ----

var comparisonOps = map[string]string{"eq": "=", "neq": "<>", "gt": ">", "gte": ">=", "lt": "<", "lte": "<="}

var textTypes = map[string]bool{
	typeText: true, typeLongText: true, typeEmail: true, typeURL: true, typePhone: true, typeSelect: true,
	typeFormula: true, typeUUID: true, typeUser: true, typeTime: true, typeCreatedBy: true, typeLastModifiedBy: true,
}

var numberTypes = map[string]bool{
	typeNumber: true, typeDecimal: true, typeCurrency: true, typePercent: true, typeRating: true, typeYear: true, typeDuration: true,
}

var timeTypes = map[string]bool{typeDateTime: true, typeCreatedTime: true, typeLastModifiedTime: true}

// buildFilter builds one filter condition. linkTarget returns the quoted records table of a link field's
// linked table.
func buildFilter(fields []*field, ref, op, value string, linkTarget func(*field) string) (filterTerm, error) {
	key := fmt.Sprintf("%s:%s:%s", ref, op, value)
	bad := func(msg string) (filterTerm, error) {
		return filterTerm{}, badRequest(fmt.Sprintf("filter[%s][%s]: %s", ref, op, msg))
	}
	if op == "is_null" && value != "true" && value != "false" {
		return bad("is_null takes true or false.")
	}

	// System values
	if col, ok := systemSortColumns[ref]; ok {
		expr := "t." + pq.QuoteIdentifier(col)
		parse := func(s string) (interface{}, bool) {
			if ref == refID {
				id, ok := DecodeRecordID(s)
				return id, ok
			}
			t, err := time.Parse(time.RFC3339Nano, s)
			return t.UTC(), err == nil
		}
		return scalarFilter(key, expr, op, value, parse, bad)
	}

	f, err := resolveField(fields, ref)
	if err != nil {
		return filterTerm{}, asBadRequest(err, "filter")
	}
	key = fmt.Sprintf("%s:%s:%s", f.ID, op, value)
	col := "t." + pq.QuoteIdentifier(f.ColumnName)

	switch {
	case f.Type == typeMultiSelect:
		switch op {
		case "contains":
			return filterTerm{key: key, build: func(a *sqlArgs) string { return fmt.Sprintf("%s = ANY(%s)", a.add(value), col) }}, nil
		case "is_null":
			return nullFilter(key, value, fmt.Sprintf("(%[1]s IS NULL OR cardinality(%[1]s) = 0)", col)), nil
		}
		return bad("multi-select fields support contains and is_null.")

	case f.Type == typeLinks:
		switch op {
		case "contains":
			id, ok := DecodeRecordID(value)
			if !ok {
				return bad("the value must be a record ID like rec_12.")
			}
			target := linkTarget(f)
			if target == "" {
				return bad("the linked table of this field no longer exists.")
			}
			// Link cells hold the linked records' primary key objects; match them against the linked table
			return filterTerm{key: key, build: func(a *sqlArgs) string {
				return fmt.Sprintf(`EXISTS (SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(%[1]s) = 'array' THEN %[1]s ELSE '[]'::jsonb END) AS k(obj)
					JOIN %[2]s AS x ON to_jsonb(x) @> k.obj WHERE x.id = %[3]s)`, col, target, a.add(id))
			}}, nil
		case "is_null":
			return nullFilter(key, value, fmt.Sprintf("(%[1]s IS NULL OR %[1]s = '[]'::jsonb)", col)), nil
		}
		return bad("link fields support contains and is_null.")

	case f.Type == typeJSON || f.Type == typeAttachment || f.Type == typeLookup:
		return bad(fmt.Sprintf("%s fields can't be filtered.", f.APIType()))

	case f.Type == typeCheckbox:
		if op == "is_null" {
			return nullFilter(key, value, col+" IS NULL"), nil
		}
		if op != "eq" && op != "neq" {
			return bad("checkbox fields support eq, neq and is_null.")
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			return bad("the value must be true or false.")
		}
		if op == "eq" && !b || op == "neq" && b {
			// Unchecked includes never set
			return filterTerm{key: key, build: func(*sqlArgs) string { return fmt.Sprintf("COALESCE(%s, false) = false", col) }}, nil
		}
		return filterTerm{key: key, build: func(*sqlArgs) string { return col + " = true" }}, nil

	case textTypes[f.Type]:
		if op == "contains" {
			pattern := "%" + likeEscaper.Replace(value) + "%"
			return filterTerm{key: key, build: func(a *sqlArgs) string { return fmt.Sprintf("%s::text ILIKE %s", col, a.add(pattern)) }}, nil
		}
		if op == "is_null" {
			return nullFilter(key, value, fmt.Sprintf("(%[1]s IS NULL OR %[1]s::text = '')", col)), nil
		}
		expr := col
		if f.Type == typeUUID {
			expr = col + "::text"
		}
		return scalarFilter(key, expr, op, value, func(s string) (interface{}, bool) { return s, true }, bad)

	case numberTypes[f.Type]:
		return scalarFilter(key, columnExpr(f), op, value, func(s string) (interface{}, bool) {
			n, err := strconv.ParseFloat(s, 64)
			return n, err == nil
		}, bad)

	case f.Type == typeDate:
		return scalarFilter(key, col, op, value, func(s string) (interface{}, bool) {
			_, err := time.Parse("2006-01-02", s)
			return s, err == nil
		}, bad)

	case timeTypes[f.Type]:
		return scalarFilter(key, col, op, value, func(s string) (interface{}, bool) {
			t, err := time.Parse(time.RFC3339Nano, s)
			return t.UTC(), err == nil
		}, bad)
	}
	return bad(fmt.Sprintf("%s fields can't be filtered.", f.APIType()))
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func nullFilter(key, value, isNullSQL string) filterTerm {
	if value == "true" {
		return filterTerm{key: key, build: func(*sqlArgs) string { return isNullSQL }}
	}
	return filterTerm{key: key, build: func(*sqlArgs) string { return "NOT " + isNullSQL }}
}

// scalarFilter handles eq, neq, gt, gte, lt, lte, in and is_null on a single value column
func scalarFilter(key, expr, op, value string, parse func(string) (interface{}, bool), bad func(string) (filterTerm, error)) (filterTerm, error) {
	if op == "is_null" {
		return nullFilter(key, value, expr+" IS NULL"), nil
	}
	if op == "in" {
		parts := splitList(value)
		if len(parts) == 0 {
			return bad("in takes a comma-separated list of values.")
		}
		parsed := make([]interface{}, len(parts))
		for i, p := range parts {
			v, ok := parse(p)
			if !ok {
				return bad(fmt.Sprintf("%q is not a valid value for this field.", p))
			}
			parsed[i] = v
		}
		return filterTerm{key: key, build: func(a *sqlArgs) string {
			placeholders := make([]string, len(parsed))
			for i, v := range parsed {
				placeholders[i] = a.add(v)
			}
			return fmt.Sprintf("%s IN (%s)", expr, strings.Join(placeholders, ", "))
		}}, nil
	}
	sqlOp, ok := comparisonOps[op]
	if !ok {
		return bad("unknown operator. Use eq, neq, gt, gte, lt, lte, in, contains or is_null.")
	}
	v, ok := parse(value)
	if !ok {
		return bad(fmt.Sprintf("%q is not a valid value for this field.", value))
	}
	return filterTerm{key: key, build: func(a *sqlArgs) string { return fmt.Sprintf("%s %s %s", expr, sqlOp, a.add(v)) }}, nil
}

// ---- SQL ----

// listSQL builds the page query: limit+1 rows, so the caller knows whether there is another page
func listSQL(table string, opts *listOptions) (string, []interface{}) {
	args := &sqlArgs{}
	var b strings.Builder
	fmt.Fprintf(&b, "SELECT row_to_json(t)::text FROM %s AS t", table)
	if len(opts.filters) > 0 {
		conds := make([]string, len(opts.filters))
		for i, f := range opts.filters {
			conds[i] = f.build(args)
		}
		b.WriteString(" WHERE " + strings.Join(conds, " AND "))
	}
	order := make([]string, len(opts.sorts))
	for i, s := range opts.sorts {
		order[i] = s.expr
		if s.desc {
			order[i] += " DESC"
		}
	}
	b.WriteString(" ORDER BY " + strings.Join(order, ", "))
	fmt.Fprintf(&b, " LIMIT %s OFFSET %s", args.add(opts.limit+1), args.add(opts.offset))
	return b.String(), args.values
}

// ---- Cursor ----

const cursorVersion = 1

type cursor struct {
	V int    `json:"v"`
	O int    `json:"o"` // offset of the next page; the opaque format allows switching to keyset paging later
	H string `json:"h"`
}

func optionsHash(opts *listOptions) string {
	parts := make([]string, 0, len(opts.filters)+len(opts.sorts))
	for _, f := range opts.filters {
		parts = append(parts, "f:"+f.key)
	}
	for _, s := range opts.sorts {
		dir := "asc"
		if s.desc {
			dir = "desc"
		}
		parts = append(parts, "s:"+s.key+":"+dir)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:8])
}

func encodeCursor(offset int, hash string) string {
	b, _ := json.Marshal(cursor{V: cursorVersion, O: offset, H: hash})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(raw, hash string) (int, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	var c cursor
	if err != nil || json.Unmarshal(b, &c) != nil || c.V != cursorVersion || c.O < 0 {
		return 0, badRequest("cursor is not valid. Use the next_cursor of the previous page.")
	}
	if c.H != hash {
		return 0, badRequest("cursor was made for a different filter or sort. Send the same filter and sort as the first page, or start again without a cursor.")
	}
	return c.O, nil
}
