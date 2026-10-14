// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
)

type TableConstraintService interface {
	// List returns the constraints of one type on a table, newest first, with their columns resolved
	List(ctx context.Context, schemaName, modelID, constraintType string) ([]dto.TableConstraintResponse, error)
	// Check looks for records that already repeat values in the columns; nothing is changed
	Check(ctx context.Context, schemaName string, req dto.CheckTableConstraintRequest) (dto.ConstraintCheckResult, error)
	// Create adds the constraint in Postgres and stores it
	Create(ctx context.Context, schemaName string, req dto.CreateTableConstraintRequest) (dto.TableConstraintResponse, error)
	// Update changes the title and message, and replaces the Postgres constraint when its columns change
	Update(ctx context.Context, schemaName, id string, req dto.UpdateTableConstraintRequest) (dto.TableConstraintResponse, error)
	// Delete drops the constraint in Postgres and removes it
	Delete(ctx context.Context, schemaName, id string) error
}
