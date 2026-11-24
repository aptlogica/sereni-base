// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package jsonfmt is the adapter of JSON files: a zip with one .json file per table, the same layout as the csv
// export. A table file is an array of rows; each row is an object keyed by the field title. It is the flat format
// of the csv adapter (table data only, the types are guessed on import), not the .sbase.json format, which holds the
// whole base. Links, lookups and rollups arrive as text (the core service turns them into text fields). The files
// of attachment fields are in an attachments/ folder of the zip, and an attachment cell holds the paths of its files.
//
// The package is not called json, so that it does not hide encoding/json.
package jsonfmt

import (
	"archive/zip"
	"bytes"
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

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/internal/ziputil"
)

const (
	// Name is the name of this format
	Name = "json"
	// Extension is the extension of the files in the zip
	Extension = ".json"
	// ZipExtension is the extension of an export
	ZipExtension = ".json.zip"
	// AttachmentDir is the folder of the zip that holds the files
	AttachmentDir = "attachments/"

	joiner = "|"
	// exactNumberLimit is the largest whole number written as a JSON number; a bigger one is written as text, so
	// that a reader that keeps numbers as 64 bit floats does not round it
	exactNumberLimit = 1e15
)

var (
	unsafeFileNameChars  = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	unsafeTableNameChars = regexp.MustCompile(`[\\/:*?"<>|]`)
)

type jsonAdapter struct{}

// New returns the JSON adapter
func New() adapter.Adapter { return jsonAdapter{} }

func (jsonAdapter) Name() string { return Name }

// Capabilities: the files hold the table and field titles and the cell values, and the attachment files travel
// next to them in the zip. Links and lookups are not carried as such.
func (jsonAdapter) Capabilities() adapter.Capabilities {
	return adapter.Capabilities{Structure: true, Rows: true, Files: true}
}

func safeName(title string) string {
	name := strings.Trim(unsafeFileNameChars.ReplaceAllString(title, "_"), "_")
	if name == "" {
		name = "base"
	}
	return name
}

// Describe: an export is always a zip
func (jsonAdapter) Describe(_ adapter.ExportOptions, title string) (string, string) {
	return safeName(title) + ZipExtension, "application/zip"
}

// Export writes one file per table, then the attachment files, as a zip. A file that cannot be read or is too large
// is left out and added to m.Skipped, and no cell points at it.
func (jsonAdapter) Export(ctx context.Context, m *dto.BaseExportFile, opt adapter.ExportOptions, out io.Writer) error {
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

	names := fileNames(m.Schema.Tables)
	for i, t := range m.Schema.Tables {
		if err := ctx.Err(); err != nil {
			return err
		}
		w, err := zw.Create(names[i])
		if err != nil {
			return err
		}
		if err := writeTable(ctx, w, t, m.Data, paths); err != nil {
			return err
		}
	}
	return zw.Close()
}

// writeTable writes the array of the rows of one table. The keys of a row are written one by one, in field order: a
// Go map would lose the order.
func writeTable(ctx context.Context, out io.Writer, t dto.BaseExportTable, data *dto.BaseExportData, paths map[string]string) error {
	w := &errWriter{w: out}
	fields := exportFields(t)
	titles := fieldTitles(fields)

	var rows []dto.BaseExportRow
	if data != nil {
		rows = data.Tables[t.Key].Rows
	}
	if len(rows) == 0 {
		w.str("[]\n")
		return w.err
	}
	w.str("[")
	for r, row := range rows {
		if r%1000 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		if r > 0 {
			w.str(",")
		}
		writeRow(w, fields, titles, row, paths)
	}
	w.str("\n]\n")
	return w.err
}

// exportFields are the fields of a table that are written, in field order: the system fields are left out
func exportFields(t dto.BaseExportTable) []dto.BaseExportField {
	fields := make([]dto.BaseExportField, 0, len(t.Fields))
	for _, f := range t.Fields {
		if !f.System {
			fields = append(fields, f)
		}
	}
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].OrderIndex < fields[j].OrderIndex })
	return fields
}

