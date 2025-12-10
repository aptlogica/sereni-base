package otp

import "time"

// ServiceInterface defines the contract for OTP service
type OtpService interface {
	StartCleanup(interval time.Duration)
	StopCleanup()
	Generate(identifier string) string
	Verify(identifier, input string) bool
}
