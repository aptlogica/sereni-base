// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"errors"
	"net/http"

	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/gin-gonic/gin"
)

// ErrorCode is a public API error code (section 4 of docs/API_PLATFORM_FOUNDATION.md)
type ErrorCode string

const (
	CodeBadRequest       ErrorCode = "bad_request"
	CodeNotFound         ErrorCode = "not_found"
	CodeAmbiguousName    ErrorCode = "ambiguous_name"
	CodeConflict         ErrorCode = "conflict"
	CodePayloadTooLarge  ErrorCode = "payload_too_large"
	CodeValidationFailed ErrorCode = "validation_failed"
	CodeRateLimited      ErrorCode = "rate_limited"
	CodeInternalError    ErrorCode = "internal_error"
	CodeQueryTimeout     ErrorCode = "query_timeout"
)

var errorStatus = map[ErrorCode]int{
	CodeBadRequest:       http.StatusBadRequest,
	CodeNotFound:         http.StatusNotFound,
	CodeAmbiguousName:    http.StatusConflict,
	CodeConflict:         http.StatusConflict,
	CodePayloadTooLarge:  http.StatusRequestEntityTooLarge,
	CodeValidationFailed: http.StatusUnprocessableEntity,
	CodeRateLimited:      http.StatusTooManyRequests,
	CodeInternalError:    http.StatusInternalServerError,
	CodeQueryTimeout:     http.StatusServiceUnavailable,
}

// Status returns the HTTP status of an error code
func (c ErrorCode) Status() int {
	if status, ok := errorStatus[c]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// ErrorDetail is one entry of error.details. Index is the position in a bulk request's array.
type ErrorDetail struct {
	Index *int   `json:"index,omitempty"`
	Field string `json:"field,omitempty"`
	ID    string `json:"id,omitempty"`
	Issue string `json:"issue,omitempty"`
}

// Error is an error the public API returns to the client as is
type Error struct {
	Code    ErrorCode
	Message string
	Details []ErrorDetail
}

func (e *Error) Error() string { return string(e.Code) + ": " + e.Message }

// NewError returns a public API error
func NewError(code ErrorCode, message string, details ...ErrorDetail) *Error {
	return &Error{Code: code, Message: message, Details: details}
}

func notFound(message string) *Error   { return NewError(CodeNotFound, message) }
func badRequest(message string) *Error { return NewError(CodeBadRequest, message) }

// at returns a detail for the record at index i of a bulk request
func at(i int, field, issue string) ErrorDetail {
	return ErrorDetail{Index: &i, Field: field, Issue: issue}
}

type errorBody struct {
	Code    ErrorCode     `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type meta struct {
	RequestID string `json:"request_id"`
}

type listMeta struct {
	RequestID  string  `json:"request_id"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
	Limit      int     `json:"limit"`
}

type envelope struct {
	Data  any        `json:"data"`
	Error *errorBody `json:"error"`
	Meta  any        `json:"meta"`
}

// requestID returns the ID the global RequestID middleware gave this request
func requestID(c *gin.Context) string {
	return c.GetString(middleware.RequestIDKey)
}

// respond writes data in the {data, error, meta} envelope
func respond(c *gin.Context, status int, data any) {
	c.JSON(status, envelope{Data: data, Meta: meta{RequestID: requestID(c)}})
}

// respondList writes a page of a list; next_cursor is null on the last page
func respondList(c *gin.Context, data any, nextCursor string, limit int) {
	m := listMeta{RequestID: requestID(c), Limit: limit}
	if nextCursor != "" {
		m.NextCursor = &nextCursor
		m.HasMore = true
	}
	c.JSON(http.StatusOK, envelope{Data: data, Meta: m})
}

// respondError writes err in the envelope. Errors that are not *Error are logged and returned as
// internal_error, so database messages never reach the client.
func respondError(c *gin.Context, err error) {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Str("request_id", requestID(c)).Msg("public API request failed")
		apiErr = NewError(CodeInternalError, "Something went wrong on our side. Please try again.")
	}
	c.AbortWithStatusJSON(apiErr.Code.Status(), envelope{
		Error: &errorBody{Code: apiErr.Code, Message: apiErr.Message, Details: apiErr.Details},
		Meta:  meta{RequestID: requestID(c)},
	})
}
