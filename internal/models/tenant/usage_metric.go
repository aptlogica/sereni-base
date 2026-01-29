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

// createTimestampColumn creates a timestamp column definition with optional null default
func createTimestampColumn(name string, notNull bool, useNull bool) models.ColumnDefinition {
	null := "NULL"
	var defaultVal *string
	if useNull {
		defaultVal = &null
	} else if notNull {
		defaultVal = StrPtr("CURRENT_TIMESTAMP")
	}
	return models.ColumnDefinition{Name: name, DataType: "timestamp", NotNull: notNull, DefaultValue: defaultVal}
}

func (tbl UsageMetric) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "metric_type", DataType: "varchar", NotNull: true},
			{Name: "metric_value", DataType: "integer", DefaultValue: StrPtr("0")},
			createTimestampColumn("period_start", false, true),
			createTimestampColumn("period_end", false, true),
			createTimestampColumn("recorded_at", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_usage_period", Columns: []string{"period_start", "period_end"}},
		},
	}
}
