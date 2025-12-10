package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"

	"github.com/google/uuid"
)

type TenantService interface {
	CreateTenant(ctx context.Context, req dto.TenantRequest) (master.Tenant, error)
	GetTenant(ctx context.Context, id uuid.UUID) (master.Tenant, error)
	GetTenantBySchema(ctx context.Context, schema string) (master.Tenant, error)
	SchemaExists(ctx context.Context, schema string) (bool, error)
	Update(ctx context.Context, tenantID string, updateData map[string]interface{}) (master.Tenant, error)
}
