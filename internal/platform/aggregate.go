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
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // timezone names work the same on every OS the server runs on

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// POST /records/aggregate answers count, count_distinct, sum, avg, min and max over the records a filter
// selects, optionally grouped by up to 3 fields. Dates and date-times can be grouped by day, week, month or
// year. The filter is the one the list endpoint uses, so both select the same records.
//
//	{"where": {...}, "search": "acme", "group_by": ["Status", {"field": "Ordered at", "by": "month"}],
//	 "metrics": [{"op": "count"}, {"op": "sum", "field": "Price", "as": "total"}],
//	 "timezone": "Asia/Kolkata", "sort": ["-total"], "limit": 100, "keys": "name"}
const (
	maxMetrics    = 10
	maxGroupBy    = 3
	maxGroups     = 1000 // groups one answer may hold; more are left out, with has_more
	defaultGroups = 100
	maxMetricName = 100
)

var aggregateBodyKeys = []string{"where", "search", "group_by", "metrics", "timezone", "sort", "limit", "keys"}

var aggregateOps = []string{"count", "count_distinct", "sum", "avg", "min", "max"}

// dateBuckets are the periods dates and date-times can be grouped by; weeks start on Monday (ISO weeks)
var dateBuckets = []string{"day", "week", "month", "year"}

// groupTextTypes are the text fields that can be grouped: values people pick or type short
var groupTextTypes = map[string]bool{typeText: true, typeEmail: true, typeURL: true, typePhone: true, typeSelect: true}

// Kinds of values, which decide what a field can be grouped and aggregated by
const (
	kindText        = "text"
	kindCheckbox    = "checkbox"
	kindNumber      = "number"
	kindDate        = "date"
	kindDateTime    = "datetime"
	kindLink        = "link"        // grouped by linked record
	kindMultiSelect = "multiselect" // grouped by option
)

// aggColumn is a field, or created_at / updated_at, used in group_by or a metric
type aggColumn struct {
	f      *field // nil for created_at and updated_at
	ref    string // created_at or updated_at when f is nil
	name   string // for messages
	col    string // SQL value: t."column", or the numeric cast of a duration
	kind   string // "" when it can't be grouped or aggregated
	via    *aggJoin
	lookup bool // a lookup: its value is the linked record's field, empty when there is no linked record
}

// aggJoin is the list a grouped value comes from: a link field's linked records (x, one row per linked record) or
// a multi-select field's options (m0.v). A record is in each group its list puts it in.
type aggJoin struct {
	list   *field     // the link or multi-select field
	target *aggTarget // the linked table; nil for multi-select
}

// aggTarget is a link field's linked table
type aggTarget struct {
	table   string   // quoted records table
	fields  []*field // its fields, for lookups
	keyCols []string // primary key columns of a custom key; nil for the default key (id)
}

// aggTargets finds a link field's linked table; nil when it no longer exists
type aggTargets func(link *field) (*aggTarget, error)

type groupTerm struct {
	c  *aggColumn
	by string // day, week, month or year; "" for the value itself
}

type metricTerm struct {
	op   string
	c    *aggColumn // nil for count
	name string     // the key in the answer
}

type aggSort struct {
	alias string // g0… or m0…
	desc  bool
}

type aggOptions struct {
	groups   []groupTerm
	metrics  []metricTerm
	sorts    []aggSort
	timezone string
	limit    int
	keysByID bool
}

// aggregateRecords answers POST (and QUERY) /records/aggregate
func (a *API) aggregateRecords(c *gin.Context) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	body, err := readOptionalBody(c)
	if err == nil {
		err = rejectURLOptions(c, "/records/aggregate", aggregateBodyKeys)
	}
	if err == nil {
		err = onlyKeys(body, "body", aggregateBodyKeys...)
	}
	if err != nil {
		respondError(c, err)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), queryTimeout)
	defer cancel()
	fail := func(err error) { respondError(c, timeoutError(ctx, err)) }

	opts, err := parseAggregate(body, tc.fields, a.aggTargets(ctx, tc.ws))
	if err != nil {
		fail(err)
		return
	}
	filterParams, err := aggregateFilterParams(body)
	if err != nil {
		fail(err)
		return
	}
	filters, err := parseFilters(filterParams, tc.fields, a.linkResolver(ctx, tc.ws))
	if err != nil {
		fail(err)
		return
	}
	query, args, err := aggregateSQL(a.recordsTable(tc.table), opts, filters)
	if err != nil {
		fail(err)
		return
	}
	rows, err := a.queryRows(ctx, query, args...)
	if err != nil {
		fail(timezoneError(err, opts.timezone))
		return
	}
	hasMore := len(opts.groups) > 0 && len(rows) > opts.limit
	if hasMore {
		rows = rows[:opts.limit]
	}
	respondGroups(c, opts.answer(rows, tc.fields), hasMore, opts.limit)
}

