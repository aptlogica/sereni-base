package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type SubscriptionPlan struct {
	ID           uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Name         string    `db:"name" json:"name,omitempty" mapstructure:"name"`
	Slug         string    `db:"slug" json:"slug,omitempty" mapstructure:"slug"`
	Description  *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	PriceMonthly *float64  `db:"price_monthly" json:"price_monthly,omitempty" mapstructure:"price_monthly"`
	PriceYearly  *float64  `db:"price_yearly" json:"price_yearly,omitempty" mapstructure:"price_yearly"`
	Currency     string    `db:"currency" json:"currency,omitempty" mapstructure:"currency"`

	// Limits
	MaxWorkspaces        *int `db:"max_workspaces" json:"max_workspaces,omitempty" mapstructure:"max_workspaces"`
	MaxBasesPerWorkspace *int `db:"max_bases_per_workspace" json:"max_bases_per_workspace,omitempty" mapstructure:"max_bases_per_workspace"`
	MaxTablesPerBase     *int `db:"max_tables_per_base" json:"max_tables_per_base,omitempty" mapstructure:"max_tables_per_base"`
	MaxRowsPerTable      *int `db:"max_rows_per_table" json:"max_rows_per_table,omitempty" mapstructure:"max_rows_per_table"`
	MaxCollaborators     *int `db:"max_collaborators" json:"max_collaborators,omitempty" mapstructure:"max_collaborators"`
	MaxAPICallsPerHour   *int `db:"max_api_calls_per_hour" json:"max_api_calls_per_hour,omitempty" mapstructure:"max_api_calls_per_hour"`
	StorageLimitGB       *int `db:"storage_limit_gb" json:"storage_limit_gb,omitempty" mapstructure:"storage_limit_gb"`

	// Features
	Features string `db:"features" json:"features,omitempty" mapstructure:"features"`
	IsActive bool   `db:"is_active" json:"is_active,omitempty" mapstructure:"is_active"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (SubscriptionPlan) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".subscription_plans", prefix)
}

func (tbl SubscriptionPlan) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "name", DataType: "varchar", NotNull: true},
			{Name: "slug", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "description", DataType: "text"},
			{Name: "price_monthly", DataType: "decimal(10,2)"},
			{Name: "price_yearly", DataType: "decimal(10,2)"},
			{Name: "currency", DataType: "varchar", DefaultValue: strPtr("'USD'")},
			{Name: "max_workspaces", DataType: "int"},
			{Name: "max_bases_per_workspace", DataType: "int"},
			{Name: "max_tables_per_base", DataType: "int"},
			{Name: "max_rows_per_table", DataType: "int"},
			{Name: "max_collaborators", DataType: "int"},
			{Name: "max_api_calls_per_hour", DataType: "int"},
			{Name: "storage_limit_gb", DataType: "int"},
			{Name: "features", DataType: "text"},
			{Name: "is_active", DataType: "boolean", DefaultValue: strPtr("true")},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
	}
}
