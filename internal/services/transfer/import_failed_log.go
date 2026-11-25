// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
)

// maxLoggedProblems is how many problems the log of a failed import lists; the rest are counted
const maxLoggedProblems = 1000

// importStepError says which step of the import failed ("creating the fields"). Its text is the one of the error it
// wraps, and errors.Is and errors.As see through it, so nothing that looks at the error changes.
type importStepError struct {
	step string
	err  error
}

func (e *importStepError) Error() string { return e.err.Error() }
func (e *importStepError) Unwrap() error { return e.err }

// failureDetail is what a failed import tells the user about its cause: the step that failed (when it is known) and the
// reason, in words that name no server internals. Both are empty when nothing can be said without leaking them.
type failureDetail struct {
	step   string
	reason string
}

var (
	// a reason that talks about the connection to a service, a credential or a machine is replaced as a whole
	sensitiveReason = regexp.MustCompile(`(?i)password|passwd|secret|token|credential|authenticat|unauthori[sz]ed|dial tcp|no such host|connection (refused|reset)|connect:|ssl|tls|host=|user=|dbname=|sslmode|i/o timeout|\b\d{1,3}(\.\d{1,3}){3}\b`)
	// a path of the server's file system
	serverPath = regexp.MustCompile(`(/[A-Za-z0-9_.\-]+){2,}`)
	// the prefix the database driver puts before its message
	driverPrefix = regexp.MustCompile(`^(pq|ERROR):\s*`)
)

// maxReasonLength is the most characters of a reason that go into a message or a log
const maxReasonLength = 300

// describeFailure finds the step and the reason of a failed import. The reason is the message meant for the user when
// the error has one, a short sentence for a refusal of the database, and otherwise the text of the error, cleaned: the
// first line only, without a driver prefix or a server path, and replaced by a general sentence when it mentions a
// credential, a connection or a machine.
func describeFailure(err error) failureDetail {
	var d failureDetail
	var stepErr *importStepError
	if errors.As(err, &stepErr) {
		d.step = stepErr.step
	}
	var detailed *app_errors.DetailedError
	switch {
	case errors.As(err, &detailed) && strings.TrimSpace(detailed.Message) != "":
		d.reason = cleanReason(detailed.Message)
	case errors.Is(err, app_errors.DatabaseError):
		d.reason = "the database refused the operation"
	default:
		d.reason = cleanReason(err.Error())
	}
	// a panic is a bug of the server: its text is not for the user
	if strings.HasPrefix(err.Error(), "panic:") {
		d.reason = ""
	}
	return d
}

// cleanReason makes the text of an error fit for the user: see describeFailure
func cleanReason(text string) string {
	text = strings.TrimSpace(strings.SplitN(text, "\n", 2)[0])
	text = driverPrefix.ReplaceAllString(text, "")
	if sensitiveReason.MatchString(text) {
		return "a problem with a service the import depends on"
	}
	text = serverPath.ReplaceAllString(text, "<path>")
	if len(text) > maxReasonLength {
		text = text[:maxReasonLength] + "..."
	}
	return strings.TrimSpace(text)
}

// failedImportLog is the text of the log of an import that failed: the header, what went wrong, and the problems the
// file has when it was checked. Nothing was created, so there is no summary of rows.
func failedImportLog(format, fileName, message string, problems []string, detail failureDetail) string {
	var b strings.Builder
	b.WriteString("Import Issues Report\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	writeLogHeader(&b, format, fileName, "")
	b.WriteString("Result: failed (nothing was created)\n")
	b.WriteString(strings.Repeat("=", 100) + "\n\n")
	b.WriteString("ERROR:\n" + message + "\n")
	if detail.step != "" {
		b.WriteString("Step: " + detail.step + "\n")
	}
	if detail.reason != "" {
		b.WriteString("Reason: " + detail.reason + "\n")
	}
	b.WriteString("\n")
	if len(problems) > 0 {
		b.WriteString(fmt.Sprintf("PROBLEMS (%d):\n", len(problems)))
		listed := problems
		if len(listed) > maxLoggedProblems {
			listed = listed[:maxLoggedProblems]
		}
		for i, p := range listed {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, p))
		}
		if len(problems) > len(listed) {
			b.WriteString(fmt.Sprintf("... and %d more problems that are not listed.\n", len(problems)-len(listed)))
		}
	}
	return capLog(b.String())
}

// importLogFacts reads what a job's result says about its error log: whether there is one, how many rows were left
// out and how many warnings there are
func importLogFacts(result string) (hasLog bool, skippedRows, warnings int) {
	var r struct {
		ErrorLog    string   `json:"error_log"`
		SkippedRows int      `json:"skipped_rows"`
		Warnings    []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(result), &r); err != nil {
		return false, 0, 0
	}
	return strings.TrimSpace(r.ErrorLog) != "", r.SkippedRows, len(r.Warnings)
}
