// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// offsetCursor is a version 1 cursor, as handed out before keyset paging
func offsetCursor(offset int, hash string) string {
	b, _ := json.Marshal(map[string]interface{}{"v": 1, "o": offset, "h": hash})
	return base64.RawURLEncoding.EncodeToString(b)
}

func rawCursor(v string) string { return base64.RawURLEncoding.EncodeToString([]byte(v)) }

// keysetSQL is the list query for query, after the record with these sort values
func keysetSQL(t *testing.T, query url.Values, keys string) (string, []interface{}) {
	t.Helper()
	opts, err := parseListOptions(query, whereFields(), testLinks)
	require.NoError(t, err)
	query.Set("cursor", encodeCursor(json.RawMessage(keys), opts.hash))
	opts, err = parseListOptions(query, whereFields(), testLinks)
	require.NoError(t, err)
	sql, args, err := listSQL(`"s"."orders"`, opts)
	require.NoError(t, err)
	return sql, args
}

func TestKeysetClause(t *testing.T) {
	cases := []struct {
		name, sort, keys, where string
		args                    []interface{}
	}{
		{"id only: the primary key", "", `[1043]`, `t.id > $1`, []interface{}{"1043"}},
		{"id descending", "-id", `[1043]`, `t."id" < $1`, []interface{}{"1043"}},
		{"descending", "-Amount", `[120.00, 1043]`,
			`((t."amount" < $1) OR (t."amount" = $1 AND t.id > $2))`, []interface{}{"120.00", "1043"}},
		{"ascending: empty values come after every value", "Amount", `[120.00, 1043]`,
			`(((t."amount" > $1 OR t."amount" IS NULL)) OR (t."amount" = $1 AND t.id > $2))`, []interface{}{"120.00", "1043"}},
		{"ascending, bookmark empty: only empty values with a higher id", "Amount", `[null, 1043]`,
			`t."amount" IS NULL AND t.id > $1`, []interface{}{"1043"}},
		{"descending, bookmark empty: every value comes after", "-Amount", `[null, 1043]`,
			`((t."amount" IS NOT NULL) OR (t."amount" IS NULL AND t.id > $1))`, []interface{}{"1043"}},
		{"mixed directions", "-Amount,Name", `[5, "Asha", 7]`,
			`((t."amount" < $1) OR (t."amount" = $1 AND (t."title" > $2 OR t."title" IS NULL)) OR (t."amount" = $1 AND t."title" = $2 AND t.id > $3))`,
			[]interface{}{"5", "Asha", "7"}},
		{"checkbox and text values keep their type", "Paid,Name", `[false, "a\"b", 7]`,
			`(((t."paid" > $1 OR t."paid" IS NULL)) OR (t."paid" = $1 AND (t."title" > $2 OR t."title" IS NULL)) OR (t."paid" = $1 AND t."title" = $2 AND t.id > $3))`,
			[]interface{}{false, `a"b`, "7"}},
		{"keys after id are left out of the query and the cursor", "id,Name", `[7]`, `t."id" > $1`, []interface{}{"7"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := url.Values{}
			if tc.sort != "" {
				q.Set("sort", tc.sort)
			}
			sql, args := keysetSQL(t, q, tc.keys)
			where := sql[strings.Index(sql, " WHERE ")+len(" WHERE ") : strings.Index(sql, " ORDER BY")]
			assert.Equal(t, tc.where, where)
			assert.Equal(t, append(tc.args, 51), args, "the bookmark values, then limit+1; no offset")
			assert.NotContains(t, sql, "OFFSET")
		})
	}
}

func TestKeysetDurationComparesNumbers(t *testing.T) {
	sql, args := keysetSQL(t, url.Values{"sort": {"-Minutes"}}, `[90, 3]`)
	assert.Contains(t, sql, `::numeric END) < $1`, "duration text is compared as the number it sorts by")
	assert.Equal(t, []interface{}{"90", "3", 51}, args)
}

func TestKeysetWithFilters(t *testing.T) {
	sql, args := keysetSQL(t, url.Values{"filter[Status]": {"Open"}, "sort": {"-Amount"}}, `[120, 9]`)
	assert.Contains(t, sql, ` WHERE (("status" = $1)) AND ((t."amount" < $2) OR (t."amount" = $2 AND t.id > $3)) ORDER BY`,
		"the bookmark comes after the filters, with the next argument numbers")
	assert.Equal(t, []interface{}{"Open", "120", "9", 51}, args)
}

