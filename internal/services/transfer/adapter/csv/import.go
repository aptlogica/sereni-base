// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package csv

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	stdcsv "encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/internal/ziputil"
)

// Import reads a zip of delimited files into a model: one table per .csv file. The delimiter of each file
// is found from its header line. The first row holds the field titles. A CSV carries no types, so each field gets
// the narrowest type that holds all of its values (see inferType); a field whose cells are all paths of files in the
// zip's attachments/ folder becomes an attachment field. The zip is the file an export of this format writes, but any
// zip of delimited files works. The input must be an io.ReaderAt (an uploaded file is) and stay open until the
// import is done, because the attachment files are read from it.
func (csvAdapter) Import(_ context.Context, in io.Reader, size int64, opt adapter.ImportOptions) (*adapter.ImportResult, error) {
	limit := opt.MaxBytes
	if limit <= 0 {
		limit = adapter.DefaultMaxImportBytes
	}
	ra, ok := in.(io.ReaderAt)
	if !ok {
		return nil, errors.New("the file could not be read")
	}
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, errors.New("the file is not a valid zip")
	}

	var tableFiles []*zip.File
	zipFiles := map[string]*zip.File{}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		zipFiles[f.Name] = f
		if isTableFile(f.Name) {
			tableFiles = append(tableFiles, f)
		}
	}
	if len(tableFiles) == 0 {
		return nil, errors.New("the zip has no .csv file")
	}
	sort.SliceStable(tableFiles, func(i, j int) bool { return tableFiles[i].Name < tableFiles[j].Name })

	model := &dto.BaseExportFile{
		Base:   dto.BaseExportBase{Title: baseTitleFromFilename(opt.Filename)},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{}},
		Data:   &dto.BaseExportData{RowCounts: map[string]int{}, Tables: map[string]dto.BaseExportTableRow{}},
	}
	refs := newAttachmentIndex(model.Data, zipFiles)
	usedKeys := map[string]bool{}
	source := &dto.ImportSource{Rows: map[string]dto.ImportRowSource{}}
	for i, f := range tableFiles {
		if int64(f.UncompressedSize64) > limit {
			return nil, fmt.Errorf("%s in the zip is too large", f.Name)
		}
		records, lines, err := readRecords(f, limit)
		if err != nil {
			return nil, fmt.Errorf("%s could not be read: %v", f.Name, err)
		}
		table, rows := buildTable(i, f.Name, records, lines, usedKeys, refs, source)
		model.Schema.Tables = append(model.Schema.Tables, table)
		model.Data.Tables[table.Key] = dto.BaseExportTableRow{Rows: rows}
		model.Data.RowCounts[table.Key] = len(rows)
	}
	model.Data.Attachments = refs.files
	var attachments *dto.Attachments
	if len(refs.files) > 0 {
		attachments = ziputil.NewZipSource(zr)
	}
	return &adapter.ImportResult{File: model, Attachments: attachments, Source: source}, nil
}

// isTableFile reports whether a zip entry is a table: a .csv that is not inside attachments/
func isTableFile(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, AttachmentDir) || strings.HasPrefix(path.Base(name), ".") || strings.HasPrefix(lower, "__macosx/") {
		return false
	}
	return strings.HasSuffix(lower, ".csv")
}

// baseTitleFromFilename names the new base after the uploaded file: "My Base.csv.zip" gives "My Base"
func baseTitleFromFilename(filename string) string {
	name := path.Base(strings.ReplaceAll(filename, "\\", "/"))
	lower := strings.ToLower(name)
	for _, suffix := range []string{".csv.zip", ".zip"} {
		if strings.HasSuffix(lower, suffix) {
			name = name[:len(name)-len(suffix)]
			break
		}
	}
	if name = strings.TrimSpace(name); name == "" || name == "." {
		return "Imported CSV"
	}
	return name
}

var delimiterCandidates = []rune{',', ';', DelimiterTab, DelimiterPipe}

