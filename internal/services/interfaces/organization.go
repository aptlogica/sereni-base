package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type OrganizationService interface {
	// CRUD operations
	CreateOrganization(ctx context.Context, schemaName string, req dto.CreateOrganizationRequest) (tenant.Organization, error)
	GetOrganization(ctx context.Context, schemaName string) (tenant.Organization, error)
	UpdateOrganization(ctx context.Context, schemaName string, id string, req dto.UpdateOrganizationRequest) (tenant.Organization, error)
	DeleteOrganization(ctx context.Context, schemaName string, id string) error
	GetOrganizationByID(ctx context.Context, schemaName string, id string) (tenant.Organization, error)
	GetOrganizationByEmail(ctx context.Context, schemaName string, email string) (tenant.Organization, error)
}
