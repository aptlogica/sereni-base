package constants

import "net/http"

var TableError = struct {
	BaseIDRequired                 ResponseCode
	BaseIDInvalid                  ResponseCode
	WorkspaceIDRequired            ResponseCode
	WorkspaceIDInvalid             ResponseCode
	TitleRequired                  ResponseCode
	TitleInvalid                   ResponseCode
	DescriptionRequired            ResponseCode
	DescriptionInvalid             ResponseCode
	OrderIndexRequired             ResponseCode
	OrderIndexInvalid              ResponseCode
	ValidationFailed               ResponseCode
	TableNotFound                  ResponseCode
	TableAlreadyExists             ResponseCode
	TableNotCreated                ResponseCode
	TableNotUpdated                ResponseCode
	TableNotDeleted                ResponseCode
	ModelIDRequired                ResponseCode
	ModelIDInvalid                 ResponseCode
	ColumnNameRequired             ResponseCode
	ColumnNameInvalid              ResponseCode
	UIDTRequired                   ResponseCode
	UIDTInvalid                    ResponseCode
	DTRequired                     ResponseCode
	DTInvalid                      ResponseCode
	VirtualRequired                ResponseCode
	VirtualInvalid                 ResponseCode
	SystemRequired                 ResponseCode
	SystemInvalid                  ResponseCode
	TypeRequired                   ResponseCode
	TypeInvalid                    ResponseCode
	ViewNotFound                   ResponseCode
	ViewUploadFailed               ResponseCode
	UpdateNotAllowed               ResponseCode
	DeleteNotAllowed               ResponseCode
	ColumnNotFound                 ResponseCode
	ColumnUpdateFailed             ResponseCode
	ColumnIdRequired               ResponseCode
	ColumnIdInvalid                ResponseCode
	ValueRequired                  ResponseCode
	ValueInvalid                   ResponseCode
	RowIdRequired                  ResponseCode
	RowIdInvalid                   ResponseCode
	InvalidColumnMetaForLinkType   ResponseCode
	MetaRequired                   ResponseCode
	MetaInvalid                    ResponseCode
	RowNotFound                    ResponseCode
	ActionRequired                 ResponseCode
	ActionInvalid                  ResponseCode
	AttachmentRequired             ResponseCode
	AttachmentInvalid              ResponseCode
	InvalidColumnMetaForLookupType ResponseCode
	LimitRequired                  ResponseCode
	LimitInvalid                   ResponseCode
	PageRequired                   ResponseCode
	PageInvalid                    ResponseCode
	AssetIdRequired                ResponseCode
	AssetIdInvalid                 ResponseCode
	ContentRequired                ResponseCode
	ContentInvalid                 ResponseCode
}{
	BaseIDRequired:                 "TBL_1001",
	BaseIDInvalid:                  "TBL_1002",
	WorkspaceIDRequired:            "TBL_1003",
	WorkspaceIDInvalid:             "TBL_1004",
	TitleRequired:                  "TBL_1005",
	TitleInvalid:                   "TBL_1006",
	DescriptionRequired:            "TBL_1007",
	DescriptionInvalid:             "TBL_1008",
	OrderIndexRequired:             "TBL_1009",
	OrderIndexInvalid:              "TBL_1010",
	ValidationFailed:               "TBL_1011",
	TableNotFound:                  "TBL_1012",
	TableAlreadyExists:             "TBL_1013",
	TableNotCreated:                "TBL_1014",
	TableNotUpdated:                "TBL_1015",
	TableNotDeleted:                "TBL_1016",
	ModelIDRequired:                "TBL_1017",
	ModelIDInvalid:                 "TBL_1018",
	ColumnNameRequired:             "TBL_1019",
	ColumnNameInvalid:              "TBL_1020",
	UIDTRequired:                   "TBL_1021",
	UIDTInvalid:                    "TBL_1022",
	DTRequired:                     "TBL_1023",
	DTInvalid:                      "TBL_1024",
	VirtualRequired:                "TBL_1025",
	VirtualInvalid:                 "TBL_1026",
	SystemRequired:                 "TBL_1027",
	SystemInvalid:                  "TBL_1028",
	TypeRequired:                   "TBL_1029",
	TypeInvalid:                    "TBL_1030",
	ViewNotFound:                   "TBL_1031",
	ViewUploadFailed:               "TBL_1032",
	UpdateNotAllowed:               "TBL_1033",
	DeleteNotAllowed:               "TBL_1034",
	ColumnNotFound:                 "TBL_1035",
	ColumnUpdateFailed:             "TBL_1036",
	ColumnIdRequired:               "TBL_1037",
	ColumnIdInvalid:                "TBL_1038",
	ValueRequired:                  "TBL_1039",
	ValueInvalid:                   "TBL_1040",
	RowIdRequired:                  "TBL_1041",
	RowIdInvalid:                   "TBL_1042",
	MetaRequired:                   "TBL_1043",
	MetaInvalid:                    "TBL_1044",
	RowNotFound:                    "TBL_1045",
	ActionRequired:                 "TBL_1046",
	ActionInvalid:                  "TBL_1047",
	InvalidColumnMetaForLinkType:   "TBL_1048",
	AttachmentRequired:             "TBL_1049",
	AttachmentInvalid:              "TBL_1050",
	InvalidColumnMetaForLookupType: "TBL_1051",
	LimitRequired:                  "TBL_1052",
	LimitInvalid:                   "TBL_1053",
	PageRequired:                   "TBL_1054",
	PageInvalid:                    "TBL_1055",
	AssetIdRequired:                "TBL_1056",
	AssetIdInvalid:                 "TBL_1057",
	ContentRequired:                "TBL_1058",
	ContentInvalid:                 "TBL_1059",
}

