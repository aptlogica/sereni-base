// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func aggFields() []*field {
	return append(whereFields(),
		&field{ID: "00000000-0000-0000-0000-000000000010", Name: "Due", ColumnName: "due", Type: typeDate},
		&field{ID: "00000000-0000-0000-0000-000000000011", Name: "Ordered at", ColumnName: "ordered_at", Type: typeDateTime},
		&field{ID: "00000000-0000-0000-0000-000000000012", Name: "Notes", ColumnName: "notes", Type: typeLongText},
	)
}

// aggBody decodes a request body the way readBody does: numbers stay json.Number
func aggBody(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	v, err := decodeJSON([]byte(raw))
	require.NoError(t, err)
	return v.(map[string]interface{})
}

func aggSQL(t *testing.T, raw string, filters ...filterTerm) (string, []interface{}) {
	t.Helper()
	opts, err := parseAggregate(aggBody(t, raw), aggFields(), nil)
	require.NoError(t, err)
	sql, args, err := aggregateSQL(`"s"."orders"`, opts, filters)
	require.NoError(t, err)
	return sql, args
}

func TestAggregateSQL(t *testing.T) {
	sql, args := aggSQL(t, `{
		"group_by": ["Status", {"field": "Ordered at", "by": "month"}],
		"metrics": [{"op": "count"}, {"op": "sum", "field": "Amount", "as": "total"}, {"op": "avg", "field": "Minutes"},
		            {"op": "count_distinct", "field": "Name", "as": "names"}, {"op": "max", "field": "Due"}],
		"timezone": "Asia/Kolkata", "sort": ["-total"], "limit": 10}`)
	assert.Equal(t, `SELECT row_to_json(g)::text FROM (SELECT NULLIF(t."status"::text, '') AS g0, `+
		`date_trunc('month', (t."ordered_at" AT TIME ZONE 'UTC') AT TIME ZONE $1)::date AS g1, `+
		`COUNT(*) AS m0, COALESCE(SUM(t."amount"), 0) AS m1, `+
		`AVG((CASE WHEN t."minutes" ~ '^\s*-?[0-9]+(\.[0-9]+)?\s*$' THEN t."minutes"::numeric END)) AS m2, `+
		`COUNT(DISTINCT NULLIF(t."title"::text, '')) AS m3, MAX(t."due") AS m4 `+
		`FROM "s"."orders" AS t GROUP BY 1, 2 ORDER BY m1 DESC NULLS LAST, g0 ASC NULLS LAST, g1 ASC NULLS LAST LIMIT $2) AS g`, sql)
	assert.Equal(t, []interface{}{"Asia/Kolkata", 11}, args, "limit+1 rows tell whether there are more groups")
}

func TestAggregateWithoutGroupsIsOneRow(t *testing.T) {
	sql, args := aggSQL(t, `{"metrics": [{"op": "count"}, {"op": "min", "field": "created_at"}]}`)
	assert.Equal(t, `SELECT row_to_json(g)::text FROM (SELECT COUNT(*) AS m0, MIN(t."created_time") AS m1 FROM "s"."orders" AS t) AS g`, sql)
	assert.Empty(t, args)
}

func TestAggregateUsesTheListFilter(t *testing.T) {
	filters, err := parseFilters(whereQuery(`{"field": "Paid", "value": true}`), aggFields(), testLinks)
	require.NoError(t, err)
	sql, args := aggSQL(t, `{"group_by": [{"field": "Due", "by": "week"}, "Paid"], "metrics": [{"op": "count"}]}`, filters...)
	assert.Contains(t, sql, `SELECT date_trunc('week', t."due")::date AS g0, COALESCE(t."paid", false) AS g1, COUNT(*) AS m0 `+
		`FROM "s"."orders" AS t WHERE ("paid" = $1) GROUP BY 1, 2 ORDER BY g0 ASC NULLS LAST, g1 ASC NULLS LAST LIMIT $2`)
	assert.Equal(t, []interface{}{true, 101}, args, "default limit 100")
}

