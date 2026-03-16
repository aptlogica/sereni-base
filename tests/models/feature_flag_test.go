package models_test

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFeatureFlag_TableName(t *testing.T) {
	flag := tenant.FeatureFlag{}
	schema := "test_schema"

	tableName := flag.TableName(schema)

	assert.Equal(t, `"test_schema".shared.feature_flags`, tableName)
}

func TestFeatureFlag_Fields(t *testing.T) {
	flagID := uuid.New()
	desc := "Enable new UI features"
	now := time.Now().UTC()

	flag := tenant.FeatureFlag{
		ID:             flagID,
		Name:           "new_ui_enabled",
		Description:    &desc,
		DefaultEnabled: true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	assert.Equal(t, flagID, flag.ID)
	assert.Equal(t, "new_ui_enabled", flag.Name)
	assert.Equal(t, "Enable new UI features", *flag.Description)
	assert.True(t, flag.DefaultEnabled)
	assert.Equal(t, now, flag.CreatedAt)
}

func TestFeatureFlag_CommonFlags(t *testing.T) {
	testCases := []struct {
		name           string
		flagName       string
		defaultEnabled bool
	}{
		{"api_v2", "api_v2_enabled", true},
		{"dark_mode", "dark_mode", false},
		{"advanced_search", "advanced_search", true},
		{"beta_features", "beta_features", false},
		{"new_dashboard", "new_dashboard", false},
		{"export_feature", "export_feature", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag := tenant.FeatureFlag{
				ID:             uuid.New(),
				Name:           tc.flagName,
				DefaultEnabled: tc.defaultEnabled,
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			}

			assert.Equal(t, tc.flagName, flag.Name)
			assert.Equal(t, tc.defaultEnabled, flag.DefaultEnabled)
		})
	}
}

func TestFeatureFlag_DisabledByDefault(t *testing.T) {
	flag := tenant.FeatureFlag{
		ID:             uuid.New(),
		Name:           "experimental_feature",
		DefaultEnabled: false,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	assert.False(t, flag.DefaultEnabled)
}

func TestFeatureFlag_EnabledByDefault(t *testing.T) {
	flag := tenant.FeatureFlag{
		ID:             uuid.New(),
		Name:           "stable_feature",
		DefaultEnabled: true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	assert.True(t, flag.DefaultEnabled)
}

func TestFeatureFlag_WithoutDescription(t *testing.T) {
	flag := tenant.FeatureFlag{
		ID:             uuid.New(),
		Name:           "undocumented_flag",
		Description:    nil,
		DefaultEnabled: false,
	}

	assert.Nil(t, flag.Description)
}

func TestFeatureFlag_TableSchema(t *testing.T) {
	flag := tenant.FeatureFlag{}
	schema := "test_schema"

	tableSchema := flag.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".shared.feature_flags`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
}
