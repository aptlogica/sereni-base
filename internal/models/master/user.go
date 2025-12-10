package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

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

	DateOfBirth *string `db:"date_of_birth" json:"date_of_birth,omitempty" format:"2006-01-02" mapstructure:"date_of_birth"`
	Country     string  `db:"country" json:"country,omitempty" mapstructure:"country"`

	CreatedAt time.Time  `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time  `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty" mapstructure:"deleted_at"`
	IsDeleted bool       `db:"is_deleted" json:"is_deleted,omitempty" mapstructure:"is_deleted"`
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
			{Name: "email", DataType: "varchar(255)", NotNull: true, Unique: true},
			{Name: "password", DataType: "varchar(255)"},
			{Name: "first_name", DataType: "varchar(100)"},
			{Name: "last_name", DataType: "varchar(100)"},
			{Name: "display_name", DataType: "varchar(150)"},
			{Name: "avatar", DataType: "text"},
			{Name: "activity_data", DataType: "jsonb"},

			// Authentication
			{Name: "auth_provider", DataType: "varchar(50)", DefaultValue: strPtr("'email'")},
			{Name: "external_id", DataType: "uuid"},
			{Name: "mfa_enabled", DataType: "boolean", DefaultValue: strPtr("false")},
			{Name: "mfa_secret", DataType: "varchar(255)"},
			{Name: "email_verified", DataType: "boolean", DefaultValue: strPtr("false")},
			{Name: "phone", DataType: "varchar(50)"},
			{Name: "phone_verified", DataType: "boolean", DefaultValue: strPtr("false")},

			// Account status
			{Name: "status", DataType: "varchar(50)", DefaultValue: strPtr("'pending'")},
			{Name: "last_login_at", DataType: "timestamptz", DefaultValue: &null},
			{Name: "last_active_at", DataType: "timestamptz", DefaultValue: &null},
			{Name: "timezone", DataType: "varchar(50)", DefaultValue: strPtr("'UTC'")},
			{Name: "locale", DataType: "varchar(10)", DefaultValue: strPtr("'en'")},

			// Security
			{Name: "failed_login_attempts", DataType: "int", DefaultValue: strPtr("0")},
			{Name: "locked_until", DataType: "timestamptz", DefaultValue: &null},
			{Name: "password_changed_at", DataType: "timestamptz", DefaultValue: &null},

			{Name: "date_of_birth", DataType: "TEXT"},
			{Name: "country", DataType: "varchar(100)"},

			// Timestamps
			{Name: "created_time", DataType: "timestamptz", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamptz", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "deleted_at", DataType: "timestamptz", DefaultValue: &null},
			{Name: "is_deleted", DataType: "boolean", DefaultValue: strPtr("false")},
		},
	}
}

func strPtr(s string) *string {
	return &s
}
