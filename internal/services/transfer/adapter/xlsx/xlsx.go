// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package xlsx is the adapter of Excel workbooks: one .xlsx file for a whole base, one sheet per table. The first
// row of a sheet holds the field titles. Numbers, booleans and dates are written as real Excel cells; everything else
// is text. Links, lookups and rollups arrive as text (the core service turns them into text fields).
//
// Without attachment files an export is the workbook itself (.xlsx). With them it is a zip (.xlsx.zip) that holds the
// workbook and the files in an attachments/ folder, the same layout as the csv export, and an attachment cell holds
// the paths of its files. Import reads either.
package xlsx

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/xuri/excelize/v2"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/internal/ziputil"
)

const (
	// Name is the name of this format
	Name = "xlsx"
	// Extension is the extension of the file
	Extension = ".xlsx"
	// ContentType is the content type of the file
	ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	// ZipExtension is the extension of an export that holds attachment files
	ZipExtension = ".xlsx.zip"
	// AttachmentDir is the folder of the zip that holds the files
	AttachmentDir = "attachments/"

	// MaxSheetNameLength is the longest name Excel allows for a sheet, counted in UTF-16 units
	MaxSheetNameLength = 31
	// MaxRows is the number of rows of a sheet, the header included
	MaxRows = 1048576
	// MaxCellLength is the longest text Excel keeps in a cell
	MaxCellLength = 32767

	joiner = "|"
	// exactNumberLimit is the largest whole number Excel holds exactly (15 digits)
	exactNumberLimit = 1e15
)

var (
	unsafeFileNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	forbiddenSheetChars = regexp.MustCompile(`[\\/?*\[\]:]`)
)

type xlsxAdapter struct{}

// New returns the Excel adapter
func New() adapter.Adapter { return xlsxAdapter{} }

func (xlsxAdapter) Name() string { return Name }

// Capabilities: a workbook holds the table and field titles and the cell values, and the attachment files travel
// next to it in a zip. Links and lookups are not carried as such.
func (xlsxAdapter) Capabilities() adapter.Capabilities {
	return adapter.Capabilities{Structure: true, Rows: true, Files: true}
}

func safeName(title string) string {
	name := strings.Trim(unsafeFileNameChars.ReplaceAllString(title, "_"), "_")
	if name == "" {
		name = "base"
	}
	return name
}

// Describe: a workbook, or a zip of the workbook and its attachment files when the export carries files
func (xlsxAdapter) Describe(opt adapter.ExportOptions, title string) (string, string) {
	if opt.IncludeFiles {
		return safeName(title) + ZipExtension, "application/zip"
	}
	return safeName(title) + Extension, ContentType
}

// Export writes the model as a workbook with one sheet per table; with IncludeFiles the workbook goes into a zip
// with the attachment files. A file that cannot be read or is too large is left out and added to m.Skipped, and no
// cell points at it. A table that has more rows than a sheet can hold, or a cell with more text than a cell can hold,
// is cut and added to m.Skipped.
func (xlsxAdapter) Export(ctx context.Context, m *dto.BaseExportFile, opt adapter.ExportOptions, out io.Writer) error {
	if !opt.IncludeFiles {
		return writeWorkbook(ctx, m, nil, out)
	}
	hasFiles := m.Data != nil && len(m.Data.Attachments) > 0
	if hasFiles && opt.OpenFile == nil {
		return errors.New("files cannot be exported: no storage provider is configured")
	}
	zw := zip.NewWriter(out)
	// pack the files first, so the files that were skipped are known when the cells are written
	paths := map[string]string{} // attachment ref -> path in the zip
	if hasFiles {
		for _, a := range m.Data.Attachments {
			path := strings.ReplaceAll(a.Path, joiner, "_")
			if _, _, err := ziputil.PackFile(ctx, zw, opt.OpenFile, a.Filename, path, a.StoragePath); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				m.Skipped = append(m.Skipped, fmt.Sprintf("attachment %s not exported: %v", a.Filename, err))
				continue
			}
			paths[a.Ref] = path
		}
	}
	w, err := zw.Create(safeName(m.Base.Title) + Extension)
	if err != nil {
		return err
	}
	if err := writeWorkbook(ctx, m, paths, w); err != nil {
		return err
	}
	return zw.Close()
}

