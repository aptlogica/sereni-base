package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"strings"
	"time"

	"serenibase/internal/constant"

	app_errors "serenibase/internal/app-errors"

	dbModels "godbgrest/pkg/models"

	"github.com/google/uuid"
)

type tenantService struct {
	tableName string
	repo      *pkg.DatabaseService
}

func NewTenantService(
	repo *pkg.DatabaseService) interfaces.TenantService {
	tableName := master.Tenant{}.TableName(constant.MasterDatabase)
	return &tenantService{
		tableName: tableName,
		repo:      repo,
	}
}

func (t *tenantService) generateSlug(orgName string) string {
	slug := orgName
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix())
	return slug
}

func (t *tenantService) generateSchemaName(slug string) string {
	if slug == "" {
		return uuid.New().String()
	}
	return slug + "_" + uuid.New().String()
}

func (t *tenantService) CreateTenant(ctx context.Context, req dto.TenantRequest) (master.Tenant, error) {
	// slug := t.generateSlug(req.OrganizationName)
	// schema := t.generateSchemaName(slug)
	slug := uuid.New().String()
	schema := t.generateSchemaName("")

	tenantData := dto.TenantInsertion{
		ID:     req.TenantID,
		Name:   "Company Name",
		Slug:   slug,
		Schema: schema,
	}

	insertedTenantData, err := t.repo.TableService.CreateRecord(ctx, t.tableName, tenantData.Map())
	if err != nil {
		return master.Tenant{}, app_errors.DatabaseError
	}

	var insertedTenant master.Tenant
	if err := helpers.MapToStruct(insertedTenantData, &insertedTenant); err != nil {
		return master.Tenant{}, app_errors.ErrMapToStruct
	}
	return insertedTenant, nil
}

func (t *tenantService) GetTenant(ctx context.Context, id uuid.UUID) (master.Tenant, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    id,
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

func (t *tenantService) GetTenantBySchema(ctx context.Context, schema string) (master.Tenant, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "schema_name",
				Operator: "eq",
				Value:    schema,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		fmt.Println("err:= ", err)
		return master.Tenant{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		fmt.Println("err:= ", err)
		return master.Tenant{}, app_errors.TenantNotFound
	}

	var tenant master.Tenant
	if err := helpers.MapToStruct(data[0], &tenant); err != nil {
		return master.Tenant{}, app_errors.ErrMapToStruct
	}
	return tenant, nil
}

func (t *tenantService) SchemaExists(ctx context.Context, schema string) (bool, error) {
	limit := 1
	query := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "schema_name",
				Operator: "eq",
				Value:    schema,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		fmt.Println("SchemaExists---->", err)
		return false, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return false, nil
	}

	return true, nil
}

func (t *tenantService) Update(ctx context.Context, tenantID string, updateData map[string]interface{}) (master.Tenant, error) {
	updatedData, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, tenantID, updateData)
	if err != nil {
		return master.Tenant{}, app_errors.DatabaseError
	}

	var tenant master.Tenant
	if err := helpers.MapToStruct(updatedData, &tenant); err != nil {
		return master.Tenant{}, app_errors.ErrMapToStruct
	}

	return tenant, nil
}