var TableErrorCodes = map[ResponseCode]MetaResponse{
	TableError.BaseIDRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Base ID is required",
		Description: "The base_id field is required",
	},
	TableError.BaseIDInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid Base ID",
		Description: "The provided base_id is invalid or malformed",
	},
	TableError.WorkspaceIDRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Workspace ID is required",
		Description: "The workspace_id field is required",
	},
	TableError.WorkspaceIDInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid Workspace ID",
		Description: "The provided workspace_id is invalid or malformed",
	},
	TableError.TitleRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Title is required",
		Description: "The title field is required",
	},
	TableError.TitleInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid title",
		Description: "The provided title is invalid or malformed",
	},
	TableError.DescriptionRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Description is required",
		Description: "The description field is required",
	},
	TableError.DescriptionInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid description",
		Description: "The provided description is invalid or malformed",
	},
	TableError.OrderIndexRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Order index is required",
		Description: "The order_index field is required",
	},
	TableError.OrderIndexInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid order index",
		Description: "The provided order_index is invalid or malformed",
	},
	TableError.ValidationFailed: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Validation failed",
		Description: "One or more fields failed validation",
	},
	TableError.TableNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Table not found",
		Description: "The specified table could not be found",
	},
	TableError.TableAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Table already exists",
		Description: "A table with the given information already exists",
	},
	TableError.TableNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Table not created",
		Description: "The table could not be created due to an internal error",
	},
	TableError.TableNotUpdated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Table not updated",
		Description: "The table could not be updated due to an internal error",
	},
	TableError.TableNotDeleted: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Table not deleted",
		Description: "The table could not be deleted due to an internal error",
	},
	TableError.ModelIDRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Model ID is required",
		Description: "The model_id field is required",
	},
	TableError.ModelIDInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid model ID",
		Description: "The provided model_id is invalid or malformed",
	},
	TableError.ColumnNameRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Column name is required",
		Description: "The column_name field is required",
	},
	TableError.ColumnNameInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid column name",
		Description: "The provided column_name is invalid or malformed",
	},
	TableError.UIDTRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "UIDT is required",
		Description: "The uidt field is required",
	},
	TableError.UIDTInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid UIDT",
		Description: "The provided uidt is invalid or malformed",
	},
	TableError.DTRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "DT is required",
		Description: "The dt field is required",
	},
	TableError.DTInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid DT",
		Description: "The provided dt is invalid or malformed",
	},
	TableError.VirtualRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Virtual is required",
		Description: "The virtual field is required",
	},
	TableError.VirtualInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid virtual",
		Description: "The provided virtual value is invalid or malformed",
	},
	TableError.SystemRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "System is required",
		Description: "The system field is required",
	},
	TableError.SystemInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid system",
		Description: "The provided system value is invalid or malformed",
	},
	TableError.TypeRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Type is required",
		Description: "The type field is required",
	},
	TableError.TypeInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid type",
		Description: "The provided type is invalid or malformed",
	},
	TableError.ViewNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "View not found",
		Description: "The requested view could not be found",
	},
	TableError.ViewUploadFailed: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "View upload failed",
		Description: "Failed to upload the view due to an internal error",
	},
	TableError.UpdateNotAllowed: {
		HTTPStatus:  http.StatusForbidden,
		Message:     "Update not allowed",
		Description: "Updates are not permitted for this table",
	},
	TableError.DeleteNotAllowed: {
		HTTPStatus:  http.StatusForbidden,
		Message:     "Delete not allowed",
		Description: "Deletes are not permitted for this table",
	},
	TableError.ColumnNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Column not found",
		Description: "The requested column could not be found",
	},
	TableError.ColumnUpdateFailed: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Column update failed",
		Description: "Failed to update the column due to an internal error",
	},
	TableError.ColumnIdRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Column ID is required",
		Description: "The column ID field is required",
	},
	TableError.ColumnIdInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid column ID",
		Description: "The provided column ID is invalid or malformed",
	},
	TableError.ValueRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Value is required",
		Description: "The value field is required",
	},
	TableError.ValueInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid value",
		Description: "The provided value is invalid or malformed",
	},
	TableError.RowIdRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Row ID is required",
		Description: "The row ID field is required",
	},
	TableError.RowIdInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid row ID",
		Description: "The provided row ID is invalid or malformed",
	},
	TableError.InvalidColumnMetaForLinkType: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid payload",
		Description: "The meta field for a link type column is missing required relation information",
	},
	TableError.MetaRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Meta is required",
		Description: "The meta field is required",
	},
	TableError.MetaInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid meta",
		Description: "The provided meta value is invalid or malformed",
	},
	TableError.RowNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Row not found",
		Description: "The requested row could not be found",
	},
	TableError.ActionRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Action is required",
		Description: "The action field is required",
	},
	TableError.ActionInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid action",
		Description: "The provided action is invalid or not supported",
	},
	TableError.AttachmentRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Attachment is required",
		Description: "The attachment field is required",
	},
	TableError.AttachmentInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid attachment",
		Description: "The provided attachment is invalid or malformed",
	},
	TableError.InvalidColumnMetaForLookupType: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid payload",
		Description: "The meta field for a lookup type column is missing required information",
	},
	TableError.LimitRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Limit is required",
		Description: "The page size (limit) field is required",
	},
	TableError.LimitInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid limit",
		Description: "The provided page size (limit) is invalid or malformed",
	},
	TableError.PageRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Page number is required",
		Description: "The page number field is required",
	},
	TableError.PageInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid page number",
		Description: "The provided page number is invalid or malformed",
	},
	TableError.AssetIdRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Asset ID is required",
		Description: "The asset ID field is required",
	},
	TableError.AssetIdInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid asset ID",
		Description: "The provided asset ID is invalid or malformed",
	},
	TableError.ContentRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Content is required",
		Description: "The content field is required",
	},
	TableError.ContentInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid content",
		Description: "The provided content is invalid or malformed",
	},
}

