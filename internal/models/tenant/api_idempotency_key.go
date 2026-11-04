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

// APIIdempotencyKey stores the response of a public API POST sent with an Idempotency-Key header, so a
// retry with the same key gets the same response instead of writing twice. Rows are kept for 24 hours.
type APIIdempotencyKey struct {
	WorkspaceID  string `db:"workspace_id" json:"workspace_id" mapstructure:"workspace_id"`
	Key          string `db:"key" json:"key" mapstructure:"key"`
	RequestHash  string `db:"request_hash" json:"request_hash" mapstructure:"request_hash"`
	Status       string `db:"status" json:"status" mapstructure:"status"`
	StatusCode   *int   `db:"status_code" json:"status_code" mapstructure:"status_code"`
	ResponseBody string `db:"response_body" json:"response_body" mapstructure:"response_body"`

	CreatedAt time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
}

// Values of APIIdempotencyKey.Status
const (
	IdempotencyInProgress = "in_progress"
	IdempotencyDone       = "done"
)

func (APIIdempotencyKey) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".api_idempotency_keys", prefix)
}

func (tbl APIIdempotencyKey) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "workspace_id", DataType: "varchar", NotNull: true},
			{Name: "key", DataType: "varchar(255)", NotNull: true},
			{Name: "request_hash", DataType: "varchar(64)", NotNull: true},
			{Name: "status", DataType: "varchar(16)", NotNull: true, DefaultValue: StrPtr("'in_progress'")},
			{Name: "status_code", DataType: "integer"},
			{Name: "response_body", DataType: "text"},
			CreateTimestampColumn("created_time", true, false),
		},
		PrimaryKey: []string{"workspace_id", "key"},
		Indexes: []models.IndexDefinition{
			{Name: "idx_api_idempotency_keys_created_time", Columns: []string{"created_time"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_api_idempotency_keys_workspace_id",
				Columns:           []string{"workspace_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".workspaces", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
