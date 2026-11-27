// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"net/url"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/providers/logquery"
)

// LogQuerier reads logs from sereni-log; logquery.Client implements it
type LogQuerier interface {
	Query(ctx context.Context, params url.Values) (logquery.Result, error)
	Count(ctx context.Context, params url.Values) (logquery.Result, error)
}

// BaseAPILogResult is the answer to send as it is: the status and JSON body of sereni-log, or one of the same shape
type BaseAPILogResult = logquery.Result

// InvalidLogFilterError is a bad filter value in the request; it is answered with 400
type InvalidLogFilterError struct{ Message string }

func (e *InvalidLogFilterError) Error() string { return e.Message }

// BaseAPILogService reads the API platform logs of one base
type BaseAPILogService interface {
	// List returns the log entries of the base, newest first by default
	List(ctx context.Context, schemaName, baseID string, req dto.BaseAPILogsRequest) (BaseAPILogResult, error)
	// Count returns the number of log entries of the base for the same filters
	Count(ctx context.Context, schemaName, baseID string, req dto.BaseAPILogsRequest) (BaseAPILogResult, error)
}
