// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Reusable validation helper functions

func getModelIDValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.ModelIDRequired
	default:
		return responseConst.TableError.ModelIDInvalid
	}
}

func getBaseIDValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.BaseIDRequired
	default:
		return responseConst.TableError.BaseIDInvalid
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

func getColumnIdValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.ColumnIdRequired
	default:
		return responseConst.TableError.ColumnIdInvalid
	}
}

func getColumnsValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required", "min":
		return responseConst.TableError.ColumnNameRequired
	default:
		return responseConst.TableError.ColumnNameInvalid
	}
}

func getTitleValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.TitleRequired
	default:
		return responseConst.TableError.TitleInvalid
	}
}

func getDescriptionValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.DescriptionRequired
	default:
		return responseConst.TableError.DescriptionInvalid
	}
}

func getOrderIndexValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.OrderIndexRequired
	default:
		return responseConst.TableError.OrderIndexInvalid
	}
}

func getActionValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required", "required_if":
		return responseConst.TableError.ActionRequired
	default:
		return responseConst.TableError.ActionInvalid
	}
}

func getValueValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required", "required_if":
		return responseConst.TableError.ValueRequired
	default:
		return responseConst.TableError.ValueInvalid
	}
}

func getMetaValidationError(tag string) responseConst.ResponseCode {
	switch tag {
	case "required":
		return responseConst.TableError.MetaRequired
	default:
		return responseConst.TableError.MetaInvalid
	}
}

