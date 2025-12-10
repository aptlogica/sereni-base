package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
)

type RoleService interface {
	CreateRole(ctx context.Context, schemaName string, req dto.RoleInsertion) (master.Role, error)
	GetRoleByName(ctx context.Context, schemaName string, name string) (master.Role, error)
}
