package services

import (
	"context"
	"encoding/json"
	"fmt"
	"godbgrest/pkg"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/models/tenant"
	"serenibase/internal/providers/logger"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"time"

	"serenibase/internal/constant"

	dbModels "godbgrest/pkg/models"
	app_errors "serenibase/internal/app-errors"

	"github.com/google/uuid"
)

type tenantManagementService struct {
	tableName           string
	repo                *pkg.DatabaseService
	tenantService       interfaces.TenantService
	subscriptionService interfaces.TenantSubscriptionService
	membershipService   interfaces.TenantMembershipService
}

func NewTenantManagementService(
	repo *pkg.DatabaseService,
	tenantService interfaces.TenantService,
	subscriptionService interfaces.TenantSubscriptionService,
	membershipService interfaces.TenantMembershipService,
) interfaces.TenantManagementService {
	tableName := master.Tenant{}.TableName(constant.MasterDatabase)
	return &tenantManagementService{
		tableName:           tableName,
		repo:                repo,
		tenantService:       tenantService,
		subscriptionService: subscriptionService,
		membershipService:   membershipService,
	}
}

func (t *tenantManagementService) createTableUsingSchema(schema dbModels.CreateTableRequest) error {
	return t.repo.TableService.CreateTable(schema)
}

func (t *tenantManagementService) createTenantSchema(ctx context.Context, schema string) error {
	if err := t.repo.TableService.CreateSchema(ctx, schema); err != nil {
		return app_errors.DatabaseError
	}

	type tenantTable interface {
		TableName(prefix string) string
		TableSchema(prefix string) dbModels.CreateTableRequest
	}

	tables := []tenantTable{
		tenant.Workspace{},
		tenant.WorkspaceMember{},
		tenant.Base{},
		tenant.Model{},
		tenant.Column{},
		tenant.View{},
		tenant.ViewColumn{},
		tenant.Relation{},
		tenant.APIToken{},
		tenant.Hook{},
		tenant.Assets{},
		tenant.Role{},
		tenant.User{},
		tenant.UserRole{},
		// tenant.TenantFeature{},
		// tenant.FeatureFlag{},
	}

	for _, table := range tables {

		if err := t.repo.TableService.CreateTable(table.(interface {
			TableSchema(string) dbModels.CreateTableRequest
		}).TableSchema(schema)); err != nil {
			fmt.Printf(table.TableName(schema), err)
			return err
		}
	}

	return nil
}

// Tenant lifecycle management
func (t *tenantManagementService) InitializeTenant(ctx context.Context, req dto.TenantRequest, planId uuid.UUID, roleId uuid.UUID) (dto.TenantResponse, error) {
	lg := logger.Get()
	// Step 1: Create the tenant
	insertedTenant, err := t.tenantService.CreateTenant(ctx, req)
	if err != nil {
		return dto.TenantResponse{}, err
	}

	var tenant dto.TenantResponse
	if err := helpers.StructToStruct(insertedTenant, &tenant); err != nil {
		return dto.TenantResponse{}, app_errors.ErrStructToStruct
	}

	// Step 2: Create the tenant subscription
	subscriptionReq := dto.TenantSubscriptionInsertion{
		ID:        uuid.New(),
		TenantID:  insertedTenant.ID,
		PlanID:    planId,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	insertedTenantSubscription, err := t.subscriptionService.CreateTenantSubscription(ctx, subscriptionReq)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to create tenant subscription")
		return dto.TenantResponse{}, err
	}

	var tenantSubscription dto.TenantSubscriptionResponse
	if err := helpers.StructToStruct(insertedTenantSubscription, &tenantSubscription); err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to map tenant subscription")
		return dto.TenantResponse{}, err
	}

	// Step 3: Create the tenant membership for the user (as owner/admin)
	membershipReq := dto.TenantMembershipInsertion{
		ID:        uuid.New(),
		TenantID:  insertedTenant.ID,
		UserID:    req.UserID,
		RoleID:    roleId,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertedTenantMenmbership, err := t.membershipService.CreateTenantMembership(ctx, membershipReq)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to create tenant membership")
		return dto.TenantResponse{}, err
	}

	var tenantMembership dto.TenantMembershipResponse
	if err := helpers.StructToStruct(insertedTenantMenmbership, &tenantMembership); err != nil {
		fmt.Printf("failed to map tenant membership: %v", err)
		return dto.TenantResponse{}, err
	}

	if err := t.createTenantSchema(ctx, insertedTenant.Schema); err != nil {
		return dto.TenantResponse{}, err
	}

	tenant.Subscription = &tenantSubscription
	tenant.Membership = &tenantMembership

	return tenant, nil
}

