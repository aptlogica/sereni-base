// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constant

const (
	// UIDTLinks is the uidt of link fields
	UIDTLinks = "links"

	// PrimaryKeyIDColumn is the system id column; rows are addressed by it even when it is not in the key
	PrimaryKeyIDColumn = "id"
)

// PrimaryKeyEligibleUIDTs are the field types that hold one short value and may be primary key fields
var PrimaryKeyEligibleUIDTs = []string{
	"text", "number", "decimal", "currency", "percent", "duration", "year", "rating",
	"date", "datetime", "time", "email", "phoneNumber", "url", "select", "boolean", "uuid",
}

// PrimaryKeyUniqueStartUIDTs are the key field types whose starting value in a new row is always unique
// (see key_fill_defaults): text-stored types get <column_name>_<timestamp>_<id>, numbers the row id, uuids a
// random uuid
var PrimaryKeyUniqueStartUIDTs = []string{
	"text", "email", "phoneNumber", "url", "select", "duration", "time",
	"number", "decimal", "currency", "percent", "year", "rating", "uuid",
}

// LinkIsMulti reports whether the link field on this side of a relation holds many linked records
func LinkIsMulti(entityRole, relationType string) bool {
	switch relationType {
	case "many-to-many":
		return true
	case "has-many":
		return entityRole == "source"
	}
	return false
}
