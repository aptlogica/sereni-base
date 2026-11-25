// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"encoding/json"
	"strings"

	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

// importFormatResult is the result a new import job starts with: only the format of the import, so a job that is still
// running can say which format it is. The result of the import (or of its failure) takes its place and carries the
// format too.
func importFormatResult(format string) string {
	raw, err := json.Marshal(map[string]string{"format": firstNonEmpty(format, "sbase")})
	if err != nil {
		return ""
	}
	return string(raw)
}

// jobFormat is the file format of a job: sbase, csv, xlsx or json, or "" when it is not known. An export says it in the
// name of its file (the name the adapter chose); an import says it in its result.
func jobFormat(job tenant.TransferJob) string {
	if jobType(job) == tenant.TransferJobTypeImport {
		var r struct {
			Format string `json:"format"`
		}
		if json.Unmarshal([]byte(job.Result), &r) != nil {
			return ""
		}
		return r.Format
	}
	name := strings.ToLower(job.FileName)
	switch {
	case strings.HasSuffix(name, ".json.zip"):
		return "json"
	case strings.HasSuffix(name, ".csv.zip"):
		return "csv"
	case strings.HasSuffix(name, ".xlsx"), strings.HasSuffix(name, ".xlsx.zip"):
		return "xlsx"
	case strings.HasSuffix(name, ".sbase.zip"):
		return "sbase"
	}
	return ""
}

// withoutErrorLog is the result of a job without the text of its error log
func withoutErrorLog(result json.RawMessage) json.RawMessage {
	if len(result) == 0 {
		return result
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(result, &fields) != nil {
		return result
	}
	if _, has := fields["error_log"]; !has {
		return result
	}
	delete(fields, "error_log")
	raw, err := json.Marshal(fields)
	if err != nil {
		return result
	}
	return raw
}
