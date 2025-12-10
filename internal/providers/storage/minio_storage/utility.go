package minio_storage

import (
	"context"
	"io"
	"serenibase/internal/providers/storage/interfaces"
	"strings"

	"github.com/minio/minio-go/v7"
)

type MinioStorageProvider struct {
	client *minio.Client
	bucket string
}

func NewLocalStorageService(client *minio.Client, bucket string) interfaces.StorageProvider {
	return &MinioStorageProvider{client: client, bucket: bucket}
}

// Delete implements interfaces.StorageProvider.
func (l *MinioStorageProvider) Delete(ctx context.Context, objectName string) error {
	return l.client.RemoveObject(ctx, l.bucket, objectName, minio.RemoveObjectOptions{})
}

// Download implements interfaces.StorageProvider.
func (l *MinioStorageProvider) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	panic("unimplemented")
}

// Exists implements interfaces.StorageProvider.
func (l *MinioStorageProvider) Exists(ctx context.Context, objectName string) (bool, error) {
	panic("unimplemented")
}

// Upload implements interfaces.StorageProvider.
func (l *MinioStorageProvider) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	objectName = strings.ReplaceAll(objectName, "\\", "/")
	_, err := l.client.PutObject(ctx, l.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	// Return the preview URL for the uploaded asset
	// Construct the full URL: endpoint/bucket/objectName
	endpoint := l.client.EndpointURL()
	scheme := "http"
	if endpoint.Scheme != "" {
		scheme = endpoint.Scheme
	}
	previewURL := scheme + "://" + endpoint.Host + "/" + l.bucket + "/" + objectName
	return previewURL, nil
}
