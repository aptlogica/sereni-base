// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package csv is the export adapter of delimited text: one file per table in a zip, plus the attachment files.
// The delimiter is a comma, a semicolon, a tab or a pipe; the text is always UTF-8 without a
// byte-order mark. Import reads the same zip back as a new base (import.go).
package csv

import (
	"archive/zip"
	"context"
	stdcsv "encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/internal/ziputil"
)

const (
	// Name is the name of this format
	Name = "csv"
	// AttachmentDir is the folder of the zip that holds the files
	AttachmentDir = "attachments/"

	// DelimiterTab, DelimiterPipe: the delimiters that are not a comma or a semicolon
	DelimiterTab  = '\t'
	DelimiterPipe = '|'
)

var (
	unsafeArchiveNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	unsafeTableNameChars   = regexp.MustCompile(`[\\/:*?"<>|]`)
)

type csvAdapter struct{}

// New returns the CSV adapter
func New() adapter.Adapter { return csvAdapter{} }

func (csvAdapter) Name() string { return Name }

// Capabilities: a CSV holds the field titles and the cell values, and the attachment files next to it. Links and
// lookups arrive as text (the core service turns them into text fields), so they are not carried as such.
func (csvAdapter) Capabilities() adapter.Capabilities {
	return adapter.Capabilities{Structure: true, Rows: true, Files: true}
}

// Describe: an export is always a zip, whatever the delimiter
func (csvAdapter) Describe(_ adapter.ExportOptions, title string) (string, string) {
	name := strings.Trim(unsafeArchiveNameChars.ReplaceAllString(title, "_"), "_")
	if name == "" {
		name = "base"
	}
	return name + ".csv.zip", "application/zip"
}

// Delimiters are the delimiters an export accepts, by the word the request uses
var Delimiters = map[string]rune{"comma": ',', "semicolon": ';', "tab": DelimiterTab, "pipe": DelimiterPipe}

// ValidDelimiter reports whether r is one of the delimiters an export accepts
func ValidDelimiter(r rune) bool {
	return r == ',' || r == ';' || r == DelimiterTab || r == DelimiterPipe
}

// Joiner is what joins several values in one cell. It is a pipe, except with the pipe delimiter, where it is a
// semicolon so that a cell never holds the delimiter between its values.
func Joiner(delimiter rune) string {
	if delimiter == DelimiterPipe {
		return ";"
	}
	return "|"
}

// Extension is the extension of the files in the zip: always .csv, whatever the delimiter
func Extension(delimiter rune) string {
	return ".csv"
}

// Export writes one file per table, then the attachment files, as a zip. A file that cannot be read or is too large
// is left out and added to m.Skipped, and no cell points at it.
func (csvAdapter) Export(ctx context.Context, m *dto.BaseExportFile, opt adapter.ExportOptions, out io.Writer) error {
	delim := opt.Delimiter
	if delim == 0 {
		delim = ','
	}
	if !ValidDelimiter(delim) {
		return fmt.Errorf("the delimiter %q is not supported", string(delim))
	}
	joiner := Joiner(delim)

	hasFiles := m.Data != nil && len(m.Data.Attachments) > 0
	if hasFiles && opt.OpenFile == nil {
		return errors.New("files cannot be exported: no storage provider is configured")
	}

	zw := zip.NewWriter(out)
	// pack the files first, so the files that were skipped are known when the cells are written
	paths := map[string]string{} // attachment ref -> path in the zip
	if hasFiles {
		for _, a := range m.Data.Attachments {
			path := cellSafe(a.Path, joiner)
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

	names := fileNames(m.Schema.Tables, Extension(delim))
	for i, t := range m.Schema.Tables {
		w, err := zw.Create(names[i])
		if err != nil {
			return err
		}
		if err := writeTable(w, t, m.Data, delim, joiner, paths); err != nil {
			return err
		}
	}
	return zw.Close()
}

// writeTable writes the header (the titles of the fields, system fields left out) and one record per row
func writeTable(w io.Writer, t dto.BaseExportTable, data *dto.BaseExportData, delim rune, joiner string, paths map[string]string) error {
	fields := make([]dto.BaseExportField, 0, len(t.Fields))
	for _, f := range t.Fields {
		if !f.System {
			fields = append(fields, f)
		}
	}
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].OrderIndex < fields[j].OrderIndex })

	cw := stdcsv.NewWriter(w)
	cw.Comma = delim
	cw.UseCRLF = true

	header := make([]string, len(fields))
	seen := map[string]int{}
	for i, f := range fields {
		header[i] = f.Title
		if n := seen[f.Title]; n > 0 {
			header[i] = fmt.Sprintf("%s (%d)", f.Title, n+1)
		}
		seen[f.Title]++
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	if data != nil {
		record := make([]string, len(fields))
		for _, row := range data.Tables[t.Key].Rows {
			for i, f := range fields {
				record[i] = cellText(f, row.Values[f.Key], joiner, paths)
			}
			if err := cw.Write(record); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

// fileNames gives each table a safe, unique file name: the title with \ / : * ? " < > | replaced by _, numbered
// when two tables end up with the same name
func fileNames(tables []dto.BaseExportTable, ext string) []string {
	out := make([]string, len(tables))
	used := map[string]bool{}
	for i, t := range tables {
		base := strings.TrimSpace(unsafeTableNameChars.ReplaceAllString(t.Title, "_"))
		if base == "" {
			base = "table"
		}
		name := base
		for n := 2; used[strings.ToLower(name)]; n++ {
			name = fmt.Sprintf("%s (%d)", base, n)
		}
		used[strings.ToLower(name)] = true
		out[i] = name + ext
	}
	return out
}

// cellSafe replaces the joiner in a path with an underscore, so a path never splits a cell
func cellSafe(path, joiner string) string {
	return strings.ReplaceAll(path, joiner, "_")
}