// detectDelimiter finds the delimiter of a file from its header line: the candidate that appears most (a comma, semicolon,
// tab or pipe, all in a .csv file). A header with none of them is a single column; the comma is then used.
func detectDelimiter(name string, header []byte) rune {
	best, bestCount := ',', 0
	for _, c := range delimiterCandidates {
		if n := bytes.Count(header, []byte(string(c))); n > bestCount {
			best, bestCount = c, n
		}
	}
	return best
}

// readRecords reads every record of a delimited file, with the line each one starts on (the header is line 1). A
// byte-order mark is accepted and dropped, a ragged row is allowed (a short row is padded later), and a bare quote
// inside a value is tolerated.
func readRecords(f *zip.File, limit int64) ([][]string, []int, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, nil, err
	}
	defer rc.Close()
	br := bufio.NewReader(io.LimitReader(rc, limit+1))
	if bom, err := br.Peek(3); err == nil && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
	}
	first, err := br.Peek(4096)
	if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
		return nil, nil, err
	}
	header := first
	if i := bytes.IndexAny(first, "\r\n"); i >= 0 {
		header = first[:i]
	}
	r := stdcsv.NewReader(br)
	r.Comma = detectDelimiter(f.Name, header)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	var records [][]string
	var lines []int
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		line, _ := r.FieldPos(0)
		records = append(records, record)
		lines = append(lines, line)
	}
	return records, lines, nil
}

// ---------------------------------------------------------------------------
// Tables and fields
// ---------------------------------------------------------------------------

var (
	nonKeyChars = regexp.MustCompile(`[^a-z0-9]+`)
	// separators of the files in one attachment cell: the joiner of an export (a pipe, or a semicolon with the pipe
	// delimiter)
	joinerChars = regexp.MustCompile(`[|;]`)
	// system field titles, which a field of a file must not have (the same rule the importer of one CSV applies)
	systemTitles = map[string]bool{"id": true, "created time": true, "last modified time": true, "created by": true, "last modified by": true}
)

const maxKeyLength = 63

func uniqueKey(prefix, text string, used map[string]bool) string {
	base := nonKeyChars.ReplaceAllString(strings.ToLower(text), "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "x"
	}
	key := prefix + base
	if len(key) > maxKeyLength-4 {
		key = key[:maxKeyLength-4]
	}
	unique := key
	for n := 2; used[unique]; n++ {
		unique = fmt.Sprintf("%s_%d", key, n)
	}
	used[unique] = true
	return unique
}

// fieldTitles cleans the header: empty titles are named "Column N", a title that is a system field's gets " (1)",
// and a repeated title is numbered, because a table cannot have two fields with one title
func fieldTitles(header []string) []string {
	out := make([]string, len(header))
	seen := map[string]int{}
	for i, h := range header {
		title := strings.TrimSpace(h)
		if title == "" {
			title = fmt.Sprintf("Column %d", i+1)
		}
		for n := 1; systemTitles[strings.ToLower(title)]; n++ {
			title = fmt.Sprintf("%s (%d)", strings.TrimSpace(h), n)
		}
		key := strings.ToLower(title)
		if n := seen[key]; n > 0 {
			seen[key]++
			title = fmt.Sprintf("%s (%d)", title, n+1)
			for seen[strings.ToLower(title)] > 0 {
				title += "_"
			}
		}
		seen[strings.ToLower(title)]++
		out[i] = title
	}
	return out
}

// buildTable turns the records of one file into a table and its rows
func buildTable(index int, fileName string, records [][]string, lines []int, usedKeys map[string]bool, refs *attachmentIndex, source *dto.ImportSource) (dto.BaseExportTable, []dto.BaseExportRow) {
	in := tableInput{index: index, fileName: fileName, title: uniqueTableTitle(index, fileName, usedKeys), records: records, lines: lines}
	return buildTitledTable(in, usedKeys, refs, source)
}

// tableInput is what one table is built from: the place of the file (or sheet) among the others, its name, the title
// chosen for the table, and its records with the line of the file each one came from
type tableInput struct {
	index    int
	fileName string
	title    string
	records  [][]string
	lines    []int
}

