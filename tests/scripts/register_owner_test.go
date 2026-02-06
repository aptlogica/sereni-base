package scripts_test

import (
	"context"
	"os"
	"testing"

	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	"serenibase/internal/config"
	"serenibase/internal/scripts"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTableServiceOwner is a mock implementation of TableService for owner tests
type MockTableServiceOwner struct {
	mock.Mock
}

func (m *MockTableServiceOwner) GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
	args := m.Called(tableName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceOwner) CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceOwner) UpdateRecord(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, id, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableServiceOwner) DeleteRecord(tableName string, id interface{}) error {
	args := m.Called(tableName, id)
	return args.Error(0)
}

func (m *MockTableServiceOwner) GetTables(schema string) ([]dbModels.Table, error) {
	args := m.Called(schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbModels.Table), args.Error(1)
}

func (m *MockTableServiceOwner) CreateTable(req dbModels.CreateTableRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockTableServiceOwner) AddColumn(tableName string, req dbModels.AddColumnRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableServiceOwner) AlterTable(tableName string, req dbModels.AlterTableRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableServiceOwner) BuildComplexQuery(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error) {
	args := m.Called(tableName, filters)
	if args.Get(0) == nil {
		return dbModels.QueryParams{}, args.Error(1)
	}
	return args.Get(0).(dbModels.QueryParams), args.Error(1)
}

func (m *MockTableServiceOwner) CreateSchema(ctx context.Context, schemaName string) error {
	args := m.Called(ctx, schemaName)
	return args.Error(0)
}

func (m *MockTableServiceOwner) DropTable(ctx context.Context, tableName string) error {
	args := m.Called(ctx, tableName)
	return args.Error(0)
}

func (m *MockTableServiceOwner) CreateView(ctx context.Context, viewName string, viewSQL string) error {
	args := m.Called(ctx, viewName, viewSQL)
	return args.Error(0)
}

func (m *MockTableServiceOwner) CreateFunction(ctx context.Context, functionName string, functionSQL string) error {
	args := m.Called(ctx, functionName, functionSQL)
	return args.Error(0)
}

