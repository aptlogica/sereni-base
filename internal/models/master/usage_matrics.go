package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type UsageMetric struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	TenantID    uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty" mapstructure:"tenant_id"`
	MetricType  string    `db:"metric_type" json:"metric_type,omitempty" mapstructure:"metric_type"`
	MetricValue int64     `db:"metric_value" json:"metric_value,omitempty" mapstructure:"metric_value"`

	PeriodStart *time.Time `db:"period_start" json:"period_start,omitempty" mapstructure:"period_start"`
	PeriodEnd   *time.Time `db:"period_end" json:"period_end,omitempty" mapstructure:"period_end"`
	RecordedAt  time.Time  `db:"recorded_at" json:"recorded_at,omitempty" mapstructure:"recorded_at"`
}

func (UsageMetric) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".usage_metrics", prefix)
}

func (tbl UsageMetric) TableSchema(prefix string) models.CreateTableRequest {
	null := "NULL"
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "tenant_id", DataType: "uuid", NotNull: true},
			{Name: "metric_type", DataType: "varchar", NotNull: true},
			{Name: "metric_value", DataType: "integer", DefaultValue: strPtr("0")},
			{Name: "period_start", DataType: "timestamp", DefaultValue: &null},
			{Name: "period_end", DataType: "timestamp", DefaultValue: &null},
			{Name: "recorded_at", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_usage_tenant_type", Columns: []string{"tenant_id", "metric_type"}},
			{Name: "idx_usage_period", Columns: []string{"period_start", "period_end"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_usage_tenant",
				Columns:           []string{"tenant_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".tenants", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
