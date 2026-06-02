// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"context"
)

type ImportSettings struct {
	RemoveDuplicateRecords bool `json:"remove_duplicate_records"`
	TrimSpaces             bool `json:"trim_extra_spaces"`
	RemoveEmptyRows        bool `json:"remove_empty_rows"`
}

type ColumnConfig struct {
	// Column identification
	ColumnName string `json:"column_name" mapstructure:"column_name"` // CSV column header / Database column name
	Title      string `json:"title" mapstructure:"title"`             // Display column name for the table

	// Data type information
	UIDT string `json:"uidt" mapstructure:"uidt"` // UI data type (text, number, email, date, etc.)

	// Optional metadata for the column (arbitrary JSON)
	Meta map[string]interface{} `json:"meta,omitempty" mapstructure:"meta"`
}

type ImportConfig struct {
	Settings ImportSettings `json:"settings"`
	Columns  []ColumnConfig `json:"columns"`
	// PrimaryColumn allows specifying which column should be treated as the primary/title column
	PrimaryColumn *ColumnConfig `json:"primary_column,omitempty" mapstructure:"primary_column"`
}

type ImportTableRequest struct {
	BaseID      string `form:"base_id" json:"base_id" binding:"required"`
	WorkspaceID string `form:"workspace_id" json:"workspace_id" binding:"required"`
	TableName   string `form:"table_name" json:"table_name" binding:"required"`
	OrderIndex  int    `form:"order_index" json:"order_index"`
	CreatedBy   string `form:"created_by" json:"created_by,omitempty"`
}

type ImportTableResponse struct {
	ImportStats *ImportStatistics `json:"import_stats,omitempty"`
	TableResponse
	TableModelViewResponse
}

type ImportStatistics struct {
	TotalRows            int    `json:"total_rows"`
	TotalColumns         int    `json:"total_columns"`
	ErrorRows            int    `json:"error_rows"`
	EmptyRows            int    `json:"empty_rows"`
	DuplicateRows        int    `json:"duplicate_rows"`
	EmptyRowsSkipped     int    `json:"empty_rows_skipped"`
	DuplicatesRemoved    int    `json:"duplicates_removed"`
	ErrorRowsFileContent string `json:"error_rows_file_content,omitempty"`
}

type ImportWithConfigRequest struct {
	BaseID      string       `form:"base_id" json:"base_id" binding:"required"`
	WorkspaceID string       `form:"workspace_id" json:"workspace_id" binding:"required"`
	Title       string       `form:"title" json:"title"`
	Description string       `form:"description" json:"description"`
	OrderIndex  float64      `form:"order_index" json:"order_index"`
	Config      ImportConfig `form:"config" json:"config"`
	CreatedBy   string       `form:"created_by" json:"created_by,omitempty"`
}

// Parameter objects to reduce function parameter counts (go:S107)
type AddColumnsWithConfigParams struct {
	Ctx           context.Context
	SchemaName    string
	Req           CreateTableRequest
	Headers       []string
	ColumnConfigs []ColumnConfig
	Primary       *ColumnConfig
	TableResp     TableResponse
}

type BuildRecordsWithConfigAndErrorsParams struct {
	DataRows      [][]string
	ColumnConfigs []ColumnConfig
	Primary       *ColumnConfig
	ColumnMap     map[int]ColumnResponse
	Req           CreateTableRequest
	Headers       []string
	Settings      ImportSettings
}

// type ImportTableResponse struct {
// 	TableResponse
// }

type Relation struct {
	Type        string `json:"type"`
	SourceTable string `json:"source_table"`
	TargetTable string `json:"target_table"`
}

// AI Table JSON structures
type AiTableField struct {
	Name string                 `json:"name"`
	Type string                 `json:"type"`
	Meta map[string]interface{} `json:"meta"`
}

type AiTable struct {
	Name   string         `json:"name"`
	Fields []AiTableField `json:"fields"`
}

type AiTableResponse struct {
	Tables    []AiTable  `json:"tables"`
	Relations []Relation `json:"relations"`
}

type ImportBaseResponse struct {
	BaseResponse
}

type AiBaseResponse struct {
	BaseName  string     `json:"base_name"`
	Relations []Relation `json:"relations"`
	Tables    []AiTable  `json:"tables"`
}