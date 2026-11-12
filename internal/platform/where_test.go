package platform

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func whereFields() []*field {
	return []*field{
		{ID: "00000000-0000-0000-0000-000000000001", Name: "Name", ColumnName: "title", Type: typeText},
		{ID: "00000000-0000-0000-0000-000000000002", Name: "Status", ColumnName: "status", Type: typeSelect},
		{ID: "00000000-0000-0000-0000-000000000003", Name: "Amount", ColumnName: "amount", Type: typeCurrency},
		{ID: "00000000-0000-0000-0000-000000000004", Name: "Paid", ColumnName: "paid", Type: typeCheckbox},
		{ID: "00000000-0000-0000-0000-000000000005", Name: "Tags", ColumnName: "tags", Type: typeMultiSelect},
		{ID: "00000000-0000-0000-0000-000000000006", Name: "Customer", ColumnName: "customer", Type: typeLinks, LinkTargetModelID: "t"},
		{ID: "00000000-0000-0000-0000-000000000007", Name: "Total", ColumnName: "total", Type: typeFormula},
		{ID: "00000000-0000-0000-0000-000000000008", Name: "Minutes", ColumnName: "minutes", Type: typeDuration},
		{ID: "00000000-0000-0000-0000-000000000009", Name: "Ref", ColumnName: "ref", Type: typeUUID},
	}
}

func testLinks(*field, int64) ([]string, bool, error) {
	return []string{`[{"id": 12}]`, `[{"code": "C12"}]`}, true, nil
}

// whereSQL returns the WHERE part of the list query and its arguments, without limit and offset
func whereSQL(t *testing.T, query url.Values) (string, []interface{}) {
	t.Helper()
	opts, err := parseListOptions(query, whereFields(), testLinks)
	require.NoError(t, err)
	sql, args, err := listSQL(`"s"."orders"`, opts)
	require.NoError(t, err)
	sql = strings.TrimPrefix(sql, `SELECT row_to_json(t)::text FROM "s"."orders" AS t`)
	sql = sql[:strings.Index(sql, " ORDER BY")]
	return strings.TrimPrefix(sql, " WHERE "), args[:len(args)-2]
}

func whereQuery(tree string) url.Values { return url.Values{"where": {tree}} }

func TestFilterParamsSQL(t *testing.T) {
	cases := []struct {
		query string
		sql   string
		args  []interface{}
	}{
		{"filter[Status]=Open", `("status" = $1)`, []interface{}{"Open"}},
		{"filter[Amount][gte]=100", `("amount" >= $1)`, []interface{}{100.0}},
		{"filter[Status][in]=Open,Pending", `("status" IN ($1, $2))`, []interface{}{"Open", "Pending"}},
		{"filter[Name][contains]=50%25", `(("title")::text ILIKE $1)`, []interface{}{`%50\%%`}},
		{"filter[Name][is_null]=true", `(("title" IS NULL) OR (("title")::text = $1))`, []interface{}{""}},
		{"filter[Name][is_null]=false", `(NOT COALESCE(((("title" IS NULL) OR (("title")::text = $1))), false))`, []interface{}{""}},
		{"filter[Paid]=false", `(("paid" = $1) OR ("paid" IS NULL))`, []interface{}{false}},
		{"filter[Paid]=true", `("paid" = $1)`, []interface{}{true}},
		{"filter[Tags][contains]=Gift", `($1 = ANY("tags"))`, []interface{}{"Gift"}},
		{"filter[Tags][is_null]=true", `(("tags" IS NULL) OR ("tags" = $1))`, []interface{}{"{}"}},
		{"filter[Customer][contains]=rec_12", `(("customer" @> $1::jsonb) OR ("customer" @> $2::jsonb))`, []interface{}{`[{"id": 12}]`, `[{"code": "C12"}]`}},
		{"filter[Minutes][gt]=30", `((CASE WHEN btrim(("minutes")::text) ~ '^-?[0-9]+([.][0-9]+)?$' THEN btrim(("minutes")::text)::numeric END) > $1)`, []interface{}{30.0}},
		{"filter[Ref]=abc", `(("ref")::text = $1)`, []interface{}{"abc"}},
		{"filter[id]=rec_7", `("id" = $1)`, []interface{}{int64(7)}},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			q, err := url.ParseQuery(tc.query)
			require.NoError(t, err)
			sql, args := whereSQL(t, q)
			assert.Equal(t, tc.sql, sql)
			assert.Equal(t, tc.args, args)
		})
	}
}

