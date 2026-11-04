// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

// APIRecordUpdate is one record of a public API update: the record's id and its new values by column name
type APIRecordUpdate struct {
	RowID  int64
	Values map[string]interface{}
}
