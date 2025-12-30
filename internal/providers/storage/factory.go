package storage

import (
	"fmt"
	"serenibase/internal/config"
	"serenibase/internal/providers/storage/http"
	"serenibase/internal/providers/storage/interfaces"
)

func NewStorage(cfg *config.StorageConfig) (interfaces.StorageProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("storage config is nil")
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("storage url is empty")
	}

	return http.New(http.Config{
		BaseURL:        cfg.URL,
		TimeoutSeconds: 60,
	})
}