func TestWhereTreeSQL(t *testing.T) {
	sql, args := whereSQL(t, whereQuery(`{"match": "all", "conditions": [
		{"field": "Amount", "op": "gte", "value": 100},
		{"match": "any", "conditions": [{"field": "Status", "value": "Open"}, {"field": "Status", "value": "Pending"}]},
		{"match": "none", "conditions": [{"field": "Paid", "value": true}]}
	]}`))
	assert.Equal(t, `(("amount" >= $1) AND (("status" = $2) OR ("status" = $3)) AND (NOT COALESCE((("paid" = $4)), false)))`, sql)
	assert.Equal(t, []interface{}{100.0, "Open", "Pending", true}, args)
}

func TestWhereDefaultsAndLists(t *testing.T) {
	// match defaults to all, op to eq; a single condition is a tree too
	sql, args := whereSQL(t, whereQuery(`{"field": "Status", "value": "Open"}`))
	assert.Equal(t, `("status" = $1)`, sql)
	assert.Equal(t, []interface{}{"Open"}, args)

	// in takes a JSON list, so values may contain commas
	sql, args = whereSQL(t, whereQuery(`{"conditions": [{"field": "Status", "op": "in", "value": ["Open, new", "Closed"]}]}`))
	assert.Equal(t, `(("status" IN ($1, $2)))`, sql)
	assert.Equal(t, []interface{}{"Open, new", "Closed"}, args)

	// is_null without a value finds empty fields
	sql, _ = whereSQL(t, whereQuery(`{"field": "Paid", "op": "is_null"}`))
	assert.Equal(t, `("paid" IS NULL)`, sql)
}

func TestWhereCombinesWithFilterParams(t *testing.T) {
	q := whereQuery(`{"match": "any", "conditions": [{"field": "Status", "value": "Open"}, {"field": "Amount", "op": "lt", "value": 5}]}`)
	q.Set("filter[Paid]", "true")
	sql, _ := whereSQL(t, q)
	assert.Equal(t, `("paid" = $1) AND (("status" = $2) OR ("amount" < $3))`, sql)
}

func TestWhereErrors(t *testing.T) {
	cases := map[string]string{
		`not json`: `where must be JSON`,
		`[]`:       `where must be a group`,
		`{"match": "xor", "conditions": [{"field": "Status", "value": "x"}]}`:                                                                         `where.match must be all, any or none.`,
		`{"conditions": [{"match": "all", "conditions": []}]}`:                                                                                        `where.conditions[0].conditions must be a list with at least one condition.`,
		`{"conditions": [{"field": "Staus", "value": "x"}]}`:                                                                                          `where.conditions[0]: No field named "Staus". Did you mean "Status"?`,
		`{"conditions": [{"field": "Total", "op": "gt", "value": 1}]}`:                                                                                `"Total" is a formula field. Formula and lookup fields are calculated when read, so they can't be used to filter or sort.`,
		`{"conditions": [{"field": "Amount", "op": "contains", "value": "1"}]}`:                                                                       `where.conditions[0]: "Amount" is a currency field. You can use: eq, neq, gt, gte, lt, lte, in, is_null.`,
		`{"conditions": [{"field": "Amount", "value": "cheap"}]}`:                                                                                     `"cheap" is not a valid value for this field.`,
		`{"conditions": [{"field": "Amount", "value": null}]}`:                                                                                        `value can't be null`,
		`{"conditions": [{"field": "Amount", "value": [1, 2]}]}`:                                                                                      `a list of values works only with the in operator.`,
		`{"conditions": [{"field": "Amount", "value": 1, "match": "all"}]}`:                                                                           `Use one or the other.`,
		`{"conditions": [{"field": "Amount", "value": 1, "operator": "gt"}]}`:                                                                         `unknown key "operator"`,
		`{"conditions": [{"field": "Amount"}]}`:                                                                                                       `where.conditions[0].value is missing.`,
		`{"conditions": [{"conditions": [{"conditions": [{"conditions": [{"conditions": [{"conditions": [{"field": "Amount", "value": 1}]}]}]}]}]}]}`: `groups can be nested at most 5 deep.`,
	}
	for tree, want := range cases {
		t.Run(tree, func(t *testing.T) {
			_, err := parseListOptions(whereQuery(tree), whereFields(), testLinks)
			require.Error(t, err)
			apiErr, ok := err.(*Error)
			require.True(t, ok)
			assert.Equal(t, CodeBadRequest, apiErr.Code)
			assert.Contains(t, apiErr.Message, want)
		})
	}
}

