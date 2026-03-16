package tests

import (
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// TestHashPassword tests password hashing
func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "MyPassword123!", false},
		{"empty password", "", false},
		{"short password", "123", false},
		{"long password", "ThisIsAVeryLongPasswordWithManyCharacters12345678901234567890", false},
		{"special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := helpers.HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}
				if hash == tt.password {
					t.Error("HashPassword() returned unhashed password")
				}
			}
		})
	}

	// Test that same password produces different hashes (due to salt)
	t.Run("different hashes for same password", func(t *testing.T) {
		password := "TestPassword123"
		hash1, _ := helpers.HashPassword(password)
		hash2, _ := helpers.HashPassword(password)
		if hash1 == hash2 {
			t.Error("HashPassword() should produce different hashes for same password")
		}
	})

	// Test with extremely long password that causes bcrypt to fail
	t.Run("extremely long password error", func(t *testing.T) {
		// bcrypt has a maximum password length of 72 bytes
		// Create a password longer than the maximum to trigger error
		longPassword := make([]byte, 73)
		for i := range longPassword {
			longPassword[i] = 'a'
		}
		hash, err := helpers.HashPassword(string(longPassword))
		// bcrypt should return an error for passwords > 72 bytes
		if err == nil {
			t.Error("HashPassword() should return error for password > 72 bytes")
		}
		// Hash should be empty on error
		if hash != "" {
			t.Error("HashPassword() should return empty hash on error")
		}
	})
}

// TestCheckPasswordHash tests password verification
func TestCheckPasswordHash(t *testing.T) {
	password := "MySecurePassword123!"
	hash, _ := helpers.HashPassword(password)

	tests := []struct {
		name     string
		password string
		hash     string
		expected bool
	}{
		{"correct password", password, hash, true},
		{"incorrect password", "WrongPassword", hash, false},
		{"empty password", "", hash, false},
		{"invalid hash", password, "invalid_hash", false},
		{"empty hash", password, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.CheckPasswordHash(tt.password, tt.hash)
			if result != tt.expected {
				t.Errorf("CheckPasswordHash() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestHashPasswordAndCheck tests the full cycle
func TestHashPasswordAndCheck(t *testing.T) {
	passwords := []string{
		"Simple123",
		"Complex!@#$%^&*()Password",
		"短密码",
		"VeryLongPasswordWithManyCharactersToTestTheLimit1234567890",
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			hash, err := helpers.HashPassword(password)
			if err != nil {
				t.Fatalf("HashPassword() error = %v", err)
			}

			if !helpers.CheckPasswordHash(password, hash) {
				t.Error("CheckPasswordHash() failed for correct password")
			}

			if helpers.CheckPasswordHash(password+"wrong", hash) {
				t.Error("CheckPasswordHash() succeeded for incorrect password")
			}
		})
	}
}

// TestCheckPasswordHash_InvalidHash tests CheckPasswordHash with invalid hashes
func TestCheckPasswordHash_InvalidHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hash     string
		expected bool
	}{
		{"empty hash", "password", "", false},
		{"invalid format", "password", "not-a-bcrypt-hash", false},
		{"wrong algorithm", "password", "$2y$10$invalid", false},
		{"truncated hash", "password", "$2a$06$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.CheckPasswordHash(tt.password, tt.hash)
			if result != tt.expected {
				t.Errorf("CheckPasswordHash() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAccessTokenClaims tests the AccessTokenClaims structure
func TestAccessTokenClaims(t *testing.T) {
	claims := helpers.AccessTokenClaims{
		UserID: "user123",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "test-subject",
			Issuer:  "test-issuer",
		},
	}

	if claims.UserID != "user123" {
		t.Errorf("AccessTokenClaims.UserID = %q, want %q", claims.UserID, "user123")
	}

	if claims.Subject != "test-subject" {
		t.Errorf("AccessTokenClaims.Subject = %q, want %q", claims.Subject, "test-subject")
	}

	if claims.Issuer != "test-issuer" {
		t.Errorf("AccessTokenClaims.Issuer = %q, want %q", claims.Issuer, "test-issuer")
	}
}
