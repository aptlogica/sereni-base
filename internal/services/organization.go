package services

import (
	"context"
	"encoding/json"
	"fmt"
	"godbgrest/pkg"
	"godbgrest/pkg/models"
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

	_, err := s.repo.TableService.CreateRecord(ctx, tblName, orgData)
	if err != nil {
		fmt.Printf("ERROR: Failed to create organization: %v\n", err)
		return tenant.Organization{}, app_errors.DatabaseError
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
	organizationData, err := s.repo.TableService.GetTableData(ctx, tblName, query)
	if err != nil {
		fmt.Printf("ERROR: Failed to get organization by ID %s: %v\n", id, err)
		return tenant.Organization{}, app_errors.DatabaseError
	}

	if len(organizationData) == 0 {
		fmt.Printf("DEBUG: Organization not found with ID: %s\n", id)
		return tenant.Organization{}, app_errors.ErrRecordNotFound
	}

	orgData := organizationData[0]

	var org tenant.Organization
	if err := helpers.MapToStruct(orgData, &org); err != nil {
		fmt.Printf("ERROR: Failed to map organization data: %v\n", err)
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

	records, err := s.repo.TableService.GetTableData(ctx, tblName, query)
	if err != nil {
		fmt.Printf("ERROR: Failed to get all organizations: %v\n", err)
		return nil, app_errors.DatabaseError
	}

	var organizations []tenant.Organization
	for _, record := range records {
		var org tenant.Organization
		if err := helpers.MapToStruct(record, &org); err != nil {
			fmt.Printf("ERROR: Failed to map organization data: %v\n", err)
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

	records, err := s.repo.TableService.GetTableData(ctx, tblName, query)
	if err != nil {
		fmt.Printf("ERROR: Failed to get organization: %v\n", err)
		return tenant.Organization{}, app_errors.DatabaseError
	}

	if len(records) == 0 {
		fmt.Printf("DEBUG: No organization found\n")
		return tenant.Organization{}, app_errors.ErrRecordNotFound
	}

	var org tenant.Organization
	if err := helpers.MapToStruct(records[0], &org); err != nil {
		fmt.Printf("ERROR: Failed to map organization data: %v\n", err)
		return tenant.Organization{}, app_errors.ErrStructToStruct
	}

	return org, nil
}

func (s *organizationService) UpdateOrganization(ctx context.Context, schemaName string, id string, req dto.UpdateOrganizationRequest) (tenant.Organization, error) {
	// Get existing organization
	organization, err := s.GetOrganizationByID(ctx, schemaName, id)
	if err != nil {
		return tenant.Organization{}, err
	}

	// Update fields if provided
	if req.Name != nil {
		organization.Name = *req.Name
	}
	if req.Description != nil {
		organization.Description = req.Description
	}
	if req.Email != nil {
		organization.Email = *req.Email
	}
	if req.Phone != nil {
		organization.Phone = req.Phone
	}
	if req.Website != nil {
		organization.Website = req.Website
	}
	if req.Logo != nil {
		organization.Logo = req.Logo
	}
	if req.Address != nil {
		organization.Address = req.Address
	}
	if req.City != nil {
		organization.City = req.City
	}
	if req.State != nil {
		organization.State = req.State
	}
	if req.Country != nil {
		organization.Country = req.Country
	}
	if req.ZipCode != nil {
		organization.ZipCode = req.ZipCode
	}
	if req.Settings != nil {
		organization.Settings = req.Settings
	}
	if req.Meta != nil {
		organization.Meta = req.Meta
	}
	if req.Status != nil {
		organization.Status = *req.Status
	}

	organization.UpdatedAt = time.Now()

	tblName := organization.TableName(schemaName)

	// Convert struct to map for database update
	orgData := map[string]interface{}{
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
		"last_modified_time": organization.UpdatedAt,
	}

	// Convert Settings and Meta to JSON if provided
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

	_, err = s.repo.TableService.UpdateRecord(ctx, tblName, id, orgData)
	if err != nil {
		fmt.Printf("ERROR: Failed to update organization %s: %v\n", id, err)
		return tenant.Organization{}, app_errors.DatabaseError
	}

	fmt.Printf("DEBUG: Organization updated successfully with ID: %s\n", id)
	return organization, nil
}

func (s *organizationService) DeleteOrganization(ctx context.Context, schemaName string, id string) error {
	organization := tenant.Organization{}
	tblName := organization.TableName(schemaName)

	err := s.repo.TableService.DeleteRecord(ctx, tblName, id)
	if err != nil {
		fmt.Printf("ERROR: Failed to delete organization %s: %v\n", id, err)
		return app_errors.DatabaseError
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

	records, err := s.repo.TableService.GetTableData(ctx, tblName, query)
	if err != nil {
		fmt.Printf("ERROR: Failed to get organization by email %s: %v\n", email, err)
		return tenant.Organization{}, app_errors.DatabaseError
	}

	if len(records) == 0 {
		fmt.Printf("DEBUG: Organization not found with email: %s\n", email)
		return tenant.Organization{}, app_errors.ErrRecordNotFound
	}

	orgData := records[0]
	var org tenant.Organization
	if err := helpers.MapToStruct(orgData, &org); err != nil {
		fmt.Printf("ERROR: Failed to map organization data: %v\n", err)
		return tenant.Organization{}, app_errors.ErrStructToStruct
	}

	return org, nil
}
