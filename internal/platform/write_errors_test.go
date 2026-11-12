// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"errors"
	"fmt"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func asAPIError(t *testing.T, err error) *Error {
	t.Helper()
	var apiErr *Error
	require.True(t, errors.As(err, &apiErr), "%v is not an API error", err)
	return apiErr
}

func TestWriteErrorForAutomationsAndBusyTables(t *testing.T) {
	ids := func(int) string { return "rec_1" }
	rejected := app_errors.WithMessage(app_errors.AutomationRejected, "Stock can't go below zero")

	// A rejection names the record and carries the automation's message
	got := asAPIError(t, writeError(&app_errors.RecordWriteError{Index: 2, Err: rejected}, ids))
	assert.Equal(t, CodeValidationFailed, got.Code)
	require.Len(t, got.Details, 1)
	assert.Equal(t, 2, *got.Details[0].Index)
	assert.Equal(t, "Stock can't go below zero", got.Details[0].Issue)

	// Without a record (a delete, or a commit), the message is the error's message
	got = asAPIError(t, writeError(rejected, ids))
	assert.Equal(t, CodeValidationFailed, got.Code)
	assert.Equal(t, "Stock can't go below zero", got.Message)

	failed := app_errors.WithMessage(app_errors.AutomationFailed, "An automation on this table failed.")
	got = asAPIError(t, writeError(&app_errors.RecordWriteError{Index: 0, Err: failed}, ids))
	assert.Equal(t, CodeInternalError, got.Code)
	assert.Equal(t, "An automation on this table failed.", got.Message)

	busy := app_errors.WithMessage(app_errors.DatabaseBusy, "The table is busy right now.")
	got = asAPIError(t, writeError(&app_errors.RecordWriteError{Index: 0, Err: busy}, ids))
	assert.Equal(t, CodeConflict, got.Code)
	assert.Equal(t, "The table is busy right now.", got.Message)

	// Other errors are unchanged
	assert.Nil(t, stoppedWriteError(errors.New("boom")))
	assert.ErrorIs(t, writeError(&app_errors.RecordWriteError{Err: app_errors.DatabaseError}, ids), app_errors.DatabaseError)
}

func TestUserMessageNeverShowsInternalText(t *testing.T) {
	assert.Equal(t, "Stock can't go below zero", userMessage(app_errors.WithMessage(app_errors.AutomationRejected, "Stock can't go below zero")))
	assert.Equal(t, "Name is taken", userMessage(&app_errors.APIError{Message: "Name is taken"}))
	// A known error without its own message: the app's message for its kind
	assert.Equal(t, "Automation not found", userMessage(fmt.Errorf("lookup: %w", app_errors.AutomationNotFound)))
	// Unknown or database errors: a plain message, not their text
	assert.Equal(t, cantSaveMessage, userMessage(errors.New(`pq: relation "x" does not exist`)))
	assert.Equal(t, cantSaveMessage, userMessage(app_errors.DatabaseError))
}
