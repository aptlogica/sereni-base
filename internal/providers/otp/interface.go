// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package otp

import "time"

// ServiceInterface defines the contract for OTP service
type OtpService interface {
	StartCleanup(interval time.Duration)
	StopCleanup()
	Generate(identifier string) string
	Verify(identifier, input string) bool
}
