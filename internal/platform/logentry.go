// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import "time"

// LogService is the service name every API platform entry carries. The log read APIs filter on it.
const LogService = "sereni-base"

// LogEntry is one API platform request as the log service takes it (api-platform-log-shipping-suggestion.md
// section 7). The names and types are final so that sending to sereni-log later changes nothing here.
// Field types never change between entries: optional fields are left out, never null.
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"` // when the request happened
	Level     string            `json:"level"`     // info, warn or error
	Message   string            `json:"message"`   // from the message file, by Fields.Event
	Service   string            `json:"service"`
	Labels    map[string]string `json:"labels"` // low cardinality: env only
	Fields    LogFields         `json:"fields"`
}

// LogFields is the free-form part of an entry. Event and Key are stable handles from the message file.
type LogFields struct {
	Event      string `json:"event"`         // log message key, e.g. log.api.client_error
	Key        string `json:"key,omitempty"` // error message key, when the error has one
	API        string `json:"api"`           // route template: the grouping key
	Path       string `json:"path"`          // exact path
	Method     string `json:"method"`
	BaseID     string `json:"base_id"` // public base ID bs_...
	Status     int    `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	RequestID  string `json:"request_id"`
	Table      string `json:"table,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`

	// For a failed request, the text the UI shows next to the status. ErrorMessage is what the client was told
	// (cut at maxErrorMessage); Friendly and Suggestion are the general explanation and the general "what to do"
	// of the error in the message file (key), the same for every request that fails this way.
	ErrorMessage string `json:"error_message,omitempty"`
	Friendly     string `json:"friendly,omitempty"`
	Suggestion   string `json:"suggestion,omitempty"`

	// Phase 2, shipped to the log service only (the local log line never carries bodies). Request and Response are
	// always JSON objects (bodyPolicy.object in bodycapture.go); the flags are left out unless the body was cut.
	RecordCount       *int           `json:"record_count,omitempty"` // size of a list answered; its body is not stored
	Request           map[string]any `json:"request,omitempty"`
	Response          map[string]any `json:"response,omitempty"`
	RequestTruncated  bool           `json:"request_truncated,omitempty"`
	ResponseTruncated bool           `json:"response_truncated,omitempty"`
}
