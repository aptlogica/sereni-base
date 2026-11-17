// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"io"

	"github.com/aptlogica/sereni-base/internal/dto"
)

// BaseTransferService builds a whole base as a model (dto.BaseExportFile) and creates a base from such a model.
// Reading and writing the file itself is the job of an adapter (internal/services/transfer/adapter).
type BaseTransferService interface {
	// ExportBase reads the structure (and, with opts.IncludeData, the rows) of a base into a file value
	ExportBase(ctx context.Context, schemaName, baseID string, opts dto.ExportBaseOptions) (*dto.BaseExportFile, error)
	// ExportDependencies lists the tables of a base and which table needs which (links, lookups, triggers), so a client
	// can warn before an export that leaves a table out
	ExportDependencies(ctx context.Context, schemaName, baseID string, includeAutomations bool) (*dto.ExportDependencies, error)
	// OpenFile opens a stored file (an attachment or the cover image) by the StoragePath of the model entry that lists
	// it. An export adapter uses it to pack the files into the file it writes (adapter.FileSource).
	OpenFile(ctx context.Context, storagePath string) (io.ReadCloser, error)
	// ImportBase validates the file and creates it as a new base in the workspace. Nothing is left behind on failure.
	ImportBase(ctx context.Context, schemaName, workspaceID, userID string, file *dto.BaseExportFile, opts dto.ImportBaseOptions) (dto.ImportBaseResult, error)
}

// BaseTransferJobService runs base exports and imports in the background, so a slow one does not hold a request open.
// A finished export is a zip kept for a limited time; the user who started it can download it, and gets the link by
// email. An import creates its base in the background; the user follows it by its job.
type BaseTransferJobService interface {
	// StartExport validates the base, stores a queued job and returns at once; a worker (on any server) runs it. The
	// job always exports with data, attachments and the cover; opts only says whether triggers and functions go in.
	// If the user already has an active export of this base, that one is returned (Reused is set) instead of a new
	// one; if they already have the most active exports of other bases, it fails with TransferLimitReached.
	StartExport(ctx context.Context, schemaName, baseID, userID string, opts dto.ExportBaseOptions) (dto.TransferJobResponse, error)
	// StartImport keeps the uploaded file, stores a queued job and returns at once; the import runs in the background.
	// A user may have a limited number of active imports (separate from exports); beyond it, TransferLimitReached.
	StartImport(ctx context.Context, schemaName, workspaceID, userID string, file io.Reader, filename string, size int64, opts dto.ImportBaseOptions) (dto.TransferJobResponse, error)
	// ListWorkspaceJobs returns the user's exports and imports of a workspace, newest first, with their progress
	ListWorkspaceJobs(ctx context.Context, schemaName, workspaceID, userID string) ([]dto.TransferJobResponse, error)
	// Get returns a job of the user (an export or an import); ErrTransferJobNotFound for a job that is missing or belongs to someone else
	Get(ctx context.Context, schemaName, jobID, userID string) (dto.TransferJobResponse, error)
	// Open opens the zip of a completed job of the user. The caller closes it.
	Open(ctx context.Context, schemaName, jobID, userID string) (io.ReadCloser, dto.TransferJobResponse, error)
}
