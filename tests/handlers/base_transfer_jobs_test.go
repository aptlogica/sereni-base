// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/gin-gonic/gin"
)

const jobID = "7d6f3a1e-5c52-4c0e-9d57-0f0f2d0a1b11"

// fakeJobs stands in for the background export service
type fakeJobs struct {
	started    dto.ExportBaseOptions
	startedFor string
	startedBy  string
	err        error
	zipPath    string
	job        dto.TransferJobResponse
	reused     bool
	imported   importCall
}

func (f *fakeJobs) StartExport(_ context.Context, _, baseID, userID string, opts dto.ExportBaseOptions) (dto.TransferJobResponse, error) {
	f.started, f.startedFor, f.startedBy = opts, baseID, userID
	if f.err != nil {
		return dto.TransferJobResponse{}, f.err
	}
	return dto.TransferJobResponse{ID: jobID, Status: "queued", BaseID: baseID, Reused: f.reused}, nil
}

func (f *fakeJobs) ListWorkspaceJobs(_ context.Context, _, _, _ string) ([]dto.TransferJobResponse, error) {
	return nil, f.err
}

func (f *fakeJobs) StartImport(_ context.Context, _, workspaceID, userID string, file io.Reader, filename string, size int64, opts dto.ImportBaseOptions) (dto.TransferJobResponse, error) {
	data, _ := io.ReadAll(file)
	f.imported = importCall{workspaceID: workspaceID, userID: userID, filename: filename, size: size, data: string(data), opts: opts}
	if f.err != nil {
		return dto.TransferJobResponse{}, f.err
	}
	return dto.TransferJobResponse{ID: jobID, Type: "import", Status: "queued"}, nil
}

type importCall struct {
	workspaceID, userID, filename, data string
	size                                int64
	opts                                dto.ImportBaseOptions
}

func (f *fakeJobs) Get(_ context.Context, _, _, _ string) (dto.TransferJobResponse, error) {
	return f.job, f.err
}

func (f *fakeJobs) Open(_ context.Context, _, _, _ string) (io.ReadCloser, dto.TransferJobResponse, error) {
	if f.err != nil {
		return nil, f.job, f.err
	}
	file, err := os.Open(f.zipPath)
	return file, f.job, err
}

func jobRequest(t *testing.T, jobs *fakeJobs, run func(h *handlers.BaseTransferHandler, c *gin.Context), path, param string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	c.Params = gin.Params{{Key: "id", Value: param}}
	c.Set("schema", "s")
	c.Set("user_id", "user-1")
	run(handlers.NewBaseTransferHandlerWithJobs(&capturingTransfer{}, jobs), c)
	return w
}

func TestExportWithDataStartsABackgroundJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jobs := &fakeJobs{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?include_data=true", nil)
	c.Params = gin.Params{{Key: "id", Value: "33cc3ba3-9fac-43d8-866f-0bd1e44548eb"}}
	c.Set("schema", "s")
	c.Set("user_id", "user-1")
	c.Set(handlers.AutomationsAllowedKey, true)
	handlers.NewBaseTransferHandlerWithJobs(&capturingTransfer{}, jobs).ExportBase(c)

	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), jobID) || !strings.Contains(w.Body.String(), "queued") {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	if jobs.startedFor != "33cc3ba3-9fac-43d8-866f-0bd1e44548eb" || jobs.startedBy != "user-1" ||
		!jobs.started.IncludeData || !jobs.started.IncludeAttachments || !jobs.started.IncludeAutomations {
		t.Fatalf("job started with %+v for %s by %s", jobs.started, jobs.startedFor, jobs.startedBy)
	}
}

func startExport(t *testing.T, jobs *fakeJobs) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?include_data=true", nil)
	c.Params = gin.Params{{Key: "id", Value: "33cc3ba3-9fac-43d8-866f-0bd1e44548eb"}}
	c.Set("schema", "s")
	c.Set("user_id", "user-1")
	handlers.NewBaseTransferHandlerWithJobs(&capturingTransfer{}, jobs).ExportBase(c)
	return w
}

// Asking again for a base that is already being exported returns the running job
func TestExportAgainReturnsTheRunningJob(t *testing.T) {
	w := startExport(t, &fakeJobs{reused: true})
	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), `"reused":true`) {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}

// A user who has the most exports active gets 429, not a 500
func TestExportOverTheLimitIsRefusedWith429(t *testing.T) {
	w := startExport(t, &fakeJobs{err: app_errors.TransferLimitReached})
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}

func TestExportWithoutDataIsStillADirectDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jobs := &fakeJobs{}
	svc := &capturingTransfer{file: &dto.BaseExportFile{Base: dto.BaseExportBase{Title: "CRM"}}}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "33cc3ba3-9fac-43d8-866f-0bd1e44548eb"}}
	c.Set("schema", "s")
	handlers.NewBaseTransferHandlerWithJobs(svc, jobs).ExportBase(c)
	if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Disposition"), ".sbase.json") || jobs.startedFor != "" {
		t.Fatalf("status %d, headers %v, job started for %q", w.Code, w.Header(), jobs.startedFor)
	}
}

func TestTransferJobStatusAndErrors(t *testing.T) {
	get := func(h *handlers.BaseTransferHandler, c *gin.Context) { h.GetExportJob(c) }

	ok := jobRequest(t, &fakeJobs{job: dto.TransferJobResponse{ID: jobID, Type: "export", Status: "running"}}, get, "/", jobID)
	if ok.Code != http.StatusOK || !strings.Contains(ok.Body.String(), `"status":"running"`) {
		t.Fatalf("status %d, body %s", ok.Code, ok.Body.String())
	}
	// someone else's job and an unknown one look the same
	if w := jobRequest(t, &fakeJobs{err: app_errors.TransferJobNotFound}, get, "/", jobID); w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
	// an id that is not a uuid never reaches the service
	if w := jobRequest(t, &fakeJobs{}, get, "/", "../../etc/passwd"); w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
}

