package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
)

type UserResetTokenService interface {
	CreateUserResetToken(ctx context.Context, req dto.UserResetTokenInsertion) (master.UserResetToken, error)
	GetUserResetToken(ctx context.Context, token string) (master.UserResetToken, error)
	DeleteTokensByUserId(ctx context.Context, userId string) error
}
