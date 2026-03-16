// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"mime/multipart"
	"github.com/aptlogica/sereni-base/internal/dto"
)

type TableManagementService interface {
	// table operations
	CreateTableWithDefaults(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error)
	UpdateTable(ctx context.Context, id string, tableData dto.UpdateTableRequest, schemaName string) (dto.TableResponse, error)
	GetTableByID(ctx context.Context, id string, schemaName string) (dto.TableResponse, error)
	GetAllTables(ctx context.Context, schemaName string) ([]dto.TableResponse, error)
	GetModelByBaseID(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error)
	GetModelByWorkspaceID(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error)
	DeleteTable(ctx context.Context, schemaName string, modelID string) error

	// column operations
	AddColumn(ctx context.Context, schemaName string, columnData dto.AddColumnRequest) (dto.ColumnResponse, error)
	GetColumnById(ctx context.Context, schemaName string, id string) (dto.ColumnResponse, error)
	GetAllColumns(ctx context.Context, schemaName string) ([]dto.ColumnResponse, error)
	GetColumnsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ColumnResponse, error)
	UpdateColumn(ctx context.Context, schemaName string, id string, req dto.ColumnUpdate) (dto.ColumnResponse, error)
	DeleteColumn(ctx context.Context, schemaName string, id string) error
	ReorderColumn(ctx context.Context, schemaName string, req dto.ReorderColumnRequest) ([]dto.ColumnResponse, error)

	// view operations
	CreateView(ctx context.Context, schemaName string, viewData dto.CreateViewRequest) (dto.ViewResponse, error)
	GetViewByID(ctx context.Context, schemaName string, id string) (dto.ViewResponse, error)
	GetAllViews(ctx context.Context, schemaName string) ([]dto.ViewResponse, error)
	GetViewsByModelID(ctx context.Context, schemaName string, modelID string) ([]dto.ViewResponse, error)
	UpdateView(ctx context.Context, schemaName string, id string, req dto.ViewUpdate) (dto.ViewResponse, error)
	DeleteView(ctx context.Context, schemaName string, id string) error

	// record operations
	CreateRow(ctx context.Context, schemaName string, req dto.CreateRowRequest) (dto.RecordResponse, error)
	CreateRowWithRecords(ctx context.Context, schemaName string, modelAlias string, record map[string]interface{}) (dto.RecordResponse, error)
	CreateRowsWithRecordsBulk(ctx context.Context, schemaName string, modelAlias string, records []map[string]interface{}) ([]dto.RecordResponse, error)
	GetAllRecords(ctx context.Context, schemaName string, modelID string) (dto.RecordsResponse, error)
	InsertRowData(ctx context.Context, schemaName string, req dto.InsertRowDataRequest) (dto.RecordResponse, error)
	DeleteRow(ctx context.Context, schemaName string, req dto.DeleteRowDataRequest) error
	UpdateRawDataForLinks(ctx context.Context, schemaName string, req dto.UpdateRowDataLinksRequest) (dto.RecordResponse, error)
	AddAttachment(ctx context.Context, schemaName string, req dto.AddAttachmentRequest, files []*multipart.FileHeader) (dto.RecordResponse, error)
	UpdateAttachment(ctx context.Context, schemaName string, req dto.UpdateAttachmentRequest) (dto.RecordResponse, error)
	BulkDeleteRows(ctx context.Context, schemaName string, req dto.BulkDeleteRowsRequest) (int, error)
	RemoveAttachments(ctx context.Context, schemaName string, req dto.RemoveAttachmentsRequest) (dto.RecordResponse, error)
}
