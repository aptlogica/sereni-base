package tests

import (
	"serenibase/internal/dto"
	"testing"

	"github.com/google/uuid"
)

func TestCreateOrganizationRequestFields(t *testing.T) {
	desc := "Test Organization"
	phone := "+1234567890"
	website := "https://example.com"
	logo := "logo.png"
	address := "123 Main St"
	city := "New York"
	state := "NY"
	country := "USA"
	zipCode := "10001"
	settings := map[string]interface{}{"key": "value"}
	meta := map[string]interface{}{"meta_key": "meta_value"}

	req := dto.CreateOrganizationRequest{
		Name:        "Test Org",
		Description: &desc,
		Email:       "test@example.com",
		Phone:       &phone,
		Website:     &website,
		Logo:        &logo,
		Address:     &address,
		City:        &city,
		State:       &state,
		Country:     &country,
		ZipCode:     &zipCode,
		Settings:    settings,
		Meta:        meta,
	}

	if req.Name != "Test Org" {
		t.Errorf("Name = %v, want %v", req.Name, "Test Org")
	}
	if req.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", req.Email, "test@example.com")
	}
	if *req.Description != "Test Organization" {
		t.Errorf("Description = %v, want %v", *req.Description, "Test Organization")
	}
	if *req.Phone != "+1234567890" {
		t.Errorf("Phone = %v, want %v", *req.Phone, "+1234567890")
	}
	if *req.Website != "https://example.com" {
		t.Errorf("Website = %v, want %v", *req.Website, "https://example.com")
	}
}

func TestUpdateOrganizationRequestFields(t *testing.T) {
	name := "Updated Org"
	desc := "Updated Description"
	email := "updated@example.com"
	status := "active"
	settings := map[string]interface{}{"updated_key": "updated_value"}
	meta := map[string]interface{}{"updated_meta": "updated_meta_value"}

	req := dto.UpdateOrganizationRequest{
		Name:        &name,
		Description: &desc,
		Email:       &email,
		Status:      &status,
		Settings:    settings,
		Meta:        meta,
	}

	if *req.Name != "Updated Org" {
		t.Errorf("Name = %v, want %v", *req.Name, "Updated Org")
	}
	if *req.Email != "updated@example.com" {
		t.Errorf("Email = %v, want %v", *req.Email, "updated@example.com")
	}
	if *req.Status != "active" {
		t.Errorf("Status = %v, want %v", *req.Status, "active")
	}
}

func TestUpdateOrganizationRequestEmpty(t *testing.T) {
	req := dto.UpdateOrganizationRequest{}

	if req.Name != nil {
		t.Errorf("Name should be nil, got %v", req.Name)
	}
	if req.Email != nil {
		t.Errorf("Email should be nil, got %v", req.Email)
	}
	if req.Status != nil {
		t.Errorf("Status should be nil, got %v", req.Status)
	}
}

func TestOrganizationResponseFields(t *testing.T) {
	id := uuid.New()
	desc := "Test Organization"
	phone := "+1234567890"
	settings := map[string]interface{}{"key": "value"}
	meta := map[string]interface{}{"meta_key": "meta_value"}

	resp := dto.OrganizationResponse{
		ID:          id,
		Name:        "Test Org",
		Description: &desc,
		Email:       "test@example.com",
		Phone:       &phone,
		Settings:    settings,
		Meta:        meta,
		Status:      "active",
		CreatedBy:   "user123",
		UpdatedBy:   "user456",
		CreatedAt:   "2026-02-02T00:00:00Z",
		UpdatedAt:   "2026-02-02T12:00:00Z",
	}

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Name != "Test Org" {
		t.Errorf("Name = %v, want %v", resp.Name, "Test Org")
	}
	if resp.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", resp.Email, "test@example.com")
	}
	if resp.Status != "active" {
		t.Errorf("Status = %v, want %v", resp.Status, "active")
	}
	if *resp.Description != "Test Organization" {
		t.Errorf("Description = %v, want %v", *resp.Description, "Test Organization")
	}
}
