// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package transfer_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	csvformat "github.com/aptlogica/sereni-base/internal/services/transfer/adapter/csv"
)

// A base exported as CSV can be imported as a new base: one table per file, the rows of each file, and the links
// (which the export wrote as text) as plain text fields. Every delimiter works.
func TestCSVExportThenImportCreatesANewBase(t *testing.T) {
	for _, delim := range []rune{',', ';', '\t', '|'} {
		c := seedCRM(t)
		svc := newService(c.db)
		ctx := context.Background()

		file, err := svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{IncludeData: true, LinksAsText: true, Format: "csv", Delimiter: delim})
		must(t, err)
		var zipped bytes.Buffer
		must(t, csvformat.New().Export(ctx, file, adapter.ExportOptions{IncludeData: true, OpenFile: svc.svc.OpenFile, Delimiter: delim}, &zipped))

		read, err := csvformat.New().Import(ctx, bytes.NewReader(zipped.Bytes()), int64(zipped.Len()), adapter.ImportOptions{Filename: "CRM.csv.zip"})
		must(t, err)
		res, err := svc.svc.ImportBase(ctx, "s", workspace, "user-1", read.File, dto.ImportBaseOptions{IncludeData: true, Title: "From CSV", Format: "csv"})
		must(t, err)
		if res.BaseID == "" || res.BaseID == c.baseID || res.Title != "From CSV" {
			t.Fatalf("delimiter %q: the import must create a new base, got %+v", string(delim), res)
		}
		if res.Tables != len(file.Schema.Tables) || res.Rows != 5 {
			t.Fatalf("delimiter %q: result = %+v, want %d tables and 5 rows", string(delim), res, len(file.Schema.Tables))
		}

		// the new base has the rows of the files, and no links: they were text
		again, err := svc.svc.ExportBase(ctx, "s", res.BaseID, dto.ExportBaseOptions{IncludeData: true})
		must(t, err)
		for _, tbl := range again.Schema.Tables {
			for _, f := range tbl.Fields {
				if f.UIDT == "links" || f.UIDT == "lookup" {
					t.Fatalf("delimiter %q: %s.%s is %s; a CSV carries text", string(delim), tbl.Title, f.Title, f.UIDT)
				}
			}
		}
		total := 0
		for _, n := range again.Data.RowCounts {
			total += n
		}
		if total != 5 {
			t.Fatalf("delimiter %q: %d rows after the import, want 5", string(delim), total)
		}
	}
}

// readCSV reads a zip of files with the csv adapter, as an upload is read
func readCSV(t *testing.T, files map[string]string, name string) *adapter.ImportResult {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for fname, content := range files {
		w, _ := zw.Create(fname)
		_, _ = io.WriteString(w, content)
	}
	must(t, zw.Close())
	read, err := csvformat.New().Import(context.Background(), bytes.NewReader(buf.Bytes()), int64(buf.Len()), adapter.ImportOptions{Filename: name})
	must(t, err)
	return read
}

// refuseBad is a database that refuses any row with the text "bad", with the reason Postgres would give
func refuseBad(values map[string]interface{}) error {
	for _, v := range values {
		if v == "bad" {
			return app_errors.WithMessage(app_errors.DatabaseError, `value "bad" is out of range for type integer (column amount)`)
		}
	}
	return nil
}

// A csv import leaves out the rows the database refuses and logs each with its file, line and reason; the other rows
// are imported. Without SkipFailedRows the first refused row fails the import.
func TestCSVImportSkipsTheRowsTheDatabaseRefusesAndLogsThem(t *testing.T) {
	files := map[string]string{
		"Orders.csv": "No,Note\n1,ok\n2,bad\n3,fine\n4,bad\n,\n3,fine\n",
		"Tags.csv":   "Label\nred\n",
	}
	read := readCSV(t, files, "Shop.csv.zip")
	db := newFakeDB()
	db.refuseRow = refuseBad
	svc := newService(db)
	res, err := svc.svc.ImportBase(context.Background(), "s", workspace, "user-1", read.File, dto.ImportBaseOptions{
		IncludeData: true, Format: "csv", SkipFailedRows: true, Source: read.Source,
	})
	must(t, err)
	if res.Rows != 4 || res.SkippedRows != 2 || res.EmptyRows != 1 || res.DuplicateRows != 1 {
		t.Fatalf("result = %+v: want 4 rows imported (3 orders + 1 tag, with a repeat), 2 skipped, 1 empty, 1 duplicate", res)
	}
	for _, want := range []string{
		"Import Issues Report", "Total Error Rows: 2", "Total Empty Rows: 1", "Total Duplicate Rows: 1",
		"Orders.csv, line 3", "Orders.csv, line 5", `Reason: value "bad" is out of range for type integer (column amount)`,
		"Row Data: 2 | bad", "Row Data: 4 | bad", "RAW CSV DATA", "No,Note", "2,bad", "4,bad",
	} {
		if !strings.Contains(res.ErrorLog, want) {
			t.Fatalf("the error log lacks %q:\n%s", want, res.ErrorLog)
		}
	}
	if strings.Contains(res.ErrorLog, "Tags.csv, line") {
		t.Fatalf("a row that was imported must not be in the log:\n%s", res.ErrorLog)
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "2 row(s) were not imported") {
		t.Fatalf("warnings = %v", res.Warnings)
	}
	// the rows that were accepted are in the new base
	stored := 0
	for _, rows := range db.rows {
		for _, r := range rows {
			for _, v := range r {
				if v == "bad" {
					t.Fatalf("a refused row was stored: %v", r)
				}
			}
		}
		stored += len(rows)
	}
	if stored != 4 {
		t.Fatalf("%d rows stored, want 4", stored)
	}
}