// buildTitledTable is buildTable for a caller that has already chosen the title of the table
func buildTitledTable(in tableInput, usedKeys map[string]bool, refs *attachmentIndex, source *dto.ImportSource) (dto.BaseExportTable, []dto.BaseExportRow) {
	index, fileName, title, records, lines := in.index, in.fileName, in.title, in.records, in.lines
	table := dto.BaseExportTable{
		Key:        uniqueKey("t_", title, usedKeys),
		Title:      title,
		OrderIndex: float64(index + 1),
		Fields:     []dto.BaseExportField{},
	}
	if len(records) == 0 {
		records = [][]string{{"Column 1"}}
		lines = []int{1}
	}
	header := records[0]
	titles := fieldTitles(header)

	kept, keptLines := keepRows(records[1:], lines, len(titles), source)
	cols := buildFields(&table, titles, kept, refs, usedKeys)
	origin := rowOrigin{index: index, fileName: fileName, tableTitle: table.Title, header: header}
	rows := buildRows(origin, kept, keptLines, cols, refs, source)
	return table, rows
}

// uniqueTableTitle names a table after its file; two files with one name (in different folders) still make two tables
// with different titles
func uniqueTableTitle(index int, fileName string, usedKeys map[string]bool) string {
	title := strings.TrimSpace(strings.TrimSuffix(path.Base(fileName), path.Ext(fileName)))
	if title == "" {
		title = fmt.Sprintf("Table %d", index+1)
	}
	for n, base := 2, title; usedKeys["title:"+strings.ToLower(title)]; n++ {
		title = fmt.Sprintf("%s (%d)", base, n)
	}
	usedKeys["title:"+strings.ToLower(title)] = true
	return title
}