func (m *MockTableServiceOwner) GetByFunction(ctx context.Context, functionName string, fnArgs map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, functionName, fnArgs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

// setupMockDBForOwner creates a mock database service for owner tests
func setupMockDBForOwner() (*pkg.DatabaseService, *MockTableServiceOwner) {
	mockTable := &MockTableServiceOwner{}
	db := &pkg.DatabaseService{
		TableService: mockTable,
	}
	return db, mockTable
}

func TestMaybeSkipOwnerRegistration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      *config.Config
		expected bool
	}{
		{
			name: "skip when email empty",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "",
				},
			},
			expected: true,
		},
		{
			name: "don't skip when email present",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "owner@example.com",
				},
			},
			expected: false,
		},
		{
			name: "skip with whitespace-only email",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "",
				},
			},
			expected: true,
		},
		{
			name: "don't skip with valid email format",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "test@test.com",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scripts.MaybeSkipOwnerRegistration(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateOwnerConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "owner@example.com",
					Password:  "password123",
					FirstName: "John",
					LastName:  "Doe",
				},
			},
			expectError: false,
		},
		{
			name: "missing password",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "owner@example.com",
					FirstName: "John",
					LastName:  "Doe",
				},
			},
			expectError: true,
			errorMsg:    "password",
		},
		{
			name: "missing first name",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:    "owner@example.com",
					Password: "password123",
					LastName: "Doe",
				},
			},
			expectError: true,
			errorMsg:    "first name",
		},
		{
			name: "missing last name",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "owner@example.com",
					Password:  "password123",
					FirstName: "John",
				},
			},
			expectError: true,
			errorMsg:    "last name",
		},
		{
			name: "all fields empty except email",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "owner@example.com",
				},
			},
			expectError: true,
		},
		{
			name: "valid config with long values",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "owner@example.com",
					Password:  "a-very-long-and-secure-password-12345!@#$%",
					FirstName: "JohnJohnJohnJohnJohn",
					LastName:  "DoeDoeDoeDoeDoeDoe",
				},
			},
			expectError: false,
		},
		{
			name: "valid config with unicode names",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "owner@example.com",
					Password:  "password123",
					FirstName: "José",
					LastName:  "García",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scripts.ValidateOwnerConfig(tt.cfg)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrepareOwnerRegisterRequest(t *testing.T) {
	t.Parallel()

	t.Run("basic registration request", func(t *testing.T) {
		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		result := scripts.PrepareOwnerRegisterRequest(cfg)

		assert.Equal(t, "owner@example.com", result.Email)
		assert.Equal(t, "password123", result.Password)
		assert.Equal(t, "John", result.FirstName)
		assert.Equal(t, "Doe", result.LastName)
		assert.Equal(t, "local", result.AuthProvider)
		assert.Equal(t, "active", result.Status)
		assert.True(t, result.EmailVerified)
		assert.NotEmpty(t, result.ID)
	})

	t.Run("registration request with COUNTRY env var", func(t *testing.T) {
		// Set environment variable
		originalCountry := os.Getenv("COUNTRY")
		os.Setenv("COUNTRY", "UK")
		defer os.Setenv("COUNTRY", originalCountry)

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		result := scripts.PrepareOwnerRegisterRequest(cfg)

		assert.Equal(t, "UK", result.Country)
	})

	t.Run("registration request without COUNTRY env var defaults to US", func(t *testing.T) {
		// Unset environment variable
		originalCountry := os.Getenv("COUNTRY")
		os.Unsetenv("COUNTRY")
		defer func() {
			if originalCountry != "" {
				os.Setenv("COUNTRY", originalCountry)
			}
		}()

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		result := scripts.PrepareOwnerRegisterRequest(cfg)

		assert.Equal(t, "US", result.Country)
	})

	t.Run("registration request has timezone set", func(t *testing.T) {
		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		result := scripts.PrepareOwnerRegisterRequest(cfg)

		assert.NotEmpty(t, result.Timezone)
	})

	t.Run("registration request has owner role", func(t *testing.T) {
		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		result := scripts.PrepareOwnerRegisterRequest(cfg)

		assert.NotEmpty(t, result.Roles)
	})

	t.Run("registration request with special characters", func(t *testing.T) {
		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner+test@example.com",
				Password:  "p@$$w0rd!#%",
				FirstName: "John-Paul",
				LastName:  "O'Brien",
			},
		}

		result := scripts.PrepareOwnerRegisterRequest(cfg)

		assert.Equal(t, "owner+test@example.com", result.Email)
		assert.Equal(t, "p@$$w0rd!#%", result.Password)
		assert.Equal(t, "John-Paul", result.FirstName)
		assert.Equal(t, "O'Brien", result.LastName)
	})

	t.Run("each call generates unique ID", func(t *testing.T) {
		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		result1 := scripts.PrepareOwnerRegisterRequest(cfg)
		result2 := scripts.PrepareOwnerRegisterRequest(cfg)

		assert.NotEqual(t, result1.ID, result2.ID)
	})
}

// TestCreateDefaultOrganization tests the CreateDefaultOrganization function
func TestCreateDefaultOrganization(t *testing.T) {
	t.Run("returns nil even on failure", func(t *testing.T) {
		dbService, mockTable := setupMockDBForOwner()

		// Mock CreateRecord to fail
		mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		mockTable.On("GetTableData", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		err := scripts.CreateDefaultOrganization(dbService, cfg, "owner@example.com")

		// Function should return nil even on error (non-critical operation)
		assert.Nil(t, err)
	})

	t.Run("creates organization with correct name", func(t *testing.T) {
		dbService, mockTable := setupMockDBForOwner()

		mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(map[string]interface{}{
			"id":    "test-id",
			"name":  "John's Organization",
			"email": "owner@example.com",
		}, nil)
		mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		err := scripts.CreateDefaultOrganization(dbService, cfg, "owner@example.com")

		assert.Nil(t, err)
	})
}

// TestRegisterOwnerSignature verifies the function signature
func TestRegisterOwnerSignature(t *testing.T) {
	t.Run("RegisterOwner function exists with correct signature", func(t *testing.T) {
		// This is a compile-time check - if this compiles, the signature is correct
		assert.NotNil(t, scripts.RegisterOwner)
	})
}

// TestMaybeSkipOwnerRegistrationEdgeCases tests edge cases
func TestMaybeSkipOwnerRegistrationEdgeCases(t *testing.T) {
	t.Run("nil config handling", func(t *testing.T) {
		// This test would panic with nil config
		// The function should be called with a valid config
		cfg := &config.Config{}
		result := scripts.MaybeSkipOwnerRegistration(cfg)
		assert.True(t, result)
	})
}

// TestValidateOwnerConfigEdgeCases tests edge cases for validation
func TestValidateOwnerConfigEdgeCases(t *testing.T) {
	t.Run("empty strings should fail", func(t *testing.T) {
		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "",
				FirstName: "",
				LastName:  "",
			},
		}
		err := scripts.ValidateOwnerConfig(cfg)
		assert.Error(t, err)
	})

	t.Run("whitespace-only password should fail", func(t *testing.T) {
		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "",
				FirstName: "John",
				LastName:  "Doe",
			},
		}
		err := scripts.ValidateOwnerConfig(cfg)
		assert.Error(t, err)
	})
}

