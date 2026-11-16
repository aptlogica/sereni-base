// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	"github.com/google/uuid"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/email"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/sbase"
	"github.com/aptlogica/sereni-base/internal/services/transfer/formats"
)

const (
	defaultExportTTL        = 24 * time.Hour
	defaultExportTimeout    = 2 * time.Hour
	defaultExportsPerUser   = 2
	defaultExportSweepEvery = time.Minute
	defaultProgressEvery    = 2 * time.Second

	// statusExpired is what a completed job reports once its file may no longer be downloaded; it is never stored
	statusExpired = "expired"

	exportFailureMessage = "the export failed; try again, and contact support if it keeps failing"
	importFailureMessage = "the import failed and nothing was created; check the file and try again, and contact support if it keeps failing"
	importInvalidMessage = "the base file is not valid"
	jobRestartMessage    = "the server restarted while the job was running; start it again"
)

// TransferJobSettings tunes the background exports. Zero values fall back to the defaults.
type TransferJobSettings struct {
	// TTL is how long a finished export can be downloaded before it is deleted
	TTL time.Duration
	// Timeout stops one export that runs too long
	Timeout time.Duration
	// MaxPerUser is how many exports one user may have active (queued or running) at once
	MaxPerUser int
	// DownloadURLTemplate is the page of the web app where the user downloads the export; %s is the job id. It goes
	// into the email only when the file store has no link of its own (see TransferFileLinker).
	DownloadURLTemplate string
	// SweepEvery is how often expired exports are deleted
	SweepEvery time.Duration
	// ProgressEvery is how often the progress of a running job is saved to the database
	ProgressEvery time.Duration
}

func (c TransferJobSettings) withDefaults() TransferJobSettings {
	if c.TTL <= 0 {
		c.TTL = defaultExportTTL
	}
	if c.Timeout <= 0 {
		c.Timeout = defaultExportTimeout
	}
	if c.MaxPerUser <= 0 {
		c.MaxPerUser = defaultExportsPerUser
	}
	if c.SweepEvery <= 0 {
		c.SweepEvery = defaultExportSweepEvery
	}
	if c.ProgressEvery <= 0 {
		c.ProgressEvery = defaultProgressEvery
	}
	return c
}

// UserEmailLookup returns the email address of a user
type UserEmailLookup func(ctx context.Context, schemaName, userID string) (string, error)

