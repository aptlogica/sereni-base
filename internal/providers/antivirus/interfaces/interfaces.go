// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"io"
)

// ScanResult represents the outcome of an antivirus scan
type ScanResult struct {
	FileName string `json:"file_name"`
	// Clean indicates whether the content was found to be clean
	Clean bool
	// Threat describes the detected threat, if any. Empty when Clean is true
	Threat string
}

// Provider abstracts antivirus operations across different backends
type Provider interface {
	// Ping checks connectivity/health of the underlying antivirus engine
	Ping(ctx context.Context) error

	// ScanReader scans the provided content stream and returns the scan result
	ScanReader(ctx context.Context, fileName string, r io.Reader) (ScanResult, error)
}
