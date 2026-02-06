package tests

import (
	"serenibase/internal/dto"
	"testing"

	"github.com/google/uuid"
)

func TestRegisterRequestFields(t *testing.T) {
	id := uuid.New()
	dob := "17-11-2025"

	req := dto.RegisterRequest{
		ID:            id,
		Email:         "test@example.com",
		FirstName:     "John",
		LastName:      "Doe",
		Password:      "password123",
		AuthProvider:  "local",
		Status:        "active",
		EmailVerified: true,
		DateOfBirth:   &dob,
		Country:       "US",
		Timezone:      "UTC",
		Roles:         "user",
	}

	if req.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", req.Email, "test@example.com")
	}
	if req.FirstName != "John" {
		t.Errorf("FirstName = %v, want %v", req.FirstName, "John")
	}
	if req.LastName != "Doe" {
		t.Errorf("LastName = %v, want %v", req.LastName, "Doe")
	}
	if req.Password != "password123" {
		t.Errorf("Password = %v, want %v", req.Password, "password123")
	}
}

func TestLoginRequestFields(t *testing.T) {
	req := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	if req.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", req.Email, "test@example.com")
	}
	if req.Password != "password123" {
		t.Errorf("Password = %v, want %v", req.Password, "password123")
	}
}

func TestTokenResponseFields(t *testing.T) {
	resp := dto.TokenResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}

	if resp.AccessToken != "access-token" {
		t.Errorf("AccessToken = %v, want %v", resp.AccessToken, "access-token")
	}
	if resp.RefreshToken != "refresh-token" {
		t.Errorf("RefreshToken = %v, want %v", resp.RefreshToken, "refresh-token")
	}
}

func TestForgotPasswordRequestFields(t *testing.T) {
	req := dto.ForgotPasswordRequest{
		Email: "test@example.com",
	}

	if req.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", req.Email, "test@example.com")
	}
}

func TestResetPasswordRequestFields(t *testing.T) {
	req := dto.ResetPasswordRequest{
		Token:       "reset-token",
		NewPassword: "newpassword123",
	}

	if req.Token != "reset-token" {
		t.Errorf("Token = %v, want %v", req.Token, "reset-token")
	}
	if req.NewPassword != "newpassword123" {
		t.Errorf("NewPassword = %v, want %v", req.NewPassword, "newpassword123")
	}
}

func TestVerifyEmailRequestFields(t *testing.T) {
	req := dto.VerifyEmailRequest{
		Token: "verify-token",
		OTP:   "123456",
	}

	if req.Token != "verify-token" {
		t.Errorf("Token = %v, want %v", req.Token, "verify-token")
	}
	if req.OTP != "123456" {
		t.Errorf("OTP = %v, want %v", req.OTP, "123456")
	}
}

func TestRegisterResponseFields(t *testing.T) {
	resp := dto.RegisterResponse{
		Token: "register-token",
	}

	if resp.Token != "register-token" {
		t.Errorf("Token = %v, want %v", resp.Token, "register-token")
	}
}

func TestLoginResponseFields(t *testing.T) {
	user := &dto.UserResponse{
		Email: "test@example.com",
	}
	token := &dto.TokenResponse{
		AccessToken:  "access",
		RefreshToken: "refresh",
	}

	resp := dto.LoginResponse{
		User:  user,
		Token: token,
	}

	if resp.User.Email != "test@example.com" {
		t.Errorf("User.Email = %v, want %v", resp.User.Email, "test@example.com")
	}
	if resp.Token.AccessToken != "access" {
		t.Errorf("Token.AccessToken = %v, want %v", resp.Token.AccessToken, "access")
	}
}

func TestResendOTPRequestFields(t *testing.T) {
	req := dto.ResendOTPRequest{
		Token: "resend-token",
	}

	if req.Token != "resend-token" {
		t.Errorf("Token = %v, want %v", req.Token, "resend-token")
	}
}

func TestRefreshTokenRequestFields(t *testing.T) {
	req := dto.RefreshTokenRequest{
		RefeshToken: "refresh-token",
	}

	if req.RefeshToken != "refresh-token" {
		t.Errorf("RefeshToken = %v, want %v", req.RefeshToken, "refresh-token")
	}
}

func TestTokenValidationRequestFields(t *testing.T) {
	req := dto.TokenValidationRequest{
		Token: "validation-token",
	}

	if req.Token != "validation-token" {
		t.Errorf("Token = %v, want %v", req.Token, "validation-token")
	}
}

func TestTokenValidationResponseFields(t *testing.T) {
	resp := dto.TokenValidationResponse{
		Valid:  true,
		UserID: "user123",
		Roles:  "admin",
	}

	if !resp.Valid {
		t.Errorf("Valid = %v, want %v", resp.Valid, true)
	}
	if resp.UserID != "user123" {
		t.Errorf("UserID = %v, want %v", resp.UserID, "user123")
	}
	if resp.Roles != "admin" {
		t.Errorf("Roles = %v, want %v", resp.Roles, "admin")
	}
}

func TestLogoutRequestFields(t *testing.T) {
	req := dto.LogoutRequest{
		Token: "logout-token",
	}

	if req.Token != "logout-token" {
		t.Errorf("Token = %v, want %v", req.Token, "logout-token")
	}
}
