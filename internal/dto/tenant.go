package dto

import (
	"time"

	"github.com/google/uuid"
)

type TenantRequest struct {
	UserID   uuid.UUID `json:"user_id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid"`
	TenantID uuid.UUID `json:"tenant_id" example:"c9e2a6a0-5678-4f34-9cd2-fedcba654321" format:"uuid"`
}

type TenantInsertion struct {
	ID     uuid.UUID `json:"id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	Slug   string    `json:"slug" example:"acme-corp" format:"string" mapstructure:"slug"`
	Name   string    `json:"name" example:"Acme Corporation" format:"string" mapstructure:"name"`
	Schema string    `json:"schema_name" example:"tenant_acme" format:"string" mapstructure:"schema_name"`
}

func (t *TenantInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":          t.ID,
		"slug":        t.Slug,
		"name":        t.Name,
		"schema_name": t.Schema,
	}
}

type TenantSubscriptionInsertion struct {
	ID        uuid.UUID `json:"id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	TenantID  uuid.UUID `json:"tenant_id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"tenant_id"`
	PlanID    uuid.UUID `json:"plan_id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"plan_id"`
	Status    string    `json:"status" example:"active" format:"string" mapstructure:"status"`
	CreatedAt time.Time `json:"created_time" example:"1700000000" format:"int64" mapstructure:"created_time"`
	UpdatedAt time.Time `json:"last_modified_time" example:"1700000000" format:"int64" mapstructure:"last_modified_time"`
}

func (t *TenantSubscriptionInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 t.ID,
		"tenant_id":          t.TenantID,
		"plan_id":            t.PlanID,
		"status":             t.Status,
		"created_time":       t.CreatedAt,
		"last_modified_time": t.UpdatedAt,
	}
}

type TenantMembershipInsertion struct {
	ID          uuid.UUID `json:"id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	TenantID    uuid.UUID `json:"tenant_id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"tenant_id"`
	UserID      uuid.UUID `json:"user_id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"user_id"`
	RoleID      uuid.UUID `json:"role_id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"role_id"`
	Status      string    `json:"status" example:"active" format:"string" mapstructure:"status"`
	Permissions string    `json:"permissions" example:"[]" format:"string" mapstructure:"permissions"`
	CreatedAt   time.Time `json:"created_time" example:"1700000000" format:"int64" mapstructure:"created_time"`
	UpdatedAt   time.Time `json:"last_modified_time" example:"1700000000" format:"int64" mapstructure:"last_modified_time"`
}

func (t *TenantMembershipInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 t.ID,
		"tenant_id":          t.TenantID,
		"user_id":            t.UserID,
		"role_id":            t.RoleID,
		"status":             t.Status,
		"permissions":        t.Permissions,
		"created_time":       t.CreatedAt,
		"last_modified_time": t.UpdatedAt,
	}
}

type TenantResponse struct {
	ID        uuid.UUID `json:"id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	Name      string    `json:"name" example:"Acme Corp" format:"string" mapstructure:"name"`
	Schema    string    `json:"schema_name" example:"acme_schema" format:"string" mapstructure:"schema_name"`
	CreatedAt time.Time `json:"created_time" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
	UpdatedAt time.Time `json:"last_modified_time" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"last_modified_time"`

	Subscription *TenantSubscriptionResponse `json:"subscription,omitempty" mapstructure:"subscription"`
	Membership   *TenantMembershipResponse   `json:"membership,omitempty" mapstructure:"membership"`
}

type TenantSubscriptionResponse struct {
	ID        uuid.UUID `json:"id" mapstructure:"id" format:"uuid"`
	TenantID  uuid.UUID `json:"tenant_id" mapstructure:"tenant_id" format:"uuid"`
	PlanID    uuid.UUID `json:"plan_id" mapstructure:"plan_id" format:"uuid"`
	Status    string    `json:"status" mapstructure:"status" format:"string"`
	CreatedAt time.Time `json:"created_time" mapstructure:"created_time" format:"date-time"`
	UpdatedAt time.Time `json:"last_modified_time" mapstructure:"last_modified_time" format:"date-time"`
}

type TenantMembershipResponse struct {
	ID          uuid.UUID `json:"id" mapstructure:"id" format:"uuid"`
	TenantID    uuid.UUID `json:"tenant_id" mapstructure:"tenant_id" format:"uuid"`
	UserID      uuid.UUID `json:"user_id" mapstructure:"user_id" format:"uuid"`
	RoleID      uuid.UUID `json:"role_id" mapstructure:"role_id" format:"uuid"`
	Status      string    `json:"status" mapstructure:"status" format:"string"`
	Permissions string    `json:"permissions" mapstructure:"permissions" format:"string"`
	CreatedAt   time.Time `json:"created_time" mapstructure:"created_time" format:"date-time"`
	UpdatedAt   time.Time `json:"last_modified_time" mapstructure:"last_modified_time" format:"date-time"`
}

type UpdateTenantRequest struct {
	Name      *string   `db:"name" json:"name,omitempty" mapstructure:"name"`
	Region    string    `db:"region" json:"region,omitempty" mapstructure:"region"`
	Timezone  string    `db:"timezone" json:"timezone,omitempty" mapstructure:"timezone"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (a UpdateTenantRequest) Map() map[string]interface{} {
	m := make(map[string]interface{})

	if a.Name != nil {
		m["name"] = *a.Name
	}
	if a.Region != "" {
		m["region"] = a.Region
	}
	if a.Timezone != "" {
		m["timezone"] = a.Timezone
	}
	if !a.UpdatedAt.IsZero() {
		m["last_modified_time"] = a.UpdatedAt
	}
	return m
}
