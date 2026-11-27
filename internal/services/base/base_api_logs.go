// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/platform"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
)

const (
	// MaxBaseAPILogLimit is the most entries one page can ask for
	MaxBaseAPILogLimit = 500
	// logWindowDefault is the window sereni-log uses without start and end; it is repeated in the empty answer
	logWindowDefault = 24 * time.Hour
)

type baseAPILogService struct {
	baseAPI interfaces.BaseAPIService
	logs    interfaces.LogQuerier
}

// NewBaseAPILogService returns the service that reads the API platform logs of a base from sereni-log
func NewBaseAPILogService(baseAPI interfaces.BaseAPIService, logs interfaces.LogQuerier) interfaces.BaseAPILogService {
	return &baseAPILogService{baseAPI: baseAPI, logs: logs}
}

func (s *baseAPILogService) List(ctx context.Context, schemaName, baseID string, req dto.BaseAPILogsRequest) (interfaces.BaseAPILogResult, error) {
	params, _, _, err := buildLogParams(req, true)
	if err != nil {
		return interfaces.BaseAPILogResult{}, err
	}
	publicID, err := s.publicID(ctx, schemaName, baseID)
	if err != nil {
		return interfaces.BaseAPILogResult{}, err
	}
	if publicID == "" {
		return emptyLogAnswer(map[string]any{"query": "", "count": 0, "entries": []any{}}), nil
	}
	params.Set("fields.base_id", publicID)
	return s.logs.Query(ctx, params)
}

func (s *baseAPILogService) Count(ctx context.Context, schemaName, baseID string, req dto.BaseAPILogsRequest) (interfaces.BaseAPILogResult, error) {
	params, start, end, err := buildLogParams(req, false)
	if err != nil {
		return interfaces.BaseAPILogResult{}, err
	}
	publicID, err := s.publicID(ctx, schemaName, baseID)
	if err != nil {
		return interfaces.BaseAPILogResult{}, err
	}
	if publicID == "" {
		if end.IsZero() {
			end = time.Now().UTC()
		}
		if start.IsZero() {
			start = end.Add(-logWindowDefault)
		}
		return emptyLogAnswer(map[string]any{"query": "", "count": 0, "start": start, "end": end}), nil
	}
	params.Set("fields.base_id", publicID)
	return s.logs.Count(ctx, params)
}

// publicID returns the base's bs_ ID, or "" when the API was never turned on (nothing was ever logged)
func (s *baseAPILogService) publicID(ctx context.Context, schemaName, baseID string) (string, error) {
	out, err := s.baseAPI.Get(ctx, schemaName, baseID)
	if err != nil {
		return "", err
	}
	return out.PublicID, nil
}

// emptyLogAnswer has the shape of an answer of sereni-log, for a base that has no logs to ask for
func emptyLogAnswer(data map[string]any) interfaces.BaseAPILogResult {
	body, _ := json.Marshal(map[string]any{"success": true, "code": "OK", "message": "ok", "data": data})
	return interfaces.BaseAPILogResult{Status: 200, Body: body}
}

func badFilter(format string, args ...any) error {
	return &interfaces.InvalidLogFilterError{Message: fmt.Sprintf(format, args...)}
}

var (
	logLevels     = map[string]bool{"debug": true, "info": true, "warn": true, "error": true, "fatal": true}
	logDirections = map[string]bool{"backward": true, "forward": true}
)

// buildLogParams checks the filters and returns the sereni-log parameters, without the base ID. service is
// always ours, and the raw query of sereni-log is never passed, so a caller cannot read another base's logs.
// start and end are returned for the answer of a base with no logs.
func buildLogParams(req dto.BaseAPILogsRequest, paged bool) (url.Values, time.Time, time.Time, error) {
	var start, end time.Time
	params := url.Values{}
	params.Set("service", platform.LogService)

	if v := strings.ToLower(strings.TrimSpace(req.Level)); v != "" {
		if !logLevels[v] {
			return nil, start, end, badFilter("level must be one of debug, info, warn, error, fatal")
		}
		params.Set("level", v)
	}
	if v := strings.TrimSpace(req.Status); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 100 || n > 599 {
			return nil, start, end, badFilter("status must be a number from 100 to 599")
		}
		params.Set("fields.status", strconv.Itoa(n))
	}
	for name, v := range map[string]string{
		"fields.method":     strings.ToUpper(strings.TrimSpace(req.Method)),
		"fields.request_id": strings.TrimSpace(req.RequestID),
		"fields.table":      strings.TrimSpace(req.Table),
		"fields.error_code": strings.TrimSpace(req.ErrorCode),
		"search":            strings.TrimSpace(req.Search),
	} {
		if v != "" {
			params.Set(name, v)
		}
	}

	var err error
	if start, err = parseLogTime("start", req.Start, params); err != nil {
		return nil, start, end, err
	}
	if end, err = parseLogTime("end", req.End, params); err != nil {
		return nil, start, end, err
	}
	if !start.IsZero() && !end.IsZero() && !end.After(start) {
		return nil, start, end, badFilter("end must be after start")
	}

	if !paged {
		return params, start, end, nil
	}
	if v := strings.TrimSpace(req.Limit); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > MaxBaseAPILogLimit {
			return nil, start, end, badFilter("limit must be a number from 1 to %d", MaxBaseAPILogLimit)
		}
		params.Set("limit", strconv.Itoa(n))
	}
	if v := strings.ToLower(strings.TrimSpace(req.Direction)); v != "" {
		if !logDirections[v] {
			return nil, start, end, badFilter("direction must be backward or forward")
		}
		params.Set("direction", v)
	}
	if v := strings.TrimSpace(req.Cursor); v != "" {
		params.Set("cursor", v)
	}
	return params, start, end, nil
}

// parseLogTime reads an RFC3339 time and sets it in params under name; empty is left out
func parseLogTime(name, value string, params url.Values) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, badFilter("%s must be a time like 2026-11-26T10:00:00Z", name)
	}
	params.Set(name, t.UTC().Format(time.RFC3339))
	return t.UTC(), nil
}
