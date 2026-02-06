package constants

import "net/http"

var TenantError = struct {
	TenantNotFound         ResponseCode
	TenantAlreadyExists    ResponseCode
	TenantNotCreated       ResponseCode
	TenantNotUpdated       ResponseCode
	TenantNotDeleted       ResponseCode
	SubscriptionNotCreated ResponseCode
	MembershipNotCreated   ResponseCode
	SubscriptionNotFound   ResponseCode
	MembershipNotFound     ResponseCode
}{
	TenantNotFound:         "TNT_3001",
	TenantAlreadyExists:    "TNT_3002",
	TenantNotCreated:       "TNT_3003",
	TenantNotUpdated:       "TNT_3004",
	TenantNotDeleted:       "TNT_3005",
	SubscriptionNotCreated: "TNT_3006",
	MembershipNotCreated:   "TNT_3007",
	SubscriptionNotFound:   "TNT_3008",
	MembershipNotFound:     "TNT_3009",
}

var TenantErrorCodes = map[ResponseCode]MetaResponse{
	TenantError.TenantNotFound:         CreateMetaResponse(http.StatusNotFound, "Tenant not found", "The specified tenant could not be found"),
	TenantError.TenantAlreadyExists:    CreateMetaResponse(http.StatusConflict, "Tenant already exists", "A tenant with the given information already exists"),
	TenantError.TenantNotCreated:       CreateMetaResponse(http.StatusInternalServerError, "Tenant not created", "The tenant could not be created due to an internal error"),
	TenantError.TenantNotUpdated:       CreateMetaResponse(http.StatusInternalServerError, "Tenant not updated", "The tenant could not be updated due to an internal error"),
	TenantError.TenantNotDeleted:       CreateMetaResponse(http.StatusInternalServerError, "Tenant not deleted", "The tenant could not be deleted due to an internal error"),
	TenantError.SubscriptionNotCreated: CreateMetaResponse(http.StatusInternalServerError, "Tenant subscription not created", "The tenant subscription could not be created due to an internal error"),
	TenantError.MembershipNotCreated:   CreateMetaResponse(http.StatusInternalServerError, "Tenant membership not created", "The tenant membership could not be created due to an internal error"),
	TenantError.SubscriptionNotFound:   CreateMetaResponse(http.StatusNotFound, "Tenant subscription not found", "The specified tenant subscription could not be found"),
	TenantError.MembershipNotFound:     CreateMetaResponse(http.StatusNotFound, "Tenant membership not found", "The specified tenant membership could not be found"),
}

var TenantSuccess = struct {
	TenantCreated       ResponseCode
	TenantUpdated       ResponseCode
	TenantDeleted       ResponseCode
	SubscriptionCreated ResponseCode
	MembershipCreated   ResponseCode
	TenantFetched       ResponseCode
}{
	TenantCreated:       "TNT_SUCCESS_3001",
	TenantUpdated:       "TNT_SUCCESS_3002",
	TenantDeleted:       "TNT_SUCCESS_3003",
	SubscriptionCreated: "TNT_SUCCESS_3004",
	MembershipCreated:   "TNT_SUCCESS_3005",
	TenantFetched:       "TNT_SUCCESS_3006",
}

var TenantSuccessCodes = map[ResponseCode]MetaResponse{
	TenantSuccess.TenantCreated:       CreateMetaResponse(http.StatusCreated, "Tenant created successfully", "The tenant has been created successfully"),
	TenantSuccess.TenantUpdated:       CreateMetaResponse(http.StatusOK, "Tenant updated successfully", "The tenant has been updated successfully"),
	TenantSuccess.TenantDeleted:       CreateMetaResponse(http.StatusOK, "Tenant deleted successfully", "The tenant has been deleted successfully"),
	TenantSuccess.SubscriptionCreated: CreateMetaResponse(http.StatusCreated, "Tenant subscription created successfully", "The tenant subscription has been created successfully"),
	TenantSuccess.MembershipCreated:   CreateMetaResponse(http.StatusCreated, "Tenant membership created successfully", "The tenant membership has been created successfully"),
	TenantSuccess.TenantFetched:       CreateMetaResponse(http.StatusOK, "Tenant fetched successfully", "The tenant has been fetched successfully"),
}
