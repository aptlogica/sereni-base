// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
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
