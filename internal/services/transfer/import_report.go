// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
)

// maxLoggedRows is how many skipped rows the error log describes; the rest are only counted
const maxLoggedRows = 1000

// skippedRow is a row an import left out, and why
type skippedRow struct {
	table  dto.BaseExportTable
	rid    string
	reason string
	cells  map[string]interface{}
}

// skipRow records a row the import leaves out and deletes the files that were uploaded for it, so a skipped row
// leaves nothing behind
func (r *importRun) skipRow(t dto.BaseExportTable, row dto.BaseExportRow, cause error, assetIDs []string) {
	r.skipped = append(r.skipped, skippedRow{table: t, rid: row.RID, reason: reasonOf(cause), cells: row.Values})
	if len(assetIDs) == 0 {
		return
	}
	gone := map[string]bool{}
	for _, id := range assetIDs {
		if err := r.s.assets.DeleteAsset(context.WithoutCancel(r.ctx), id, r.schema); err == nil {
			gone[id] = true
		}
	}
	kept := r.assetIDs[:0]
	for _, id := range r.assetIDs {
		if !gone[id] {
			kept = append(kept, id)
		}
	}
	r.assetIDs = kept
}

// reasonOf says why a row was refused, in the words of the database when it gave a reason
func reasonOf(err error) string {
	var detailed *app_errors.DetailedError
	if errors.As(err, &detailed) && strings.TrimSpace(detailed.Message) != "" {
		return detailed.Message
	}
	if errors.Is(err, app_errors.DatabaseError) {
		return "the database refused the row"
	}
	return err.Error()
}

// finishReport counts what was left out and writes the error log, for an import that skips failed rows
func (r *importRun) finishReport() {
	if !r.skipFailed {
		return
	}
	r.result.SkippedRows = len(r.skipped)
	r.result.Rows -= len(r.skipped)
	if r.source != nil {
		r.result.EmptyRows = r.source.EmptyRows
		r.result.DuplicateRows = r.source.DuplicateRows
	}
	if len(r.skipped) > 0 {
		r.warn("%d row(s) were not imported because the database refused them; the error log says which and why", len(r.skipped))
	}
	r.result.ErrorLog = r.errorLog()
}

// errorLog is the text of the report: a summary, then each row that was left out with its file, line, reason and
// cells, then the raw rows. It has the form of the log of the import of a table from a CSV. It is empty when no row
// was left out: blank and repeated rows alone are only counted, they are not an error.
func (r *importRun) errorLog() string {
	if len(r.skipped) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Import Issues Report\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	b.WriteString(strings.Repeat("=", 100) + "\n\n")
	b.WriteString("SUMMARY:\n")
	b.WriteString(fmt.Sprintf("Total Error Rows: %d\n", len(r.skipped)))
	b.WriteString(fmt.Sprintf("Total Empty Rows: %d\n", r.result.EmptyRows))
	b.WriteString(fmt.Sprintf("Total Duplicate Rows: %d\n", r.result.DuplicateRows))
	b.WriteString(strings.Repeat("-", 100) + "\n\n")

	b.WriteString("ROWS NOT IMPORTED (the database refused them):\n\n")
	logged := r.skipped
	if len(logged) > maxLoggedRows {
		logged = logged[:maxLoggedRows]
	}
	var raw strings.Builder
	var lastHeader string
	for i, s := range logged {
		src, known := rowSourceOf(r.source, s.rid)
		where := fmt.Sprintf("table %q", s.table.Title)
		cells := cellsOf(s)
		if known {
			where = fmt.Sprintf("%s, line %d (table %q)", src.File, src.Line, src.Table)
			cells = src.Cells
		}
		b.WriteString(fmt.Sprintf("[Error %d] %s\n", i+1, where))
		b.WriteString(fmt.Sprintf("Reason: %s\n", s.reason))
		b.WriteString(fmt.Sprintf("Row Data: %s\n\n", strings.Join(cells, " | ")))
		if known {
			if header := src.File + "\n" + strings.Join(escapeCells(src.Headers), ","); header != lastHeader {
				raw.WriteString("\n# " + src.File + "\n" + strings.Join(escapeCells(src.Headers), ",") + "\n")
				lastHeader = header
			}
			raw.WriteString(strings.Join(escapeCells(src.Cells), ",") + "\n")
		}
	}
	if len(r.skipped) > len(logged) {
		b.WriteString(fmt.Sprintf("... and %d more rows that are not listed.\n\n", len(r.skipped)-len(logged)))
	}
	if raw.Len() > 0 {
		b.WriteString(strings.Repeat("-", 100) + "\n")
		b.WriteString("RAW CSV DATA (the rows that were not imported, as they were in the files):\n")
		b.WriteString(raw.String())
	}
	return b.String()
}

// rowSource finds where a row came from; a nil source knows nothing
func rowSourceOf(src *dto.ImportSource, rid string) (dto.ImportRowSource, bool) {
	if src == nil {
		return dto.ImportRowSource{}, false
	}
	row, ok := src.Rows[rid]
	return row, ok
}

// cellsOf is the values of a row by field key, for a row that has no source line
func cellsOf(s skippedRow) []string {
	var out []string
	for _, f := range s.table.Fields {
		if v, ok := s.cells[f.Key]; ok {
			out = append(out, fmt.Sprint(v))
		}
	}
	return out
}

// escapeCells quotes the cells that need it, so a line of them is a line of a CSV
func escapeCells(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		if strings.ContainsAny(c, ",\"\n\r") {
			c = "\"" + strings.ReplaceAll(c, "\"", "\"\"") + "\""
		}
		out[i] = c
	}
	return out
}