func TestCSVImportWithoutSkipFailedRowsFailsOnTheFirstRefusedRow(t *testing.T) {
	read := readCSV(t, map[string]string{"Orders.csv": "No,Note\n1,ok\n2,bad\n"}, "Shop.csv.zip")
	db := newFakeDB()
	db.refuseRow = refuseBad
	svc := newService(db)
	_, err := svc.svc.ImportBase(context.Background(), "s", workspace, "user-1", read.File, dto.ImportBaseOptions{IncludeData: true, Format: "csv"})
	if err == nil {
		t.Fatal("a refused row must fail an import that does not skip rows")
	}
	for _, rows := range db.rows {
		if len(rows) > 0 {
			t.Fatal("a failed import must leave nothing behind")
		}
	}
}

// A log exists only when a row was left out: blank and repeated rows are counted, but they are not an error
func TestCSVImportErrorLogOnlyWhenARowWasSkipped(t *testing.T) {
	run := func(csv string) dto.ImportBaseResult {
		read := readCSV(t, map[string]string{"T.csv": csv}, "x.csv.zip")
		res, err := newService(newFakeDB()).svc.ImportBase(context.Background(), "s", workspace, "user-1", read.File,
			dto.ImportBaseOptions{IncludeData: true, Format: "csv", SkipFailedRows: true, Source: read.Source})
		must(t, err)
		return res
	}
	clean := run("a,b\n1,x\n2,y\n")
	if clean.ErrorLog != "" || clean.SkippedRows != 0 || clean.Rows != 2 {
		t.Fatalf("a clean file has no log: %+v", clean)
	}
	blanks := run("a,b\n1,x\n,\n1,x\n")
	if blanks.SkippedRows != 0 || blanks.EmptyRows != 1 || blanks.DuplicateRows != 1 || blanks.ErrorLog != "" {
		t.Fatalf("blank and repeated rows are counted and give no log: %+v\n%s", blanks, blanks.ErrorLog)
	}
}

// The log says what was imported (format, file, base), lists the warnings, and is also written when there is only a warning
func TestImportErrorLogHasAHeaderAndTheWarnings(t *testing.T) {
	files := map[string]string{"Orders.csv": "No,Note\n1,ok\n2,bad\n"}
	read := readCSV(t, files, "Shop.csv.zip")
	db := newFakeDB()
	db.refuseRow = refuseBad
	res, err := newService(db).svc.ImportBase(context.Background(), "s", workspace, "user-1", read.File, dto.ImportBaseOptions{
		IncludeData: true, Format: "csv", SkipFailedRows: true, Source: read.Source, FileName: "Shop.csv.zip",
	})
	must(t, err)
	for _, want := range []string{
		"Import Issues Report", "Format: CSV      File: Shop.csv.zip", "Base: Shop", "Result: completed with issues",
		"Total Error Rows: 1", "Warnings: 1", "WARNINGS:", "- 1 row(s) were not imported", "ROWS NOT IMPORTED",
	} {
		if !strings.Contains(res.ErrorLog, want) {
			t.Fatalf("the error log lacks %q:\n%s", want, res.ErrorLog)
		}
	}

	// a Sereni Base import skips no rows, but its warnings still make a log
	c := seedCRM(t)
	file, _ := exportZip(t, c, c.baseID)
	res, err = newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: true, FileName: "crm.sbase.json"})
	must(t, err)
	if res.SkippedRows != 0 || !strings.Contains(res.ErrorLog, "WARNINGS:") || !strings.Contains(res.ErrorLog, "upload the .sbase.zip") || !strings.Contains(res.ErrorLog, "Format: Sereni Base      File: crm.sbase.json") || strings.Contains(res.ErrorLog, "ROWS NOT IMPORTED") {
		t.Fatalf("a warning gives a log without rows: %+v\n%s", res, res.ErrorLog)
	}
}

