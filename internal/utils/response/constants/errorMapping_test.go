package constants

import (
	"fmt"
	"testing"
)

// TestMappingHasErrorCodes ensures each mapped error has a corresponding ErrorCodes entry.
func TestMappingHasErrorCodes(t *testing.T) {
	for _, entry := range AllErrorMappings() {
		if _, ok := ErrorCodes[entry.Code]; !ok {
			t.Fatalf("no ErrorCodes entry for mapped error %v -> %s", entry.Err, entry.Code)
		}
	}
}

// TestMapErrorUsesErrorsIs ensures wrapped errors still resolve.
func TestMapErrorUsesErrorsIs(t *testing.T) {
	for _, entry := range AllErrorMappings() {
		wrapped := fmt.Errorf("wrap: %w", entry.Err)
		code := MapError(wrapped)
		if code != entry.Code {
			t.Fatalf("expected code %s for wrapped error %v, got %s", entry.Code, entry.Err, code)
		}
	}
}
