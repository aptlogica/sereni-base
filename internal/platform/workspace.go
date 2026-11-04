// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package platform holds the workspace public API: each workspace's public ID and, later, the
// /api/v1/{ws} routes that let developers use a workspace's data from their own apps.
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
	publicIDPrefix   = "ws_"
	publicIDLength   = 10
	publicIDAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	// Largest multiple of the alphabet size that fits in a byte; bytes from here up are dropped so
	// every character is equally likely
	publicIDByteLimit = 252
	// Two workspaces drawing the same of 36^10 IDs is very unlikely, so a few attempts are plenty
	maxPublicIDAttempts = 5

	// Name Postgres gives the UNIQUE constraint on workspace_api.public_id
	publicIDConstraint = "workspace_api_public_id_key"
	// Postgres code for a row that references a missing row (here: the workspace)
	pqForeignKeyViolation = "23503"
)

// NewPublicID returns a random workspace public ID: "ws_" followed by 10 characters from 0-9a-z
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

// BasePath returns the path of a workspace's public API
func BasePath(publicID string) string {
	return "/api/v1/" + publicID
}

type workspaceAPIService struct {
	repo *pkg.DatabaseService
}

func NewWorkspaceAPIService(repo *pkg.DatabaseService) interfaces.WorkspaceAPIService {
	return &workspaceAPIService{repo: repo}
}

func (s *workspaceAPIService) Get(ctx context.Context, schemaName, workspaceID string) (dto.WorkspaceAPIResponse, error) {
	row, found, err := s.find(schemaName, workspaceID)
	if err != nil {
		return dto.WorkspaceAPIResponse{}, err
	}
	if !found {
		return dto.WorkspaceAPIResponse{WorkspaceID: workspaceID}, nil
	}
	return toResponse(row), nil
}

func (s *workspaceAPIService) Enable(ctx context.Context, schemaName, workspaceID string) (dto.WorkspaceAPIResponse, error) {
	row, found, err := s.find(schemaName, workspaceID)
	if err != nil {
		return dto.WorkspaceAPIResponse{}, err
	}
	if !found {
		if row, err = s.create(schemaName, workspaceID); err != nil {
			return dto.WorkspaceAPIResponse{}, err
		}
	}
	if !row.Enabled {
		// Turned off earlier: turn it back on and keep the public ID, so the address stays the same
		if row, err = s.turnOn(schemaName, workspaceID); err != nil {
			return dto.WorkspaceAPIResponse{}, err
		}
	}
	return toResponse(row), nil
}

func (s *workspaceAPIService) find(schemaName, workspaceID string) (tenant.WorkspaceAPI, bool, error) {
	limit := 1
	rows, err := s.repo.TableService.GetTableData(tenant.WorkspaceAPI{}.TableName(schemaName), dbModels.QueryParams{
		Select:  []string{"*"},
		Filters: []dbModels.QueryFilter{{Column: "workspace_id", Operator: "eq", Value: workspaceID}},
		Limit:   &limit,
	})
	if err != nil {
		return tenant.WorkspaceAPI{}, false, app_errors.LogDatabaseError(err, "failed to fetch workspace API")
	}
	if len(rows) == 0 {
		return tenant.WorkspaceAPI{}, false, nil
	}
	row, err := toWorkspaceAPI(rows[0])
	return row, err == nil, err
}

// create inserts the workspace's row, already enabled, drawing a new public ID when the first one is taken
func (s *workspaceAPIService) create(schemaName, workspaceID string) (tenant.WorkspaceAPI, error) {
	tableName := tenant.WorkspaceAPI{}.TableName(schemaName)
	for attempt := 0; attempt < maxPublicIDAttempts; attempt++ {
		publicID, err := NewPublicID()
		if err != nil {
			return tenant.WorkspaceAPI{}, err
		}
		created, err := s.repo.TableService.CreateRecord(tableName, map[string]interface{}{
			"workspace_id": workspaceID,
			"public_id":    publicID,
			"enabled":      true,
			"enabled_at":   time.Now().UTC(),
		})
		if err == nil {
			return toWorkspaceAPI(created)
		}
		if constraint, _, ok := app_errors.UniqueViolation(err); ok {
			if constraint == publicIDConstraint {
				continue
			}
			// The primary key: another request enabled this workspace at the same moment
			row, found, findErr := s.find(schemaName, workspaceID)
			if findErr != nil {
				return tenant.WorkspaceAPI{}, findErr
			}
			if found {
				return row, nil
			}
		}
		if dbErr, ok := app_errors.DBError(err); ok && dbErr.Code == pqForeignKeyViolation {
			return tenant.WorkspaceAPI{}, app_errors.WorkspaceNotFound
		}
		return tenant.WorkspaceAPI{}, app_errors.LogDatabaseError(err, "failed to create workspace API")
	}
	return tenant.WorkspaceAPI{}, app_errors.WorkspaceAPIPublicIDFailed
}

func (s *workspaceAPIService) turnOn(schemaName, workspaceID string) (tenant.WorkspaceAPI, error) {
	now := time.Now().UTC()
	// UpdateRecord matches on an id column, which this table doesn't have
	updated, err := s.repo.TableService.UpdateByColumns(tenant.WorkspaceAPI{}.TableName(schemaName),
		dbModels.ComplexFilter{
			Filters: []dbModels.QueryFilter{{Column: "workspace_id", Operator: "eq", Value: workspaceID}},
			Logic:   "AND",
		},
		map[string]any{"enabled": true, "enabled_at": now, "last_modified_time": now})
	if err != nil {
		return tenant.WorkspaceAPI{}, app_errors.LogDatabaseError(err, "failed to enable workspace API")
	}
	return toWorkspaceAPI(updated)
}

func toWorkspaceAPI(data map[string]interface{}) (tenant.WorkspaceAPI, error) {
	var row tenant.WorkspaceAPI
	if err := helpers.MapToStruct(data, &row); err != nil {
		return tenant.WorkspaceAPI{}, fmt.Errorf("failed to map workspace API data: %w", err)
	}
	return row, nil
}

func toResponse(row tenant.WorkspaceAPI) dto.WorkspaceAPIResponse {
	return dto.WorkspaceAPIResponse{
		WorkspaceID: row.WorkspaceID,
		Enabled:     row.Enabled,
		PublicID:    row.PublicID,
		BasePath:    BasePath(row.PublicID),
		EnabledAt:   row.EnabledAt,
	}
}
