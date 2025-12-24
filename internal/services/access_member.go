package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	dbModels "godbgrest/pkg/models"

	"github.com/google/uuid"
)

type accessMemberService struct {
	repo *pkg.DatabaseService
}

func NewAccessMemberService(repo *pkg.DatabaseService) interfaces.AccessMemberService {
	return &accessMemberService{repo: repo}
}

func (s *accessMemberService) AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := fmt.Sprintf("\"%s\".access_members", schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(ctx, tableName, req.Map())
	if err != nil {
		return nil, err
	}

	var accessMember tenant.AccessMember
	if err := helpers.MapToStruct(insertedData, &accessMember); err != nil {
		return nil, err
	}
	return accessMember, nil
}

func (s *accessMemberService) RemoveRoleFromUser(ctx context.Context, schemaName string, userID, scopeID string, scopeType string) error {
	if userID == "" {
		return app_errors.ErrRecordNotFound
	}
	if scopeType == "" {
		return app_errors.InvalidScopeType
	}

	tableName := fmt.Sprintf("\"%s\".access_members", schemaName)

	// Delete all access members for user in this scope
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
			{
				Column:   "scope_type",
				Operator: "eq",
				Value:    scopeType,
			},
		},
	}

	if scopeID != "" {
		query.Filters = append(query.Filters, dbModels.QueryFilter{
			Column:   "scope_id",
			Operator: "eq",
			Value:    scopeID,
		})
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return app_errors.DatabaseError
	}

	if len(data) == 0 {
		return app_errors.AccessMemberNotFound
	}

	// Delete each access member record
	for _, item := range data {
		var am tenant.AccessMember
		if err := helpers.MapToStruct(item, &am); err != nil {
			return app_errors.ErrMapToStruct
		}

		filter := dbModels.QueryFilter{
			Column:   "id",
			Operator: "eq",
			Value:    am.ID.String(),
		}
		deleteErr := s.repo.TableService.DeleteRecord(ctx, tableName, filter)
		if deleteErr != nil {
			return app_errors.AccessMemberDeleteFailed
		}
	}

	return nil
}

func (s *accessMemberService) GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
	tableName := fmt.Sprintf("\"%s\".access_members", schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
		},
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var members []dto.AccessMemberDTO
	for _, item := range data {
		var member dto.AccessMemberDTO
		if err := helpers.MapToStruct(item, &member); err != nil {
			continue
		}
		members = append(members, member)
	}
	return members, nil
}

func (s *accessMemberService) GetUserAccessByScope(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	if userID == "" {
		return nil, app_errors.ErrRecordNotFound
	}
	if scopeType == "" {
		return nil, app_errors.InvalidScopeType
	}

	tableName := fmt.Sprintf("\"%s\".access_members", schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
			{
				Column:   "scope_type",
				Operator: "eq",
				Value:    scopeType,
			},
		},
	}

	if scopeID != nil && *scopeID != "" {
		query.Filters = append(query.Filters, dbModels.QueryFilter{
			Column:   "scope_id",
			Operator: "eq",
			Value:    *scopeID,
		})
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var members []dto.AccessMemberDTO
	for _, item := range data {
		var member dto.AccessMemberDTO
		if err := helpers.MapToStruct(item, &member); err != nil {
			continue
		}
		members = append(members, member)
	}
	return members, nil
}

func (s *accessMemberService) GetScopeMembers(ctx context.Context, schemaName string, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	tableName := fmt.Sprintf("\"%s\".access_members", schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "scope_type",
				Operator: "eq",
				Value:    scopeType,
			},
		},
	}

	if scopeID != nil && *scopeID != "" {
		query.Filters = append(query.Filters, dbModels.QueryFilter{
			Column:   "scope_id",
			Operator: "eq",
			Value:    *scopeID,
		})
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var members []dto.AccessMemberDTO
	for _, item := range data {
		var member dto.AccessMemberDTO
		if err := helpers.MapToStruct(item, &member); err != nil {
			continue
		}
		members = append(members, member)
	}
	return members, nil
}

