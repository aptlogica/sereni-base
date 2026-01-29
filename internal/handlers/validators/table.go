package validators

import (
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/go-playground/validator"
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

func getAddColumnValidationError(field, tag string) responseConst.ResponseCode {
	switch field {
	case "ModelID":
		if tag == "required" {
			return responseConst.TableError.ModelIDRequired
		}
		return responseConst.TableError.ModelIDInvalid
	case "BaseID":
		if tag == "required" {
			return responseConst.TableError.BaseIDRequired
		}
		return responseConst.TableError.BaseIDInvalid
	case "Title":
		if tag == "required" {
			return responseConst.TableError.TitleRequired
		}
		return responseConst.TableError.TitleInvalid
	case "Description":
		if tag == "required" {
			return responseConst.TableError.DescriptionRequired
		}
		return responseConst.TableError.DescriptionInvalid
	case "UIDT":
		if tag == "required" {
			return responseConst.TableError.UIDTRequired
		}
		return responseConst.TableError.UIDTInvalid
	case "Meta":
		if tag == "required" {
			return responseConst.TableError.DTRequired
		}
		return responseConst.TableError.DTInvalid
	case "OrderIndex":
		if tag == "required" {
			return responseConst.TableError.OrderIndexRequired
		}
		return responseConst.TableError.OrderIndexInvalid
	case "Virtual":
		if tag == "required" {
			return responseConst.TableError.VirtualRequired
		}
		return responseConst.TableError.VirtualInvalid
	case "System":
		if tag == "required" {
			return responseConst.TableError.SystemRequired
		}
		return responseConst.TableError.SystemInvalid
	default:
		return responseConst.Error.ValidationFailed
	}
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
