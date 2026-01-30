package dto

import (
	"time"

	"github.com/google/uuid"
)

type BaseSubscriptionPlan struct {
	ID                   uuid.UUID `json:"id" example:"b3e1f2c0-1234-4a56-9b8c-abcdef123456" format:"uuid" mapstructure:"id"`
	Name                 string    `json:"name" example:"Pro Plan" format:"string" mapstructure:"name"`
	Slug                 string    `json:"slug" example:"pro-plan" format:"string" mapstructure:"slug"`
	Description          *string   `json:"description,omitempty" example:"Advanced plan with more features" format:"string" mapstructure:"description"`
	Currency             string    `json:"currency,omitempty" example:"USD" format:"string" mapstructure:"currency"`
	MaxWorkspaces        *int      `json:"max_workspaces,omitempty" example:"10" format:"int" mapstructure:"max_workspaces"`
	MaxBasesPerWorkspace *int      `json:"max_bases_per_workspace,omitempty" example:"5" format:"int" mapstructure:"max_bases_per_workspace"`
	MaxTablesPerBase     *int      `json:"max_tables_per_base,omitempty" example:"20" format:"int" mapstructure:"max_tables_per_base"`
	MaxRowsPerTable      *int      `json:"max_rows_per_table,omitempty" example:"100000" format:"int" mapstructure:"max_rows_per_table"`
	MaxCollaborators     *int      `json:"max_collaborators,omitempty" example:"50" format:"int" mapstructure:"max_collaborators"`
	MaxAPICallsPerHour   *int      `json:"max_api_calls_per_hour,omitempty" example:"10000" format:"int" mapstructure:"max_api_calls_per_hour"`
	StorageLimitGB       *int      `json:"storage_limit_gb,omitempty" example:"100" format:"int" mapstructure:"storage_limit_gb"`
	Features             string    `json:"features,omitempty" example:"[\"priority_support\",\"advanced_analytics\"]" format:"string" mapstructure:"features"`
	IsActive             bool      `json:"is_active,omitempty" example:"true" format:"boolean" mapstructure:"is_active"`
	CreatedAt            time.Time `json:"created_time" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"created_time"`
	UpdatedAt            time.Time `json:"last_modified_time" example:"2024-06-01T12:00:00Z" format:"date-time" mapstructure:"last_modified_time"`
}

type PlanInsertion struct {
	BaseSubscriptionPlan
}

func (p *PlanInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                      p.ID,
		"name":                    p.Name,
		"slug":                    p.Slug,
		"description":             p.Description,
		"currency":                p.Currency,
		"max_workspaces":          p.MaxWorkspaces,
		"max_bases_per_workspace": p.MaxBasesPerWorkspace,
		"max_tables_per_base":     p.MaxTablesPerBase,
		"max_rows_per_table":      p.MaxRowsPerTable,
		"max_collaborators":       p.MaxCollaborators,
		"max_api_calls_per_hour":  p.MaxAPICallsPerHour,
		"storage_limit_gb":        p.StorageLimitGB,
		"features":                p.Features,
		"is_active":               p.IsActive,
		"created_time":            p.CreatedAt,
		"last_modified_time":      p.UpdatedAt,
	}
}

type SubscriptionPlanResponse struct {
	BaseSubscriptionPlan
}
