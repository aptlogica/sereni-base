// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// filterParams returns n filter[Amount] values, the way ?filter[Amount]=0&filter[Amount]=1… arrives
func filterParams(n int) url.Values {
	q := url.Values{}
	for i := 0; i < n; i++ {
		q.Add("filter[Amount]", fmt.Sprint(i))
	}
	return q
}

// whereConditions returns a where tree with n conditions
func whereConditions(n int) string {
	conds := make([]string, n)
	for i := range conds {
		conds[i] = `{"field": "Amount", "value": 1}`
	}
	return `{"conditions": [` + strings.Join(conds, ",") + `]}`
}

func TestConditionLimitCountsFilterParams(t *testing.T) {
	_, err := parseListOptions(filterParams(maxConditions), whereFields(), testLinks)
	require.NoError(t, err, "exactly the limit is allowed")

	_, err = parseListOptions(filterParams(maxConditions+1), whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 50 conditions across filter[...] and where.")
}

// filter[...] and where share one limit, so splitting conditions between them doesn't get around it
func TestConditionLimitIsSharedWithWhere(t *testing.T) {
	q := filterParams(30)
	q.Set("where", whereConditions(20))
	_, err := parseListOptions(q, whereFields(), testLinks)
	require.NoError(t, err, "30 + 20 = 50 is allowed")

	q.Set("where", whereConditions(21))
	_, err = parseListOptions(q, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 50 conditions")
}

func TestInValueLimit(t *testing.T) {
	values := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprint(i)
		}
		return out
	}
	_, err := parseListOptions(url.Values{"filter[Amount][in]": {strings.Join(values(maxInValues), ",")}}, whereFields(), testLinks)
	require.NoError(t, err)

	_, err = parseListOptions(url.Values{"filter[Amount][in]": {strings.Join(values(maxInValues+1), ",")}}, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filter[Amount][in]: in takes at most 100 values.")

	// The same limit applies to an in list in the where tree
	list := `["` + strings.Join(values(maxInValues+1), `","`) + `"]`
	_, err = parseListOptions(whereQuery(`{"field": "Status", "op": "in", "value": `+list+`}`), whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "in takes at most 100 values.")
}

// Fetching records by a list of IDs may ask for a full page of them
func TestIDInLimitIsOnePage(t *testing.T) {
	ids := func(n int) string {
		out := make([]string, n)
		for i := range out {
			out[i] = EncodeRecordID(int64(i + 1))
		}
		return strings.Join(out, ",")
	}
	_, err := parseListOptions(url.Values{"filter[id][in]": {ids(maxLimit)}}, whereFields(), testLinks)
	require.NoError(t, err)

	_, err = parseListOptions(url.Values{"filter[id][in]": {ids(maxLimit + 1)}}, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "in takes at most 200 values.")
}

// A formula field is left out of the API, but naming one says why
func TestFormulaReferenceSaysWhy(t *testing.T) {
	fields := whereFields()
	formulas := &hiddenFormulas{byID: map[string]string{"00000000-0000-0000-0000-0000000000f1": "Profit"}, byName: map[string]bool{"Profit": true}}
	for _, f := range fields {
		f.formulas = formulas
	}
	for _, q := range []url.Values{
		{"filter[Profit]": {"1"}},
		{"sort": {"-Profit"}},
		{"select": {"Profit"}},
		{"filter[fld_000000000000000000000000000000f1]": {"1"}},
	} {
		_, err := parseListOptions(q, fields, testLinks)
		require.Error(t, err, q)
		assert.Contains(t, err.Error(), `"Profit" is a formula field. Formula fields aren't in the API yet`, q)
	}

	// Names that aren't formulas still get the usual error and suggestion
	_, err := parseListOptions(url.Values{"sort": {"Statu"}}, fields, testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `No field named "Statu".`)
}

func TestSortTermLimit(t *testing.T) {
	five := "Status,-Amount,created_at,updated_at,id"
	opts, err := parseListOptions(url.Values{"sort": {five}}, whereFields(), testLinks)
	require.NoError(t, err)
	assert.Len(t, opts.sorts, 5)

	// The id tie-breaker added at the end doesn't count
	opts, err = parseListOptions(url.Values{"sort": {"Status,-Amount,created_at,updated_at,Name"}}, whereFields(), testLinks)
	require.NoError(t, err)
	assert.Len(t, opts.sorts, 6)

	_, err = parseListOptions(url.Values{"sort": {five + ",Name"}}, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sort: at most 5 fields.")
}

func TestSortFieldOnceAndNothingAfterID(t *testing.T) {
	// G10: a field listed twice, by name, ID or direction, is refused
	for _, sort := range []string{"Status,Status", "Status,-Status", "Status,fld_00000000000000000000000000000002", "id,-id"} {
		_, err := parseListOptions(url.Values{"sort": {sort}}, whereFields(), testLinks)
		require.Error(t, err, sort)
		assert.Contains(t, err.Error(), "is listed twice.", sort)
	}

	// G25: fields after id can't change the order, so they are left out; they are still checked
	opts, err := parseListOptions(url.Values{"sort": {"-Amount,id,Name"}}, whereFields(), testLinks)
	require.NoError(t, err)
	keys := []string{}
	for _, s := range opts.sorts {
		keys = append(keys, s.key)
	}
	assert.Equal(t, []string{"00000000-0000-0000-0000-000000000003", "id"}, keys)
	other, _ := parseListOptions(url.Values{"sort": {"-Amount,id"}}, whereFields(), testLinks)
	assert.Equal(t, other.hash, opts.hash, "the same order, so the same cursor")
	_, err = parseListOptions(url.Values{"sort": {"id,Nmae"}}, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `No field named "Nmae"`)
}
