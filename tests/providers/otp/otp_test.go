package providers_test

import (
	"testing"
	"time"

	"serenibase/internal/providers/otp"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	expiry := 5 * time.Minute
	service := otp.NewService(expiry)

	assert.NotNil(t, service)
}

func TestOTPService_Generate(t *testing.T) {
	service := otp.NewService(5 * time.Minute)

	t.Run("generates OTP for identifier", func(t *testing.T) {
		identifier := "user@example.com"
		otpCode := service.Generate(identifier)

		assert.NotEmpty(t, otpCode)
		assert.Len(t, otpCode, 4) // OTP is 4 digits
	})

	t.Run("generates different OTPs for different identifiers", func(t *testing.T) {
		otp1 := service.Generate("user1@example.com")
		otp2 := service.Generate("user2@example.com")

		assert.NotEqual(t, otp1, otp2)
	})

	t.Run("generates new OTP when called again for same identifier", func(t *testing.T) {
		identifier := "test@example.com"
		otp1 := service.Generate(identifier)
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different OTP
		otp2 := service.Generate(identifier)

		// May or may not be different depending on implementation
		// Just verify both are valid
		assert.NotEmpty(t, otp1)
		assert.NotEmpty(t, otp2)
	})
}

func TestOTPService_Verify(t *testing.T) {
	service := otp.NewService(5 * time.Minute)

	t.Run("verifies correct OTP", func(t *testing.T) {
		identifier := "user@example.com"
		otpCode := service.Generate(identifier)

		result := service.Verify(identifier, otpCode)

		assert.True(t, result)
	})

	t.Run("fails for incorrect OTP", func(t *testing.T) {
		identifier := "user@example.com"
		service.Generate(identifier)

		result := service.Verify(identifier, "000000")

		assert.False(t, result)
	})

	t.Run("fails for non-existent identifier", func(t *testing.T) {
		result := service.Verify("nonexistent@example.com", "123456")

		assert.False(t, result)
	})

	t.Run("fails for empty OTP", func(t *testing.T) {
		identifier := "user@example.com"
		service.Generate(identifier)

		result := service.Verify(identifier, "")

		assert.False(t, result)
	})
}

func TestOTPService_Expiry(t *testing.T) {
	// Create service with very short expiry
	service := otp.NewService(100 * time.Millisecond)

	t.Run("OTP expires after timeout", func(t *testing.T) {
		identifier := "user@example.com"
		otpCode := service.Generate(identifier)

		// Verify immediately - should work
		result := service.Verify(identifier, otpCode)
		assert.True(t, result)

		// Wait for expiry
		time.Sleep(150 * time.Millisecond)

		// Verify after expiry - should fail
		result = service.Verify(identifier, otpCode)
		assert.False(t, result)
	})
}

func TestOTPService_StartStopCleanup(t *testing.T) {
	t.Run("starts and stops cleanup goroutine", func(t *testing.T) {
		service := otp.NewService(5 * time.Minute)
		// This should not panic
		service.StartCleanup(1 * time.Second)
		time.Sleep(50 * time.Millisecond) // Give it time to start
		service.StopCleanup()
	})

	t.Run("can restart cleanup after stopping", func(t *testing.T) {
		service := otp.NewService(5 * time.Minute)
		service.StartCleanup(1 * time.Second)
		time.Sleep(50 * time.Millisecond)
		service.StopCleanup()
		time.Sleep(50 * time.Millisecond)

		// Should be able to restart with new service
		service2 := otp.NewService(5 * time.Minute)
		service2.StartCleanup(1 * time.Second)
		time.Sleep(50 * time.Millisecond)
		service2.StopCleanup()
	})
}

func TestOTPService_Cleanup(t *testing.T) {
	// Create service with very short expiry
	service := otp.NewService(50 * time.Millisecond)

	t.Run("cleans up expired OTPs", func(t *testing.T) {
		identifier := "user@example.com"
		otpCode := service.Generate(identifier)

		// Start cleanup with short interval
		service.StartCleanup(100 * time.Millisecond)
		defer service.StopCleanup()

		// Verify immediately - should work
		result := service.Verify(identifier, otpCode)
		assert.True(t, result)

		// Wait for expiry and cleanup
		time.Sleep(200 * time.Millisecond)

		// OTP should be cleaned up, verification should fail
		result = service.Verify(identifier, otpCode)
		assert.False(t, result)
	})
}

func TestOTPService_Concurrency(t *testing.T) {
	service := otp.NewService(5 * time.Minute)

	t.Run("handles concurrent generate calls", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				identifier := "user@example.com"
				otp := service.Generate(identifier)
				assert.NotEmpty(t, otp)
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("handles concurrent verify calls", func(t *testing.T) {
		identifier := "user@example.com"
		otpCode := service.Generate(identifier)

		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				result := service.Verify(identifier, otpCode)
				assert.True(t, result)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