// transferJobService runs the exports in this process: StartExport saves the job and starts it at once, in its own
// goroutine. It is meant for a single server; the job rows only survive a restart as a record (see Recover).
type transferJobService struct {
	store     TransferJobStore
	files     TransferFileStore
	transfer  interfaces.BaseTransferService
	bases     interfaces.BaseManagementService
	mailer    email.EmailService
	userEmail UserEmailLookup
	cfg       TransferJobSettings
	// format writes the zip: the export goes through the adapter of the SereniBase format
	format adapter.Adapter
	now    func() time.Time

	// start serializes the per-user check and the insert, so two clicks at the same moment cannot both pass it
	start  sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewBaseTransferJobService builds the background export service on the database: the jobs are rows of the
// transfer_jobs table and the zips live in files. It fails the jobs a restart interrupted and starts the sweep that
// deletes expired exports. mailer and userEmail may be nil; no email is then sent.
func NewBaseTransferJobService(
	repo *pkg.DatabaseService,
	files TransferFileStore,
	transfer interfaces.BaseTransferService,
	bases interfaces.BaseManagementService,
	mailer email.EmailService,
	userEmail UserEmailLookup,
	cfg TransferJobSettings,
) interfaces.BaseTransferJobService {
	svc := newTransferJobService(NewDBTransferJobStore(repo), files, transfer, bases, mailer, userEmail, cfg)
	svc.Recover(svc.ctx)
	svc.wg.Add(1)
	go svc.sweep()
	return svc
}

// NewBaseTransferJobServiceWithStore is NewBaseTransferJobService on any job store, without the restart recovery and
// the sweep loop (tests call those steps themselves).
func NewBaseTransferJobServiceWithStore(
	store TransferJobStore,
	files TransferFileStore,
	transfer interfaces.BaseTransferService,
	bases interfaces.BaseManagementService,
	mailer email.EmailService,
	userEmail UserEmailLookup,
	cfg TransferJobSettings,
) interfaces.BaseTransferJobService {
	return newTransferJobService(store, files, transfer, bases, mailer, userEmail, cfg)
}

func newTransferJobService(
	store TransferJobStore,
	files TransferFileStore,
	transfer interfaces.BaseTransferService,
	bases interfaces.BaseManagementService,
	mailer email.EmailService,
	userEmail UserEmailLookup,
	cfg TransferJobSettings,
) *transferJobService {
	ctx, cancel := context.WithCancel(context.Background())
	format, _ := formats.Registry().Get(sbase.Name)
	return &transferJobService{
		store: store, files: files, transfer: transfer, bases: bases, mailer: mailer, userEmail: userEmail,
		cfg: cfg.withDefaults(), format: format, now: time.Now, ctx: ctx, cancel: cancel,
	}
}

// Stop interrupts the running exports and stops the sweep. An interrupted job stays "running" in the table until
// the next start, which fails it (Recover).
func (s *transferJobService) Stop() {
	s.cancel()
	s.wg.Wait()
}

// exportFileKey is where the zip of an export is kept. The last part of the key is the name the file is downloaded
// with, so the storage link (the one in the email) ends with the same name as the download from the app.
func (s *transferJobService) exportFileKey(job tenant.TransferJob) string {
	name := job.FileName
	if name == "" {
		// a job that has not completed has no file name yet; it is the one the finished job gets
		name, _ = s.format.Describe(adapter.ExportOptions{IncludeData: true}, job.BaseTitle)
	}
	return "exports/" + job.ID + "/" + name
}

// legacyExportFileKey is where exports made by an earlier version were kept, under the job id only
func legacyExportFileKey(jobID string) string { return "exports/" + jobID + ".zip" }

// deleteExportFile removes the zip of an export, wherever it was kept
func (s *transferJobService) deleteExportFile(ctx context.Context, job tenant.TransferJob) error {
	err := s.files.Delete(ctx, s.exportFileKey(job))
	if err == nil {
		_ = s.files.Delete(ctx, legacyExportFileKey(job.ID))
	}
	return err
}

// importFileKey is where the file an import was started with is kept until the job has read it
func importFileKey(jobID string) string { return "imports/" + jobID }

// jobType is the type of a job; a row made before imports existed has none and is an export
func jobType(j tenant.TransferJob) string {
	if j.Type == "" {
		return tenant.TransferJobTypeExport
	}
	return j.Type
}

func (s *transferJobService) StartExport(ctx context.Context, schemaName, baseID, userID string, opts dto.ExportBaseOptions) (dto.TransferJobResponse, error) {
	base, err := s.bases.GetBaseByID(ctx, schemaName, baseID)
	if err != nil {
		return dto.TransferJobResponse{}, err
	}

	s.start.Lock()
	defer s.start.Unlock()

	jobs, err := s.store.List(ctx, schemaName)
	if err != nil {
		return dto.TransferJobResponse{}, err
	}
	active := 0
	for _, j := range jobs {
		if j.UserID != userID || jobType(j) != tenant.TransferJobTypeExport || !isActive(j) {
			continue
		}
		if j.BaseID == baseID {
			// the user clicked again: they get the export that is already on its way
			resp := s.response(j)
			resp.Reused = true
			return resp, nil
		}
		active++
	}
	if active >= s.cfg.MaxPerUser {
		return dto.TransferJobResponse{}, app_errors.TransferLimitReached
	}

	now := s.now().UTC()
	job := tenant.TransferJob{
		ID:        uuid.NewString(),
		Type:      tenant.TransferJobTypeExport,
		BaseID:    baseID,
		BaseTitle: base.Title,
		// kept so the jobs of a workspace can be listed
		WorkspaceID: base.WorkspaceID,
		UserID:      userID,
		Status:      tenant.TransferJobQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Create(ctx, schemaName, job); err != nil {
		return dto.TransferJobResponse{}, err
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.process(schemaName, job, opts.IncludeAutomations)
	}()
	return s.response(job), nil
}

// isActive reports whether a job is waiting or running
func isActive(j tenant.TransferJob) bool {
	return j.Status == tenant.TransferJobQueued || j.Status == tenant.TransferJobRunning
}

// StartImport keeps the uploaded file, stores a queued job and returns at once; the import runs in the background.
// A user may have as many active imports as active exports (MaxPerUser), counted separately.
func (s *transferJobService) StartImport(ctx context.Context, schemaName, workspaceID, userID string, file io.Reader, filename string, size int64, opts dto.ImportBaseOptions) (dto.TransferJobResponse, error) {
	id := uuid.NewString()
	// The job reads the file after this request has ended, so it is kept. It is stored before the limit is checked so
	// that a big upload does not hold the lock; it is removed again when the job is refused.
	if _, err := s.files.Put(ctx, importFileKey(id), file, size); err != nil {
		return dto.TransferJobResponse{}, err
	}

	now := s.now().UTC()
	job := tenant.TransferJob{
		ID:          id,
		Type:        tenant.TransferJobTypeImport,
		WorkspaceID: workspaceID,
		BaseTitle:   opts.Title,
		FileName:    filename,
		SizeBytes:   size,
		UserID:      userID,
		Status:      tenant.TransferJobQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.createWithinLimit(ctx, schemaName, job); err != nil {
		_ = s.files.Delete(context.Background(), importFileKey(id))
		return dto.TransferJobResponse{}, err
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processImport(schemaName, job, opts)
	}()
	return s.response(job), nil
}

// createWithinLimit stores an import job unless the user already has the most active imports
func (s *transferJobService) createWithinLimit(ctx context.Context, schemaName string, job tenant.TransferJob) error {
	s.start.Lock()
	defer s.start.Unlock()
	jobs, err := s.store.List(ctx, schemaName)
	if err != nil {
		return err
	}
	active := 0
	for _, j := range jobs {
		if j.UserID == job.UserID && jobType(j) == job.Type && isActive(j) {
			active++
		}
	}
	if active >= s.cfg.MaxPerUser {
		return app_errors.TransferLimitReached
	}
	return s.store.Create(ctx, schemaName, job)
}

// ListWorkspaceJobs returns the user's exports and imports of the workspace, newest first. Expired jobs are left out
// (their records are being deleted); failed ones stay until they expire, so the user can read why.
func (s *transferJobService) ListWorkspaceJobs(ctx context.Context, schemaName, workspaceID, userID string) ([]dto.TransferJobResponse, error) {
	jobs, err := s.store.List(ctx, schemaName)
	if err != nil {
		return nil, err
	}
	var mine []tenant.TransferJob
	for _, j := range jobs {
		if j.UserID == userID && j.WorkspaceID == workspaceID {
			mine = append(mine, j)
		}
	}
	sort.Slice(mine, func(a, b int) bool { return mine[a].CreatedAt.After(mine[b].CreatedAt) })
	out := make([]dto.TransferJobResponse, 0, len(mine))
	for _, j := range mine {
		resp := s.response(j)
		if resp.Status == statusExpired {
			continue
		}
		out = append(out, resp)
	}
	return out, nil
}

func (s *transferJobService) Get(ctx context.Context, schemaName, jobID, userID string) (dto.TransferJobResponse, error) {
	job, err := s.owned(ctx, schemaName, jobID, userID)
	if err != nil {
		return dto.TransferJobResponse{}, err
	}
	return s.response(job), nil
}

func (s *transferJobService) Open(ctx context.Context, schemaName, jobID, userID string) (io.ReadCloser, dto.TransferJobResponse, error) {
	job, err := s.owned(ctx, schemaName, jobID, userID)
	if err != nil {
		return nil, dto.TransferJobResponse{}, err
	}
	if jobType(job) != tenant.TransferJobTypeExport {
		return nil, dto.TransferJobResponse{}, app_errors.TransferJobNotFound // only an export has a file to download
	}
	resp := s.response(job)
	switch resp.Status {
	case statusExpired:
		return nil, resp, app_errors.TransferJobExpired
	case tenant.TransferJobCompleted:
	default:
		return nil, resp, app_errors.TransferJobNotReady
	}
	f, err := s.files.Open(ctx, s.exportFileKey(job))
	if errors.Is(err, ErrTransferFileMissing) {
		f, err = s.files.Open(ctx, legacyExportFileKey(job.ID)) // an export of an earlier version
	}
	if errors.Is(err, ErrTransferFileMissing) {
		// the file is gone (deleted by hand, or the storage was replaced): it cannot be downloaded any more
		return nil, resp, app_errors.TransferJobExpired
	}
	if err != nil {
		return nil, resp, err
	}
	return f, resp, nil
}

// owned returns the job when it belongs to the user. Someone else's job looks the same as a missing one.
func (s *transferJobService) owned(ctx context.Context, schemaName, jobID, userID string) (tenant.TransferJob, error) {
	job, err := s.store.Get(ctx, schemaName, jobID)
	if err != nil {
		return tenant.TransferJob{}, err
	}
	if userID == "" || job.UserID != userID {
		return tenant.TransferJob{}, app_errors.TransferJobNotFound
	}
	return job, nil
}

func (s *transferJobService) response(job tenant.TransferJob) dto.TransferJobResponse {
	resp := dto.TransferJobResponse{
		ID:          job.ID,
		Type:        jobType(job),
		BaseID:      job.BaseID,
		BaseTitle:   job.BaseTitle,
		WorkspaceID: job.WorkspaceID,
		Status:      job.Status,
		Progress:    job.Progress,
		FileName:    job.FileName,
		SizeBytes:   job.SizeBytes,
		Error:       job.Error,
		ExpiresAt:   job.ExpiresAt,
		CreatedAt:   job.CreatedAt,
	}
	if job.Result != "" {
		resp.Result = json.RawMessage(job.Result)
	}
	if job.Status == tenant.TransferJobCompleted {
		resp.Progress = 100
		if job.ExpiresAt != nil && !job.ExpiresAt.After(s.now()) {
			resp.Status = statusExpired
		} else if resp.Type == tenant.TransferJobTypeExport {
			resp.DownloadURL = fmt.Sprintf("/api/v1/export-job/%s/download", job.ID)
			resp.FileURL = job.FileURL
		}
	}
	return resp
}

// ---------------------------------------------------------------------------
// Running a job
// ---------------------------------------------------------------------------

// process runs one job: builds the zip into the file store, then tells the user. includeAutomations is the user's
// permission to read triggers and functions, decided when they clicked Export; it is passed in memory.
func (s *transferJobService) process(schemaName string, job tenant.TransferJob, includeAutomations bool) {
	ctx, cancel := context.WithTimeout(s.ctx, s.cfg.Timeout)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			s.fail(schemaName, job, fmt.Errorf("panic: %v", r))
		}
	}()

	logger.Get().Info().Str("job_id", job.ID).Str("base_id", job.BaseID).Msg("base export started")
	s.update(schemaName, job.ID, map[string]interface{}{"status": tenant.TransferJobRunning, "progress": 0})
	report, stopProgress := s.progressReporter(schemaName, job.ID, 0, 100)
	defer stopProgress()
	ctx = WithProgress(ctx, report)

	size, fileName, fileURL, err := s.build(ctx, schemaName, job, includeAutomations)
	stopProgress() // before the final update, so a late save can never overwrite it
	if err != nil {
		if s.ctx.Err() != nil {
			return // the server is stopping; the next start fails the job (Recover)
		}
		s.fail(schemaName, job, err)
		return
	}

	job.FileName, job.SizeBytes, job.FileURL = fileName, size, fileURL
	logger.Get().Info().Str("job_id", job.ID).Int64("size_bytes", size).Msg("base export completed")
	s.update(schemaName, job.ID, map[string]interface{}{
		"status":     tenant.TransferJobCompleted,
		"progress":   100,
		"file_name":  fileName,
		"file_url":   job.FileURL,
		"size_bytes": size,
		"expires_at": s.now().UTC().Add(s.cfg.TTL),
	})
	s.notify(schemaName, job)
}

// importFileError is what is wrong with the file of an import, in words for the user
type importFileError struct{ msg string }

func (e *importFileError) Error() string { return e.msg }

// processImport runs one import: reads the kept file into a model through the adapter and creates the base. The
// options (the user's permission to create triggers among them) were decided when the user started the import.
func (s *transferJobService) processImport(schemaName string, job tenant.TransferJob, opts dto.ImportBaseOptions) {
	ctx, cancel := context.WithTimeout(s.ctx, s.cfg.Timeout)
	defer cancel()
	// the uploaded file is not needed once the job has read it, whatever the outcome
	defer func() { _ = s.files.Delete(context.Background(), importFileKey(job.ID)) }()
	defer func() {
		if r := recover(); r != nil {
			s.fail(schemaName, job, fmt.Errorf("panic: %v", r))
		}
	}()

	logger.Get().Info().Str("job_id", job.ID).Str("workspace_id", job.WorkspaceID).Msg("base import started")
	s.update(schemaName, job.ID, map[string]interface{}{"status": tenant.TransferJobRunning, "progress": 0})
	// reading the file takes the first 5 percent; creating the base takes the rest
	report, stopProgress := s.progressReporter(schemaName, job.ID, 5, 100)
	defer stopProgress()
	ctx = WithProgress(ctx, report)

	result, err := s.runImport(ctx, schemaName, job, opts)
	stopProgress() // before the final update, so a late save can never overwrite it
	if err != nil {
		if s.ctx.Err() != nil {
			return // the server is stopping; the next start fails the job (Recover)
		}
		s.fail(schemaName, job, err)
		return
	}

	raw, _ := json.Marshal(result)
	logger.Get().Info().Str("job_id", job.ID).Str("base_id", result.BaseID).Msg("base import completed")
	s.update(schemaName, job.ID, map[string]interface{}{
		"status":     tenant.TransferJobCompleted,
		"progress":   100,
		"base_id":    result.BaseID,
		"base_title": result.Title,
		"result":     string(raw),
		"expires_at": s.now().UTC().Add(s.cfg.TTL),
	})
}

// runImport reads the kept file and creates the base from it
func (s *transferJobService) runImport(ctx context.Context, schemaName string, job tenant.TransferJob, opts dto.ImportBaseOptions) (dto.ImportBaseResult, error) {
	// A zip is read by position, so the file is copied from the file store into a temporary local file first
	tmp, err := os.CreateTemp("", "sbase-import-*")
	if err != nil {
		return dto.ImportBaseResult{}, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	src, err := s.files.Open(ctx, importFileKey(job.ID))
	if err != nil {
		return dto.ImportBaseResult{}, err
	}
	size, err := io.Copy(tmp, src)
	_ = src.Close()
	if err != nil {
		return dto.ImportBaseResult{}, err
	}
	// plain JSON is read from the current position, so go back to the start of the copy
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return dto.ImportBaseResult{}, err
	}
	reportProgress(ctx, 0) // the file is read; the reporter starts counting from here

	read, err := s.format.Import(ctx, tmp, size, adapter.ImportOptions{Filename: job.FileName, MaxBytes: adapter.DefaultMaxImportBytes})
	if err != nil {
		return dto.ImportBaseResult{}, &importFileError{msg: err.Error()}
	}
	opts.Attachments = read.Attachments
	return s.transfer.ImportBase(ctx, schemaName, job.WorkspaceID, job.UserID, read.File, opts)
}

// progressFiles counts the files the adapter reads while it packs the zip and reports 70..99 percent
type progressFiles struct {
	src   adapter.FileSource
	ctx   context.Context
	total int
	mu    sync.Mutex
	done  int
}

func (p *progressFiles) OpenFile(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	rc, err := p.src.OpenFile(ctx, storagePath)
	if err == nil {
		p.mu.Lock()
		p.done++
		done := p.done
		p.mu.Unlock()
		reportFraction(p.ctx, 70, 99, min(done, p.total), p.total)
	}
	return rc, err
}

// progressReporter follows the progress of a running job. The export or import calls the returned function as often as
// it likes, which only remembers the latest value (scaled to from..to percent, never going down). A scheduler saves it
// to the database every cfg.ProgressEvery, and only when it changed. The returned stop ends the scheduler and waits for
// it, so after stop returns nothing more is written; it can be called more than once.
func (s *transferJobService) progressReporter(schemaName, jobID string, from, to int) (ProgressFunc, func()) {
	var (
		mu     sync.Mutex
		latest = -1
		saved  = -1
	)
	report := func(percent int) {
		// the job is completed only by its final update, never by a step
		scaled := min(from+(to-from)*percent/100, 99)
		mu.Lock()
		if scaled > latest {
			latest = scaled
		}
		mu.Unlock()
	}
	flush := func() {
		mu.Lock()
		value := latest
		changed := value > saved
		if changed {
			saved = value
		}
		mu.Unlock()
		if changed {
			s.update(schemaName, jobID, map[string]interface{}{"progress": value})
		}
	}

	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		ticker := time.NewTicker(s.cfg.ProgressEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				flush()
			case <-done:
				return
			}
		}
	}()
	var once sync.Once
	return report, func() {
		once.Do(func() {
			close(done)
			<-finished
		})
	}
}

