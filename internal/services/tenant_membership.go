package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"godbgrest/pkg"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"time"

	"serenibase/internal/constant"

	dbModels "godbgrest/pkg/models"
	app_errors "serenibase/internal/app-errors"

	"github.com/google/uuid"
)

type tenantMembershipService struct {
	tableName string
	repo      *pkg.DatabaseService
}

func NewTenantMembershipService(repo *pkg.DatabaseService) interfaces.TenantMembershipService {
	tableName := master.TenantMembership{}.TableName(constant.MasterDatabase)
	return &tenantMembershipService{tableName: tableName, repo: repo}
}

func (t *tenantMembershipService) CreateTenantMembership(ctx context.Context, tenantMembershipData dto.TenantMembershipInsertion) (master.TenantMembership, error) {
	if tenantMembershipData.ID == uuid.Nil {
		tenantMembershipData.ID = uuid.New()
	}
	tenantMembershipData.CreatedAt = time.Now()
	tenantMembershipData.UpdatedAt = time.Now()

	membershipMap := tenantMembershipData.Map()

	insertedData, err := t.repo.TableService.CreateRecord(ctx, t.tableName, membershipMap)
	if err != nil {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	var insertedMembership master.TenantMembership
	if err := helpers.MapToStruct(insertedData, &insertedMembership); err != nil {
		return master.TenantMembership{}, app_errors.ErrMapToStruct
	}

	return insertedMembership, nil
}

func (t *tenantMembershipService) GetTenantMembership(ctx context.Context, tenantID string, userID uuid.UUID) (master.TenantMembership, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "tenant_id",
				Operator: "eq",
				Value:    tenantID,
			},
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	var membership master.TenantMembership
	if err := helpers.MapToStruct(data[0], &membership); err != nil {
		return master.TenantMembership{}, app_errors.ErrMapToStruct
	}

	return membership, nil
}

func (t *tenantMembershipService) GetTenantMembershipByUser(ctx context.Context, userID uuid.UUID) (master.TenantMembership, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	var membership master.TenantMembership
	if err := helpers.MapToStruct(data[0], &membership); err != nil {
		return master.TenantMembership{}, app_errors.ErrMapToStruct
	}

	return membership, nil
}

func (t *tenantMembershipService) GetTenantMembershipByID(ctx context.Context, membershipID uuid.UUID) (master.TenantMembership, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    membershipID,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	var membership master.TenantMembership
	if err := helpers.MapToStruct(data[0], &membership); err != nil {
		return master.TenantMembership{}, app_errors.ErrMapToStruct
	}

	return membership, nil
}

func (t *tenantMembershipService) GetTenantMembers(ctx context.Context, tenantID string) ([]master.TenantMembership, error) {
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "tenant_id",
				Operator: "eq",
				Value:    tenantID,
			},
		},
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var memberships []master.TenantMembership
	for _, record := range data {
		var membership master.TenantMembership
		if err := helpers.MapToStruct(record, &membership); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		memberships = append(memberships, membership)
	}

	return memberships, nil
}

func (t *tenantMembershipService) GetUserTenants(ctx context.Context, userID uuid.UUID) ([]master.TenantMembership, error) {
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
		},
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var memberships []master.TenantMembership
	for _, record := range data {
		var membership master.TenantMembership
		if err := helpers.MapToStruct(record, &membership); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		memberships = append(memberships, membership)
	}

	return memberships, nil
}

func (t *tenantMembershipService) UpdateTenantMembership(ctx context.Context, membershipID uuid.UUID, updates map[string]interface{}) (master.TenantMembership, error) {
	updates["last_modified_time"] = time.Now()

	conditions := map[string]interface{}{
		"id": membershipID,
	}

	updatedData, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	if err != nil {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	var membership master.TenantMembership
	if err := helpers.MapToStruct(updatedData, &membership); err != nil {
		return master.TenantMembership{}, app_errors.ErrMapToStruct
	}

	return membership, nil
}

func (t *tenantMembershipService) UpdateUserRole(ctx context.Context, tenantID, userID, roleID uuid.UUID) error {
	updates := map[string]interface{}{
		"role_id":    roleID,
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantMembershipService) UpdateUserPermissions(ctx context.Context, tenantID, userID uuid.UUID, permissions string) error {
	updates := map[string]interface{}{
		"permissions": permissions,
		"last_modified_time":  time.Now(),
	}

	conditions := map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantMembershipService) RemoveTenantMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	conditions := map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
	}

	err := t.repo.TableService.DeleteRecord(ctx, t.tableName, conditions)
	return err
}

