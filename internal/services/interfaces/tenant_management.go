package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"

	"github.com/google/uuid"
)

type TenantManagementService interface {
	// Tenant lifecycle management
	InitializeTenant(ctx context.Context, req dto.TenantRequest, planId uuid.UUID, roleId uuid.UUID) (dto.TenantResponse, error)
	GetTenantByUserId(ctx context.Context, userId uuid.UUID) (dto.TenantResponse, error)
	ValidateSchema(ctx context.Context, schema string) (bool, error)
	// AddUserToTenant(ctx context.Context, schema string, userData dto.AddUserRequest, userId string) (master.Tenant, error)
	// DeactivateTenant(ctx context.Context, tenantID uuid.UUID) error
	// ReactivateTenant(ctx context.Context, tenantID uuid.UUID) error
	// DeleteTenant(ctx context.Context, tenantID uuid.UUID) error

	// // Tenant information
	GetTenant(ctx context.Context, tenantID string) (master.Tenant, error)
	GetTenantBySchema(ctx context.Context, schema string) (master.Tenant, error)
	GetTenantInfoBySchema(ctx context.Context, schema string) (dto.TenantResponse, error)
	UpdateTenantBySchema(ctx context.Context, schema string, req dto.UpdateTenantRequest) (dto.TenantResponse, error)
	// GetTenantBySlug(ctx context.Context, slug string) (master.Tenant, error)
	// GetTenantByDomain(ctx context.Context, domain string) (master.Tenant, error)
	// UpdateTenant(ctx context.Context, tenantID uuid.UUID, updates map[string]interface{}) (master.Tenant, error)

	// // Tenant settings and configuration
	// UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, settings map[string]interface{}) error
	// GetTenantSettings(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error)
	// UpdateTenantLimits(ctx context.Context, tenantID uuid.UUID, limits map[string]int) error
	// GetTenantLimits(ctx context.Context, tenantID uuid.UUID) (map[string]int, error)

	// // Tenant analytics and monitoring
	// GetTenantUsage(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error)
	// GetTenantActivity(ctx context.Context, tenantID uuid.UUID, days int) ([]map[string]interface{}, error)
	// GetTenantHealth(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error)

	// // Bulk operations
	// GetAllTenants(ctx context.Context, filters map[string]interface{}) ([]master.Tenant, error)
	// GetTenantsByStatus(ctx context.Context, status string) ([]master.Tenant, error)
	// GetTenantsByRegion(ctx context.Context, region string) ([]master.Tenant, error)

	// // Schema management
	// CreateTenantSchema(ctx context.Context, tenantID uuid.UUID) error
	// UpdateTenantSchema(ctx context.Context, tenantID uuid.UUID, version string) error
	// GetTenantSchemaVersion(ctx context.Context, tenantID uuid.UUID) (string, error)

	// // Migration and maintenance
	// RunTenantMigration(ctx context.Context, tenantID uuid.UUID) error
	// BackupTenantData(ctx context.Context, tenantID uuid.UUID) (string, error)
	// RestoreTenantData(ctx context.Context, tenantID uuid.UUID, backupPath string) error
}