// countingWriter passes everything on and counts the bytes, which are the size of the zip
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// build reads the base and streams its zip into the file store. It returns the size, the name the download gets and
// the link the file store gave to the zip (empty when it has none).
func (s *transferJobService) build(ctx context.Context, schemaName string, job tenant.TransferJob, includeAutomations bool) (int64, string, string, error) {
	file, err := s.transfer.ExportBase(ctx, schemaName, job.BaseID, dto.ExportBaseOptions{
		IncludeData:        true,
		IncludeAttachments: true,
		IncludeAutomations: includeAutomations,
	})
	if err != nil {
		return 0, "", "", err
	}

	// the zip takes the last part of the progress: each stored file that is packed moves it on
	exportOpts := adapter.ExportOptions{IncludeData: true, Files: s.transfer}
	if file.Data != nil && len(file.Data.Attachments) > 0 {
		exportOpts.Files = &progressFiles{src: s.transfer, ctx: ctx, total: len(file.Data.Attachments)}
	}
	reportProgress(ctx, 70)

	// The zip is written into a pipe that the file store reads from, so it never sits whole on disk or in memory
	pr, pw := io.Pipe()
	type putResult struct {
		url string
		err error
	}
	putDone := make(chan putResult, 1)
	go func() {
		url, err := s.files.Put(ctx, s.exportFileKey(job), pr, -1)
		pr.CloseWithError(err) // a failed upload stops the writer instead of leaving it blocked
		putDone <- putResult{url: url, err: err}
	}()
	counted := &countingWriter{w: pw}
	// the core service reads the base into the model; the adapter writes the model as a file (with the files it lists)
	writeErr := s.format.Export(ctx, file, exportOpts, counted)
	pw.CloseWithError(writeErr) // nil ends the stream normally
	put := <-putDone
	if writeErr != nil || put.err != nil {
		_ = s.deleteExportFile(context.Background(), job)
		if writeErr != nil {
			return 0, "", "", writeErr
		}
		return 0, "", "", put.err
	}

	fileName, _ := s.format.Describe(exportOpts, file.Base.Title)
	return counted.n, fileName, put.url, nil
}

