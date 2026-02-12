package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

var AppConfig *Config

type Config struct {
	Server                     ServerConfig                     `mapstructure:"server"`
	Auth                       AuthConfig                       `mapstructure:"auth"`
	Email                      EmailConfig                      `mapstructure:"email"`
	Storage                    StorageConfig                    `mapstructure:"storage"`
	Log                        LogConfig                        `mapstructure:"log"`
	Asset                      AssetConfig                      `mapstructure:"asset"`
	Antivirus                  AntivirusConfig                  `mapstructure:"antivirus"`
	TemporaryAddedUserPassword TemporaryAddedUserPasswordConfig `mapstructure:"temporary_added_user_password"`
	OwnerRegistration          OwnerRegistrationConfig          `mapstructure:"owner_registration"`
	CORS                       CORSConfig                       `mapstructure:"cors"`
	Database                   DatabaseConfig                   `mapstructure:"database"`
}

type ServerConfig struct {
	Port         string `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	Scheme       string `mapstructure:"scheme"` // "http" or "https"
	Env          string `mapstructure:"env"`    // "dev" or "prod"
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	DatabaseName string `mapstructure:"database_name"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	Driver       string `mapstructure:"driver"`
}

type EmailConfig struct {
	URL string `mapstructure:"url"`
	// SMTPHost     string `mapstructure:"smtp_host"`
	// SMTPPort     string `mapstructure:"smtp_port"`
	// SMTPUsername string `mapstructure:"smtp_username"`
	// SMTPPassword string `mapstructure:"smtp_password"`
	// FromEmail    string `mapstructure:"from_email"`
	// FromName     string `mapstructure:"from_name"`
}

type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	AccessTokenExpiry  int    `mapstructure:"access_token_expiry"`
	RefreshTokenExpiry int    `mapstructure:"refresh_token_expiry"`
	Issuer             string `mapstructure:"issuer"`
}

type AuthConfig struct {
	URL              string    `mapstructure:"url"` // Kept for backward compat if needed, or remove if unused. User asked to remove social/keycloak.
	ResetPasswordURL string    `mapstructure:"reset_password_url"`
	JWT              JWTConfig `mapstructure:"jwt"`
}

type StorageConfig struct {
	URL string `mapstructure:"url"`
}

type AntivirusConfig struct {
	URL string `mapstructure:"url"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	ErrorFile  string `mapstructure:"error_file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type AssetConfig struct {
	MaxSize int `mapstructure:"max_size"`
}

// TemporaryAddedUserPasswordConfig holds config for temporary password assigned to newly added users so they can change it later
type TemporaryAddedUserPasswordConfig struct {
	Value string `mapstructure:"value"`
}

