package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/utils/response/constants"
)

func TestTableErrorCodes(t *testing.T) {
	// Test that all TableError fields are properly initialized
	tableErrorFields := map[string]constants.ResponseCode{
		"BaseIDRequired":      constants.TableError.BaseIDRequired,
		"BaseIDInvalid":       constants.TableError.BaseIDInvalid,
		"WorkspaceIDRequired": constants.TableError.WorkspaceIDRequired,
		"WorkspaceIDInvalid":  constants.TableError.WorkspaceIDInvalid,
		"TitleRequired":       constants.TableError.TitleRequired,
		"TitleInvalid":        constants.TableError.TitleInvalid,
		"TableNotFound":       constants.TableError.TableNotFound,
		"TableAlreadyExists":  constants.TableError.TableAlreadyExists,
		"TableNotCreated":     constants.TableError.TableNotCreated,
		"TableNotUpdated":     constants.TableError.TableNotUpdated,
		"TableNotDeleted":     constants.TableError.TableNotDeleted,
		"ColumnNameRequired":  constants.TableError.ColumnNameRequired,
		"ColumnNameInvalid":   constants.TableError.ColumnNameInvalid,
		"ColumnNotFound":      constants.TableError.ColumnNotFound,
		"ValueRequired":       constants.TableError.ValueRequired,
		"ValueInvalid":        constants.TableError.ValueInvalid,
		"RowIdRequired":       constants.TableError.RowIdRequired,
		"RowIdInvalid":        constants.TableError.RowIdInvalid,
		"MetaRequired":        constants.TableError.MetaRequired,
		"MetaInvalid":         constants.TableError.MetaInvalid,
		"LimitRequired":       constants.TableError.LimitRequired,
		"LimitInvalid":        constants.TableError.LimitInvalid,
		"PageRequired":        constants.TableError.PageRequired,
		"PageInvalid":         constants.TableError.PageInvalid,
	}

	// Test that all fields have non-empty values
	for fieldName, code := range tableErrorFields {
		if code == "" {
			t.Errorf("TableError.%s is empty", fieldName)
		}
		if string(code) == "" {
			t.Errorf("TableError.%s string conversion is empty", fieldName)
		}
	}

	// Test that all table error codes exist in ErrorCodes map
	for fieldName, code := range tableErrorFields {
		if _, exists := constants.ErrorCodes[code]; !exists {
			t.Errorf("TableError.%s code %s not found in ErrorCodes map", fieldName, code)
		}
	}
}

func TestTableSuccessCodes(t *testing.T) {
	// Test that all TableSuccess fields are properly initialized
	tableSuccessFields := map[string]constants.ResponseCode{
		"TableCreated":    constants.TableSuccess.TableCreated,
		"TableUpdated":    constants.TableSuccess.TableUpdated,
		"TableDeleted":    constants.TableSuccess.TableDeleted,
		"TableFetched":    constants.TableSuccess.TableFetched,
		"ColumnAdded":     constants.TableSuccess.ColumnAdded,
		"ColumnFetched":   constants.TableSuccess.ColumnFetched,
		"ColumnUpdated":   constants.TableSuccess.ColumnUpdated,
		"ColumnDeleted":   constants.TableSuccess.ColumnDeleted,
		"ViewCreated":     constants.TableSuccess.ViewCreated,
		"ViewFetched":     constants.TableSuccess.ViewFetched,
		"ViewUpdated":     constants.TableSuccess.ViewUpdated,
		"ViewDeleted":     constants.TableSuccess.ViewDeleted,
		"RecordCreated":   constants.TableSuccess.RecordCreated,
		"RecordsFetched":  constants.TableSuccess.RecordsFetched,
		"RowDataInserted": constants.TableSuccess.RowDataInserted,
		"RowDeleted":      constants.TableSuccess.RowDeleted,
		"ColumnReordered": constants.TableSuccess.ColumnReordered,
	}

	// Test that all fields have non-empty values
	for fieldName, code := range tableSuccessFields {
		if code == "" {
			t.Errorf("TableSuccess.%s is empty", fieldName)
		}
		if string(code) == "" {
			t.Errorf("TableSuccess.%s string conversion is empty", fieldName)
		}
	}

	// Test that all table success codes exist in SuccessCodes map
	for fieldName, code := range tableSuccessFields {
		if _, exists := constants.SuccessCodes[code]; !exists {
			t.Errorf("TableSuccess.%s code %s not found in SuccessCodes map", fieldName, code)
		}
	}
}