// notify emails the user the link to the finished export. The zip is never attached: it can be large, and mail
// servers often refuse a big message. The export is already downloadable, so a problem here is only logged.
func (s *transferJobService) notify(schemaName string, job tenant.TransferJob) {
	lg := logger.Get()
	if s.mailer == nil || s.userEmail == nil {
		lg.Warn().Str("job_id", job.ID).Msg("export finished but no email is sent: the email service is not set up")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	to, err := s.userEmail(ctx, schemaName, job.UserID)
	if err != nil || strings.TrimSpace(to) == "" {
		lg.Warn().Err(err).Str("job_id", job.ID).Msg("export finished but the user has no email address to send it to")
		return
	}

	// the link to the file in the storage service; without one, the page of the web app (DownloadURLTemplate)
	link := job.FileURL
	if link == "" {
		link = s.cfg.DownloadURLTemplate
		if strings.Contains(link, "%s") {
			link = fmt.Sprintf(link, job.ID)
		}
	}
	if isLocalLink(link) {
		// An attachment has the same problem: the storage service builds its links from STORAGE_SERVER_IP, and the
		// default is localhost, which only works on the machine that runs it.
		lg.Warn().Str("job_id", job.ID).Str("link", link).
			Msg("the export link points at localhost, so it will not open for the user: set STORAGE_SERVER_SCHEME and STORAGE_SERVER_IP of the storage service to its public address")
	}
	content := email.ExportReadyEmail(job.BaseTitle, link, durationText(s.cfg.TTL))
	s.mailer.Enqueue(email.EmailJob{To: to, Subject: content.Subject, Body: content.Body})
	lg.Info().Str("job_id", job.ID).Str("to", maskEmail(to)).
		Msg("export email with the download link handed to the email service (its log says whether it was sent)")
}

// isLocalLink reports whether a link points at the machine it was made on
func isLocalLink(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	}
	return false
}

