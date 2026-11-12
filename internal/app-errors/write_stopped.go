// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package app_errors

import (
	"errors"
	"strings"

	"github.com/lib/pq"
)

// Writes to a user table stopped by the table's automations or by a busy database
var (
	// AutomationRejected: a trigger function raised an error on purpose (RAISE EXCEPTION); its message is for the user
	AutomationRejected = errors.New("an automation rejected the record")
	// AutomationFailed: a trigger function failed (a mistake in its code)
	AutomationFailed = errors.New("an automation of the table failed")
	// DatabaseBusy: the write lost a deadlock, a serialization conflict or a lock wait; trying again works
	DatabaseBusy = errors.New("the table is busy")
)

// InvalidFieldValue: a value the field's settings don't allow (an unknown option, a bad email, ...)
var InvalidFieldValue = errors.New("value not allowed for the field")

const (
	pqRaiseException = "P0001"
	// maxAutomationMessage keeps a raised message to a readable length
	maxAutomationMessage = 500
)

// busyCodes are Postgres codes for writes that may succeed when tried again: deadlock_detected,
// serialization_failure and lock_not_available
var busyCodes = map[string]bool{"40P01": true, "40001": true, "55P03": true}

// AutomationError tells whether a write was stopped by code running in a trigger function. rejected is true
// when the function raised the error itself (RAISE), and message is what it raised; otherwise the function
// failed and message is empty.
func AutomationError(err error) (message string, rejected bool, ok bool) {
	var pqErr *pq.Error
	if err == nil || !errors.As(err, &pqErr) {
		return "", false, false
	}
	// RAISE EXCEPTION without an ERRCODE is P0001; a RAISE with its own ERRCODE is still "at RAISE"
	if string(pqErr.Code) == pqRaiseException || strings.Contains(pqErr.Where, " at RAISE") {
		message = strings.TrimSpace(pqErr.Message)
		if r := []rune(message); len(r) > maxAutomationMessage {
			message = string(r[:maxAutomationMessage]) + "…"
		}
		return message, true, true
	}
	if strings.Contains(pqErr.Where, "PL/pgSQL function") {
		return "", false, true
	}
	return "", false, false
}

// IsDatabaseBusy tells whether a write failed only because of other writes at the same time
func IsDatabaseBusy(err error) bool {
	if err == nil {
		return false
	}
	dbErr, ok := DBError(err)
	return ok && busyCodes[dbErr.Code]
}
