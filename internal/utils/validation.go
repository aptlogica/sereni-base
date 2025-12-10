package utils

import (
	"regexp"
)

var (
	// SQL injection prevention patterns
	sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select)`),
		regexp.MustCompile(`(?i)(drop\s+table)`),
		regexp.MustCompile(`(?i)(delete\s+from)`),
		regexp.MustCompile(`(?i)(insert\s+into)`),
		regexp.MustCompile(`(?i)(update\s+\w+\s+set)`),
		regexp.MustCompile(`(?i)(exec\s*\()`),
		regexp.MustCompile(`(?i)(script\s*:)`),
		regexp.MustCompile(`(\-\-)`),
		regexp.MustCompile(`(\/\*)`),
	}

	// Valid PostgreSQL identifier pattern
	identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]`)
)
