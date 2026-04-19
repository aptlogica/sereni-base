// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import "github.com/google/uuid"

// CreateOrganizationRequest for creating a new organization
type CreateOrganizationRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description *string                `json:"description"`
	Email       string                 `json:"email" binding:"required,email"`
	Phone       *string                `json:"phone"`
	Website     *string                `json:"website"`
	Logo        *string                `json:"logo"`
	Address     *string                `json:"address"`
	City        *string                `json:"city"`
	State       *string                `json:"state"`
	Country     *string                `json:"country"`
	ZipCode     *string                `json:"zip_code"`
	Settings    map[string]interface{} `json:"settings"`
	Meta        map[string]interface{} `json:"meta"`
}

// UpdateOrganizationRequest for updating an organization
type UpdateOrganizationRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Email       *string                `json:"email"`
	Phone       *string                `json:"phone"`
	Website     *string                `json:"website"`
	Logo        *string                `json:"logo"`
	Address     *string                `json:"address"`
	City        *string                `json:"city"`
	State       *string                `json:"state"`
	Country     *string                `json:"country"`
	ZipCode     *string                `json:"zip_code"`
	Settings    map[string]interface{} `json:"settings"`
	Meta        map[string]interface{} `json:"meta"`
	Status      *string                `json:"status"`
}

// OrganizationResponse for API responses
type OrganizationResponse struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	Email       string                 `json:"email"`
	Phone       *string                `json:"phone"`
	Website     *string                `json:"website"`
	Logo        *string                `json:"logo"`
	Address     *string                `json:"address"`
	City        *string                `json:"city"`
	State       *string                `json:"state"`
	Country     *string                `json:"country"`
	ZipCode     *string                `json:"zip_code"`
	Settings    map[string]interface{} `json:"settings"`
	Meta        map[string]interface{} `json:"meta"`
	Status      string                 `json:"status"`
	CreatedBy   string                 `json:"created_by"`
	UpdatedBy   string                 `json:"last_modified_by"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}