func TestListSQLKeepsTodaysOrder(t *testing.T) {
	opts, err := parseListOptions(url.Values{"sort": {"-Amount,Name"}}, whereFields(), testLinks)
	require.NoError(t, err)
	sql, args, err := listSQL(`"s"."orders"`, opts)
	require.NoError(t, err)
	// Postgres's defaults, written out: empty values last when ascending, first when descending
	assert.Equal(t, `SELECT row_to_json(t)::text, json_build_array(t."amount", t."title", t.id)::text FROM "s"."orders" AS t`+
		` ORDER BY t."amount" DESC NULLS FIRST, t."title" ASC NULLS LAST, t.id ASC NULLS LAST LIMIT $1`, sql)
	assert.Equal(t, []interface{}{51}, args)
}

func TestOldOffsetCursorStillWorks(t *testing.T) {
	opts, err := parseListOptions(url.Values{"filter[Status]": {"Open"}}, whereFields(), testLinks)
	require.NoError(t, err)
	q := url.Values{"filter[Status]": {"Open"}, "cursor": {offsetCursor(100, opts.hash)}}
	opts, err = parseListOptions(q, whereFields(), testLinks)
	require.NoError(t, err)
	assert.Equal(t, 100, opts.offset)
	assert.Nil(t, opts.after)
	sql, args, err := listSQL(`"s"."orders"`, opts)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(sql, " LIMIT $2 OFFSET $3"), sql)
	assert.Equal(t, []interface{}{"Open", 51, 100}, args)

	// From another filter: refused as before
	_, err = parseListOptions(url.Values{"cursor": {offsetCursor(100, opts.hash)}}, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different filter or sort")
}

func TestKeysetCursorChecks(t *testing.T) {
	opts, err := parseListOptions(url.Values{"sort": {"-Amount"}}, whereFields(), testLinks)
	require.NoError(t, err)
	h := opts.hash
	parse := func(cursor string) error {
		_, err := parseListOptions(url.Values{"sort": {"-Amount"}, "cursor": {cursor}}, whereFields(), testLinks)
		return err
	}
	require.NoError(t, parse(encodeCursor(json.RawMessage(`[1.5, 3]`), h)))
	require.NoError(t, parse(encodeCursor(json.RawMessage(`[null, 3]`), h)))

	notValid := map[string]string{
		"not base64":               "%%%",
		"not JSON":                 rawCursor("nope"),
		"unknown version":          rawCursor(`{"v":3,"k":[1,3],"h":"` + h + `"}`),
		"one value too few":        encodeCursor(json.RawMessage(`[3]`), h),
		"one value too many":       encodeCursor(json.RawMessage(`[1, 2, 3]`), h),
		"empty id":                 encodeCursor(json.RawMessage(`[1, null]`), h),
		"an object as a value":     encodeCursor(json.RawMessage(`[{"a": 1}, 3]`), h),
		"a list as a value":        encodeCursor(json.RawMessage(`[[1], 3]`), h),
		"values not a list":        encodeCursor(json.RawMessage(`{"a": 1}`), h),
		"version 2 with an offset": rawCursor(`{"v":2,"o":5,"k":[1,3],"h":"` + h + `"}`),
		"version 1 with values":    rawCursor(`{"v":1,"o":5,"k":[1,3],"h":"` + h + `"}`),
		"negative offset":          rawCursor(`{"v":1,"o":-1,"h":"` + h + `"}`),
	}
	for name, cursor := range notValid {
		err := parse(cursor)
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "cursor is not valid", name)
	}

	// Another sort: the hash says so before the number of values is checked
	err = parse(encodeCursor(json.RawMessage(`[3]`), "0000000000000000"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different filter or sort")
}

func TestKeysetErrorMapsBadValues(t *testing.T) {
	bad := &pq.Error{Code: "22P02"} // invalid_text_representation: "abc" for a number column
	withCursor := &listOptions{after: []interface{}{"abc", json.Number("3")}}
	err := keysetError(bad, withCursor)
	assert.Contains(t, err.Error(), "cursor is not valid")

	assert.Equal(t, error(bad), keysetError(bad, &listOptions{}), "without a cursor it's not the cursor's fault")
	assert.Equal(t, error(bad), keysetError(bad, &listOptions{offset: 50}), "nor with an old offset cursor")
	plain := errors.New("connection reset")
	assert.Equal(t, plain, keysetError(plain, withCursor))
	other := &pq.Error{Code: "42703"} // undefined_column
	assert.Equal(t, error(other), keysetError(other, withCursor))
}
