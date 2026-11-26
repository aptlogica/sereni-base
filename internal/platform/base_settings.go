// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/lib/pq"
)

// Settings of a base's API live in base_api.settings (jsonb), one key per feature. A missing key means the
// feature is off, so no feature needs a backfill. Each feature has its own typed struct; unknown keys are
// rejected on write (see dto.UpdateBaseAPISettingsRequest) and ignored on read.
const (
	settingsKeyBaseManagement = "base_management"
	// featureBaseManagement is the name requireFeature takes
	featureBaseManagement = "base_management"
)

// baseManagementSettings is the base_management key of base_api.settings
type baseManagementSettings struct {
	Enabled   bool       `json:"enabled"`
	EnabledBy string     `json:"enabled_by,omitempty"`
	EnabledAt *time.Time `json:"enabled_at,omitempty"`
}

// baseSettings is every feature's settings of one base, with defaults for what is missing
type baseSettings struct {
	BaseManagement baseManagementSettings `json:"base_management"`
}

// parseBaseSettings reads base_api.settings. Unreadable settings count as "everything off".
func parseBaseSettings(raw []byte) baseSettings {
	var s baseSettings
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

func (s baseSettings) enabled(feature string) bool {
	switch feature {
	case featureBaseManagement:
		return s.BaseManagement.Enabled
	}
	return false
}

// UpdateSettings applies the settings in the request, one feature at a time and in one statement per feature
// (settings || jsonb_build_object(key, value)), so features saved together never overwrite each other.
func (s *baseAPIService) UpdateSettings(ctx context.Context, schemaName, baseID, userID string, req dto.UpdateBaseAPISettingsRequest) (dto.BaseAPIResponse, error) {
	row, found, err := s.find(schemaName, baseID)
	if err != nil {
		return dto.BaseAPIResponse{}, err
	}
	if !found || !row.Enabled {
		return dto.BaseAPIResponse{}, errBaseAPINotEnabled
	}
	if req.BaseManagement != nil && req.BaseManagement.Enabled != nil {
		value := baseManagementSettings{Enabled: *req.BaseManagement.Enabled}
		if value.Enabled {
			// The user who turns it on is the actor of every schema write made through the API
			now := time.Now().UTC()
			value.EnabledBy, value.EnabledAt = userID, &now
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return dto.BaseAPIResponse{}, err
		}
		if _, err := s.repo.DB.ExecContext(ctx, fmt.Sprintf(
			`UPDATE %s.base_api SET settings = settings || jsonb_build_object($1::text, $2::jsonb), last_modified_time = now()
			 WHERE base_id::text = $3`, pq.QuoteIdentifier(schemaName)), settingsKeyBaseManagement, string(encoded), baseID); err != nil {
			return dto.BaseAPIResponse{}, fmt.Errorf("update base API settings: %w", err)
		}
		// The per-base spec lists the write paths only while the feature is on: rebuild it on the next request
		if _, err := s.repo.DB.ExecContext(ctx, fmt.Sprintf(
			`UPDATE %s.base_api SET schema_version = schema_version + 1 WHERE base_id::text = $1`, pq.QuoteIdentifier(schemaName)), baseID); err != nil {
			return dto.BaseAPIResponse{}, fmt.Errorf("bump schema version: %w", err)
		}
	}
	return s.Get(ctx, schemaName, baseID)
}

// settings reads base_api.settings of a base
func (s *baseAPIService) settings(ctx context.Context, schemaName, baseID string) baseSettings {
	var raw []byte
	err := s.repo.DB.QueryRow(fmt.Sprintf(`SELECT settings::text FROM %s.base_api WHERE base_id::text = $1`,
		pq.QuoteIdentifier(schemaName)), baseID).Scan(&raw)
	if err != nil {
		return baseSettings{}
	}
	return parseBaseSettings(raw)
}

func toSettingsResponse(s baseSettings) *dto.BaseAPISettings {
	return &dto.BaseAPISettings{BaseManagement: dto.BaseManagementSettings{
		Enabled: s.BaseManagement.Enabled, EnabledBy: s.BaseManagement.EnabledBy, EnabledAt: s.BaseManagement.EnabledAt,
	}}
}

// errBaseAPINotEnabled: settings exist only once the base's API has been turned on
var errBaseAPINotEnabled = app_errors.InvalidPayload
