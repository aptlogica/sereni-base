// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/providers/logship"
	"github.com/rs/zerolog"
)

// LogSink receives one entry per logged API platform request. Write must be quick and must not fail the
// request: the middleware calls it from the request path.
type LogSink interface {
	Write(entry LogEntry)
}

// ZerologSink writes entries through the application's zerolog logger. A nil Logger uses the global one.
type ZerologSink struct {
	Logger *zerolog.Logger
}

// Write logs the entry as one line with its fields as flat keys, so the stdout line is searchable by
// request_id, base_id, event and the rest.
func (s ZerologSink) Write(e LogEntry) {
	lg := s.Logger
	if lg == nil {
		lg = logger.Get()
	}
	level := zerolog.WarnLevel
	switch e.Level {
	case "error":
		level = zerolog.ErrorLevel
	case "info":
		level = zerolog.InfoLevel
	}
	ev := lg.WithLevel(level).
		Str("event", e.Fields.Event).
		Str("api", e.Fields.API).
		Str("path", e.Fields.Path).
		Str("method", e.Fields.Method).
		Str("base_id", e.Fields.BaseID).
		Int("status", e.Fields.Status).
		Int64("duration_ms", e.Fields.DurationMS).
		Str("request_id", e.Fields.RequestID)
	if e.Fields.Table != "" {
		ev = ev.Str("table", e.Fields.Table)
	}
	if e.Fields.ErrorCode != "" {
		ev = ev.Str("error_code", e.Fields.ErrorCode)
	}
	if e.Fields.Key != "" {
		ev = ev.Str("key", e.Fields.Key)
	}
	if e.Fields.ErrorMessage != "" {
		ev = ev.Str("error_message", e.Fields.ErrorMessage)
	}
	if e.Fields.Suggestion != "" {
		ev = ev.Str("suggestion", e.Fields.Suggestion)
	}
	ev.Msg(e.Message)
}

// MultiSink writes every entry to each of its sinks, in order. A sink that panics does not stop the others.
type MultiSink []LogSink

func (m MultiSink) Write(e LogEntry) {
	for _, s := range m {
		func() {
			defer func() { _ = recover() }()
			s.Write(e)
		}()
	}
}

// Shipper takes entries for the log service. *logship.Client is the implementation.
type Shipper interface {
	Enqueue(logship.Entry) bool
}

// LogServiceSink hands entries to the background shipper (Phase 2). Enqueue never blocks: a full queue drops
// the entry, and the shipper counts and reports that.
type LogServiceSink struct {
	Shipper Shipper
}

// Write converts the entry to the log service's format and queues it
func (s LogServiceSink) Write(e LogEntry) {
	s.Shipper.Enqueue(logship.Entry{
		Timestamp: e.Timestamp,
		Level:     e.Level,
		Message:   e.Message,
		Service:   e.Service,
		Labels:    e.Labels,
		Fields:    e.Fields,
	})
}

// LogShipDropped is the logship.Config.OnDrop callback. It writes the log.ship.batch_dropped message to the
// local log only; it must never go through the shipper, or a failing log service would feed itself.
func LogShipDropped(count int, reason string) {
	logger.Get().Warn().
		Str("event", eventBatchDropped).
		Int("count", count).
		Str("reason", reason).
		Msg(defaultMessages.render(eventBatchDropped, map[string]any{"count": count, "reason": reason}))
}
