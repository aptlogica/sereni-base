package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type UserRoleService interface {
	CreateUserRole(ctx context.Context, schemaName string, req dto.UserRoleInsertion) (tenant.UserRole, error)
}
