package local_storage

import (
	"serenibase/internal/config"
	"serenibase/internal/utils/file"
	"serenibase/internal/providers/storage/interfaces"

	app_errors "serenibase/internal/app-errors"
)

func NewStorageProvider(cfg *config.StorageDevConfig) (interfaces.StorageProvider, error) {

	err := file.CreateDirIfNotExists(cfg.Path, 0755)
	if err != nil {
		return nil, app_errors.FolderCreateFailed
	}

	utility := NewLocalStorageService(cfg.Path)
	return utility, nil
}
