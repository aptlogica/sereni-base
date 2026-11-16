package platform

import (
	"fmt"
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

// fakeLinks is a linkResolver without a database. Customer links to a table with Country, Notes and Company;
// matching finds `matches` linked records and remembers the last condition it was asked for.
type fakeLinks struct {
	matches  int
	lastCond filterTerm
}

func (*fakeLinks) keys(*field, int64) ([]string, bool, error) {
	return []string{`[{"id": 12}]`, `[{"code": "C12"}]`}, true, nil
}

func (*fakeLinks) target(link *field) ([]*field, bool, error) {
	if link.LinkTargetModelID == "gone" {
		return nil, false, nil
	}
	return []*field{
		{ID: "00000000-0000-0000-0000-000000000101", Name: "Country", ColumnName: "country", Type: typeSelect},
		{ID: "00000000-0000-0000-0000-000000000102", Name: "Since", ColumnName: "since", Type: typeDate},
		{ID: "00000000-0000-0000-0000-000000000103", Name: "Company", ColumnName: "company", Type: typeLinks, LinkTargetModelID: "c"},
	}, true, nil
}

func (l *fakeLinks) matching(_ *field, cond filterTerm, limit int) ([]string, int, error) {
	l.lastCond = cond
	var arrays []string
	n := min(l.matches, limit+1)
	for i := 1; i <= n; i++ {
		arrays = append(arrays, fmt.Sprintf(`[{"id": %d}]`, i))
	}
	return arrays, n, nil
}

var testLinks linkResolver = &fakeLinks{matches: 2}

// whereSQL returns the WHERE part of the list query and its arguments, without limit
func whereSQL(t *testing.T, query url.Values) (string, []interface{}) {
	t.Helper()
	opts, err := parseListOptions(query, whereFields(), testLinks)
	require.NoError(t, err)
	sql, args, err := listSQL(`"s"."orders"`, opts)
	require.NoError(t, err)
	sql = sql[strings.Index(sql, ` FROM "s"."orders" AS t`)+len(` FROM "s"."orders" AS t`):]
	sql = sql[:strings.Index(sql, " ORDER BY")]
	return strings.TrimPrefix(sql, " WHERE "), args[:len(args)-1]
}

func whereQuery(tree string) url.Values { return url.Values{"where": {tree}} }

func TestFilterParamsSQL(t *testing.T) {
	cases := []struct {
		query string
		sql   string
		args  []interface{}
	}{
		{"filter[Status]=Open", `("status" = $1)`, []interface{}{"Open"}},
		{"filter[Amount][gte]=100", `("amount" >= $1)`, []interface{}{"100"}},
		{"filter[Status][in]=Open,Pending", `("status" IN ($1, $2))`, []interface{}{"Open", "Pending"}},
		{"filter[Name][contains]=50%25", `(("title")::text ILIKE $1)`, []interface{}{`%50\%%`}},
		{"filter[Name][is_null]=true", `(("title" IS NULL) OR (("title")::text = $1))`, []interface{}{""}},
		{"filter[Name][is_null]=false", `(NOT COALESCE(((("title" IS NULL) OR (("title")::text = $1))), false))`, []interface{}{""}},
		{"filter[Paid]=false", `(("paid" = $1) OR ("paid" IS NULL))`, []interface{}{false}},
		{"filter[Paid]=true", `("paid" = $1)`, []interface{}{true}},
		{"filter[Tags][contains]=Gift", `($1 = ANY("tags"))`, []interface{}{"Gift"}},
		{"filter[Tags][is_null]=true", `(("tags" IS NULL) OR ("tags" = $1))`, []interface{}{"{}"}},
		{"filter[Customer][contains]=rec_12", `(("customer" @> $1::jsonb) OR ("customer" @> $2::jsonb))`, []interface{}{`[{"id": 12}]`, `[{"code": "C12"}]`}},
		{"filter[Minutes][gt]=30", `((CASE WHEN btrim(("minutes")::text) ~ '^-?[0-9]+([.][0-9]+)?$' THEN btrim(("minutes")::text)::numeric END) > $1)`, []interface{}{"30"}},
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
	assert.Equal(t, []interface{}{"100", "Open", "Pending", true}, args)
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
		`{"conditions": [{"field": "Total", "op": "gt", "value": 1}]}`:                                                                                `"Total" is a formula field. Its value is calculated when read, so it can't be used to filter or sort.`,
		`{"conditions": [{"field": "Amount", "op": "contains", "value": "1"}]}`:                                                                       `where.conditions[0]: "Amount" is a currency field. You can use: eq, neq, gt, gte, lt, lte, in, is_null.`,
		`{"conditions": [{"field": "Amount", "value": "cheap"}]}`:                                                                                     `"cheap" is not a valid value for this field. Use a number, like 12.5.`,
		`{"conditions": [{"field": "Amount", "op": "in", "value": [1, "x"]}]}`:                                                                        `"x" is not a valid value for this field. Use a number`,
		`{"conditions": [{"field": "created_at", "op": "gte", "value": "2026-11-01"}]}`:                                                               `"2026-11-01" is not a valid value for this field. Use a date-time with a time zone (RFC 3339), like 2026-11-01T00:00:00Z`,
		`{"conditions": [{"field": "id", "value": "12"}]}`:                                                                                            `Use a record ID like rec_12.`,
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

// G29: numbers go to Postgres as the text sent; G27: multi-user contains matches whole user IDs
func TestNumberAndMultiUserFilters(t *testing.T) {
	fields := append(whereFields(),
		&field{ID: "00000000-0000-0000-0000-000000000030", Name: "Qty", ColumnName: "qty", Type: typeNumber},
		&field{ID: "00000000-0000-0000-0000-000000000031", Name: "Owners", ColumnName: "owners", Type: typeUser, Multiple: true})
	where := func(q url.Values) (string, []interface{}, error) {
		opts, err := parseListOptions(q, fields, testLinks)
		if err != nil {
			return "", nil, err
		}
		sql, args, _, err := whereClause(opts.filters, 1)
		return strings.TrimPrefix(sql, " WHERE "), args, err
	}
	ok := func(param, value string) (string, []interface{}) {
		t.Helper()
		sql, args, err := where(url.Values{param: {value}})
		require.NoError(t, err, param)
		return sql, args
	}

	// Exact: no float64 rounding
	_, args := ok("filter[Amount]", "12345678901234567.89")
	assert.Equal(t, []interface{}{"12345678901234567.89"}, args)
	// A whole-number field compares as is with whole numbers...
	sql, args := ok("filter[Qty][gt]", " 2 ")
	assert.Equal(t, `("qty" > $1)`, sql)
	assert.Equal(t, []interface{}{"2"}, args)
	// ...and as numbers with a fraction or a number past its range, which Postgres would refuse for INTEGER
	for _, v := range []string{"1.5", "99999999999", "1e3"} {
		sql, _ = ok("filter[Qty][gt]", v)
		assert.Contains(t, sql, `btrim(("qty")::text)`, v)
	}
	sql, _ = ok("filter[Qty][in]", "1,2.5")
	assert.Contains(t, sql, `btrim(("qty")::text)`)
	// Go reads these as numbers, Postgres doesn't
	for _, v := range []string{"Inf", "NaN", "0x10", "1_000", ""} {
		_, _, err := where(url.Values{"filter[Amount]": {v}})
		require.Error(t, err, v)
		assert.Contains(t, err.Error(), "Use a number", v)
	}

	// Multi-user: one whole ID, with _ escaped, alone, first, last or in the middle, with or without a space
	_, args = ok("filter[Owners][contains]", "u_1")
	assert.Equal(t, []interface{}{"u_1", `u\_1,%`, `%,u\_1`, `%, u\_1`, `%,u\_1,%`, `%, u\_1,%`}, args)
	_, _ = ok("filter[Owners][is_null]", "true")
	for param, want := range map[string]string{
		"filter[Owners]":           `"Owners" is a user field. You can use: contains, is_null.`,
		"filter[Owners][contains]": `the value must be one user ID.`,
		"sort":                     `records can't be sorted by "Owners"`,
	} {
		value := "Owners"
		if param == "filter[Owners][contains]" {
			value = "u1,u2"
		} else if param == "filter[Owners]" {
			value = "u1"
		}
		_, _, err := where(url.Values{param: {value}})
		require.Error(t, err, param)
		assert.Contains(t, err.Error(), want, param)
	}
}

// G6: a condition on a field of the linked records, named Customer.Country or by a lookup
func TestLinkedFieldFilters(t *testing.T) {
	fields := append(whereFields(),
		&field{ID: "00000000-0000-0000-0000-000000000040", Name: "Customer country", Type: typeLookup,
			LookupLinkColumnID: "00000000-0000-0000-0000-000000000006", LookupTargetColumnID: "00000000-0000-0000-0000-000000000101"},
		&field{ID: "00000000-0000-0000-0000-000000000041", Name: "Old", ColumnName: "old", Type: typeLinks, LinkTargetModelID: "gone"})
	run := func(q url.Values, matches int) (*listOptions, *fakeLinks, error) {
		links := &fakeLinks{matches: matches}
		opts, err := parseListOptions(q, fields, links)
		return opts, links, err
	}
	sqlOf := func(opts *listOptions) (string, []interface{}) {
		sql, args, _, err := whereClause(opts.filters, 1)
		require.NoError(t, err)
		return strings.TrimPrefix(sql, " WHERE "), args
	}

	// The linked records where the condition holds are found first; then their keys are matched in the link
	opts, links, err := run(url.Values{"filter[Customer.Country]": {"India"}}, 2)
	require.NoError(t, err)
	inner, innerArgs, _, _ := whereClause([]filterTerm{links.lastCond}, 1)
	assert.Equal(t, ` WHERE ("country" = $1)`, inner, "the condition runs on the linked table")
	assert.Equal(t, []interface{}{"India"}, innerArgs)
	sql, args := sqlOf(opts)
	assert.Equal(t, `(("customer" @> $1::jsonb) OR ("customer" @> $2::jsonb))`, sql)
	assert.Equal(t, []interface{}{`[{"id": 1}]`, `[{"id": 2}]`}, args)

	// The lookup is the same filter, with the same cursor
	byLookup, _, err := run(url.Values{"filter[Customer country]": {"India"}}, 2)
	require.NoError(t, err)
	assert.Equal(t, opts.hash, byLookup.hash)

	// none: records with no linked customer in India, including records with no customer
	opts, _, err = run(whereQuery(`{"match": "none", "conditions": [{"field": "Customer.Country", "value": "India"}]}`), 1)
	require.NoError(t, err)
	sql, _ = sqlOf(opts)
	assert.Equal(t, `(NOT COALESCE(((("customer" @> $1::jsonb))), false))`, sql)

	// No linked record matches: no record does
	opts, _, err = run(url.Values{"filter[Customer.Country]": {"Mars"}}, 0)
	require.NoError(t, err)
	sql, _ = sqlOf(opts)
	assert.Equal(t, `("id" IS NULL)`, sql)

	for _, tc := range []struct {
		query   url.Values
		matches int
		want    string
	}{
		{url.Values{"filter[Customer.Country]": {"x"}}, maxLinkedMatches + 1, `matches more than 1000 records in the linked table of "Customer"`},
		{url.Values{"filter[Customer.Contry]": {"x"}}, 1, `No field named "Contry". Did you mean "Country"?`},
		{url.Values{"filter[Customer.Company][contains]": {"rec_1"}}, 1, `filters follow one link: "Company" in the linked records of "Customer" is a links field.`},
		{url.Values{"filter[Customer.Since][contains]": {"2026"}}, 1, `"Since" is a date field. You can use: eq, neq, gt`},
		{url.Values{"filter[Old.Country]": {"x"}}, 1, `the linked table of "Old" no longer exists.`},
		{url.Values{"sort": {"Customer country"}}, 1, `sort: "Customer country" is a lookup field.`},
	} {
		_, _, err := run(tc.query, tc.matches)
		require.Error(t, err, tc.query)
		assert.Contains(t, err.Error(), tc.want, tc.query)
	}
}

// G4: the expand budget is shared by every expanded field of the request
func TestTakeBudget(t *testing.T) {
	budget := 5
	assert.Equal(t, []int64{1, 2, 3}, takeBudget([]int64{1, 2, 3}, &budget))
	assert.Equal(t, []int64{4, 5}, takeBudget([]int64{4, 5, 6}, &budget), "only what is left")
	assert.Empty(t, takeBudget([]int64{7}, &budget), "spent: links keep only their id")
	assert.Equal(t, 0, budget)
}

// G30: a value that can't be read says what it must look like
func TestFilterValueHints(t *testing.T) {
	cases := map[string]string{
		`{"field": "Due", "value": "01/11/2026"}`:                     `"01/11/2026" is not a valid value for this field. Use a date as YYYY-MM-DD, like 2026-11-01.`,
		`{"field": "Ordered at", "op": "gte", "value": "2026-11-01"}`: `Use a date-time with a time zone (RFC 3339)`,
	}
	for tree, want := range cases {
		_, err := parseFilters(whereQuery(tree), aggFields(), testLinks)
		require.Error(t, err, tree)
		assert.Contains(t, err.Error(), want, tree)
	}
}

func TestWhereConditionLimit(t *testing.T) {
	conds := make([]string, maxConditions+1)
	for i := range conds {
		conds[i] = `{"field": "Amount", "value": 1}`
	}
	_, err := parseListOptions(whereQuery(`{"conditions": [`+strings.Join(conds, ",")+`]}`), whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 50 conditions")
}

// Formula values are calculated in the app, so the stored value can be out of date; the API leaves them out
func TestFormulaFieldsAreLeftOut(t *testing.T) {
	_, ok := newField("00000000-0000-0000-0000-0000000000f1", "Total", "total", "Formula", "", false, 0, nil)
	assert.False(t, ok)
	_, ok = newField("00000000-0000-0000-0000-0000000000f2", "Title", "title", "text", "", false, 0, nil)
	assert.True(t, ok)
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
	cursor := encodeCursor([]byte(`[50]`), a)
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