func CreateTableValidationErrors(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "BaseID":
		return getBaseIDValidationError(tag)
	case "WorkspaceID":
		switch tag {
		case "required":
			return responseConst.TableError.WorkspaceIDRequired
		default:
			return responseConst.TableError.WorkspaceIDInvalid
		}
	case "Title":
		return getTitleValidationError(tag)
	case "Description":
		return getDescriptionValidationError(tag)
	case "OrderIndex":
		return getOrderIndexValidationError(tag)
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
		return getModelIDValidationError(tag)
	case "BaseID":
		return getBaseIDValidationError(tag)
	case "Title":
		return getTitleValidationError(tag)
	case "Description":
		return getDescriptionValidationError(tag)
	case "UIDT":
		switch tag {
		case "required":
			return responseConst.TableError.UIDTRequired
		default:
			return responseConst.TableError.UIDTInvalid
		}
	case "Meta":
		switch tag {
		case "required":
			return responseConst.TableError.DTRequired
		default:
			return responseConst.TableError.DTInvalid
		}
	case "OrderIndex":
		return getOrderIndexValidationError(tag)
	case "Virtual":
		switch tag {
		case "required":
			return responseConst.TableError.VirtualRequired
		default:
			return responseConst.TableError.VirtualInvalid
		}
	case "System":
		switch tag {
		case "required":
			return responseConst.TableError.SystemRequired
		default:
			return responseConst.TableError.SystemInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func CreateViewValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "Meta":
		return getMetaValidationError(tag)
	case "BaseID":
		return getBaseIDValidationError(tag)
	case "Title":
		return getTitleValidationError(tag)
	case "Description":
		return getDescriptionValidationError(tag)
	case "Type":
		switch tag {
		case "required":
			return responseConst.TableError.TypeRequired
		default:
			return responseConst.TableError.TypeInvalid
		}
	case "OrderIndex":
		return getOrderIndexValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

// ValidateCreateViewMeta enforces meta requirements for selected view types.
func ValidateCreateViewMeta(req dto.CreateViewRequest) (responseConst.ResponseCode, bool) {
	if req.Meta == nil {
		return responseConst.TableError.MetaRequired, true
	}

	viewType := strings.ToLower(strings.TrimSpace(req.Type))
	type metaFieldRule struct {
		key         string
		requiredErr responseConst.ResponseCode
		invalidErr  responseConst.ResponseCode
	}

	requiredMetaByType := map[string][]metaFieldRule{
		"gallery": {
			{key: "attachment_field_id", requiredErr: responseConst.TableError.ViewAttachmentFieldIDRequired, invalidErr: responseConst.TableError.ViewAttachmentFieldIDInvalid},
		},
		"kanban": {
			{key: "view_target_field", requiredErr: responseConst.TableError.ViewTargetFieldRequired, invalidErr: responseConst.TableError.ViewTargetFieldInvalid},
		},
		"calendar": {
			{key: "date_field_id", requiredErr: responseConst.TableError.ViewDateFieldIDRequired, invalidErr: responseConst.TableError.ViewDateFieldIDInvalid},
		},
		"calender": {
			{key: "date_field_id", requiredErr: responseConst.TableError.ViewDateFieldIDRequired, invalidErr: responseConst.TableError.ViewDateFieldIDInvalid},
		},
		"ganttchart": {
			{key: "start_date_field_id", requiredErr: responseConst.TableError.ViewStartDateFieldIDRequired, invalidErr: responseConst.TableError.ViewStartDateFieldIDInvalid},
			{key: "end_date_field_id", requiredErr: responseConst.TableError.ViewEndDateFieldIDRequired, invalidErr: responseConst.TableError.ViewEndDateFieldIDInvalid},
		},
	}

	requiredMetaKeys, shouldValidate := requiredMetaByType[viewType]
	if !shouldValidate {
		return "", false
	}

	for _, rule := range requiredMetaKeys {
		value, exists := (*req.Meta)[rule.key]
		if !exists {
			return rule.requiredErr, true
		}

		valueStr, ok := value.(string)
		if !ok || strings.TrimSpace(valueStr) == "" {
			return rule.invalidErr, true
		}

		if _, err := uuid.Parse(valueStr); err != nil {
			return rule.invalidErr, true
		}
	}

	return "", false
}

func CreateRowRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func InsertRowDataRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "ColumnId":
		return getColumnIdValidationError(tag)
	case "RowId":
		return getRowIdValidationError(tag)
	case "Value":
		return getValueValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func UpdateRowRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "RowId":
		return getRowIdValidationError(tag)
	case "Values":
		return getValueValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func DeleteRowDataRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "RowId":
		return getRowIdValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func BulkDeleteRowsRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "RowIds":
		switch tag {
		case "required", "min":
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
		return getModelIDValidationError(tag)
	case "ColumnId":
		return getColumnIdValidationError(tag)
	case "SourceRowId", "TargetRowId":
		return getRowIdValidationError(tag)
	case "Action":
		return getActionValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func RemoveAttachmentsRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "ColumnId":
		return getColumnIdValidationError(tag)
	case "RowId":
		return getRowIdValidationError(tag)
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
		return getModelIDValidationError(tag)
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
	case "SourceColumnID", "TargetColumnID":
		return getColumnIdValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func UpdateAttachmentRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()
	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "ColumnId":
		return getColumnIdValidationError(tag)
	case "RowId":
		return getRowIdValidationError(tag)
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
		return getModelIDValidationError(tag)
	case "ColumnId":
		return getColumnIdValidationError(tag)
	case "Updates":
		switch tag {
		case "required", "min":
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
		return getModelIDValidationError(tag)
	case "ColumnId":
		return getColumnIdValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func TrimWhitespaceRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	if strings.HasPrefix(field, "Columns") {
		return getColumnsValidationError(tag)
	}

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "TrimMode":
		return getActionValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func CaseNormalizationRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	if strings.HasPrefix(field, "Columns") {
		return getColumnsValidationError(tag)
	}

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "CaseFormat":
		return getActionValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func FindReplaceRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	if strings.HasPrefix(field, "Columns") {
		return getColumnsValidationError(tag)
	}

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "FindValue":
		return getValueValidationError(tag)
	case "MatchType":
		return getActionValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func RemoveSpecialCharactersRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	if strings.HasPrefix(field, "Columns") {
		return getColumnsValidationError(tag)
	}

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "SpecialCharactersType":
		return getActionValidationError(tag)
	case "CustomCharacter":
		return getValueValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func RemoveDuplicatesRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	if strings.HasPrefix(field, "Columns") {
		return getColumnsValidationError(tag)
	}

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "Duplicate", "KeepRule":
		return getActionValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func RemoveFormattingRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	if strings.HasPrefix(field, "Columns") {
		return getColumnsValidationError(tag)
	}

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "Formatting":
		return getActionValidationError(tag)
	case "CustomPattern":
		return getValueValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func MergeColumnsRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	if strings.HasPrefix(field, "Columns") {
		return getColumnsValidationError(tag)
	}

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "MergeFormat":
		return getActionValidationError(tag)
	case "CustomSeparator":
		return getValueValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func ExtractSubstringRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "ColumnId":
		return getColumnsValidationError(tag)
	case "ExtractionType":
		return getActionValidationError(tag)
	case "ExtractionMethod":
		return getActionValidationError(tag)
	case "StartAfter", "EndBefore":
		return getValueValidationError(tag)
	default:
		return responseConst.Error.ValidationFailed
	}
}

func ColumnSplitRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "ColumnID":
		return getColumnIdValidationError(tag)
	case "SplitBy":
		return getMetaValidationError(tag)
	case "Where":
		return getActionValidationError(tag)
	case "Limit":
		return responseConst.TableError.MetaInvalid
	default:
		return responseConst.Error.ValidationFailed
	}
}

func FuzzyDuplicatesRequestValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	if strings.HasPrefix(field, "Columns") {
		return getColumnsValidationError(tag)
	}

	switch field {
	case "ModelID":
		return getModelIDValidationError(tag)
	case "Duplicate", "KeepRule":
		return getActionValidationError(tag)
	case "Threshold":
		return responseConst.TableError.ActionInvalid
	default:
		return responseConst.Error.ValidationFailed
	}
}
