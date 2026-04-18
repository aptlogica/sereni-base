// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/go-playground/validator/v10"
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
