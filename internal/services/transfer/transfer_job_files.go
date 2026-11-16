// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	storageinterfaces "github.com/aptlogica/sereni-base/internal/providers/storage/interfaces"
)

// ErrTransferFileMissing is returned by an TransferFileStore for a file that is not there (it expired, or was deleted)
var ErrTransferFileMissing = fmt.Errorf("export file not found")

// TransferFileStore keeps the finished export zips. Every server must see the same files, so with more than one
// server it has to be shared storage (the storage service); a local folder only works with a single server.
type TransferFileStore interface {
	// Put stores what r delivers under key; size is -1 when it is not known in advance. It returns the link the store
	// gives to the file (for the storage service, the same kind of link an attachment gets when it is uploaded), or ""
	// when the store has none.
	Put(ctx context.Context, key string, r io.Reader, size int64) (url string, err error)
	// Open returns the file, or ErrTransferFileMissing
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the file; a file that is already gone is not an error
	Delete(ctx context.Context, key string) error
}

// ---------------------------------------------------------------------------
// Storage service
// ---------------------------------------------------------------------------

type storageExportFiles struct {
	storage storageinterfaces.StorageProvider
}

// NewStorageTransferFileStore keeps the files in the storage service. The link of a stored file is the one the storage
// service returns when the file is uploaded, which is how an attachment gets its link too. The storage service builds
// it from its own settings (STORAGE_SERVER_SCHEME, STORAGE_SERVER_IP and the bucket), so a deployment sets the public
// address once, there, for attachments and exports alike. Anyone who has the link can download the file (as with an
// attachment), so the name in it is the job id, which cannot be guessed, and the file is deleted when the export
// expires.
func NewStorageTransferFileStore(storage storageinterfaces.StorageProvider) TransferFileStore {
	return &storageExportFiles{storage: storage}
}

func (s *storageExportFiles) Put(ctx context.Context, key string, r io.Reader, size int64) (string, error) {
	resp, err := s.storage.Upload(ctx, key, r, size, "application/zip")
	if err != nil {
		return "", err
	}
	return resp.Url, nil
}

func (s *storageExportFiles) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	body, err := s.storage.Download(ctx, key)
	if err == nil {
		return body, nil
	}
	// the storage service answers every failure the same way, so ask whether the file exists at all
	if exists, existsErr := s.storage.Exists(ctx, key); existsErr == nil && !exists {
		return nil, ErrTransferFileMissing
	}
	return nil, err
}

func (s *storageExportFiles) Delete(ctx context.Context, key string) error {
	err := s.storage.Delete(ctx, key)
	if err == nil {
		return nil
	}
	if exists, existsErr := s.storage.Exists(ctx, key); existsErr == nil && !exists {
		return nil
	}
	return err
}

// ---------------------------------------------------------------------------
// Local folder
// ---------------------------------------------------------------------------

type localExportFiles struct {
	dir string
}

// NewLocalTransferFileStore keeps the export zips in a folder of this machine. Only for a single server.
func NewLocalTransferFileStore(dir string) TransferFileStore {
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "sbase-exports")
	}
	return &localExportFiles{dir: dir}
}

// path maps a key to a file inside dir; it never leaves dir, whatever the key holds. The folders of a key become part
// of the file name, so two exports with the same file name (the same base title) never share a file.
func (l *localExportFiles) path(key string) string {
	flat := strings.ReplaceAll(strings.Trim(key, "/"), "/", "__")
	return filepath.Join(l.dir, filepath.Base(flat))
}

func (l *localExportFiles) Put(_ context.Context, key string, r io.Reader, _ int64) (string, error) {
	if err := os.MkdirAll(l.dir, 0o700); err != nil {
		return "", err
	}
	// written under another name first, so a download never sees half a file
	tmp, err := os.CreateTemp(l.dir, ".upload-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	// a file in a folder of this machine has no link
	return "", os.Rename(tmp.Name(), l.path(key))
}

func (l *localExportFiles) Open(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(l.path(key))
	if os.IsNotExist(err) {
		return nil, ErrTransferFileMissing
	}
	return f, err
}

func (l *localExportFiles) Delete(_ context.Context, key string) error {
	if err := os.Remove(l.path(key)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
