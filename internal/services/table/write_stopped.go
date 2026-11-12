// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
)

// Messages for writes stopped by the table's automations or by a busy database
const (
	automationRejectedFallback = "An automation on this table rejected the change."
	automationFailedMessage    = "An automation on this table failed, so nothing was saved. Check the table's automations."
	tableBusyMessage           = "The table is busy right now, so nothing was saved. Try again in a moment."
)

// writeStoppedError turns a write stopped by a trigger function or by a busy database into an error with a
// clear message: what the function raised (RAISE EXCEPTION 'Stock is empty'), that an automation
// failed, or that the table is busy. ok is false for any other error.
func writeStoppedError(err error) (error, bool) {
	if message, rejected, ok := app_errors.AutomationError(err); ok {
		if rejected {
			logger.Get().Info().Err(err).Msg("write rejected by an automation")
			if message == "" {
				message = automationRejectedFallback
			}
			return app_errors.WithMessage(app_errors.AutomationRejected, message), true
		}
		logger.Get().Error().Err(err).Msg("automation failed during a write")
		return app_errors.WithMessage(app_errors.AutomationFailed, automationFailedMessage), true
	}
	if app_errors.IsDatabaseBusy(err) {
		logger.Get().Warn().Err(err).Msg("write stopped by other writes")
		return app_errors.WithMessage(app_errors.DatabaseBusy, tableBusyMessage), true
	}
	return nil, false
}
