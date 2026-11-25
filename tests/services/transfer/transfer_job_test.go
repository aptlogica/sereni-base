// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package transfer_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/email"
	storageinterfaces "github.com/aptlogica/sereni-base/internal/providers/storage/interfaces"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	transfer "github.com/aptlogica/sereni-base/internal/services/transfer"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/sbase"
)

// memoryJobStore keeps export jobs in memory
type memoryJobStore struct {
	mu   sync.Mutex
	jobs map[string]tenant.TransferJob
}

func newMemoryJobStore() *memoryJobStore {
	return &memoryJobStore{jobs: map[string]tenant.TransferJob{}}
}

func (m *memoryJobStore) Create(_ context.Context, _ string, job tenant.TransferJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *memoryJobStore) Get(_ context.Context, _, id string) (tenant.TransferJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return tenant.TransferJob{}, app_errors.TransferJobNotFound
	}
	return job, nil
}

func (m *memoryJobStore) Update(_ context.Context, _, id string, fields map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[id]
	for k, v := range fields {
		switch k {
		case "status":
			job.Status = v.(string)
		case "progress":
			job.Progress = v.(int)
		case "file_name":
			job.FileName = v.(string)
		case "base_id":
			job.BaseID = v.(string)
		case "base_title":
			job.BaseTitle = v.(string)
		case "result":
			job.Result = v.(string)
		case "file_url":
			job.FileURL = v.(string)
		case "size_bytes":
			job.SizeBytes = v.(int64)
		case "error":
			job.Error = v.(string)
		case "expires_at":
			t := v.(time.Time)
			job.ExpiresAt = &t
		case "last_modified_time":
			job.UpdatedAt = v.(time.Time)
		}
	}
	m.jobs[id] = job
	return nil
}

func (m *memoryJobStore) List(_ context.Context, _ string) ([]tenant.TransferJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]tenant.TransferJob, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out, nil
}

func (m *memoryJobStore) Delete(_ context.Context, _, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.jobs, id)
	return nil
}

func (m *memoryJobStore) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.jobs)
}

// fakeExporter stands in for the base transfer service: it builds a small model (the SereniBase adapter then
// writes the zip), or fails
type fakeExporter struct {
	interfaces.BaseTransferService
	// description makes the model, and so the zip, bigger
	description string
	fail        error
	// gate holds the export until it is closed
	gate    chan struct{}
	opts    dto.ExportBaseOptions
	mu      sync.Mutex
	started int

	// import side: what the job gave to ImportBase, and how it should answer. realImport, when set, runs the import
	// on a real service instead, so a failure comes with the step it happened in.
	realImport interfaces.BaseTransferService
	importErr  error
	importGate chan struct{}
	imports    []importRecord
}

// importRecord is one call of ImportBase
type importRecord struct {
	title, workspace, user string
	opts                   dto.ImportBaseOptions
}

func (f *fakeExporter) ImportBase(ctx context.Context, schema, workspaceID, userID string, file *dto.BaseExportFile, opts dto.ImportBaseOptions) (dto.ImportBaseResult, error) {
	if f.realImport != nil {
		return f.realImport.ImportBase(ctx, schema, workspaceID, userID, file, opts)
	}
	f.mu.Lock()
	f.imports = append(f.imports, importRecord{title: file.Base.Title, workspace: workspaceID, user: userID, opts: opts})
	err, gate := f.importErr, f.importGate
	f.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return dto.ImportBaseResult{}, ctx.Err()
		}
	}
	if err != nil {
		return dto.ImportBaseResult{}, err
	}
	title := opts.Title
	if title == "" {
		title = file.Base.Title
	}
	return dto.ImportBaseResult{BaseID: "new-base-1", Title: title, Tables: len(file.Schema.Tables), Rows: 7}, nil
}

func (f *fakeExporter) importCalls() []importRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]importRecord(nil), f.imports...)
}

func (f *fakeExporter) ExportBase(ctx context.Context, _, _ string, opts dto.ExportBaseOptions) (*dto.BaseExportFile, error) {
	f.mu.Lock()
	f.opts = opts
	f.started++
	f.mu.Unlock()
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.fail != nil {
		return nil, f.fail
	}
	f.mu.Lock()
	description := f.description
	f.mu.Unlock()
	return &dto.BaseExportFile{Base: dto.BaseExportBase{Title: "My CRM / 2026", Description: description}}, nil
}

func (f *fakeExporter) setDescription(d string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.description = d
}

// zipTitle opens a zip written by the adapter and returns the title of base.sbase.json
func zipTitle(t *testing.T, data []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	for _, f := range zr.File {
		if f.Name != "base.sbase.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer rc.Close()
		var doc struct {
			Base struct {
				Title string `json:"title"`
			} `json:"base"`
		}
		if err := json.NewDecoder(rc).Decode(&doc); err != nil {
			t.Fatal(err)
		}
		return doc.Base.Title
	}
	t.Fatal("the zip has no base.sbase.json")
	return ""
}

// randomText is text that does not shrink much when zipped
func randomText(n int) string {
	b := make([]byte, n/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (f *fakeExporter) startedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.started
}

type fakeBases struct {
	interfaces.BaseManagementService
}

func (fakeBases) GetBaseByID(_ context.Context, _, id string) (tenant.Base, error) {
	if id == "missing" {
		return tenant.Base{}, app_errors.BaseNotFound
	}
	return tenant.Base{Title: "My CRM / 2026", WorkspaceID: "ws-1"}, nil
}

// recordingMailer collects the emails
type recordingMailer struct {
	mu   sync.Mutex
	sent []email.EmailJob
}

func (r *recordingMailer) Start(int) {}
func (r *recordingMailer) Stop()     {}
func (r *recordingMailer) Enqueue(j email.EmailJob) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, j)
}

