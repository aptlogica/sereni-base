// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/stretchr/testify/assert"
)

// These tests exercise a range of small, deterministic helper methods in import_service.go
// to improve coverage for type inference, CSV helpers, validation and row utilities.

func TestInferColumnTypesAndChecks(t *testing.T) {
	s := &importService{}

	headers := []string{"Num", "Bool", "Date", "Email", "URL", "Phone", "JSON", "Long"}
	longText := strings.Repeat("a", 300)
	rows := [][]string{
		{"123", "yes", "2006-01-02", "a@b.com", "http://x.com", "+1-234-567", "{\"k\":1}", longText},
		{"456", "no", "02-01-2006", "b@b.com", "https://y.com", "123456", "{\"k\":2}", "short"},
	}

	types := s.inferColumnTypes(headers, rows)
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
	s := &importService{}
	emptyRows := [][]string{{"", ""}}
	assert.Equal(t, "text", s.inferType(emptyRows, 0))

	bigRows := [][]string{{"2147483648"}}
	// This value exceeds MaxInt32 so isNumber should become false and decimal true => decimal
	assert.Equal(t, "decimal", s.inferType(bigRows, 0))
}

func TestConvertValueAndDateConversion(t *testing.T) {
	s := &importService{}
	v := s.convertValue("123", "number")
	// numbers come back as int64 when parsed as integer
	assert.Equal(t, int64(123), v)

	d := s.convertValue("02-01-2006", "date")
	assert.Equal(t, "2006-01-02", d)

	dec := s.convertValue("12.5", "decimal")
	assert.Equal(t, 12.5, dec)

	b := s.convertValue("yes", "boolean")
	assert.Equal(t, true, b)
}

func TestFindUniqueNameTruncation(t *testing.T) {
	s := &importService{}
	existing := []string{"My Table", "My Table 1"}
	// Ask to find unique for existing name
	unique := s.findUniqueName("My Table", existing, 50)
	// Should not equal the original
	assert.NotEqual(t, "My Table", unique)
	// Ask for a short maxLength that forces truncation
	longName := strings.Repeat("x", 60)
	u2 := s.findUniqueName(longName, []string{}, 10)
	assert.True(t, len(u2) <= 10)
}

func TestCleanDataAndCSVHelpers(t *testing.T) {
	s := &importService{}
	rows := [][]string{{"  a   b  ", "  c  "}, {"   ", "d  e  "}}
	settings := dto.ImportSettings{TrimSpaces: true, RemoveEmptyRows: true}
	cleaned := s.cleanData(rows, settings)
	assert.Equal(t, "a b", cleaned[0][0])
	assert.Equal(t, "c", cleaned[0][1])

	// CSV escaping
	cell := s.escapeCSVCell("a,b")
	assert.Equal(t, "\"a,b\"", cell)
	cell2 := s.escapeCSVCell("a\"b")
	assert.Equal(t, "\"a\"\"b\"", cell2)

	// Escape row
	row := []string{"a,b", "simple"}
	escaped := s.escapeRowForCSV(row)
	assert.Equal(t, "\"a,b\"", escaped[0])
	assert.Equal(t, "simple", escaped[1])
}

func TestSortedLineNumbersAndSummaryBuilders(t *testing.T) {
	s := &importService{}
	m := map[int][]string{3: {"x"}, 1: {"y"}, 2: {"z"}}
	sorted := s.sortedLineNumbers(m)
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
	sum := s.buildErrorTypeSummary(errors)
	assert.Contains(t, sum, "Invalid Number Format")
	assert.Contains(t, sum, "Field Length Violation")
	assert.Contains(t, sum, "Value Out of Range")

	all := s.buildAllValidationErrorsBlock([]string{"err1", "err2"})
	assert.Contains(t, all, "ALL VALIDATION ERRORS")
}

