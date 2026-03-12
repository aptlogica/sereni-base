// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"serenibase/internal/utils/response"
	"serenibase/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// HandlerUtil provides common handler utilities
type HandlerUtil struct{}

// NewHandlerUtil creates a new handler utility
func NewHandlerUtil() *HandlerUtil {
	return &HandlerUtil{}
}

// BindAndValidateJSON binds JSON request body and validates it
// Returns true if binding and validation succeed, false otherwise
func (hu *HandlerUtil) BindAndValidateJSON(c *gin.Context, req interface{}, validationErrorFunc func(validator.FieldError) constants.ResponseCode) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validationErrorFunc(ve[0]))
			return false
		}
		response.CheckAndSendError(c, err)
		return false
	}
	return true
}

// GetSchemaFromContext extracts schema name from context
func (hu *HandlerUtil) GetSchemaFromContext(c *gin.Context) (string, bool) {
	schemaVal, ok := c.Get("schema")
	if !ok {
		return "", false
	}
	schema, _ := schemaVal.(string)
	return schema, schema != ""
}

// GetUserIDFromContext extracts user ID from context
func (hu *HandlerUtil) GetUserIDFromContext(c *gin.Context) (string, bool) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		return "", false
	}
	userID, _ := userIDVal.(string)
	return userID, userID != ""
}

// GetBothFromContext extracts both schema and user_id from context
func (hu *HandlerUtil) GetBothFromContext(c *gin.Context) (schema string, userID string, ok bool) {
	schemaVal, hasSchema := c.Get("schema")
	userIDVal, hasUserID := c.Get("user_id")

	if !hasSchema || !hasUserID {
		return "", "", false
	}

	schema, _ = schemaVal.(string)
	userID, _ = userIDVal.(string)

	return schema, userID, schema != "" && userID != ""
}

// SendSuccessResponse sends a success response
func (hu *HandlerUtil) SendSuccessResponse(c *gin.Context, code constants.ResponseCode, data interface{}) {
	response.SendSuccess(c, code, data)
}

// SendErrorResponse sends an error response
func (hu *HandlerUtil) SendErrorResponse(c *gin.Context, err interface{}) {
	response.CheckAndSendError(c, err.(error))
}

// SendInvalidPayloadError sends invalid payload error
func (hu *HandlerUtil) SendInvalidPayloadError(c *gin.Context) {
	response.SendError(c, constants.Error.InvalidPayload)
}

// SendUnauthorizedError sends unauthorized error
func (hu *HandlerUtil) SendUnauthorizedError(c *gin.Context) {
	response.SendError(c, constants.Error.UnauthorizedAccess)
}
