package auth

import (
	"context"
	"serenibase/internal/models/tenant"
)

type Tokens struct {
	AccessToken  string `json:"access_token" mapstructure:"access_token"`
	RefreshToken string `json:"refresh_token" mapstructure:"refresh_token"`
}

type Claims struct {
	UserId   string `json:"user_id"`
	TenantId string `json:"tenant_id"`
	Roles    string `json:"roles"`
}

type AuthResult struct {
	IdentityProvider string
	KeycloakUserId   string
	AccessToken      string
	RefreshToken     string
	IDToken          string
	FirstName        string
	LastName         string
	Email            string
}

type AuthProvider interface {
	GenerateToken(ctx context.Context, user tenant.User) (Tokens, error)
	RefreshToken(ctx context.Context, token string) (Tokens, error)
	ValidateToken(ctx context.Context, tokenStr string) (Claims, error)
	Login(ctx context.Context, email, password string) (Tokens, error)
	Register(ctx context.Context, email, password string, roles []string) error
}
