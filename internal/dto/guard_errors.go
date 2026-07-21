// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"fmt"
	"net/http"
)

// GuardError represents an authorization guard failure
// Used by permission, role, and attribute guards
type GuardError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"-"` // Not serialized
	Details    map[string]interface{} `json:"details,omitempty"`
}

// Error implements error interface
func (e *GuardError) Error() string {
	return e.Message
}

// PermissionDeniedError creates a permission check failure error
// Used when user lacks required resource+action permission
// NOTE: Permission/Role/Scope helper constructors were removed
// because the current RBAC middleware uses centralized
// response helpers. Keep InvalidContextError and GuardError type.

// InvalidContextError indicates context is missing required values
func InvalidContextError(missingKey string) *GuardError {
	return &GuardError{
		Code:       "INVALID_CONTEXT",
		Message:    fmt.Sprintf("Missing required context: %s", missingKey),
		StatusCode: http.StatusInternalServerError,
		Details: map[string]interface{}{
			"missing_key": missingKey,
			"type":        "context",
		},
	}
}