// fieldTitles are the keys of the fields, already encoded; a title used twice is numbered
func fieldTitles(fields []dto.BaseExportField) [][]byte {
	titles := make([][]byte, len(fields))
	seen := map[string]int{}
	for i, f := range fields {
		title := f.Title
		if n := seen[f.Title]; n > 0 {
			title = fmt.Sprintf("%s (%d)", f.Title, n+1)
		}
		seen[f.Title]++
		titles[i] = marshal(clean(title))
	}
	return titles
}

// writeRow writes one row as an object, its keys in field order
func writeRow(w *errWriter, fields []dto.BaseExportField, titles [][]byte, row dto.BaseExportRow, paths map[string]string) {
	if len(fields) == 0 {
		w.str("\n  {}")
		return
	}
	w.str("\n  {")
	for i, f := range fields {
		if i > 0 {
			w.str(",")
		}
		w.str("\n    ")
		w.raw(titles[i])
		w.str(": ")
		w.raw(marshal(jsonValue(f, row.Values[f.Key], paths)))
	}
	w.str("\n  }")
}

// errWriter keeps the first write error, so the writing code does not check every call
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) raw(b []byte) {
	if e.err == nil {
		_, e.err = e.w.Write(b)
	}
}

func (e *errWriter) str(s string) { e.raw([]byte(s)) }

// marshal is json.Marshal without the escaping of <, > and & (a cell is data, not HTML)
func marshal(v interface{}) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return []byte("null")
	}
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// fileNames gives each table a safe, unique file name: the title with \ / : * ? " < > | replaced by _, numbered
// when two tables end up with the same name
func fileNames(tables []dto.BaseExportTable) []string {
	out := make([]string, len(tables))
	used := map[string]bool{}
	for i, t := range tables {
		base := strings.TrimSpace(unsafeTableNameChars.ReplaceAllString(clean(t.Title), "_"))
		if base == "" {
			base = "table"
		}
		name := base
		for n := 2; used[strings.ToLower(name)]; n++ {
			name = fmt.Sprintf("%s (%d)", base, n)
		}
		used[strings.ToLower(name)] = true
		out[i] = name + Extension
	}
	return out
}

// jsonValue turns the value of a cell into what the file stores:
//
//	number, decimal, currency, percent, duration, year, rating   a number (text when it is too big to be exact)
//	boolean                                                      true or false
//	date                                                         "yyyy-mm-dd"
//	datetime                                                     RFC 3339, in UTC
//	multiSelect                                                  an array of text
//	attachment                                                   an array of the paths of the packed files
//	json                                                         the JSON value itself
//	the rest                                                     text
//
// A cell with no value (or only empty text) is null.
func jsonValue(f dto.BaseExportField, v interface{}, paths map[string]string) interface{} {
	if v == nil {
		return nil
	}
	switch f.UIDT {
	case "number", "decimal", "currency", "percent", "duration", "year", "rating":
		if n, ok := exactNumber(v); ok {
			return n
		}
	case "boolean":
		if b, ok := v.(bool); ok {
			return b
		}
	case "date":
		if t, ok := toTime(v); ok {
			return t.Format("2006-01-02")
		}
	case "datetime":
		if t, ok := toTime(v); ok {
			return t.Format(time.RFC3339)
		}
	case "multiSelect":
		return cleanAll(toStrings(v))
	case "attachment":
		return attachmentPaths(v, paths)
	case "json":
		return v
	}
	return textOrNil(valueText(v))
}

// exactNumber is the number as a JSON number, unless it is too big to be exact
func exactNumber(v interface{}) (json.RawMessage, bool) {
	n, ok := toFloat(v)
	if !ok || n <= -exactNumberLimit || n >= exactNumberLimit {
		return nil, false
	}
	return json.RawMessage(strconv.FormatFloat(n, 'f', -1, 64)), true
}

// attachmentPaths are the paths of the packed files of an attachment cell; a file that was skipped has no path
func attachmentPaths(v interface{}, paths map[string]string) []string {
	out := []string{}
	for _, ref := range toStrings(v) {
		if p, ok := paths[ref]; ok {
			out = append(out, p)
		}
	}
	return out
}

func textOrNil(s string) interface{} {
	if s = clean(s); s == "" {
		return nil
	}
	return s
}

func cleanAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, clean(s))
	}
	return out
}

// clean drops the control characters other than tab, newline and carriage return
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' || (r >= 0x20 && r != 0xFFFE && r != 0xFFFF) {
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
		return string(marshal(x))
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
