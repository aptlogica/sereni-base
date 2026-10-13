// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package app_errors

import (
	"errors"
	"regexp"
	"strings"

	"github.com/lib/pq"
)

// pqUniqueViolation is the Postgres code for a duplicate key
const pqUniqueViolation = "23505"

var uniqueViolationRegexp = regexp.MustCompile(`duplicate key value violates unique constraint "([^"]+)"`)

// UniqueViolation returns the constraint a write broke when Postgres rejected it as a duplicate,
// with Postgres' detail, e.g. `Key (email)=(a@x.com) already exists.`
func UniqueViolation(err error) (constraint, detail string, ok bool) {
	if err == nil {
		return "", "", false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		if pqErr.Code != pqUniqueViolation {
			return "", "", false
		}
		return pqErr.Constraint, pqErr.Detail, true
	}
	// Layers that keep only the text of the error
	if match := uniqueViolationRegexp.FindStringSubmatch(err.Error()); match != nil {
		return match[1], "", true
	}
	return "", "", false
}

// PostgresMessage is the message of a Postgres error, without the driver's prefix
func PostgresMessage(err error) string {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Message
	}
	return strings.TrimPrefix(err.Error(), "pq: ")
}