func TestTransferJobDownload(t *testing.T) {
	download := func(h *handlers.BaseTransferHandler, c *gin.Context) { h.DownloadExportJob(c) }
	zipPath := filepath.Join(t.TempDir(), "x.zip")
	if err := os.WriteFile(zipPath, []byte("zip-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	w := jobRequest(t, &fakeJobs{zipPath: zipPath, job: dto.TransferJobResponse{FileName: "CRM.sbase.zip"}}, download, "/", jobID)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/zip" || w.Body.String() != "zip-bytes" ||
		!strings.Contains(w.Header().Get("Content-Disposition"), `filename="CRM.sbase.zip"`) {
		t.Fatalf("status %d, headers %v, body %q", w.Code, w.Header(), w.Body.String())
	}

	for err, want := range map[error]int{
		app_errors.TransferJobNotFound: http.StatusNotFound,
		app_errors.TransferJobNotReady: http.StatusConflict,
		app_errors.TransferJobExpired:  http.StatusGone,
	} {
		if w := jobRequest(t, &fakeJobs{err: err}, download, "/", jobID); w.Code != want {
			t.Fatalf("%v: status %d, want %d", err, w.Code, want)
		}
	}
}

func TestTransferJobEndpointsNeedBackgroundExportsToBeSetUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: jobID}}
	handlers.NewBaseTransferHandler(&capturingTransfer{}).GetExportJob(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Import as a background job
// ---------------------------------------------------------------------------

// a minimal valid file; the handler reads it only to refuse a broken one at once
const importFileJSON = `{"base":{"title":"From file"},"schema":{"tables":[]},"data":null}`

func importWithJobs(t *testing.T, jobs *fakeJobs, svc *capturingTransfer, fields map[string]string, filename, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var payload bytes.Buffer
	mw := multipart.NewWriter(&payload)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	part, _ := mw.CreateFormFile("file", filename)
	_, _ = io.WriteString(part, body)
	_ = mw.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", &payload)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Set("schema", "s")
	c.Set("user_id", "user-1")
	handlers.NewBaseTransferHandlerWithJobs(svc, jobs).ImportBase(c)
	return w
}

// An import returns 202 with a job and keeps the uploaded file for the job
func TestImportStartsABackgroundJob(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := importWithJobs(t, jobs, svc, map[string]string{"title": " My copy ", "include_data": "false", "include_automations": "true"}, "crm.sbase.json", importFileJSON)

	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), jobID) || !strings.Contains(w.Body.String(), `"type":"import"`) {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	got := jobs.imported
	if got.workspaceID != "ws-1" || got.userID != "user-1" || got.filename != "crm.sbase.json" {
		t.Fatalf("job started with %+v", got)
	}
	if got.data != importFileJSON || got.size != int64(len(importFileJSON)) {
		t.Fatalf("the job must get the whole uploaded file, got %d bytes: %q", got.size, got.data)
	}
	if got.opts.Title != "My copy" || got.opts.IncludeData || !got.opts.IncludeAutomations {
		t.Fatalf("options: %+v", got.opts)
	}
	if svc.called {
		t.Fatal("the base must not be created during the request")
	}
}

// A broken file is still refused at once, before any job exists
func TestImportOfABrokenFileIsRefusedBeforeAJobStarts(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := importWithJobs(t, jobs, svc, nil, "x.sbase.json", "{not json")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "the file is not valid JSON") {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	if jobs.imported.workspaceID != "" {
		t.Fatal("no job may start for a broken file")
	}
}

func TestImportOverTheLimitIsRefusedWith429(t *testing.T) {
	w := importWithJobs(t, &fakeJobs{err: app_errors.TransferLimitReached}, &capturingTransfer{}, nil, "x.sbase.json", importFileJSON)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}

// Without background jobs set up the import still runs during the request, as before
func TestImportWithoutJobsRunsDuringTheRequest(t *testing.T) {
	svc := importRequest(t, nil)
	if !svc.called {
		t.Fatal("the import must run")
	}
}

func TestImportJobStatusIsOnlyForImports(t *testing.T) {
	get := func(h *handlers.BaseTransferHandler, c *gin.Context) { h.GetImportJob(c) }

	done := jobRequest(t, &fakeJobs{job: dto.TransferJobResponse{ID: jobID, Type: "import", Status: "completed", BaseID: "b-1"}}, get, "/", jobID)
	if done.Code != http.StatusOK || !strings.Contains(done.Body.String(), `"base_id":"b-1"`) {
		t.Fatalf("status %d, body %s", done.Code, done.Body.String())
	}
	// an export job id on the import route, and the other way round, look like jobs that do not exist
	if w := jobRequest(t, &fakeJobs{job: dto.TransferJobResponse{ID: jobID, Type: "export", Status: "completed"}}, get, "/", jobID); w.Code != http.StatusNotFound {
		t.Fatalf("export job on the import route: status %d", w.Code)
	}
	export := func(h *handlers.BaseTransferHandler, c *gin.Context) { h.GetExportJob(c) }
	if w := jobRequest(t, &fakeJobs{job: dto.TransferJobResponse{ID: jobID, Type: "import", Status: "completed"}}, export, "/", jobID); w.Code != http.StatusNotFound {
		t.Fatalf("import job on the export route: status %d", w.Code)
	}
	if w := jobRequest(t, &fakeJobs{err: app_errors.TransferJobNotFound}, get, "/", jobID); w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
}