func (t *tenantMembershipService) generateInvitationToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (t *tenantMembershipService) InviteUserToTenant(ctx context.Context, tenantID, invitedBy uuid.UUID, email, roleID string) (master.TenantMembership, error) {
	// Generate invitation token
	token := t.generateInvitationToken()
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	// Create membership with invitation details
	membershipData := dto.TenantMembershipInsertion{
		ID:          uuid.New(),
		TenantID:    tenantID,
		UserID:      uuid.Nil, // Will be set when user accepts invitation
		RoleID:      uuid.MustParse(roleID),
		Status:      "invited",
		Permissions: "[]",
	}

	membershipMap := membershipData.Map()
	membershipMap["invited_by"] = invitedBy.String()
	membershipMap["invited_at"] = time.Now()
	membershipMap["invitation_token"] = token
	membershipMap["invitation_expires_at"] = expiresAt.Unix()

	insertedData, err := t.repo.TableService.CreateRecord(ctx, t.tableName, membershipMap)
	if err != nil {
		return master.TenantMembership{}, app_errors.DatabaseError
	}

	var membership master.TenantMembership
	if err := helpers.MapToStruct(insertedData, &membership); err != nil {
		return master.TenantMembership{}, app_errors.ErrMapToStruct
	}

	return membership, nil
}

func (t *tenantMembershipService) AcceptInvitation(ctx context.Context, invitationToken string) error {
	// Find membership by invitation token
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "invitation_token",
				Operator: "eq",
				Value:    invitationToken,
			},
			{
				Column:   "status",
				Operator: "eq",
				Value:    "invited",
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return app_errors.DatabaseError
	}

	if len(data) == 0 {
		return app_errors.DatabaseError
	}

	var membership master.TenantMembership
	if err := helpers.MapToStruct(data[0], &membership); err != nil {
		return app_errors.ErrMapToStruct
	}

	// Check if invitation is expired
	if membership.InvitationExpiresAt != nil && time.Now().After(*membership.InvitationExpiresAt) {
		return fmt.Errorf("invitation has expired")
	}

	// Update membership to accepted
	updates := map[string]interface{}{
		"status":                "active",
		"joined_at":             time.Now(),
		"invitation_token":      nil,
		"invitation_expires_at": nil,
		"last_modified_time":            time.Now(),
	}

	updateConditions := map[string]interface{}{
		"id": membership.ID,
	}

	_, err = t.repo.TableService.UpdateRecord(ctx, t.tableName, updateConditions, updates)
	return err
}

func (t *tenantMembershipService) RejectInvitation(ctx context.Context, invitationToken string) error {
	conditions := map[string]interface{}{
		"invitation_token": invitationToken,
		"status":           "invited",
	}

	err := t.repo.TableService.DeleteRecord(ctx, t.tableName, conditions)
	return err
}

func (t *tenantMembershipService) GetPendingInvitations(ctx context.Context, tenantID uuid.UUID) ([]master.TenantMembership, error) {
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "tenant_id",
				Operator: "eq",
				Value:    tenantID,
			},
			{
				Column:   "status",
				Operator: "eq",
				Value:    "invited",
			},
		},
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var memberships []master.TenantMembership
	for _, record := range data {
		var membership master.TenantMembership
		if err := helpers.MapToStruct(record, &membership); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		memberships = append(memberships, membership)
	}

	return memberships, nil
}

func (t *tenantMembershipService) UpdateLastAccess(ctx context.Context, tenantID, userID uuid.UUID) error {
	updates := map[string]interface{}{
		"last_access_at": time.Now(),
		"last_modified_time":     time.Now(),
	}

	conditions := map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantMembershipService) CheckUserPermission(ctx context.Context, tenantID string, userID uuid.UUID, permission string) (bool, error) {
	membership, err := t.GetTenantMembership(ctx, tenantID, userID)
	if err != nil {
		return false, err
	}

	if membership.Status != "active" {
		return false, nil
	}

	// Parse permissions JSON
	var permissions []string
	if err := json.Unmarshal([]byte(membership.Permissions), &permissions); err != nil {
		return false, err
	}

	// Check if permission exists
	for _, perm := range permissions {
		if perm == permission || perm == "*" { // "*" means all permissions
			return true, nil
		}
	}

	return false, nil
}
