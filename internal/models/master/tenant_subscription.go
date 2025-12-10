package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type TenantSubscription struct {
	ID       uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	TenantID uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty" mapstructure:"tenant_id"`
	PlanID   uuid.UUID `db:"plan_id" json:"plan_id,omitempty" mapstructure:"plan_id"`
	Status   string    `db:"status" json:"status,omitempty" mapstructure:"status"`

	// Billing cycle
	CurrentPeriodStart *time.Time `db:"current_period_start" json:"current_period_start,omitempty" mapstructure:"current_period_start"`
	CurrentPeriodEnd   *time.Time `db:"current_period_end" json:"current_period_end,omitempty" mapstructure:"current_period_end"`
	TrialStart         *time.Time `db:"trial_start" json:"trial_start,omitempty" mapstructure:"trial_start"`
	TrialEnd           *time.Time `db:"trial_end" json:"trial_end,omitempty" mapstructure:"trial_end"`

	// Payment provider
	PaymentProvider             *string `db:"payment_provider" json:"payment_provider,omitempty" mapstructure:"payment_provider"`
	PaymentProviderSubscription *string `db:"payment_provider_subscription_id" json:"payment_provider_subscription_id,omitempty" mapstructure:"payment_provider_subscription_id"`
	PaymentProviderCustomer     *string `db:"payment_provider_customer_id" json:"payment_provider_customer_id,omitempty" mapstructure:"payment_provider_customer_id"`

	CreatedAt  time.Time  `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt  time.Time  `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
	CanceledAt *time.Time `db:"canceled_at" json:"canceled_at,omitempty" mapstructure:"canceled_at"`
}

func (TenantSubscription) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".tenant_subscriptions", prefix)
}

func (tbl TenantSubscription) TableSchema(prefix string) models.CreateTableRequest {
	null := "NULL"
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "tenant_id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "plan_id", DataType: "uuid", NotNull: true},
			{Name: "status", DataType: "varchar", NotNull: true},
			{Name: "current_period_start", DataType: "timestamp", DefaultValue: &null},
			{Name: "current_period_end", DataType: "timestamp", DefaultValue: &null},
			{Name: "trial_start", DataType: "timestamp", DefaultValue: &null},
			{Name: "trial_end", DataType: "timestamp", DefaultValue: &null},
			{Name: "payment_provider", DataType: "varchar"},
			{Name: "payment_provider_subscription_id", DataType: "varchar"},
			{Name: "payment_provider_customer_id", DataType: "varchar"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "canceled_at", DataType: "timestamp", DefaultValue: &null},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_tenant_subscriptions_tenant", Columns: []string{"tenant_id"}},
			{Name: "idx_tenant_subscriptions_plan", Columns: []string{"plan_id"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_tenant_subscriptions_tenant",
				Columns:           []string{"tenant_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".tenants", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
			{
				Name:              "fk_tenant_subscriptions_plan",
				Columns:           []string{"plan_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".subscription_plans", prefix),
				ReferencedColumns: []string{"id"},
			},
		},
	}
}