func (s *accessMemberService) GetUserPermissions(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.PermissionWithDetails, error) {
	// Get user's roles for this scope
	accessMembers, err := s.GetUserAccessByScope(ctx, schemaName, userID, scopeType, scopeID)
	if err != nil {
		return nil, err
	}

	if len(accessMembers) == 0 {
		return []dto.PermissionWithDetails{}, nil
	}

	// Collect all unique permissions from all roles
	permissionMap := make(map[string]dto.PermissionWithDetails)
	rolePermSvc := NewRolePermissionService(s.repo)

	for _, member := range accessMembers {
		roleID, err := uuid.Parse(member.RoleID)
		if err != nil {
			continue
		}

		permissions, err := rolePermSvc.GetPermissionsByRole(ctx, schemaName, roleID)
		if err != nil {
			continue
		}

		for _, perm := range permissions {
			permissionMap[perm.ID.String()] = perm
		}
	}

	var result []dto.PermissionWithDetails
	for _, perm := range permissionMap {
		result = append(result, perm)
	}
	return result, nil
}

func (s *accessMemberService) CheckUserPermission(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, resourceCode, actionCode string) (bool, error) {
	// Get user's permissions for this scope
	permissions, err := s.GetUserPermissions(ctx, schemaName, userID, scopeType, scopeID)
	if err != nil {
		return false, err
	}

	// Check if any permission matches
	for _, perm := range permissions {
		if perm.ResourceCode == resourceCode && perm.ActionCode == actionCode {
			return true, nil
		}
	}

	return false, nil
}

func (s *accessMemberService) GetUserHighestRole(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) (*dto.AccessRoleDTO, error) {
	if userID == "" {
		return nil, app_errors.ErrRecordNotFound
	}
	if scopeType == "" {
		return nil, app_errors.InvalidScopeType
	}

	// Get all user roles for this scope
	accessMembers, err := s.GetUserAccessByScope(ctx, schemaName, userID, scopeType, scopeID)
	if err != nil {
		return nil, err
	}

	if len(accessMembers) == 0 {
		return nil, app_errors.AccessMemberNotFound
	}

	// Find the role with highest priority
	var highestRole *dto.AccessRoleDTO
	var highestPriority int = -1
	roleSvc := NewAccessRoleService(s.repo)

	for _, member := range accessMembers {
		roleID, err := uuid.Parse(member.RoleID)
		if err != nil {
			return nil, app_errors.ErrRecordNotFound
		}

		role, err := roleSvc.GetAccessRoleByID(ctx, schemaName, roleID)
		if err != nil {
			return nil, app_errors.RoleNotFound
		}

		if role.Priority > highestPriority {
			highestPriority = role.Priority
			highestRole = &dto.AccessRoleDTO{
				ID:          role.ID,
				Name:        role.Name,
				ScopeLevel:  role.ScopeLevel,
				Priority:    role.Priority,
				Description: role.Description,
				IsDefault:   role.IsDefault,
				CreatedAt:   role.CreatedAt,
				UpdatedAt:   role.UpdatedAt,
			}
		}
	}

	if highestRole == nil {
		return nil, app_errors.RoleNotFound
	}

	return highestRole, nil
}

func (s *accessMemberService) BulkAssignRoleToUsers(ctx context.Context, schemaName string, req dto.BulkAssignRoleRequest) error {
	if len(req.UserIDs) == 0 {
		return app_errors.EmptyUserList
	}

	var assignmentErrors []error
	for _, userID := range req.UserIDs {
		if userID == "" {
			continue
		}

		assignReq := dto.AccessMemberDTO{
			ID:         uuid.New(),
			UserID:     userID,
			ScopeType:  req.ScopeType,
			ScopeID:    req.ScopeID,
			RoleID:     req.RoleID,
			AssignedBy: req.AssignedBy,
		}

		_, err := s.AssignRoleToUser(ctx, schemaName, assignReq)
		if err != nil {
			assignmentErrors = append(assignmentErrors, err)
		}
	}

	// If all assignments failed, return error
	if len(assignmentErrors) == len(req.UserIDs) {
		return app_errors.BulkAssignmentFailed
	}

	return nil
}

func (s *accessMemberService) BulkRemoveRoleFromUsers(ctx context.Context, schemaName string, userIDs []string, scopeType string, scopeID *string, roleID string) error {
	if len(userIDs) == 0 {
		return app_errors.EmptyUserList
	}

	if scopeType == "" {
		return app_errors.InvalidScopeType
	}

	var removalErrors []error
	scopeIDStr := ""
	if scopeID != nil {
		scopeIDStr = *scopeID
	}

	for _, userID := range userIDs {
		if userID == "" {
			continue
		}

		err := s.RemoveRoleFromUser(ctx, schemaName, userID, scopeIDStr, scopeType)
		if err != nil {
			removalErrors = append(removalErrors, err)
		}
	}

	// If all removals failed, return error
	if len(removalErrors) == len(userIDs) {
		return app_errors.BulkRemovalFailed
	}

	return nil
}
