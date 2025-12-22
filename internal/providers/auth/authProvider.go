package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	appErrors "serenibase/internal/app-errors"
	"serenibase/internal/config"
	"serenibase/internal/models/master"
	"strings"
)

// Unified response type for all API interactions.
type APIResponse struct {
	Success bool        `json:"success"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func NewAuthProvider(cfg *config.AuthConfig) (AuthProvider, error) {
	return &AuthProviderService{
		AuthConfig: cfg,
	}, nil
}

type AuthProviderService struct {
	AuthConfig *config.AuthConfig
}

// parseAPIResponse handles all responses and returns error, code, and possibly data.
func parseAPIResponse(resp *http.Response) (APIResponse, error) {
	defer resp.Body.Close()

	var respWrap APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&respWrap); err != nil {
		return APIResponse{}, fmt.Errorf("JSON decode failed: %w", err)
	}
	if !respWrap.Success {
		return respWrap, &appErrors.APIError{
			Code:    respWrap.Code,
			Message: respWrap.Message,
			Details: respWrap.Details,
		}
	}
	return respWrap, nil
}

func (a *AuthProviderService) GenerateToken(ctx context.Context, user master.User) (Tokens, error) {
	body, err := json.Marshal(map[string]string{
		"email":    user.Email,
		"password": user.Password,
	})
	if err != nil {
		return Tokens{}, appErrors.ErrJSONMarshal
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/login", a.AuthConfig.URL),
		bytes.NewReader(body),
	)
	if err != nil {
		return Tokens{}, appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Tokens{}, appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return Tokens{}, err
	}

	tokenMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		return Tokens{}, fmt.Errorf("invalid token response format")
	}
	return Tokens{
		AccessToken:  getStringFromMap(tokenMap, "access_token"),
		RefreshToken: getStringFromMap(tokenMap, "refresh_token"),
	}, nil
}

func (a *AuthProviderService) VerifyToken(ctx context.Context, token string) (interface{}, error) {
	body, err := json.Marshal(map[string]string{
		"token": token,
	})
	if err != nil {
		return nil, appErrors.ErrJSONMarshal
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/verify-token", a.AuthConfig.URL),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		if err != nil && strings.Contains(err.Error(), "token has expired") {
			return nil, appErrors.AuthProviderTokenExpired
		}
		return nil, err
	}
	return apiResp.Data, nil
}

func (a *AuthProviderService) RefreshToken(ctx context.Context, token string) (Tokens, error) {
	body, err := json.Marshal(map[string]string{
		"refresh_token": token,
	})
	if err != nil {
		return Tokens{}, appErrors.ErrJSONMarshal
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/refresh", a.AuthConfig.URL),
		bytes.NewReader(body),
	)
	if err != nil {
		return Tokens{}, appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Tokens{}, appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return Tokens{}, err
	}

	tokenMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		return Tokens{}, fmt.Errorf("invalid token response format")
	}
	return Tokens{
		AccessToken:  getStringFromMap(tokenMap, "access_token"),
		RefreshToken: getStringFromMap(tokenMap, "refresh_token"),
	}, nil
}

// Adjusted to use POST /auth/verify-token with JSON body: { "token": ... }
func (a *AuthProviderService) ValidateToken(ctx context.Context, token string) (Claims, error) {
	// Remove "Bearer " prefix if present in the token
	const bearerPrefix = "Bearer "
	if strings.HasPrefix(token, bearerPrefix) {
		token = strings.TrimSpace(token[len(bearerPrefix):])
	}
	type VerifyTokenRequest struct {
		Token string `json:"token" validate:"required"`
	}

	reqBody := VerifyTokenRequest{
		Token: token,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return Claims{}, appErrors.ErrJSONMarshal
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/validate-token", a.AuthConfig.URL),
		bytes.NewReader(jsonBody),
	)

	if err != nil {
		return Claims{}, appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Claims{}, appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		// Check if the error message contains indication of Keycloak token expiration
		if err != nil && strings.Contains(err.Error(), "expired") {
			return Claims{}, appErrors.TokenExpired
		}
		if err != nil && strings.Contains(err.Error(), "token not active") {
			return Claims{}, appErrors.TokenUnauthorized
		}
		return Claims{}, err
	}
	fmt.Println("apiResp: ", apiResp)

	// The response "data" is expected to be a map[string]interface{} with various fields.
	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		return Claims{}, fmt.Errorf("invalid claims data format")
	}

	// Populate Claims model
	claims := Claims{
		UserId:   getStringFromMap(dataMap, "user_id"),
		TenantId: getStringFromMap(dataMap, "tenant_id"),
		Roles:    getStringFromMap(dataMap, "roles"),
	}

	return claims, nil
}

func (a *AuthProviderService) Ping(ctx context.Context) (interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/auth/ping", a.AuthConfig.URL),
		nil,
	)
	if err != nil {
		return nil, appErrors.ErrHTTPRequestCreation
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return nil, err
	}
	return apiResp.Data, nil
}

func (a *AuthProviderService) AddUser(ctx context.Context, user master.User, tenant_id string, roles string) (Tokens, error) {
	body := map[string]interface{}{
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"password":   user.Password,
		"attributes": map[string]interface{}{
			"tenant_id": tenant_id,
			"roles":     roles,
			"user_id":   user.ID.String(),
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return Tokens{}, appErrors.InvalidPayload
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/register", a.AuthConfig.URL),
		strings.NewReader(string(jsonBody)),
	)

	if err != nil {
		return Tokens{}, appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Tokens{}, appErrors.ErrHTTPDoRequest
	}
	fmt.Println("resp: ", resp)

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return Tokens{}, err
	}

	tokenMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		return Tokens{}, fmt.Errorf("invalid token response format")
	}
	return Tokens{
		AccessToken:  getStringFromMap(tokenMap, "access_token"),
		RefreshToken: getStringFromMap(tokenMap, "refresh_token"),
	}, nil
}

func (a *AuthProviderService) ResetPassword(ctx context.Context, email string, newPassword string) error {
	body := map[string]string{
		"email":        email,
		"new_password": newPassword,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return appErrors.InvalidPayload
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/reset-password", a.AuthConfig.URL),
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return err
	}
	if !apiResp.Success {
		return appErrors.AuthProviderSetPasswordFailed
	}
	return nil
}

func (a *AuthProviderService) HandleCallback(ctx context.Context, code string) (*AuthResult, error) {
	body := map[string]string{
		"code": code,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, appErrors.InvalidPayload
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/callback", a.AuthConfig.URL),
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return nil, appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return nil, err
	}
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal callback data: %w", err)
	}
	var result AuthResult
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal callback data: %w", err)
	}
	return &result, nil
}

func (a *AuthProviderService) AddOrUpdateUserAttributesToKeycloakUser(ctx context.Context, keycloakUserID string, attributes map[string]interface{}) error {
	body := map[string]interface{}{
		"attributes": attributes,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return appErrors.InvalidPayload
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		fmt.Sprintf("%s/auth/users/%s/attributes", a.AuthConfig.URL, keycloakUserID),
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return appErrors.AuthProviderUserCreateFailed
	}
	if !apiResp.Success {
		return appErrors.AuthProviderUserCreateFailed
	}

	return nil
}

func (a *AuthProviderService) SetEmailVerified(ctx context.Context, userID string) error {
	// Prepare payload with user ID
	type SetEmailVerifiedRequest struct {
		UserID string `json:"user_id"`
	}

	body := SetEmailVerifiedRequest{
		UserID: userID,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return appErrors.InvalidPayload
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/set-email-verified", a.AuthConfig.URL),
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return appErrors.ErrHTTPDoRequest
	}

	apiResp, parseErr := parseAPIResponse(resp)
	if parseErr != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to set email verified: %w", parseErr)
	}
	if !apiResp.Success {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to set email verified: operation was not successful")
	}
	return nil
}

func (a *AuthProviderService) CheckUserExistsByEmailAndReturnUser(ctx context.Context, email string) (exists bool, keycloakUserID string, attributes map[string]string, err error) {
	// Prepare request body as JSON
	bodyMap := map[string]string{
		"email": email,
	}
	jsonBody, err := json.Marshal(bodyMap)
	if err != nil {
		return false, "", nil, appErrors.InvalidPayload
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/check-user-exists", a.AuthConfig.URL),
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return false, "", nil, appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "", nil, appErrors.ErrHTTPDoRequest
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, "", nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, "", nil, appErrors.UserNotFound
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return false, "", nil, err
	}
	dataMap, ok := apiResp.Data.(map[string]interface{})
	if !ok {
		return false, "", nil, fmt.Errorf("invalid data for user exists")
	}

	// "attributes" is inside apiResp.Data["attributes"]
	attrs := make(map[string]string)
	if m, ok := dataMap["attributes"].(map[string]interface{}); ok {
		for k, v := range m {
			if str, ok := v.(string); ok {
				attrs[k] = str
			}
		}
	}

	// keycloak user ID is apiResp.Data["id"]
	kcID := getStringFromMap(dataMap, "id")
	return true, kcID, attrs, nil
}

func (a *AuthProviderService) GetProviderURL(provider string) string {
	return fmt.Sprintf("%s/auth/provider/%s", a.AuthConfig.URL, provider)
}

func (a *AuthProviderService) Logout(ctx context.Context, refreshToken string) error {
	body := map[string]string{
		"refresh_token": refreshToken,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return appErrors.InvalidPayload
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/logout", a.AuthConfig.URL),
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return appErrors.ErrHTTPRequestCreation
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return appErrors.AuthProviderLoginFailed
	}
	if !apiResp.Success {
		return appErrors.AuthProviderLoginFailed
	}
	return nil
}

// Helper to extract string value from map[string]interface{}
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func (a *AuthProviderService) DisableUser(ctx context.Context, keycloakUserID string) error {
	body := map[string]string{
		"user_id": keycloakUserID,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return appErrors.InvalidPayload
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/disable-user", a.AuthConfig.URL),
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return appErrors.ErrHTTPRequestCreation
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return appErrors.UserNotFound
	}
	if !apiResp.Success {
		return appErrors.UserNotFound
	}
	return nil
}

func (a *AuthProviderService) EnableUser(ctx context.Context, keycloakUserID string) error {
	body := map[string]string{
		"user_id": keycloakUserID,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return appErrors.InvalidPayload
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/auth/enable-user", a.AuthConfig.URL),
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return appErrors.ErrHTTPRequestCreation
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return appErrors.UserNotFound
	}
	if !apiResp.Success {
		return appErrors.UserNotFound
	}
	return nil
}

func (a *AuthProviderService) DeleteUser(ctx context.Context, keycloakUserID string) error {
	body := map[string]string{
		"user_id": keycloakUserID,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return appErrors.InvalidPayload
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/auth/delete-user", a.AuthConfig.URL),
		strings.NewReader(string(jsonBody)),
	)
	if err != nil {
		return appErrors.ErrHTTPRequestCreation
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return appErrors.ErrHTTPDoRequest
	}

	apiResp, err := parseAPIResponse(resp)
	if err != nil {
		return appErrors.UserNotFound
	}
	if !apiResp.Success {
		return appErrors.UserNotFound
	}
	return nil
}
