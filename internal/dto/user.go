// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"mime/multipart"
	"time"

	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
)

type UserResponse struct {
	ID           uuid.UUID   `db:"id" json:"id" mapstructure:"id"`
	Email        string      `db:"email" json:"email" mapstructure:"email"`
	FirstName    string      `db:"first_name" json:"first_name" mapstructure:"first_name"`
	LastName     string      `db:"last_name" json:"last_name" mapstructure:"last_name"`
	DisplayName  string      `db:"display_name" json:"display_name" mapstructure:"display_name"`
	Avatar       string      `db:"avatar" json:"avatar" mapstructure:"avatar"`
	ActivityData interface{} `db:"activity_data" json:"activity_data" mapstructure:"activity_data"`

	// Authentication
	AuthProvider  string    `db:"auth_provider" json:"auth_provider" mapstructure:"auth_provider"`
	ExternalID    uuid.UUID `db:"external_id" json:"external_id" mapstructure:"external_id"`
	MFAEnabled    bool      `db:"mfa_enabled" json:"mfa_enabled" mapstructure:"mfa_enabled"`
	MFASecret     string    `db:"mfa_secret" json:"mfa_secret" mapstructure:"mfa_secret"`
	EmailVerified bool      `db:"email_verified" json:"email_verified" mapstructure:"email_verified"`
	Phone         string    `db:"phone" json:"phone" mapstructure:"phone"`
	PhoneVerified bool      `db:"phone_verified" json:"phone_verified" mapstructure:"phone_verified"`

	// Account status
	Status       string     `db:"status" json:"status" mapstructure:"status"`
	LastLoginAt  *time.Time `db:"last_login" json:"last_login_at" mapstructure:"last_login"`
	LastActiveAt *time.Time `db:"last_active_at" json:"last_active_at" mapstructure:"last_active_at"`
	Timezone     string     `db:"timezone" json:"timezone" mapstructure:"timezone"`
	Locale       string     `db:"locale" json:"locale" mapstructure:"locale"`

	// Security
	FailedLoginAttempts int        `db:"failed_login_attempts" json:"failed_login_attempts" mapstructure:"failed_login_attempts"`
	LockedUntil         *time.Time `db:"locked_until" json:"locked_until" mapstructure:"locked_until"`
	PasswordChangedAt   *time.Time `db:"password_changed_at" json:"password_changed_at" mapstructure:"password_changed_at"`

	CreatedAt   time.Time  `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt   time.Time  `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
	DeletedAt   *time.Time `db:"deleted_at" json:"deleted_at" mapstructure:"deleted_at"`
	IsDeleted   bool       `db:"is_deleted" json:"is_deleted" mapstructure:"is_deleted"`
	DateOfBirth *string    `db:"date_of_birth" json:"dob" mapstructure:"date_of_birth"`
	Country     string     `db:"country" json:"country" mapstructure:"country"`
}

type UserInsertion struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	Email         string     `db:"email" json:"email"`
	Password      string     `db:"password" json:"password"`
	AuthProvider  string     `db:"auth_provider" json:"auth_provider"`
	FirstName     string     `db:"first_name" json:"first_name"`
	LastName      string     `db:"last_name" json:"last_name"`
	DisplayName   string     `db:"display_name" json:"display_name"`
	CreatedAt     time.Time  `db:"created_time" json:"created_time"`
	UpdatedAt     time.Time  `db:"last_modified_time" json:"last_modified_time"`
	DeletedAt     *time.Time `db:"deleted_at" json:"deleted_at"`
	DateOfBirth   *string    `db:"dob" json:"date_of_birth,omitempty" format:"2006-01-02"`
	Country       string     `db:"country" json:"country,omitempty"`
	Timezone      string     `db:"timezone" json:"timezone,omitempty"`
	Status        string     `db:"status" json:"status"`
	EmailVerified bool       `db:"email_verified" json:"email_verified"`
	Roles         string     `db:"roles" json:"roles,omitempty"`
}