func (r *recordingMailer) emails() []email.EmailJob {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]email.EmailJob(nil), r.sent...)
}

type jobFixture struct {
	svc      interfaces.BaseTransferJobService
	store    *memoryJobStore
	files    transfer.TransferFileStore
	exporter *fakeExporter
	mailer   *recordingMailer
	dir      string
}

func newJobFixture(t *testing.T, cfg transfer.TransferJobSettings, lookup transfer.UserEmailLookup) *jobFixture {
	t.Helper()
	f := &jobFixture{store: newMemoryJobStore(), exporter: &fakeExporter{}, mailer: &recordingMailer{}, dir: t.TempDir()}
	f.files = transfer.NewLocalTransferFileStore(f.dir)
	f.svc = f.newService(t, cfg, lookup)
	return f
}

// newService builds a service on the fixture's store and file store
func (f *jobFixture) newService(t *testing.T, cfg transfer.TransferJobSettings, lookup transfer.UserEmailLookup) interfaces.BaseTransferJobService {
	t.Helper()
	if cfg.DownloadURLTemplate == "" {
		cfg.DownloadURLTemplate = "https://app.example.com/exports/%s"
	}
	if lookup == nil {
		lookup = func(context.Context, string, string) (string, error) { return "owner@example.com", nil }
	}
	svc := transfer.NewBaseTransferJobServiceWithStore(f.store, f.files, f.exporter, fakeBases{}, f.mailer, lookup, cfg)
	t.Cleanup(func() { svc.(interface{ Stop() }).Stop() })
	return svc
}

