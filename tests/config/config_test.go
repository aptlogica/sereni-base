package tests

import (
	"os"
	"serenibase/internal/config"
	"testing"
)

// TestLoad tests the config loading functionality
func TestLoad(t *testing.T) {
	t.Run("load with defaults", func(t *testing.T) {
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}

		// Test default values
		if cfg.Server.Port != "8080" && os.Getenv("SERVER_PORT") == "" {
			t.Errorf("Server.Port = %q, want default %q", cfg.Server.Port, "8080")
		}

		if cfg.Server.Host != "0.0.0.0" && os.Getenv("SERVER_HOST") == "" {
			t.Errorf("Server.Host = %q, want default %q", cfg.Server.Host, "0.0.0.0")
		}
	})

	t.Run("load with environment variables", func(t *testing.T) {
		// Set test environment variables
		os.Setenv("SERVER_PORT", "9090")
		os.Setenv("SERVER_HOST", "127.0.0.1")
		os.Setenv("DATABASE_NAME", "testdb")
		defer func() {
			os.Unsetenv("SERVER_PORT")
			os.Unsetenv("SERVER_HOST")
			os.Unsetenv("DATABASE_NAME")
		}()

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Server.Port != "9090" {
			t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "9090")
		}

		if cfg.Server.Host != "127.0.0.1" {
			t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
		}

		if cfg.Database.DatabaseName != "testdb" {
			t.Errorf("Database.DatabaseName = %q, want %q", cfg.Database.DatabaseName, "testdb")
		}
	})
}

// TestServerConfig tests the ServerConfig structure
func TestServerConfig(t *testing.T) {
	tests := []struct {
		name   string
		config config.ServerConfig
	}{
		{
			name: "valid server config",
			config: config.ServerConfig{
				Port:         "8080",
				Host:         "localhost",
				ReadTimeout:  30,
				WriteTimeout: 30,
				Scheme:       "http",
				Env:          "dev",
			},
		},
		{
			name: "https config",
			config: config.ServerConfig{
				Port:         "443",
				Host:         "example.com",
				ReadTimeout:  60,
				WriteTimeout: 60,
				Scheme:       "https",
				Env:          "prod",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Port == "" {
				t.Error("Port should not be empty")
			}
			if tt.config.Host == "" {
				t.Error("Host should not be empty")
			}
			if tt.config.Scheme != "http" && tt.config.Scheme != "https" {
				t.Errorf("Scheme = %q, want 'http' or 'https'", tt.config.Scheme)
			}
		})
	}
}

// TestDatabaseConfig tests the DatabaseConfig structure
func TestDatabaseConfig(t *testing.T) {
	tests := []struct {
		name   string
		config config.DatabaseConfig
	}{
		{
			name: "postgres config",
			config: config.DatabaseConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				Password:     "password",
				DatabaseName: "testdb",
				SSLMode:      "disable",
				MaxOpenConns: 25,
				MaxIdleConns: 5,
				Driver:       "postgres",
			},
		},
		{
			name: "mysql config",
			config: config.DatabaseConfig{
				Host:         "localhost",
				Port:         3306,
				Username:     "root",
				Password:     "password",
				DatabaseName: "testdb",
				MaxOpenConns: 10,
				MaxIdleConns: 2,
				Driver:       "mysql",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Host == "" {
				t.Error("Host should not be empty")
			}
			if tt.config.Port <= 0 {
				t.Error("Port should be positive")
			}
			if tt.config.DatabaseName == "" {
				t.Error("DatabaseName should not be empty")
			}
			if tt.config.Driver == "" {
				t.Error("Driver should not be empty")
			}
		})
	}
}

// TestAuthConfig tests the AuthConfig structure
func TestAuthConfig(t *testing.T) {
	config := config.AuthConfig{
		URL:              "http://auth.example.com",
		ResetPasswordURL: "http://example.com/reset?token=%s",
		JWT: config.JWTConfig{
			Secret:             "secret-key",
			AccessTokenExpiry:  3600,
			RefreshTokenExpiry: 86400,
			Issuer:             "serenibase",
		},
	}

	if config.URL == "" {
		t.Error("URL should not be empty")
	}

	if config.JWT.Secret == "" {
		t.Error("JWT Secret should not be empty")
	}

	if config.JWT.AccessTokenExpiry <= 0 {
		t.Error("Access token expiry should be positive")
	}

	if config.JWT.RefreshTokenExpiry <= 0 {
		t.Error("Refresh token expiry should be positive")
	}

	if config.JWT.Issuer == "" {
		t.Error("JWT Issuer should not be empty")
	}
}