func TestWhereConditionLimit(t *testing.T) {
	conds := make([]string, maxWhereConditions+1)
	for i := range conds {
		conds[i] = `{"field": "Amount", "value": 1}`
	}
	_, err := parseListOptions(whereQuery(`{"conditions": [`+strings.Join(conds, ",")+`]}`), whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 50 conditions")
}

func TestFormulaCantBeSortedOrFiltered(t *testing.T) {
	_, err := parseListOptions(url.Values{"sort": {"Total"}}, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "calculated when read")
	_, err = parseListOptions(url.Values{"filter[Total]": {"1"}}, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "calculated when read")
}

func TestWhereCursorHash(t *testing.T) {
	hash := func(tree string) string {
		opts, err := parseListOptions(whereQuery(tree), whereFields(), testLinks)
		require.NoError(t, err)
		return opts.hash
	}
	a := hash(`{"match": "any", "conditions": [{"field": "Status", "value": "Open"}, {"field": "Amount", "value": 1}]}`)
	// Same conditions in another order, keys in another order: same filter
	b := hash(`{"conditions": [{"value": 1, "field": "Amount"}, {"field": "Status", "value": "Open"}], "match": "any"}`)
	c := hash(`{"match": "all", "conditions": [{"field": "Status", "value": "Open"}, {"field": "Amount", "value": 1}]}`)
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)

	// A cursor from the first page only works with the same where
	cursor := encodeCursor(50, a)
	q := whereQuery(`{"match": "all", "conditions": [{"field": "Status", "value": "Open"}, {"field": "Amount", "value": 1}]}`)
	q.Set("cursor", cursor)
	_, err := parseListOptions(q, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different filter or sort")
}

// Filters without a where keep the hash they had before the where tree existed, so old cursors keep working
func TestFilterParamHashUnchanged(t *testing.T) {
	opts, err := parseListOptions(url.Values{"filter[Status]": {"Open"}}, whereFields(), testLinks)
	require.NoError(t, err)
	assert.Equal(t, optionsHash(&listOptions{
		filters: []filterTerm{{key: "00000000-0000-0000-0000-000000000002:eq:Open"}},
		sorts:   []sortTerm{{key: refID}},
	}), opts.hash)
}

func TestEmptySelectReturnsAllFields(t *testing.T) {
	all := whereFields()
	for _, q := range []url.Values{{}, {"select": {""}}, {"select": {" , "}}} {
		opts, err := parseListOptions(q, all, testLinks)
		require.NoError(t, err)
		assert.Len(t, opts.fields, len(all), "select %q", q["select"])
	}
	opts, err := parseListOptions(url.Values{"select": {"Status"}}, all, testLinks)
	require.NoError(t, err)
	require.Len(t, opts.fields, 1)
	assert.Equal(t, "Status", opts.fields[0].Name)
	// Only system values asked for: just those
	opts, err = parseListOptions(url.Values{"select": {"id"}}, all, testLinks)
	require.NoError(t, err)
	assert.Empty(t, opts.fields)
}

func TestEmptyWhereAndFilterFilterNothing(t *testing.T) {
	none, err := parseListOptions(url.Values{}, whereFields(), testLinks)
	require.NoError(t, err)
	empty := []url.Values{
		{"where": {""}}, {"where": {"  "}}, {"where": {"{}"}}, {"where": {"null"}},
		{"where": {`{"conditions": []}`}}, {"where": {`{"match": "any"}`}}, {"where": {`{"match": "none", "conditions": null}`}},
		{"filter": {""}}, {"filter": {"{}"}}, {"filter": {"null"}},
	}
	for _, q := range empty {
		opts, err := parseListOptions(q, whereFields(), testLinks)
		require.NoError(t, err, "%v", q)
		assert.Empty(t, opts.filters, "%v", q)
		assert.Equal(t, none.hash, opts.hash, "%v: the same cursor hash as no filter", q)
	}

	// Still errors: a bad match, an empty group inside the tree, a bare filter with a value
	for _, q := range []url.Values{
		{"where": {`{"match": "xor"}`}},
		{"where": {`{"conditions": [{"match": "any", "conditions": []}]}`}},
		{"where": {"[]"}},
		{"filter": {"Open"}},
	} {
		_, err := parseListOptions(q, whereFields(), testLinks)
		assert.Error(t, err, "%v", q)
	}
}