// aggTargets finds link fields' linked tables, once each per request
func (a *API) aggTargets(ctx context.Context, ws workspaceRef) aggTargets {
	found := map[*field]*aggTarget{}
	return func(link *field) (*aggTarget, error) {
		if t, ok := found[link]; ok {
			return t, nil
		}
		tc, err := a.linkedTable(ctx, ws, link)
		if err != nil || tc == nil {
			return nil, err
		}
		key, err := a.recordKeyOf(ctx, tc)
		if err != nil {
			return nil, err
		}
		t := &aggTarget{table: a.recordsTable(tc.table), fields: tc.fields}
		if !key.idKey {
			t.keyCols = key.cols
		}
		found[link] = t
		return t, nil
	}
}

// aggregateFilterParams reads where and search from the body as the list endpoint reads them from the URL
func aggregateFilterParams(body map[string]interface{}) (url.Values, error) {
	params := url.Values{}
	if v, ok := body["where"]; ok && v != nil {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, badRequest("where must be a JSON object.")
		}
		params.Set("where", string(raw))
	}
	if v, ok := body["search"]; ok && v != nil {
		s, ok := v.(string)
		if !ok {
			return nil, badRequest("search must be text.")
		}
		params.Set("search", s)
	}
	return params, nil
}

// ---- Parsing ----

