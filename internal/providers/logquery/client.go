// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package logquery reads logs from sereni-log (GET /logs/query and /logs/count). It does not read the answer:
// the status and the body of sereni-log are handed back as they are.
package logquery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
	// maxBodyBytes caps what is read from sereni-log: 500 entries of at most 32 KB each
	maxBodyBytes = 20 << 20
)

var (
	// ErrNotConfigured means LOG_SERVICE_URL is empty
	ErrNotConfigured = errors.New("log service is not configured")
	// ErrTimeout means sereni-log did not answer in time
	ErrTimeout = errors.New("log service timed out")
	// ErrUnavailable means sereni-log cannot be reached, failed (5xx), or refused our API key (401)
	ErrUnavailable = errors.New("log service is unavailable")
)

// Config holds the settings of the client
type Config struct {
	URL     string        // LOG_SERVICE_URL, including /api/v1
	APIKey  string        // sent as X-API-Key when set
	Timeout time.Duration // default 10s
}

// Result is the answer of sereni-log, unread
type Result struct {
	Status int
	Body   []byte
}

// Client calls sereni-log
type Client struct {
	cfg  Config
	http *http.Client
}

// New returns a client. With an empty URL every call returns ErrNotConfigured.
func New(cfg Config) *Client {
	cfg.URL = strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// Query calls GET /logs/query with params
func (c *Client) Query(ctx context.Context, params url.Values) (Result, error) {
	return c.get(ctx, "/logs/query", params)
}

// Count calls GET /logs/count with params
func (c *Client) Count(ctx context.Context, params url.Values) (Result, error) {
	return c.get(ctx, "/logs/count", params)
}

// get returns the answer for 200 and 400 (a bad filter, which the caller should see as it is). Any other
// status is ErrUnavailable: 401 means our key is wrong, which is not the caller's login failing.
func (c *Client) get(ctx context.Context, path string, params url.Values) (Result, error) {
	if c == nil || c.cfg.URL == "" {
		return Result{}, ErrNotConfigured
	}
	target := c.cfg.URL + path
	if len(params) > 0 {
		target += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Result{}, mapTransportError(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return Result{}, mapTransportError(err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusBadRequest:
		return Result{Status: resp.StatusCode, Body: body}, nil
	default:
		return Result{}, fmt.Errorf("%w: sereni-log answered %d", ErrUnavailable, resp.StatusCode)
	}
}

func mapTransportError(err error) error {
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}
	return fmt.Errorf("%w: %v", ErrUnavailable, err)
}
