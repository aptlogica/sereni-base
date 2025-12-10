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
	TenantError.TenantNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Tenant not found",
		Description: "The specified tenant could not be found",
	},
	TenantError.TenantAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Tenant already exists",
		Description: "A tenant with the given information already exists",
	},
	TenantError.TenantNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Tenant not created",
		Description: "The tenant could not be created due to an internal error",
	},
	TenantError.TenantNotUpdated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Tenant not updated",
		Description: "The tenant could not be updated due to an internal error",
	},
	TenantError.TenantNotDeleted: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Tenant not deleted",
		Description: "The tenant could not be deleted due to an internal error",
	},
	TenantError.SubscriptionNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Tenant subscription not created",
		Description: "The tenant subscription could not be created due to an internal error",
	},
	TenantError.MembershipNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Tenant membership not created",
		Description: "The tenant membership could not be created due to an internal error",
	},
	TenantError.SubscriptionNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Tenant subscription not found",
		Description: "The specified tenant subscription could not be found",
	},
	TenantError.MembershipNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Tenant membership not found",
		Description: "The specified tenant membership could not be found",
	},
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
	TenantSuccess.TenantCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Tenant created successfully",
		Description: "The tenant has been created successfully",
	},
	TenantSuccess.TenantUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Tenant updated successfully",
		Description: "The tenant has been updated successfully",
	},
	TenantSuccess.TenantDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Tenant deleted successfully",
		Description: "The tenant has been deleted successfully",
	},
	TenantSuccess.SubscriptionCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Tenant subscription created successfully",
		Description: "The tenant subscription has been created successfully",
	},
	TenantSuccess.MembershipCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Tenant membership created successfully",
		Description: "The tenant membership has been created successfully",
	},
	TenantSuccess.TenantFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Tenant fetched successfully",
		Description: "The tenant has been fetched successfully",
	},
}