func TestAggregateDefaultNamesAndSortByGroup(t *testing.T) {
	opts, err := parseAggregate(aggBody(t, `{"group_by": ["Status"], "metrics": [{"op": "count"}, {"op": "avg", "field": "Amount"}],
		"sort": ["-Status", "count"]}`), aggFields(), nil)
	require.NoError(t, err)
	assert.Equal(t, "count", opts.metrics[0].name)
	assert.Equal(t, "avg_Amount", opts.metrics[1].name)
	assert.Equal(t, []aggSort{{alias: "g0", desc: true}, {alias: "m0"}}, opts.sorts)
}

func TestAggregateErrors(t *testing.T) {
	cases := map[string]string{
		`{}`:              `metrics is required`,
		`{"metrics": []}`: `metrics is required`,
		`{"metrics": [{"op": "median", "field": "Amount"}]}`:                               `metrics[0].op must be one of count, count_distinct, sum, avg, min, max.`,
		`{"metrics": [{"op": "sum"}]}`:                                                     `metrics[0]: sum needs a field.`,
		`{"metrics": [{"op": "count", "field": "Amount"}]}`:                                `count counts records and takes no field`,
		`{"metrics": [{"op": "sum", "field": "Status"}]}`:                                  `metrics[0]: sum doesn't work on "Status", a single_select field. sum takes a number field`,
		`{"metrics": [{"op": "max", "field": "Name"}]}`:                                    `max takes a number, date or date-time field.`,
		`{"metrics": [{"op": "sum", "field": "Amout"}]}`:                                   `metrics[0].field: No field named "Amout". Did you mean "Amount"?`,
		`{"metrics": [{"op": "sum", "field": "Total"}]}`:                                   `"Total" is a formula field. Its value is calculated when read, so it can't be grouped or aggregated.`,
		`{"metrics": [{"op": "count"}, {"op": "count"}]}`:                                  `metrics[1]: two metrics are named "count". Give one a different name with "as".`,
		`{"metrics": [{"op": "count", "as": ""}]}`:                                         `metrics[0].as must be text of 1 to 100 characters.`,
		`{"metrics": [{"op": "count", "extra": 1}]}`:                                       `metrics[0]: unknown key "extra"`,
		`{"metrics": [{"op": "count"}], "group_by": ["Ordered at"]}`:                       `group_by[0]: "Ordered at" is a date-time field. Group it by day, week, month or year`,
		`{"metrics": [{"op": "count"}], "group_by": [{"field": "Status", "by": "month"}]}`: `group_by[0]: by works only with date and date-time fields; "Status" is a single_select field.`,
		`{"metrics": [{"op": "count"}], "group_by": [{"field": "Due", "by": "hour"}]}`:     `group_by[0].by must be day, week, month, year.`,
		`{"metrics": [{"op": "count"}], "group_by": ["Notes"]}`:                            `"Notes" is a long_text field and can't be grouped.`,
		`{"metrics": [{"op": "count"}], "group_by": ["Status", "Status"]}`:                 `group_by[1]: "Status" is already in group_by.`,
		`{"metrics": [{"op": "count"}], "group_by": ["Status", "Paid", "Due", "Name"]}`:    `group_by: at most 3 fields.`,
		`{"metrics": [{"op": "count"}], "group_by": ["id"]}`:                               `can't be grouped or aggregated by id`,
		`{"metrics": [{"op": "count"}], "sort": ["Amount"]}`:                               `sort: "Amount" is not a metric name or a group_by field.`,
		`{"metrics": [{"op": "count"}], "sort": ["count", "-count"]}`:                      `sort: "count" is listed twice.`,
		`{"metrics": [{"op": "count"}], "limit": 1001}`:                                    `limit must be a whole number from 1 to 1000.`,
		`{"metrics": [{"op": "count"}], "limit": "5"}`:                                     `limit must be a whole number from 1 to 1000.`,
		`{"metrics": [{"op": "count"}], "timezone": "Mars/Base"}`:                          `timezone: "Mars/Base" is not a time zone.`,
		`{"metrics": [{"op": "count"}], "timezone": "Local"}`:                              `is not a time zone`,
		`{"metrics": [{"op": "count"}], "keys": "x"}`:                                      `keys must be id or name.`,
	}
	for body, want := range cases {
		_, err := parseAggregate(aggBody(t, body), aggFields(), nil)
		require.Error(t, err, body)
		assert.Contains(t, err.Error(), want, body)
	}

	tooMany := `{"metrics": [`
	for i := 0; i <= maxMetrics; i++ {
		if i > 0 {
			tooMany += ","
		}
		tooMany += `{"op": "count", "as": "c` + string(rune('a'+i)) + `"}`
	}
	_, err := parseAggregate(aggBody(t, tooMany+`]}`), aggFields(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metrics: at most 10.")
}

func TestAggregateAnswer(t *testing.T) {
	opts, err := parseAggregate(aggBody(t, `{"group_by": ["Status", "created_at"], "metrics": [{"op": "count"}, {"op": "max", "field": "Ordered at"}]}`), aggFields(), nil)
	if err == nil {
		t.Fatal("created_at without by must be refused")
	}
	opts, err = parseAggregate(aggBody(t, `{"group_by": ["Status", {"field": "created_at", "by": "day"}],
		"metrics": [{"op": "count"}, {"op": "max", "field": "Ordered at", "as": "last"}]}`), aggFields(), nil)
	require.NoError(t, err)
	rows := []map[string]interface{}{
		{"g0": "Open", "g1": "2026-11-13", "m0": json.Number("3"), "m1": "2026-11-13T10:15:00"},
		{"g0": nil, "g1": "2026-11-12", "m0": json.Number("1"), "m1": nil},
	}
	raw, err := json.Marshal(opts.answer(rows, aggFields()))
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"group": {"Status": "Open", "created_at": "2026-11-13"}, "values": {"count": 3, "last": "2026-11-13T10:15:00Z"}},
		{"group": {"Status": null, "created_at": "2026-11-12"}, "values": {"count": 1, "last": null}}
	]`, string(raw))

	// keys=id keys group fields by their IDs
	opts, err = parseAggregate(aggBody(t, `{"group_by": ["Status"], "metrics": [{"op": "count"}], "keys": "id"}`), aggFields(), nil)
	require.NoError(t, err)
	raw, _ = json.Marshal(opts.answer([]map[string]interface{}{{"g0": "Open", "m0": json.Number("2")}}, aggFields()))
	assert.JSONEq(t, `[{"group": {"fld_00000000000000000000000000000002": "Open"}, "values": {"count": 2}}]`, string(raw))

	// Without group_by the group is {}
	opts, err = parseAggregate(aggBody(t, `{"metrics": [{"op": "count"}]}`), aggFields(), nil)
	require.NoError(t, err)
	raw, _ = json.Marshal(opts.answer([]map[string]interface{}{{"m0": json.Number("0")}}, aggFields()))
	assert.JSONEq(t, `[{"group": {}, "values": {"count": 0}}]`, string(raw))
}

// ---- Grouping by link, multi-select and lookup ----

// linkAggFields adds lookups through Customer, and a second link, to aggFields
func linkAggFields() []*field {
	customer := "00000000-0000-0000-0000-000000000006"
	return append(aggFields(),
		&field{ID: "00000000-0000-0000-0000-000000000020", Name: "Country", Type: typeLookup,
			LookupLinkColumnID: customer, LookupTargetColumnID: "00000000-0000-0000-0000-000000000101"},
		&field{ID: "00000000-0000-0000-0000-000000000021", Name: "Customer since", Type: typeLookup,
			LookupLinkColumnID: customer, LookupTargetColumnID: "00000000-0000-0000-0000-000000000102"},
		&field{ID: "00000000-0000-0000-0000-000000000022", Name: "VIP", Type: typeLookup,
			LookupLinkColumnID: customer, LookupTargetColumnID: "00000000-0000-0000-0000-000000000103"},
		&field{ID: "00000000-0000-0000-0000-000000000023", Name: "Customer notes", Type: typeLookup,
			LookupLinkColumnID: customer, LookupTargetColumnID: "00000000-0000-0000-0000-000000000104"},
		&field{ID: "00000000-0000-0000-0000-000000000024", Name: "Owner", ColumnName: "owner", Type: typeLinks, LinkTargetModelID: "u"},
	)
}

// testTargets is Customers (default key, or a custom key on "code") and Users
func testTargets(customKey bool) aggTargets {
	customers := &aggTarget{table: `"s"."customers"`, fields: []*field{
		{ID: "00000000-0000-0000-0000-000000000101", Name: "Country", ColumnName: "country", Type: typeSelect},
		{ID: "00000000-0000-0000-0000-000000000102", Name: "Since", ColumnName: "since", Type: typeDate},
		{ID: "00000000-0000-0000-0000-000000000103", Name: "VIP", ColumnName: "vip", Type: typeCheckbox},
		{ID: "00000000-0000-0000-0000-000000000104", Name: "Notes", ColumnName: "notes", Type: typeLongText},
	}}
	if customKey {
		customers.keyCols = []string{"code"}
	}
	return func(link *field) (*aggTarget, error) {
		switch link.Name {
		case "Customer":
			return customers, nil
		case "Owner":
			return &aggTarget{table: `"s"."users"`}, nil
		}
		return nil, nil
	}
}

func linkAggSQL(t *testing.T, raw string, customKey bool, filters ...filterTerm) string {
	t.Helper()
	opts, err := parseAggregate(aggBody(t, raw), linkAggFields(), testTargets(customKey))
	require.NoError(t, err)
	sql, _, err := aggregateSQL(`"s"."orders"`, opts, filters)
	require.NoError(t, err)
	return sql
}

const customerJoin = `LEFT JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(t."customer"::jsonb) = 'array' ` +
	`THEN t."customer"::jsonb ELSE '[]'::jsonb END) AS l0(v) ON true LEFT JOIN "s"."customers" AS x ON ` +
	`x.id = (CASE WHEN l0.v - 'id' = '{}'::jsonb THEN l0.v->>'id' END)::bigint`

func TestAggregateGroupByLink(t *testing.T) {
	// One row per record and linked record, then grouped; the filter runs on the table before the join
	filters, err := parseFilters(whereQuery(`{"field": "Paid", "value": true}`), aggFields(), testLinks)
	require.NoError(t, err)
	sql := linkAggSQL(t, `{"group_by": ["Customer"], "metrics": [{"op": "sum", "field": "Amount", "as": "total"}], "sort": ["-total"]}`, false, filters...)
	assert.Equal(t, `SELECT row_to_json(g)::text FROM (SELECT t."__g0" AS g0, COALESCE(SUM(t."amount"), 0) AS m0 FROM `+
		`(SELECT DISTINCT ON (t.id, x.id) t.*, x.id AS "__g0" FROM (SELECT * FROM "s"."orders" AS t WHERE ("paid" = $1)) AS t `+
		customerJoin+`) AS t GROUP BY 1 ORDER BY m0 DESC NULLS LAST, g0 ASC NULLS LAST LIMIT $2) AS g`, sql)

	// A custom primary key: cells hold the key columns, or still {"id": n}
	sql = linkAggSQL(t, `{"group_by": ["Customer"], "metrics": [{"op": "count"}]}`, true)
	assert.Contains(t, sql, `LEFT JOIN "s"."customers" AS x ON (x.id = (CASE WHEN l0.v - 'id' = '{}'::jsonb THEN l0.v->>'id' END)::bigint `+
		`OR (to_jsonb(x."code") = l0.v->'code'))`)
}

func TestAggregateGroupByMultiSelectAndLookup(t *testing.T) {
	sql := linkAggSQL(t, `{"group_by": ["Tags", "Status"], "metrics": [{"op": "count"}]}`, false)
	assert.Equal(t, `SELECT row_to_json(g)::text FROM (SELECT t."__g0" AS g0, NULLIF(t."status"::text, '') AS g1, COUNT(*) AS m0 FROM `+
		`(SELECT DISTINCT ON (t.id, NULLIF(m0.v, '')) t.*, NULLIF(m0.v, '') AS "__g0" FROM (SELECT * FROM "s"."orders" AS t) AS t `+
		`LEFT JOIN LATERAL unnest(COALESCE(t."tags", '{}'::text[])) AS m0(v) ON true) AS t `+
		`GROUP BY 1, 2 ORDER BY g0 ASC NULLS LAST, g1 ASC NULLS LAST LIMIT $1) AS g`, sql)

	// Lookups through the same link share one join; a record counts once per country, however many of its
	// customers are in it
	sql = linkAggSQL(t, `{"group_by": ["Country", {"field": "Customer since", "by": "year"}, "VIP"], "metrics": [{"op": "count"}]}`, false)
	assert.Contains(t, sql, `(SELECT DISTINCT ON (t.id, NULLIF(x."country"::text, ''), date_trunc('year', x."since")::date, `+
		`CASE WHEN x.id IS NOT NULL THEN COALESCE(x."vip", false) END) t.*, NULLIF(x."country"::text, '') AS "__g0", `+
		`date_trunc('year', x."since")::date AS "__g1", CASE WHEN x.id IS NOT NULL THEN COALESCE(x."vip", false) END AS "__g2" `+
		`FROM (SELECT * FROM "s"."orders" AS t) AS t `+customerJoin+`) AS t`)

	// A link and a lookup through it can be grouped together, with one join
	sql = linkAggSQL(t, `{"group_by": ["Customer", "Country"], "metrics": [{"op": "count"}]}`, false)
	assert.Contains(t, sql, `SELECT t."__g0" AS g0, t."__g1" AS g1, COUNT(*) AS m0`)
	assert.Equal(t, 1, strings.Count(sql, "jsonb_array_elements"), "one join for both")
}

func TestAggregateListGroupErrors(t *testing.T) {
	cases := map[string]string{
		`{"metrics": [{"op": "count"}], "group_by": ["Customer", "Owner"]}`:              `group_by[1]: group_by can use one link or multi-select field per request, and it already has "Customer"; a lookup counts as its link field.`,
		`{"metrics": [{"op": "count"}], "group_by": ["Tags", "Country"]}`:                `group_by can use one link or multi-select field per request`,
		`{"metrics": [{"op": "count"}], "group_by": ["Customer notes"]}`:                 `group_by[0]: "Customer notes" looks up a long_text field, which can't be grouped.`,
		`{"metrics": [{"op": "count"}], "group_by": [{"field": "Tags", "by": "month"}]}`: `by works only with date and date-time fields; "Tags" is a multi_select field.`,
		`{"metrics": [{"op": "count_distinct", "field": "Country"}]}`:                    `metrics[0]: "Country" is a lookup field; it can be used in group_by, not in metrics.`,
		`{"metrics": [{"op": "count_distinct", "field": "Customer"}]}`:                   `"Customer" is a links field; it can be used in group_by, not in metrics.`,
		`{"metrics": [{"op": "count_distinct", "field": "Tags"}]}`:                       `"Tags" is a multi_select field; it can be used in group_by, not in metrics.`,
	}
	for body, want := range cases {
		_, err := parseAggregate(aggBody(t, body), linkAggFields(), testTargets(false))
		require.Error(t, err, body)
		assert.Contains(t, err.Error(), want, body)
	}

	// The linked table was deleted
	_, err := parseAggregate(aggBody(t, `{"metrics": [{"op": "count"}], "group_by": ["Country"]}`), linkAggFields(),
		func(*field) (*aggTarget, error) { return nil, nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), `group_by[0]: the linked table of "Customer" no longer exists.`)
}

func TestAggregateLinkGroupAnswer(t *testing.T) {
	opts, err := parseAggregate(aggBody(t, `{"group_by": ["Customer", "Country"], "metrics": [{"op": "count"}]}`), linkAggFields(), testTargets(false))
	require.NoError(t, err)
	rows := []map[string]interface{}{
		{"g0": json.Number("12"), "g1": "India", "m0": json.Number("2")},
		{"g0": nil, "g1": nil, "m0": json.Number("1")},
	}
	raw, err := json.Marshal(opts.answer(rows, linkAggFields()))
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"group": {"Customer": {"id": "rec_12"}, "Country": "India"}, "values": {"count": 2}},
		{"group": {"Customer": null, "Country": null}, "values": {"count": 1}}
	]`, string(raw))
}
