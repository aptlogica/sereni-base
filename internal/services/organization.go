package services

import (
	"context"
	"encoding/json"
	"fmt"
	"go-postgres-rest/pkg"
	"go-postgres-rest/pkg/models"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"time"

	app_errors "serenibase/internal/app-errors"

	"github.com/google/uuid"
)

type organizationService struct {
	repo *pkg.DatabaseService
}

const (
	ErrMapOrganizationData = "ERROR: Failed to map organization data: %v\n"
)

func NewOrganizationService(repo *pkg.DatabaseService) interfaces.OrganizationService {
	return &organizationService{
		repo: repo,
	}
}

// Helper function to create int pointer
func ptrInt(i int) *int {
	return &i
}

func (s *organizationService) CreateOrganization(ctx context.Context, schemaName string, req dto.CreateOrganizationRequest) (tenant.Organization, error) {
	organization := tenant.Organization{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Email:       req.Email,
		Phone:       req.Phone,
		Website:     req.Website,
		Logo:        req.Logo,
		Address:     req.Address,
		City:        req.City,
		State:       req.State,
		Country:     req.Country,
		ZipCode:     req.ZipCode,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tblName := organization.TableName(schemaName)

	// Convert struct to map for database insertion
	orgData := map[string]interface{}{
		"id":                 organization.ID.String(),
		"name":               organization.Name,
		"description":        organization.Description,
		"email":              organization.Email,
		"phone":              organization.Phone,
		"website":            organization.Website,
		"logo":               organization.Logo,
		"address":            organization.Address,
		"city":               organization.City,
		"state":              organization.State,
		"country":            organization.Country,
		"zip_code":           organization.ZipCode,
		"status":             organization.Status,
		"created_time":       organization.CreatedAt,
		"last_modified_time": organization.UpdatedAt,
	}

	// Only add settings/meta if provided
	if req.Settings != nil {
		jsonBytes, err := json.Marshal(req.Settings)
		if err != nil {
			fmt.Printf("ERROR: Failed to marshal settings: %v\n", err)
			return tenant.Organization{}, fmt.Errorf("invalid settings format")
		}
		orgData["settings"] = string(jsonBytes)
	}

	if req.Meta != nil {
		jsonBytes, err := json.Marshal(req.Meta)
		if err != nil {
			fmt.Printf("ERROR: Failed to marshal meta: %v\n", err)
			return tenant.Organization{}, fmt.Errorf("invalid meta format")
		}
		orgData["meta"] = string(jsonBytes)
	}

	_, err := s.repo.TableService.CreateRecord(tblName, orgData)
	if err != nil {
		fmt.Printf("ERROR: Failed to create organization: %v\n", err)
		return tenant.Organization{}, app_errors.LogDatabaseError(err, "failed to create organization")
	}

	fmt.Printf("DEBUG: Organization created successfully with ID: %s\n", organization.ID.String())
	return organization, nil
}

func (s *organizationService) GetOrganizationByID(ctx context.Context, schemaName string, id string) (tenant.Organization, error) {
	if id == "" {
		return tenant.Organization{}, fmt.Errorf("organization ID cannot be empty")
	}

	organization := tenant.Organization{}
	tblName := organization.TableName(schemaName)

	limit := 1
	query := models.QueryParams{
		Select: []string{"*"},
		Filters: []models.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    id,
			},
		},
		Limit: &limit,
	}

	// Fetch organization row(s)
	organizationData, err := s.repo.TableService.GetTableData(tblName, query)
	if err != nil {
		fmt.Printf("ERROR: Failed to get organization by ID %s: %v\n", id, err)
		return tenant.Organization{}, app_errors.LogDatabaseError(err, "failed to get organization by id")
	}

	if len(organizationData) == 0 {
		fmt.Printf("DEBUG: Organization not found with ID: %s\n", id)
		return tenant.Organization{}, app_errors.ErrRecordNotFound
	}

	orgData := organizationData[0]

	var org tenant.Organization
	if err := helpers.MapToStruct(orgData, &org); err != nil {
		fmt.Printf(ErrMapOrganizationData, err)
		return tenant.Organization{}, app_errors.ErrStructToStruct
	}

	return org, nil
}

func (s *organizationService) GetAllOrganizations(ctx context.Context, schemaName string) ([]tenant.Organization, error) {
	organization := tenant.Organization{}
	tblName := organization.TableName(schemaName)

	query := models.QueryParams{
		Select: []string{"*"},
	}

	records, err := s.repo.TableService.GetTableData(tblName, query)
	if err != nil {
		fmt.Printf("ERROR: Failed to get all organizations: %v\n", err)
		return nil, app_errors.LogDatabaseError(err, "failed to get all organizations")
	}

	var organizations []tenant.Organization
	for _, record := range records {
		var org tenant.Organization
		if err := helpers.MapToStruct(record, &org); err != nil {
			fmt.Printf(ErrMapOrganizationData, err)
			continue
		}
		organizations = append(organizations, org)
	}

	fmt.Printf("DEBUG: Retrieved %d organizations\n", len(organizations))
	return organizations, nil
}

