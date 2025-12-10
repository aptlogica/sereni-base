package clamav

import (
	"context"
	"errors"
	"io"

	"serenibase/internal/providers/antivirus/interfaces"

	"github.com/dutchcoders/go-clamd"
)

// Config holds the configuration for the ClamAV provider
type Config struct {
	// Address is the host:port of the clamd service, e.g. 127.0.0.1:3310
	Address string
	// TimeoutSeconds is the network timeout for clamd operations
	TimeoutSeconds int
}

// Provider implements interfaces.Provider for ClamAV (clamd)
type Provider struct {
	config Config
	clamd  *clamd.Clamd
}

// New creates a new ClamAV antivirus provider instance
func New(cfg Config) (*Provider, error) {
	if cfg.Address == "" {
		return nil, errors.New("clamav: address is required")
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 30
	}
	// Use "tcp" for ClamdTCP, or "unix" for ClamdUnix
	c := clamd.NewClamd("tcp://" + cfg.Address)
	return &Provider{config: cfg, clamd: c}, nil
}

// Ping verifies clamd availability by sending a PING command.
func (p *Provider) Ping(ctx context.Context) error {
	return p.clamd.Ping()
}

// ScanReader scans a stream for malware using clamd INSTREAM.
func (p *Provider) ScanReader(ctx context.Context,fileName string, r io.Reader) (interfaces.ScanResult, error) {
	resultChan, err := p.clamd.ScanStream(r, make(chan bool))
	if err != nil {
		return interfaces.ScanResult{
			FileName: fileName,
			Clean:  false,
			Threat: err.Error(),
		}, errors.New("clamav scan failed for file " + fileName + ": " + err.Error())
	}

	for scanResult := range resultChan {
		switch scanResult.Status {
		case clamd.RES_OK:
			// clean
			return interfaces.ScanResult{
				FileName: fileName,
				Clean:  true,
				Threat: "",
			}, nil
		case clamd.RES_FOUND:
			// infected
			return interfaces.ScanResult{
				FileName: fileName,
				Clean:  false,
				Threat: scanResult.Description,
			}, errors.New("virus detected and infected file: " + fileName + scanResult.Description)
		case clamd.RES_ERROR, clamd.RES_PARSE_ERROR:
			return interfaces.ScanResult{
				FileName: fileName,
				Clean:  false,
				Threat: scanResult.Description,
			}, errors.New("clamav scan error on : " + fileName + scanResult.Description)
		}
	}
	// If no result, something went wrong
	return interfaces.ScanResult{
		FileName: fileName,
		Clean:  false,
		Threat: "no scan result returned",
	}, errors.New("clamav: no scan result returned ")
}
