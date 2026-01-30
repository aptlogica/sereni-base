package services

import (
	"context"
	"fmt"
	"go-postgres-rest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	common "serenibase/internal/services/common"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	dbModels "go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

const (
	AccessMembersTableFormat = "\"%s\".access_members"
)

type accessMemberService struct {
	repo *pkg.DatabaseService
}

func NewAccessMemberService(repo *pkg.DatabaseService) interfaces.AccessMemberService {
	return &accessMemberService{repo: repo}
}

func (s *accessMemberService) getTableName(schemaName string) string {
	return fmt.Sprintf(AccessMembersTableFormat, schemaName)
}

func (s *accessMemberService) mapToAccessMemberDTOs(data []map[string]interface{}) []dto.AccessMemberDTO {
	members, err := common.MapToStructList[dto.AccessMemberDTO](data)
	if err != nil {
		return []dto.AccessMemberDTO{}
	}
	return members
}

func (s *accessMemberService) AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := s.getTableName(schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(tableName, req.Map())
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

	tableName := s.getTableName(schemaName)

	// Delete all access members for user in this scope
	filters := []dbModels.QueryFilter{
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
	}

	if scopeID != "" {
		filters = append(filters, dbModels.QueryFilter{
			Column:   "scope_id",
			Operator: "eq",
			Value:    scopeID,
		})
	}

	query := common.CreateMultiFilterQuery(filters, nil, nil)
	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to get access members for removal")
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
		deleteErr := s.repo.TableService.DeleteRecord(tableName, filter)
		if deleteErr != nil {
			return app_errors.AccessMemberDeleteFailed
		}
	}

	return nil
}

// RemoveAccessMemberByID deletes an access member record directly by its ID
// This is more reliable than searching by composite key (user_id, scope_id, scope_type)
func (s *accessMemberService) RemoveAccessMemberByID(ctx context.Context, schemaName string, memberID string) error {
	if memberID == "" {
		return app_errors.ErrRecordNotFound
	}

	tableName := s.getTableName(schemaName)

	// Pass just the ID string, not a QueryFilter struct
	deleteErr := s.repo.TableService.DeleteRecord(tableName, memberID)
	if deleteErr != nil {
		return deleteErr
	}

	return nil
}

func (s *accessMemberService) GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
	tableName := s.getTableName(schemaName)
	filters := []dbModels.QueryFilter{
		{
			Column:   "user_id",
			Operator: "eq",
			Value:    userID,
		},
	}
	query := common.CreateMultiFilterQuery(filters, nil, nil)

	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch user access members")
	}

	return s.mapToAccessMemberDTOs(data), nil
}

func (s *accessMemberService) GetUserAccessByScope(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	if userID == "" {
		return nil, app_errors.ErrRecordNotFound
	}
	if scopeType == "" {
		return nil, app_errors.InvalidScopeType
	}

	tableName := s.getTableName(schemaName)
	filters := []dbModels.QueryFilter{
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
	}

	if scopeID != nil && *scopeID != "" {
		filters = append(filters, dbModels.QueryFilter{
			Column:   "scope_id",
			Operator: "eq",
			Value:    *scopeID,
		})
	}

	query := common.CreateMultiFilterQuery(filters, nil, nil)
	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch user access by scope")
	}

	return s.mapToAccessMemberDTOs(data), nil
}

func (s *accessMemberService) GetScopeMembers(ctx context.Context, schemaName string, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	tableName := s.getTableName(schemaName)
	filters := []dbModels.QueryFilter{
		{
			Column:   "scope_type",
			Operator: "eq",
			Value:    scopeType,
		},
	}

	if scopeID != nil && *scopeID != "" {
		filters = append(filters, dbModels.QueryFilter{
			Column:   "scope_id",
			Operator: "eq",
			Value:    *scopeID,
		})
	}

	query := common.CreateMultiFilterQuery(filters, nil, nil)
	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch scope members")
	}

	return s.mapToAccessMemberDTOs(data), nil
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

// UpdateRoleForUser updates the role for a user in a specific scope
func (s *accessMemberService) UpdateRoleForUser(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, newRoleID string) error {
	if userID == "" {
		return app_errors.ErrRecordNotFound
	}
	if scopeType == "" {
		return app_errors.InvalidScopeType
	}

	tableName := s.getTableName(schemaName)

	// Find existing access member record
	filters := []dbModels.QueryFilter{
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
	}

	if scopeID != nil && *scopeID != "" {
		filters = append(filters, dbModels.QueryFilter{
			Column:   "scope_id",
			Operator: "eq",
			Value:    *scopeID,
		})
	}

	query := common.CreateMultiFilterQuery(filters, nil, nil)
	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to fetch access member for role update")
	}

	if len(data) == 0 {
		return app_errors.AccessMemberNotFound
	}

	var am tenant.AccessMember
	if err := helpers.MapToStruct(data[0], &am); err != nil {
		return app_errors.ErrMapToStruct
	}

	// Update the role_id for the access member
	updateData := map[string]interface{}{
		"role_id": newRoleID,
	}

	_, err = s.repo.TableService.UpdateRecord(tableName, am.ID.String(), updateData)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to update role for user")
	}

	return nil
}
