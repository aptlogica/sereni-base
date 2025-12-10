package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID     uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Slug   string    `db:"slug" json:"slug,omitempty" mapstructure:"slug"`
	Name   string    `db:"name" json:"name,omitempty" mapstructure:"name"`
	Domain *string   `db:"domain" json:"domain,omitempty" mapstructure:"domain"`
	Schema string    `db:"schema_name" json:"schema_name,omitempty" mapstructure:"schema_name"`

	// Status & settings
	Status   string  `db:"status" json:"status,omitempty" mapstructure:"status"`
	Region   string  `db:"region" json:"region,omitempty" mapstructure:"region"`
	Timezone string  `db:"timezone" json:"timezone,omitempty" mapstructure:"timezone"`
	Settings *string `db:"settings" json:"settings,omitempty" mapstructure:"settings"`

	// Subscription info
	SubscriptionTier        string     `db:"subscription_tier" json:"subscription_tier,omitempty" mapstructure:"subscription_tier"`
	SubscriptionStatus      string     `db:"subscription_status" json:"subscription_status,omitempty" mapstructure:"subscription_status"`
	SubscriptionPeriodStart *time.Time `db:"subscription_period_start" json:"subscription_period_start,omitempty" mapstructure:"subscription_period_start"`
	SubscriptionPeriodEnd   *time.Time `db:"subscription_period_end" json:"subscription_period_end,omitempty" mapstructure:"subscription_period_end"`
	TrialEndsAt             *time.Time `db:"trial_ends_at" json:"trial_ends_at,omitempty" mapstructure:"trial_ends_at"`

	// Resource limits
	MaxWorkspaces        int `db:"max_workspaces" json:"max_workspaces,omitempty" mapstructure:"max_workspaces"`
	MaxBasesPerWorkspace int `db:"max_bases_per_workspace" json:"max_bases_per_workspace,omitempty" mapstructure:"max_bases_per_workspace"`
	MaxTablesPerBase     int `db:"max_tables_per_base" json:"max_tables_per_base,omitempty" mapstructure:"max_tables_per_base"`
	MaxRowsPerTable      int `db:"max_rows_per_table" json:"max_rows_per_table,omitempty" mapstructure:"max_rows_per_table"`
	MaxCollaborators     int `db:"max_collaborators" json:"max_collaborators,omitempty" mapstructure:"max_collaborators"`
	MaxAPICallsPerHour   int `db:"max_api_calls_per_hour" json:"max_api_calls_per_hour,omitempty" mapstructure:"max_api_calls_per_hour"`
	StorageLimitGB       int `db:"storage_limit_gb" json:"storage_limit_gb,omitempty" mapstructure:"storage_limit_gb"`

	// Security
	EncryptionEnabled   bool `db:"encryption_enabled" json:"encryption_enabled,omitempty" mapstructure:"encryption_enabled"`
	AuditLoggingEnabled bool `db:"audit_logging_enabled" json:"audit_logging_enabled,omitempty" mapstructure:"audit_logging_enabled"`
	SSOEnabled          bool `db:"sso_enabled" json:"sso_enabled,omitempty" mapstructure:"sso_enabled"`
	IPWhitelistEnabled  bool `db:"ip_whitelist_enabled" json:"ip_whitelist_enabled,omitempty" mapstructure:"ip_whitelist_enabled"`

	// Schema management
	SchemaVersion    string     `db:"schema_version" json:"schema_version,omitempty" mapstructure:"schema_version"`
	LastMigrationRun *time.Time `db:"last_migration_run" json:"last_migration_run,omitempty" mapstructure:"last_migration_run"`

	// Timestamps
	CreatedAt time.Time  `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time  `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty" mapstructure:"deleted_at"`
	IsDeleted bool       `db:"is_deleted" json:"is_deleted,omitempty" mapstructure:"is_deleted"`
}

func (Tenant) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".tenants", prefix)
}

func (tbl Tenant) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "slug", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "name", DataType: "varchar", NotNull: true},
			{Name: "domain", DataType: "varchar", Unique: true},
			{Name: "schema_name", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "status", DataType: "varchar", DefaultValue: strPtr("'active'")},
			{Name: "region", DataType: "varchar", DefaultValue: strPtr("'us-east-1'")},
			{Name: "timezone", DataType: "varchar", DefaultValue: strPtr("'UTC'")},
			{Name: "settings", DataType: "text"},
			{Name: "subscription_tier", DataType: "varchar", DefaultValue: strPtr("'free'")},
			{Name: "subscription_status", DataType: "varchar", DefaultValue: strPtr("'active'")},
			{Name: "subscription_period_start", DataType: "timestamp"},
			{Name: "subscription_period_end", DataType: "timestamp"},
			{Name: "trial_ends_at", DataType: "timestamp"},
			{Name: "max_workspaces", DataType: "integer", DefaultValue: strPtr("1")},
			{Name: "max_bases_per_workspace", DataType: "integer", DefaultValue: strPtr("5")},
			{Name: "max_tables_per_base", DataType: "integer", DefaultValue: strPtr("10")},
			{Name: "max_rows_per_table", DataType: "integer", DefaultValue: strPtr("1000")},
			{Name: "max_collaborators", DataType: "integer", DefaultValue: strPtr("5")},
			{Name: "max_api_calls_per_hour", DataType: "integer", DefaultValue: strPtr("1000")},
			{Name: "storage_limit_gb", DataType: "integer", DefaultValue: strPtr("1")},
			{Name: "encryption_enabled", DataType: "boolean", DefaultValue: strPtr("true")},
			{Name: "audit_logging_enabled", DataType: "boolean", DefaultValue: strPtr("false")},
			{Name: "sso_enabled", DataType: "boolean", DefaultValue: strPtr("false")},
			{Name: "ip_whitelist_enabled", DataType: "boolean", DefaultValue: strPtr("false")},
			{Name: "schema_version", DataType: "varchar", DefaultValue: strPtr("'1.0.0'")},
			{Name: "last_migration_run", DataType: "timestamp"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "deleted_at", DataType: "timestamp"},
			{Name: "is_deleted", DataType: "boolean", DefaultValue: strPtr("false")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_tenants_slug", Columns: []string{"slug"}},
			{Name: "idx_tenants_status", Columns: []string{"status"}},
			{Name: "idx_tenants_schema", Columns: []string{"schema_name"}},
		},
	}
}
