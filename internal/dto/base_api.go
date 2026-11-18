// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import "time"

// BaseAPIResponse is the public API status of a base. PublicID, BasePath and EnabledAt are
// empty until the API is turned on for the first time.
type BaseAPIResponse struct {
	BaseID  string `json:"base_id"`
	Enabled bool   `json:"enabled"`
	// Permanent public ID, e.g. bs_8f3k2m9qx7
	PublicID string `json:"public_id,omitempty"`
	// Path of the base's public API, e.g. /api/v1/bs_8f3k2m9qx7
	BasePath  string     `json:"base_path,omitempty"`
	EnabledAt *time.Time `json:"enabled_at,omitempty"`
}