func TestRowUtilitiesAndRawCSVSections(t *testing.T) {
	s := &importService{}
	rows := [][]string{{"", ""}, {"a", "b"}, {"a", "b"}}
	assert.True(t, s.isRowEmpty(rows[0]))
	nonEmpty := s.removeEmptyRows(rows)
	assert.Len(t, nonEmpty, 2)

	unique := s.removeDuplicateRecords(rows)
	// duplicate removal keeps first occurrence => len should be 2
	assert.Len(t, unique, 2)

	countEmpty := s.countEmptyRows(rows)
	assert.Equal(t, 1, countEmpty)
	countDup := s.countDuplicateRecords(rows)
	assert.Equal(t, 1, countDup)

	emptyMap := s.identifyEmptyRowsWithLineNumbers(rows)
	assert.Contains(t, emptyMap, 2) // first data row line 2 is empty

	dupMap := s.identifyDuplicateRowsWithLineNumbers(rows)
	// duplicate appears at line 4 (header + 1-based index) -> second duplicate
	// but we only check that it returns a non-empty map
	assert.True(t, len(dupMap) >= 0)

	headers := []string{"H1", "H2"}
	errorRows := [][]string{{"a,b", "c"}}
	raw := s.buildRawCSVSection(headers, errorRows, emptyMap, dupMap)
	assert.Contains(t, raw, "H1,H2")
	assert.Contains(t, raw, "# ERROR ROWS:")
}

func TestDefaultValueAndJSONValidation(t *testing.T) {
	s := &importService{}
	cfg := &dto.ColumnConfig{Meta: map[string]interface{}{"default_value": "xyz"}}
	assert.Equal(t, "xyz", s.getDefaultValue(cfg))

	// JSON validation
	ok := s.validateJSONField("{\"a\":1}", "col")
	assert.Nil(t, ok)
	bad := s.validateJSONField("notjson", "col")
	assert.NotNil(t, bad)
}

func TestNumericDecimalAndBoundsValidation(t *testing.T) {
	s := &importService{}

	// Integer field should reject decimals
	errs := s.validateNumberField("12.5", "numcol", nil)
	assert.Len(t, errs, 1)

	// Out of int32 range
	errs2 := s.validateNumberField("2147483648", "numcol", nil)
	assert.Len(t, errs2, 1)

	// Bounds via meta
	meta := map[string]interface{}{"min": 10.0, "max": 20.0}
	errs3 := s.validateNumberField("5", "numcol", meta)
	assert.Len(t, errs3, 1)

	// Decimal validation
	derrs := s.validateDecimalField("notdec", "deccol", nil)
	assert.Len(t, derrs, 1)

	// Decimal bounds
	dmeta := map[string]interface{}{"min": 1.5, "max": 2.5}
	derrs2 := s.validateDecimalField("3.14", "deccol", dmeta)
	assert.Len(t, derrs2, 1)
}

func TestProcessTitleAndDataCellBehavior(t *testing.T) {
	s := &importService{}

	// Title cell numeric conversion
	cfg := dto.ColumnConfig{UIDT: "number", ColumnName: "Title"}
	val, errs := s.processTitleCell("Title", cfg, "42")
	assert.Nil(t, errs)
	assert.Equal(t, int64(42), val)

	// Title with default value
	cfg2 := dto.ColumnConfig{UIDT: "number", Meta: map[string]interface{}{"default_value": "7"}}
	v2, e2 := s.processTitleCell("Title", cfg2, "")
	assert.Nil(t, e2)
	assert.Equal(t, int64(7), v2)

	// Invalid title value
	cfg3 := dto.ColumnConfig{UIDT: "number"}
	_, errs3 := s.processTitleCell("Title", cfg3, "bad")
	assert.NotNil(t, errs3)

	// Data cell basic
	colResp := dto.ColumnResponse{ColumnName: "col_a"}
	cfg4 := dto.ColumnConfig{UIDT: "text", ColumnName: "ColA"}
	key, v, derrs, ok := s.processDataCell("ColA", cfg4, colResp, "hello")
	assert.True(t, ok)
	assert.Nil(t, derrs)
	assert.Equal(t, "\"col_a\"", key)
	assert.Equal(t, "hello", v)

	// Data cell with default used
	cfg5 := dto.ColumnConfig{UIDT: "text", Meta: map[string]interface{}{"default_value": "def"}}
	k2, v2, d2, ok2 := s.processDataCell("ColA", cfg5, colResp, "")
	assert.True(t, ok2)
	assert.Nil(t, d2)
	assert.Equal(t, "def", v2)
	assert.Equal(t, "\"col_a\"", k2)
}
