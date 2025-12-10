package local_storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"serenibase/internal/config"
	"serenibase/internal/providers/storage/interfaces"
	"strings"
)

type LocalStorageProvider struct {
	path string
}

func NewLocalStorageService(path string) interfaces.StorageProvider {
	return &LocalStorageProvider{path: path}
}

// Delete removes the file at the given objectName.
func (l *LocalStorageProvider) Delete(ctx context.Context, objectName string) error {
	fullPath := filepath.Join(l.path, objectName)
	return os.Remove(fullPath)
}

// Download opens the file for reading.
func (l *LocalStorageProvider) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	fullPath := filepath.Join(l.path, objectName)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// Exists checks if the file exists.
func (l *LocalStorageProvider) Exists(ctx context.Context, objectName string) (bool, error) {
	fullPath := filepath.Join(l.path, objectName)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Upload saves the file to the local path.
func (l *LocalStorageProvider) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	fullPath := filepath.Join(l.path, objectName)
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}
	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = io.CopyN(file, reader, size)
	if err != nil {
		return "", err
	}
	server_url := config.AppConfig.Server.Scheme + "://" + config.AppConfig.Server.Host + ":" + config.AppConfig.Server.Port + "/"
	filePath := server_url + strings.ReplaceAll(fullPath, "\\", "/")

	return filePath, nil
}