// maskEmail keeps a log line useful without writing a whole address into it: a***@example.com
func maskEmail(addr string) string {
	at := strings.LastIndex(addr, "@")
	if at <= 0 {
		return "***"
	}
	return addr[:1] + "***" + addr[at:]
}

func durationText(d time.Duration) string {
	if d < time.Hour {
		minutes := max(1, int(d.Round(time.Minute)/time.Minute))
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := int(d.Round(time.Hour) / time.Hour)
	if hours <= 1 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", hours)
}

// fail records why a job failed. The user gets a short message; the real error goes to the log.
func (s *transferJobService) fail(schemaName string, job tenant.TransferJob, cause error) {
	logger.Get().Error().Err(cause).Str("job_id", job.ID).Str("type", jobType(job)).Msg("base transfer failed")
	fields := map[string]interface{}{
		"status":     tenant.TransferJobFailed,
		"error":      exportFailureMessage,
		"expires_at": s.now().UTC().Add(s.cfg.TTL),
	}
	if jobType(job) == tenant.TransferJobTypeImport {
		fields["error"] = importFailureMessage
		var fileErr *importFileError
		var invalid *dto.ImportValidationError
		switch {
		case errors.As(cause, &fileErr):
			// what is wrong with the file is meant for the user
			fields["error"] = fileErr.Error()
		case errors.As(cause, &invalid):
			fields["error"] = importInvalidMessage
			if raw, err := json.Marshal(map[string]interface{}{"problems": invalid.Problems}); err == nil {
				fields["result"] = string(raw)
			}
		}
	} else {
		_ = s.deleteExportFile(context.Background(), job)
	}
	s.update(schemaName, job.ID, fields)
}

func (s *transferJobService) update(schemaName, jobID string, fields map[string]interface{}) {
	fields["last_modified_time"] = s.now().UTC()
	if err := s.store.Update(context.Background(), schemaName, jobID, fields); err != nil {
		logger.Get().Error().Err(err).Str("job_id", jobID).Msg("could not update the export job")
	}
}

// ---------------------------------------------------------------------------
// Restart and cleanup
// ---------------------------------------------------------------------------

// Recover fails the jobs a restart interrupted: their work was lost with the process.
func (s *transferJobService) Recover(ctx context.Context) {
	jobs, err := s.store.List(ctx, constant.MasterDatabase)
	if err != nil {
		logger.Get().Error().Err(err).Msg("could not look for interrupted exports")
		return
	}
	for _, job := range jobs {
		if !isActive(job) {
			continue
		}
		if jobType(job) == tenant.TransferJobTypeImport {
			_ = s.files.Delete(ctx, importFileKey(job.ID)) // the uploaded file is not needed any more
		} else {
			_ = s.deleteExportFile(ctx, job) // a half-written zip
		}
		s.update(constant.MasterDatabase, job.ID, map[string]interface{}{
			"status":     tenant.TransferJobFailed,
			"error":      jobRestartMessage,
			"expires_at": s.now().UTC().Add(s.cfg.TTL),
		})
	}
}

// sweep deletes expired exports every SweepEvery until the service stops.
func (s *transferJobService) sweep() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.SweepEvery)
	defer ticker.Stop()
	for {
		s.SweepOnce(s.ctx)
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// SweepOnce deletes the exports that have expired: the file and the record.
func (s *transferJobService) SweepOnce(ctx context.Context) {
	jobs, err := s.store.List(ctx, constant.MasterDatabase)
	if err != nil {
		if ctx.Err() == nil {
			logger.Get().Error().Err(err).Msg("could not look for expired exports")
		}
		return
	}
	now := s.now()
	for _, job := range jobs {
		if job.ExpiresAt == nil || job.ExpiresAt.After(now) {
			continue
		}
		if jobType(job) == tenant.TransferJobTypeExport {
			if err := s.deleteExportFile(ctx, job); err != nil {
				logger.Get().Warn().Err(err).Str("job_id", job.ID).Msg("could not delete an expired export file; will try again")
				continue
			}
		}
		if err := s.store.Delete(ctx, constant.MasterDatabase, job.ID); err != nil {
			logger.Get().Warn().Err(err).Str("job_id", job.ID).Msg("could not delete an expired export record")
		}
	}
}
