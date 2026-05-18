// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/go-playground/validator/v10"
)

func CreateTableValidationErrors(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "BaseID":
		switch tag {
		case "required":
			return responseConst.TableError.BaseIDRequired
		default:
			return responseConst.TableError.BaseIDInvalid
		}
	case "WorkspaceID":
		switch tag {
		case "required":
			return responseConst.TableError.WorkspaceIDRequired
		default:
			return responseConst.TableError.WorkspaceIDInvalid
		}
	case "Title":
		switch tag {
		case "required":
			return responseConst.TableError.TitleRequired
		default:
			return responseConst.TableError.TitleInvalid
		}
	case "Description":
		switch tag {
		case "required":
			return responseConst.TableError.DescriptionRequired
		default:
			return responseConst.TableError.DescriptionInvalid
		}
	case "OrderIndex":
		switch tag {
		case "required":
			return responseConst.TableError.OrderIndexRequired
		default:
			return responseConst.TableError.OrderIndexInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func AddColumnValidator(e validator.FieldError) responseConst.ResponseCode {
	return getAddColumnValidationError(e.Field(), e.Tag())
}

// fieldErrorCodes holds the required and invalid error codes for each field
type fieldErrorCodes struct {
	required responseConst.ResponseCode
	invalid  responseConst.ResponseCode
}

func getAddColumnValidationError(field, tag string) responseConst.ResponseCode {
	fieldErrors := map[string]fieldErrorCodes{
		"ModelID": {
			required: responseConst.TableError.ModelIDRequired,
			invalid:  responseConst.TableError.ModelIDInvalid,
		},
		"BaseID": {
			required: responseConst.TableError.BaseIDRequired,
			invalid:  responseConst.TableError.BaseIDInvalid,
		},
		"Title": {
			required: responseConst.TableError.TitleRequired,
			invalid:  responseConst.TableError.TitleInvalid,
		},
		"Description": {
			required: responseConst.TableError.DescriptionRequired,
			invalid:  responseConst.TableError.DescriptionInvalid,
		},
		"UIDT": {
			required: responseConst.TableError.UIDTRequired,
			invalid:  responseConst.TableError.UIDTInvalid,
		},
		"Meta": {
			required: responseConst.TableError.DTRequired,
			invalid:  responseConst.TableError.DTInvalid,
		},
		"OrderIndex": {
			required: responseConst.TableError.OrderIndexRequired,
			invalid:  responseConst.TableError.OrderIndexInvalid,
		},
		"Virtual": {
			required: responseConst.TableError.VirtualRequired,
			invalid:  responseConst.TableError.VirtualInvalid,
		},
		"System": {
			required: responseConst.TableError.SystemRequired,
			invalid:  responseConst.TableError.SystemInvalid,
		},
	}

	if errors, exists := fieldErrors[field]; exists {
		if tag == "required" {
			return errors.required
		}
		return errors.invalid
	}

	return responseConst.Error.ValidationFailed
}

func CreateViewValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "Meta":
		switch tag {
		case "required":
			return responseConst.TableError.MetaRequired
		default:
			return responseConst.TableError.MetaInvalid
		}
	case "BaseID":
		switch tag {
		case "required":
			return responseConst.TableError.BaseIDRequired
		default:
			return responseConst.TableError.BaseIDInvalid
		}
	case "Title":
		switch tag {
		case "required":
			return responseConst.TableError.TitleRequired
		default:
			return responseConst.TableError.TitleInvalid
		}
	case "Description":
		switch tag {
		case "required":
			return responseConst.TableError.DescriptionRequired
		default:
			return responseConst.TableError.DescriptionInvalid
		}
	case "Type":
		switch tag {
		case "required":
			return responseConst.TableError.TypeRequired
		default:
			return responseConst.TableError.TypeInvalid
		}
	case "OrderIndex":
		switch tag {
		case "required":
			return responseConst.TableError.OrderIndexRequired
		default:
			return responseConst.TableError.OrderIndexInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func CreateRowRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func InsertRowDataRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "ColumnId":
		switch tag {
		case "required":
			return responseConst.TableError.ColumnIdRequired
		default:
			return responseConst.TableError.ColumnIdInvalid
		}
	case "RowId":
		switch tag {
		case "required":
			return responseConst.TableError.RowIdRequired
		default:
			return responseConst.TableError.RowIdInvalid
		}
	case "Value":
		switch tag {
		case "required":
			return responseConst.TableError.ValueRequired
		default:
			return responseConst.TableError.ValueInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func UpdateRowRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "RowId":
		switch tag {
		case "required":
			return responseConst.TableError.RowIdRequired
		default:
			return responseConst.TableError.RowIdInvalid
		}
	case "Values":
		switch tag {
		case "required":
			return responseConst.TableError.ValueRequired
		default:
			return responseConst.TableError.ValueInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func DeleteRowDataRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "RowId":
		switch tag {
		case "required":
			return responseConst.TableError.RowIdRequired
		default:
			return responseConst.TableError.RowIdInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func BulkDeleteRowsRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "RowIds":
		switch tag {
		case "required":
			return responseConst.TableError.RowIdRequired
		case "min":
			return responseConst.TableError.RowIdRequired
		default:
			return responseConst.TableError.RowIdInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func UpdateRowDataLinksRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "ColumnId":
		switch tag {
		case "required":
			return responseConst.TableError.ColumnIdRequired
		default:
			return responseConst.TableError.ColumnIdInvalid
		}
	case "SourceRowId":
		return getRowIdValidationError(tag)
	case "TargetRowId":
		return getRowIdValidationError(tag)
	case "Action":
		switch tag {
		case "required":
			return responseConst.TableError.ActionRequired
		case "oneof":
			return responseConst.TableError.ActionInvalid
		default:
			return responseConst.TableError.ActionInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func getRowIdValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.RowIdRequired
	default:
		return responseConst.TableError.RowIdInvalid
	}
}

func RemoveAttachmentsRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "ColumnId":
		switch tag {
		case "required":
			return responseConst.TableError.ColumnIdRequired
		default:
			return responseConst.TableError.ColumnIdInvalid
		}
	case "RowId":
		switch tag {
		case "required":
			return responseConst.TableError.RowIdRequired
		default:
			return responseConst.TableError.RowIdInvalid
		}
	case "Attachments":
		switch tag {
		case "required":
			return responseConst.TableError.AttachmentRequired
		default:
			return responseConst.TableError.AttachmentInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func PaginationRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "PageSize":
		switch tag {
		case "required":
			return responseConst.TableError.LimitRequired
		default:
			return responseConst.TableError.LimitInvalid
		}
	case "PageNumber":
		switch tag {
		case "required":
			return responseConst.TableError.PageRequired
		default:
			return responseConst.TableError.PageInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func ReorderColumnRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "SourceColumnID":
		return getColumnIdValidationError(tag)
	case "TargetColumnID":
		return getColumnIdValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func getColumnIdValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.ColumnIdRequired
	default:
		return responseConst.TableError.ColumnIdInvalid
	}
}

func UpdateAttachmentRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "ColumnId":
		switch tag {
		case "required":
			return responseConst.TableError.ColumnIdRequired
		default:
			return responseConst.TableError.ColumnIdInvalid
		}
	case "RowId":
		switch tag {
		case "required":
			return responseConst.TableError.RowIdRequired
		default:
			return responseConst.TableError.RowIdInvalid
		}
	case "AssetId":
		switch tag {
		case "required":
			return responseConst.TableError.AssetIdRequired
		default:
			return responseConst.TableError.AssetIdInvalid
		}
	case "Content":
		switch tag {
		case "required":
			return responseConst.TableError.ContentRequired
		default:
			return responseConst.TableError.ContentInvalid
		}

	default:
		return responseConst.Error.ValidationFailed
	}
}

func BulkUpdateColumnsRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "ColumnId":
		switch tag {
		case "required":
			return responseConst.TableError.ColumnIdRequired
		default:
			return responseConst.TableError.ColumnIdInvalid
		}
	case "Updates":
		switch tag {
		case "required":
			return responseConst.TableError.UpdatesRequired
		case "min":
			return responseConst.TableError.UpdatesRequired
		default:
			return responseConst.TableError.UpdatesInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func ResetColumnValuesRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		switch tag {
		case "required":
			return responseConst.TableError.ModelIDRequired
		default:
			return responseConst.TableError.ModelIDInvalid
		}
	case "ColumnId":
		switch tag {
		case "required":
			return responseConst.TableError.ColumnIdRequired
		default:
			return responseConst.TableError.ColumnIdInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}
