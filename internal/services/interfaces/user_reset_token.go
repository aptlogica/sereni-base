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

type UserResetTokenService interface {
	CreateUserResetToken(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error)
	GetUserResetToken(ctx context.Context, token string) (tenant.UserResetToken, error)
	DeleteTokensByUserId(ctx context.Context, userId string) error
}
