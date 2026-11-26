// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package platform holds the base public API: each base's public ID and the
// /api/v1/{bs} routes that let developers use a base's data from their own apps.
package platform

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
)

const (
	publicIDPrefix   = "bs_"
	publicIDLength   = 10
	publicIDAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	// Largest multiple of the alphabet size that fits in a byte; bytes from here up are dropped so
	// every character is equally likely
	publicIDByteLimit = 252
	// Two bases drawing the same of 36^10 IDs is very unlikely, so a few attempts are plenty
	maxPublicIDAttempts = 5

	// Name Postgres gives the UNIQUE constraint on base_api.public_id
	publicIDConstraint = "base_api_public_id_key"
	// Postgres code for a row that references a missing row (here: the base)
	pqForeignKeyViolation = "23503"
)

// NewPublicID returns a random base public ID: "bs_" followed by 10 characters from 0-9a-z
func NewPublicID() (string, error) {
	id := make([]byte, 0, publicIDLength)
	buf := make([]byte, publicIDLength*2)
	for len(id) < publicIDLength {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to read random bytes: %w", err)
		}
		for _, b := range buf {
			if b >= publicIDByteLimit {
				continue
			}
			id = append(id, publicIDAlphabet[int(b)%len(publicIDAlphabet)])
			if len(id) == publicIDLength {
				break
			}
		}
	}
	return publicIDPrefix + string(id), nil
}

// BasePath returns the path of a base's public API
func BasePath(publicID string) string {
	return "/api/v1/" + publicID
}

type baseAPIService struct {
	repo *pkg.DatabaseService
}

func NewBaseAPIService(repo *pkg.DatabaseService) interfaces.BaseAPIService {
	return &baseAPIService{repo: repo}
}

func (s *baseAPIService) Get(ctx context.Context, schemaName, baseID string) (dto.BaseAPIResponse, error) {
	row, found, err := s.find(schemaName, baseID)
	if err != nil {
		return dto.BaseAPIResponse{}, err
	}
	if !found {
		return dto.BaseAPIResponse{BaseID: baseID}, nil
	}
	out := toResponse(row)
	out.Settings = toSettingsResponse(s.settings(ctx, schemaName, baseID))
	return out, nil
}

func (s *baseAPIService) Enable(ctx context.Context, schemaName, baseID string) (dto.BaseAPIResponse, error) {
	row, found, err := s.find(schemaName, baseID)
	if err != nil {
		return dto.BaseAPIResponse{}, err
	}
	if !found {
		if row, err = s.create(schemaName, baseID); err != nil {
			return dto.BaseAPIResponse{}, err
		}
	}
	if !row.Enabled {
		// Turned off earlier: turn it back on and keep the public ID, so the address stays the same
		if row, err = s.turnOn(schemaName, baseID); err != nil {
			return dto.BaseAPIResponse{}, err
		}
	}
	return toResponse(row), nil
}

func (s *baseAPIService) find(schemaName, baseID string) (tenant.BaseAPI, bool, error) {
	limit := 1
	rows, err := s.repo.TableService.GetTableData(tenant.BaseAPI{}.TableName(schemaName), dbModels.QueryParams{
		Select:  []string{"*"},
		Filters: []dbModels.QueryFilter{{Column: "base_id", Operator: "eq", Value: baseID}},
		Limit:   &limit,
	})
	if err != nil {
		return tenant.BaseAPI{}, false, app_errors.LogDatabaseError(err, "failed to fetch base API")
	}
	if len(rows) == 0 {
		return tenant.BaseAPI{}, false, nil
	}
	row, err := toBaseAPI(rows[0])
	return row, err == nil, err
}

// create inserts the base's row, already enabled, drawing a new public ID when the first one is taken
func (s *baseAPIService) create(schemaName, baseID string) (tenant.BaseAPI, error) {
	tableName := tenant.BaseAPI{}.TableName(schemaName)
	for attempt := 0; attempt < maxPublicIDAttempts; attempt++ {
		publicID, err := NewPublicID()
		if err != nil {
			return tenant.BaseAPI{}, err
		}
		created, err := s.repo.TableService.CreateRecord(tableName, map[string]interface{}{
			"base_id":    baseID,
			"public_id":  publicID,
			"enabled":    true,
			"enabled_at": time.Now().UTC(),
		})
		if err == nil {
			return toBaseAPI(created)
		}
		if constraint, _, ok := app_errors.UniqueViolation(err); ok {
			if constraint == publicIDConstraint {
				continue
			}
			// The primary key: another request enabled this base at the same moment
			row, found, findErr := s.find(schemaName, baseID)
			if findErr != nil {
				return tenant.BaseAPI{}, findErr
			}
			if found {
				return row, nil
			}
		}
		if dbErr, ok := app_errors.DBError(err); ok && dbErr.Code == pqForeignKeyViolation {
			return tenant.BaseAPI{}, app_errors.BaseNotFound
		}
		return tenant.BaseAPI{}, app_errors.LogDatabaseError(err, "failed to create base API")
	}
	return tenant.BaseAPI{}, app_errors.BaseAPIPublicIDFailed
}

func (s *baseAPIService) turnOn(schemaName, baseID string) (tenant.BaseAPI, error) {
	now := time.Now().UTC()
	// UpdateRecord matches on an id column, which this table doesn't have
	updated, err := s.repo.TableService.UpdateByColumns(tenant.BaseAPI{}.TableName(schemaName),
		dbModels.ComplexFilter{
			Filters: []dbModels.QueryFilter{{Column: "base_id", Operator: "eq", Value: baseID}},
			Logic:   "AND",
		},
		map[string]any{"enabled": true, "enabled_at": now, "last_modified_time": now})
	if err != nil {
		return tenant.BaseAPI{}, app_errors.LogDatabaseError(err, "failed to enable base API")
	}
	return toBaseAPI(updated)
}

func toBaseAPI(data map[string]interface{}) (tenant.BaseAPI, error) {
	var row tenant.BaseAPI
	if err := helpers.MapToStruct(data, &row); err != nil {
		return tenant.BaseAPI{}, fmt.Errorf("failed to map base API data: %w", err)
	}
	return row, nil
}

func toResponse(row tenant.BaseAPI) dto.BaseAPIResponse {
	return dto.BaseAPIResponse{
		BaseID:    row.BaseID,
		Enabled:   row.Enabled,
		PublicID:  row.PublicID,
		BasePath:  BasePath(row.PublicID),
		EnabledAt: row.EnabledAt,
	}
}