// writeWorkbook writes the workbook; paths gives the path of each attachment ref whose file was packed
func writeWorkbook(ctx context.Context, m *dto.BaseExportFile, paths map[string]string, out io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	styles, err := newStyles(f)
	if err != nil {
		return err
	}

	names := SheetNames(m.Schema.Tables)
	if len(names) == 0 {
		names = []string{"Sheet1"} // a workbook always has a sheet
	}
	for i, name := range names {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				return err
			}
			continue
		}
		if _, err := f.NewSheet(name); err != nil {
			return err
		}
	}
	for i, t := range m.Schema.Tables {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writeSheet(ctx, f, names[i], t, m, styles, paths); err != nil {
			return err
		}
	}
	return f.Write(out)
}

type styles struct {
	header, date, datetime int
}

func newStyles(f *excelize.File) (styles, error) {
	var s styles
	var err error
	if s.header, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}}); err != nil {
		return s, err
	}
	dateFmt, datetimeFmt := "yyyy-mm-dd", "yyyy-mm-dd hh:mm:ss"
	if s.date, err = f.NewStyle(&excelize.Style{CustomNumFmt: &dateFmt}); err != nil {
		return s, err
	}
	s.datetime, err = f.NewStyle(&excelize.Style{CustomNumFmt: &datetimeFmt})
	return s, err
}

// writeSheet writes the header (the titles of the fields, system fields left out) and one row per record
func writeSheet(ctx context.Context, f *excelize.File, sheet string, t dto.BaseExportTable, m *dto.BaseExportFile, st styles, paths map[string]string) error {
	fields := make([]dto.BaseExportField, 0, len(t.Fields))
	for _, fd := range t.Fields {
		if !fd.System {
			fields = append(fields, fd)
		}
	}
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].OrderIndex < fields[j].OrderIndex })

	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}
	if len(fields) > 0 {
		if err := sw.SetColWidth(1, len(fields), 20); err != nil {
			return err
		}
	}

	header := make([]interface{}, len(fields))
	seen := map[string]int{}
	for i, fd := range fields {
		title := fd.Title
		if n := seen[fd.Title]; n > 0 {
			title = fmt.Sprintf("%s (%d)", fd.Title, n+1)
		}
		seen[fd.Title]++
		header[i] = excelize.Cell{StyleID: st.header, Value: clean(title)}
	}
	if err := setRow(sw, 1, header); err != nil {
		return err
	}

	if m.Data != nil {
		rows := m.Data.Tables[t.Key].Rows
		if len(rows) > MaxRows-1 {
			m.Skipped = append(m.Skipped, fmt.Sprintf("%s: only the first %d rows fit in a sheet; %d rows left out", t.Title, MaxRows-1, len(rows)-(MaxRows-1)))
			rows = rows[:MaxRows-1]
		}
		cut := map[string]bool{}
		record := make([]interface{}, len(fields))
		for r, row := range rows {
			if r%1000 == 0 {
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			for i, fd := range fields {
				cell, truncated := excelCell(fd, row.Values[fd.Key], st, paths)
				if truncated && !cut[fd.Key] {
					cut[fd.Key] = true
					m.Skipped = append(m.Skipped, fmt.Sprintf("%s.%s: text longer than %d characters was cut", t.Title, fd.Title, MaxCellLength))
				}
				record[i] = cell
			}
			if err := setRow(sw, r+2, record); err != nil {
				return err
			}
		}
	}
	return sw.Flush()
}

func setRow(sw *excelize.StreamWriter, row int, values []interface{}) error {
	cell, err := excelize.CoordinatesToCellName(1, row)
	if err != nil {
		return err
	}
	return sw.SetRow(cell, values)
}

