// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"
)

type CreateTableRequest struct {
	BaseID      string  `json:"base_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	WorkspaceID string  `json:"workspace_id" binding:"required" example:"workspace-abc-123"`
	Title       string  `json:"title" binding:"required" example:"My Table" format:"string"`
	Description string  `json:"description" example:"A description of the table" format:"string"`
	OrderIndex  float64 `json:"order_index,omitempty" example:"1.0"`
	CreatedBy   string  `json:"created_by,omitempty"`
}

type TableResponse struct {
	Model   ModelResponse            `json:"model" mapstructure:"model"`
	Columns []ColumnResponse         `json:"columns" mapstructure:"columns"`
	Views   []ViewResponse           `json:"views" mapstructure:"views"`
	Records []map[string]interface{} `json:"records" mapstructure:"records"`
}

type TablePageResponse struct {
	Columns []ColumnResponse         `json:"columns" mapstructure:"columns"`
	Records []map[string]interface{} `json:"records" mapstructure:"records"`
}

type UpdateTableRequest struct {
	Title       *string                `db:"title" json:"title,omitempty" mapstructure:"title"`
	Description *string                `json:"description" example:"A description of the table" format:"string"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	UpdatedBy   string                 `json:"last_modified_by,omitempty"`
	UpdatedAt   time.Time              `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (a UpdateTableRequest) Map() map[string]interface{} {
	m := make(map[string]interface{})

	if a.Title != nil {
		m["title"] = a.Title
	}
	if a.Description != nil {
		m["Description"] = a.Description
	}
	if a.Meta != nil {
		m["meta"] = a.Meta
	}
	if !a.UpdatedAt.IsZero() {
		m["last_modified_time"] = a.UpdatedAt
	}
	if a.UpdatedBy != "" {
		m["last_modified_by"] = a.UpdatedBy
	}
	return m
}

type CreateRowRequest struct {
	ModelID   string `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	CreatedBy string `json:"created_by,omitempty"`
}

type RecordResponse struct {
	Record map[string]interface{} `json:"record"`
}

type RecordsResponse struct {
	Records []map[string]interface{} `json:"records"`
}

type InsertRowDataRequest struct {
	ModelID   string       `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnId  string       `json:"column_id" binding:"required" example:"col-123"`
	RowId     int          `json:"row_id" binding:"required" example:"1"`
	Value     *interface{} `json:"value"`
	UpdatedBy string       `json:"last_modified_by,omitempty"`
}

type DeleteRowDataRequest struct {
	ModelID string `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	RowId   int    `json:"row_id" binding:"required" example:"1"`
}

type BulkDeleteRowsRequest struct {
	ModelID string `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	RowIds  []int  `json:"row_ids" binding:"required,min=1"`
}

type UpdateRowDataLinksRequest struct {
	ModelID     string `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnId    string `json:"column_id" binding:"required" example:"col-123"`
	SourceRowId int    `json:"source_row_id" binding:"required" example:"1"`
	TargetRowId int    `json:"target_row_id" binding:"required" example:"2"`
	Action      string `json:"action" binding:"required,oneof=link unlink" example:"link"`
	UpdatedBy   string `json:"last_modified_by,omitempty"`
}

type AddAttachmentRequest struct {
	ModelID  string `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnId string `json:"column_id" binding:"required" example:"col-123"`
	RowId    int    `json:"row_id" binding:"required" example:"1"`
}

type UpdateAttachmentRequest struct {
	ModelID  string      `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnId string      `json:"column_id" binding:"required" example:"col-123"`
	RowId    int         `json:"row_id" binding:"required" example:"1"`
	AssetId  string      `json:"asset_id" binding:"required" example:"asset-456"`
	Content  AssetUpdate `json:"content" binding:"required" example:"{\"file_name\": \"document.pdf\", \"file_size\": 102400, \"file_type\": \"application/pdf\"}"`
}

type RemoveAttachmentsRequest struct {
	ModelID     string   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnId    string   `json:"column_id" binding:"required" example:"col-123"`
	RowId       int      `json:"row_id" binding:"required" example:"1"`
	Attachments []string `json:"attachments" binding:"required" example:"asset-456"`
}

type PaginationRequest struct {
	ModelID    string `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	PageSize   int    `json:"page_size" binding:"required" example:"30"`
	PageNumber int    `json:"page_number" binding:"required" example:"1"`
}
