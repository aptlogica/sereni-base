// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/go-playground/validator"
)

func BulkInsertValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "IDs":
		switch tag {
		case "required":
			return responseConst.AssetError.IdsRequired
		default:
			return responseConst.AssetError.IdsInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func UpdateValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Title":
		switch tag {
		case "required":
			return responseConst.AssetError.TitleRequired
		default:
			return responseConst.AssetError.TitleInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}