// waitFor polls the job until done says so
func (f *jobFixture) waitFor(t *testing.T, id, user string, done func(dto.TransferJobResponse) bool) dto.TransferJobResponse {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		job, err := f.svc.Get(context.Background(), constant.MasterDatabase, id, user)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if done(job) {
			return job
		}
		if time.Now().After(deadline) {
			t.Fatalf("the job did not get there in time: %+v", job)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// waitEmails waits until the mailer has n emails
func (f *jobFixture) waitEmails(t *testing.T, n int) []email.EmailJob {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if mails := f.mailer.emails(); len(mails) >= n {
			return mails
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %d emails, got %d", n, len(f.mailer.emails()))
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func completed(j dto.TransferJobResponse) bool { return j.Status == "completed" }
func failed(j dto.TransferJobResponse) bool    { return j.Status == "failed" }

func start(t *testing.T, f *jobFixture, user string) dto.TransferJobResponse {
	t.Helper()
	return startBase(t, f.svc, user, "base-1")
}

func startBase(t *testing.T, svc interfaces.BaseTransferJobService, user, base string) dto.TransferJobResponse {
	t.Helper()
	job, err := svc.StartExport(context.Background(), constant.MasterDatabase, base, user, dto.ExportBaseOptions{
		IncludeData:        true,
		IncludeAttachments: true,
		IncludeAutomations: true,
	})
	if err != nil {
		t.Fatalf("StartExport: %v", err)
	}
	return job
}

func TestTransferJobRunsInTheBackgroundAndCanBeDownloaded(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.gate = make(chan struct{})

	job := start(t, f, "user-1")
	if job.Status != "queued" || job.Type != "export" || job.ID == "" || job.BaseTitle != "My CRM / 2026" || job.DownloadURL != "" {
		t.Fatalf("a new job is queued and has nothing to download yet: %+v", job)
	}
	// while it runs the download is refused
	if _, _, err := f.svc.Open(context.Background(), constant.MasterDatabase, job.ID, "user-1"); !errors.Is(err, app_errors.TransferJobNotReady) {
		t.Fatalf("Open of a running job = %v, want TransferJobNotReady", err)
	}

	close(f.exporter.gate)
	done := f.waitFor(t, job.ID, "user-1", completed)
	if done.FileName != "My_CRM_2026.sbase.zip" || done.SizeBytes <= 0 ||
		done.ExpiresAt == nil || done.DownloadURL != "/api/v1/export-job/"+job.ID+"/download" {
		t.Fatalf("completed job: %+v", done)
	}
	if done.Progress != 100 {
		t.Fatalf("a completed job is at 100 percent, got %d", done.Progress)
	}
	// the job always exports data, files and the cover; automations follow the request
	if !f.exporter.opts.IncludeData || !f.exporter.opts.IncludeAttachments || !f.exporter.opts.IncludeAutomations {
		t.Fatalf("export options: %+v", f.exporter.opts)
	}

	file, _, err := f.svc.Open(context.Background(), constant.MasterDatabase, job.ID, "user-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer file.Close()
	data, _ := io.ReadAll(file)
	if int64(len(data)) != done.SizeBytes {
		t.Fatalf("downloaded %d bytes, the job says %d", len(data), done.SizeBytes)
	}
	// the adapter wrote a real SereniBase zip
	if title := zipTitle(t, data); title != "My CRM / 2026" {
		t.Fatalf("title in base.sbase.json = %q", title)
	}
	// the zip is stored under the job id and the download name, never under raw user text
	if _, err := os.Stat(filepath.Join(f.dir, "exports__"+job.ID+"__My_CRM_2026.sbase.zip")); err != nil {
		t.Fatalf("zip file: %v", err)
	}
}

// The email carries the link and never the file, however small or large the zip is
func TestTransferJobEmailsOnlyTheLinkNeverTheFile(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	small := start(t, f, "user-1")
	f.waitFor(t, small.ID, "user-1", completed)
	mails := f.waitEmails(t, 1)
	if mails[0].To != "owner@example.com" {
		t.Fatalf("emails: %+v", mails)
	}
	if !strings.Contains(mails[0].Body, "https://app.example.com/exports/"+small.ID) {
		t.Fatalf("the email must link to the download page: %s", mails[0].Body)
	}

	// a large zip is the same: a link, no attachment
	f.exporter.setDescription(randomText(200000))
	large := start(t, f, "user-1")
	done := f.waitFor(t, large.ID, "user-1", completed)
	if done.SizeBytes < 50000 {
		t.Fatalf("the zip should be large: %d bytes", done.SizeBytes)
	}
	mails = f.waitEmails(t, 2)
	if len(mails) != 2 || !strings.Contains(mails[1].Body, large.ID) {
		t.Fatalf("emails: %+v", mails)
	}
}

func TestTransferJobWithoutAnEmailAddressStillCompletes(t *testing.T) {
	noEmail := func(context.Context, string, string) (string, error) { return "", errors.New("no such user") }
	f := newJobFixture(t, transfer.TransferJobSettings{}, noEmail)
	job := start(t, f, "user-1")
	f.waitFor(t, job.ID, "user-1", completed)
	time.Sleep(50 * time.Millisecond)
	if n := len(f.mailer.emails()); n != 0 {
		t.Fatalf("%d emails sent to a user without an address", n)
	}
}

func TestTransferJobFailureIsReportedWithoutInternalDetails(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.fail = errors.New("pq: password authentication failed for user admin")
	job := start(t, f, "user-1")
	done := f.waitFor(t, job.ID, "user-1", failed)
	if strings.Contains(done.Error, "pq:") || strings.Contains(done.Error, "password") || done.Error == "" {
		t.Fatalf("the message for the user must not carry internals: %q", done.Error)
	}
	if done.DownloadURL != "" || len(f.mailer.emails()) != 0 {
		t.Fatalf("a failed export has no download and sends no email: %+v", done)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "exports__"+job.ID+"__My_CRM_2026.sbase.zip")); !os.IsNotExist(err) {
		t.Fatalf("no zip may be left behind: %v", err)
	}
	if _, _, err := f.svc.Open(context.Background(), constant.MasterDatabase, job.ID, "user-1"); !errors.Is(err, app_errors.TransferJobNotReady) {
		t.Fatalf("Open of a failed job = %v", err)
	}
}

func TestTransferJobIsPrivateToItsOwner(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	job := start(t, f, "user-1")
	f.waitFor(t, job.ID, "user-1", completed)

	for _, who := range []string{"user-2", ""} {
		if _, err := f.svc.Get(context.Background(), constant.MasterDatabase, job.ID, who); !errors.Is(err, app_errors.TransferJobNotFound) {
			t.Fatalf("Get as %q = %v, want TransferJobNotFound", who, err)
		}
		if _, _, err := f.svc.Open(context.Background(), constant.MasterDatabase, job.ID, who); !errors.Is(err, app_errors.TransferJobNotFound) {
			t.Fatalf("Open as %q = %v, want TransferJobNotFound", who, err)
		}
	}
	if _, err := f.svc.Get(context.Background(), constant.MasterDatabase, "nope", "user-1"); !errors.Is(err, app_errors.TransferJobNotFound) {
		t.Fatalf("an unknown job = %v", err)
	}
}

func TestStartExportRefusesABaseThatDoesNotExist(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	if _, err := f.svc.StartExport(context.Background(), constant.MasterDatabase, "missing", "user-1", dto.ExportBaseOptions{}); !errors.Is(err, app_errors.BaseNotFound) {
		t.Fatalf("StartExport = %v, want BaseNotFound", err)
	}
	if f.store.count() != 0 {
		t.Fatal("no job may be stored for a missing base")
	}
}

func TestExpiredExportCannotBeDownloadedAndIsSweptAway(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{TTL: 50 * time.Millisecond}, nil)
	job := start(t, f, "user-1")
	f.waitFor(t, job.ID, "user-1", completed)
	time.Sleep(80 * time.Millisecond)

	got, err := f.svc.Get(context.Background(), constant.MasterDatabase, job.ID, "user-1")
	if err != nil || got.Status != "expired" || got.DownloadURL != "" {
		t.Fatalf("an expired job reports it: %+v, %v", got, err)
	}
	if _, _, err := f.svc.Open(context.Background(), constant.MasterDatabase, job.ID, "user-1"); !errors.Is(err, app_errors.TransferJobExpired) {
		t.Fatalf("Open of an expired job = %v", err)
	}

	f.svc.(interface{ SweepOnce(context.Context) }).SweepOnce(context.Background())
	if _, err := os.Stat(filepath.Join(f.dir, "exports__"+job.ID+"__My_CRM_2026.sbase.zip")); !os.IsNotExist(err) {
		t.Fatalf("the sweep must delete the file: %v", err)
	}
	if _, err := f.svc.Get(context.Background(), constant.MasterDatabase, job.ID, "user-1"); !errors.Is(err, app_errors.TransferJobNotFound) {
		t.Fatalf("the sweep must delete the record: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Restart
// ---------------------------------------------------------------------------

func TestRecoverFailsTheJobsARestartInterrupted(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	for id, status := range map[string]string{"queued-job": "queued", "running-job": "running", "done-job": "completed"} {
		_ = f.store.Create(context.Background(), constant.MasterDatabase, tenant.TransferJob{ID: id, UserID: "user-1", Status: status})
	}
	// a half-written zip of an interrupted job
	_, _ = f.files.Put(context.Background(), "exports/running-job.zip", strings.NewReader("half"), -1)

	f.svc.(interface{ Recover(context.Context) }).Recover(context.Background())
	for id, want := range map[string]string{"queued-job": "failed", "running-job": "failed", "done-job": "completed"} {
		job, _ := f.store.Get(context.Background(), constant.MasterDatabase, id)
		if job.Status != want {
			t.Fatalf("%s is %s, want %s", id, job.Status, want)
		}
	}
	if job, _ := f.store.Get(context.Background(), constant.MasterDatabase, "running-job"); !strings.Contains(job.Error, "restarted") || job.ExpiresAt == nil {
		t.Fatalf("the user is told why, and the record is cleaned up later: %+v", job)
	}
	if _, err := f.files.Open(context.Background(), "exports/running-job.zip"); !errors.Is(err, transfer.ErrTransferFileMissing) {
		t.Fatalf("the half-written zip must be deleted: %v", err)
	}
}

// Stopping the server interrupts a running export without calling it failed: the next start does that (Recover)
func TestStoppingTheServerLeavesARunningJobForTheNextStart(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.gate = make(chan struct{}) // the export never finishes
	job := start(t, f, "user-1")
	f.waitFor(t, job.ID, "user-1", func(j dto.TransferJobResponse) bool { return j.Status == "running" })

	f.svc.(interface{ Stop() }).Stop()
	got, _ := f.store.Get(context.Background(), constant.MasterDatabase, job.ID)
	if got.Status != "running" {
		t.Fatalf("status after stop = %s", got.Status)
	}
	if len(f.mailer.emails()) != 0 {
		t.Fatal("an interrupted export sends nothing")
	}
}

// ---------------------------------------------------------------------------
// Limits
// ---------------------------------------------------------------------------

func TestUserMayHaveTwoActiveExportsAndAskingAgainReusesTheRunningOne(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.gate = make(chan struct{}) // keeps the exports active

	first := startBase(t, f.svc, "user-1", "base-1")
	again := startBase(t, f.svc, "user-1", "base-1")
	if again.ID != first.ID || !again.Reused || first.Reused {
		t.Fatalf("a second click on the same base must give the same job: first %+v, again %+v", first, again)
	}

	second := startBase(t, f.svc, "user-1", "base-2")
	if second.ID == first.ID {
		t.Fatal("another base is another export")
	}
	// a third at the same time is refused
	if _, err := f.svc.StartExport(context.Background(), constant.MasterDatabase, "base-3", "user-1", dto.ExportBaseOptions{}); !errors.Is(err, app_errors.TransferLimitReached) {
		t.Fatalf("third export = %v, want TransferLimitReached", err)
	}
	// but asking again for one of the two is still fine
	if got := startBase(t, f.svc, "user-1", "base-2"); got.ID != second.ID || !got.Reused {
		t.Fatalf("got %+v", got)
	}
	// another user has their own two
	if other := startBase(t, f.svc, "user-2", "base-1"); other.ID == first.ID {
		t.Fatal("jobs are not shared between users")
	}

	// when one finishes the user can start another
	close(f.exporter.gate)
	f.waitFor(t, first.ID, "user-1", completed)
	f.waitFor(t, second.ID, "user-1", completed)
	third := startBase(t, f.svc, "user-1", "base-3")
	if third.ID == first.ID || third.ID == second.ID {
		t.Fatalf("a finished export is not reused: %+v", third)
	}
	// and the same base again after it finished is a new export
	if again := startBase(t, f.svc, "user-1", "base-1"); again.ID == first.ID {
		t.Fatalf("a finished export is not reused: %+v", again)
	}
}

// Two clicks at the very same moment must not make two exports
func TestSimultaneousClicksMakeOneExport(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.gate = make(chan struct{})

	ids := make(chan string, 20)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job, err := f.svc.StartExport(context.Background(), constant.MasterDatabase, "base-1", "user-1", dto.ExportBaseOptions{})
			if err != nil {
				t.Errorf("StartExport: %v", err)
				return
			}
			ids <- job.ID
		}()
	}
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		seen[id] = true
	}
	if len(seen) != 1 || f.store.count() != 1 {
		t.Fatalf("%d distinct jobs returned, %d stored; want 1", len(seen), f.store.count())
	}
	close(f.exporter.gate)
}

// There is no limit on how many exports run at once: every one starts right away
func TestEveryExportStartsAtOnce(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.gate = make(chan struct{})
	users := []string{"user-a", "user-b", "user-c", "user-d", "user-e", "user-f"}
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = startBase(t, f.svc, u, "base-1").ID
	}
	deadline := time.Now().Add(3 * time.Second)
	for f.exporter.startedCount() < len(users) {
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d exports started", f.exporter.startedCount(), len(users))
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(f.exporter.gate)
	for i, id := range ids {
		f.waitFor(t, id, users[i], completed)
	}
}

// ---------------------------------------------------------------------------
// File stores
// ---------------------------------------------------------------------------

func TestLocalTransferFileStore(t *testing.T) {
	dir := t.TempDir()
	files := transfer.NewLocalTransferFileStore(dir)
	ctx := context.Background()

	if _, err := files.Put(ctx, "exports/a.zip", strings.NewReader("hello"), -1); err != nil {
		t.Fatal(err)
	}
	r, err := files.Open(ctx, "exports/a.zip")
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := io.ReadAll(r); string(data) != "hello" {
		t.Fatalf("read %q", data)
	}
	_ = r.Close()

	if _, err := files.Open(ctx, "exports/none.zip"); !errors.Is(err, transfer.ErrTransferFileMissing) {
		t.Fatalf("Open of a missing file = %v", err)
	}
	if err := files.Delete(ctx, "exports/a.zip"); err != nil {
		t.Fatal(err)
	}
	if err := files.Delete(ctx, "exports/a.zip"); err != nil {
		t.Fatalf("deleting a file that is gone is not an error: %v", err)
	}
	// a key can never point outside the folder
	if _, err := files.Put(ctx, "../../escape.zip", strings.NewReader("x"), -1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "..__..__escape.zip")); err != nil {
		t.Fatalf("the file must be inside the folder: %v", err)
	}
	// a failed upload leaves nothing behind
	if _, err := files.Put(ctx, "exports/b.zip", io.MultiReader(strings.NewReader("par"), errReader{}), -1); err == nil {
		t.Fatal("a reader that fails must fail the upload")
	}
	if _, err := files.Open(ctx, "exports/b.zip"); !errors.Is(err, transfer.ErrTransferFileMissing) {
		t.Fatalf("half a file must not be visible: %v", err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

// fakeStorage stands in for the storage service
type fakeStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
	// failWith makes every call fail (the service is down)
	failWith error
}

func (f *fakeStorage) Upload(_ context.Context, name string, r io.Reader, _ int64, _ string) (storageinterfaces.UploadResponse, error) {
	if f.failWith != nil {
		return storageinterfaces.UploadResponse{}, f.failWith
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return storageinterfaces.UploadResponse{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.objects == nil {
		f.objects = map[string][]byte{}
	}
	f.objects[name] = data
	// the storage service answers with the link of the file, as it does for an attachment
	return storageinterfaces.UploadResponse{Path: name, Url: "https://files.example.com/bucket/" + name}, nil
}

func (f *fakeStorage) Download(_ context.Context, name string) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return nil, f.failWith
	}
	data, ok := f.objects[name]
	if !ok {
		return nil, errors.New("storage http: download failed with status 404")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeStorage) Delete(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return f.failWith
	}
	if _, ok := f.objects[name]; !ok {
		return errors.New("storage http: delete failed with status 404")
	}
	delete(f.objects, name)
	return nil
}

func (f *fakeStorage) Exists(_ context.Context, name string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return false, f.failWith
	}
	_, ok := f.objects[name]
	return ok, nil
}

func TestStorageServiceTransferFileStore(t *testing.T) {
	storage := &fakeStorage{}
	files := transfer.NewStorageTransferFileStore(storage)
	ctx := context.Background()

	link, err := files.Put(ctx, "exports/a.zip", strings.NewReader("hello"), -1)
	if err != nil {
		t.Fatal(err)
	}
	if link != "https://files.example.com/bucket/exports/a.zip" {
		t.Fatalf("Put must return the link the storage service gave: %q", link)
	}
	r, err := files.Open(ctx, "exports/a.zip")
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := io.ReadAll(r); string(data) != "hello" {
		t.Fatalf("read %q", data)
	}
	_ = r.Close()

	// a file that is not there is "missing"; a storage service that is down is an error, not "expired"
	if _, err := files.Open(ctx, "exports/none.zip"); !errors.Is(err, transfer.ErrTransferFileMissing) {
		t.Fatalf("Open of a missing file = %v", err)
	}
	if err := files.Delete(ctx, "exports/none.zip"); err != nil {
		t.Fatalf("deleting a file that is gone is not an error: %v", err)
	}
	storage.failWith = errors.New("connection refused")
	if _, err := files.Open(ctx, "exports/a.zip"); err == nil || errors.Is(err, transfer.ErrTransferFileMissing) {
		t.Fatalf("an unreachable storage service must not look like an expired export: %v", err)
	}
	if err := files.Delete(ctx, "exports/a.zip"); err == nil {
		t.Fatal("a failed delete must be reported so the sweep tries again")
	}
}

// The service works the same on the storage service: the zip goes there and comes back from there.
func TestTransferJobOnTheStorageService(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.svc.(interface{ Stop() }).Stop() // the fixture's own service uses the local folder; use only the one below
	storage := &fakeStorage{}
	f.files = transfer.NewStorageTransferFileStore(storage)
	f.svc = f.newService(t, transfer.TransferJobSettings{}, nil)

	job := startBase(t, f.svc, "user-1", "base-9")
	f.waitFor(t, job.ID, "user-1", completed)
	storage.mu.Lock()
	data := storage.objects["exports/"+job.ID+"/My_CRM_2026.sbase.zip"]
	storage.mu.Unlock()
	if title := zipTitle(t, data); title != "My CRM / 2026" {
		t.Fatalf("the storage service holds a zip with title %q", title)
	}

	// when the storage service is down the export fails cleanly
	storage.failWith = errors.New("connection refused")
	broken := startBase(t, f.svc, "user-1", "base-10")
	if got := f.waitFor(t, broken.ID, "user-1", failed); strings.Contains(got.Error, "refused") {
		t.Fatalf("the user must not see internals: %q", got.Error)
	}
}

// The link the storage service gives to the zip when it is uploaded (the same kind of link an attachment gets) goes
// into the job, the status response and the email
func TestExportJobUsesTheLinkTheStorageServiceGave(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.svc.(interface{ Stop() }).Stop()
	f.files = transfer.NewStorageTransferFileStore(&fakeStorage{})
	f.svc = f.newService(t, transfer.TransferJobSettings{}, nil)

	job := startBase(t, f.svc, "user-1", "base-1")
	if job.FileURL != "" {
		t.Fatalf("there is no file yet: %+v", job)
	}
	done := f.waitFor(t, job.ID, "user-1", completed)
	want := "https://files.example.com/bucket/exports/" + job.ID + "/My_CRM_2026.sbase.zip"
	if done.FileURL != want {
		t.Fatalf("file_url = %q, want %q", done.FileURL, want)
	}
	if stored, _ := f.store.Get(context.Background(), constant.MasterDatabase, job.ID); stored.FileURL != want {
		t.Fatalf("the link must be saved in the job: %q", stored.FileURL)
	}
	mails := f.waitEmails(t, 1)
	if !strings.Contains(mails[0].Body, want) {
		t.Fatalf("the email must carry the link to the file: %s", mails[0].Body)
	}
	if strings.Contains(mails[0].Body, "app.example.com") {
		t.Fatal("the web app page must not be used when the storage service gave a link")
	}
}

// A local folder gives no link: the email falls back to the page of the web app
func TestWithoutAFileLinkTheEmailLinksToTheWebAppPage(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	job := start(t, f, "user-1")
	done := f.waitFor(t, job.ID, "user-1", completed)
	if done.FileURL != "" {
		t.Fatalf("a local folder has no link: %+v", done)
	}
	mails := f.waitEmails(t, 1)
	if !strings.Contains(mails[0].Body, "https://app.example.com/exports/"+job.ID) {
		t.Fatalf("the email must link to the web app page: %s", mails[0].Body)
	}
}

// ---------------------------------------------------------------------------
// Import as a background job
// ---------------------------------------------------------------------------

const uploadedJSON = `{"base":{"title":"Uploaded"},"schema":{"tables":[]},"data":null}`

func startImport(t *testing.T, f *jobFixture, user, filename string, body []byte, opts dto.ImportBaseOptions) dto.TransferJobResponse {
	t.Helper()
	job, err := f.svc.StartImport(context.Background(), constant.MasterDatabase, "ws-1", user, dto.ImportUpload{File: bytes.NewReader(body), Filename: filename, Size: int64(len(body))}, opts)
	if err != nil {
		t.Fatalf("StartImport: %v", err)
	}
	return job
}

// filesInStore lists the files kept by the local file store
func filesInStore(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestImportJobCreatesTheBaseInTheBackground(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.importGate = make(chan struct{})

	job := startImport(t, f, "user-1", "crm.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{Title: "My copy", IncludeData: true, IncludeAutomations: true})
	if job.Type != "import" || job.Status != "queued" || job.FileName != "crm.sbase.json" || job.BaseID != "" {
		t.Fatalf("a new import job: %+v", job)
	}
	if got := filesInStore(t, f.dir); len(got) != 1 {
		t.Fatalf("the uploaded file must be kept for the job: %v", got)
	}

	close(f.exporter.importGate)
	done := f.waitFor(t, job.ID, "user-1", completed)
	if done.BaseID != "new-base-1" || done.BaseTitle != "My copy" || done.DownloadURL != "" || done.FileURL != "" {
		t.Fatalf("a completed import points at the new base and has nothing to download: %+v", done)
	}
	var result struct {
		BaseID string `json:"base_id"`
		Title  string `json:"title"`
		Rows   int    `json:"rows"`
	}
	if err := json.Unmarshal(done.Result, &result); err != nil || result.BaseID != "new-base-1" || result.Rows != 7 {
		t.Fatalf("result = %s (%v)", done.Result, err)
	}

	// the job read the kept file through the adapter and gave the model and the options to the core service
	calls := f.exporter.importCalls()
	if len(calls) != 1 || calls[0].title != "Uploaded" || calls[0].workspace != "ws-1" || calls[0].user != "user-1" {
		t.Fatalf("ImportBase calls: %+v", calls)
	}
	if calls[0].opts.Title != "My copy" || !calls[0].opts.IncludeData || !calls[0].opts.IncludeAutomations {
		t.Fatalf("the options decided at the request must reach the import: %+v", calls[0].opts)
	}
	// the kept file is not needed any more
	if got := filesInStore(t, f.dir); len(got) != 0 {
		t.Fatalf("the uploaded file must be deleted after the import: %v", got)
	}
}

// An import goes on after the request that started it has ended
func TestImportJobSurvivesTheEndOfTheRequest(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.importGate = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	job, err := f.svc.StartImport(ctx, constant.MasterDatabase, "ws-1", "user-1", dto.ImportUpload{File: strings.NewReader(uploadedJSON), Filename: "x.sbase.json", Size: int64(len(uploadedJSON))}, dto.ImportBaseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	cancel() // the HTTP request is over
	close(f.exporter.importGate)
	f.waitFor(t, job.ID, "user-1", completed)
}

func TestImportJobReadsAZipUpload(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	var zipped bytes.Buffer
	model := &dto.BaseExportFile{Base: dto.BaseExportBase{Title: "Zipped"}, Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{}}}
	if err := sbase.New().Export(context.Background(), model, adapter.ExportOptions{IncludeData: true}, &zipped); err != nil {
		t.Fatal(err)
	}
	job := startImport(t, f, "user-1", "crm.sbase.zip", zipped.Bytes(), dto.ImportBaseOptions{IncludeData: true, IncludeAttachments: true})
	f.waitFor(t, job.ID, "user-1", completed)

	calls := f.exporter.importCalls()
	if len(calls) != 1 || calls[0].title != "Zipped" {
		t.Fatalf("ImportBase calls: %+v", calls)
	}
	if calls[0].opts.Attachments == nil {
		t.Fatal("a zip upload gives the import the files it carries")
	}
}

func TestImportJobFailuresAreReportedForTheUser(t *testing.T) {
	// a file that is not valid: the user sees every problem
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.importErr = &dto.ImportValidationError{Problems: []string{"schema.tables[0]: key \"t_a\" is already used", "data: bad row"}}
	job := startImport(t, f, "user-1", "x.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	done := f.waitFor(t, job.ID, "user-1", failed)
	if done.Error != "the base file is not valid" || !strings.Contains(string(done.Result), "is already used") || !strings.Contains(string(done.Result), "bad row") {
		t.Fatalf("validation failure: %+v / %s", done, done.Result)
	}

	// anything else: a short message, no internals
	f2 := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f2.exporter.importErr = errors.New("pq: password authentication failed for user admin")
	job = startImport(t, f2, "user-1", "x.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	done = f2.waitFor(t, job.ID, "user-1", failed)
	if strings.Contains(done.Error, "pq:") || strings.Contains(done.Error, "password") || !strings.Contains(done.Error, "nothing was created") {
		t.Fatalf("internal failure: %+v", done)
	}
	// the failed import keeps a log for the user, and it holds no internals either
	if strings.Contains(string(done.Result), "pq:") || strings.Contains(string(done.Result), "password") || !done.HasLog {
		t.Fatalf("the log of an internal failure: %+v / %s", done, done.Result)
	}

	// a file the adapter cannot read: its own words
	f3 := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	job = startImport(t, f3, "user-1", "x.sbase.json", []byte("{not json"), dto.ImportBaseOptions{})
	done = f3.waitFor(t, job.ID, "user-1", failed)
	if !strings.Contains(done.Error, "the file is not valid JSON") {
		t.Fatalf("unreadable file: %+v", done)
	}
	if len(f3.exporter.importCalls()) != 0 {
		t.Fatal("nothing may be created from a file that cannot be read")
	}

	// whatever happened, the kept file is gone
	for _, fx := range []*jobFixture{f, f2, f3} {
		if got := filesInStore(t, fx.dir); len(got) != 0 {
			t.Fatalf("uploaded file left behind: %v", got)
		}
	}
}

// Imports and exports have a limit each; a refused import leaves nothing behind
func TestUserMayHaveTwoActiveImportsAndOneDoesNotBlockExports(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.importGate = make(chan struct{})

	a := startImport(t, f, "user-1", "a.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	b := startImport(t, f, "user-1", "b.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	if a.ID == b.ID {
		t.Fatal("every import is its own job")
	}
	_, err := f.svc.StartImport(context.Background(), constant.MasterDatabase, "ws-1", "user-1", dto.ImportUpload{File: strings.NewReader(uploadedJSON), Filename: "c.sbase.json", Size: int64(len(uploadedJSON))}, dto.ImportBaseOptions{})
	if !errors.Is(err, app_errors.TransferLimitReached) {
		t.Fatalf("third import = %v, want TransferLimitReached", err)
	}
	if got := filesInStore(t, f.dir); len(got) != 2 {
		t.Fatalf("a refused import must not keep its file: %v", got)
	}

	// exports are counted on their own, and so are other users
	if e := startBase(t, f.svc, "user-1", "base-1"); e.Type != "export" {
		t.Fatalf("export: %+v", e)
	}
	if other := startImport(t, f, "user-2", "d.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{}); other.Status != "queued" {
		t.Fatalf("another user: %+v", other)
	}

	// when one finishes, the user can start another
	close(f.exporter.importGate)
	f.waitFor(t, a.ID, "user-1", completed)
	f.waitFor(t, b.ID, "user-1", completed)
	startImport(t, f, "user-1", "e.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
}

func TestImportJobIsPrivateAndHasNothingToDownload(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	job := startImport(t, f, "user-1", "x.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	f.waitFor(t, job.ID, "user-1", completed)

	if _, err := f.svc.Get(context.Background(), constant.MasterDatabase, job.ID, "user-2"); !errors.Is(err, app_errors.TransferJobNotFound) {
		t.Fatalf("someone else's import = %v", err)
	}
	if _, _, err := f.svc.Open(context.Background(), constant.MasterDatabase, job.ID, "user-1"); !errors.Is(err, app_errors.TransferJobNotFound) {
		t.Fatalf("an import has no file to download: %v", err)
	}
}

// A restart fails the imports it interrupted and deletes the file they kept
func TestRecoverFailsInterruptedImportsAndDeletesTheirFiles(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	_ = f.store.Create(context.Background(), constant.MasterDatabase, tenant.TransferJob{ID: "imp-1", Type: "import", UserID: "user-1", Status: "running"})
	_, _ = f.files.Put(context.Background(), "imports/imp-1", strings.NewReader(uploadedJSON), -1)

	f.svc.(interface{ Recover(context.Context) }).Recover(context.Background())
	job, _ := f.store.Get(context.Background(), constant.MasterDatabase, "imp-1")
	if job.Status != "failed" || !strings.Contains(job.Error, "restarted") || job.ExpiresAt == nil {
		t.Fatalf("interrupted import: %+v", job)
	}
	if _, err := f.files.Open(context.Background(), "imports/imp-1"); !errors.Is(err, transfer.ErrTransferFileMissing) {
		t.Fatalf("the kept file must be deleted: %v", err)
	}
}

// Stopping the server leaves a running import for the next start to fail, and creates nothing
func TestStoppingTheServerInterruptsARunningImport(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.importGate = make(chan struct{}) // never opened
	job := startImport(t, f, "user-1", "x.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	f.waitFor(t, job.ID, "user-1", func(j dto.TransferJobResponse) bool { return j.Status == "running" })

	f.svc.(interface{ Stop() }).Stop()
	got, _ := f.store.Get(context.Background(), constant.MasterDatabase, job.ID)
	if got.Status != "running" || got.BaseID != "" {
		t.Fatalf("after stop: %+v", got)
	}
}

// The jobs of a workspace are listed for their owner only, newest first, with their progress
func TestListWorkspaceJobsListsOnlyTheUsersJobs(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	mine := startBase(t, f.svc, "user-1", "base-1")
	other := startBase(t, f.svc, "user-2", "base-2")
	f.waitFor(t, mine.ID, "user-1", completed)
	f.waitFor(t, other.ID, "user-2", completed)

	lister := f.svc.(interface {
		ListWorkspaceJobs(context.Context, string, string, string) ([]dto.TransferJobResponse, error)
	})
	jobs, err := lister.ListWorkspaceJobs(context.Background(), constant.MasterDatabase, "ws-1", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != mine.ID || jobs[0].Progress != 100 || jobs[0].WorkspaceID != "ws-1" {
		t.Fatalf("user-1 must see only their own job, with its progress: %+v", jobs)
	}
	none, _ := lister.ListWorkspaceJobs(context.Background(), constant.MasterDatabase, "ws-2", "user-1")
	if len(none) != 0 {
		t.Fatalf("another workspace has no jobs: %+v", none)
	}
}

// a failed import keeps a log the user can download: the header, the error, and the problems of an invalid file
func TestFailedImportJobKeepsAnErrorLog(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.importErr = &dto.ImportValidationError{Problems: []string{"schema.tables[0]: key \"t_a\" is already used", "data: bad row"}}
	job := startImport(t, f, "user-1", "Sales.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	done := f.waitFor(t, job.ID, "user-1", failed)
	if !done.HasLog {
		t.Fatalf("a failed import has a log: %+v", done)
	}
	var result struct {
		ErrorLog string   `json:"error_log"`
		Problems []string `json:"problems"`
	}
	if err := json.Unmarshal(done.Result, &result); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Import Issues Report", "Format: Sereni Base", "File: Sales.sbase.json", "Result: failed", "ERROR:", "the base file is not valid", "PROBLEMS (2):", `1. schema.tables[0]: key "t_a" is already used`, "2. data: bad row"} {
		if !strings.Contains(result.ErrorLog, want) {
			t.Fatalf("the log lacks %q:\n%s", want, result.ErrorLog)
		}
	}
	if len(result.Problems) != 2 {
		t.Fatalf("the problems are still listed apart: %v", result.Problems)
	}
}

// the log of a finished import with nothing to report does not exist
func TestFinishedImportJobWithoutIssuesHasNoLog(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	job := startImport(t, f, "user-1", "x.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	done := f.waitFor(t, job.ID, "user-1", completed)
	if done.HasLog || done.SkippedRows != 0 || done.WarningCount != 0 {
		t.Fatalf("no log expected: %+v", done)
	}
}

// ---------------------------------------------------------------------------
// Removing a job from the list
// ---------------------------------------------------------------------------

func TestFinishedExportCanBeRemovedWithItsFile(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	job := start(t, f, "user-1")
	done := f.waitFor(t, job.ID, "user-1", completed)
	if _, _, err := f.svc.Open(context.Background(), constant.MasterDatabase, job.ID, "user-1"); err != nil {
		t.Fatalf("the file is there before: %v", err)
	}
	if err := f.svc.Remove(context.Background(), constant.MasterDatabase, done.ID, "user-1"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := f.svc.Get(context.Background(), constant.MasterDatabase, job.ID, "user-1"); !errors.Is(err, app_errors.TransferJobNotFound) {
		t.Fatalf("the job is gone: %v", err)
	}
	if _, err := f.files.Open(context.Background(), "exports/"+job.ID+".zip"); err == nil {
		t.Fatal("the file is deleted with the job")
	}
}

func TestRunningJobCannotBeRemoved(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.gate = make(chan struct{})
	job := start(t, f, "user-1")
	f.waitFor(t, job.ID, "user-1", func(j dto.TransferJobResponse) bool { return j.Status == "running" })
	if err := f.svc.Remove(context.Background(), constant.MasterDatabase, job.ID, "user-1"); !errors.Is(err, app_errors.TransferJobNotReady) {
		t.Fatalf("remove of a running job = %v, want TransferJobNotReady", err)
	}
	close(f.exporter.gate)
}

func TestFailedImportCanBeRemoved(t *testing.T) {
	f := newJobFixture(t, transfer.TransferJobSettings{}, nil)
	f.exporter.importErr = errors.New("boom")
	job := startImport(t, f, "user-1", "x.sbase.json", []byte(uploadedJSON), dto.ImportBaseOptions{})
	f.waitFor(t, job.ID, "user-1", func(j dto.TransferJobResponse) bool { return j.Status == "failed" })
	if err := f.svc.Remove(context.Background(), constant.MasterDatabase, job.ID, "user-1"); err != nil {
		t.Fatalf("remove: %v", err)
	}
}