var TableSuccess = struct {
	TableCreated    ResponseCode
	TableUpdated    ResponseCode
	TableDeleted    ResponseCode
	TableFetched    ResponseCode
	ColumnAdded     ResponseCode
	ColumnFetched   ResponseCode
	ColumnUpdated   ResponseCode
	ViewCreated     ResponseCode
	ViewFetched     ResponseCode
	ViewUpdated     ResponseCode
	ViewDeleted     ResponseCode
	ColumnDeleted   ResponseCode
	RecordCreated   ResponseCode
	RecordsFetched  ResponseCode
	RowDataInserted ResponseCode
	RowDeleted      ResponseCode
	ColumnReordered ResponseCode
}{
	TableCreated:    "TBL_SUCCESS_5001",
	TableUpdated:    "TBL_SUCCESS_5002",
	TableDeleted:    "TBL_SUCCESS_5003",
	TableFetched:    "TBL_SUCCESS_5004",
	ColumnAdded:     "TBL_SUCCESS_5005",
	ColumnFetched:   "TBL_SUCCESS_5006",
	ColumnUpdated:   "TBL_SUCCESS_5011",
	ViewCreated:     "TBL_SUCCESS_5007",
	ViewFetched:     "TBL_SUCCESS_5008",
	ViewUpdated:     "TBL_SUCCESS_5009",
	ViewDeleted:     "TBL_SUCCESS_5010",
	ColumnDeleted:   "TBL_SUCCESS_5012",
	RecordCreated:   "TBL_SUCCESS_5013",
	RecordsFetched:  "TBL_SUCCESS_5014",
	RowDataInserted: "TBL_SUCCESS_5015",
	RowDeleted:      "TBL_SUCCESS_5016",
	ColumnReordered: "TBL_SUCCESS_5017",
}

