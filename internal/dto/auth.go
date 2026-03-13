// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import "github.com/google/uuid"

type RegisterRequest struct {
	ID            uuid.UUID `json:"id," format:"uuid"`
	Email         string    `json:"email" binding:"required,email" example:"johndoe@example.com" format:"email"`
	FirstName     string    `json:"first_name" binding:"required" example:"John" format:"string"`
	LastName      string    `json:"last_name" binding:"required" example:"Doe" format:"string"`
	Password      string    `json:"password" binding:"required,min=8" example:"strongpassword123" format:"string"`
	AuthProvider  string    `json:"auth_provider," example:"local" format:"string"`
	Status        string    `json:"status," example:"active" format:"string"`
	EmailVerified bool      `json:"email_verified," example:"true" format:"bool"`
	DateOfBirth   *string   `json:"dob," example:"17-11-2025" format:"string"`
	Country       string    `json:"country," example:"US" format:"string"`
	Timezone      string    `json:"timezone," example:"UTC" format:"string"`
	Roles         string    `json:"roles," example:"user" format:"string"`
}

type RegisterResponse struct {
	Token string `json:"token" binding:"required" format:"token"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"johndoe@example.com" format:"email"`
	Password string `json:"password" binding:"required,min=8" example:"strongpassword123" format:"string"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required" format:"jwt"`
	OTP   string `json:"otp" binding:"required" example:"123456" format:"string"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token" binding:"required" format:"jwt"`
	RefreshToken string `json:"refresh_token" binding:"required" format:"jwt"`
}

type LoginResponse struct {
	User *UserResponse `json:"user" binding:"required" format:"object"`
	// Tenant    *TenantResponse    `json:"tenant," format:"object"`
	Token *TokenResponse `json:"token" binding:"required" format:"object"`
	// Workspace *WorkspaceResponse `json:"workspace," format:"object"`
}

type ResendOTPRequest struct {
	Token string `json:"token" binding:"required" format:"jwt"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" format:"jwt"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email" example:"johndoe@example.com" format:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required" example:"36a72062-10c7-41d4-8c6f-e4625b211a56" format:"uuid"`
	NewPassword string `json:"new_password" binding:"required,min=8" example:"newstrongpassword456" format:"string"`
}

type TokenValidationRequest struct {
	Token string `json:"token" binding:"required" format:"jwt"`
}

type TokenValidationResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,"`
	Roles  string `json:"roles,"`
}

type LogoutRequest struct {
	Token string `json:"token" binding:"required" format:"jwt"`
}
