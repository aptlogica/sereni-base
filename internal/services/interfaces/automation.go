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
	// List returns automations; modelID and automationType are optional filters (empty = all)
	List(ctx context.Context, schemaName, modelID, automationType string) ([]tenant.Automation, error)
	Delete(ctx context.Context, schemaName, id string) error
}