// isEmptyRecord reports whether every cell of a record is blank
func isEmptyRecord(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// rowSignature identifies a row by its trimmed cells, to find rows that repeat an earlier row
func rowSignature(row []string) string {
	var sig strings.Builder
	for _, c := range row {
		sig.WriteString(strings.TrimSpace(c))
		sig.WriteByte(0x1f)
	}
	return sig.String()
}

// keepRows returns the rows without the empty ones, padded to the header's width; keptLines are their lines in the
// file. Rows that repeat an earlier row are counted for the report and still imported.
func keepRows(body [][]string, lines []int, width int, source *dto.ImportSource) ([][]string, []int) {
	var kept [][]string
	var keptLines []int
	seenRows := map[string]bool{}
	for i, rec := range body {
		if isEmptyRecord(rec) {
			source.EmptyRows++
			continue
		}
		row := make([]string, width)
		copy(row, rec)
		sig := rowSignature(row)
		if seenRows[sig] {
			source.DuplicateRows++
		}
		seenRows[sig] = true
		kept = append(kept, row)
		keptLines = append(keptLines, lines[i+1])
	}
	return kept, keptLines
}

// columnSet holds the field key and the inferred type of every column
type columnSet struct {
	keys  []string
	types []columnType
}

// rowOrigin describes the file and table the rows of buildRows come from
type rowOrigin struct {
	index      int
	fileName   string
	tableTitle string
	header     []string
}

// buildFields infers the type of every column, adds the fields to the table and returns their keys and types
func buildFields(table *dto.BaseExportTable, titles []string, kept [][]string, refs *attachmentIndex, usedKeys map[string]bool) columnSet {
	fieldKeys := make([]string, len(titles))
	types := make([]columnType, len(titles))
	for c := range titles {
		column := make([]string, len(kept))
		for r := range kept {
			column[r] = strings.TrimSpace(kept[r][c])
		}
		types[c] = inferType(column, refs)
		fieldKeys[c] = uniqueKey("f_", table.Key+"_"+titles[c], usedKeys)
		table.Fields = append(table.Fields, dto.BaseExportField{
			Key: fieldKeys[c], Title: titles[c], UIDT: types[c].uidt, OrderIndex: float64(c + 1),
		})
	}
	return columnSet{keys: fieldKeys, types: types}
}

// rowValues converts the cells of one row; an empty cell has no value, whatever the type: it is left out of the row
// and stored as NULL. Without this a date column got "" (which the database refuses) and a true / false column got false.
func rowValues(rec []string, cols columnSet, refs *attachmentIndex) map[string]interface{} {
	values := map[string]interface{}{}
	for c := range cols.keys {
		cell := strings.TrimSpace(rec[c])
		if cell == "" {
			continue
		}
		if v, ok := cols.types[c].convert(cell, refs); ok {
			values[cols.keys[c]] = v
		}
	}
	return values
}

// buildRows converts the kept records into rows and records where each one came from
func buildRows(origin rowOrigin, kept [][]string, keptLines []int, cols columnSet, refs *attachmentIndex, source *dto.ImportSource) []dto.BaseExportRow {
	rows := make([]dto.BaseExportRow, 0, len(kept))
	for r, rec := range kept {
		values := rowValues(rec, cols, refs)
		rid := fmt.Sprintf("r_%d_%d", origin.index+1, r+1)
		source.Rows[rid] = dto.ImportRowSource{File: origin.fileName, Table: origin.tableTitle, Line: keptLines[r], Headers: origin.header, Cells: rec}
		rows = append(rows, dto.BaseExportRow{RID: rid, Values: values})
	}
	return rows
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type columnType struct {
	uidt    string
	convert func(cell string, refs *attachmentIndex) (interface{}, bool)
}

var (
	integerPattern  = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,14})$`)
	decimalPattern  = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,14})\.[0-9]{1,15}$`)
	datePattern     = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	datetimePattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}[T ][0-9]{2}:[0-9]{2}(:[0-9]{2}(\.[0-9]+)?)?(Z|[+-][0-9]{2}:?[0-9]{2})?$`)
)

const longTextLength = 255

// inferType gives a column the narrowest type that holds every non-empty cell, and the way to convert a cell:
//
//	all cells are paths of files in attachments/   attachment
//	whole numbers (no leading zero) that fit a 32 bit integer   number
//	numbers with a decimal point, or whole numbers that are too big for a number field (up to 15 digits)   decimal
//	true / false (any case)                       boolean
//	YYYY-MM-DD                                    date
//	ISO 8601 date and time                        datetime
//	anything else                                 text, or longText when a cell has a line break or is long
//
// An empty column is text. A cell that is empty is left out of its row, whatever the type (see buildTable).
func inferType(cells []string, refs *attachmentIndex) columnType {
	if refs != nil && allCells(cells, refs.isFileCell) {
		return attachmentColumnType()
	}
	if t, ok := scalarColumnType(cells); ok {
		return t
	}
	return textColumnType(cells)
}

// allCells reports whether at least one cell is non-empty and every non-empty cell satisfies ok
func allCells(cells []string, ok func(string) bool) bool {
	seen := false
	for _, c := range cells {
		if c == "" {
			continue
		}
		seen = true
		if !ok(c) {
			return false
		}
	}
	return seen
}

func attachmentColumnType() columnType {
	return columnType{"attachment", func(cell string, r *attachmentIndex) (interface{}, bool) {
		// the cell is an array of refs, as in a file read from JSON
		var list []interface{}
		for _, ref := range r.refsOf(cell) {
			list = append(list, ref)
		}
		return list, len(list) > 0
	}}
}

func isIntegerOrDecimal(c string) bool {
	return integerPattern.MatchString(c) || decimalPattern.MatchString(c)
}

func isBoolean(c string) bool {
	l := strings.ToLower(c)
	return l == "true" || l == "false"
}

func isDate(c string) bool { return datePattern.MatchString(c) && validDate(c) }

func isDateTime(c string) bool { return datetimePattern.MatchString(c) && validDateTime(c) }

func parseFloatCell(cell string, _ *attachmentIndex) (interface{}, bool) {
	f, err := strconv.ParseFloat(cell, 64)
	return f, err == nil
}

func verbatimCell(cell string, _ *attachmentIndex) (interface{}, bool) { return cell, true }

// scalarColumnType returns the number, decimal, boolean, date or datetime type when every non-empty cell fits it
func scalarColumnType(cells []string) (columnType, bool) {
	switch {
	case allCells(cells, isInt32):
		// the number field is a 32 bit integer: a whole number that does not fit is a decimal (a 64 bit value in
		// PostgreSQL, exact up to 15 digits) instead of a value the database refuses
		// a number, as in a file read from JSON (up to 15 digits is exact)
		return columnType{"number", parseFloatCell}, true
	case allCells(cells, isIntegerOrDecimal):
		return columnType{"decimal", parseFloatCell}, true
	case allCells(cells, isBoolean):
		return columnType{"boolean", func(cell string, _ *attachmentIndex) (interface{}, bool) {
			return strings.EqualFold(cell, "true"), true
		}}, true
	case allCells(cells, isDate):
		return columnType{"date", verbatimCell}, true
	case allCells(cells, isDateTime):
		return columnType{"datetime", verbatimCell}, true
	}
	return columnType{}, false
}

// textColumnType is text, or longText when a cell has a line break or is long
func textColumnType(cells []string) columnType {
	uidt := "text"
	for _, c := range cells {
		if strings.ContainsAny(c, "\r\n") || len([]rune(c)) > longTextLength {
			uidt = "longText"
			break
		}
	}
	return columnType{uidt, func(cell string, _ *attachmentIndex) (interface{}, bool) { return cell, cell != "" }}
}

// isInt32 reports whether a cell is a whole number that fits the integer type of a number field
func isInt32(c string) bool {
	if !integerPattern.MatchString(c) {
		return false
	}
	n, err := strconv.ParseInt(c, 10, 64)
	return err == nil && n >= math.MinInt32 && n <= math.MaxInt32
}

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func validDateTime(s string) bool {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02T15:04:05Z0700", "2006-01-02T15:04:05.999999999Z0700"} {
		if _, err := time.Parse(layout, s); err == nil {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

// attachmentIndex lists the files of attachments/ that a cell names, each once, and gives each a ref
type attachmentIndex struct {
	zipFiles map[string]*zip.File
	byPath   map[string]string // path -> ref
	files    []dto.BaseExportAttachment
}

func newAttachmentIndex(_ *dto.BaseExportData, zipFiles map[string]*zip.File) *attachmentIndex {
	return &attachmentIndex{zipFiles: zipFiles, byPath: map[string]string{}}
}

// isFilePath reports whether a path is a file directly in attachments/ that the zip holds (the form the importer accepts)
func (a *attachmentIndex) isFilePath(p string) bool {
	if !strings.HasPrefix(p, AttachmentDir) || strings.Contains(p[len(AttachmentDir):], "/") {
		return false
	}
	_, ok := a.zipFiles[p]
	return ok && attachmentPathPattern.MatchString(p)
}

// refPrefix is the "a12_" an export puts in front of the name of an attachment file
var refPrefix = regexp.MustCompile(`^a[0-9]+_`)

var attachmentPathPattern = regexp.MustCompile(`^attachments/[A-Za-z0-9_.-]{1,200}$`)

// isFileCell reports whether a cell names only files of the zip
func (a *attachmentIndex) isFileCell(cell string) bool {
	parts := splitCell(cell)
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		if !a.isFilePath(p) {
			return false
		}
	}
	return true
}

func splitCell(cell string) []string {
	var out []string
	for _, p := range joinerChars.Split(cell, -1) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// refsOf gives the refs of the files of a cell, adding each file to the list the first time it is named
func (a *attachmentIndex) refsOf(cell string) []string {
	var refs []string
	for _, p := range splitCell(cell) {
		ref, ok := a.byPath[p]
		if !ok {
			f, exists := a.zipFiles[p]
			if !exists {
				continue
			}
			ref = fmt.Sprintf("a%d", len(a.files)+1)
			a.byPath[p] = ref
			name := path.Base(p)
			// an export names a file "<ref>_<name>"; the file goes back to the name it had
			if trimmed := refPrefix.ReplaceAllString(name, ""); trimmed != "" {
				name = trimmed
			}
			a.files = append(a.files, dto.BaseExportAttachment{
				Ref:         ref,
				Filename:    name,
				ContentType: mime.TypeByExtension(strings.ToLower(path.Ext(name))),
				Size:        int64(f.UncompressedSize64),
				Path:        p,
			})
		}
		refs = append(refs, ref)
	}
	return refs
}