// need to optimize
func (t *tenantManagementService) GetTenantByUserId(ctx context.Context, userId uuid.UUID) (dto.TenantResponse, error) {
	membership, err := t.membershipService.GetTenantMembershipByUser(ctx, userId)
	if err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to get tenant membership: %w", err)
	}

	subscription, err := t.subscriptionService.GetTenantSubscription(ctx, membership.TenantID.String())
	if err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to get tenant subscription: %w", err)
	}

	tenantData, err := t.tenantService.GetTenant(ctx, membership.TenantID)
	if err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to get tenant: %w", err)
	}

	var tenant dto.TenantResponse
	if err := helpers.StructToStruct(tenantData, &tenant); err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to map tenant membership: %w", err)
	}

	var tenantSubscription dto.TenantSubscriptionResponse
	if err := helpers.StructToStruct(subscription, &tenantSubscription); err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to map tenant subscription: %w", err)
	}

	var tenantMembership dto.TenantMembershipResponse
	if err := helpers.StructToStruct(membership, &tenantMembership); err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to map tenant membership: %w", err)
	}

	tenant.Subscription = &tenantSubscription
	tenant.Membership = &tenantMembership

	return tenant, nil
}

func (t *tenantManagementService) DeactivateTenant(ctx context.Context, tenantID string) error {
	updates := map[string]interface{}{
		"status":             "pending",
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantManagementService) ReactivateTenant(ctx context.Context, tenantID string) error {
	updates := map[string]interface{}{
		"status":             "active",
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantManagementService) DeleteTenant(ctx context.Context, tenantID string) error {
	updates := map[string]interface{}{
		"is_deleted":         true,
		"deleted_at":         time.Now(),
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

// Tenant information
func (t *tenantManagementService) GetTenant(ctx context.Context, tenantID string) (master.Tenant, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    tenantID,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return master.Tenant{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return master.Tenant{}, app_errors.TenantNotFound
	}

	var tenant master.Tenant
	if err := helpers.MapToStruct(data[0], &tenant); err != nil {
		return master.Tenant{}, app_errors.ErrMapToStruct
	}

	return tenant, nil
}

func (t *tenantManagementService) ValidateSchema(ctx context.Context, schema string) (bool, error) {
	return t.tenantService.SchemaExists(ctx, schema)
}

func (t *tenantManagementService) GetTenantBySlug(ctx context.Context, slug string) (master.Tenant, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "slug",
				Operator: "eq",
				Value:    slug,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return master.Tenant{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return master.Tenant{}, app_errors.DatabaseError
	}

	var tenant master.Tenant
	if err := helpers.MapToStruct(data[0], &tenant); err != nil {
		return master.Tenant{}, app_errors.ErrMapToStruct
	}

	return tenant, nil
}

func (t *tenantManagementService) GetTenantByDomain(ctx context.Context, domain string) (master.Tenant, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "domain",
				Operator: "eq",
				Value:    domain,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return master.Tenant{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return master.Tenant{}, app_errors.DatabaseError
	}

	var tenant master.Tenant
	if err := helpers.MapToStruct(data[0], &tenant); err != nil {
		return master.Tenant{}, app_errors.ErrMapToStruct
	}

	return tenant, nil
}

func (t *tenantManagementService) UpdateTenantBySchema(ctx context.Context, schema string, req dto.UpdateTenantRequest) (dto.TenantResponse, error) {
	tenant, err := t.GetTenantBySchema(ctx, schema)
	if err != nil {
		return dto.TenantResponse{}, err
	}

	req.UpdatedAt = time.Now()
	updateFields := req.Map()
	if len(updateFields) == 0 {
		return dto.TenantResponse{}, app_errors.InvalidPayload
	}

	updatedTenant, err := t.tenantService.Update(ctx, tenant.ID.String(), updateFields)
	if err != nil {
		return dto.TenantResponse{}, err
	}

	lg := logger.Get()
	lg.Debug().Interface("tenant", updatedTenant).Msg("Updated tenant successfully")

	var tenantData dto.TenantResponse
	if err := helpers.StructToStruct(updatedTenant, &tenantData); err != nil {
		return dto.TenantResponse{}, app_errors.ErrStructToStruct
	}

	return tenantData, nil
}

// Tenant settings and configuration
func (t *tenantManagementService) UpdateTenantSettings(ctx context.Context, tenantID string, settings map[string]interface{}) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"settings":           string(settingsJSON),
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err = t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantManagementService) GetTenantSettings(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	tenant, err := t.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if tenant.Settings == nil || *tenant.Settings == "" {
		return make(map[string]interface{}), nil
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(*tenant.Settings), &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func (t *tenantManagementService) UpdateTenantLimits(ctx context.Context, tenantID string, limits map[string]int) error {
	updates := make(map[string]interface{})
	updates["last_modified_time"] = time.Now()

	// Map limit keys to database columns
	limitMappings := map[string]string{
		"max_workspaces":          "max_workspaces",
		"max_bases_per_workspace": "max_bases_per_workspace",
		"max_tables_per_base":     "max_tables_per_base",
		"max_rows_per_table":      "max_rows_per_table",
		"max_collaborators":       "max_collaborators",
		"max_api_calls_per_hour":  "max_api_calls_per_hour",
		"storage_limit_gb":        "storage_limit_gb",
	}

	for key, value := range limits {
		if dbColumn, exists := limitMappings[key]; exists {
			updates[dbColumn] = value
		}
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantManagementService) GetTenantLimits(ctx context.Context, tenantID string) (map[string]int, error) {
	tenant, err := t.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	limits := map[string]int{
		"max_workspaces":          tenant.MaxWorkspaces,
		"max_bases_per_workspace": tenant.MaxBasesPerWorkspace,
		"max_tables_per_base":     tenant.MaxTablesPerBase,
		"max_rows_per_table":      tenant.MaxRowsPerTable,
		"max_collaborators":       tenant.MaxCollaborators,
		"max_api_calls_per_hour":  tenant.MaxAPICallsPerHour,
		"storage_limit_gb":        tenant.StorageLimitGB,
	}

	return limits, nil
}

// Tenant analytics and monitoring
func (t *tenantManagementService) GetTenantUsage(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	// This would typically aggregate data from various tables
	// For now, returning basic tenant info
	tenant, err := t.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	members, err := t.membershipService.GetTenantMembers(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	subscription, err := t.subscriptionService.GetTenantSubscription(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	usage := map[string]interface{}{
		"tenant_id":           tenant.ID,
		"tenant_name":         tenant.Name,
		"tenant_status":       tenant.Status,
		"member_count":        len(members),
		"subscription_status": subscription.Status,
		"created_time":        tenant.CreatedAt,
		"last_updated":        tenant.UpdatedAt,
	}

	return usage, nil
}

func (t *tenantManagementService) GetTenantActivity(ctx context.Context, tenantID string, days int) ([]map[string]interface{}, error) {
	// This would typically query activity logs
	// For now, returning basic activity info
	tenant, err := t.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	activity := []map[string]interface{}{
		{
			"tenant_id":   tenant.ID,
			"tenant_name": tenant.Name,
			"activity":    "tenant_accessed",
			"timestamp":   tenant.UpdatedAt,
		},
	}

	return activity, nil
}

func (t *tenantManagementService) GetTenantHealth(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	tenant, err := t.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	subscription, err := t.subscriptionService.GetTenantSubscription(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	health := map[string]interface{}{
		"tenant_id":           tenant.ID,
		"tenant_name":         tenant.Name,
		"status":              tenant.Status,
		"subscription_status": subscription.Status,
		"schema_version":      tenant.SchemaVersion,
		"last_migration":      tenant.LastMigrationRun,
		"health_score":        95, // Placeholder
		"issues":              []string{},
	}

	return health, nil
}

// Bulk operations
func (t *tenantManagementService) GetAllTenants(ctx context.Context, filters map[string]interface{}) ([]master.Tenant, error) {
	query := dbModels.QueryParams{}

	// Apply filters
	for key, value := range filters {
		query.Filters = append(query.Filters, dbModels.QueryFilter{
			Column:   key,
			Operator: "eq",
			Value:    value,
		})
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var tenants []master.Tenant
	for _, record := range data {
		var tenant master.Tenant
		if err := helpers.MapToStruct(record, &tenant); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

func (t *tenantManagementService) GetTenantsByStatus(ctx context.Context, status string) ([]master.Tenant, error) {
	filters := map[string]interface{}{
		"status": status,
	}
	return t.GetAllTenants(ctx, filters)
}

func (t *tenantManagementService) GetTenantsByRegion(ctx context.Context, region string) ([]master.Tenant, error) {
	filters := map[string]interface{}{
		"region": region,
	}
	return t.GetAllTenants(ctx, filters)
}

// Schema management
func (t *tenantManagementService) CreateTenantSchema(ctx context.Context, tenantID string) error {
	// This would typically create a new schema for the tenant
	// For now, just updating the schema version
	updates := map[string]interface{}{
		"schema_version":     "1.0.0",
		"last_migration_run": time.Now(),
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantManagementService) UpdateTenantSchema(ctx context.Context, tenantID string, version string) error {
	updates := map[string]interface{}{
		"schema_version":     version,
		"last_migration_run": time.Now(),
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantManagementService) GetTenantSchemaVersion(ctx context.Context, tenantID string) (string, error) {
	tenant, err := t.GetTenant(ctx, tenantID)
	if err != nil {
		return "", err
	}

	return tenant.SchemaVersion, nil
}

// Migration and maintenance
func (t *tenantManagementService) RunTenantMigration(ctx context.Context, tenantID string) error {
	// This would typically run database migrations for the tenant
	// For now, just updating the last migration timestamp
	updates := map[string]interface{}{
		"last_migration_run": time.Now(),
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantManagementService) BackupTenantData(ctx context.Context, tenantID string) (string, error) {
	// This would typically create a backup of the tenant's data
	// For now, returning a placeholder backup path
	backupPath := fmt.Sprintf("/backups/tenant_%s_%d.sql", tenantID, time.Now())
	return backupPath, nil
}

func (t *tenantManagementService) RestoreTenantData(ctx context.Context, tenantID string, backupPath string) error {
	// This would typically restore tenant data from a backup
	// For now, just updating the last migration timestamp
	updates := map[string]interface{}{
		"last_migration_run": time.Now(),
		"last_modified_time": time.Now(),
	}

	conditions := map[string]interface{}{
		"id": tenantID,
	}

	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
	return err
}

func (t *tenantManagementService) GetTenantBySchema(ctx context.Context, schema string) (master.Tenant, error) {
	return t.tenantService.GetTenantBySchema(ctx, schema)
}

func (t *tenantManagementService) GetTenantInfoBySchema(ctx context.Context, schema string) (dto.TenantResponse, error) {
	tenantData, err := t.tenantService.GetTenantBySchema(ctx, schema)
	if err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to get tenant by schema: %w", err)
	}

	var tenant dto.TenantResponse
	if err := helpers.StructToStruct(tenantData, &tenant); err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to map tenant: %w", err)
	}

	// Fetch subscription if needed
	subscription, err := t.subscriptionService.GetTenantSubscription(ctx, tenant.ID.String())
	if err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to get tenant subscription: %w", err)
	}

	var tenantSubscription dto.TenantSubscriptionResponse
	if err := helpers.StructToStruct(subscription, &tenantSubscription); err != nil {
		return dto.TenantResponse{}, fmt.Errorf("failed to map tenant subscription: %w", err)
	}
	tenant.Subscription = &tenantSubscription

	return tenant, nil
}
