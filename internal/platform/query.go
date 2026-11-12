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
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	pgsql "github.com/aptlogica/go-postgres-rest/pkg/database/postgres"
	"github.com/aptlogica/go-postgres-rest/pkg/models"
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

// filterSQL turns filters into SQL. Only its SQL building is used, so it needs no database.
var filterSQL = pgsql.NewPostgresDbServiceInstance(nil)

// filterTerm is one filter[...] parameter or the where tree, as a go-postgres-rest filter group
type filterTerm struct {
	key  string // canonical form, for the cursor hash
	node models.ComplexFilter
}

type sortTerm struct {
	key  string
	expr string
	desc bool
}

// listOptions are the query parameters of GET /records (and the body of POST /records/query)
type listOptions struct {
	limit    int
	offset   int
	fields   []*field // fields to return; all fields when select is not given or empty
	filters  []filterTerm
	sorts    []sortTerm
	keysByID bool
	hash     string // filter and sort, so a cursor can't be reused with different ones
}

// linkResolver returns the JSON arrays a link cell contains when it links to record id of the field's linked
// table: [{"id": n}], and on a table with a custom primary key also the key columns' object. ok is false when
// the linked table no longer exists.
type linkResolver func(f *field, id int64) (arrays []string, ok bool, err error)

// emptyParam tells whether every value of a parameter is empty: "", "{}" or "null"
func emptyParam(values []string) bool {
	for _, v := range values {
		switch strings.TrimSpace(v) {
		case "", "{}", "null":
		default:
			return false
		}
	}
	return true
}

var filterParamPattern = regexp.MustCompile(`^filter\[([^\[\]]+)\](?:\[([a-z_]+)\])?$`)

