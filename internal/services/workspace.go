package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	dbModels "godbgrest/pkg/models"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"strings"
	"time"

	app_errors "serenibase/internal/app-errors"

	"github.com/google/uuid"
)

type workspaceService struct {
	repo *pkg.DatabaseService
}

func NewWorkspaceService(repo *pkg.DatabaseService) interfaces.WorkspaceService {
	return &workspaceService{repo: repo}
}

func (s *workspaceService) createSlug(title string) string {
	title = strings.ToLower(strings.ReplaceAll(title, " ", "_"))
	return fmt.Sprintf("%s-workspace-%s", title, uuid.NewString()[0:8])
}

// WorkspaceInsertion implements interfaces.WorkspaceService.
func (s *workspaceService) WorkspaceInsertion(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string) (tenant.Workspace, error) {
	workspaceData := dto.WorkspaceInsertion{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description,
		Slug:        s.createSlug(req.Title),
		Meta:        map[string]interface{}{},
		IsDefault:   true,
		CreatedBy:   req.CreatedBy,
		UpdatedBy:   req.CreatedBy,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	tableName := tenant.Workspace{}.TableName(schemaName)
	s.ensureAuditColumns(ctx, schemaName)

	insertedData, err := s.repo.TableService.CreateRecord(ctx, tableName, workspaceData.Map())
	if err != nil {
		fmt.Println("insertedData: ", err)
		return tenant.Workspace{}, app_errors.DatabaseError
	}

	var insertedWorkspace tenant.Workspace

	if err := helpers.MapToStruct(insertedData, &insertedWorkspace); err != nil {
		return tenant.Workspace{}, app_errors.ErrMapToStruct
	}

	return insertedWorkspace, nil
}

func (s *workspaceService) ensureAuditColumns(ctx context.Context, schemaName string) {
	tableName := fmt.Sprintf("\"%s\".\"workspaces\"", schemaName)
	columns := []string{"created_by", "last_modified_by"}
	for _, col := range columns {
		req := dbModels.AddColumnRequest{
			Column: dbModels.ColumnDefinition{
				Name:     col,
				DataType: "varchar",
			},
		}
		if err := s.repo.TableService.AddColumn(tableName, req); err != nil {
			fmt.Printf("DEBUG: Failed to add column %s to %s: %v\n", col, tableName, err)
		}
	}
}

func (s *workspaceService) CreateWorkspace(ctx context.Context, schemaName string) (tenant.Workspace, error) {
	if s.repo == nil || s.repo.TableService == nil {
		return tenant.Workspace{}, fmt.Errorf("repository not initialized")
	}

	workspace := tenant.Workspace{}

	if err := s.repo.TableService.CreateTable(workspace.TableSchema(schemaName)); err != nil {
		return tenant.Workspace{}, err
	}
	
	// Ensure columns exist (for existing tables)
	s.ensureAuditColumns(ctx, schemaName)

	return workspace, nil
}

func (s *workspaceService) GetWorkspaceByID(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
	if id == "" {
		return tenant.Workspace{}, fmt.Errorf("workspace ID cannot be empty")
	}
	workspace := tenant.Workspace{}
	tableName := workspace.TableName(schemaName)

	limit := 1
	query := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    id,
			},
		},
		Limit: &limit,
	}

	// Fetch workspace row(s)
	workspacesData, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return tenant.Workspace{}, fmt.Errorf("failed to fetch workspace: %w", err)
	}

	if workspacesData == nil || len(workspacesData) == 0 {
		return tenant.Workspace{}, fmt.Errorf("workspace not found with id: %s", id)
	}

	workspaceData := workspacesData[0]

	var ws tenant.Workspace
	if err := helpers.MapToStruct(workspaceData, &ws); err != nil {
		return tenant.Workspace{}, fmt.Errorf("failed to map workspace data: %w", err)
	}

	return ws, nil
}

func (s *workspaceService) GetAllWorkspaces(ctx context.Context, schemaName string) ([]tenant.Workspace, error) {
	workspace := tenant.Workspace{}
	tableName := workspace.TableName(schemaName)
	params := dbModels.QueryParams{
		Select: []string{"*"},
	}

	rows, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workspaces: %w", err)
	}
	if len(rows) == 0 {
		return []tenant.Workspace{}, nil
	}

	workspaces := make([]tenant.Workspace, 0, len(rows))
	for _, row := range rows {
		var ws tenant.Workspace
		if err := helpers.MapToStruct(row, &ws); err != nil {
			return nil, fmt.Errorf("failed to map workspace data: %w", err)
		}
		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

func (s *workspaceService) UpdateWorkspace(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate) (tenant.Workspace, error) {
	if id == "" {
		return tenant.Workspace{}, fmt.Errorf("workspace ID cannot be empty")
	}
	workspace := tenant.Workspace{}
	tableName := workspace.TableName(schemaName)

	// Check if workspace exists
	existingWorkspace, err := s.GetWorkspaceByID(ctx, schemaName, id)
	if err != nil {
		return tenant.Workspace{}, fmt.Errorf("workspace not found: %w", err)
	}

	// Prepare update data
	updateData := req.Map()
	if len(updateData) == 0 {
		return existingWorkspace, nil // Nothing to update
	}
	updateData["last_modified_time"] = time.Now().UTC()

	fmt.Println("updateData: ", updateData)
	// Perform update
	updatedRows, err := s.repo.TableService.UpdateRecord(ctx, tableName, id, updateData)
	if err != nil {
		fmt.Println("err: ", err)
		return tenant.Workspace{}, app_errors.DatabaseError
	}
	if updatedRows == nil || len(updatedRows) == 0 {
		return tenant.Workspace{}, app_errors.InvalidPayload
	}

	// Return updated workspace
	return s.GetWorkspaceByID(ctx, schemaName, id)
}

func (s *workspaceService) DeleteWorkspace(ctx context.Context, schemaName string, id string) error {
	if id == "" {
		return fmt.Errorf("workspace ID cannot be empty")
	}
	workspace := tenant.Workspace{}
	tableName := workspace.TableName(schemaName)

	// Check if workspace exists
	_, err := s.GetWorkspaceByID(ctx, schemaName, id)
	if err != nil {
		return fmt.Errorf("workspace not found: %w", err)
	}

	// Perform deletion
	if err := s.repo.TableService.DeleteRecord(ctx, tableName, id); err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	return nil
}

func (s *workspaceService) GetBulkWorkspaces(ctx context.Context, schemaName string, ids []string) ([]tenant.Workspace, error) {
	if len(ids) == 0 {
		return []tenant.Workspace{}, nil
	}

	tableName := tenant.Workspace{}.TableName(schemaName)

	filters := []dbModels.QueryFilter{
		{
			Column:   "id",
			Operator: "in",
			Value:    ids,
		},
	}

	params := dbModels.QueryParams{
		Select:  []string{"*"},
		Filters: filters,
	}

	rows, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
	}
	if len(rows) == 0 {
		return []tenant.Workspace{}, nil
	}

	workspaces := make([]tenant.Workspace, 0, len(rows))
	for _, row := range rows {
		var workspace tenant.Workspace
		if err := helpers.MapToStruct(row, &workspace); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		workspaces = append(workspaces, workspace)
	}

	return workspaces, nil
}
