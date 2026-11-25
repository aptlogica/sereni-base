// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package transfer_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	transfer "github.com/aptlogica/sereni-base/internal/services/transfer"
)

func startCSV(t *testing.T, f *jobFixture, delimiter rune) dto.TransferJobResponse {
	t.Helper()
	job, err := f.svc.StartExport(context.Background(), constant.MasterDatabase, "base-1", "user-1", dto.ExportBaseOptions{
		IncludeData: true, Format: "csv", Delimiter: delimiter, LinksAsText: true,
	})
	if err != nil {
		t.Fatalf("StartExport: %v", err)
	}
	return job
}

// A csv export goes through the csv adapter: the job, the stored file and the download all carry the .csv.zip name
func TestCSVExportJobWritesACsvZip(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	job := startCSV(t, f, '\t')
	if job.FileName != "My_CRM_2026.csv.zip" {
		t.Fatalf("the job knows its file name from the start: %+v", job)
	}
	done := f.waitFor(t, job.ID, "user-1", completed)
	if done.FileName != "My_CRM_2026.csv.zip" || done.SizeBytes <= 0 {
		t.Fatalf("completed job: %+v", done)
	}
	if f.exporter.opts.Format != "csv" || f.exporter.opts.Delimiter != '\t' || !f.exporter.opts.LinksAsText {
		t.Fatalf("export options: %+v", f.exporter.opts)
	}
	file, _, err := f.svc.Open(context.Background(), constant.MasterDatabase, job.ID, "user-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer file.Close()
	data, _ := io.ReadAll(file)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("the download is not a zip: %v", err)
	}
	for _, e := range zr.File {
		if e.Name == "base.sbase.json" {
			t.Fatalf("a csv export must not hold the sbase file")
		}
	}
	if _, err := os.Stat(filepath.Join(f.dir, "exports__"+job.ID+"__My_CRM_2026.csv.zip")); err != nil {
		t.Fatalf("zip file: %v", err)
	}
}

// The default (no format) is still the SereniBase file
func TestExportJobWithoutAFormatIsSbase(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	job := start(t, f, "user-1")
	if job.FileName != "My_CRM_2026.sbase.zip" {
		t.Fatalf("file name %q", job.FileName)
	}
	f.waitFor(t, job.ID, "user-1", completed)
}

// A failed csv export leaves no file behind: the file is removed under the name it was stored with
func TestFailedCSVExportLeavesNoFile(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.fail = errors.New("boom")
	job := startCSV(t, f, ',')
	f.waitFor(t, job.ID, "user-1", failed)
	entries, _ := os.ReadDir(f.dir)
	if len(entries) != 0 {
		t.Fatalf("files left after a failed export: %v", entries)
	}
}

// An import job with format csv reads the kept zip with the csv adapter: the new base is named after the file and
// has one table per csv file
func TestImportJobReadsACsvZip(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	var zipped bytes.Buffer
	zw := zip.NewWriter(&zipped)
	for name, content := range map[string]string{"Customers.csv": "Name,Age\nAcme,3\n", "Orders.csv": "No\tTotal\nO-1\t9.5\n"} {
		w, _ := zw.Create(name)
		_, _ = io.WriteString(w, content)
	}
	_ = zw.Close()

	job, err := f.svc.StartImport(context.Background(), constant.MasterDatabase, "ws-1", "user-1",
		dto.ImportUpload{File: bytes.NewReader(zipped.Bytes()), Filename: "Sales CRM.csv.zip", Size: int64(zipped.Len())}, dto.ImportBaseOptions{IncludeData: true, Format: "csv", SkipFailedRows: true})
	if err != nil {
		t.Fatalf("StartImport: %v", err)
	}
	done := f.waitFor(t, job.ID, "user-1", completed)
	if done.BaseID == "" {
		t.Fatalf("the import must complete: %+v", done)
	}
	calls := f.exporter.importCalls()
	if len(calls) != 1 || calls[0].title != "Sales CRM" {
		t.Fatalf("ImportBase calls: %+v", calls)
	}
	if calls[0].opts.Format != "csv" {
		t.Fatalf("options: %+v", calls[0].opts)
	}
	// the job read the file with the csv adapter and gives the import where each row came from
	if calls[0].opts.Source == nil || len(calls[0].opts.Source.Rows) != 2 {
		t.Fatalf("source: %+v", calls[0].opts.Source)
	}
}