func parseAggregate(body map[string]interface{}, fields []*field, targets aggTargets) (*aggOptions, error) {
	opts := &aggOptions{limit: defaultGroups, timezone: "UTC"}

	switch body["keys"] {
	case nil, "name":
	case "id":
		opts.keysByID = true
	default:
		return nil, badRequest("keys must be id or name.")
	}

	if v, ok := body["limit"]; ok && v != nil {
		n, err := strconv.Atoi(fmt.Sprint(v))
		if _, isNumber := v.(json.Number); !isNumber || err != nil || n < 1 || n > maxGroups {
			return nil, badRequest(fmt.Sprintf("limit must be a whole number from 1 to %d.", maxGroups))
		}
		opts.limit = n
	}

	if v, ok := body["timezone"]; ok && v != nil {
		tz, ok := v.(string)
		if !ok || !validTimezone(tz) {
			return nil, badRequest(fmt.Sprintf("timezone: %v is not a time zone. Use a name like \"Asia/Kolkata\", \"Europe/Berlin\" or \"UTC\".", jsonText(v)))
		}
		opts.timezone = tz
	}

	groups, err := parseGroupBy(body["group_by"], fields, targets)
	if err != nil {
		return nil, err
	}
	opts.groups = groups

	metrics, err := parseMetrics(body["metrics"], fields, targets, opts.keysByID)
	if err != nil {
		return nil, err
	}
	opts.metrics = metrics

	if v, ok := body["sort"]; ok && v != nil {
		list, err := stringOrList(v, "sort")
		if err != nil {
			return nil, err
		}
		if opts.sorts, err = parseAggregateSort(splitList(list), opts, fields, targets); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

// validTimezone tells whether name is a time zone both Go and Postgres know by that name
func validTimezone(name string) bool {
	if name == "" || name == "Local" || strings.ContainsAny(name, "'\\") {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

// timezoneError turns Postgres refusing a time zone into bad_request. timezone is checked against Go's list
// first; the two lists are nearly the same, but a name only Go knows would otherwise be a 500.
func timezoneError(err error, timezone string) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "22023" && strings.Contains(pqErr.Message, "time zone") {
		return badRequest(fmt.Sprintf("timezone: %q is not a time zone the database knows. Use a name like \"Asia/Kolkata\", \"Europe/Berlin\" or \"UTC\".", timezone))
	}
	return err
}

// jsonText shows a body value in an error message
func jsonText(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

// aggregateColumn finds a field, created_at or updated_at, and what it can be used for
func aggregateColumn(fields []*field, ref, label string, targets aggTargets) (*aggColumn, error) {
	if ref == refCreatedAt || ref == refUpdatedAt {
		return &aggColumn{ref: ref, name: ref, col: "t." + pq.QuoteIdentifier(systemSortColumns[ref]), kind: kindDateTime}, nil
	}
	if ref == refID {
		return nil, badRequest(fmt.Sprintf("%s: records can't be grouped or aggregated by id. To count records use {\"op\": \"count\"}.", label))
	}
	f, err := resolveField(fields, ref)
	if err != nil {
		return nil, fieldLookupError(err, label, fields, ref)
	}
	switch f.Type {
	case typeMultiSelect:
		return &aggColumn{f: f, name: f.Name, col: "m0.v", kind: kindMultiSelect, via: &aggJoin{list: f}}, nil
	case typeLinks:
		target, err := linkTarget(f, label, targets)
		if err != nil {
			return nil, err
		}
		return &aggColumn{f: f, name: f.Name, col: "x.id", kind: kindLink, via: &aggJoin{list: f, target: target}}, nil
	case typeLookup:
		return lookupColumn(fields, f, label, targets)
	}
	if computedTypes[f.Type] {
		return nil, badRequest(fmt.Sprintf("%s: %q is a %s field. Its value is calculated when read, so it can't be grouped or aggregated.",
			label, f.Name, f.APIType()))
	}
	return &aggColumn{f: f, name: f.Name, col: columnExpr(f), kind: aggregateKind(f)}, nil
}

// linkTarget is a link field's linked table, for grouping by the link or a lookup through it
func linkTarget(link *field, label string, targets aggTargets) (*aggTarget, error) {
	var target *aggTarget
	if targets != nil {
		var err error
		if target, err = targets(link); err != nil {
			return nil, err
		}
	}
	if target == nil {
		return nil, badRequest(fmt.Sprintf("%s: the linked table of %q no longer exists.", label, link.Name))
	}
	return target, nil
}

// lookupColumn is a lookup as a group: the looked-up field of each linked record, grouped like that field
func lookupColumn(fields []*field, f *field, label string, targets aggTargets) (*aggColumn, error) {
	link := fieldByID(fields, f.LookupLinkColumnID)
	if link == nil || link.Type != typeLinks {
		return nil, badRequest(fmt.Sprintf("%s: the link field of the lookup %q no longer exists.", label, f.Name))
	}
	target, err := linkTarget(link, label, targets)
	if err != nil {
		return nil, err
	}
	tf := fieldByID(target.fields, f.LookupTargetColumnID)
	if tf == nil {
		return nil, badRequest(fmt.Sprintf("%s: the field the lookup %q looks up no longer exists.", label, f.Name))
	}
	c := &aggColumn{f: f, name: f.Name, col: columnExprAs(tf, "x"), kind: aggregateKind(tf), lookup: true,
		via: &aggJoin{list: link, target: target}}
	if c.kind == "" {
		return nil, badRequest(fmt.Sprintf("%s: %q looks up a %s field, which can't be grouped. Lookups of text, select, checkbox, number, date and date-time fields can.",
			label, f.Name, tf.APIType()))
	}
	return c, nil
}

// aggregateKind is the kind of a field's values for aggregations; "" when it can't be grouped or aggregated
func aggregateKind(f *field) string {
	switch {
	case f.Type == typeLinks:
		return kindLink
	case f.Type == typeMultiSelect:
		return kindMultiSelect
	case groupTextTypes[f.Type]:
		return kindText
	case f.Type == typeCheckbox:
		return kindCheckbox
	case numberTypes[f.Type]:
		return kindNumber
	case f.Type == typeDate:
		return kindDate
	case timeTypes[f.Type]:
		return kindDateTime
	}
	return ""
}

// aggregateUses says what a kind of field can be used for in an aggregate, for the API docs
func aggregateUses(kind string) string {
	switch kind {
	case kindText, kindCheckbox:
		return "group_by, count_distinct"
	case kindNumber:
		return "group_by, count_distinct, sum, avg, min, max"
	case kindDate:
		return "group_by (also by day, week, month, year), count_distinct, min, max"
	case kindDateTime:
		return "group_by by day, week, month or year, min, max"
	case kindLink, kindMultiSelect:
		return "group_by (a record counts in each group it belongs to)"
	}
	return ""
}

// parseGroupBy reads group_by: a list of field names or IDs, or {"field": …, "by": "month"} objects
func parseGroupBy(v interface{}, fields []*field, targets aggTargets) ([]groupTerm, error) {
	var items []interface{}
	switch v := v.(type) {
	case nil:
		return nil, nil
	case []interface{}:
		items = v
	default:
		items = []interface{}{v}
	}
	if len(items) > maxGroupBy {
		return nil, badRequest(fmt.Sprintf("group_by: at most %d fields.", maxGroupBy))
	}
	var out []groupTerm
	seen := map[string]bool{}
	var list *field // the one link or multi-select field the groups may unpack
	for i, item := range items {
		label := fmt.Sprintf("group_by[%d]", i)
		ref, by := "", ""
		switch item := item.(type) {
		case string:
			ref = item
		case map[string]interface{}:
			if err := onlyKeys(item, label, "field", "by"); err != nil {
				return nil, err
			}
			ref, _ = item["field"].(string)
			if raw, ok := item["by"]; ok {
				s, ok := raw.(string)
				if !ok || !containsString(dateBuckets, s) {
					return nil, badRequest(fmt.Sprintf("%s.by must be %s.", label, strings.Join(dateBuckets, ", ")))
				}
				by = s
			}
		}
		if strings.TrimSpace(ref) == "" {
			return nil, badRequest(fmt.Sprintf(`%s must be a field name or ID, or {"field": "Ordered at", "by": "month"}.`, label))
		}
		c, err := aggregateColumn(fields, ref, label, targets)
		if err != nil {
			return nil, err
		}
		if c.via != nil {
			if list != nil && list != c.via.list {
				return nil, badRequest(fmt.Sprintf("%s: group_by can use one link or multi-select field per request, and it already has %q; a lookup counts as its link field.",
					label, list.Name))
			}
			list = c.via.list
		}
		switch {
		case c.kind == kindDateTime && by == "":
			return nil, badRequest(fmt.Sprintf(`%s: %q is a date-time field. Group it by day, week, month or year, e.g. {"field": %q, "by": "month"}.`,
				label, c.name, ref))
		case by != "" && c.kind != kindDate && c.kind != kindDateTime:
			return nil, badRequest(fmt.Sprintf("%s: by works only with date and date-time fields; %q is a %s.", label, c.name, c.describe()))
		case c.kind == "":
			return nil, badRequest(fmt.Sprintf("%s: %q is a %s and can't be grouped. Group by text, select, multi-select, link, lookup, checkbox, number, date or date-time fields.",
				label, c.name, c.describe()))
		}
		// A field once: twice (even by different periods) would give two groups with the same key
		if seen[c.identity()] {
			return nil, badRequest(fmt.Sprintf("%s: %q is already in group_by. Group a field once; for another period, send a second request.", label, c.name))
		}
		seen[c.identity()] = true
		out = append(out, groupTerm{c: c, by: by})
	}
	return out, nil
}

// parseMetrics reads metrics: a list of {"op": …, "field": …, "as": …}
func parseMetrics(v interface{}, fields []*field, targets aggTargets, keysByID bool) ([]metricTerm, error) {
	items, ok := v.([]interface{})
	if !ok || len(items) == 0 {
		return nil, badRequest(`metrics is required: a list like [{"op": "count"}, {"op": "sum", "field": "Price"}].`)
	}
	if len(items) > maxMetrics {
		return nil, badRequest(fmt.Sprintf("metrics: at most %d.", maxMetrics))
	}
	var out []metricTerm
	names := map[string]bool{}
	for i, item := range items {
		label := fmt.Sprintf("metrics[%d]", i)
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, badRequest(fmt.Sprintf(`%s must be an object like {"op": "sum", "field": "Price"}.`, label))
		}
		if err := onlyKeys(obj, label, "op", "field", "as"); err != nil {
			return nil, err
		}
		op, _ := obj["op"].(string)
		if !containsString(aggregateOps, op) {
			return nil, badRequest(fmt.Sprintf("%s.op must be one of %s.", label, strings.Join(aggregateOps, ", ")))
		}
		m := metricTerm{op: op, name: "count"}
		ref, hasField := obj["field"].(string)
		switch {
		case op == "count" && obj["field"] != nil:
			return nil, badRequest(fmt.Sprintf("%s: count counts records and takes no field. To count different values use count_distinct.", label))
		case op != "count" && (!hasField || strings.TrimSpace(ref) == ""):
			return nil, badRequest(fmt.Sprintf("%s: %s needs a field.", label, op))
		case op != "count":
			c, err := aggregateColumn(fields, ref, label+".field", targets)
			if err != nil {
				return nil, err
			}
			if c.via != nil {
				return nil, badRequest(fmt.Sprintf("%s: %q is a %s field; it can be used in group_by, not in metrics.", label, c.name, c.f.APIType()))
			}
			if !metricAllowed(op, c.kind) {
				return nil, badRequest(fmt.Sprintf("%s: %s doesn't work on %q, a %s. %s", label, op, c.name, c.describe(), metricHelp(op)))
			}
			m.c = c
			// keys=id clients get names that survive a rename: avg_fld_…; others get avg_Price
			m.name = op + "_" + c.name
			if keysByID && c.f != nil {
				m.name = op + "_" + c.f.PublicID()
			}
		}
		if raw, ok := obj["as"]; ok {
			s, ok := raw.(string)
			if !ok || strings.TrimSpace(s) == "" || len([]rune(s)) > maxMetricName {
				return nil, badRequest(fmt.Sprintf("%s.as must be text of 1 to %d characters.", label, maxMetricName))
			}
			m.name = s
		}
		if names[m.name] {
			return nil, badRequest(fmt.Sprintf(`%s: two metrics are named %q. Give one a different name with "as".`, label, m.name))
		}
		names[m.name] = true
		out = append(out, m)
	}
	return out, nil
}

func metricAllowed(op, kind string) bool {
	switch op {
	case "sum", "avg":
		return kind == kindNumber
	case "min", "max":
		return kind == kindNumber || kind == kindDate || kind == kindDateTime
	case "count_distinct":
		return kind == kindText || kind == kindCheckbox || kind == kindNumber || kind == kindDate
	}
	return false
}

func metricHelp(op string) string {
	switch op {
	case "sum", "avg":
		return op + " takes a number field (number, decimal, currency, percent, rating, year, duration)."
	case "min", "max":
		return op + " takes a number, date or date-time field."
	}
	return "count_distinct takes a text, select, checkbox, number or date field."
}

// parseAggregateSort reads sort: metric names or group_by fields, - for descending
func parseAggregateSort(refs []string, opts *aggOptions, fields []*field, targets aggTargets) ([]aggSort, error) {
	if len(refs) > maxSortTerms {
		return nil, badRequest(fmt.Sprintf("sort: at most %d fields.", maxSortTerms))
	}
	var out []aggSort
	used := map[string]bool{}
	for _, raw := range refs {
		desc := strings.HasPrefix(raw, "-")
		ref := strings.TrimPrefix(raw, "-")
		alias := ""
		for i, m := range opts.metrics {
			if m.name == ref {
				alias = fmt.Sprintf("m%d", i)
				break
			}
		}
		if alias == "" {
			if c, err := aggregateColumn(fields, ref, "sort", targets); err == nil {
				for i, g := range opts.groups {
					if g.c.identity() == c.identity() {
						alias = fmt.Sprintf("g%d", i)
						break
					}
				}
			}
		}
		if alias == "" {
			return nil, badRequest(fmt.Sprintf("sort: %q is not a metric name or a group_by field.", ref))
		}
		if used[alias] {
			return nil, badRequest(fmt.Sprintf("sort: %q is listed twice.", ref))
		}
		used[alias] = true
		out = append(out, aggSort{alias: alias, desc: desc})
	}
	return out, nil
}

// identity tells two columns apart: the field's ID, or created_at / updated_at
func (c *aggColumn) identity() string {
	if c.f != nil {
		return c.f.ID
	}
	return c.ref
}

// describe is the column's type for messages: "long_text field"
func (c *aggColumn) describe() string {
	if c.f != nil {
		return c.f.APIType() + " field"
	}
	return "date-time"
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// ---- SQL ----

// groupExpr is the SQL value a group is made of. Empty text counts as empty and an unchecked checkbox as
// false, as in filters. Date-times are stored in UTC; tz decides which day a moment belongs to.
func (g groupTerm) expr(tz string) string {
	c := g.c
	switch c.kind {
	case kindLink:
		return c.col // the linked record's id; NULL without linked records or when it was deleted
	case kindMultiSelect:
		return fmt.Sprintf("NULLIF(%s, '')", c.col)
	case kindText:
		return fmt.Sprintf("NULLIF(%s::text, '')", c.col)
	case kindCheckbox:
		if c.lookup {
			// Without a linked record there is no value, not false
			return fmt.Sprintf("CASE WHEN x.id IS NOT NULL THEN COALESCE(%s, false) END", c.col)
		}
		return fmt.Sprintf("COALESCE(%s, false)", c.col)
	case kindDate:
		if g.by != "" {
			return fmt.Sprintf("date_trunc('%s', %s)::date", g.by, c.col)
		}
	case kindDateTime:
		return fmt.Sprintf("date_trunc('%s', (%s AT TIME ZONE 'UTC') AT TIME ZONE %s)::date", g.by, c.col, tz)
	}
	return c.col
}

func (m metricTerm) expr() string {
	switch m.op {
	case "count":
		return "COUNT(*)"
	case "count_distinct":
		value := m.c.col
		if m.c.kind == kindText {
			value = fmt.Sprintf("NULLIF(%s::text, '')", m.c.col)
		}
		return fmt.Sprintf("COUNT(DISTINCT %s)", value)
	case "sum":
		return fmt.Sprintf("COALESCE(SUM(%s), 0)", m.c.col)
	}
	return fmt.Sprintf("%s(%s)", strings.ToUpper(m.op), m.c.col)
}

// aggregateSQL builds the query: one row per group (limit+1, to know there are more), or one row in all
// when nothing is grouped. Columns are named g0…, m0…: field names and metric names never go into the SQL.
func aggregateSQL(table string, opts *aggOptions, filters []filterTerm) (string, []interface{}, error) {
	where, args, next, err := whereClause(filters, 1)
	if err != nil {
		return "", nil, err
	}
	tz := ""
	for _, g := range opts.groups {
		if g.c.kind == kindDateTime {
			tz = fmt.Sprintf("$%d", next)
			args = append(args, opts.timezone)
			next++
			break
		}
	}
	// The records to aggregate: the table, or, when a group comes from a list (links or options), one row per
	// record and group it is in
	from := table + " AS t" + where
	var via *aggJoin
	for _, g := range opts.groups {
		if g.c.via != nil {
			via = g.c.via
			break
		}
	}
	listed := map[int]bool{}
	if via != nil {
		var unpacked []string
		for i, g := range opts.groups {
			if g.c.via != nil {
				listed[i] = true
				unpacked = append(unpacked, g.expr(tz))
			}
		}
		// The filter's columns aren't prefixed with t, so it runs on the table alone, before the join. DISTINCT ON
		// keeps one row per record and group: a record linked to two customers in India counts once for India.
		inner := fmt.Sprintf("(SELECT * FROM %s AS t%s) AS t", table, where)
		picked := make([]string, len(unpacked))
		for i, u := range unpacked {
			picked[i] = fmt.Sprintf(`%s AS "__g%d"`, u, i)
		}
		from = fmt.Sprintf("(SELECT DISTINCT ON (t.id, %s) t.*, %s FROM %s %s) AS t",
			strings.Join(unpacked, ", "), strings.Join(picked, ", "), inner, via.joinSQL())
	}

	cols := make([]string, 0, len(opts.groups)+len(opts.metrics))
	n := 0
	for i, g := range opts.groups {
		expr := g.expr(tz)
		if listed[i] {
			expr = fmt.Sprintf(`t."__g%d"`, n)
			n++
		}
		cols = append(cols, fmt.Sprintf("%s AS g%d", expr, i))
	}
	for i, m := range opts.metrics {
		cols = append(cols, fmt.Sprintf("%s AS m%d", m.expr(), i))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "SELECT %s FROM %s", strings.Join(cols, ", "), from)
	if len(opts.groups) > 0 {
		positions := make([]string, len(opts.groups))
		for i := range opts.groups {
			positions[i] = strconv.Itoa(i + 1)
		}
		order := []string{}
		used := map[string]bool{}
		for _, s := range opts.sorts {
			dir := "ASC"
			if s.desc {
				dir = "DESC"
			}
			order = append(order, fmt.Sprintf("%s %s NULLS LAST", s.alias, dir))
			used[s.alias] = true
		}
		// Then the groups in order, so the answer is the same every time
		for i := range opts.groups {
			if alias := fmt.Sprintf("g%d", i); !used[alias] {
				order = append(order, alias+" ASC NULLS LAST")
			}
		}
		fmt.Fprintf(&b, " GROUP BY %s ORDER BY %s LIMIT $%d", strings.Join(positions, ", "), strings.Join(order, ", "), next)
		args = append(args, opts.limit+1)
	}
	return fmt.Sprintf("SELECT row_to_json(g)::text FROM (%s) AS g", b.String()), args, nil
}

// joinSQL unpacks the list: a multi-select's options as m0.v, or a link cell's items as l0.v with the linked
// record as x. A record with an empty list is kept once, with NULLs, so it lands in the empty group.
func (j *aggJoin) joinSQL() string {
	col := "t." + pq.QuoteIdentifier(j.list.ColumnName)
	if j.target == nil {
		return fmt.Sprintf("LEFT JOIN LATERAL unnest(COALESCE(%s, '{}'::text[])) AS m0(v) ON true", col)
	}
	// Link cells hold the linked records' key objects: {"id": 12}, or the key columns on a table with a custom
	// primary key (where {"id": 12} can still occur)
	match := `x.id = (CASE WHEN l0.v - 'id' = '{}'::jsonb THEN l0.v->>'id' END)::bigint`
	if len(j.target.keyCols) > 0 {
		keys := make([]string, len(j.target.keyCols))
		for i, k := range j.target.keyCols {
			keys[i] = fmt.Sprintf("to_jsonb(x.%s) = l0.v->%s", pq.QuoteIdentifier(k), pq.QuoteLiteral(k))
		}
		match = fmt.Sprintf("(%s OR (%s))", match, strings.Join(keys, " AND "))
	}
	return fmt.Sprintf("LEFT JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(%[1]s::jsonb) = 'array' THEN %[1]s::jsonb ELSE '[]'::jsonb END) AS l0(v) ON true "+
		"LEFT JOIN %[2]s AS x ON %[3]s", col, j.target.table, match)
}

// ---- Answer ----

type groupOut struct {
	Group  orderedFields `json:"group"`
	Values orderedFields `json:"values"`
}

// linkGroupValue is a link group's value: the linked record as in records ({"id": "rec_12"}), or nil
func linkGroupValue(v interface{}) interface{} {
	n, ok := v.(json.Number)
	if !ok {
		return nil
	}
	id, err := n.Int64()
	if err != nil {
		return nil
	}
	return orderedFields{{key: refID, value: EncodeRecordID(id)}}
}

// answer turns the rows into groups: group values keyed like record fields, metric values by metric name
func (o *aggOptions) answer(rows []map[string]interface{}, fields []*field) []groupOut {
	keys := map[string]string{} // field ID -> key: the name, or the ID with keys=id or a shared name
	for f, key := range fieldKeys(fields, o.keysByID) {
		keys[f.ID] = key
	}
	out := make([]groupOut, 0, len(rows))
	for _, row := range rows {
		item := groupOut{Group: orderedFields{}, Values: orderedFields{}}
		for i, g := range o.groups {
			key := g.c.ref
			if g.c.f != nil {
				key = keys[g.c.f.ID]
			}
			v := row[fmt.Sprintf("g%d", i)]
			if g.c.kind == kindLink {
				v = linkGroupValue(v)
			}
			item.Group = append(item.Group, fieldValue{key: key, value: v})
		}
		for i, m := range o.metrics {
			v := row[fmt.Sprintf("m%d", i)]
			if m.c != nil && m.c.kind == kindDateTime && v != nil {
				v = apiTime(v) // min / max of a date-time: RFC 3339 in UTC, as in records
			}
			item.Values = append(item.Values, fieldValue{key: m.name, value: v})
		}
		out = append(out, item)
	}
	return out
}
