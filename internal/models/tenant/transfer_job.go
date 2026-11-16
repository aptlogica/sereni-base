// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg/models"
)

// Transfer job types stored in transfer_jobs.type
const (
	TransferJobTypeExport = "export"
	TransferJobTypeImport = "import"
)

// Transfer job states stored in transfer_jobs.status
const (
	TransferJobQueued    = "queued"
	TransferJobRunning   = "running"
	TransferJobCompleted = "completed"
	TransferJobFailed    = "failed"
)

// TransferJob is one background export of a base to a .sbase.zip, or one background import of an uploaded file as a new base. An export zip is kept until ExpiresAt.
type TransferJob struct {
	ID string `db:"id" json:"id" mapstructure:"id"`
	// Type is "export" or "import"
	Type        string     `db:"type" json:"type" mapstructure:"type"`
	BaseID      string     `db:"base_id" json:"base_id" mapstructure:"base_id"`
	BaseTitle   string     `db:"base_title" json:"base_title" mapstructure:"base_title"`
	WorkspaceID string     `db:"workspace_id" json:"workspace_id" mapstructure:"workspace_id"`
	UserID      string     `db:"user_id" json:"user_id" mapstructure:"user_id"`
	Status      string     `db:"status" json:"status" mapstructure:"status"`
	Progress    int        `db:"progress" json:"progress" mapstructure:"progress"`
	FileName    string     `db:"file_name" json:"file_name" mapstructure:"file_name"`
	FileURL     string     `db:"file_url" json:"file_url" mapstructure:"file_url"`
	SizeBytes   int64      `db:"size_bytes" json:"size_bytes" mapstructure:"size_bytes"`
	Error       string     `db:"error" json:"error" mapstructure:"error"`
	Result      string     `db:"result" json:"result" mapstructure:"result"`
	ExpiresAt   *time.Time `db:"expires_at" json:"expires_at,omitempty" mapstructure:"expires_at"`
	CreatedAt   time.Time  `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt   time.Time  `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
}

func (TransferJob) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".transfer_jobs", prefix)
}

func (tbl TransferJob) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "type", DataType: "varchar", NotNull: true, DefaultValue: StrPtr("'export'")},
			{Name: "base_id", DataType: "varchar"},
			{Name: "base_title", DataType: "varchar"},
			{Name: "workspace_id", DataType: "varchar"},
			{Name: "user_id", DataType: "varchar", NotNull: true},
			{Name: "status", DataType: "varchar", NotNull: true, DefaultValue: StrPtr("'queued'")},
			{Name: "file_name", DataType: "varchar"},
			{Name: "file_url", DataType: "text"},
			{Name: "size_bytes", DataType: "double precision", DefaultValue: StrPtr("0")},
			{Name: "error", DataType: "text"},
			{Name: "result", DataType: "text"},
			{Name: "expires_at", DataType: "timestamp"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_transfer_jobs_user_id", Columns: []string{"user_id"}},
			{Name: "idx_transfer_jobs_status", Columns: []string{"status"}},
		},
	}
}