// excelCell turns the value of a cell into what the sheet stores: a number, a boolean, a date, or text. It reports
// whether the text had to be cut.
//
//	number, decimal, currency, percent, duration, year, rating   a number (as text when it is too big to be exact)
//	boolean                                                      TRUE or FALSE
//	date, datetime                                               an Excel date, shown as yyyy-mm-dd or yyyy-mm-dd hh:mm:ss
//	attachment                                                   the paths of the packed files, joined with "|"
//	lists                                                        the values, joined with "|"
//	json and the rest                                            text
//
// A text value is never a formula: the stream writer stores a string as a string, so a value such as =SUM(A1) is
// shown as it is.
func excelCell(f dto.BaseExportField, v interface{}, st styles, paths map[string]string) (interface{}, bool) {
	if v == nil {
		return nil, false
	}
	switch f.UIDT {
	case "number", "decimal", "currency", "percent", "duration", "year", "rating":
		if n, ok := toFloat(v); ok && n > -exactNumberLimit && n < exactNumberLimit {
			return n, false
		}
	case "boolean":
		if b, ok := v.(bool); ok {
			return b, false
		}
	case "date":
		if t, ok := toTime(v); ok {
			return excelize.Cell{StyleID: st.date, Value: t}, false
		}
	case "datetime":
		if t, ok := toTime(v); ok {
			return excelize.Cell{StyleID: st.datetime, Value: t}, false
		}
	case "attachment":
		var out []string
		for _, ref := range toStrings(v) {
			if p, ok := paths[ref]; ok {
				out = append(out, p)
			}
		}
		return text(strings.Join(out, joiner))
	case "json":
		if raw, err := json.Marshal(v); err == nil {
			return text(string(raw))
		}
	}
	return text(valueText(v))
}

// text cleans a text cell and cuts it to what a cell can hold
func text(s string) (interface{}, bool) {
	s = clean(s)
	if s == "" {
		return nil, false
	}
	if runes := []rune(s); len(runes) > MaxCellLength {
		return string(runes[:MaxCellLength]), true
	}
	return s, false
}

// clean drops the characters an XML document cannot hold (control characters other than tab, newline and carriage return)
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' || (r >= 0x20 && r != 0xFFFE && r != 0xFFFF && !(r >= 0xD800 && r <= 0xDFFF)) {
			return r
		}
		return -1
	}, s)
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		n, err := x.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return n, err == nil
	}
	return 0, false
}

var timeLayouts = []string{
	time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02",
}

// toTime reads a date or a date and time; a time with a zone is converted to UTC, as the file holds no zone
func toTime(v interface{}) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x.UTC(), true
	case string:
		s := strings.TrimSpace(x)
		for _, layout := range timeLayouts {
			if t, err := time.Parse(layout, s); err == nil {
				return t.UTC(), true
			}
		}
	}
	return time.Time{}, false
}

func valueText(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		return x.String()
	case time.Time:
		return x.UTC().Format(time.RFC3339)
	case []string:
		return strings.Join(x, joiner)
	case []interface{}:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			if s := valueText(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, joiner)
	case map[string]interface{}:
		raw, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprint(x)
		}
		return string(raw)
	default:
		return fmt.Sprint(x)
	}
}

func toStrings(v interface{}) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{x}
	}
	return nil
}

// SheetNames gives each table a sheet name Excel accepts, unique ignoring case:
//
//	at most 31 characters (UTF-16 units)
//	\ / ? * [ ] :  are replaced by _
//	no ' at the start or the end
//	empty names become Sheet<N>
//	"History" is reserved by Excel, so a table of that name gets a number
//
// A name already taken gets " (2)", " (3)", ... and is cut so that the whole name stays within the limit.
func SheetNames(tables []dto.BaseExportTable) []string {
	out := make([]string, len(tables))
	used := map[string]bool{"history": true}
	for i, t := range tables {
		base := forbiddenSheetChars.ReplaceAllString(t.Title, "_")
		base = strings.Trim(strings.TrimSpace(base), "'")
		base = strings.TrimSpace(base)
		if base == "" {
			base = fmt.Sprintf("Sheet%d", i+1)
		}
		name := cutName(base, MaxSheetNameLength)
		for n := 2; used[strings.ToLower(name)]; n++ {
			suffix := fmt.Sprintf(" (%d)", n)
			name = cutName(base, MaxSheetNameLength-len(suffix)) + suffix
		}
		used[strings.ToLower(name)] = true
		out[i] = name
	}
	return out
}

// cutName cuts a name to at most limit UTF-16 units, without splitting a character; it never ends with a space or '
func cutName(name string, limit int) string {
	units := 0
	var b strings.Builder
	for _, r := range name {
		n := len(utf16.Encode([]rune{r}))
		if units+n > limit {
			break
		}
		units += n
		b.WriteRune(r)
	}
	return strings.TrimRight(b.String(), " '")
}

// errNotWorkbook is what the user sees for a file that is not an Excel workbook
var errNotWorkbook = errors.New("the file is not a valid .xlsx workbook")
