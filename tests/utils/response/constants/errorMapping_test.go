package tests

import (
	"fmt"
	"testing"

	"serenibase/internal/utils/response/constants"
)

// TestMappingHasErrorCodes ensures each mapped error has a corresponding ErrorCodes entry.
func TestMappingHasErrorCodes(t *testing.T) {
	for _, entry := range constants.AllErrorMappings() {
		if _, ok := constants.ErrorCodes[entry.Code]; !ok {
			t.Fatalf("no ErrorCodes entry for mapped error %v -> %s", entry.Err, entry.Code)
		}
	}
}

// TestMapErrorUsesErrorsIs ensures wrapped errors still resolve.
func TestMapErrorUsesErrorsIs(t *testing.T) {
	for _, entry := range constants.AllErrorMappings() {
		wrapped := fmt.Errorf("wrap: %w", entry.Err)
		code := constants.MapError(wrapped)
		if code != entry.Code {
			t.Fatalf("expected code %s for wrapped error %v, got %s", entry.Code, entry.Err, code)
		}
	}
}
