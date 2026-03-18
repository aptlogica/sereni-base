package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewAuthHandler(t *testing.T) {
	handler := handlers.NewAuthHandler(nil)
	assert.NotNil(t, handler)
}

func TestAuthHandler_LoginUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		requestBody  interface{}
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:         "missing_email",
			requestBody:  map[string]string{"password": "password123"},
			mockSetup:    func(m *mocks.MockAuthManagementService) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "missing_password",
			requestBody:  map[string]string{"email": "test@example.com"},
			mockSetup:    func(m *mocks.MockAuthManagementService) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:        "successful login",
			requestBody: map[string]string{"email": "test@example.com", "password": "password123"},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().Login(gomock.Any(), "test@example.com", "password123").Return(
					dto.LoginResponse{
						User:  &dto.UserResponse{Email: "test@example.com"},
						Token: &dto.TokenResponse{AccessToken: "token", RefreshToken: "refresh"},
					}, nil,
				)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.LoginUser(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_LoginUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().Login(gomock.Any(), "test@example.com", "password123").Return(dto.LoginResponse{}, errors.New("login failed"))
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "password123"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.LoginUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_VerifyEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		requestBody  interface{}
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:        "successful verify",
			requestBody: dto.VerifyEmailRequest{Token: "valid-token", OTP: "123456"},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().VerifyEmail(gomock.Any(), gomock.Any()).Return(
					dto.LoginResponse{
						User:  &dto.UserResponse{Email: "test@example.com"},
						Token: &dto.TokenResponse{AccessToken: "token"},
					}, nil,
				)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/verify-email", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.VerifyEmail(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_VerifyEmail_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/verify-email", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.VerifyEmail(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_VerifyEmail_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().VerifyEmail(gomock.Any(), gomock.Any()).Return(dto.LoginResponse{}, errors.New("verify failed"))
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.VerifyEmailRequest{Token: "t", OTP: "123456"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/verify-email", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.VerifyEmail(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ResendOTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		requestBody  interface{}
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:         "invalid json",
			requestBody:  "invalid",
			mockSetup:    func(m *mocks.MockAuthManagementService) {},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:        "successful resend",
			requestBody: dto.ResendOTPRequest{Token: "valid-token"},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().ResendOTP(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "service error",
			requestBody: dto.ResendOTPRequest{Token: "valid-token"},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().ResendOTP(gomock.Any(), gomock.Any()).Return(errors.New("email not found"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/resend-otp", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.ResendOTP(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		requestBody  interface{}
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:         "invalid json",
			requestBody:  "invalid",
			mockSetup:    func(m *mocks.MockAuthManagementService) {},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "successful refresh",
			requestBody: dto.RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().RefreshToken(gomock.Any(), gomock.Any()).Return(
					dto.TokenResponse{AccessToken: "new-access-token"}, nil,
				)
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "service error",
			requestBody: dto.RefreshTokenRequest{
				RefreshToken: "invalid-token",
			},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().RefreshToken(gomock.Any(), gomock.Any()).Return(
					dto.TokenResponse{}, errors.New("invalid refresh token"),
				)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/refresh-token", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.RefreshToken(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_ForgotPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		requestBody  interface{}
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:        "successful request",
			requestBody: dto.ForgotPasswordRequest{Email: "test@example.com"},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().ForgotPassword(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "service error",
			requestBody: dto.ForgotPasswordRequest{Email: "notfound@example.com"},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().ForgotPassword(gomock.Any(), gomock.Any()).Return(errors.New("user not found"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/forgot-password", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.ForgotPassword(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_ForgotPassword_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/forgot", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ForgotPassword(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ResetPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		requestBody  interface{}
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:        "successful reset",
			requestBody: dto.ResetPasswordRequest{Token: "valid-token", NewPassword: "newpass123"},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().ResetPassword(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "service error",
			requestBody: dto.ResetPasswordRequest{Token: "invalid-token", NewPassword: "newpass123"},
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().ResetPassword(gomock.Any(), gomock.Any()).Return(errors.New("invalid token"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/reset-password", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.ResetPassword(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_ResetPassword_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/reset", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ResetPassword(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Health(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health", nil)

	handler.Health(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_HealthLive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health/live", nil)

	handler.HealthLive(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_HealthReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health/ready", nil)

	handler.HealthReady(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ValidateToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		requestBody  string
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:        "successful validation",
			requestBody: `{"token": "valid-token"}`,
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().ValidateToken(gomock.Any(), "valid-token").Return(
					dto.TokenValidationResponse{Valid: true}, nil,
				)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "service error",
			requestBody: `{"token": "invalid-token"}`,
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().ValidateToken(gomock.Any(), "invalid-token").Return(
					dto.TokenValidationResponse{}, errors.New("invalid token"),
				)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/validate-token", bytes.NewBufferString(tt.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.ValidateToken(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_ValidateToken_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/token/validate", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ValidateToken(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_VerifyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		token        string
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:  "successful verify",
			token: "valid-token",
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().VerifyToken(gomock.Any(), "valid-token").Return(
					dto.TokenValidationResponse{Valid: true}, nil,
				)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:  "service error",
			token: "invalid-token",
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().VerifyToken(gomock.Any(), "invalid-token").Return(
					dto.TokenValidationResponse{}, errors.New("invalid token"),
				)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			body := bytes.NewBufferString(`{"token":"` + tt.token + `"}`)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/verify-token", body)
			c.Request.Header.Set("Content-Type", "application/json")

			handler.VerifyToken(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_VerifyToken_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/token/verify", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.VerifyToken(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Logout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		requestBody  string
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:        "successful logout",
			requestBody: `{"token": "valid-token"}`,
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().Logout(gomock.Any(), "valid-token").Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:        "service error",
			requestBody: `{"token": "invalid-token"}`,
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().Logout(gomock.Any(), "invalid-token").Return(errors.New("logout failed"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/logout", bytes.NewBufferString(tt.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Logout(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_Logout_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/logout", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Logout(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_GetUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		schema       string
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:   "successful get users",
			schema: "test_schema",
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().GetUsers(gomock.Any(), "test_schema").Return(
					[]dto.UserWithRole{{Email: "test@example.com"}}, nil,
				)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:   "service error",
			schema: "test_schema",
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().GetUsers(gomock.Any(), "test_schema").Return(
					nil, errors.New("database error"),
				)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/users", nil)
			c.Set("schema", tt.schema)

			handler.GetUsers(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestAuthHandler_GetActiveUsersForAssign(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		schema       string
		mockSetup    func(*mocks.MockAuthManagementService)
		expectedCode int
	}{
		{
			name:   "successful get active users",
			schema: "test_schema",
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().GetActiveUsersForAssign(gomock.Any(), "test_schema").Return(
					[]dto.UserWithRole{{Email: "test@example.com"}}, nil,
				)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:   "service error",
			schema: "test_schema",
			mockSetup: func(m *mocks.MockAuthManagementService) {
				m.EXPECT().GetActiveUsersForAssign(gomock.Any(), "test_schema").Return(
					nil, errors.New("fetch failed"),
				)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockAuthManagementService(ctrl)
			tt.mockSetup(mockService)
			handler := handlers.NewAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/users/active", nil)
			c.Set("schema", tt.schema)

			handler.GetActiveUsersForAssign(c)
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

// Test for UpdateUserAccess - currently at 0% coverage
func TestAuthHandler_UpdateUserAccess_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().AssignUserToWorkspace(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	handler := handlers.NewAuthHandler(mockService)

	requestBody := dto.CreateMemberRequest{
		UserID: uuid.New().String(),
		Membership: []dto.MembershipRequest{
			{WorkspaceID: uuid.New().String(), Role: "editor"},
		},
	}
	body, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/access/user", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", uuid.New().String())

	handler.UpdateUserAccess(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_UpdateUserAccess_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/access/user", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateUserAccess(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_UpdateUserAccess_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().AssignUserToWorkspace(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("update failed"))
	handler := handlers.NewAuthHandler(mockService)

	requestBody := dto.CreateMemberRequest{
		UserID: uuid.New().String(),
		Membership: []dto.MembershipRequest{
			{WorkspaceID: uuid.New().String(), Role: "editor"},
		},
	}
	body, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/access/user", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", uuid.New().String())

	handler.UpdateUserAccess(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthHandler_AddUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().AddUser(gomock.Any(), "test", gomock.Any(), "user123").Return(tenant.User{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("email", "test@example.com")
	_ = writer.WriteField("firstname", "Test")
	_ = writer.WriteField("lastname", "User")
	_ = writer.WriteField("membership", `[{"workspace_id":"w1","role":"Admin","bases":[{"base_id":"b1","role":"Editor"}]}]`)
	part, _ := writer.CreateFormFile("profile_pic", "avatar.png")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.AddUser(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuthHandler_AddUser_InvalidMembership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("email", "test@example.com")
	_ = writer.WriteField("membership", "invalid")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.AddUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_AddUser_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("firstname", "Test")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.AddUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_EditUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().EditUser(gomock.Any(), "test", gomock.Any(), "user123").Return(dto.UserResponse{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("user_id", "u1")
	_ = writer.WriteField("firstname", "Updated")
	_ = writer.WriteField("lastname", "User")
	_ = writer.WriteField("is_coowner", "true")
	_ = writer.WriteField("membership", `[{"workspace_id":"w1","role":"Admin","bases":[{"base_id":"b1","role":"Editor"}]}]`)
	part, _ := writer.CreateFormFile("profile_pic", "avatar.png")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/edit", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.EditUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_EditUser_CoOwnerCaseInsensitive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	// Verify that EditUser is called with IsCoOwner set to true regardless of input case
	mockService.EXPECT().EditUser(gomock.Any(), "test", gomock.MatcherFunc(func(x interface{}) bool {
		req, ok := x.(dto.EditUserRequest)
		return ok && req.IsCoOwner != nil && *req.IsCoOwner == true
	}), "user123").Return(dto.UserResponse{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	testCases := []string{"true", "True", "TRUE", "1"}
	for _, isCoOwnerVal := range testCases {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("user_id", "u1")
		_ = writer.WriteField("is_coowner", isCoOwnerVal)
		_ = writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/users/edit", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		c.Set("schema", "test")
		c.Set("user_id", "user123")

		handler.EditUser(c)
		assert.Equal(t, http.StatusOK, w.Code, "should accept is_coowner value: %s", isCoOwnerVal)
	}
}

func TestAuthHandler_EditUser_CoOwnerFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().EditUser(gomock.Any(), "test", gomock.MatcherFunc(func(x interface{}) bool {
		req, ok := x.(dto.EditUserRequest)
		return ok && req.IsCoOwner != nil && *req.IsCoOwner == false
	}), "user123").Return(dto.UserResponse{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("user_id", "u1")
	_ = writer.WriteField("is_coowner", "false")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/edit", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.EditUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_EditUser_MissingUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("firstname", "Updated")
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/edit", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	handler.EditUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().DeleteUserCompletely(gomock.Any(), "test", "u1").Return(nil)
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.RemoveUserRequest{UserID: "u1"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/remove", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.RemoveUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveUser_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/remove", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RemoveUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().DeleteUserCompletely(gomock.Any(), "test", "u1").Return(errors.New("delete failed"))
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.RemoveUserRequest{UserID: "u1"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/remove", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.RemoveUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_AssignUserToWorkspace_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().AssignUserToWorkspace(gomock.Any(), "test", gomock.Any(), "user123").Return(nil)
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.CreateMemberRequest{
		UserID:     "u1",
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "Admin"}},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/assign", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.AssignUserToWorkspace(c)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuthHandler_AssignUserToWorkspace_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().AssignUserToWorkspace(gomock.Any(), "test", gomock.Any(), "user123").Return(errors.New("assign failed"))
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.CreateMemberRequest{
		UserID:     "u1",
		Membership: []dto.MembershipRequest{{WorkspaceID: "w1", Role: "Admin"}},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/assign", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.AssignUserToWorkspace(c)
	assert.NotEqual(t, http.StatusCreated, w.Code)
}

func TestAuthHandler_AssignUserToWorkspace_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/assign", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.AssignUserToWorkspace(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveUserFromWorkspace_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().RemoveUserFromWorkspace(gomock.Any(), "test", "w1", "u1", "user123").Return(nil)
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.RemoveMemberRequest{UserID: "u1"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/workspaces/w1/remove", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.RemoveUserFromWorkspace(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveUserFromWorkspace_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/workspaces/w1/remove", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "w1"}}

	handler.RemoveUserFromWorkspace(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveUserFromBase_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().RemoveUserFromBase(gomock.Any(), "test", "b1", "u1", "user123").Return(nil)
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.RemoveMemberRequest{UserID: "u1"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bases/b1/remove", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.RemoveUserFromBase(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveUserFromBase_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/bases/b1/remove", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "b1"}}

	handler.RemoveUserFromBase(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_GetWorkspaceMembers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().GetWorkspaceMembers(gomock.Any(), "test", "w1").Return([]dto.WorkspaceMemberResponse{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/w1/members", nil)
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")

	handler.GetWorkspaceMembers(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_GetBaseMembers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().GetBaseMembers(gomock.Any(), "test", "b1").Return([]dto.WorkspaceMemberResponse{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/bases/b1/members", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")

	handler.GetBaseMembers(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_GetWorkspaceMembersWithRole_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().GetWorkspaceMembersWithRole(gomock.Any(), "test", "w1").Return([]dto.UserWithRole{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/workspaces/w1/members-role", nil)
	c.Params = gin.Params{{Key: "id", Value: "w1"}}
	c.Set("schema", "test")

	handler.GetWorkspaceMembersWithRole(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_GetBaseMembersWithRole_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().GetBaseMembersWithRole(gomock.Any(), "test", "b1").Return([]dto.UserWithRole{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/bases/b1/members-role", nil)
	c.Params = gin.Params{{Key: "id", Value: "b1"}}
	c.Set("schema", "test")

	handler.GetBaseMembersWithRole(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_UpdatePassword_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().UpdatePassword(gomock.Any(), "test", "u1", gomock.Any()).Return(nil)
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.UpdateUserPasswordRequest{OldPassword: "oldpass", NewPassword: "newpass123"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/u1/password", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "u1"}}
	c.Set("schema", "test")

	handler.UpdatePassword(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_UpdatePassword_MissingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	body, _ := json.Marshal(dto.UpdateUserPasswordRequest{OldPassword: "oldpass", NewPassword: "newpass123"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users//password", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.UpdatePassword(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_UpdatePassword_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/u1/password", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "u1"}}

	handler.UpdatePassword(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ActivateUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().ActivateUser(gomock.Any(), "test", "u1").Return(dto.UserResponse{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.ActivateUserRequest{UserID: "u1"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/activate", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.ActivateUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ActivateUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().ActivateUser(gomock.Any(), "test", "u1").Return(dto.UserResponse{}, errors.New("activate failed"))
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.ActivateUserRequest{UserID: "u1"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/activate", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.ActivateUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_ActivateUser_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/activate", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ActivateUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_DeactivateUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().DeactivateUser(gomock.Any(), "test", "u1").Return(dto.UserResponse{}, nil)
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.DeactivateUserRequest{UserID: "u1"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/deactivate", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.DeactivateUser(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_DeactivateUser_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().DeactivateUser(gomock.Any(), "test", "u1").Return(dto.UserResponse{}, errors.New("deactivate failed"))
	handler := handlers.NewAuthHandler(mockService)

	body, _ := json.Marshal(dto.DeactivateUserRequest{UserID: "u1"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/deactivate", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")

	handler.DeactivateUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_DeactivateUser_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/users/deactivate", bytes.NewBufferString("invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.DeactivateUser(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveAccessMemberByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockAuthManagementService(ctrl)
	mockService.EXPECT().RemoveAccessMemberByID(gomock.Any(), "test", "am1", "user123").Return(nil)
	handler := handlers.NewAuthHandler(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/access/am1", nil)
	c.Params = gin.Params{{Key: "id", Value: "am1"}}
	c.Set("schema", "test")
	c.Set("user_id", "user123")

	handler.RemoveAccessMemberByID(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_RemoveAccessMemberByID_MissingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewAuthHandler(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/access/", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	handler.RemoveAccessMemberByID(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