func (u *UserInsertion) Map() map[string]interface{} {
	result := map[string]interface{}{
		"id":                 u.ID,
		"email":              u.Email,
		"password":           u.Password,
		"auth_provider":      u.AuthProvider,
		"first_name":         u.FirstName,
		"last_name":          u.LastName,
		"display_name":       u.DisplayName,
		"created_time":       u.CreatedAt,
		"last_modified_time": u.UpdatedAt,
		"deleted_at":         u.DeletedAt,
		"timezone":           u.Timezone,
		"status":             u.Status,
		"roles":              u.Roles,
		"email_verified":     u.EmailVerified,
	}
	// Only include date_of_birth and country if they are not nil/empty
	// These fields may not exist in all schemas
	// if u.DateOfBirth != nil {
	// 	result["dob"] = u.DateOfBirth
	// }
	// if u.Country != "" {
	// 	result["country"] = u.Country
	// }
	return result
}

type UpdateUserProfileRequest struct {
	FirstName    *string                 `json:"first_name" form:"first_name" mapstructure:"first_name"`
	LastName     *string                 `json:"last_name" form:"last_name" mapstructure:"last_name"`
	DisplayName  *string                 `json:"display_name" form:"display_name" mapstructure:"display_name"`
	ActivityData *map[string]interface{} `json:"activity_data,omitempty" form:"activity_data" mapstructure:"activity_data"`
	UpdatedAt    time.Time               `json:"last_modified_time" mapstructure:"last_modified_time"`
	DateOfBirth  *string                 `json:"dob" form:"dob" mapstructure:"date_of_birth"`
	Country      *string                 `json:"country" form:"country" mapstructure:"country"`
	Timezone     *string                 `json:"timezone" form:"timezone" mapstructure:"timezone"`
	Locale       *string                 `json:"locale" form:"locale" mapstructure:"locale"`
	ProfilePic   *multipart.FileHeader   `form:"avatar"`
}

func (u *UpdateUserProfileRequest) Map() map[string]interface{} {
	m := make(map[string]interface{})
	if u.FirstName != nil {
		m["first_name"] = *u.FirstName
	}
	if u.LastName != nil {
		m["last_name"] = *u.LastName
	}
	if u.DisplayName != nil {
		m["display_name"] = *u.DisplayName
	}
	if u.ActivityData != nil {
		m["activity_data"] = helpers.InterfaceToJSONString(*u.ActivityData)
	}
	if u.DateOfBirth != nil {
		m["date_of_birth"] = *u.DateOfBirth
	}
	if u.Country != nil {
		m["country"] = *u.Country
	}
	if u.Timezone != nil {
		m["timezone"] = *u.Timezone
	}
	if u.Locale != nil {
		m["locale"] = *u.Locale
	}
	// Always add UpdatedAt as it is not a pointer and presumably always set.
	m["last_modified_time"] = u.UpdatedAt
	return m
}

type UpdateUserPasswordRequest struct {
	OldPassword string `json:"old_password" mapstructure:"old_password" binding:"required"`
	NewPassword string `json:"new_password" mapstructure:"new_password" binding:"required"`
}

func (u *UpdateUserPasswordRequest) Map() map[string]interface{} {
	m := make(map[string]interface{})
	m["old_password"] = u.OldPassword
	m["new_password"] = u.NewPassword
	return m
}

type AddUserRequest struct {
	Email      string                `form:"email" json:"email" mapstructure:"email" binding:"required,email"`
	FirstName  string                `form:"firstname" json:"firstname" mapstructure:"firstname"`
	LastName   string                `form:"lastname" json:"lastname" mapstructure:"lastname"`
	ProfilePic *multipart.FileHeader `form:"profile_pic" json:"profile_pic" mapstructure:"profile_pic"`
	IsCoOwner  bool                  `form:"is_coowner" json:"is_coowner" mapstructure:"is_coowner"`
	Membership []MembershipRequest   `json:"membership" mapstructure:"membership"`
}

// EditUserRequest for updating user details - all fields are optional
type EditUserRequest struct {
	UserID     string                `json:"user_id" mapstructure:"user_id" binding:"required"`
	FirstName  *string               `form:"firstname" json:"firstname" mapstructure:"firstname"`
	LastName   *string               `form:"lastname" json:"lastname" mapstructure:"lastname"`
	ProfilePic *multipart.FileHeader `form:"profile_pic" json:"profile_pic" mapstructure:"profile_pic"`
	IsCoOwner  *bool                 `form:"is_coowner" json:"is_coowner" mapstructure:"is_coowner"`
	Membership []MembershipRequest   `json:"membership" mapstructure:"membership"`
}