// GetOrganization - Get the single organization (there's only one per system)
func (s *organizationService) GetOrganization(ctx context.Context, schemaName string) (tenant.Organization, error) {
	organization := tenant.Organization{}
	tblName := organization.TableName(schemaName)

	query := models.QueryParams{
		Select: []string{"*"},
		Limit:  ptrInt(1),
	}

	records, err := s.repo.TableService.GetTableData(tblName, query)
	if err != nil {
		fmt.Printf("ERROR: Failed to get organization: %v\n", err)
		return tenant.Organization{}, app_errors.LogDatabaseError(err, "failed to get organization")
	}

	if len(records) == 0 {
		fmt.Printf("DEBUG: No organization found\n")
		return tenant.Organization{}, app_errors.ErrRecordNotFound
	}

	var org tenant.Organization
	if err := helpers.MapToStruct(records[0], &org); err != nil {
		fmt.Printf(ErrMapOrganizationData, err)
		return tenant.Organization{}, app_errors.ErrStructToStruct
	}

	return org, nil
}

func (s *organizationService) UpdateOrganization(ctx context.Context, schemaName string, id string, req dto.UpdateOrganizationRequest) (tenant.Organization, error) {
	organization, err := s.GetOrganizationByID(ctx, schemaName, id)
	if err != nil {
		return tenant.Organization{}, err
	}

	updateOrganizationFields(&organization, req)
	organization.UpdatedAt = time.Now()

	orgData := buildOrganizationUpdateMap(organization)
	if err := addJSONFieldsToUpdateMap(orgData, req); err != nil {
		return tenant.Organization{}, err
	}

	tblName := organization.TableName(schemaName)
	_, err = s.repo.TableService.UpdateRecord(tblName, id, orgData)
	if err != nil {
		fmt.Printf("ERROR: Failed to update organization %s: %v\n", id, err)
		return tenant.Organization{}, app_errors.LogDatabaseError(err, "failed to update organization")
	}

	fmt.Printf("DEBUG: Organization updated successfully with ID: %s\n", id)
	return organization, nil
}

func updateOrganizationFields(org *tenant.Organization, req dto.UpdateOrganizationRequest) {
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Description != nil {
		org.Description = req.Description
	}
	if req.Email != nil {
		org.Email = *req.Email
	}
	if req.Phone != nil {
		org.Phone = req.Phone
	}
	if req.Website != nil {
		org.Website = req.Website
	}
	if req.Logo != nil {
		org.Logo = req.Logo
	}
	if req.Address != nil {
		org.Address = req.Address
	}
	if req.City != nil {
		org.City = req.City
	}
	if req.State != nil {
		org.State = req.State
	}
	if req.Country != nil {
		org.Country = req.Country
	}
	if req.ZipCode != nil {
		org.ZipCode = req.ZipCode
	}
	if req.Settings != nil {
		org.Settings = req.Settings
	}
	if req.Meta != nil {
		org.Meta = req.Meta
	}
	if req.Status != nil {
		org.Status = *req.Status
	}
}

func buildOrganizationUpdateMap(org tenant.Organization) map[string]interface{} {
	return map[string]interface{}{
		"name":               org.Name,
		"description":        org.Description,
		"email":              org.Email,
		"phone":              org.Phone,
		"website":            org.Website,
		"logo":               org.Logo,
		"address":            org.Address,
		"city":               org.City,
		"state":              org.State,
		"country":            org.Country,
		"zip_code":           org.ZipCode,
		"status":             org.Status,
		"last_modified_time": org.UpdatedAt,
	}
}

func addJSONFieldsToUpdateMap(orgData map[string]interface{}, req dto.UpdateOrganizationRequest) error {
	if req.Settings != nil {
		jsonBytes, err := json.Marshal(req.Settings)
		if err != nil {
			fmt.Printf("ERROR: Failed to marshal settings: %v\n", err)
			return fmt.Errorf("invalid settings format")
		}
		orgData["settings"] = string(jsonBytes)
	}

	if req.Meta != nil {
		jsonBytes, err := json.Marshal(req.Meta)
		if err != nil {
			fmt.Printf("ERROR: Failed to marshal meta: %v\n", err)
			return fmt.Errorf("invalid meta format")
		}
		orgData["meta"] = string(jsonBytes)
	}

	return nil
}

func (s *organizationService) DeleteOrganization(ctx context.Context, schemaName string, id string) error {
	organization := tenant.Organization{}
	tblName := organization.TableName(schemaName)

	err := s.repo.TableService.DeleteRecord(tblName, id)
	if err != nil {
		fmt.Printf("ERROR: Failed to delete organization %s: %v\n", id, err)
		return app_errors.LogDatabaseError(err, "failed to delete organization")
	}

	fmt.Printf("DEBUG: Organization deleted successfully with ID: %s\n", id)
	return nil
}

func (s *organizationService) GetOrganizationByEmail(ctx context.Context, schemaName string, email string) (tenant.Organization, error) {
	organization := tenant.Organization{}
	tblName := organization.TableName(schemaName)

	limit := 1
	query := models.QueryParams{
		Select: []string{"*"},
		Filters: []models.QueryFilter{
			{
				Column:   "email",
				Operator: "eq",
				Value:    email,
			},
		},
		Limit: &limit,
	}

	records, err := s.repo.TableService.GetTableData(tblName, query)
	if err != nil {
		fmt.Printf("ERROR: Failed to get organization by email %s: %v\n", email, err)
		return tenant.Organization{}, app_errors.LogDatabaseError(err, "failed to get organization by email")
	}

	if len(records) == 0 {
		fmt.Printf("DEBUG: Organization not found with email: %s\n", email)
		return tenant.Organization{}, app_errors.ErrRecordNotFound
	}

	orgData := records[0]
	var org tenant.Organization
	if err := helpers.MapToStruct(orgData, &org); err != nil {
		fmt.Printf(ErrMapOrganizationData, err)
		return tenant.Organization{}, app_errors.ErrStructToStruct
	}

	return org, nil
}
