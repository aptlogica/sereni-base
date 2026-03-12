// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
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