func TestTableErrorCodesMap(t *testing.T) {
	// Test that TableErrorCodes map has expected entries
	expectedTableErrorCodes := []constants.ResponseCode{
		"TBL_1001", // BaseIDRequired
		"TBL_1002", // BaseIDInvalid
		"TBL_1003", // WorkspaceIDRequired
		"TBL_1004", // WorkspaceIDInvalid
		"TBL_1005", // TitleRequired
		"TBL_1012", // TableNotFound
		"TBL_1013", // TableAlreadyExists
		"TBL_1014", // TableNotCreated
		"TBL_1015", // TableNotUpdated
		"TBL_1016", // TableNotDeleted
		"TBL_1019", // ColumnNameRequired
		"TBL_1035", // ColumnNotFound
		"TBL_1039", // ValueRequired
		"TBL_1041", // RowIdRequired
		"TBL_1043", // MetaRequired
		"TBL_1052", // LimitRequired
		"TBL_1054", // PageRequired
	}

	for _, code := range expectedTableErrorCodes {
		if _, exists := constants.ErrorCodes[code]; !exists {
			t.Errorf("Expected table error code %s not found in ErrorCodes", code)
		}
	}
}

func TestTableSuccessCodesMap(t *testing.T) {
	// Test that TableSuccessCodes map has expected entries
	expectedTableSuccessCodes := []constants.ResponseCode{
		"TBL_SUCCESS_5001", // TableCreated
		"TBL_SUCCESS_5002", // TableUpdated
		"TBL_SUCCESS_5003", // TableDeleted
		"TBL_SUCCESS_5004", // TableFetched
		"TBL_SUCCESS_5005", // ColumnAdded
		"TBL_SUCCESS_5006", // ColumnFetched
		"TBL_SUCCESS_5007", // ViewCreated
		"TBL_SUCCESS_5008", // ViewFetched
		"TBL_SUCCESS_5009", // ViewUpdated
		"TBL_SUCCESS_5010", // ViewDeleted
		"TBL_SUCCESS_5011", // ColumnUpdated
		"TBL_SUCCESS_5012", // ColumnDeleted
		"TBL_SUCCESS_5013", // RecordCreated
		"TBL_SUCCESS_5014", // RecordsFetched
		"TBL_SUCCESS_5015", // RowDataInserted
		"TBL_SUCCESS_5016", // RowDeleted
		"TBL_SUCCESS_5017", // ColumnReordered
	}

	for _, code := range expectedTableSuccessCodes {
		if _, exists := constants.SuccessCodes[code]; !exists {
			t.Errorf("Expected table success code %s not found in SuccessCodes", code)
		}
	}
}

func TestTableErrorCodesHTTPStatus(t *testing.T) {
	// Test that table error codes have appropriate HTTP status codes
	testCases := []struct {
		code            constants.ResponseCode
		expectedStatus  int
		expectedMessage string
	}{
		{"TBL_1001", http.StatusBadRequest, "Base ID is required"},
		{"TBL_1002", http.StatusBadRequest, "Invalid Base ID"},
		{"TBL_1003", http.StatusBadRequest, "Workspace ID is required"},
		{"TBL_1004", http.StatusBadRequest, "Invalid Workspace ID"},
		{"TBL_1005", http.StatusBadRequest, "Title is required"},
		{"TBL_1012", http.StatusNotFound, "Table not found"},
		{"TBL_1013", http.StatusConflict, "Table already exists"},
		{"TBL_1014", http.StatusInternalServerError, "Table not created"},
		{"TBL_1015", http.StatusInternalServerError, "Table not updated"},
		{"TBL_1016", http.StatusInternalServerError, "Table not deleted"},
		{"TBL_1019", http.StatusBadRequest, "Column name is required"},
		{"TBL_1035", http.StatusNotFound, "Column not found"},
		{"TBL_1039", http.StatusBadRequest, "Value is required"},
		{"TBL_1041", http.StatusBadRequest, "Row ID is required"},
		{"TBL_1043", http.StatusBadRequest, "Meta is required"},
		{"TBL_1052", http.StatusBadRequest, "Limit is required"},
		{"TBL_1054", http.StatusBadRequest, "Page number is required"},
	}

	for _, tc := range testCases {
		if meta, exists := constants.ErrorCodes[tc.code]; exists {
			if meta.HTTPStatus != tc.expectedStatus {
				t.Errorf("Table error code %s has HTTP status %d, expected %d", tc.code, meta.HTTPStatus, tc.expectedStatus)
			}
			if meta.Message != tc.expectedMessage {
				t.Errorf("Table error code %s has message '%s', expected '%s'", tc.code, meta.Message, tc.expectedMessage)
			}
		} else {
			t.Errorf("Table error code %s not found in ErrorCodes", tc.code)
		}
	}
}

