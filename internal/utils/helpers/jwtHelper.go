// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package helpers

import (
	"time"

	appConfig "github.com/aptlogica/sereni-base/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateCustomJWT(attributes map[string]interface{}, subject string, expiresAfter int64) (string, error) {
	secret := []byte(appConfig.AppConfig.Auth.JWT.Secret)

	claims := jwt.MapClaims{
		"sub": subject,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(expiresAfter) * time.Second).Unix(),
	}

	// Insert other attributes (except sub/iat/exp which are set explicitly)
	for k, v := range attributes {
		// Protect against overwriting reserved claims
		if k == "sub" || k == "iat" || k == "exp" {
			continue
		}
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// Create and return values as per instruction
func DecodeJWT(tokenString string) (jwt.MapClaims, error) {
	secret := []byte(appConfig.AppConfig.Auth.JWT.Secret)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