// OwnerRegistrationConfig holds config for pre-registering an owner user
type OwnerRegistrationConfig struct {
	FirstName string `mapstructure:"first_name"`
	LastName  string `mapstructure:"last_name"`
	Email     string `mapstructure:"email"`
	Password  string `mapstructure:"password"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   string `mapstructure:"allowed_origins"`
	AllowedMethods   string `mapstructure:"allowed_methods"`
	AllowedHeaders   string `mapstructure:"allowed_headers"`
	AllowCredentials bool   `mapstructure:"allow_credentials"`
}

func Load() (*Config, error) {
	// Load .env file
	_ = godotenv.Load()

	// Setup Viper to read from environment variables
	viper.AutomaticEnv()

	// Set environment variable binding prefix
	viper.SetEnvPrefix("")

	// Bind individual environment variables
	// Server Config
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	viper.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	viper.BindEnv("server.scheme", "SERVER_SCHEME")
	viper.BindEnv("server.env", "SERVER_ENV")

	// Database Config
	viper.BindEnv("database.host", "DATABASE_HOST")
	viper.BindEnv("database.port", "DATABASE_PORT")
	viper.BindEnv("database.username", "DATABASE_USERNAME")
	viper.BindEnv("database.password", "DATABASE_PASSWORD")
	viper.BindEnv("database.database_name", "DATABASE_NAME")
	viper.BindEnv("database.ssl_mode", "DATABASE_SSL_MODE")
	viper.BindEnv("database.max_open_conns", "DATABASE_MAX_OPEN_CONNS")
	viper.BindEnv("database.max_idle_conns", "DATABASE_MAX_IDLE_CONNS")
	viper.BindEnv("database.driver", "DATABASE_DRIVER")

	// Auth Config
	viper.BindEnv("auth.url", "AUTH_URL")
	viper.BindEnv("auth.reset_password_url", "AUTH_RESET_PASSWORD_URL")
	viper.BindEnv("auth.jwt.secret", "AUTH_JWT_SECRET")
	viper.BindEnv("auth.jwt.access_token_expiry", "AUTH_JWT_ACCESS_TOKEN_EXPIRY")
	viper.BindEnv("auth.jwt.refresh_token_expiry", "AUTH_JWT_REFRESH_TOKEN_EXPIRY")
	viper.BindEnv("auth.jwt.issuer", "AUTH_JWT_ISSUER")

	// Redis Config
	viper.BindEnv("redis.enabled", "REDIS_ENABLED")
	viper.BindEnv("redis.url", "REDIS_URL")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")

	// Email Config
	viper.BindEnv("email.url", "EMAIL_URL")

	// Antivirus Config
	viper.BindEnv("antivirus.url", "ANTIVIRUS_URL")

	// Storage Config
	viper.BindEnv("storage.url", "STORAGE_URL")

	// Log Config
	viper.BindEnv("log.level", "LOG_LEVEL")
	viper.BindEnv("log.file", "LOG_FILE")
	viper.BindEnv("log.max_size", "LOG_MAX_SIZE")
	viper.BindEnv("log.max_backups", "LOG_MAX_BACKUPS")
	viper.BindEnv("log.max_age", "LOG_MAX_AGE")
	viper.BindEnv("log.compress", "LOG_COMPRESS")

	// Asset Config
	viper.BindEnv("asset.max_size", "ASSET_MAX_SIZE")

	// Temporary User Password Config
	viper.BindEnv("temporary_added_user_password.value", "TEMPORARY_USER_PASSWORD")

	// Owner Registration Config
	viper.BindEnv("owner_registration.first_name", "OWNER_FIRST_NAME")
	viper.BindEnv("owner_registration.last_name", "OWNER_LAST_NAME")
	viper.BindEnv("owner_registration.email", "OWNER_EMAIL")
	viper.BindEnv("owner_registration.password", "OWNER_PASSWORD")

	// CORS Config
	viper.BindEnv("cors.allowed_origins", "CORS_ALLOWED_ORIGINS")
	viper.BindEnv("cors.allowed_methods", "CORS_ALLOWED_METHODS")
	viper.BindEnv("cors.allowed_headers", "CORS_ALLOWED_HEADERS")
	viper.BindEnv("cors.allow_credentials", "CORS_ALLOW_CREDENTIALS")

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.scheme", "http")
	viper.SetDefault("server.env", "dev")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.username", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.database_name", "serenibase")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.driver", "postgres")

	viper.SetDefault("auth.url", "http://localhost:8081")
	viper.SetDefault("auth.reset_password_url", "http://localhost:5050/reset-password?token=%s")
	viper.SetDefault("auth.jwt.access_token_expiry", 3600)
	viper.SetDefault("auth.jwt.refresh_token_expiry", 86400)
	viper.SetDefault("auth.jwt.secret", "default-secret-change-me")
	viper.SetDefault("auth.jwt.issuer", "serenibase")

	viper.SetDefault("email.url", "http://localhost:8082/api/v1/email")

	viper.SetDefault("storage.driver", "dev")
	viper.SetDefault("storage.dev.path", "./assets")
	viper.SetDefault("storage.minio.endpoint", "192.170.1.29:9000")
	viper.SetDefault("storage.minio.access_key", "minioadmin")
	viper.SetDefault("storage.minio.secret_key", "minioadmin")
	viper.SetDefault("storage.minio.bucket", "serenibase")
	viper.SetDefault("storage.minio.use_ssl", false)
	viper.SetDefault("storage.minio.region", "us-east-1")
	viper.SetDefault("storage.aws.access_key", "YOUR_AWS_ACCESS_KEY")
	viper.SetDefault("storage.aws.secret_key", "YOUR_AWS_SECRET_KEY")
	viper.SetDefault("storage.aws.bucket", "your-s3-bucket-name")
	viper.SetDefault("storage.aws.region", "us-east-1")
	viper.SetDefault("storage.aws.use_ssl", true)

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "app.log")
	viper.SetDefault("log.max_size", 50)
	viper.SetDefault("log.max_backups", 10)
	viper.SetDefault("log.max_age", 30)
	viper.SetDefault("log.compress", true)

	viper.SetDefault("asset.max_size", 5242880)

	viper.SetDefault("antivirus.driver", "clamav")
	viper.SetDefault("antivirus.clamav.address", "192.170.1.77:3310")
	viper.SetDefault("antivirus.clamav.timeout_seconds", 30)

	viper.SetDefault("temporary_added_user_password.value", "FC4i;<S8q?~0")

	viper.SetDefault("owner_registration.first_name", "Admin")
	viper.SetDefault("owner_registration.last_name", "User")
	viper.SetDefault("owner_registration.email", "admin@example.com")
	viper.SetDefault("owner_registration.password", "CHANGEME_OWNER_PASSWORD")

	viper.SetDefault("cors.allowed_origins", "*")
	viper.SetDefault("cors.allowed_methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
	viper.SetDefault("cors.allowed_headers", "Content-Type,Content-Length,Accept-Encoding,X-CSRF-Token,Authorization,accept,origin,Cache-Control,X-Requested-With,schema,workspace,base")
	viper.SetDefault("cors.allow_credentials", true)

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
