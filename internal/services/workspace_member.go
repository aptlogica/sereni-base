package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	dbModels "godbgrest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/models/tenant"
	"serenibase/internal/providers/logger"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"time"
)

type workspaceMemberService struct {
	repo *pkg.DatabaseService
}

func NewWorkspaceMemberService(
	repo *pkg.DatabaseService,
) interfaces.WorkspaceMemberService {
	return &workspaceMemberService{
		repo: repo,
	}
}

// GetAllWorkspaceMembers returns all workspace members for the given schema
func (s *workspaceMemberService) GetAllWorkspaceMembersByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error) {
	tableName := tenant.WorkspaceMember{}.TableName(schemaName)

	params := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userId,
			},
		},
	}

	rows, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
	}
	if len(rows) == 0 {
		return []tenant.WorkspaceMember{}, app_errors.WorkspaceMemberNotFound
	}

	members := make([]tenant.WorkspaceMember, 0, len(rows))
	for _, row := range rows {
		var member tenant.WorkspaceMember
		if err := helpers.MapToStruct(row, &member); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		members = append(members, member)
	}

	return members, nil
}

func (s *workspaceMemberService) GetWorkspaceMemberByUserAndWorkspace(ctx context.Context, schemaName string, userId string, workspaceId string) (*tenant.WorkspaceMember, error) {
	tableName := tenant.WorkspaceMember{}.TableName(schemaName)

	params := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userId,
			},
			{
				Column:   "workspace_id",
				Operator: "eq",
				Value:    workspaceId,
			},
		},
	}

	rows, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
	}
	if len(rows) == 0 {
		return nil, app_errors.WorkspaceMemberNotFound
	}

	var member tenant.WorkspaceMember
	if err := helpers.MapToStruct(rows[0], &member); err != nil {
		return nil, app_errors.ErrMapToStruct
	}
	return &member, nil
}

// DeleteWorkspaceMember deletes a workspace member by ID in the given schema
func (s *workspaceMemberService) DeleteWorkspaceMember(ctx context.Context, schemaName string, id string) error {
	tableName := tenant.WorkspaceMember{}.TableName(schemaName)

	if err := s.repo.TableService.DeleteRecord(ctx, tableName, id); err != nil {
		return fmt.Errorf("failed to delete workspace member: %w", err)
	}

	return nil
}

// GetWorkspaceMemberByUser retrieves all workspace memberships for a specific user in a schema.
func (s *workspaceMemberService) GetWorkspaceMemberByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error) {
	tableName := tenant.WorkspaceMember{}.TableName(schemaName)

	params := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userId,
			},
		},
	}

	rows, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
	}
	if len(rows) == 0 {
		return nil, app_errors.WorkspaceMemberNotFound
	}

	members := make([]tenant.WorkspaceMember, 0, len(rows))
	for _, row := range rows {
		var member tenant.WorkspaceMember
		if err := helpers.MapToStruct(row, &member); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		members = append(members, member)
	}

	return members, nil
}

// GetWorkspaceMembersByWorkspace retrieves all workspace members for a specific workspace in a schema.
func (s *workspaceMemberService) GetWorkspaceMembersByWorkspace(ctx context.Context, schemaName string, workspaceId string) ([]tenant.WorkspaceMember, error) {
	lg := logger.Get()
	tableName := tenant.WorkspaceMember{}.TableName(schemaName)

	params := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "workspace_id",
				Operator: "eq",
				Value:    workspaceId,
			},
		},
	}

	rows, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
	}
	lg.Debug().Interface("rows", rows).Msg("Retrieved workspace members data")

	if len(rows) == 0 {
		return nil, app_errors.WorkspaceMemberNotFound
	}

	members := make([]tenant.WorkspaceMember, 0, len(rows))
	for _, row := range rows {
		var member tenant.WorkspaceMember
		if err := helpers.MapToStruct(row, &member); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		members = append(members, member)
	}
	lg.Debug().Interface("members", members).Msg("Retrieved members for workspace")

	return members, nil
}

// DeleteUserMappings removes all workspace member mappings for a given user across all workspaces in a schema.
func (s *workspaceMemberService) DeleteUserMappings(ctx context.Context, schemaName string, userId string) error {
	// First, get all workspace members for this user using the existing function
	members, err := s.GetAllWorkspaceMembersByUser(ctx, schemaName, userId)
	if err != nil {
		return err
	}
	tableName := tenant.WorkspaceMember{}.TableName(schemaName)

	for _, member := range members {
		if err := s.repo.TableService.DeleteRecord(ctx, tableName, member.ID); err != nil {
			return app_errors.DatabaseError
		}
	}

	return nil
}

// UpdateWorkspaceMemberBases updates the bases_ids and access_level for an existing workspace member.
// For limited_access, it replaces the bases_ids with the new ones provided.
// For full_access, it sets bases_ids to "*".
func (s *workspaceMemberService) UpdateWorkspaceMemberBases(ctx context.Context, schemaName string, workspaceId string, userId string, accessLevel string, basesIds string) error {
	lg := logger.Get()
	// Get existing workspace member
	existingMember, err := s.GetWorkspaceMemberByUserAndWorkspace(ctx, schemaName, userId, workspaceId)
	if err != nil {
		return err
	}

	tableName := tenant.WorkspaceMember{}.TableName(schemaName)

	// Prepare update data
	updateData := map[string]interface{}{
		"access_level":       accessLevel,
		"last_modified_time": time.Now(),
	}

	// Handle bases_ids based on access level
	if accessLevel == "full_access" {
		// Full access gets all bases
		updateData["bases_ids"] = "*"
	} else {
		// Limited access: replace with new base IDs (no merging)
		updateData["bases_ids"] = basesIds
	}

	// Update the record - convert UUID to string
	recordID := existingMember.ID.String()
	lg.Debug().Str("id", recordID).Interface("updateData", updateData).Msg("Updating workspace member bases")
	_, err = s.repo.TableService.UpdateRecord(ctx, tableName, recordID, updateData)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to update workspace member bases")
		return app_errors.DatabaseError
	}

	lg.Info().Str("id", recordID).Msg("Successfully updated workspace member")
	return nil
}
