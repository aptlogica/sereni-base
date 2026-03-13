// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"serenibase/internal/providers/antivirus/interfaces"
)

// Config holds the configuration for the HTTP antivirus client
type Config struct {
	BaseURL        string
	TimeoutSeconds int
}

// Client implements interfaces.Provider for HTTP-based antivirus service
type Client struct {
	config     Config
	httpClient *http.Client
}

// New creates a new HTTP antivirus client
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("antivirus http: base URL is required")
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 30
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
	}, nil
}

// Ping verifies antivirus service availability
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("antivirus http: failed to create ping request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("antivirus http: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("antivirus http: ping returned status %d", resp.StatusCode)
	}

	return nil
}

// ScanResult represents the response from the antivirus API
type scanResultResponse struct {
	FileName string `json:"file_name"`
	Clean    bool   `json:"clean"`
	Threat   string `json:"threat"`
}

// ScanReader scans a stream for malware using the HTTP antivirus service
func (c *Client) ScanReader(ctx context.Context, fileName string, r io.Reader) (interfaces.ScanResult, error) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create form file
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return interfaces.ScanResult{
			FileName: fileName,
			Clean:    false,
			Threat:   "failed to create form file",
		}, fmt.Errorf("antivirus http: failed to create form file: %w", err)
	}

	// Copy file content to form
	if _, err := io.Copy(part, r); err != nil {
		return interfaces.ScanResult{
			FileName: fileName,
			Clean:    false,
			Threat:   "failed to copy file content",
		}, fmt.Errorf("antivirus http: failed to copy file content: %w", err)
	}

	// Close the writer to finalize the multipart message
	if err := writer.Close(); err != nil {
		return interfaces.ScanResult{
			FileName: fileName,
			Clean:    false,
			Threat:   "failed to finalize multipart form",
		}, fmt.Errorf("antivirus http: failed to close writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/scan", &buf)
	if err != nil {
		return interfaces.ScanResult{
			FileName: fileName,
			Clean:    false,
			Threat:   "failed to create request",
		}, fmt.Errorf("antivirus http: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return interfaces.ScanResult{
			FileName: fileName,
			Clean:    false,
			Threat:   "failed to send request",
		}, fmt.Errorf("antivirus http: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return interfaces.ScanResult{
			FileName: fileName,
			Clean:    false,
			Threat:   fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)),
		}, fmt.Errorf("antivirus http: scan failed with status %d", resp.StatusCode)
	}

	// Parse response
	var result scanResultResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return interfaces.ScanResult{
			FileName: fileName,
			Clean:    false,
			Threat:   "failed to parse response",
		}, fmt.Errorf("antivirus http: failed to decode response: %w", err)
	}

	// If file is infected, return error
	if !result.Clean {
		return interfaces.ScanResult{
			FileName: result.FileName,
			Clean:    false,
			Threat:   result.Threat,
		}, fmt.Errorf("virus detected in file '%s': %s", result.FileName, result.Threat)
	}

	return interfaces.ScanResult{
		FileName: result.FileName,
		Clean:    true,
		Threat:   "",
	}, nil
}
