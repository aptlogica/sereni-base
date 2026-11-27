// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

// BaseAPILogsRequest holds the filters of GET /base/{id}/api/logs and /api/logs/count. All are text here and
// are checked by the service, so every bad value is the same 400. The base and the service are never taken
// from the caller.
type BaseAPILogsRequest struct {
	Level     string `form:"level"`
	Status    string `form:"status"`
	Method    string `form:"method"`
	RequestID string `form:"request_id"`
	Table     string `form:"table"`
	ErrorCode string `form:"error_code"`
	Search    string `form:"search"`
	Start     string `form:"start"`
	End       string `form:"end"`
	// Paging: used by the list only
	Limit     string `form:"limit"`
	Direction string `form:"direction"`
	Cursor    string `form:"cursor"`
}
