// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package table_test

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// firstRecordColumns is what the bulk insert writes for every record of a batch: the keys of the first record
func firstRecordColumns(records []map[string]interface{}) []string {
	var cols []string
	for k := range records[0] {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

func keysOf(m map[string]interface{}) []string {
	var cols []string
	for k := range m {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

// An empty cell is left out of its record, so a column that is empty in the first row has no key there. The bulk
// insert writes the columns of the first record, so the values of that column in later rows were lost.
func TestImportBatchKeepsAValueThatTheFirstRowDoesNotHave(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	params := dto.BuildRecordsWithConfigAndErrorsParams{
		Headers: []string{"Title", "Name", "Address"},
		Req:     dto.CreateTableRequest{CreatedBy: "user"},
		ColumnConfigs: []dto.ColumnConfig{
			{ColumnName: "Title", UIDT: "text"}, {ColumnName: "Name", UIDT: "text"}, {ColumnName: "Address", UIDT: "text"},
		},
		ColumnMap: map[int]dto.ColumnResponse{0: {ColumnName: "title"}, 1: {ColumnName: "name"}, 2: {ColumnName: "address"}},
		DataRows: [][]string{
			{"", "", "pune"},
			{"aa", "aaa", ""},
			{"bb", "bbb", ""},
		},
	}
	records, errorRows, _ := svc.BuildRecordsWithConfigAndErrors(params, logger.Get())
	require.Len(t, records, 3)
	require.Empty(t, errorRows)

	// the cause: the first record has no title or name, so the columns written for the batch have neither
	assert.NotContains(t, firstRecordColumns(records), `"title"`)
	assert.NotContains(t, firstRecordColumns(records), `"name"`)

	aligned := services.AlignBulkRecords(records)
	require.Len(t, aligned, 3)
	// every record has the same keys, so the first one names every column
	for i, rec := range aligned {
		assert.Equal(t, keysOf(aligned[0]), keysOf(rec), "record %d", i)
	}
	assert.Contains(t, firstRecordColumns(aligned), `"title"`)
	assert.Contains(t, firstRecordColumns(aligned), `"name"`)
	assert.Contains(t, firstRecordColumns(aligned), `"address"`)

	// the values are all there; a cell that was empty is nil (NULL)
	assert.Nil(t, aligned[0][`"title"`])
	assert.Nil(t, aligned[0][`"name"`])
	assert.Equal(t, "pune", aligned[0][`"address"`])
	assert.Equal(t, "aa", aligned[1][`"title"`])
	assert.Equal(t, "aaa", aligned[1][`"name"`])
	assert.Nil(t, aligned[1][`"address"`])
	assert.Equal(t, "bb", aligned[2][`"title"`])
	assert.Equal(t, "bbb", aligned[2][`"name"`])
}

func TestAlignBulkRecords(t *testing.T) {
	t.Run("records with the same keys are unchanged", func(t *testing.T) {
		in := []map[string]interface{}{{"a": 1, "b": "x"}, {"a": 2, "b": "y"}}
		assert.Equal(t, in, services.AlignBulkRecords(in))
	})

	t.Run("a missing key becomes nil, in every record", func(t *testing.T) {
		in := []map[string]interface{}{{"a": 1}, {"b": 2}, {"a": 3, "c": 4}}
		out := services.AlignBulkRecords(in)
		assert.Equal(t, []map[string]interface{}{
			{"a": 1, "b": nil, "c": nil},
			{"a": nil, "b": 2, "c": nil},
			{"a": 3, "b": nil, "c": 4},
		}, out)
	})

	t.Run("the records given are not changed", func(t *testing.T) {
		in := []map[string]interface{}{{"a": 1}, {"b": 2}}
		services.AlignBulkRecords(in)
		assert.Equal(t, []map[string]interface{}{{"a": 1}, {"b": 2}}, in)
	})

	t.Run("a value that is false, zero or empty text is kept as it is", func(t *testing.T) {
		in := []map[string]interface{}{{"a": false, "b": 0, "c": ""}, {"a": true}}
		out := services.AlignBulkRecords(in)
		assert.Equal(t, false, out[0]["a"])
		assert.Equal(t, 0, out[0]["b"])
		assert.Equal(t, "", out[0]["c"])
		assert.Nil(t, out[1]["b"])
	})

	t.Run("no records and one record", func(t *testing.T) {
		assert.Empty(t, services.AlignBulkRecords(nil))
		assert.Equal(t, []map[string]interface{}{{"a": 1}}, services.AlignBulkRecords([]map[string]interface{}{{"a": 1}}))
	})
}

// A plain database error gets the reason the database gave, so an import can tell the user why a row was refused
func TestWithDatabaseReason(t *testing.T) {
	pgErr := &pq.Error{Code: "22003", Message: `value "100021000410003" is out of range for type integer`, Column: "amount"}
	wrapped := fmt.Errorf(`failed to insert record into table "public"."customer_churn_1795157569_5358": %w`, pgErr)

	got := services.WithDatabaseReason(app_errors.DatabaseError, wrapped)
	require.ErrorIs(t, got, app_errors.DatabaseError, "it stays a database error")
	assert.Equal(t, `database error: value "100021000410003" is out of range for type integer (column amount)`, got.Error())
	assert.NotContains(t, got.Error(), "customer_churn", "the table name of the full error is not given")

	t.Run("no column", func(t *testing.T) {
		got := services.WithDatabaseReason(app_errors.DatabaseError, &pq.Error{Code: "22007", Message: `invalid input syntax for type date: ""`})
		assert.Equal(t, `database error: invalid input syntax for type date: ""`, got.Error())
	})
	t.Run("another error is returned as it was", func(t *testing.T) {
		assert.Equal(t, app_errors.UniqueKeyViolation, services.WithDatabaseReason(app_errors.UniqueKeyViolation, wrapped))
	})
	t.Run("an error that is not from the database has no reason to add", func(t *testing.T) {
		assert.Equal(t, app_errors.DatabaseError, services.WithDatabaseReason(app_errors.DatabaseError, errors.New("boom")))
	})
}
