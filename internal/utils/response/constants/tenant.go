// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var TenantError = struct {
	TenantNotFound       ResponseCode
	TenantAlreadyExists  ResponseCode
	TenantNotCreated     ResponseCode
	TenantNotUpdated     ResponseCode
	TenantNotDeleted     ResponseCode
	MembershipNotCreated ResponseCode
	MembershipNotFound   ResponseCode
}{
	TenantNotFound:       "TNT_3001",
	TenantAlreadyExists:  "TNT_3002",
	TenantNotCreated:     "TNT_3003",
	TenantNotUpdated:     "TNT_3004",
	TenantNotDeleted:     "TNT_3005",
	MembershipNotCreated: "TNT_3007",
	MembershipNotFound:   "TNT_3009",
}

var TenantErrorCodes = map[ResponseCode]MetaResponse{
	TenantError.TenantNotFound:       CreateMetaResponse(http.StatusNotFound, "Tenant not found", "The specified tenant could not be found"),
	TenantError.TenantAlreadyExists:  CreateMetaResponse(http.StatusConflict, "Tenant already exists", "A tenant with the given information already exists"),
	TenantError.TenantNotCreated:     CreateMetaResponse(http.StatusInternalServerError, "Tenant not created", "The tenant could not be created due to an internal error"),
	TenantError.TenantNotUpdated:     CreateMetaResponse(http.StatusInternalServerError, "Tenant not updated", "The tenant could not be updated due to an internal error"),
	TenantError.TenantNotDeleted:     CreateMetaResponse(http.StatusInternalServerError, "Tenant not deleted", "The tenant could not be deleted due to an internal error"),
	TenantError.MembershipNotCreated: CreateMetaResponse(http.StatusInternalServerError, "Tenant membership not created", "The tenant membership could not be created due to an internal error"),
	TenantError.MembershipNotFound:   CreateMetaResponse(http.StatusNotFound, "Tenant membership not found", "The specified tenant membership could not be found"),
}

var TenantSuccess = struct {
	TenantCreated     ResponseCode
	TenantUpdated     ResponseCode
	TenantDeleted     ResponseCode
	MembershipCreated ResponseCode
	TenantFetched     ResponseCode
}{
	TenantCreated:     "TNT_SUCCESS_3001",
	TenantUpdated:     "TNT_SUCCESS_3002",
	TenantDeleted:     "TNT_SUCCESS_3003",
	MembershipCreated: "TNT_SUCCESS_3005",
	TenantFetched:     "TNT_SUCCESS_3006",
}

var TenantSuccessCodes = map[ResponseCode]MetaResponse{
	TenantSuccess.TenantCreated:     CreateMetaResponse(http.StatusCreated, "Tenant created successfully", "The tenant has been created successfully"),
	TenantSuccess.TenantUpdated:     CreateMetaResponse(http.StatusOK, "Tenant updated successfully", "The tenant has been updated successfully"),
	TenantSuccess.TenantDeleted:     CreateMetaResponse(http.StatusOK, "Tenant deleted successfully", "The tenant has been deleted successfully"),
	TenantSuccess.MembershipCreated: CreateMetaResponse(http.StatusCreated, "Tenant membership created successfully", "The tenant membership has been created successfully"),
	TenantSuccess.TenantFetched:     CreateMetaResponse(http.StatusOK, "Tenant fetched successfully", "The tenant has been fetched successfully"),
}
