// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/aptlogica/sereni-base/internal/constant"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID              `db:"id" json:"id" mapstructure:"id"`
	Email        string                 `db:"email" json:"email" mapstructure:"email"`
	Password     string                 `db:"password" json:"password" mapstructure:"password"`
	FirstName    string                 `db:"first_name" json:"first_name" mapstructure:"first_name"`
	LastName     string                 `db:"last_name" json:"last_name" mapstructure:"last_name"`
	DisplayName  string                 `db:"display_name" json:"display_name" mapstructure:"display_name"`
	Avatar       string                 `db:"avatar" json:"avatar" mapstructure:"avatar"`
	ActivityData map[string]interface{} `db:"activity_data" json:"activity_data" mapstructure:"activity_data"`

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
	Roles               string     `db:"roles" json:"roles" mapstructure:"roles"`
	DateOfBirth         string     `db:"date_of_birth" json:"date_of_birth" format:"2006-01-02" mapstructure:"date_of_birth"`
	Country             string     `db:"country" json:"country" mapstructure:"country"`

	CreatedAt time.Time  `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt time.Time  `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at" mapstructure:"deleted_at"`
	IsDeleted bool       `db:"is_deleted" json:"is_deleted" mapstructure:"is_deleted"`
}

func (User) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".users", prefix)
}

func (tbl User) TableSchema(prefix string) models.CreateTableRequest {
	null := "NULL"

	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			// Core fields
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "email", DataType: constant.DBTypeVarchar255Lower, NotNull: true, Unique: true},
			{Name: "password", DataType: constant.DBTypeVarchar255Lower},
			{Name: "first_name", DataType: constant.DBTypeVarchar100},
			{Name: "last_name", DataType: constant.DBTypeVarchar100},
			{Name: "display_name", DataType: "varchar(150)"},
			{Name: "avatar", DataType: "text"},
			{Name: "activity_data", DataType: "jsonb"},

			// Authentication
			{Name: "auth_provider", DataType: constant.DBTypeVarchar50, DefaultValue: StrPtr("'email'")},
			{Name: "external_id", DataType: "uuid"},
			{Name: "mfa_enabled", DataType: "boolean", DefaultValue: StrPtr("false")},
			{Name: "mfa_secret", DataType: constant.DBTypeVarchar255Lower},
			{Name: "email_verified", DataType: "boolean", DefaultValue: StrPtr("false")},
			{Name: "phone", DataType: constant.DBTypeVarchar50},
			{Name: "phone_verified", DataType: "boolean", DefaultValue: StrPtr("false")},

			// Account status
			{Name: "status", DataType: constant.DBTypeVarchar50, DefaultValue: StrPtr("'pending'")},
			{Name: "last_login_at", DataType: "timestamptz", DefaultValue: &null},
			{Name: "last_active_at", DataType: "timestamptz", DefaultValue: &null},
			{Name: "timezone", DataType: constant.DBTypeVarchar50, DefaultValue: StrPtr("'UTC'")},
			{Name: "locale", DataType: "varchar(10)", DefaultValue: StrPtr("'en'")},

			// Security
			{Name: "failed_login_attempts", DataType: "int", DefaultValue: StrPtr("0")},
			{Name: "locked_until", DataType: "timestamptz", DefaultValue: &null},
			{Name: "password_changed_at", DataType: "timestamptz", DefaultValue: &null},
			{Name: "roles", DataType: constant.DBTypeVarchar255Lower, NotNull: true, DefaultValue: StrPtr("'user'")},

			{Name: "date_of_birth", DataType: "TEXT"},
			{Name: "country", DataType: constant.DBTypeVarchar100},

			// Timestamps
			{Name: "created_time", DataType: "timestamptz", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamptz", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "deleted_at", DataType: "timestamptz", DefaultValue: &null},
			{Name: "is_deleted", DataType: "boolean", DefaultValue: StrPtr("false")},
		},
	}
}
