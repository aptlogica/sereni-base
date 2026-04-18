// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package auth

import (
	"context"
)

type Tokens struct {
	AccessToken  string `json:"access_token" mapstructure:"access_token"`
	RefreshToken string `json:"refresh_token" mapstructure:"refresh_token"`
}

type Claims struct {
	UserId string `json:"user_id"`
	Roles  string `json:"roles"`
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
	RefreshToken(ctx context.Context, reqBody AuthServiceRefreshRequest) (Tokens, error)
	ValidateToken(ctx context.Context, tokenStr string) (Claims, error)
	Login(ctx context.Context, reqBody AuthServiceLoginRequest) (Tokens, error)
}
