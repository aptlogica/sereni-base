// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

// CreateTableConstraintRequest is the payload for POST /table-constraint/create.
// A unique key needs column_ids; a check constraint needs a condition.
type CreateTableConstraintRequest struct {
	// Optional, check constraints: lets the popup show the final Postgres name before saving
	ID        string   `json:"id"`
	ModelID   string   `json:"model_id" binding:"required"`
	Type      string   `json:"type" binding:"required,oneof=unique check"`
	Title     string   `json:"title" binding:"required"`
	ColumnIDs []string `json:"column_ids"`
	CheckInput
	// Set by the handler from the logged-in user
	CreatedBy string `json:"-"`
}

// UpdateTableConstraintRequest is the payload for PUT /table-constraint/:id; the table and type stay the same
type UpdateTableConstraintRequest struct {
	Title     string   `json:"title" binding:"required"`
	ColumnIDs []string `json:"column_ids"`
	CheckInput
	// Set by the handler from the logged-in user
	UpdatedBy string `json:"-"`
}

// CheckTableConstraintRequest is the payload for POST /table-constraint/check
type CheckTableConstraintRequest struct {
	ModelID string `json:"model_id" binding:"required"`
	// unique (default) or check
	Type      string   `json:"type" binding:"omitempty,oneof=unique check"`
	ColumnIDs []string `json:"column_ids"`
	CheckInput
}

// CheckInput is what a check constraint adds to a request and a response
type CheckInput struct {
	// One of the check categories, e.g. range. Only a label: it doesn't limit the condition
	Category string `json:"category,omitempty"`
	// The SQL condition inside CHECK (...), e.g. age_1790232948 BETWEEN 18 AND 100
	Condition string `json:"condition,omitempty"`
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
	ID             string             `json:"id"`
	Title          string             `json:"title"`
	ConstraintName string             `json:"constraint_name"`
	Columns        []ConstraintColumn `json:"columns"`
	Status         string             `json:"status"`
	// Check constraints only
	CheckInput
	// Schema-qualified Postgres table, for the SQL preview
	TableName string `json:"table_name,omitempty"`
	// The constraint as Postgres has it, from pg_get_constraintdef
	Definition string `json:"definition,omitempty"`
}

// ConstraintCheckResult says whether existing records already repeat values in the chosen columns
type ConstraintCheckResult struct {
	Valid bool `json:"valid"`
	// Set when not valid, e.g. Duplicate data in column "EMAIL". Remove the duplicates and try again.
	Message string `json:"message,omitempty"`
	// Check constraints: how many records break the condition
	Violations int `json:"violations,omitempty"`
}
