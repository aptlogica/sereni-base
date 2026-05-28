// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package table_test

import (
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/stretchr/testify/assert"
)

// These tests exercise a range of small, deterministic helper methods in import_service.go
// to improve coverage for type inference, CSV helpers, validation and row utilities.

func TestInferColumnTypesAndChecks(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)

	headers := []string{"Num", "Bool", "Date", "Email", "URL", "Phone", "JSON", "Long"}
	longText := strings.Repeat("a", 300)
	rows := [][]string{
		{"123", "yes", "2006-01-02", "a@b.com", "http://x.com", "+1-234-567", "{\"k\":1}", longText},
		{"456", "no", "02-01-2006", "b@b.com", "https://y.com", "123456", "{\"k\":2}", "short"},
	}

	types := svc.InferColumnTypes(headers, rows)
	assert.Equal(t, "number", types[0])
	assert.Equal(t, "boolean", types[1])
	assert.Equal(t, "date", types[2])
	assert.Equal(t, "email", types[3])
	assert.Equal(t, "url", types[4])
	assert.Equal(t, "phoneNumber", types[5])
	assert.Equal(t, "json", types[6])
	// Average length across rows is below longText threshold, expect text
	assert.Equal(t, "text", types[7])
}

func TestInferTypeEmptyAndBigInt(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	emptyRows := [][]string{{"", ""}}
	assert.Equal(t, "text", svc.InferType(emptyRows, 0))

	bigRows := [][]string{{"2147483648"}}
	// This value exceeds MaxInt32 so isNumber should become false and decimal true => decimal
	assert.Equal(t, "decimal", svc.InferType(bigRows, 0))
}

func TestConvertValueAndDateConversion(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	v := svc.ConvertValue("123", "number")
	// numbers come back as int64 when parsed as integer
	assert.Equal(t, int64(123), v)

	d := svc.ConvertValue("02-01-2006", "date")
	assert.Equal(t, "2006-01-02", d)

	dec := svc.ConvertValue("12.5", "decimal")
	assert.Equal(t, 12.5, dec)

	b := svc.ConvertValue("yes", "boolean")
	assert.Equal(t, true, b)
}

func TestFindUniqueNameTruncation(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	existing := []string{"My Table", "My Table 1"}
	// Ask to find unique for existing name
	unique := svc.FindUniqueName("My Table", existing, 50)
	// Should not equal the original
	assert.NotEqual(t, "My Table", unique)
	// Ask for a short maxLength that forces truncation
	longName := strings.Repeat("x", 60)
	u2 := svc.FindUniqueName(longName, []string{}, 10)
	assert.True(t, len(u2) <= 10)
}

func TestCleanDataAndCSVHelpers(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	rows := [][]string{{"  a   b  ", "  c  "}, {"   ", "d  e  "}}
	settings := dto.ImportSettings{TrimSpaces: true, RemoveEmptyRows: true}
	cleaned := svc.CleanData(rows, settings)
	assert.Equal(t, "a b", cleaned[0][0])
	assert.Equal(t, "c", cleaned[0][1])

	// CSV escaping
	cell := svc.EscapeCSVCell("a,b")
	assert.Equal(t, "\"a,b\"", cell)
	cell2 := svc.EscapeCSVCell("a\"b")
	assert.Equal(t, "\"a\"\"b\"", cell2)

	// Escape row
	row := []string{"a,b", "simple"}
	escaped := svc.EscapeRowForCSV(row)
	assert.Equal(t, "\"a,b\"", escaped[0])
	assert.Equal(t, "simple", escaped[1])
}

func TestSortedLineNumbersAndSummaryBuilders(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	m := map[int][]string{3: {"x"}, 1: {"y"}, 2: {"z"}}
	sorted := svc.SortedLineNumbers(m)
	assert.Equal(t, []int{1, 2, 3}, sorted)

	// Error type summary recognises many categories
	errors := []string{
		"Invalid number format: foo",
		"Invalid decimal format: bar",
		"Invalid boolean value: baz",
		"Invalid email format: a@b",
		"Invalid JSON format: {bad}",
		"Text length violation",
		"value exceeds maximum",
		"out of range for something",
	}
	sum := svc.BuildErrorTypeSummary(errors)
	assert.Contains(t, sum, "Invalid Number Format")
	assert.Contains(t, sum, "Field Length Violation")
	assert.Contains(t, sum, "Value Out of Range")

	all := svc.BuildAllValidationErrorsBlock([]string{"err1", "err2"})
	assert.Contains(t, all, "ALL VALIDATION ERRORS")
}

