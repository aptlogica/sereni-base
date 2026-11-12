// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"errors"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func detailedMessage(t *testing.T, err error) string {
	t.Helper()
	var detailed *app_errors.DetailedError
	require.True(t, errors.As(err, &detailed), "%v has no message for users", err)
	return detailed.Message
}

func TestWriteStoppedError(t *testing.T) {
	err, ok := writeStoppedError(&pq.Error{Code: "P0001", Message: "Stock can't go below zero"})
	require.True(t, ok)
	assert.ErrorIs(t, err, app_errors.AutomationRejected)
	assert.Equal(t, "Stock can't go below zero", detailedMessage(t, err))

	err, ok = writeStoppedError(&pq.Error{Code: "P0001", Message: ""})
	require.True(t, ok)
	assert.Equal(t, automationRejectedFallback, detailedMessage(t, err))

	err, ok = writeStoppedError(&pq.Error{Code: "42703", Message: `column "qty" does not exist`, Where: "PL/pgSQL function s.f() line 2 at IF"})
	require.True(t, ok)
	assert.ErrorIs(t, err, app_errors.AutomationFailed)
	assert.Equal(t, automationFailedMessage, detailedMessage(t, err))

	err, ok = writeStoppedError(&pq.Error{Code: "40P01", Message: "deadlock detected"})
	require.True(t, ok)
	assert.ErrorIs(t, err, app_errors.DatabaseBusy)
	assert.Equal(t, tableBusyMessage, detailedMessage(t, err))

	_, ok = writeStoppedError(&pq.Error{Code: "23505"})
	assert.False(t, ok, "key violations keep their own messages")
}

// The shared write-error helpers check automations and busy tables before anything else, without reading
// the table's keys
func TestWriteErrorHelpersReportAutomations(t *testing.T) {
	s := tableManagementService{}
	raised := &pq.Error{Code: "P0001", Message: "Stock can't go below zero"}
	assert.ErrorIs(t, s.keyWriteError(context.Background(), "s", "t", raised, "write"), app_errors.AutomationRejected)
	assert.ErrorIs(t, s.recordWriteError(context.Background(), "s", raised, "write"), app_errors.AutomationRejected)
}
