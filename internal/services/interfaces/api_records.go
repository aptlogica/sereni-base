// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

// APIRecordService writes records for the workspace public API. Values are keyed by column name and already
// converted to what the column stores. Each call is all-or-nothing: a failed record is reported as an
// *app_errors.RecordWriteError with its position, and nothing is written.
type APIRecordService interface {
	// CreateRecordsForAPI inserts the records and returns them as stored (row_to_json of each row)
	CreateRecordsForAPI(ctx context.Context, schemaName string, model tenant.Model, records []map[string]interface{}) ([]map[string]interface{}, error)
	// UpdateRecordsForAPI changes the given values of each record and returns the records as stored
	UpdateRecordsForAPI(ctx context.Context, schemaName string, model tenant.Model, updates []dto.APIRecordUpdate) ([]map[string]interface{}, error)
	// DeleteRecordsForAPI deletes the records; when some don't exist, nothing is deleted and their ids are returned
	DeleteRecordsForAPI(ctx context.Context, schemaName string, model tenant.Model, ids []int64) (missing []int64, err error)
}
