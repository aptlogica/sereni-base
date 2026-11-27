// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Log message keys of the message file (messages/en-US.json) that the access log uses
const (
	eventRequest      = "log.api.request"
	eventClientError  = "log.api.client_error"
	eventServerError  = "log.api.server_error"
	eventUnknownBase  = "log.api.unknown_base"
	eventRateLimited  = "log.api.rate_limited"
	eventBatchDropped = "log.ship.batch_dropped" // written by LogShipDropped, not by the middleware
)

// usedLogEvents lists the log keys the code above uses, so a test can check them against the message file
var usedLogEvents = []string{eventRequest, eventClientError, eventServerError, eventUnknownBase, eventRateLimited, eventBatchDropped}

// Set by respondError for the access log: the final error of the request
const (
	ctxErrorCode = "platform.errorCode"
	ctxErrorKey  = "platform.errorKey" // message key of the error catalog, when the error has one

	ctxErrorMessage = "platform.errorMessage" // what the client was told

	// maxErrorMessage caps the error text of an entry: it can carry names from the request
	maxErrorMessage = 500
)

// errorMessageKey is the key of the general message of an error code in the message file
func errorMessageKey(code ErrorCode) string { return "error." + string(code) }

// errorKeyUnknownBase is the general message of a 404 that came before the base was resolved
const errorKeyUnknownBase = "error.unknown_base"

// WithLogSink turns on the API platform access log: each request is written to sink as one entry. With
// logAll every request is logged; without it only those that end with status 400 or higher. env is the
// label the entries carry (SERVER_ENV). Call before Register.
func (a *API) WithLogSink(sink LogSink, env string, logAll bool) *API {
	a.logSink = sink
	a.logEnv = env
	a.logAll = logAll
	return a
}

// WithLogBodies turns on body capture for the log service: level is off, errors or all, maxBytes caps each
// body (default 8 KB, at most 12 KB), and redactFields are masked in addition to the built-in secret names.
// Bodies are only put in entries; the local log line (ZerologSink) never prints them. Call before Register.
func (a *API) WithLogBodies(level string, maxBytes int, redactFields []string) *API {
	a.bodies = newBodyPolicy(level, maxBytes, redactFields)
	return a
}

// accessLog logs the requests of the /:bs group, all or only the failed ones. It is first in the chain, so it
// also sees what resolveBase and the rate limiter abort. It runs around the handler and only observes: a
// problem here can lose a log line, never change the answer. The docs pages are not logged.
func (a *API) accessLog() gin.HandlerFunc {
	sink, logAll, bodies := a.logSink, a.logAll, a.bodies
	if sink == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if isDocsRoute(c.FullPath()) {
			c.Next()
			return
		}
		start := a.now()
		var tap *bodyTap
		if bodies != nil {
			tap = bodies.start(c)
		}
		c.Next()
		if status := c.Writer.Status(); logAll || status >= http.StatusBadRequest {
			a.writeAccessLog(sink, c, start, status, bodies, tap)
		}
	}
}

// isDocsRoute tells the spec and docs pages of a base, which are not worth a log entry
func isDocsRoute(fullPath string) bool {
	return strings.HasSuffix(fullPath, "/:bs/openapi.json") || strings.HasSuffix(fullPath, "/:bs/docs") ||
		strings.HasSuffix(fullPath, "/:bs/redoc")
}

func (a *API) writeAccessLog(sink LogSink, c *gin.Context, start time.Time, status int, bodies *bodyPolicy, tap *bodyTap) {
	defer func() { _ = recover() }() // logging must never fail a request

	code := c.GetString(ctxErrorCode)
	_, baseResolved := c.Get(ctxBase)
	event := logEvent(status, code, baseResolved)
	key := c.GetString(ctxErrorKey)
	if event == eventUnknownBase {
		key = errorKeyUnknownBase // "not found" would point at a table or record
	}

	api := c.FullPath()
	if api == "" {
		api = c.Request.URL.Path
	}
	f := LogFields{
		Event:      event,
		Key:        key,
		API:        api,
		Path:       c.Request.URL.Path,
		Method:     c.Request.Method,
		BaseID:     c.Param("bs"),
		Status:     status,
		DurationMS: a.now().Sub(start).Milliseconds(),
		RequestID:  requestID(c),
		Table:      c.Param("table"),
		ErrorCode:  code,
	}
	if status >= http.StatusBadRequest {
		f.ErrorMessage = cleanText(c.GetString(ctxErrorMessage), maxErrorMessage)
		if m, ok := defaultMessages.get(key); ok {
			f.Friendly = m.Message
			f.Suggestion = defaultMessages.renderAction(key, nil)
		}
	}
	if n, ok := c.Get(ctxRecordCount); ok {
		count := n.(int)
		f.RecordCount = &count
	}
	if bodies != nil && tap != nil && bodies.wants(status) {
		f.Request, f.RequestTruncated = bodies.object(tap.req, c.Request.Header.Get("Content-Type"))
		if f.RecordCount == nil { // a list is stored as its size only
			f.Response, f.ResponseTruncated = bodies.object(tap.resp, tap.writer.Header().Get("Content-Type"))
		}
	}
	level := "warn"
	switch {
	case status >= http.StatusInternalServerError:
		level = "error"
	case status < http.StatusBadRequest:
		level = "info"
	}
	if m, ok := defaultMessages.get(event); ok {
		level = m.Level
	}
	sink.Write(LogEntry{
		Timestamp: start,
		Level:     level,
		Message: defaultMessages.render(event, map[string]any{
			"method": f.Method, "api": f.API, "status": f.Status, "base_id": f.BaseID,
		}),
		Service: LogService,
		Labels:  map[string]string{"env": a.logEnv},
		Fields:  f,
	})
}

// logEvent picks the log message of a failed request
func logEvent(status int, errorCode string, baseResolved bool) string {
	switch {
	case status < http.StatusBadRequest:
		return eventRequest
	case ErrorCode(errorCode) == CodeRateLimited:
		return eventRateLimited
	case status == http.StatusNotFound && !baseResolved:
		return eventUnknownBase
	case status >= http.StatusInternalServerError:
		return eventServerError
	default:
		return eventClientError
	}
}
