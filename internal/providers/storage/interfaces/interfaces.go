// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"io"
)

// UploadResponse represents the response from an upload operation
type UploadResponse struct {
	Message string `json:"message"`
	Path    string `json:"path"`
	Url     string `json:"url"`
}

// StorageProvider abstracts file storage operations across different backends
type StorageProvider interface {
	Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (UploadResponse, error)
	Download(ctx context.Context, objectName string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectName string) error
	Exists(ctx context.Context, objectName string) (bool, error)
}
