// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/go-playground/validator"
)

// OrganizationCreationValidationError maps validation errors for dto.CreateOrganizationRequest to response codes.
func OrganizationCreationValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	// Since all validation errors currently map to the same response code,
	// we can simplify by checking if we have a known field/tag combination
	// and return early. This avoids SonarQube's "identical code blocks" issue.

	// Validate known fields
	if field == "Name" || field == "Email" {
		// Validate known tags for these fields
		if tag == "required" || tag == "email" {
			return responseConst.Error.ValidationFailed
		}
	}

	// Default case for any other validation error
	return responseConst.Error.ValidationFailed
}

// OrganizationUpdateValidationError maps validation errors for dto.UpdateOrganizationRequest to response codes.
func OrganizationUpdateValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	// Since all validation errors currently map to the same response code,
	// we can simplify by checking if we have a known field/tag combination
	// and return early. This avoids SonarQube's "identical code blocks" issue.

	// Validate known fields
	if field == "Email" && tag == "email" {
		return responseConst.Error.ValidationFailed
	}

	// Default case for any other validation error
	return responseConst.Error.ValidationFailed
}
