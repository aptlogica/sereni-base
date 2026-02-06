package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHook_TableName(t *testing.T) {
	hook := tenant.Hook{}
	schema := "test_schema"

	tableName := hook.TableName(schema)

	assert.Equal(t, `"test_schema".hooks`, tableName)
}

func TestHook_Fields(t *testing.T) {
	hookID := uuid.New()
	modelID := uuid.New().String()
	baseID := uuid.New().String()
	desc := "Webhook for record creation"
	url := "https://api.example.com/webhook"
	headers := `{"Authorization": "Bearer token"}`
	payload := `{"event": "{{event}}"}`
	condition := `status == "active"`
	now := time.Now().UTC()

	hook := tenant.Hook{
		ID:              hookID,
		ModelID:         &modelID,
		BaseID:          baseID,
		Title:           "Create Record Hook",
		Description:     &desc,
		Type:            "webhook",
		Event:           "after_create",
		Operation:       "POST",
		URL:             &url,
		Headers:         &headers,
		Payload:         &payload,
		Condition:       &condition,
		AsyncProcessing: true,
		Retries:         3,
		RetryInterval:   60,
		Timeout:         30,
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.Equal(t, hookID, hook.ID)
	assert.Equal(t, modelID, *hook.ModelID)
	assert.Equal(t, baseID, hook.BaseID)
	assert.Equal(t, "Create Record Hook", hook.Title)
	assert.Equal(t, "webhook", hook.Type)
	assert.Equal(t, "after_create", hook.Event)
	assert.Equal(t, "POST", hook.Operation)
	assert.True(t, hook.AsyncProcessing)
	assert.Equal(t, 3, hook.Retries)
	assert.Equal(t, 60, hook.RetryInterval)
	assert.Equal(t, 30, hook.Timeout)
	assert.True(t, hook.Active)
}

func TestHook_HookEvents(t *testing.T) {
	testCases := []struct {
		name      string
		event     string
		operation string
	}{
		{"after_create", "after_create", "POST"},
		{"after_update", "after_update", "POST"},
		{"after_delete", "after_delete", "POST"},
		{"before_create", "before_create", "POST"},
		{"before_update", "before_update", "POST"},
		{"before_delete", "before_delete", "POST"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hook := tenant.Hook{
				ID:        uuid.New(),
				BaseID:    uuid.New().String(),
				Title:     "Event Hook",
				Type:      "webhook",
				Event:     tc.event,
				Operation: tc.operation,
				Active:    true,
			}

			assert.Equal(t, tc.event, hook.Event)
			assert.Equal(t, tc.operation, hook.Operation)
		})
	}
}

func TestHook_InactiveHook(t *testing.T) {
	hook := tenant.Hook{
		ID:        uuid.New(),
		BaseID:    uuid.New().String(),
		Title:     "Disabled Hook",
		Type:      "webhook",
		Event:     "after_create",
		Operation: "POST",
		Active:    false,
	}

	assert.False(t, hook.Active)
}

func TestHook_WithRetrySettings(t *testing.T) {
	hook := tenant.Hook{
		ID:              uuid.New(),
		BaseID:          uuid.New().String(),
		Title:           "Retry Hook",
		Type:            "webhook",
		Event:           "after_create",
		Operation:       "POST",
		AsyncProcessing: true,
		Retries:         5,
		RetryInterval:   120,
		Timeout:         60,
		Active:          true,
	}

	assert.True(t, hook.AsyncProcessing)
	assert.Equal(t, 5, hook.Retries)
	assert.Equal(t, 120, hook.RetryInterval)
	assert.Equal(t, 60, hook.Timeout)
}

func TestHook_SyncHook(t *testing.T) {
	hook := tenant.Hook{
		ID:              uuid.New(),
		BaseID:          uuid.New().String(),
		Title:           "Sync Hook",
		Type:            "webhook",
		Event:           "before_create",
		Operation:       "POST",
		AsyncProcessing: false,
		Timeout:         10,
		Active:          true,
	}

	assert.False(t, hook.AsyncProcessing)
}

func TestHook_TableSchema(t *testing.T) {
	hook := tenant.Hook{}
	schema := "test_schema"

	tableSchema := hook.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".hooks`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
