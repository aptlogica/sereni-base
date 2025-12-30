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
