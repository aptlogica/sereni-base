// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
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

type TableModelViewResponse struct {
	Model ModelResponse  `json:"model" mapstructure:"model"`
	Views []ViewResponse `json:"views" mapstructure:"views"`
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

type CreateRowOrBulkInsertRequest struct {
	ModelID   string                   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Rows      []map[string]interface{} `json:"rows"`
	CreatedBy string                   `json:"created_by,omitempty"`
	UpdatedBy string                   `json:"last_modified_by,omitempty"`
}

type UpdateRowRequest struct {
	ModelID   string                 `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	RowId     int                    `json:"row_id" binding:"required" example:"1"`
	Values    map[string]interface{} `json:"values" binding:"required"`
	UpdatedBy string                 `json:"last_modified_by,omitempty"`
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

type UpdateColumnsRequest struct {
	Id    interface{} `json:"id" binding:"required" example:"row-id-123"`
	Value interface{} `json:"value" binding:"required" example:"New Value"`
}

type BulkUpdateColumnsRequest struct {
	ModelID  string                 `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnID string                 `json:"column_id" binding:"required" example:"col-123"`
	Updates  []UpdateColumnsRequest `json:"updates" binding:"required,min=1"`
}

type ResetColumnValuesRequest struct {
	ModelID  string `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnId string `json:"column_id" binding:"required" example:"col-123"`
}

type UpdateColumnValueRequest struct {
	Id     interface{} `json:"id"`
	Column string      `json:"column"`
	Value  interface{} `json:"value"`
}

type MergeColumnsRequest struct {
	ModelID            string   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Columns            []string `json:"columns" binding:"required,min=2,dive,required" example:"123e4567-e89b-12d3-a456-426614174000,223e4567-e89b-12d3-a456-426614174000"`
	NewColumnTitle     string   `json:"new_column_title,omitempty" example:"Merged Column"`
	MergeFormat        string   `json:"merge_format" binding:"required,oneof=space comma dash custom" example:"comma"`
	CustomSeparator    string   `json:"custom_separator,omitempty" binding:"required_if=MergeFormat custom" example:";"`
	KeepOriginalColumn bool     `json:"keep_original_column" example:"true"`
	AddAtEnd           bool     `json:"add_at_end" example:"false"`
}

type MergeColumnsResponse struct {
	TotalScanned     int    `json:"total_scanned"`
	TotalUpdated     int    `json:"total_updated"`
	TotalSkipped     int    `json:"total_skipped"`
	TotalRows        int    `json:"total_rows"`
	TotalRowsUpdated int    `json:"total_rows_updated"`
	TotalRowsSkipped int    `json:"total_rows_skipped"`
	GeneratedColumn  string `json:"generated_column_name"`
}

type TrimWhitespaceRequest struct {
	ModelID  string   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Columns  []string `json:"columns" binding:"required,min=1,dive,required" example:"123e4567-e89b-12d3-a456-426614174000,223e4567-e89b-12d3-a456-426614174000"`
	TrimMode string   `json:"trim_mode" binding:"required,oneof=trim_both trim_leading trim_trailing collapse_spaces" example:"trim_both"`
}

type TrimWhitespaceResponse struct {
	TotalScanned     int `json:"total_scanned"`
	TotalUpdated     int `json:"total_updated"`
	TotalSkipped     int `json:"total_skipped"`
	TotalRows        int `json:"total_rows"`
	TotalRowsUpdated int `json:"total_rows_updated"`
	TotalRowsSkipped int `json:"total_rows_skipped"`
}

type CaseNormalizationRequest struct {
	ModelID    string   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Columns    []string `json:"columns" binding:"required,min=1,dive,required" example:"123e4567-e89b-12d3-a456-426614174000,223e4567-e89b-12d3-a456-426614174000"`
	CaseFormat string   `json:"case_format" binding:"required,oneof=lowercase uppercase title_case sentence_case" example:"title_case"`
}

type CaseNormalizationResponse struct {
	TotalScanned     int `json:"total_scanned"`
	TotalUpdated     int `json:"total_updated"`
	TotalSkipped     int `json:"total_skipped"`
	TotalRows        int `json:"total_rows"`
	TotalRowsUpdated int `json:"total_rows_updated"`
	TotalRowsSkipped int `json:"total_rows_skipped"`
}

type FindReplaceRequest struct {
	ModelID      string   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Columns      []string `json:"columns" binding:"required,min=1,dive,required" example:"col-123,col-456"`
	FindValue    string   `json:"find_value" binding:"required" example:"NY"`
	ReplaceValue string   `json:"replace_value" example:"New York"`
	MatchType    string   `json:"match_type" binding:"required,oneof=match_case ignore_case match_entire_value" example:"ignore_case"`
}

type FindReplaceResponse struct {
	TotalScanned     int `json:"total_scanned"`
	TotalMatched     int `json:"total_matched"`
	TotalUpdated     int `json:"total_updated"`
	TotalSkipped     int `json:"total_skipped"`
	TotalRows        int `json:"total_rows"`
	TotalRowsUpdated int `json:"total_rows_updated"`
	TotalRowsSkipped int `json:"total_rows_skipped"`
}

type RemoveSpecialCharactersRequest struct {
	ModelID               string   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Columns               []string `json:"columns" binding:"required,min=1,dive,required" example:"123e4567-e89b-12d3-a456-426614174000,223e4567-e89b-12d3-a456-426614174000"`
	SpecialCharactersType string   `json:"special_characters_type" binding:"required,oneof=symbols currency_symbols brackets punctuation custom" example:"symbols"`
	CustomCharacter       []string `json:"custom,omitempty" binding:"required_if=SpecialCharactersType custom"`
}

type RemoveSpecialCharactersResponse struct {
	TotalScanned     int `json:"total_scanned"`
	TotalMatched     int `json:"total_matched"`
	TotalUpdated     int `json:"total_updated"`
	TotalSkipped     int `json:"total_skipped"`
	TotalRows        int `json:"total_rows"`
	TotalRowsUpdated int `json:"total_rows_updated"`
	TotalRowsSkipped int `json:"total_rows_skipped"`
}

type RemoveFormattingRequest struct {
	ModelID       string   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Columns       []string `json:"columns" binding:"required,min=1,dive,required" example:"phone"`
	Formatting    string   `json:"formatting" binding:"required,oneof=currency percentage separator phone date custom" example:"phone"`
	CustomPattern []string `json:"custom_pattern,omitempty" binding:"required_if=Formatting custom,dive,required" example:"[\"-\",\"_\"]"`
}

type RemoveFormattingResponse struct {
	ScannedRecords int `json:"scanned_records"`
	UpdatedRecords int `json:"updated_records"`
	SkippedRecords int `json:"skipped_records"`
	FailedRecords  int `json:"failed_records"`
}

// ExtractSubstringRequest defines the payload for extracting substrings from a single column
type ExtractSubstringRequest struct {
	ModelID            string `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnId           string `json:"column_id" binding:"required" example:"col-123"`
	ExtractionMethod   string `json:"extraction_method" binding:"required,oneof=extraction_type between_characters" example:"extraction_type"`
	ExtractionType     string `json:"extraction_type" binding:"omitempty,oneof=email keywords mentions tags url domain emoji phone prefix" example:"email"`
	StartAfter         string `json:"start_after" example:"@"`
	EndBefore          string `json:"end_before" example:"."`
	KeepOriginalColumn bool   `json:"keep_original_column" example:"true"`
	AddAtEnd           bool   `json:"add_at_end" example:"false"`
}

// ExtractSubstringResponse summarizes the result of the extract-substring operation
type ExtractSubstringResponse struct {
	Column          string `json:"column"`
	GeneratedColumn string `json:"generated_column_name"`
	ExtractionType  string `json:"extraction_type"`
	ScannedRecords  int    `json:"scanned_records"`
	UpdatedRecords  int    `json:"updated_records"`
	SkippedRecords  int    `json:"skipped_records"`
	FailedRecords   int    `json:"failed_records"`
}

type RemoveDuplicatesRequest struct {
	ModelID   string   `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Columns   []string `json:"columns" binding:"required,min=1,dive,required" example:"col-123,col-456"`
	Duplicate string   `json:"duplicate" binding:"required,oneof=remove_row remove_duplicates remove_duplicates_matchCase" example:"remove_row"`
	KeepRule  string   `json:"keep_rule" binding:"required,oneof=keep_first keep_last keep_latest_updated" example:"keep_first"`
}

type RemoveDuplicatesResponse struct {
	TotalRowsAffected  int64 `json:"total_rows_affected"`
	TotalDuplicateRows int64 `json:"total_duplicate_rows"`
}

type ColumnSplitRequest struct {
	ModelID      uuid.UUID      `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	ColumnID     uuid.UUID      `json:"column_id" binding:"required" example:"223e4567-e89b-12d3-a456-426614174000"`
	SplitBy      SplitByRequest `json:"split_by" binding:"required"`
	KeepOriginal bool           `json:"keep_original"`
	Where        string         `json:"where" binding:"required,oneof=next end" example:"next"`
	Limit        *int           `json:"limit,omitempty" binding:"omitempty,gte=2" example:"3"`
}

func (r *ColumnSplitRequest) UnmarshalJSON(data []byte) error {
	type Alias ColumnSplitRequest
	aux := struct {
		Limit interface{} `json:"limit"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Limit != nil {
		switch v := aux.Limit.(type) {
		case float64:
			// Reject non-integer numeric values
			if v != math.Trunc(v) {
				return fmt.Errorf("invalid limit value: non-integer %v", v)
			}
			val := int(v)
			r.Limit = &val
		case string:
			val, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("invalid limit value: %s", v)
			}
			r.Limit = &val
		default:
			return fmt.Errorf("invalid type for limit: %T", aux.Limit)
		}
	}
	return nil
}

type SplitByRequest struct {
	Type   string                 `json:"type" binding:"required,oneof=separator fixed_length pattern" example:"separator"`
	Config map[string]interface{} `json:"config" binding:"required"`
}

type ColumnSplitResponse struct {
	Message        string   `json:"message"`
	CreatedColumns []string `json:"createdColumns"`
}

type FuzzyDuplicatesRequest struct {
	ModelID            string            `json:"model_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Columns            []string          `json:"columns" binding:"required,min=1,dive,required" example:"col-123"`
	Threshold          string            `json:"threshold" binding:"required,oneof=low medium high" example:"medium"`
	Duplicate          string            `json:"duplicate" binding:"required,oneof=remove_row remove_duplicates" example:"remove_row"`
	KeepRule           string            `json:"keep_rule" binding:"required,oneof=keep_first keep_last keep_latest_updated" example:"keep_first"`
	DeduplicationMode  string            `json:"deduplication_mode,omitempty" binding:"omitempty,oneof=automatic manual" example:"automatic"`
	RowActions         map[string]string `json:"row_actions,omitempty"`
}

type FuzzyDuplicatesResponse struct {
	TotalRowsAffected  int64 `json:"total_rows_affected"`
	TotalDuplicateRows int64 `json:"total_duplicate_rows"`
}
