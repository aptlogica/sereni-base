package antivirus

import (
	"fmt"
	"strings"

	"serenibase/internal/config"
	"serenibase/internal/providers/antivirus/clamav"
	"serenibase/internal/providers/antivirus/interfaces"
)

// NewAntivirus constructs an antivirus provider based on configuration
func NewAntivirus(cfg *config.AntivirusConfig) (interfaces.Provider, error) {
	switch strings.ToLower(cfg.Driver) {
	case "clamav":
		return clamav.New(clamav.Config{
			Address:        cfg.ClamAV.Address,
			TimeoutSeconds: cfg.ClamAV.TimeoutSeconds,
		})
	default:
		return nil, fmt.Errorf("unsupported antivirus driver: %s", cfg.Driver)
	}
}
