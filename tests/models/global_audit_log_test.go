package models_test

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGlobalAuditLog_TableName(t *testing.T) {
	log := tenant.GlobalAuditLog{}
	schema := "test_schema"

	tableName := log.TableName(schema)

	assert.Equal(t, `"test_schema".global_audit_logs`, tableName)
}

func TestGlobalAuditLog_Fields(t *testing.T) {
	logID := uuid.New()
	userID := uuid.New()
	requestID := uuid.New()
	resourceType := "workspace"
	resourceID := uuid.New().String()
	ipAddress := "192.168.1.100"
	userAgent := "Mozilla/5.0"
	eventData := `{"action": "create", "details": "Created workspace"}`
	now := time.Now().UTC()

	log := tenant.GlobalAuditLog{
		ID:            logID,
		UserID:        userID,
		EventType:     "workspace.created",
		EventCategory: "workspace",
		ResourceType:  &resourceType,
		ResourceID:    &resourceID,
		IPAddress:     &ipAddress,
		UserAgent:     &userAgent,
		RequestID:     requestID,
		EventData:     &eventData,
		CreatedAt:     now,
	}

	assert.Equal(t, logID, log.ID)
	assert.Equal(t, userID, log.UserID)
	assert.Equal(t, "workspace.created", log.EventType)
	assert.Equal(t, "workspace", log.EventCategory)
	assert.Equal(t, "workspace", *log.ResourceType)
	assert.Equal(t, resourceID, *log.ResourceID)
	assert.Equal(t, "192.168.1.100", *log.IPAddress)
	assert.Equal(t, "Mozilla/5.0", *log.UserAgent)
	assert.Equal(t, requestID, log.RequestID)
}

func TestGlobalAuditLog_EventTypes(t *testing.T) {
	testCases := []struct {
		name     string
		event    string
		category string
	}{
		{"user_login", "user.login", "authentication"},
		{"user_logout", "user.logout", "authentication"},
		{"workspace_created", "workspace.created", "workspace"},
		{"workspace_deleted", "workspace.deleted", "workspace"},
		{"base_created", "base.created", "base"},
		{"record_updated", "record.updated", "record"},
		{"member_invited", "member.invited", "member"},
		{"permission_changed", "permission.changed", "permission"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log := tenant.GlobalAuditLog{
				ID:            uuid.New(),
				UserID:        uuid.New(),
				EventType:     tc.event,
				EventCategory: tc.category,
				RequestID:     uuid.New(),
				CreatedAt:     time.Now().UTC(),
			}

			assert.Equal(t, tc.event, log.EventType)
			assert.Equal(t, tc.category, log.EventCategory)
		})
	}
}

func TestGlobalAuditLog_WithResourceTracking(t *testing.T) {
	resourceType := "base"
	resourceID := uuid.New().String()

	log := tenant.GlobalAuditLog{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		EventType:     "base.updated",
		EventCategory: "base",
		ResourceType:  &resourceType,
		ResourceID:    &resourceID,
		RequestID:     uuid.New(),
		CreatedAt:     time.Now().UTC(),
	}

	assert.NotNil(t, log.ResourceType)
	assert.NotNil(t, log.ResourceID)
	assert.Equal(t, "base", *log.ResourceType)
}

func TestGlobalAuditLog_WithoutResourceTracking(t *testing.T) {
	log := tenant.GlobalAuditLog{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		EventType:     "user.login",
		EventCategory: "authentication",
		ResourceType:  nil,
		ResourceID:    nil,
		RequestID:     uuid.New(),
		CreatedAt:     time.Now().UTC(),
	}

	assert.Nil(t, log.ResourceType)
	assert.Nil(t, log.ResourceID)
}

func TestGlobalAuditLog_WithEventData(t *testing.T) {
	eventData := `{"old_value": "active", "new_value": "inactive", "reason": "user request"}`

	log := tenant.GlobalAuditLog{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		EventType:     "status.changed",
		EventCategory: "system",
		EventData:     &eventData,
		RequestID:     uuid.New(),
		CreatedAt:     time.Now().UTC(),
	}

	assert.NotNil(t, log.EventData)
	assert.Contains(t, *log.EventData, "old_value")
	assert.Contains(t, *log.EventData, "new_value")
}

func TestGlobalAuditLog_TableSchema(t *testing.T) {
	log := tenant.GlobalAuditLog{}
	schema := "test_schema"

	tableSchema := log.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".global_audit_logs`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
	assert.NotEmpty(t, tableSchema.ForeignKeys)
}
