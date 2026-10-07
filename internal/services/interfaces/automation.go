// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

type AutomationService interface {
	Create(ctx context.Context, schemaName string, req dto.CreateAutomationRequest) (tenant.Automation, error)
	// Update saves the title and query; Postgres is not changed until the entry is run
	Update(ctx context.Context, schemaName, id string, req dto.UpdateAutomationRequest) (tenant.Automation, error)
	// Run applies the saved query in Postgres, replacing the trigger or function installed for the entry
	Run(ctx context.Context, schemaName, id string) (tenant.Automation, error)
	// List returns the automations of one table, newest first
	List(ctx context.Context, schemaName, modelID string) ([]tenant.Automation, error)
	Delete(ctx context.Context, schemaName, id string) error
}
