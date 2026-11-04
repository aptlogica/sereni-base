// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Public ID prefixes. Bases, tables and fields are UUIDs without dashes; records are the row's integer id.
const (
	PrefixBase   = "bse_"
	PrefixTable  = "tbl_"
	PrefixField  = "fld_"
	PrefixRecord = "rec_"
)

// EncodeUUID returns prefix + the UUID without dashes, e.g. tbl_3f2a9c1e...; id is a dashed or plain UUID
func EncodeUUID(prefix, id string) string {
	if u, err := uuid.Parse(id); err == nil {
		return prefix + strings.ReplaceAll(u.String(), "-", "")
	}
	return prefix + strings.ReplaceAll(strings.ToLower(id), "-", "")
}

// DecodeUUID returns the dashed UUID in a prefixed ID. ok is false when value lacks the prefix or the
// rest is not 32 hex characters.
func DecodeUUID(prefix, value string) (string, bool) {
	rest, found := strings.CutPrefix(value, prefix)
	if !found || len(rest) != 32 {
		return "", false
	}
	u, err := uuid.Parse(rest)
	if err != nil {
		return "", false
	}
	return u.String(), true
}

// EncodeRecordID returns rec_<id>, e.g. rec_1042
func EncodeRecordID(id int64) string {
	return PrefixRecord + strconv.FormatInt(id, 10)
}

// DecodeRecordID returns the row id in rec_<id>. Leading zeros, signs and zero are rejected, so every
// record has exactly one ID.
func DecodeRecordID(value string) (int64, bool) {
	rest, found := strings.CutPrefix(value, PrefixRecord)
	if !found || rest == "" || rest[0] == '0' {
		return 0, false
	}
	for _, r := range rest {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// isUUIDID reports whether value starts with the prefix, i.e. must be resolved as an ID, not a name
func isUUIDID(prefix, value string) bool {
	return strings.HasPrefix(value, prefix)
}
