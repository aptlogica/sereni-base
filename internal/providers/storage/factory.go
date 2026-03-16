// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package storage

import (
	"fmt"
	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/providers/storage/http"
	"github.com/aptlogica/sereni-base/internal/providers/storage/interfaces"
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
