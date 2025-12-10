package auth

import (
	"context"
	"serenibase/internal/models/master"
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
	GenerateToken(ctx context.Context, user master.User) (Tokens, error)
	RefreshToken(ctx context.Context, token string) (Tokens, error)
	ValidateToken(ctx context.Context, authHeader string) (Claims, error)
	Ping(ctx context.Context) (interface{}, error)

	AddUser(ctx context.Context, user master.User, tenant_id string, roles string) (Tokens, error)
	ResetPassword(ctx context.Context, email string, newPassword string) error
	HandleCallback(ctx context.Context, code string) (*AuthResult, error)
	AddOrUpdateUserAttributesToKeycloakUser(ctx context.Context, keycloakUserID string, attributes map[string]interface{}) error
	SetEmailVerified(ctx context.Context, keycloakUserID string) error
	CheckUserExistsByEmailAndReturnUser(ctx context.Context, email string) (exists bool, keycloakUserID string, attributes map[string]string, err error)

	GetProviderURL(provider string) string
	Logout(ctx context.Context, refreshToken string) error
	DisableUser(ctx context.Context, keycloakUserID string) error
	EnableUser(ctx context.Context, keycloakUserID string) error
	DeleteUser(ctx context.Context, keycloakUserID string) error
}
