// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package response

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	appErrors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	responseConstants "github.com/aptlogica/sereni-base/internal/utils/response/constants"
)

// StandardResponse represents a standard API response format
type StandardResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Meta      *MetaInfo   `json:"meta,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// CreatedResponse sends a created response
type MetaInfo struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"http_status"`
}

func CreatedResponse(c *gin.Context, data interface{}, message ...string) {
	response := StandardResponse{
		Success:   true,
		Data:      data,
		Message:   defaultMessage(message, "Resource created successfully"),
		RequestID: requestIDFromContext(c),
	}

	c.JSON(http.StatusCreated, response)
}

func defaultMessage(message []string, fallback string) string {
	if len(message) > 0 {
		return message[0]
	}
	return fallback
}

func buildSuccessMeta(code responseConstants.ResponseCode) (MetaInfo, bool) {
	meta, ok := responseConstants.SuccessCodes[code]
	if !ok {
		return MetaInfo{
			Code:       string(code),
			HTTPStatus: http.StatusOK,
		}, false
	}
	return MetaInfo{
		Code:       string(code),
		HTTPStatus: meta.HTTPStatus,
	}, true
}

func buildErrorMeta(code responseConstants.ResponseCode) (MetaInfo, bool) {
	meta, ok := responseConstants.ErrorCodes[code]
	if !ok {
		return MetaInfo{
			Code:       string(responseConstants.Error.InternalError),
			HTTPStatus: http.StatusInternalServerError,
		}, false
	}
	return MetaInfo{
		Code:       string(code),
		HTTPStatus: meta.HTTPStatus,
	}, true
}

func SendSuccess(ctx *gin.Context, code responseConstants.ResponseCode, data interface{}) {
	meta, ok := buildSuccessMeta(code)
	message := responseConstants.SuccessCodes[code].Message
	if !ok {
		// keep predictable output even if code is not registered
		message = "success"
	}
	response := StandardResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Meta:      &meta,
		RequestID: requestIDFromContext(ctx),
	}
	ctx.JSON(meta.HTTPStatus, response)
}

func SendError(ctx *gin.Context, code responseConstants.ResponseCode) {
	meta, ok := buildErrorMeta(code)
	message := responseConstants.ErrorCodes[code].Message
	if !ok {
		message = "Internal server error"
		code = responseConstants.Error.InternalError
	}
	response := StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    string(code),
			Message: message,
		},
		Meta:      &meta,
		RequestID: requestIDFromContext(ctx),
	}
	ctx.JSON(meta.HTTPStatus, response)
}

func SendErrorWithMessage(ctx *gin.Context, code responseConstants.ResponseCode, message string) {
	meta, ok := buildErrorMeta(code)
	if !ok {
		code = responseConstants.Error.InternalError
	}
	response := StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    string(code),
			Message: message,
		},
		Meta:      &meta,
		RequestID: requestIDFromContext(ctx),
	}
	ctx.JSON(meta.HTTPStatus, response)
}

func CheckAndSendError(ctx *gin.Context, err error) {
	// Log the original error
	logger.Get().Error().Err(err).Msg("API Error")

	// Validation errors: return validation code and aggregated details
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		details := strings.Join(FormatValidationError(err), "; ")
		code := responseConstants.Error.ValidationFailed
		meta, _ := buildErrorMeta(code)
		ctx.JSON(meta.HTTPStatus, StandardResponse{
			Success: false,
			Error: &ErrorInfo{
				Code:    string(code),
				Message: responseConstants.ErrorCodes[code].Message,
				Details: details,
			},
			Meta:      &meta,
			RequestID: requestIDFromContext(ctx),
		})
		return
	}

	// Check if it's an APIError (from external API)
	var apiErr *appErrors.APIError
	if errors.As(err, &apiErr) {
		// Send the error message from the API directly to the frontend
		status := apiErr.StatusCode
		if status == 0 {
			status = http.StatusBadRequest
		}
		meta := MetaInfo{Code: apiErr.Code, HTTPStatus: status}
		ctx.JSON(status, StandardResponse{
			Success: false,
			Error: &ErrorInfo{
				Code:    apiErr.Code,
				Message: apiErr.Message,
				Details: fmt.Sprintf("%v", apiErr.Details),
			},
			Meta:      &meta,
			RequestID: requestIDFromContext(ctx),
		})
		return
	}

	code := responseConstants.MapError(err)
	if code == "" {
		// No mapped code: surface the original error message instead of a generic internal error
		code = responseConstants.Error.InternalError
		SendErrorWithMessage(ctx, code, err.Error())
		return
	}
	SendError(ctx, code)
}

func FormatValidationError(err error) []string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		var out []string
		for _, fe := range ve {
			msg := fmt.Sprintf("Field '%s' is %s", fe.Field(), fe.Tag())
			out = append(out, msg)
		}
		return out
	}
	// fallback for non-validation errors
	return []string{err.Error()}
}

func requestIDFromContext(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	requestIDVal, ok := ctx.Get("request_id")
	if !ok {
		return ""
	}
	requestID, _ := requestIDVal.(string)
	return requestID
}

type SuccessMetaInfoSwager struct {
	Code       string `json:"code" example:"USER_CREATED"`
	HTTPStatus int    `json:"http_status" example:"201"`
}

type ErrorMetaInfoSwager struct {
	Code       string `json:"code" example:"USER_VAL_2001"`
	HTTPStatus int    `json:"http_status" example:"400"`
}

type SuccessResponseSwager struct {
	Success bool                   `json:"success" example:"true"`
	Message string                 `json:"message,omitempty" example:"User created successfully"`
	Data    interface{}            `json:"data,omitempty"`
	Meta    *SuccessMetaInfoSwager `json:"meta,omitempty"`
}

type ErrorResponseSwager struct {
	Success bool                 `json:"success" example:"false"`
	Message string               `json:"message,omitempty" example:"Invalid input"`
	Error   *ErrorInfoSwager     `json:"error,omitempty"`
	Meta    *ErrorMetaInfoSwager `json:"meta,omitempty"`
}

type ErrorInfoSwager struct {
	Code    string `json:"code" example:"USER_VAL_2001"`
	Message string `json:"message" example:"Email is required"`
}