// A table or field name whose name in the database would be over 63 characters is refused in plain words, with all
// of them listed at once, and the log of the failed import says the same
func TestImportRefusesNamesThatExceedTheDatabaseLimit(t *testing.T) {
	// "Getting Started - Quick Reference Links (Quick Links) 2026-11-25_12-53": 64 letters, digits and spaces + 16 = 80
	title := "Getting Started - Quick Reference Links (Quick Links) 2026-11-25_12-53"
	read := readCSV(t, map[string]string{title + ".csv": "a," + strings.Repeat("c", 60) + "\n1,x\n", "Short.csv": "a\n1\n"}, "Shop.csv.zip")
	_, err := newService(newFakeDB()).svc.ImportBase(context.Background(), "s", workspace, "user-1", read.File, dto.ImportBaseOptions{IncludeData: true, Format: "csv", SkipFailedRows: true, Source: read.Source, FileName: "Shop.csv.zip"})
	var invalid *dto.ImportValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) != 2 {
		t.Fatalf("err = %v", err)
	}
	wantTable := `schema.tables[0]: the table name "Getting Started - Quick Reference Links ..." exceeds the database limit of 63 characters: its database name would be 80 characters (the letters, digits and spaces of the name, plus 16 that the app adds). Shorten the name by at least 17 characters`
	if invalid.Problems[0] != wantTable {
		t.Fatalf("problem = %q\nwant      %q", invalid.Problems[0], wantTable)
	}
	if !strings.Contains(invalid.Problems[1], `schema.tables[0].fields[1]: the field name "`) || !strings.Contains(invalid.Problems[1], "exceeds the database limit of 63 characters") {
		t.Fatalf("problem = %q", invalid.Problems[1])
	}
}

// A name that fits is accepted: 47 letters is the most (symbols are dropped, a space counts as one), because 16 characters are added; a name that starts with a digit gets "table_" in front
func TestImportAcceptsNamesUpToTheDatabaseLimit(t *testing.T) {
	ok := strings.Repeat("a", 47)
	for name, title := range map[string]string{"47 letters": ok, "symbols are dropped": strings.Repeat("a", 47) + "(!!!)???"} {
		read := readCSV(t, map[string]string{title + ".csv": "a\n1\n"}, "Shop.csv.zip")
		if _, err := newService(newFakeDB()).svc.ImportBase(context.Background(), "s", workspace, "user-1", read.File, dto.ImportBaseOptions{IncludeData: true, Format: "csv", SkipFailedRows: true, Source: read.Source}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for name, title := range map[string]string{"48 letters": strings.Repeat("a", 48), "41 letters after a digit": "1" + strings.Repeat("a", 41)} {
		read := readCSV(t, map[string]string{title + ".csv": "a\n1\n"}, "Shop.csv.zip")
		_, err := newService(newFakeDB()).svc.ImportBase(context.Background(), "s", workspace, "user-1", read.File, dto.ImportBaseOptions{IncludeData: true, Format: "csv", SkipFailedRows: true, Source: read.Source})
		var invalid *dto.ImportValidationError
		if !errors.As(err, &invalid) {
			t.Fatalf("%s must be refused: %v", name, err)
		}
	}
}

// A name of a Sereni Base file that is over the limit is refused, in plain words: which name, how long, and the limit
func TestImportRefusesAFileNameThatIsTooLongInPlainWords(t *testing.T) {
	c := seedCRM(t)
	file, _ := exportZip(t, c, c.baseID)
	file = roundTrip(t, file)
	file.Schema.Tables[0].Title = strings.Repeat("B", 300)
	_, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", file, dto.ImportBaseOptions{IncludeData: true})
	var invalid *dto.ImportValidationError
	if !errors.As(err, &invalid) || len(invalid.Problems) != 1 {
		t.Fatalf("err = %v", err)
	}
	want := `schema.tables[0]: the table name "` + strings.Repeat("B", 40) + `..." is too long: it has 300 characters and the limit is 255`
	if invalid.Problems[0] != want {
		t.Fatalf("problem = %q\nwant      %q", invalid.Problems[0], want)
	}
}
