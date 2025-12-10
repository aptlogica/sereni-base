package helpers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("072887741d659e1f4bacade5c5947226")

func GenerateCustomJWT(attributes map[string]interface{}, subject string, expiresAfter int64) (string, error) {
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
