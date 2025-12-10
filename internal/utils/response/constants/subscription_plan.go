package constants

import "net/http"

var SubscriptionPlanError = struct {
	PlanNotFound        ResponseCode
	PlanAlreadyExists   ResponseCode
	PlanNotCreated      ResponseCode
	PlanNotUpdated      ResponseCode
	PlanNotDeleted      ResponseCode
}{
	PlanNotFound:      "SUB_5001",
	PlanAlreadyExists: "SUB_5002",
	PlanNotCreated:    "SUB_5003",
	PlanNotUpdated:    "SUB_5004",
	PlanNotDeleted:    "SUB_5005",
}

var SubscriptionPlanErrorCodes = map[ResponseCode]MetaResponse{
	SubscriptionPlanError.PlanNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Subscription plan not found",
		Description: "The specified subscription plan could not be found",
	},
	SubscriptionPlanError.PlanAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Subscription plan already exists",
		Description: "A subscription plan with the given information already exists",
	},
	SubscriptionPlanError.PlanNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Subscription plan not created",
		Description: "The subscription plan could not be created due to an internal error",
	},
	SubscriptionPlanError.PlanNotUpdated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Subscription plan not updated",
		Description: "The subscription plan could not be updated due to an internal error",
	},
	SubscriptionPlanError.PlanNotDeleted: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Subscription plan not deleted",
		Description: "The subscription plan could not be deleted due to an internal error",
	},
}

var SubscriptionPlanSuccess = struct {
	PlanCreated ResponseCode
	PlanUpdated ResponseCode
	PlanDeleted ResponseCode
}{
	PlanCreated: "SUB_SUCCESS_5001",
	PlanUpdated: "SUB_SUCCESS_5002",
	PlanDeleted: "SUB_SUCCESS_5003",
}

var SubscriptionPlanSuccessCodes = map[ResponseCode]MetaResponse{
	SubscriptionPlanSuccess.PlanCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Subscription plan created successfully",
		Description: "The subscription plan has been created successfully",
	},
	SubscriptionPlanSuccess.PlanUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Subscription plan updated successfully",
		Description: "The subscription plan has been updated successfully",
	},
	SubscriptionPlanSuccess.PlanDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Subscription plan deleted successfully",
		Description: "The subscription plan has been deleted successfully",
	},
}
