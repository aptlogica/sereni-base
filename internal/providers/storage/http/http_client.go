// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
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
	"net/url"
	"time"

	"github.com/aptlogica/sereni-base/internal/providers/storage/interfaces"
)

// Config holds the configuration for the HTTP storage client
type Config struct {
	BaseURL        string
	TimeoutSeconds int
}

const (
	ErrCreateRequest = "storage http: failed to create request: %w"
	ErrSendRequest   = "storage http: failed to send request: %w"
)

// Client implements interfaces.StorageProvider for HTTP-based storage service
type Client struct {
	config     Config
	httpClient *http.Client
}

// New creates a new HTTP storage client
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("storage http: base URL is required")
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 60 // Storage operations may take longer
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
	}, nil
}

// Upload uploads a file to the storage service
func (c *Client) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (interfaces.UploadResponse, error) {
	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file
	part, err := writer.CreateFormFile("file", objectName)
	if err != nil {
		return interfaces.UploadResponse{}, fmt.Errorf("storage http: failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, reader); err != nil {
		return interfaces.UploadResponse{}, fmt.Errorf("storage http: failed to copy file content: %w", err)
	}

	// Add the path field
	if err := writer.WriteField("path", objectName); err != nil {
		return interfaces.UploadResponse{}, fmt.Errorf("storage http: failed to write path field: %w", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		return interfaces.UploadResponse{}, fmt.Errorf("storage http: failed to close writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/storage/upload", &buf)
	if err != nil {
		return interfaces.UploadResponse{}, fmt.Errorf(ErrCreateRequest, err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return interfaces.UploadResponse{}, fmt.Errorf(ErrSendRequest, err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return interfaces.UploadResponse{}, fmt.Errorf("storage http: upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result interfaces.UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return interfaces.UploadResponse{}, fmt.Errorf("storage http: failed to decode response: %w", err)
	}

	// Return the parsed response
	return result, nil
}

// Download downloads a file from the storage service
func (c *Client) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	// Build URL with query parameter
	downloadURL := fmt.Sprintf("%s/storage/download?path=%s", c.config.BaseURL, url.QueryEscape(objectName))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf(ErrCreateRequest, err)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(ErrSendRequest, err)
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("storage http: download failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Return the response body as ReadCloser
	return resp.Body, nil
}

// Delete deletes a file from the storage service
func (c *Client) Delete(ctx context.Context, objectName string) error {
	// Build URL with query parameter
	deleteURL := fmt.Sprintf("%s/storage/delete?path=%s", c.config.BaseURL, url.QueryEscape(objectName))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf(ErrCreateRequest, err)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf(ErrSendRequest, err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("storage http: delete failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Exists checks if a file exists in the storage service
func (c *Client) Exists(ctx context.Context, objectName string) (bool, error) {
	// Try to download the file and check if it exists
	downloadURL := fmt.Sprintf("%s/storage/download?path=%s", c.config.BaseURL, url.QueryEscape(objectName))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return false, fmt.Errorf(ErrCreateRequest, err)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf(ErrSendRequest, err)
	}
	defer resp.Body.Close()

	// 200 = exists, 404 = doesn't exist, other = error
	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("storage http: exists check failed with status %d: %s", resp.StatusCode, string(bodyBytes))
}
