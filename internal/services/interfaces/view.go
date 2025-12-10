package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
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
