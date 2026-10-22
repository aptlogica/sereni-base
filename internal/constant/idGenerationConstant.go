// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constant

const (
	// MetaIDGenerationKey is the column meta key holding the uuid generation settings
	MetaIDGenerationKey = "idGeneration"

	// IDModeAuto means the system creates the value of every new record; nobody can write it
	IDModeAuto = "auto"
	// IDModeManual means people enter the value; it is only checked to be a valid UUID
	IDModeManual = "manual"

	// UIDTUUID is the uidt of uuid fields
	UIDTUUID = "uuid"

	// UUIDVersionDefault is the version of auto uuid fields that don't name one
	UUIDVersionDefault = "v7"

	// UUIDBackfillBatchSize is how many empty cells are filled per round trip
	UUIDBackfillBatchSize = 1000
)

// UUIDVersionsGeneratable are the versions the system can create
var UUIDVersionsGeneratable = []string{"v7", "v4", "v1"}