type MembershipRequest struct {
	WorkspaceID string           `json:"workspace_id" mapstructure:"workspace_id"`
	Role        string           `json:"role" mapstructure:"role"`
	Bases       []BaseMembership `json:"bases" mapstructure:"bases"`
}

type BaseMembership struct {
	BaseID string `json:"base_id" mapstructure:"base_id"`
	Role   string `json:"role" mapstructure:"role"`
}

type RemoveUserRequest struct {
	UserID string `json:"user_id" mapstructure:"user_id" binding:"required"`
}

type ActivateUserRequest struct {
	UserID string `json:"user_id" mapstructure:"user_id" binding:"required"`
}

type DeactivateUserRequest struct {
	UserID string `json:"user_id" mapstructure:"user_id" binding:"required"`
}

type UserWorkspaceResponse struct {
	ID          uuid.UUID              `db:"id" json:"id" mapstructure:"id"`
	Title       string                 `db:"title" json:"title" mapstructure:"title"`
	Description *string                `db:"description" json:"description" mapstructure:"description"`
	Slug        string                 `db:"slug" json:"slug" mapstructure:"slug"`
	Meta        map[string]interface{} `db:"meta" json:"meta" mapstructure:"meta"`
	IsDefault   bool                   `db:"is_default" json:"is_default" mapstructure:"is_default"`
	Status      string                 `db:"status" json:"status" mapstructure:"status"`

	AccessLevel string `db:"access_level" json:"access_level" mapstructure:"access_level"`

	CreatedAt time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
}

type UserWithRole struct {
	ID           uuid.UUID   `db:"id" json:"id" mapstructure:"id"`
	Email        string      `db:"email" json:"email" mapstructure:"email"`
	FirstName    string      `db:"first_name" json:"first_name" mapstructure:"first_name"`
	LastName     string      `db:"last_name" json:"last_name" mapstructure:"last_name"`
	DisplayName  string      `db:"display_name" json:"display_name" mapstructure:"display_name"`
	Avatar       string      `db:"avatar" json:"avatar" mapstructure:"avatar"`
	ActivityData interface{} `db:"activity_data" json:"activity_data" mapstructure:"activity_data"`

	// Authentication
	AuthProvider  string `db:"auth_provider" json:"auth_provider" mapstructure:"auth_provider"`
	EmailVerified bool   `db:"email_verified" json:"email_verified" mapstructure:"email_verified"`
	Phone         string `db:"phone" json:"phone" mapstructure:"phone"`
	PhoneVerified bool   `db:"phone_verified" json:"phone_verified" mapstructure:"phone_verified"`

	// Account status
	Status       string     `db:"status" json:"status" mapstructure:"status"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at" mapstructure:"last_login_at"`
	LastActiveAt *time.Time `db:"last_active_at" json:"last_active_at" mapstructure:"last_active_at"`
	Timezone     string     `db:"timezone" json:"timezone" mapstructure:"timezone"`
	Country      string     `db:"country" json:"country" mapstructure:"country"`
	Locale       string     `db:"locale" json:"locale" mapstructure:"locale"`

	// Security
	PasswordChangedAt *string `db:"password_changed_at" json:"password_changed_at" mapstructure:"password_changed_at"`

	CreatedAt string `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt string `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`

	Roles []map[string]interface{} `json:"roles" db:"roles" mapstructure:"roles"`
}

// BaseAccessInfo contains base information with access level
type BaseAccessInfo struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

// WorkspaceAccessInfo contains workspace information with bases and access level
type WorkspaceAccessInfo struct {
	ID          uuid.UUID        `json:"id"`
	Title       string           `json:"title"`
	AccessLevel string           `json:"access_level"`
	Bases       []BaseAccessInfo `json:"bases"`
}

// UserAccessDetailsResponse contains comprehensive user access information
type UserAccessDetailsResponse struct {
	Workspaces []WorkspaceAccessInfo `json:"workspaces"`
}
