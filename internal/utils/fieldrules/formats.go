// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package fieldrules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// The checks below are the app's field inputs' checks (base-ui src/components/common/Fields: Email.tsx,
// PhoneNumber.tsx, URL.tsx, Time.tsx), so both sides agree on what is valid. Keep them in step.

var (
	emailLocalPattern   = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+$`)
	emailSegmentPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	phonePattern        = regexp.MustCompile(`^\+?[1-9][0-9]{0,15}$`)
	time24Pattern       = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	time12Pattern       = regexp.MustCompile(`^(0?[1-9]|1[0-2]):([0-5][0-9])\s*([AaPp][Mm])$`)
)

// ValidEmail: one @, a local part of letters, digits and ._%+- (up to 64), and a domain of at least two
// dot-separated parts of letters, digits and - (up to 255), the last at least 2 long, not starting with -
func ValidEmail(email string) bool {
	if email == "" || len(email) > 320 {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local, domain := parts[0], parts[1]
	if local == "" || len(local) > 64 || !emailLocalPattern.MatchString(local) {
		return false
	}
	if domain == "" || len(domain) > 255 {
		return false
	}
	segments := strings.Split(domain, ".")
	if len(segments) < 2 {
		return false
	}
	for i, s := range segments {
		if !emailSegmentPattern.MatchString(s) {
			return false
		}
		if i == len(segments)-1 && (len(s) < 2 || strings.HasPrefix(s, "-")) {
			return false
		}
	}
	return true
}

// ValidPhone: after removing spaces, dashes and brackets, an optional + and 1 to 16 digits not starting with 0
func ValidPhone(phone string) bool {
	stripped := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '\f', '\v', '-', '(', ')':
			return -1
		}
		return r
	}, phone)
	return phonePattern.MatchString(stripped)
}

// ValidURL: an optional http:// or https://, a host whose first part is lowercase letters, digits, - and .,
// a top-level part of 2 to 6 lowercase letters, and an optional path of letters, digits, spaces and /.-_
func ValidURL(url string) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return true
	}
	// Only one protocol is removed, as in the app
	rest := url
	if strings.HasPrefix(url, "https://") {
		rest = url[len("https://"):]
	} else if strings.HasPrefix(url, "http://") {
		rest = url[len("http://"):]
	}
	parts := strings.Split(rest, ".")
	if len(parts) < 2 {
		return false
	}
	if !allChars(parts[0], isDomainChar) || parts[0] == "" {
		return false
	}
	last := parts[len(parts)-1]
	tld := last
	if i := strings.IndexByte(last, '/'); i >= 0 {
		tld = last[:i]
	}
	if len(tld) < 2 || len(tld) > 6 || !allChars(tld, isTLDChar) {
		return false
	}
	if i := strings.IndexByte(rest, '/'); i >= 0 && !allChars(rest[i+1:], isPathChar) {
		return false
	}
	return true
}

// ParseTime reads a time as the app stores it, HH:MM (24-hour), from 24-hour input or from 12-hour input
// like "2:15 PM"
func ParseTime(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if time24Pattern.MatchString(s) {
		return s, true
	}
	m := time12Pattern.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	hour, _ := strconv.Atoi(m[1])
	pm := strings.EqualFold(m[3], "pm")
	if pm && hour != 12 {
		hour += 12
	}
	if !pm && hour == 12 {
		hour = 0
	}
	return fmt.Sprintf("%02d:%s", hour, m[2]), true
}

func allChars(s string, ok func(rune) bool) bool {
	for _, r := range s {
		if !ok(r) {
			return false
		}
	}
	return true
}

func isDomainChar(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || r == '-' || r == '.'
}

func isTLDChar(r rune) bool { return (r >= 'a' && r <= 'z') || r == '.' }

func isPathChar(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		r == '/' || r == ' ' || r == '.' || r == '-' || r == '_'
}