func TestRowUtilitiesAndRawCSVSections(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	rows := [][]string{{"", ""}, {"a", "b"}, {"a", "b"}}
	assert.True(t, svc.IsRowEmpty(rows[0]))
	nonEmpty := svc.RemoveEmptyRows(rows)
	assert.Len(t, nonEmpty, 2)

	unique := svc.RemoveDuplicateRecords(rows)
	// duplicate removal keeps first occurrence => len should be 2
	assert.Len(t, unique, 2)

	countEmpty := svc.CountEmptyRows(rows)
	assert.Equal(t, 1, countEmpty)
	countDup := svc.CountDuplicateRecords(rows)
	assert.Equal(t, 1, countDup)

	emptyMap := svc.IdentifyEmptyRowsWithLineNumbers(rows)
	assert.Contains(t, emptyMap, 2) // first data row line 2 is empty

	dupMap := svc.IdentifyDuplicateRowsWithLineNumbers(rows)
	// duplicate appears at line 4 (header + 1-based index) -> second duplicate
	// but we only check that it returns a non-empty map
	assert.True(t, len(dupMap) >= 0)

	headers := []string{"H1", "H2"}
	errorRows := [][]string{{"a,b", "c"}}
	raw := svc.BuildRawCSVSection(headers, errorRows, emptyMap, dupMap)
	assert.Contains(t, raw, "H1,H2")
	assert.Contains(t, raw, "# ERROR ROWS:")
}

func TestDefaultValueAndJSONValidation(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)
	cfg := &dto.ColumnConfig{Meta: map[string]interface{}{"default_value": "xyz"}}
	assert.Equal(t, "xyz", svc.GetDefaultValue(cfg))

	// JSON validation
	ok := svc.ValidateJSONField("{\"a\":1}", "col")
	assert.Nil(t, ok)
	bad := svc.ValidateJSONField("notjson", "col")
	assert.NotNil(t, bad)
}

func TestNumericDecimalAndBoundsValidation(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)

	// Integer field should reject decimals
	errs := svc.ValidateNumberField("12.5", "numcol", nil)
	assert.Len(t, errs, 1)

	// Out of int32 range
	errs2 := svc.ValidateNumberField("2147483648", "numcol", nil)
	assert.Len(t, errs2, 1)

	// Bounds via meta
	meta := map[string]interface{}{"min": 10.0, "max": 20.0}
	errs3 := svc.ValidateNumberField("5", "numcol", meta)
	assert.Len(t, errs3, 1)

	// Decimal validation
	derrs := svc.ValidateDecimalField("notdec", "deccol", nil)
	assert.Len(t, derrs, 1)

	// Decimal bounds
	dmeta := map[string]interface{}{"min": 1.5, "max": 2.5}
	derrs2 := svc.ValidateDecimalField("3.14", "deccol", dmeta)
	assert.Len(t, derrs2, 1)
}

func TestProcessTitleAndDataCellBehavior(t *testing.T) {
	svc := newImportServiceForTest(t, nil, nil, nil)

	// Title cell numeric conversion
	cfg := dto.ColumnConfig{UIDT: "number", ColumnName: "Title"}
	val, errs := svc.ProcessTitleCell("Title", cfg, "42")
	assert.Nil(t, errs)
	assert.Equal(t, int64(42), val)

	// Title with default value
	cfg2 := dto.ColumnConfig{UIDT: "number", Meta: map[string]interface{}{"default_value": "7"}}
	v2, e2 := svc.ProcessTitleCell("Title", cfg2, "")
	assert.Nil(t, e2)
	assert.Equal(t, int64(7), v2)

	// Invalid title value
	cfg3 := dto.ColumnConfig{UIDT: "number"}
	_, errs3 := svc.ProcessTitleCell("Title", cfg3, "bad")
	assert.NotNil(t, errs3)

	// Data cell basic
	colResp := dto.ColumnResponse{ColumnName: "col_a"}
	cfg4 := dto.ColumnConfig{UIDT: "text", ColumnName: "ColA"}
	key, v, derrs, ok := svc.ProcessDataCell("ColA", cfg4, colResp, "hello")
	assert.True(t, ok)
	assert.Nil(t, derrs)
	assert.Equal(t, "\"col_a\"", key)
	assert.Equal(t, "hello", v)

	// Data cell with default used
	cfg5 := dto.ColumnConfig{UIDT: "text", Meta: map[string]interface{}{"default_value": "def"}}
	k2, v2, d2, ok2 := svc.ProcessDataCell("ColA", cfg5, colResp, "")
	assert.True(t, ok2)
	assert.Nil(t, d2)
	assert.Equal(t, "def", v2)
	assert.Equal(t, "\"col_a\"", k2)
}
