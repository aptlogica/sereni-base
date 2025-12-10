package interfaces

import (
	"context"
	"io"
)

// StorageProvider abstracts file storage operations across different backends
type StorageProvider interface {
	Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error)
	Download(ctx context.Context, objectName string) (io.ReadCloser, error)
	Delete(ctx context.Context, objectName string) error
	Exists(ctx context.Context, objectName string) (bool, error)
}
