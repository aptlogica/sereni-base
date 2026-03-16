package tests

import (
	"github.com/aptlogica/sereni-base/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUpdateUserProfileRequestMap(t *testing.T) {
	firstName := "Jane"
	lastName := "Smith"
	displayName := "Jane Smith"
	timezone := "America/New_York"
	locale := "en-US"
	country := "CA"
	dob := "1995-05-15"
	activityData := map[string]interface{}{"lastActivity": "login"}
	now := time.Now()

	req := &dto.UpdateUserProfileRequest{
		FirstName:    &firstName,
		LastName:     &lastName,
		DisplayName:  &displayName,
		Timezone:     &timezone,
		Locale:       &locale,
		Country:      &country,
		DateOfBirth:  &dob,
		ActivityData: &activityData,
		UpdatedAt:    now,
	}

	m := req.Map()

	if m["first_name"] != firstName {
		t.Errorf("Map() first_name = %v, want %v", m["first_name"], firstName)
	}
	if m["last_name"] != lastName {
		t.Errorf("Map() last_name = %v, want %v", m["last_name"], lastName)
	}
	if m["display_name"] != displayName {
		t.Errorf("Map() display_name = %v, want %v", m["display_name"], displayName)
	}
	if m["timezone"] != timezone {
		t.Errorf("Map() timezone = %v, want %v", m["timezone"], timezone)
	}
	if m["locale"] != locale {
		t.Errorf("Map() locale = %v, want %v", m["locale"], locale)
	}
	if m["country"] != country {
		t.Errorf("Map() country = %v, want %v", m["country"], country)
	}
}

func TestUpdateUserProfileRequestMapEmpty(t *testing.T) {
	now := time.Now()

	req := &dto.UpdateUserProfileRequest{
		UpdatedAt: now,
	}

	m := req.Map()

	if len(m) != 1 {
		t.Errorf("Map() length = %d, want 1", len(m))
	}
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}

func TestUserInsertionMap(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	dob := "1990-01-15"

	user := dto.UserInsertion{
		ID:            id,
		Email:         "test@example.com",
		Password:      "hashedpassword",
		AuthProvider:  "local",
		FirstName:     "John",
		LastName:      "Doe",
		DisplayName:   "John Doe",
		CreatedAt:     now,
		UpdatedAt:     now,
		DateOfBirth:   &dob,
		Country:       "US",
		Timezone:      "UTC",
		Status:        "active",
		EmailVerified: true,
		Roles:         "user",
	}

	m := user.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["email"] != "test@example.com" {
		t.Errorf("Map() email = %v, want %v", m["email"], "test@example.com")
	}
	if m["password"] != "hashedpassword" {
		t.Errorf("Map() password = %v, want %v", m["password"], "hashedpassword")
	}
	if m["auth_provider"] != "local" {
		t.Errorf("Map() auth_provider = %v, want %v", m["auth_provider"], "local")
	}
	if m["first_name"] != "John" {
		t.Errorf("Map() first_name = %v, want %v", m["first_name"], "John")
	}
	if m["last_name"] != "Doe" {
		t.Errorf("Map() last_name = %v, want %v", m["last_name"], "Doe")
	}
	if m["display_name"] != "John Doe" {
		t.Errorf("Map() display_name = %v, want %v", m["display_name"], "John Doe")
	}
	if m["timezone"] != "UTC" {
		t.Errorf("Map() timezone = %v, want %v", m["timezone"], "UTC")
	}
	if m["status"] != "active" {
		t.Errorf("Map() status = %v, want %v", m["status"], "active")
	}
	if m["email_verified"] != true {
		t.Errorf("Map() email_verified = %v, want %v", m["email_verified"], true)
	}
}

func TestUpdateUserPasswordRequestMap(t *testing.T) {
	req := dto.UpdateUserPasswordRequest{
		OldPassword: "oldpass123",
		NewPassword: "newpass456",
	}

	m := req.Map()

	if m["old_password"] != "oldpass123" {
		t.Errorf("Map() old_password = %v, want %v", m["old_password"], "oldpass123")
	}
	if m["new_password"] != "newpass456" {
		t.Errorf("Map() new_password = %v, want %v", m["new_password"], "newpass456")
	}
}

func TestUserResponseFields(t *testing.T) {
	id := uuid.New()
	externalID := uuid.New()
	now := time.Now()
	dob := "1990-05-15"

	user := dto.UserResponse{
		ID:                  id,
		Email:               "user@example.com",
		FirstName:           "Jane",
		LastName:            "Smith",
		DisplayName:         "Jane S",
		Avatar:              "avatar.png",
		AuthProvider:        "google",
		ExternalID:          externalID,
		MFAEnabled:          true,
		EmailVerified:       true,
		Phone:               "+1234567890",
		PhoneVerified:       true,
		Status:              "active",
		LastLoginAt:         &now,
		LastActiveAt:        &now,
		Timezone:            "America/New_York",
		Locale:              "en-US",
		FailedLoginAttempts: 0,
		CreatedAt:           now,
		UpdatedAt:           now,
		IsDeleted:           false,
		DateOfBirth:         &dob,
		Country:             "US",
	}

	if user.ID != id {
		t.Errorf("ID = %v, want %v", user.ID, id)
	}
	if user.Email != "user@example.com" {
		t.Errorf("Email = %v, want %v", user.Email, "user@example.com")
	}
	if user.MFAEnabled != true {
		t.Errorf("MFAEnabled = %v, want %v", user.MFAEnabled, true)
	}
	if user.Status != "active" {
		t.Errorf("Status = %v, want %v", user.Status, "active")
	}
}
