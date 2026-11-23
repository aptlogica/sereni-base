// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package ziputil holds the zip code the file-format adapters share.
package ziputil

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
)

// PackFile copies one stored file into the zip under path and returns the size and checksum of what was actually
// read. A file larger than adapter.MaxAttachmentBytes is an error.
func PackFile(ctx context.Context, zw *zip.Writer, openFile adapter.OpenFileFunc, name, path, storagePath string) (int64, string, error) {
	rc, err := openFile(ctx, storagePath)
	if err != nil {
		return 0, "", fmt.Errorf("read %q: %w", name, err)
	}
	defer rc.Close()

	fw, err := zw.Create(path)
	if err != nil {
		return 0, "", err
	}
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(fw, hash), io.LimitReader(rc, adapter.MaxAttachmentBytes+1))
	if err != nil {
		return 0, "", fmt.Errorf("read %q: %w", name, err)
	}
	if n > adapter.MaxAttachmentBytes {
		return 0, "", fmt.Errorf("%q is larger than %d MB", name, adapter.MaxAttachmentBytes>>20)
	}
	return n, "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

// NewZipSource wraps an opened zip as the attachments of an import: the files of the zip, by path.
func NewZipSource(zr *zip.Reader) *dto.Attachments {
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			files[f.Name] = f
		}
	}
	return &dto.Attachments{
		Has: func(path string) bool {
			_, ok := files[path]
			return ok
		},
		Open: func(path string) (io.ReadCloser, error) {
			f, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("%s is not in the zip", path)
			}
			return f.Open()
		},
	}
}
