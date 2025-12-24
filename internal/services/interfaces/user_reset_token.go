package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type UserResetTokenService interface {
	CreateUserResetToken(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error)
	GetUserResetToken(ctx context.Context, token string) (tenant.UserResetToken, error)
	DeleteTokensByUserId(ctx context.Context, userId string) error
}
