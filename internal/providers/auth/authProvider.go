package auth

import (
	"context"
	"fmt"
	"time"

	appErrors "serenibase/internal/app-errors"
	"serenibase/internal/config"
	"serenibase/internal/models/tenant"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func NewAuthProvider(cfg *config.AuthConfig) (AuthProvider, error) {
	return &AuthProviderService{
		AuthConfig: cfg,
	}, nil
}

type AuthProviderService struct {
	AuthConfig *config.AuthConfig
}

// Custom Claims structure
type CustomClaims struct {
	UserId        string `json:"user_id"`
	Email         string `json:"email"`
	Roles         string `json:"roles,omitempty"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

func (a *AuthProviderService) GenerateToken(ctx context.Context, user tenant.User) (Tokens, error) {
	// Create Access Token
	accessClaims := CustomClaims{
		UserId:        user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Roles:         user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(a.AuthConfig.JWT.AccessTokenExpiry) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    a.AuthConfig.JWT.Issuer,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccessToken, err := accessToken.SignedString([]byte(a.AuthConfig.JWT.Secret))
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Create Refresh Token
	refreshClaims := CustomClaims{
		UserId: user.ID.String(),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(a.AuthConfig.JWT.RefreshTokenExpiry) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    a.AuthConfig.JWT.Issuer,
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefreshToken, err := refreshToken.SignedString([]byte(a.AuthConfig.JWT.Secret))
	if err != nil {
		return Tokens{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return Tokens{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
	}, nil
}

func (a *AuthProviderService) ValidateToken(ctx context.Context, tokenStr string) (Claims, error) {
	// Remove "Bearer " prefix if present
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.AuthConfig.JWT.Secret), nil
	})

	if err != nil {
		// Differentiate expired vs invalid if needed, but for now simple error
		return Claims{}, appErrors.TokenInvalid
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return Claims{
			UserId: claims.UserId,
			Roles:  claims.Roles,
		}, nil
	}

	return Claims{}, appErrors.TokenInvalid
}

func (a *AuthProviderService) RefreshToken(ctx context.Context, tokenStr string) (Tokens, error) {
	// Validate the refresh token
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.AuthConfig.JWT.Secret), nil
	})

	if err != nil {
		return Tokens{}, appErrors.TokenInvalid
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		user := tenant.User{
			Email:         claims.Email,
			Roles:         claims.Roles,
			EmailVerified: claims.EmailVerified,
		}

		if claims.UserId != "" {
			uid, err := uuid.Parse(claims.UserId)
			if err == nil {
				user.ID = uid
			}
		}

		return a.GenerateToken(ctx, user)
	}

	return Tokens{}, appErrors.TokenInvalid
}