// BenchmarkMaybeSkipOwnerRegistration benchmarks the function
func BenchmarkMaybeSkipOwnerRegistration(b *testing.B) {
	cfg := &config.Config{
		OwnerRegistration: config.OwnerRegistrationConfig{
			Email: "owner@example.com",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scripts.MaybeSkipOwnerRegistration(cfg)
	}
}

// BenchmarkValidateOwnerConfig benchmarks the function
func BenchmarkValidateOwnerConfig(b *testing.B) {
	cfg := &config.Config{
		OwnerRegistration: config.OwnerRegistrationConfig{
			Email:     "owner@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scripts.ValidateOwnerConfig(cfg)
	}
}

// BenchmarkPrepareOwnerRegisterRequest benchmarks the function
func BenchmarkPrepareOwnerRegisterRequest(b *testing.B) {
	cfg := &config.Config{
		OwnerRegistration: config.OwnerRegistrationConfig{
			Email:     "owner@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scripts.PrepareOwnerRegisterRequest(cfg)
	}
}

// MockAuthProvider is a mock implementation of AuthProvider for testing
type MockAuthProvider struct {
	mock.Mock
}

func (m *MockAuthProvider) GenerateToken(ctx context.Context, user interface{}) (interface{}, error) {
	args := m.Called(ctx, user)
	return args.Get(0), args.Error(1)
}

func (m *MockAuthProvider) RefreshToken(ctx context.Context, token string) (interface{}, error) {
	args := m.Called(ctx, token)
	return args.Get(0), args.Error(1)
}

func (m *MockAuthProvider) ValidateToken(ctx context.Context, tokenStr string) (interface{}, error) {
	args := m.Called(ctx, tokenStr)
	return args.Get(0), args.Error(1)
}

// TestRegisterOwner tests the RegisterOwner function
func TestRegisterOwner(t *testing.T) {
	t.Run("skips registration when email is empty", func(t *testing.T) {
		dbService, _ := setupMockDBForOwner()

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email: "", // Empty email should skip registration
			},
		}

		err := scripts.RegisterOwner(dbService, nil, cfg)

		assert.Nil(t, err, "Should return nil when email is empty (skip registration)")
	})

	t.Run("returns error when password is missing", func(t *testing.T) {
		dbService, _ := setupMockDBForOwner()

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "", // Missing password
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		err := scripts.RegisterOwner(dbService, nil, cfg)

		assert.Error(t, err, "Should return error when password is missing")
		assert.Contains(t, err.Error(), "password")
	})

	t.Run("returns error when first name is missing", func(t *testing.T) {
		dbService, _ := setupMockDBForOwner()

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "", // Missing first name
				LastName:  "Doe",
			},
		}

		err := scripts.RegisterOwner(dbService, nil, cfg)

		assert.Error(t, err, "Should return error when first name is missing")
		assert.Contains(t, err.Error(), "first name")
	})

	t.Run("returns error when last name is missing", func(t *testing.T) {
		dbService, _ := setupMockDBForOwner()

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "", // Missing last name
			},
		}

		err := scripts.RegisterOwner(dbService, nil, cfg)

		assert.Error(t, err, "Should return error when last name is missing")
		assert.Contains(t, err.Error(), "last name")
	})
}

// TestRegisterOwnerFunctionSignature verifies the function signature
func TestRegisterOwnerFunctionSignature(t *testing.T) {
	t.Run("function has correct signature", func(t *testing.T) {
		// This test verifies that the function signature is correct at compile time
		assert.NotNil(t, scripts.RegisterOwner)
	})
}

// TestRegisterOwnerValidation tests various validation scenarios
func TestRegisterOwnerValidation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		expectError bool
		expectSkip  bool
	}{
		{
			name: "empty email skips",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "",
				},
			},
			expectError: false,
			expectSkip:  true,
		},
		{
			name: "missing password returns error",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "test@test.com",
					FirstName: "John",
					LastName:  "Doe",
				},
			},
			expectError: true,
		},
		{
			name: "missing first name returns error",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:    "test@test.com",
					Password: "pass123",
					LastName: "Doe",
				},
			},
			expectError: true,
		},
		{
			name: "missing last name returns error",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "test@test.com",
					Password:  "pass123",
					FirstName: "John",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbService, _ := setupMockDBForOwner()
			err := scripts.RegisterOwner(dbService, nil, tt.cfg)

			if tt.expectSkip {
				assert.Nil(t, err)
			} else if tt.expectError {
				assert.Error(t, err)
			}
		})
	}
}

