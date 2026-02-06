package constants

import "net/http"

var SubscriptionPlanError = struct {
	PlanNotFound      ResponseCode
	PlanAlreadyExists ResponseCode
	PlanNotCreated    ResponseCode
	PlanNotUpdated    ResponseCode
	PlanNotDeleted    ResponseCode
}{
	PlanNotFound:      "SUB_5001",
	PlanAlreadyExists: "SUB_5002",
	PlanNotCreated:    "SUB_5003",
	PlanNotUpdated:    "SUB_5004",
	PlanNotDeleted:    "SUB_5005",
}

var SubscriptionPlanErrorCodes = map[ResponseCode]MetaResponse{
	SubscriptionPlanError.PlanNotFound:      CreateMetaResponse(http.StatusNotFound, "Subscription plan not found", "The specified subscription plan could not be found"),
	SubscriptionPlanError.PlanAlreadyExists: CreateMetaResponse(http.StatusConflict, "Subscription plan already exists", "A subscription plan with the given information already exists"),
	SubscriptionPlanError.PlanNotCreated:    CreateMetaResponse(http.StatusInternalServerError, "Subscription plan not created", "The subscription plan could not be created due to an internal error"),
	SubscriptionPlanError.PlanNotUpdated:    CreateMetaResponse(http.StatusInternalServerError, "Subscription plan not updated", "The subscription plan could not be updated due to an internal error"),
	SubscriptionPlanError.PlanNotDeleted:    CreateMetaResponse(http.StatusInternalServerError, "Subscription plan not deleted", "The subscription plan could not be deleted due to an internal error"),
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
	SubscriptionPlanSuccess.PlanCreated: CreateMetaResponse(http.StatusCreated, "Subscription plan created successfully", "The subscription plan has been created successfully"),
	SubscriptionPlanSuccess.PlanUpdated: CreateMetaResponse(http.StatusOK, "Subscription plan updated successfully", "The subscription plan has been updated successfully"),
	SubscriptionPlanSuccess.PlanDeleted: CreateMetaResponse(http.StatusOK, "Subscription plan deleted successfully", "The subscription plan has been deleted successfully"),
}
