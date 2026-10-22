// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"crypto/rand"
	"fmt"
	"strings"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"

	"github.com/google/uuid"
)

// v1 UUIDs carry a node id, which defaults to the machine's network (MAC) address. A random node keeps that
// address out of every ID; the multicast bit marks the node as random (RFC 9562).
func init() {
	node := make([]byte, 6)
	if _, err := rand.Read(node); err != nil {
		return
	}
	node[0] |= 0x01
	uuid.SetNodeID(node)
}

// NewUUID creates one value for an auto uuid field. Values are made here, not by the database, so no
// extension or column default is needed.
func NewUUID(version string) (uuid.UUID, error) {
	switch version {
	case "v7":
		return uuid.NewV7()
	case "v4":
		return uuid.NewRandom()
	case "v1":
		return uuid.NewUUID()
	}
	return uuid.Nil, app_errors.UUIDConfigInvalid
}

// UUIDCellValue checks a value written to a manual uuid field and returns it in canonical form
// (lowercase 8-4-4-4-12). Any version is accepted; nil and max UUIDs are not.
func UUIDCellValue(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprint(value)
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return nil, nil
	}
	u, err := uuid.Parse(str)
	if err != nil || u == uuid.Nil || u == uuid.Max {
		return nil, app_errors.UUIDInvalid
	}
	return u.String(), nil
}
