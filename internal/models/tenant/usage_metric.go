package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type UsageMetric struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
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
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "metric_type", DataType: "varchar", NotNull: true},
			{Name: "metric_value", DataType: "integer", DefaultValue: StrPtr("0")},
			CreateTimestampColumn("period_start", false, true),
			CreateTimestampColumn("period_end", false, true),
			CreateTimestampColumn("recorded_at", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_usage_period", Columns: []string{"period_start", "period_end"}},
		},
	}
}
