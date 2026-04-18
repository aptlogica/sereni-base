// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package models

// Meta represents the standard meta object included in every response.
type Meta struct {
	Code       string `json:"code" example:"USER_SUCCESS_2001"`
	HTTPStatus int    `json:"http_status" example:"200"`
}

// ErrorDetail describes the specific error returned to clients.
type ErrorDetail struct {
	Code    string `json:"code" example:"USER_VAL_2001"`
	Message string `json:"message" example:"Email is required"`
}

// ErrorResponse represents a standard API error response.
type ErrorResponse struct {
	Success bool         `json:"success" example:"false"`
	Message string       `json:"message,omitempty" example:"Validation failed"`
	Error   *ErrorDetail `json:"error,omitempty"`
	Meta    *Meta        `json:"meta,omitempty"`
}

// SuccessResponse represents a standard API success response.
type SuccessResponse struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message,omitempty" example:"Operation completed successfully"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}