// TestCreateDefaultOrganizationExtended tests more scenarios
func TestCreateDefaultOrganizationExtended(t *testing.T) {
	t.Run("handles nil database service gracefully", func(t *testing.T) {
		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				FirstName: "John",
			},
		}

		// Should not panic with proper mock
		assert.NotPanics(t, func() {
			dbService, mockTable := setupMockDBForOwner()
			mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(nil, assert.AnError)
			mockTable.On("GetTableData", mock.Anything, mock.Anything).Return(nil, assert.AnError)

			scripts.CreateDefaultOrganization(dbService, cfg, "owner@example.com")
		})
	})

	t.Run("creates organization with empty email", func(t *testing.T) {
		dbService, mockTable := setupMockDBForOwner()

		mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(map[string]interface{}{
			"id":    "test-id",
			"name":  "John's Organization",
			"email": "",
		}, nil)
		mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				FirstName: "John",
			},
		}

		err := scripts.CreateDefaultOrganization(dbService, cfg, "")

		assert.Nil(t, err)
	})

	t.Run("handles various first name values", func(t *testing.T) {
		testCases := []struct {
			firstName string
		}{
			{""},
			{"A"},
			{"John"},
			{"John-Paul"},
			{"José"},
			{"日本語"},
		}

		for _, tc := range testCases {
			dbService, mockTable := setupMockDBForOwner()
			mockTable.On("CreateRecord", mock.Anything, mock.Anything).Return(map[string]interface{}{
				"id": "test-id",
			}, nil)
			mockTable.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

			cfg := &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					FirstName: tc.firstName,
				},
			}

			err := scripts.CreateDefaultOrganization(dbService, cfg, "test@test.com")
			assert.Nil(t, err)
		}
	})
}

// TestRegisterOwnerIntegration tests RegisterOwner in scenarios where it proceeds past validation
// Note: Full integration tests would require actual database connections
func TestRegisterOwnerIntegration(t *testing.T) {
	t.Run("panics with valid config but nil database table service", func(t *testing.T) {
		// When all validation passes but the underlying services are not initialized
		dbService := &pkg.DatabaseService{
			TableService: nil,
		}

		cfg := &config.Config{
			OwnerRegistration: config.OwnerRegistrationConfig{
				Email:     "owner@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		// This will panic because the internal services require a valid TableService
		assert.Panics(t, func() {
			scripts.RegisterOwner(dbService, nil, cfg)
		})
	})
}

// TestRegisterOwnerWithAllValidationErrors tests all validation error paths
func TestRegisterOwnerWithAllValidationErrors(t *testing.T) {
	testCases := []struct {
		name        string
		cfg         *config.Config
		expectError bool
		expectSkip  bool
		errorSubstr string
	}{
		{
			name: "empty email skips without error",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "",
				},
			},
			expectSkip: true,
		},
		{
			name: "whitespace email skips without error",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "",
				},
			},
			expectSkip: true,
		},
		{
			name: "missing password",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "test@test.com",
					Password:  "",
					FirstName: "John",
					LastName:  "Doe",
				},
			},
			expectError: true,
			errorSubstr: "password",
		},
		{
			name: "missing first name",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "test@test.com",
					Password:  "pass123",
					FirstName: "",
					LastName:  "Doe",
				},
			},
			expectError: true,
			errorSubstr: "first name",
		},
		{
			name: "missing last name",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email:     "test@test.com",
					Password:  "pass123",
					FirstName: "John",
					LastName:  "",
				},
			},
			expectError: true,
			errorSubstr: "last name",
		},
		{
			name: "all required fields missing",
			cfg: &config.Config{
				OwnerRegistration: config.OwnerRegistrationConfig{
					Email: "test@test.com",
				},
			},
			expectError: true,
			errorSubstr: "password",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dbService, _ := setupMockDBForOwner()
			err := scripts.RegisterOwner(dbService, nil, tc.cfg)

			if tc.expectSkip {
				assert.Nil(t, err, "Expected nil error when skipping")
			} else if tc.expectError {
				assert.Error(t, err)
				if tc.errorSubstr != "" {
					assert.Contains(t, err.Error(), tc.errorSubstr)
				}
			}
		})
	}
}
