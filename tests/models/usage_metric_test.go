package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUsageMetric_TableName(t *testing.T) {
	metric := tenant.UsageMetric{}
	schema := "test_schema"

	tableName := metric.TableName(schema)

	assert.Equal(t, `"test_schema".usage_metrics`, tableName)
}

func TestUsageMetric_Fields(t *testing.T) {
	metricID := uuid.New()
	periodStart := time.Now().UTC().Add(-24 * time.Hour)
	periodEnd := time.Now().UTC()
	recordedAt := time.Now().UTC()

	metric := tenant.UsageMetric{
		ID:          metricID,
		MetricType:  "api_requests",
		MetricValue: 15000,
		PeriodStart: &periodStart,
		PeriodEnd:   &periodEnd,
		RecordedAt:  recordedAt,
	}

	assert.Equal(t, metricID, metric.ID)
	assert.Equal(t, "api_requests", metric.MetricType)
	assert.Equal(t, int64(15000), metric.MetricValue)
	assert.NotNil(t, metric.PeriodStart)
	assert.NotNil(t, metric.PeriodEnd)
	assert.Equal(t, recordedAt, metric.RecordedAt)
}

func TestUsageMetric_MetricTypes(t *testing.T) {
	testCases := []struct {
		name        string
		metricType  string
		metricValue int64
	}{
		{"api_requests", "api_requests", 10000},
		{"storage_bytes", "storage_bytes", 5368709120},
		{"active_users", "active_users", 150},
		{"records_created", "records_created", 5000},
		{"records_updated", "records_updated", 2000},
		{"records_deleted", "records_deleted", 100},
		{"webhook_calls", "webhook_calls", 500},
		{"export_operations", "export_operations", 50},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metric := tenant.UsageMetric{
				ID:          uuid.New(),
				MetricType:  tc.metricType,
				MetricValue: tc.metricValue,
				RecordedAt:  time.Now().UTC(),
			}

			assert.Equal(t, tc.metricType, metric.MetricType)
			assert.Equal(t, tc.metricValue, metric.MetricValue)
		})
	}
}

func TestUsageMetric_DailyMetric(t *testing.T) {
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	metric := tenant.UsageMetric{
		ID:          uuid.New(),
		MetricType:  "daily_api_calls",
		MetricValue: 25000,
		PeriodStart: &dayStart,
		PeriodEnd:   &dayEnd,
		RecordedAt:  time.Now().UTC(),
	}

	assert.NotNil(t, metric.PeriodStart)
	assert.NotNil(t, metric.PeriodEnd)
	duration := metric.PeriodEnd.Sub(*metric.PeriodStart)
	assert.Equal(t, 24*time.Hour, duration)
}

func TestUsageMetric_MonthlyMetric(t *testing.T) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	metric := tenant.UsageMetric{
		ID:          uuid.New(),
		MetricType:  "monthly_storage",
		MetricValue: 1073741824, // 1 GB in bytes
		PeriodStart: &monthStart,
		PeriodEnd:   &monthEnd,
		RecordedAt:  time.Now().UTC(),
	}

	assert.NotNil(t, metric.PeriodStart)
	assert.NotNil(t, metric.PeriodEnd)
	assert.Equal(t, int64(1073741824), metric.MetricValue)
}

func TestUsageMetric_WithoutPeriod(t *testing.T) {
	metric := tenant.UsageMetric{
		ID:          uuid.New(),
		MetricType:  "instant_metric",
		MetricValue: 100,
		PeriodStart: nil,
		PeriodEnd:   nil,
		RecordedAt:  time.Now().UTC(),
	}

	assert.Nil(t, metric.PeriodStart)
	assert.Nil(t, metric.PeriodEnd)
}

func TestUsageMetric_ZeroValue(t *testing.T) {
	metric := tenant.UsageMetric{
		ID:          uuid.New(),
		MetricType:  "inactive_users",
		MetricValue: 0,
		RecordedAt:  time.Now().UTC(),
	}

	assert.Equal(t, int64(0), metric.MetricValue)
}

func TestUsageMetric_LargeValue(t *testing.T) {
	metric := tenant.UsageMetric{
		ID:          uuid.New(),
		MetricType:  "total_storage_bytes",
		MetricValue: 1099511627776, // 1 TB in bytes
		RecordedAt:  time.Now().UTC(),
	}

	assert.Equal(t, int64(1099511627776), metric.MetricValue)
}

func TestUsageMetric_TableSchema(t *testing.T) {
	metric := tenant.UsageMetric{}
	schema := "test_schema"

	tableSchema := metric.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".usage_metrics`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
}
