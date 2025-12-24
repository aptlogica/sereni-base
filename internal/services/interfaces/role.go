package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type RoleService interface {
	CreateRole(ctx context.Context, schemaName string, req dto.RoleInsertion) (tenant.Role, error)
	GetRoleByName(ctx context.Context, schemaName string, name string) (tenant.Role, error)
}
