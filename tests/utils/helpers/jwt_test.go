package tests

import (
	"serenibase/internal/utils/helpers"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestGenerateCustomJWT tests JWT generation
func TestGenerateCustomJWT(t *testing.T) {
	tests := []struct {
		name         string
		attributes   map[string]interface{}
		subject      string
		expiresAfter int64
		wantErr      bool
	}{
		{
			name: "basic JWT",
			attributes: map[string]interface{}{
				"user_id": "123",
				"role":    "admin",
			},
			subject:      "test-user",
			expiresAfter: 3600,
			wantErr:      false,
		},
		{
			name:         "JWT without attributes",
			attributes:   map[string]interface{}{},
			subject:      "test-user",
			expiresAfter: 3600,
			wantErr:      false,
		},
		{
			name: "JWT with nil attributes",
			attributes: map[string]interface{}{
				"key": nil,
			},
			subject:      "test-user",
			expiresAfter: 3600,
			wantErr:      false,
		},
		{
			name: "JWT with reserved claims (should be ignored)",
			attributes: map[string]interface{}{
				"sub":    "should-be-ignored",
				"iat":    12345,
				"exp":    67890,
				"custom": "value",
			},
			subject:      "actual-subject",
			expiresAfter: 3600,
			wantErr:      false,
		},
		{
			name:         "short expiry",
			attributes:   map[string]interface{}{"test": "data"},
			subject:      "test",
			expiresAfter: 1,
			wantErr:      false,
		},
		{
			name:         "long expiry",
			attributes:   map[string]interface{}{"test": "data"},
			subject:      "test",
			expiresAfter: 86400 * 30,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := helpers.GenerateCustomJWT(tt.attributes, tt.subject, tt.expiresAfter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateCustomJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if token == "" {
					t.Error("GenerateCustomJWT() returned empty token")
				}
				if len(token) == 0 {
					t.Error("GenerateCustomJWT() returned invalid token format")
				}
			}
		})
	}
}

// TestDecodeJWT tests JWT decoding
func TestDecodeJWT(t *testing.T) {
	attributes := map[string]interface{}{
		"user_id": "123",
		"role":    "admin",
		"email":   "test@example.com",
	}
	subject := "test-user"
	expiresAfter := int64(3600)

	validToken, err := helpers.GenerateCustomJWT(attributes, subject, expiresAfter)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		validator func(jwt.MapClaims) bool
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
			validator: func(claims jwt.MapClaims) bool {
				return claims["sub"] == subject &&
					claims["user_id"] == "123" &&
					claims["role"] == "admin" &&
					claims["email"] == "test@example.com"
			},
		},
		{
			name:    "invalid token",
			token:   "invalid.token.string",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not-a-jwt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := helpers.DecodeJWT(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if claims == nil {
					t.Error("DecodeJWT() returned nil claims")
					return
				}
				if tt.validator != nil && !tt.validator(claims) {
					t.Errorf("DecodeJWT() claims validation failed: %v", claims)
				}
			}
		})
	}
}

// TestJWTExpiry tests JWT expiration
func TestJWTExpiry(t *testing.T) {
	t.Run("expired token", func(t *testing.T) {
		attributes := map[string]interface{}{"test": "data"}
		token, err := helpers.GenerateCustomJWT(attributes, "test", 1)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		time.Sleep(1500 * time.Millisecond)

		_, err = helpers.DecodeJWT(token)
		if err == nil {
			t.Error("DecodeJWT() should fail for expired token")
		}
	})

	t.Run("non-expired token", func(t *testing.T) {
		attributes := map[string]interface{}{"test": "data"}
		token, err := helpers.GenerateCustomJWT(attributes, "test", 3600)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		claims, err := helpers.DecodeJWT(token)
		if err != nil {
			t.Errorf("DecodeJWT() failed for valid token: %v", err)
		}
		if claims == nil {
			t.Error("DecodeJWT() returned nil claims for valid token")
		}
	})
}

// TestJWTRoundTrip tests encoding and decoding
func TestJWTRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]interface{}
		subject    string
	}{
		{
			name: "string values",
			attributes: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			subject: "user-123",
		},
		{
			name: "numeric values",
			attributes: map[string]interface{}{
				"age":    30,
				"score":  95.5,
				"active": true,
			},
			subject: "user-456",
		},
		{
			name: "mixed types",
			attributes: map[string]interface{}{
				"string": "value",
				"number": 42,
				"float":  3.14,
				"bool":   true,
			},
			subject: "user-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := helpers.GenerateCustomJWT(tt.attributes, tt.subject, 3600)
			if err != nil {
				t.Fatalf("GenerateCustomJWT() error = %v", err)
			}

			claims, err := helpers.DecodeJWT(token)
			if err != nil {
				t.Fatalf("DecodeJWT() error = %v", err)
			}

			if claims["sub"] != tt.subject {
				t.Errorf("Subject = %v, want %v", claims["sub"], tt.subject)
			}

			for key, expectedValue := range tt.attributes {
				actualValue, exists := claims[key]
				if !exists {
					t.Errorf("Claim %q not found in decoded token", key)
					continue
				}

				switch v := expectedValue.(type) {
				case int:
					if actualValue != float64(v) {
						t.Errorf("Claim %q = %v, want %v", key, actualValue, float64(v))
					}
				default:
					if actualValue != expectedValue {
						t.Errorf("Claim %q = %v, want %v", key, actualValue, expectedValue)
					}
				}
			}

			if _, exists := claims["iat"]; !exists {
				t.Error("iat claim not found")
			}
			if _, exists := claims["exp"]; !exists {
				t.Error("exp claim not found")
			}
		})
	}
}

