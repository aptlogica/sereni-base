// serenibase/internal/dto/workspace_dto.go
// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"time"

	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
)

// WorkspaceInsertion DTO for inserting workspace into DB
type WorkspaceInsertion struct {
	ID          uuid.UUID              `db:"id" json:"id,omitempty"`
	Title       string                 `db:"title" json:"title,omitempty"`
	Description *string                `db:"description" json:"description,omitempty"`
	Slug        string                 `db:"slug" json:"slug,omitempty"`
	Meta        map[string]interface{} `db:"meta" json:"meta"`
	IsDefault   bool                   `db:"is_default" json:"is_default,omitempty"`
	Status      string                 `db:"status" json:"status,omitempty"`
	CreatedBy   string                 `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy   string                 `db:"last_modified_by" json:"last_modified_by,omitempty"`
	CreatedAt   time.Time              `db:"created_time" json:"created_time,omitempty"`
	UpdatedAt   time.Time              `db:"last_modified_time" json:"last_modified_time,omitempty"`
}

type WorkspaceUpdate struct {
	Title       *string                 `db:"title" json:"title,omitempty"`
	Description *string                 `db:"description" json:"description,omitempty"`
	Slug        *string                 `db:"slug" json:"slug,omitempty"`
	Meta        *map[string]interface{} `db:"meta" json:"meta,omitempty"`
	IsDefault   *bool                   `db:"is_default" json:"is_default,omitempty"`
	Status      *string                 `db:"status" json:"status,omitempty"`
	UpdatedBy   string                  `db:"last_modified_by" json:"last_modified_by,omitempty"`
	UpdatedAt   time.Time               `db:"last_modified_time" json:"last_modified_time,omitempty"`
}

// Map converts WorkspaceInsertion → map[string]interface{} for DB insert
func (w *WorkspaceInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 w.ID,
		"title":              w.Title,
		"description":        w.Description,
		"slug":               w.Slug,
		"meta":               helpers.InterfaceToJSONString(w.Meta),
		"is_default":         w.IsDefault,
		"status":             w.Status,
		"created_by":         w.CreatedBy,
		"last_modified_by":   w.UpdatedBy,
		"created_time":       w.CreatedAt,
		"last_modified_time": w.UpdatedAt,
	}
}

func (w *WorkspaceUpdate) Map() map[string]interface{} {
	result := make(map[string]interface{})
	if w.Title != nil {
		result["title"] = *w.Title
	}
	if w.Description != nil {
		result["description"] = *w.Description
	}
	if w.Slug != nil {
		result["slug"] = *w.Slug
	}
	if w.Meta != nil {
		result["meta"] = helpers.InterfaceToJSONString(*w.Meta)
	}
	if w.IsDefault != nil {
		result["is_default"] = *w.IsDefault
	}
	if w.Status != nil {
		result["status"] = *w.Status
	}
	if w.UpdatedBy != "" {
		result["last_modified_by"] = w.UpdatedBy
	}
	result["last_modified_time"] = w.UpdatedAt
	return result
}

type CreateWorkspaceRequest struct {
	Title       string  `db:"title" json:"title,omitempty"`
	Description *string `db:"description" json:"description,omitempty"`
	CreatedBy   string  `json:"created_by,omitempty"`
}

// type UpdateWorkspaceRequest struct {
// 	Title       *string `db:"title" json:"title,omitempty"`
// 	Description *string `db:"description" json:"description,omitempty"`
// }

type WorkspaceResponse struct {
	ID          uuid.UUID              `db:"id" json:"id" mapstructure:"id"`
	Title       string                 `db:"title" json:"title" mapstructure:"title"`
	Description *string                `db:"description" json:"description" mapstructure:"description"`
	Slug        string                 `db:"slug" json:"slug" mapstructure:"slug"`
	Meta        map[string]interface{} `db:"meta" json:"meta" mapstructure:"meta"`
	IsDefault   bool                   `db:"is_default" json:"is_default" mapstructure:"is_default"`
	Status      string                 `db:"status" json:"status" mapstructure:"status"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`

	CreatedAt time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`

	Bases []BaseResponse `db:"bases" json:"bases" mapstructure:"bases"`
}

type WorkspaceMemberInsertion struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	WorkspaceID string    `db:"workspace_id" json:"workspace_id,omitempty" mapstructure:"workspace_id"`
	UserID      string    `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
	AccessLevel string    `db:"access_level" json:"access_level,omitempty" mapstructure:"access_level"`
	BasesIds    string    `db:"bases_ids" json:"bases_ids,omitempty" mapstructure:"bases_ids"`
	CreatedAt   time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt   time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (wmi *WorkspaceMemberInsertion) Map() map[string]interface{} {
	result := make(map[string]interface{})
	result["id"] = wmi.ID
	result["workspace_id"] = wmi.WorkspaceID
	result["user_id"] = wmi.UserID
	result["access_level"] = wmi.AccessLevel
	result["bases_ids"] = wmi.BasesIds
	result["created_time"] = wmi.CreatedAt
	result["last_modified_time"] = wmi.UpdatedAt
	return result
}

type CreateMemberRequest struct {
	UserID     string              `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
	Membership []MembershipRequest `json:"membership" mapstructure:"membership"`
}

type RemoveMemberRequest struct {
	UserID string `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
}

// WorkspaceMemberResponse embeds UserResponse, so all UserResponse fields are included.
type WorkspaceMemberResponse struct {
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
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at" mapstructure:"last_login_at"`
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
	AccessLevel string     `json:"access_level" mapstructure:"access_level"`
}

// AddMultipleMembersRequest is used to add multiple users to a workspace at once
type AddMultipleMembersRequest struct {
	WorkspaceID string   `json:"workspace_id,omitempty"`
	UserIDs     []string `json:"user_ids" binding:"required,min=1"`
	AccessLevel string   `json:"access_level" binding:"required"`
	BasesIds    string   `json:"bases_ids,omitempty"`
}

// AddMultipleMembersResponse contains the results of bulk member addition
type AddMultipleMembersResponse struct {
	SuccessCount int                `json:"success_count"`
	FailureCount int                `json:"failure_count"`
	Successes    []MemberAddSuccess `json:"successes"`
	Failures     []MemberAddFailure `json:"failures"`
}

// MemberAddSuccess represents a successfully added member
type MemberAddSuccess struct {
	UserID string `json:"user_id"`
}

// MemberAddFailure represents a failed member addition
type MemberAddFailure struct {
	UserID string `json:"user_id"`
	Error  string `json:"error"`
}

// BulkAddMembersRequest for adding multiple members at once
type BulkAddMembersRequest struct {
	Members []BulkMemberRequest `json:"members" binding:"required,min=1"`
}

// BulkMemberRequest contains user_id and their memberships
type BulkMemberRequest struct {
	UserID      string              `json:"user_id" binding:"required"`
	Memberships []MembershipRequest `json:"memberships" binding:"required,min=1"`
}

// BulkAddMembersResponse contains results of bulk member addition
type BulkAddMembersResponse struct {
	Success []string           `json:"success"`
	Failed  []MemberAddFailure `json:"failed"`
	Total   int                `json:"total"`
}

// BulkAddBaseMembers for adding members to bases
type BulkAddBaseMembersRequest struct {
	Members []BulkMemberRequest `json:"members" binding:"required,min=1"`
}

// BulkBaseMemberRequest for adding user to bases
type BulkBaseMemberRequest struct {
	UserID   string           `json:"user_id" binding:"required"`
	BaseRole []BaseMembership `json:"base_role" binding:"required,min=1"`
}
