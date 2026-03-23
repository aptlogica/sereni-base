package providers_test

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/providers/otp"
	"github.com/stretchr/testify/assert"
)

// TestOTPService_StartCleanup tests the cleanup functionality
func TestOTPService_StartCleanup(t *testing.T) {
	t.Run("starts and stops cleanup without error", func(t *testing.T) {
		service := otp.NewService(100 * time.Millisecond)

		// Start cleanup with short interval
		service.StartCleanup(50 * time.Millisecond)

		// Generate some OTPs
		service.Generate("user1@example.com")
		service.Generate("user2@example.com")

		// Wait for cleanup to run at least once
		time.Sleep(200 * time.Millisecond)

		// Stop cleanup gracefully
		service.StopCleanup()
	})

	t.Run("cleanup removes expired OTPs", func(t *testing.T) {
		// Create service with very short expiry
		service := otp.NewService(50 * time.Millisecond)

		// Start cleanup
		service.StartCleanup(30 * time.Millisecond)

		// Generate OTP
		identifier := "cleanup-test@example.com"
		generatedOTP := service.Generate(identifier)

		// Verify immediately - should work
		assert.True(t, service.Verify(identifier, generatedOTP))

		// Wait for expiry and cleanup
		time.Sleep(100 * time.Millisecond)

		// Should be expired now
		assert.False(t, service.Verify(identifier, generatedOTP))

		service.StopCleanup()
	})
}

// TestOTPService_ConcurrentAccess tests thread safety
func TestOTPService_ConcurrentAccess(t *testing.T) {
	service := otp.NewService(5 * time.Minute)

	done := make(chan bool)

	// Concurrent generates
	for i := 0; i < 10; i++ {
		go func(id int) {
			identifier := "user" + string(rune('A'+id)) + "@example.com"
			otp := service.Generate(identifier)
			assert.NotEmpty(t, otp)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestOTPService_ConcurrentVerify tests concurrent verify operations
func TestOTPService_ConcurrentVerify(t *testing.T) {
	service := otp.NewService(5 * time.Minute)

	// Generate OTPs first
	identifiers := []string{"user1@example.com", "user2@example.com", "user3@example.com"}
	otps := make(map[string]string)
	for _, id := range identifiers {
		otps[id] = service.Generate(id)
	}

	done := make(chan bool)

	// Concurrent verifies
	for _, id := range identifiers {
		go func(identifier, expectedOTP string) {
			result := service.Verify(identifier, expectedOTP)
			assert.True(t, result)
			done <- true
		}(id, otps[id])
	}

	// Wait for all goroutines
	for range identifiers {
		<-done
	}
}

// TestOTPService_OverwriteOTP tests OTP overwrite behavior
func TestOTPService_OverwriteOTP(t *testing.T) {
	service := otp.NewService(5 * time.Minute)
	identifier := "overwrite@example.com"

	// Generate first OTP
	otp1 := service.Generate(identifier)
	assert.NotEmpty(t, otp1)

	// Generate second OTP - should overwrite
	otp2 := service.Generate(identifier)
	assert.NotEmpty(t, otp2)

	// First OTP should no longer work
	result := service.Verify(identifier, otp1)
	// May or may not work depending on if they're the same
	if otp1 != otp2 {
		assert.False(t, result, "Old OTP should be invalid after regeneration")
	}

	// Second OTP should work
	result = service.Verify(identifier, otp2)
	assert.True(t, result)
}

// TestOTPService_EmptyIdentifier tests empty identifier handling
func TestOTPService_EmptyIdentifier(t *testing.T) {
	service := otp.NewService(5 * time.Minute)

	// Generate with empty identifier
	otp := service.Generate("")
	assert.NotEmpty(t, otp)

	// Verify with empty identifier
	result := service.Verify("", otp)
	assert.True(t, result)
}

// TestOTPService_SpecialCharacters tests identifiers with special characters
func TestOTPService_SpecialCharacters(t *testing.T) {
	service := otp.NewService(5 * time.Minute)

	tests := []struct {
		name       string
		identifier string
	}{
		{"email with plus", "user+tag@example.com"},
		{"email with dots", "user.name@example.com"},
		{"phone number", "+1-555-123-4567"},
		{"unicode", "用户@example.com"},
		{"long identifier", "a" + string(make([]byte, 100)) + "@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp := service.Generate(tt.identifier)
			assert.NotEmpty(t, otp)
			assert.Len(t, otp, 4)

			result := service.Verify(tt.identifier, otp)
			assert.True(t, result)
		})
	}
}

// TestOTPService_MultipleVerifyAttempts tests multiple verification attempts
func TestOTPService_MultipleVerifyAttempts(t *testing.T) {
	service := otp.NewService(5 * time.Minute)
	identifier := "multi-verify@example.com"

	otp := service.Generate(identifier)

	// Multiple successful verifications (OTP not consumed)
	for i := 0; i < 5; i++ {
		result := service.Verify(identifier, otp)
		assert.True(t, result, "Verification %d should succeed", i+1)
	}
}

// TestOTPService_ExpiryBoundary tests OTP expiry at boundary
func TestOTPService_ExpiryBoundary(t *testing.T) {
	// Create service with 100ms expiry
	service := otp.NewService(100 * time.Millisecond)

	identifier := "boundary@example.com"
	otp := service.Generate(identifier)

	// Should work immediately
	assert.True(t, service.Verify(identifier, otp))

	// Wait until just before expiry
	time.Sleep(80 * time.Millisecond)
	assert.True(t, service.Verify(identifier, otp), "Should still work before expiry")

	// Wait past expiry
	time.Sleep(50 * time.Millisecond)
	assert.False(t, service.Verify(identifier, otp), "Should fail after expiry")
}

// TestOTPService_LongExpiry tests with long expiry duration
func TestOTPService_LongExpiry(t *testing.T) {
	// Create service with 24 hour expiry
	service := otp.NewService(24 * time.Hour)

	identifier := "long-expiry@example.com"
	otp := service.Generate(identifier)

	// Should work
	assert.True(t, service.Verify(identifier, otp))
}

// TestOTPService_ZeroExpiry tests with zero expiry duration
func TestOTPService_ZeroExpiry(t *testing.T) {
	service := otp.NewService(0)

	identifier := "zero-expiry@example.com"
	otp := service.Generate(identifier)

	// With zero expiry, OTP should be immediately expired
	// Small sleep to ensure time comparison works
	time.Sleep(time.Millisecond)
	result := service.Verify(identifier, otp)
	assert.False(t, result, "OTP with zero expiry should fail")
}

// TestOTPService_NegativeExpiry tests with negative expiry duration
func TestOTPService_NegativeExpiry(t *testing.T) {
	// Negative expiry should make all OTPs expire immediately
	service := otp.NewService(-1 * time.Hour)

	identifier := "negative-expiry@example.com"
	otp := service.Generate(identifier)

	result := service.Verify(identifier, otp)
	assert.False(t, result, "OTP with negative expiry should fail")
}