func TestTableSuccessCodesHTTPStatus(t *testing.T) {
	// Test that table success codes have appropriate HTTP status codes
	testCases := []struct {
		code            constants.ResponseCode
		expectedStatus  int
		expectedMessage string
	}{
		{"TBL_SUCCESS_5001", http.StatusCreated, "Table created successfully"},
		{"TBL_SUCCESS_5002", http.StatusOK, "Table updated successfully"},
		{"TBL_SUCCESS_5003", http.StatusOK, "Table deleted successfully"},
		{"TBL_SUCCESS_5004", http.StatusOK, "Table fetched successfully"},
		{"TBL_SUCCESS_5005", http.StatusOK, "Column added successfully"},
		{"TBL_SUCCESS_5006", http.StatusOK, "Column fetched successfully"},
		{"TBL_SUCCESS_5007", http.StatusCreated, "View created successfully"},
		{"TBL_SUCCESS_5008", http.StatusOK, "View fetched successfully"},
		{"TBL_SUCCESS_5009", http.StatusOK, "View updated successfully"},
		{"TBL_SUCCESS_5010", http.StatusOK, "View deleted successfully"},
		{"TBL_SUCCESS_5011", http.StatusOK, "Column updated successfully"},
		{"TBL_SUCCESS_5012", http.StatusOK, "Column deleted successfully"},
		{"TBL_SUCCESS_5013", http.StatusCreated, "Record created successfully"},
		{"TBL_SUCCESS_5014", http.StatusOK, "Records fetched successfully"},
		{"TBL_SUCCESS_5015", http.StatusCreated, "Row data inserted successfully"},
		{"TBL_SUCCESS_5016", http.StatusOK, "Row deleted successfully"},
		{"TBL_SUCCESS_5017", http.StatusOK, "Column reordered successfully"},
	}

	for _, tc := range testCases {
		if meta, exists := constants.SuccessCodes[tc.code]; exists {
			if meta.HTTPStatus != tc.expectedStatus {
				t.Errorf("Table success code %s has HTTP status %d, expected %d", tc.code, meta.HTTPStatus, tc.expectedStatus)
			}
			if meta.Message != tc.expectedMessage {
				t.Errorf("Table success code %s has message '%s', expected '%s'", tc.code, meta.Message, tc.expectedMessage)
			}
		} else {
			t.Errorf("Table success code %s not found in SuccessCodes", tc.code)
		}
	}
}

func TestTableErrorCodePatterns(t *testing.T) {
	// Test that table error codes follow expected patterns
	for code := range constants.ErrorCodes {
		codeStr := string(code)
		if len(codeStr) > 0 && codeStr[:4] == "TBL_" && !strings.Contains(codeStr, "SUCCESS") {
			// This is a table error code, test it has proper structure
			if len(codeStr) < 8 {
				t.Errorf("Table error code %s is too short", code)
			}
		}
	}
}

func TestTableSuccessCodePatterns(t *testing.T) {
	// Test that table success codes follow expected patterns
	for code := range constants.SuccessCodes {
		if len(string(code)) > 0 && string(code)[:12] == "TBL_SUCCESS_" {
			// This is a table success code, test it has proper structure
			if len(string(code)) < 15 {
				t.Errorf("Table success code %s is too short", code)
			}
		}
	}
}
