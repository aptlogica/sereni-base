// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
)

type BaseAPIService interface {
	// Get returns the base's API status; a base that never turned the API on is returned as disabled
	Get(ctx context.Context, schemaName, baseID string) (dto.BaseAPIResponse, error)
	// Enable gives the base its permanent public ID if it has none and turns the API on.
	// Enabling an enabled base changes nothing.
	Enable(ctx context.Context, schemaName, baseID string) (dto.BaseAPIResponse, error)
}
