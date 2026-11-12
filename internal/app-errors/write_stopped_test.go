// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package app_errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/lib/pq"
)

func TestAutomationError(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		message      string
		rejected, ok bool
	}{
		{"raise exception", &pq.Error{Code: "P0001", Message: "Stock can't go below zero", Where: "PL/pgSQL function s.stock_guard() line 3 at RAISE"},
			"Stock can't go below zero", true, true},
		{"raise with its own errcode", &pq.Error{Code: "23514", Message: " Qty must be positive ", Where: "PL/pgSQL function s.f() line 2 at RAISE"},
			"Qty must be positive", true, true},
		{"wrapped by the library", &dbModels.DBError{Code: "P0001", Message: "No", Err: fmt.Errorf("insert: %w", &pq.Error{Code: "P0001", Message: "No"})},
			"No", true, true},
		{"mistake in the function", &pq.Error{Code: "22012", Message: "division by zero", Where: "PL/pgSQL function s.f() line 4 at PERFORM"},
			"", false, true},
		{"not from a trigger", &pq.Error{Code: "22012", Message: "division by zero"}, "", false, false},
		{"not a Postgres error", errors.New("boom"), "", false, false},
		{"nil", nil, "", false, false},
	}
	for _, tc := range cases {
		message, rejected, ok := AutomationError(tc.err)
		if message != tc.message || rejected != tc.rejected || ok != tc.ok {
			t.Errorf("%s: got (%q, %v, %v), want (%q, %v, %v)", tc.name, message, rejected, ok, tc.message, tc.rejected, tc.ok)
		}
	}
}

func TestAutomationErrorShortensLongMessages(t *testing.T) {
	message, _, _ := AutomationError(&pq.Error{Code: "P0001", Message: strings.Repeat("é", 600)})
	if got := len([]rune(message)); got != maxAutomationMessage+1 {
		t.Fatalf("got %d characters, want %d and an ellipsis", got, maxAutomationMessage)
	}
}

func TestIsDatabaseBusy(t *testing.T) {
	for _, code := range []string{"40P01", "40001", "55P03"} {
		if !IsDatabaseBusy(&pq.Error{Code: pq.ErrorCode(code)}) {
			t.Errorf("%s should be busy", code)
		}
	}
	for _, err := range []error{&pq.Error{Code: "23505"}, errors.New("deadlock"), nil} {
		if IsDatabaseBusy(err) {
			t.Errorf("%v should not be busy", err)
		}
	}
}
