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

type ViewService interface {

	// Insert a new view record
	Create(ctx context.Context, req dto.ViewInsertion, schemaName string) (tenant.View, error)

	// Fetch a single view by ID
	GetViewByID(ctx context.Context, schemaName, id string) (tenant.View, error)

	// Fetch all views for a schema
	GetAllViews(ctx context.Context, schemaName string) ([]tenant.View, error)

	GetViewsByModelID(ctx context.Context, schemaName string, modelID string) ([]tenant.View, error)

	// Update a view by ID
	UpdateView(ctx context.Context, schemaName, id string, req dto.ViewUpdate) (tenant.View, error)

	// Delete a view by ID
	DeleteView(ctx context.Context, schemaName, id string) error
}
