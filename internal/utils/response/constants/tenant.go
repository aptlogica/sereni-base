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
	TenantError.TenantNotFound:         createMetaResponse(http.StatusNotFound, "Tenant not found", "The specified tenant could not be found"),
	TenantError.TenantAlreadyExists:    createMetaResponse(http.StatusConflict, "Tenant already exists", "A tenant with the given information already exists"),
	TenantError.TenantNotCreated:       createMetaResponse(http.StatusInternalServerError, "Tenant not created", "The tenant could not be created due to an internal error"),
	TenantError.TenantNotUpdated:       createMetaResponse(http.StatusInternalServerError, "Tenant not updated", "The tenant could not be updated due to an internal error"),
	TenantError.TenantNotDeleted:       createMetaResponse(http.StatusInternalServerError, "Tenant not deleted", "The tenant could not be deleted due to an internal error"),
	TenantError.SubscriptionNotCreated: createMetaResponse(http.StatusInternalServerError, "Tenant subscription not created", "The tenant subscription could not be created due to an internal error"),
	TenantError.MembershipNotCreated:   createMetaResponse(http.StatusInternalServerError, "Tenant membership not created", "The tenant membership could not be created due to an internal error"),
	TenantError.SubscriptionNotFound:   createMetaResponse(http.StatusNotFound, "Tenant subscription not found", "The specified tenant subscription could not be found"),
	TenantError.MembershipNotFound:     createMetaResponse(http.StatusNotFound, "Tenant membership not found", "The specified tenant membership could not be found"),
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
	TenantSuccess.TenantCreated:       createMetaResponse(http.StatusCreated, "Tenant created successfully", "The tenant has been created successfully"),
	TenantSuccess.TenantUpdated:       createMetaResponse(http.StatusOK, "Tenant updated successfully", "The tenant has been updated successfully"),
	TenantSuccess.TenantDeleted:       createMetaResponse(http.StatusOK, "Tenant deleted successfully", "The tenant has been deleted successfully"),
	TenantSuccess.SubscriptionCreated: createMetaResponse(http.StatusCreated, "Tenant subscription created successfully", "The tenant subscription has been created successfully"),
	TenantSuccess.MembershipCreated:   createMetaResponse(http.StatusCreated, "Tenant membership created successfully", "The tenant membership has been created successfully"),
	TenantSuccess.TenantFetched:       createMetaResponse(http.StatusOK, "Tenant fetched successfully", "The tenant has been fetched successfully"),
}
