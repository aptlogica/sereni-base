package tests

import (
	"fmt"
	"net/http"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/utils/response/constants"
)

// TestRelationErrorCodes checks the link/lookup errors map to their codes and HTTP statuses.
func TestRelationErrorCodes(t *testing.T) {
	cases := []struct {
		err    error
		code   constants.ResponseCode
		raw    string
		status int
	}{
		{app_errors.SelfReferenceNotAllowed, constants.TableError.SelfReferenceNotAllowed, "TBL_1075", http.StatusBadRequest},
		{app_errors.LinkCycleDetected, constants.TableError.LinkCycleDetected, "TBL_1076", http.StatusBadRequest},
		{app_errors.InvalidLookupLinkColumn, constants.TableError.InvalidLookupLinkColumn, "TBL_1077", http.StatusBadRequest},
		{app_errors.LinkTargetRowNotFound, constants.TableError.LinkTargetRowNotFound, "TBL_1078", http.StatusNotFound},
		{app_errors.InvalidLookupTargetColumn, constants.TableError.InvalidLookupTargetColumn, "TBL_1079", http.StatusBadRequest},
		{app_errors.LinkColumnNotWritable, constants.TableError.LinkColumnNotWritable, "TBL_1080", http.StatusBadRequest},
	}

	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			if string(c.code) != c.raw {
				t.Fatalf("code = %s, want %s", c.code, c.raw)
			}
			if got := constants.MapError(c.err); got != c.code {
				t.Fatalf("MapError(%v) = %s, want %s", c.err, got, c.code)
			}
			// Wrapped errors still resolve.
			if got := constants.MapError(fmt.Errorf("wrap: %w", c.err)); got != c.code {
				t.Fatalf("MapError(wrapped %v) = %s, want %s", c.err, got, c.code)
			}
			meta, ok := constants.TableErrorCodes[c.code]
			if !ok {
				t.Fatalf("no TableErrorCodes entry for %s", c.code)
			}
			if meta.HTTPStatus != c.status {
				t.Fatalf("status = %d, want %d", meta.HTTPStatus, c.status)
			}
			if meta.Message == "" || meta.Description == "" {
				t.Fatalf("missing message/description for %s", c.code)
			}
		})
	}
}
