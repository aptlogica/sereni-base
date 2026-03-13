// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package antivirus

import (
	"fmt"

	"serenibase/internal/config"
	"serenibase/internal/providers/antivirus/http"
	"serenibase/internal/providers/antivirus/interfaces"
)

// NewAntivirus constructs an antivirus provider based on configuration
func NewAntivirus(cfg *config.AntivirusConfig) (interfaces.Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("antivirus config is nil")
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("antivirus url is empty")
	}

	return http.New(http.Config{
		BaseURL:        cfg.URL,
		TimeoutSeconds: 30,
	})
}