// parseListOptions reads the list query parameters. Problems are bad_request errors.
func parseListOptions(query url.Values, fields []*field, links linkResolver) (*listOptions, error) {
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

	// An empty select (select= or "select": []) returns all fields, as when select is not given
	if refs := splitList(strings.Join(query["select"], ",")); len(refs) > 0 {
		selected := []*field{}
		seen := map[*field]bool{}
		for _, ref := range refs {
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
		// A bare filter= or filter={} filters nothing
		if k == "filter" && emptyParam(query[k]) {
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
			term, err := buildFilter(fields, m[1], op, value, links)
			if err != nil {
				return nil, err
			}
			opts.filters = append(opts.filters, term)
		}
	}

	// The where tree; combined with the filter[...] parameters using AND
	// Left out, empty, null, {} or a group without conditions: no filter
	if term, ok, err := parseWhereParam(query.Get("where"), fields, links); err != nil {
		return nil, err
	} else if ok {
		opts.filters = append(opts.filters, term)
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

var unsortableTypes = map[string]bool{typeMultiSelect: true, typeLinks: true, typeLookup: true, typeFormula: true, typeAttachment: true, typeJSON: true}

func buildSort(fields []*field, ref string, desc bool) (sortTerm, error) {
	if col, ok := systemSortColumns[ref]; ok {
		return sortTerm{key: ref, expr: "t." + pq.QuoteIdentifier(col), desc: desc}, nil
	}
	f, err := resolveField(fields, ref)
	if err != nil {
		return sortTerm{}, asBadRequest(err, "sort")
	}
	if unsortableTypes[f.Type] {
		if computedTypes[f.Type] {
			return sortTerm{}, badRequest("sort: " + computedFieldMessage(f))
		}
		return sortTerm{}, badRequest(fmt.Sprintf("sort: records can't be sorted by %q (%s field).", f.Name, f.APIType()))
	}
	return sortTerm{key: f.ID, expr: columnExpr(f), desc: desc}, nil
}

// columnExpr is the SQL value of a field for sorting. Duration is stored as text minutes, so it is sorted as a
// number.
func columnExpr(f *field) string {
	col := "t." + pq.QuoteIdentifier(f.ColumnName)
	if f.Type == typeDuration {
		return fmt.Sprintf(`(CASE WHEN %[1]s ~ '^\s*-?[0-9]+(\.[0-9]+)?\s*$' THEN %[1]s::numeric END)`, col)
	}
	return col
}

// ---- Filter ----

// Every filter is a go-postgres-rest filter group; the library writes the SQL. Field types whose values need
// more than one comparison (empty text, unchecked checkboxes, empty lists) are small any-of groups.

// computedTypes are worked out when a record is read, so there is no stored value to filter or sort by
var computedTypes = map[string]bool{typeFormula: true, typeLookup: true}

var textTypes = map[string]bool{
	typeText: true, typeLongText: true, typeEmail: true, typeURL: true, typePhone: true, typeSelect: true,
	typeUUID: true, typeUser: true, typeTime: true, typeCreatedBy: true, typeLastModifiedBy: true,
}

var numberTypes = map[string]bool{
	typeNumber: true, typeDecimal: true, typeCurrency: true, typePercent: true, typeRating: true, typeYear: true, typeDuration: true,
}

var timeTypes = map[string]bool{typeDateTime: true, typeCreatedTime: true, typeLastModifiedTime: true}

func computedFieldMessage(f *field) string {
	return fmt.Sprintf("%q is a %s field. Formula and lookup fields are calculated when read, so they can't be used to filter or sort.",
		f.Name, f.APIType())
}

// leaf is a group holding one condition
func leaf(col, op string, value interface{}, cast string) models.ComplexFilter {
	return models.ComplexFilter{Filters: []models.QueryFilter{{Column: col, Operator: op, Value: value, Cast: cast}}}
}

// anyOf matches when at least one of the groups matches
func anyOf(nodes ...models.ComplexFilter) models.ComplexFilter {
	return models.ComplexFilter{Logic: "OR", Groups: nodes}
}

// negate matches when the group doesn't; a group that is NULL (an empty value) counts as not matching
func negate(node models.ComplexFilter) models.ComplexFilter {
	return models.ComplexFilter{Logic: "AND", Groups: []models.ComplexFilter{node}, Not: true}
}

// filterCondition is one condition, from a filter[...] parameter or from the where tree
type filterCondition struct {
	label   string // where a problem is reported: filter[Price][gt] or where.conditions[0]
	ref     string
	op      string
	value   string
	list    []string // the values of in, when given as a list
	hasList bool
}

// buildFilter builds the condition of one filter[...] parameter
func buildFilter(fields []*field, ref, op, value string, links linkResolver) (filterTerm, error) {
	return buildCondition(fields, filterCondition{label: fmt.Sprintf("filter[%s][%s]", ref, op), ref: ref, op: op, value: value}, links)
}

// buildCondition builds one condition. Every column, operator and value is checked here, so the library only
// ever gets conditions it can write.
func buildCondition(fields []*field, c filterCondition, links linkResolver) (filterTerm, error) {
	ref, op, value := c.ref, c.op, c.value
	listKey := ""
	if c.hasList {
		listKey = "[" + strings.Join(c.list, "\x1f") + "]"
	}
	key := fmt.Sprintf("%s:%s:%s%s", ref, op, value, listKey)
	bad := func(msg string) (filterTerm, error) {
		return filterTerm{}, badRequest(fmt.Sprintf("%s: %s", c.label, msg))
	}
	if c.hasList && op != "in" {
		return bad("a list of values works only with the in operator.")
	}
	if op == "is_null" && value != "true" && value != "false" {
		return bad("is_null takes true or false.")
	}
	values := func() []string {
		if c.hasList {
			return c.list
		}
		return splitList(value)
	}

	// System values
	if col, ok := systemSortColumns[ref]; ok {
		if !slices.Contains(scalarOperators, op) {
			return bad(fmt.Sprintf("%s takes %s.", ref, strings.Join(scalarOperators, ", ")))
		}
		parse := func(s string) (interface{}, bool) {
			if ref == refID {
				id, ok := DecodeRecordID(s)
				return id, ok
			}
			t, err := time.Parse(time.RFC3339Nano, s)
			return t.UTC(), err == nil
		}
		return scalarFilter(key, pq.QuoteIdentifier(col), "", op, value, values, parse, bad)
	}

	f, err := resolveField(fields, ref)
	if err != nil {
		return filterTerm{}, fieldLookupError(err, c.label, fields, ref)
	}
	key = fmt.Sprintf("%s:%s:%s%s", f.ID, op, value, listKey)
	ops := filterOperators(f)
	if len(ops) == 0 {
		if computedTypes[f.Type] {
			return bad(computedFieldMessage(f))
		}
		return bad(fmt.Sprintf("%q is a %s field and can't be used to filter.", f.Name, f.APIType()))
	}
	if !slices.Contains(ops, op) {
		return bad(fmt.Sprintf("%q is a %s field. You can use: %s.", f.Name, f.APIType(), strings.Join(ops, ", ")))
	}
	col := pq.QuoteIdentifier(f.ColumnName)
	if pgsql.ValidateColumnName(col) != nil {
		return bad(fmt.Sprintf("%q can't be used to filter: its column name isn't supported.", f.Name))
	}

	switch {
	case f.Type == typeMultiSelect:
		if op == "contains" {
			return filterTerm{key: key, node: leaf(col, "any", value, "")}, nil
		}
		return nullTerm(key, value, anyOf(leaf(col, "is_null", nil, ""), leaf(col, "eq", "{}", ""))), nil

	case f.Type == typeLinks:
		if op == "is_null" {
			return nullTerm(key, value, anyOf(leaf(col, "is_null", nil, ""), leaf(col, "eq", "[]", ""))), nil
		}
		id, ok := DecodeRecordID(value)
		if !ok {
			return bad("the value must be a record ID like rec_12.")
		}
		// Link cells hold the linked records' primary key objects
		arrays, ok, err := links(f, id)
		if err != nil {
			return filterTerm{}, err
		}
		if !ok {
			return bad("the linked table of this field no longer exists.")
		}
		nodes := make([]models.ComplexFilter, len(arrays))
		for i, a := range arrays {
			nodes[i] = leaf(col, "jsonb_contains", a, "")
		}
		return filterTerm{key: key, node: anyOf(nodes...)}, nil

	case f.Type == typeCheckbox:
		if op == "is_null" {
			return nullTerm(key, value, leaf(col, "is_null", nil, "")), nil
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			return bad("the value must be true or false.")
		}
		if op == "eq" && !b || op == "neq" && b {
			// Unchecked includes never set
			return filterTerm{key: key, node: anyOf(leaf(col, "eq", false, ""), leaf(col, "is_null", nil, ""))}, nil
		}
		return filterTerm{key: key, node: leaf(col, "eq", true, "")}, nil

	case textTypes[f.Type]:
		if op == "contains" {
			pattern := "%" + likeEscaper.Replace(value) + "%"
			return filterTerm{key: key, node: leaf(col, "ilike", pattern, pgsql.CastText)}, nil
		}
		if op == "is_null" {
			return nullTerm(key, value, anyOf(leaf(col, "is_null", nil, ""), leaf(col, "eq", "", pgsql.CastText))), nil
		}
		cast := ""
		if f.Type == typeUUID {
			cast = pgsql.CastText
		}
		return scalarFilter(key, col, cast, op, value, values, func(s string) (interface{}, bool) { return s, true }, bad)

	case numberTypes[f.Type]:
		cast := ""
		if f.Type == typeDuration {
			// Stored as text minutes
			cast = pgsql.CastSafeNumeric
		}
		return scalarFilter(key, col, cast, op, value, values, func(s string) (interface{}, bool) {
			n, err := strconv.ParseFloat(s, 64)
			return n, err == nil
		}, bad)

	case f.Type == typeDate:
		return scalarFilter(key, col, "", op, value, values, func(s string) (interface{}, bool) {
			_, err := time.Parse("2006-01-02", s)
			return s, err == nil
		}, bad)

	case timeTypes[f.Type]:
		return scalarFilter(key, col, "", op, value, values, func(s string) (interface{}, bool) {
			t, err := time.Parse(time.RFC3339Nano, s)
			return t.UTC(), err == nil
		}, bad)
	}
	return bad(fmt.Sprintf("%q is a %s field and can't be used to filter.", f.Name, f.APIType()))
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// nullTerm is isNull for is_null=true and its opposite for is_null=false
func nullTerm(key, value string, isNull models.ComplexFilter) filterTerm {
	if value == "true" {
		return filterTerm{key: key, node: isNull}
	}
	return filterTerm{key: key, node: negate(isNull)}
}

// scalarFilter handles eq, neq, gt, gte, lt, lte, in and is_null on a single value column
func scalarFilter(key, col, cast, op, value string, values func() []string, parse func(string) (interface{}, bool),
	bad func(string) (filterTerm, error)) (filterTerm, error) {
	if op == "is_null" {
		return nullTerm(key, value, leaf(col, "is_null", nil, cast)), nil
	}
	if op == "in" {
		parts := values()
		if len(parts) == 0 {
			return bad("in takes a list of values (comma-separated in filter[...]).")
		}
		parsed := make([]interface{}, len(parts))
		for i, p := range parts {
			v, ok := parse(p)
			if !ok {
				return bad(fmt.Sprintf("%q is not a valid value for this field.", p))
			}
			parsed[i] = v
		}
		return filterTerm{key: key, node: leaf(col, "in", parsed, cast)}, nil
	}
	if !slices.Contains([]string{"eq", "neq", "gt", "gte", "lt", "lte"}, op) {
		return bad(fmt.Sprintf("unknown operator. Use %s.", strings.Join(scalarOperators, ", ")))
	}
	v, ok := parse(value)
	if !ok {
		return bad(fmt.Sprintf("%q is not a valid value for this field.", value))
	}
	return filterTerm{key: key, node: leaf(col, op, v, cast)}, nil
}

// fieldLookupError reports a field that doesn't exist as bad_request, suggesting the closest field name
func fieldLookupError(err error, label string, fields []*field, ref string) error {
	apiErr, ok := err.(*Error)
	if !ok || apiErr.Code != CodeNotFound {
		return err
	}
	msg := fmt.Sprintf("%s: %s", label, apiErr.Message)
	if name := closestFieldName(fields, ref); name != "" {
		msg += fmt.Sprintf(" Did you mean %q?", name)
	}
	return badRequest(msg)
}

// closestFieldName returns the field name nearest to ref (case-insensitive, at most 2 edits), or ""
func closestFieldName(fields []*field, ref string) string {
	want := strings.ToLower(ref)
	best, bestDist := "", 3
	for _, f := range fields {
		d := editDistance(want, strings.ToLower(f.Name))
		if d < bestDist && d < len([]rune(want)) {
			best, bestDist = f.Name, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// ---- SQL ----

// errFilterSQL means a checked filter still produced no SQL; it should never happen
var errFilterSQL = errors.New("platform: a filter condition produced no SQL")

// checkFilter makes sure the library writes every condition of the group, so none can be dropped silently
func checkFilter(node models.ComplexFilter) error {
	if len(node.Filters) == 0 && len(node.Groups) == 0 {
		return errFilterSQL
	}
	for _, f := range node.Filters {
		if sql, _, _ := filterSQL.BuildFilterCondition(f, 1); sql == "" {
			return errFilterSQL
		}
	}
	for _, g := range node.Groups {
		if err := checkFilter(g); err != nil {
			return err
		}
	}
	return nil
}

// listSQL builds the page query: limit+1 rows, so the caller knows whether there is another page
func listSQL(table string, opts *listOptions) (string, []interface{}, error) {
	var b strings.Builder
	var args []interface{}
	next := 1
	fmt.Fprintf(&b, "SELECT row_to_json(t)::text FROM %s AS t", table)
	if len(opts.filters) > 0 {
		all := models.ComplexFilter{Logic: "AND", Groups: make([]models.ComplexFilter, len(opts.filters))}
		for i, f := range opts.filters {
			all.Groups[i] = f.node
		}
		if err := checkFilter(all); err != nil {
			return "", nil, err
		}
		where, whereArgs, n := filterSQL.BuildComplexFilter(all, next)
		if where == "" {
			return "", nil, errFilterSQL
		}
		b.WriteString(" WHERE " + where)
		args, next = whereArgs, n
	}
	order := make([]string, len(opts.sorts))
	for i, s := range opts.sorts {
		order[i] = s.expr
		if s.desc {
			order[i] += " DESC"
		}
	}
	b.WriteString(" ORDER BY " + strings.Join(order, ", "))
	fmt.Fprintf(&b, " LIMIT $%d OFFSET $%d", next, next+1)
	args = append(args, opts.limit+1, opts.offset)
	return b.String(), args, nil
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