var TableSuccessCodes = map[ResponseCode]MetaResponse{
	TableSuccess.TableCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Table created successfully",
		Description: "The table has been created successfully",
	},
	TableSuccess.TableUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Table updated successfully",
		Description: "The table has been updated successfully",
	},
	TableSuccess.TableDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Table deleted successfully",
		Description: "The table has been deleted successfully",
	},
	TableSuccess.TableFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Table fetched successfully",
		Description: "The table has been fetched successfully",
	},
	TableSuccess.ColumnAdded: {
		HTTPStatus:  http.StatusOK,
		Message:     "Column added successfully",
		Description: "The column has been added to the table successfully",
	},
	TableSuccess.ColumnFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Column fetched successfully",
		Description: "The column has been fetched successfully",
	},
	TableSuccess.ViewCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "View created successfully",
		Description: "The view has been created successfully",
	},
	TableSuccess.ViewFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "View fetched successfully",
		Description: "The view has been fetched successfully",
	},
	TableSuccess.ViewUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "View updated successfully",
		Description: "The view has been updated successfully",
	},
	TableSuccess.ViewDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "View deleted successfully",
		Description: "The view has been deleted successfully",
	},
	TableSuccess.ColumnUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Column updated successfully",
		Description: "The column has been updated successfully",
	},
	TableSuccess.ColumnDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Column deleted successfully",
		Description: "The column has been deleted successfully",
	},
	TableSuccess.RecordCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Record created successfully",
		Description: "The record has been created successfully",
	},
	TableSuccess.RecordsFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Records fetched successfully",
		Description: "The records have been fetched successfully",
	},
	TableSuccess.RowDataInserted: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Row data inserted successfully",
		Description: "The row data has been inserted successfully",
	},
	TableSuccess.RowDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Row deleted successfully",
		Description: "The row has been deleted successfully",
	},
	TableSuccess.ColumnReordered: {
		HTTPStatus:  http.StatusOK,
		Message:     "Column reordered successfully",
		Description: "The columns have been reordered successfully",
	},
}