// TestEmailConfig tests the EmailConfig structure
func TestEmailConfig(t *testing.T) {
	tests := []struct {
		name   string
		config config.EmailConfig
	}{
		{
			name: "valid email config",
			config: config.EmailConfig{
				URL: "http://email-service:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.URL == "" {
				t.Error("URL should not be empty")
			}
		})
	}
}

// TestStorageConfig tests the StorageConfig structure
func TestStorageConfig(t *testing.T) {
	config := config.StorageConfig{
		URL: "http://storage-service:8080",
	}

	if config.URL == "" {
		t.Error("URL should not be empty")
	}
}

// TestLogConfig tests the LogConfig structure
func TestLogConfig(t *testing.T) {
	config := config.LogConfig{
		Level:      "info",
		File:       "app.log",
		ErrorFile:  "error.log",
		MaxSize:    50,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[config.Level] && config.Level != "" {
		t.Errorf("Invalid log level: %s", config.Level)
	}

	if config.MaxSize < 0 {
		t.Error("MaxSize should not be negative")
	}

	if config.MaxBackups < 0 {
		t.Error("MaxBackups should not be negative")
	}

	if config.MaxAge < 0 {
		t.Error("MaxAge should not be negative")
	}
}

// TestAssetConfig tests the AssetConfig structure
func TestAssetConfig(t *testing.T) {
	config := config.AssetConfig{
		MaxSize: 5242880, // 5MB
	}

	if config.MaxSize <= 0 {
		t.Error("MaxSize should be positive")
	}
}

// TestAntivirusConfig tests the AntivirusConfig structure
func TestAntivirusConfig(t *testing.T) {
	config := config.AntivirusConfig{
		URL: "http://antivirus-service:8080",
	}

	if config.URL == "" {
		t.Error("URL should not be empty")
	}
}

// TestTemporaryAddedUserPasswordConfig tests the structure
func TestTemporaryAddedUserPasswordConfig(t *testing.T) {
	config := config.TemporaryAddedUserPasswordConfig{
		Value: "TempPassword123!",
	}

	if config.Value == "" {
		t.Error("Value should not be empty")
	}

	if len(config.Value) < 8 {
		t.Error("Password should be at least 8 characters")
	}
}

// TestOwnerRegistrationConfig tests the OwnerRegistrationConfig structure
func TestOwnerRegistrationConfig(t *testing.T) {
	config := config.OwnerRegistrationConfig{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Password:  "SecurePass123!",
	}

	if config.FirstName == "" {
		t.Error("FirstName should not be empty")
	}

	if config.LastName == "" {
		t.Error("LastName should not be empty")
	}

	if config.Email == "" {
		t.Error("Email should not be empty")
	}

	if config.Password == "" {
		t.Error("Password should not be empty")
	}
}

// TestCORSConfig tests the CORSConfig structure
func TestCORSConfig(t *testing.T) {
	config := config.CORSConfig{
		AllowedOrigins:   "*",
		AllowedMethods:   "GET,POST,PUT,DELETE",
		AllowedHeaders:   "Content-Type,Authorization",
		AllowCredentials: true,
	}

	if config.AllowedOrigins == "" {
		t.Error("AllowedOrigins should not be empty")
	}

	if config.AllowedMethods == "" {
		t.Error("AllowedMethods should not be empty")
	}
}

// TestGlobalAppConfig tests the global AppConfig variable
func TestGlobalAppConfig(t *testing.T) {
	// Save original value
	original := config.AppConfig

	// Test setting global config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: "8080",
			Host: "localhost",
		},
	}

	config.AppConfig = cfg

	if config.AppConfig.Server.Port != "8080" {
		t.Error("Global AppConfig not set correctly")
	}

	// Restore original value
	config.AppConfig = original
}
