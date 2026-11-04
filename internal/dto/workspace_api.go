// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import "time"

// WorkspaceAPIResponse is the public API status of a workspace. PublicID, BasePath and EnabledAt are
// empty until the API is turned on for the first time.
type WorkspaceAPIResponse struct {
	WorkspaceID string `json:"workspace_id"`
	Enabled     bool   `json:"enabled"`
	// Permanent public ID, e.g. ws_8f3k2m9qx7
	PublicID string `json:"public_id,omitempty"`
	// Path of the workspace's public API, e.g. /api/v1/ws_8f3k2m9qx7
	BasePath  string     `json:"base_path,omitempty"`
	EnabledAt *time.Time `json:"enabled_at,omitempty"`
}
