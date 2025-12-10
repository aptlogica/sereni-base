package storage

import (
	"serenibase/internal/config"
	"serenibase/internal/providers/storage/interfaces"
	"serenibase/internal/providers/storage/local_storage"
	"serenibase/internal/providers/storage/minio_storage"

	"strings"
)

func NewStorage(cfg *config.StorageConfig) (interfaces.StorageProvider, error) {
	switch strings.ToLower(cfg.Driver) {
	case "dev":
		return local_storage.NewStorageProvider(&cfg.Dev)

	case "minio":
		return minio_storage.NewStorageProvider(&cfg.Minio)

	case "aws":
		return local_storage.NewStorageProvider(&cfg.Dev)

	default:
		return local_storage.NewStorageProvider(&cfg.Dev)
	}
}
