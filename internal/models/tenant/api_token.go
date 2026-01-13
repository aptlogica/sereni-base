package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type APIToken struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	UserID      string    `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
	WorkspaceID *string   `db:"workspace_id" json:"workspace_id,omitempty" mapstructure:"workspace_id"`
	BaseID      *string   `db:"base_id" json:"base_id,omitempty" mapstructure:"base_id"`

	Name      string `db:"name" json:"name,omitempty" mapstructure:"name"`
	TokenHash string `db:"token_hash" json:"token_hash,omitempty" mapstructure:"token_hash"`
	Prefix    string `db:"prefix" json:"prefix,omitempty" mapstructure:"prefix"`

	// Permissions and scope
	Permissions *string `db:"permissions" json:"permissions,omitempty" mapstructure:"permissions"`
	Scopes      *string `db:"scopes" json:"scopes,omitempty" mapstructure:"scopes"`

	// Usage limits
	RateLimitPerHour *int `db:"rate_limit_per_hour" json:"rate_limit_per_hour,omitempty" mapstructure:"rate_limit_per_hour"`

	// Status and expiry
	Status     string     `db:"status" json:"status,omitempty" mapstructure:"status"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at,omitempty" mapstructure:"last_used_at"`
	UsageCount int64      `db:"usage_count" json:"usage_count,omitempty" mapstructure:"usage_count"`
	ExpiresAt  *time.Time `db:"expires_at" json:"expires_at,omitempty" mapstructure:"expires_at"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (APIToken) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".api_tokens", prefix)
}

func (tbl APIToken) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "user_id", DataType: "varchar", NotNull: true},
			{Name: "workspace_id", DataType: "varchar"},
			{Name: "base_id", DataType: "varchar"},
			{Name: "name", DataType: "varchar", NotNull: true},
			{Name: "token_hash", DataType: "varchar", NotNull: true},
			{Name: "prefix", DataType: "varchar", NotNull: true},
			{Name: "permissions", DataType: "text"},
			{Name: "scopes", DataType: "text"},
			{Name: "rate_limit_per_hour", DataType: "integer"},
			{Name: "status", DataType: "varchar", DefaultValue: StrPtr("'active'")},
			{Name: "last_used_at", DataType: "timestamp", NotNull: false},
			{Name: "usage_count", DataType: "integer", DefaultValue: StrPtr("0")},
			{Name: "metadata", DataType: "jsonb", NotNull: false},
			{Name: "expires_at", DataType: "timestamp"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_api_tokens_user_id", Columns: []string{"user_id"}},
			{Name: "idx_api_tokens_workspace_id", Columns: []string{"workspace_id"}},
			{Name: "idx_api_tokens_base_id", Columns: []string{"base_id"}},
			{Name: "idx_api_tokens_status", Columns: []string{"status"}},
			{Name: "idx_api_tokens_prefix", Columns: []string{"prefix"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_api_tokens_workspace_id",
				Columns:           []string{"workspace_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".workspaces", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
			{
				Name:              "fk_api_tokens_base_id",
				Columns:           []string{"base_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".bases", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
