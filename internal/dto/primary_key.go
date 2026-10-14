// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

// UpdatePrimaryKeyRequest is the payload for PUT /table/:id/primary-key; column IDs in key order
type UpdatePrimaryKeyRequest struct {
	ColumnIDs []string `json:"column_ids" binding:"required"`
}

// PrimaryKeyColumn is one field of the table as shown in the primary key settings
type PrimaryKeyColumn struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	ColumnName string `json:"column_name"`
	UIDT       string `json:"uidt"`
	System     bool   `json:"system"`
	InKey      bool   `json:"in_key"`
	Eligible   bool   `json:"eligible"`
	// Why the field cannot be a key field; empty when Eligible
	Reason string `json:"reason,omitempty"`
}

// PrimaryKeyResponse describes the primary key of a table
type PrimaryKeyResponse struct {
	ModelID string `json:"model_id"`
	// Key fields in key order
	ColumnIDs   []string `json:"column_ids"`
	ColumnNames []string `json:"column_names"`
	// Locked is true while link fields store this table's key values
	Locked      bool               `json:"locked"`
	LinkColumns []string           `json:"link_columns"`
	Columns     []PrimaryKeyColumn `json:"columns"`
}

// PrimaryKeyDuplicate is one set of key values shared by several records
type PrimaryKeyDuplicate struct {
	// Key values by column name
	Values map[string]interface{} `json:"values"`
	Count  int64                  `json:"count"`
	// IDs of the records sharing the values (at most 10)
	RecordIDs []int64 `json:"record_ids"`
}

// PrimaryKeyRecord identifies a record that blocks a primary key change
type PrimaryKeyRecord struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	// Titles of the key fields that are empty in this record
	Missing []string `json:"missing,omitempty"`
}

// PrimaryKeyEmptyField counts the records in which one key field is empty
type PrimaryKeyEmptyField struct {
	Column string `json:"column"`
	Title  string `json:"title"`
	Count  int64  `json:"count"`
}

// PrimaryKeyViolationDetails is returned when existing records do not satisfy a new primary key
type PrimaryKeyViolationDetails struct {
	// Message states exactly what failed, e.g. "Code is empty in 1 record (#3 Fin)"
	Message string `json:"message"`
	// Key column names and titles, in key order
	Columns      []string `json:"columns"`
	ColumnTitles []string `json:"column_titles"`
	// Records with at least one empty key field
	NullCount   int64                  `json:"null_count"`
	EmptyFields []PrimaryKeyEmptyField `json:"empty_fields"`
	NullRecords []PrimaryKeyRecord     `json:"null_records"`
	// Key value combinations used by more than one record
	DuplicateGroups int64                 `json:"duplicate_groups"`
	Duplicates      []PrimaryKeyDuplicate `json:"duplicates"`
}
