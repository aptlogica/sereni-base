// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

// TableConstraintOptions are the settings of a unique key
type TableConstraintOptions struct {
	// Empty values count as equal (NULLS NOT DISTINCT), so only one record may leave the columns empty
	NullsNotDistinct bool `json:"nulls_not_distinct"`
}

// CreateTableConstraintRequest is the payload for POST /table-constraint/create
type CreateTableConstraintRequest struct {
	ModelID   string                 `json:"model_id" binding:"required"`
	Type      string                 `json:"type" binding:"required,oneof=unique"`
	Title     string                 `json:"title" binding:"required"`
	ColumnIDs []string               `json:"column_ids" binding:"required,min=1"`
	Options   TableConstraintOptions `json:"options"`
	// Set by the handler from the logged-in user
	CreatedBy string `json:"-"`
}

// UpdateTableConstraintRequest is the payload for PUT /table-constraint/:id; the table and type stay the same
type UpdateTableConstraintRequest struct {
	Title     string                 `json:"title" binding:"required"`
	ColumnIDs []string               `json:"column_ids" binding:"required,min=1"`
	Options   TableConstraintOptions `json:"options"`
	// Set by the handler from the logged-in user
	UpdatedBy string `json:"-"`
}

// CheckTableConstraintRequest is the payload for POST /table-constraint/check
type CheckTableConstraintRequest struct {
	ModelID   string                 `json:"model_id" binding:"required"`
	ColumnIDs []string               `json:"column_ids" binding:"required,min=1"`
	Options   TableConstraintOptions `json:"options"`
}

// ConstraintColumn is a column of a constraint, resolved from its ID
type ConstraintColumn struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Constraint status: whether Postgres still has the constraint
const (
	ConstraintStatusOK      = "ok"
	ConstraintStatusMissing = "missing"
)

// TableConstraintResponse is what the settings panel needs to show and edit a constraint
type TableConstraintResponse struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	ConstraintName string                 `json:"constraint_name"`
	Columns        []ConstraintColumn     `json:"columns"`
	Options        TableConstraintOptions `json:"options"`
	Status         string                 `json:"status"`
}

// ConstraintCheckResult says whether existing records already repeat values in the chosen columns
type ConstraintCheckResult struct {
	Valid bool `json:"valid"`
	// Set when not valid, e.g. Duplicate data in column "EMAIL". Remove the duplicates and try again.
	Message string `json:"message,omitempty"`
}