// A zip with no csv file fails the job with a message the user can act on
func TestImportJobOfACsvZipWithoutCsvFilesFails(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	var zipped bytes.Buffer
	zw := zip.NewWriter(&zipped)
	w, _ := zw.Create("readme.txt")
	_, _ = io.WriteString(w, "hello")
	_ = zw.Close()
	job, err := f.svc.StartImport(context.Background(), constant.MasterDatabase, "ws-1", "user-1",
		dto.ImportUpload{File: bytes.NewReader(zipped.Bytes()), Filename: "x.csv.zip", Size: int64(zipped.Len())}, dto.ImportBaseOptions{IncludeData: true, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	done := f.waitFor(t, job.ID, "user-1", failed)
	if !strings.Contains(done.Error, "no .csv file") {
		t.Fatalf("error = %q", done.Error)
	}
}

func csvZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, text := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(text))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// A failure while the base is being created says which step failed and why, in the message and in the log, and the log
// is the one the download endpoint serves
func TestFailedImportJobSaysWhichStepFailedAndWhy(t *testing.T) {
	db := newFakeDB()
	db.failOnUIDT = "number"
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.realImport = newService(db).svc

	body := csvZip(t, map[string]string{"Orders.csv": "No,Note\n1,ok\n2,fine\n"})
	job := startImport(t, f, "user-1", "Shop.csv.zip", body, dto.ImportBaseOptions{IncludeData: true, Format: "csv", SkipFailedRows: true})
	done := f.waitFor(t, job.ID, "user-1", failed)

	want := `the import failed while creating the fields: create field "No" in table "Orders": boom; nothing was created`
	if done.Error != want {
		t.Fatalf("error = %q, want %q", done.Error, want)
	}
	var result struct {
		ErrorLog string `json:"error_log"`
	}
	if err := json.Unmarshal(done.Result, &result); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"Format: CSV      File: Shop.csv.zip", "Result: failed (nothing was created)", "ERROR:\n" + want, "Step: creating the fields", `Reason: create field "No" in table "Orders": boom`} {
		if !strings.Contains(result.ErrorLog, line) {
			t.Fatalf("the log lacks %q:\n%s", line, result.ErrorLog)
		}
	}
	if strings.Contains(result.ErrorLog, "contact support") {
		t.Fatalf("the generic text must not be what the user is told:\n%s", result.ErrorLog)
	}
}

// What an error says about a connection, a credential or the server's files is not given to the user: the step still is
func TestFailedImportJobKeepsInternalsOutOfTheLog(t *testing.T) {
	for name, cause := range map[string]string{
		"a credential": "pq: password authentication failed for user admin",
		"a connection": "dial tcp 10.0.0.5:5432: connect: connection refused",
		"a panic":      "panic: runtime error: index out of range [3] with length 2",
	} {
		f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
		f.exporter.importErr = errors.New(cause)
		job := startImport(t, f, "user-1", "x.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
		done := f.waitFor(t, job.ID, "user-1", failed)
		text := done.Error + string(done.Result)
		for _, leak := range []string{"password", "admin", "10.0.0.5", "5432", "pq:", "index out of range"} {
			if strings.Contains(text, leak) {
				t.Fatalf("%s: %q leaked:\n%s", name, leak, text)
			}
		}
		if !done.HasLog {
			t.Fatalf("%s: a failed import has a log", name)
		}
	}
}
