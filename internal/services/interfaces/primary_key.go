// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
)

type PrimaryKeyService interface {
	// Get returns the primary key of a table and which fields can be key fields
	Get(ctx context.Context, schemaName, modelID string) (dto.PrimaryKeyResponse, error)
	// Update replaces the primary key after checking that existing records satisfy it
	Update(ctx context.Context, schemaName, modelID string, req dto.UpdatePrimaryKeyRequest) (dto.PrimaryKeyResponse, error)
	// EnsureColumnNotInKey fails when the column is part of its table's primary key
	EnsureColumnNotInKey(ctx context.Context, schemaName, modelID, columnName string) error
}
