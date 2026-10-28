// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package app_errors

import (
	"errors"
	"regexp"
	"strings"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/lib/pq"
)

// pqUniqueViolation is the Postgres code for a duplicate key
const pqUniqueViolation = "23505"

var uniqueViolationRegexp = regexp.MustCompile(`duplicate key value violates unique constraint "([^"]+)"`)

// DBError returns the typed Postgres error in err's chain. The library returns *models.DBError; a bare
// *pq.Error (from queries run on repo.DB directly) is converted, so callers handle both the same way.
func DBError(err error) (*dbModels.DBError, bool) {
	if dbErr, ok := dbModels.AsDBError(err); ok {
		return dbErr, true
	}
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return nil, false
	}
	return &dbModels.DBError{
		Code:       string(pqErr.Code),
		Constraint: pqErr.Constraint,
		Schema:     pqErr.Schema,
		Table:      pqErr.Table,
		Column:     pqErr.Column,
		Detail:     pqErr.Detail,
		Message:    pqErr.Message,
		Err:        err,
	}, true
}

// UniqueViolation returns the constraint a write broke when Postgres rejected it as a duplicate,
// with Postgres' detail, e.g. `Key (email)=(a@x.com) already exists.`
func UniqueViolation(err error) (constraint, detail string, ok bool) {
	if err == nil {
		return "", "", false
	}
	if dbErr, ok := DBError(err); ok {
		if dbErr.Code != pqUniqueViolation {
			return "", "", false
		}
		return dbErr.Constraint, dbErr.Detail, true
	}
	// Layers that keep only the text of the error
	if match := uniqueViolationRegexp.FindStringSubmatch(err.Error()); match != nil {
		return match[1], "", true
	}
	return "", "", false
}

// pqCheckViolation is the Postgres code for a row that breaks a CHECK constraint
const pqCheckViolation = "23514"

var checkViolationRegexp = regexp.MustCompile(`violates check constraint "([^"]+)"`)

// CheckViolation returns the CHECK constraint a write broke
func CheckViolation(err error) (constraint string, ok bool) {
	if err == nil {
		return "", false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Constraint, pqErr.Code == pqCheckViolation
	}
	// Layers that keep only the text of the error
	if match := checkViolationRegexp.FindStringSubmatch(err.Error()); match != nil {
		return match[1], true
	}
	return "", false
}

// PostgresMessage is the message of a Postgres error, without the driver's prefix
func PostgresMessage(err error) string {
	if dbErr, ok := DBError(err); ok {
		return dbErr.Message
	}
	return strings.TrimPrefix(err.Error(), "pq: ")
}