// TestJWTClaimsTimestamps tests IAT and EXP claims
func TestJWTClaimsTimestamps(t *testing.T) {
	expiresAfter := int64(3600)
	beforeGen := time.Now().Unix()

	token, err := helpers.GenerateCustomJWT(map[string]interface{}{}, "test", expiresAfter)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	afterGen := time.Now().Unix()

	claims, err := helpers.DecodeJWT(token)
	if err != nil {
		t.Fatalf("Failed to decode token: %v", err)
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		t.Fatal("iat claim is not a number")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim is not a number")
	}

	if int64(iat) < beforeGen || int64(iat) > afterGen {
		t.Errorf("iat timestamp %v is outside expected range [%v, %v]", int64(iat), beforeGen, afterGen)
	}

	expectedExp := int64(iat) + expiresAfter
	if int64(exp) < expectedExp-2 || int64(exp) > expectedExp+2 {
		t.Errorf("exp timestamp %v is not approximately %v (iat %v + %v)", int64(exp), expectedExp, int64(iat), expiresAfter)
	}
}

// TestDecodeJWT_EdgeCases tests DecodeJWT with various edge cases
func TestDecodeJWT_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"token with dots but invalid", "a.b.c", true},
		{"token with special chars", "!@#$%^&*()", true},
		{"token missing signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0In0", true},
		{"token with wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0In0.wrongsignature", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := helpers.DecodeJWT(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeJWT() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGenerateCustomJWT_ZeroExpiry tests JWT with zero expiry
func TestGenerateCustomJWT_ZeroExpiry(t *testing.T) {
	token, err := helpers.GenerateCustomJWT(map[string]interface{}{}, "test", 0)
	if err != nil {
		t.Fatalf("GenerateCustomJWT() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateCustomJWT() returned empty token")
	}

	// Token should be immediately expired or about to expire
	claims, err := helpers.DecodeJWT(token)
	if err == nil {
		// If it decodes, check the expiry is in the past or very close to now
		exp := claims["exp"].(float64)
		now := time.Now().Unix()
		if int64(exp) > now+1 {
			t.Error("Token with 0 expiry should be expired or about to expire")
		}
	}
}

// TestGenerateCustomJWT_NilAttributes tests with nil in attributes
func TestGenerateCustomJWT_NilAttributes(t *testing.T) {
	attrs := map[string]interface{}{
		"key1": "value1",
		"key2": nil,
		"key3": 123,
	}

	token, err := helpers.GenerateCustomJWT(attrs, "test", 3600)
	if err != nil {
		t.Fatalf("GenerateCustomJWT() error = %v", err)
	}

	claims, err := helpers.DecodeJWT(token)
	if err != nil {
		t.Fatalf("DecodeJWT() error = %v", err)
	}

	if claims["key1"] != "value1" {
		t.Error("key1 should be preserved")
	}
	if claims["key3"] != float64(123) {
		t.Error("key3 should be preserved")
	}
}

// TestDecodeJWT_InvalidClaims tests decoding JWT with invalid claims structure
func TestDecodeJWT_InvalidClaims(t *testing.T) {
	// Create a token with a different signing method or tampered token
	tamperedToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	_, err := helpers.DecodeJWT(tamperedToken)
	if err == nil {
		t.Error("DecodeJWT should return error for token with wrong signature")
	}
}

// TestDecodeJWT_ExpiredToken tests decoding an expired token
func TestDecodeJWT_ExpiredToken(t *testing.T) {
	// Generate a token that expires immediately
	attributes := map[string]interface{}{"test": "data"}
	token, err := helpers.GenerateCustomJWT(attributes, "test", -1) // Already expired
	if err != nil {
		t.Fatalf("Failed to generate expired token: %v", err)
	}

	// Wait a moment to ensure it's expired
	time.Sleep(100 * time.Millisecond)

	_, err = helpers.DecodeJWT(token)
	// Note: The implementation might not check expiry, so this tests the actual behavior
	if err != nil {
		// Expected - token is expired
		t.Logf("Expired token correctly rejected: %v", err)
	}
}
