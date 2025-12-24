package config

import (
	"github.com/spf13/viper"
)

var AppConfig *Config

type Config struct {
	Server                     ServerConfig                     `mapstructure:"server"`
	Auth                       AuthConfig                       `mapstructure:"auth"`
	Redis                      RedisConfig                      `mapstructure:"redis"`
	Email                      EmailConfig                      `mapstructure:"email"`
	Storage                    StorageConfig                    `mapstructure:"storage"`
	Log                        LogConfig                        `mapstructure:"log"`
	Asset                      AssetConfig                      `mapstructure:"asset"`
	Antivirus                  AntivirusConfig                  `mapstructure:"antivirus"`
	TemporaryAddedUserPassword TemporaryAddedUserPasswordConfig `mapstructure:"temporary_added_user_password"`
	OwnerRegistration          OwnerRegistrationConfig          `mapstructure:"owner_registration"`
}

type ServerConfig struct {
	Port         string `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	Scheme       string `mapstructure:"scheme"` // "http" or "https"
	Env          string `mapstructure:"env"`    // "dev" or "prod"
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
	Driver string             `mapstructure:"driver"`
	Dev    StorageDevConfig   `mapstructure:"dev"`
	Minio  StorageMinioConfig `mapstructure:"minio"`
	AWS    StorageAWSConfig   `mapstructure:"aws"`
}

type StorageDevConfig struct {
	Path string `mapstructure:"path"`
}

type StorageMinioConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	Region    string `mapstructure:"region"`
}

type StorageAWSConfig struct {
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

type AntivirusConfig struct {
	Driver string       `mapstructure:"driver"`
	ClamAV ClamAVConfig `mapstructure:"clamav"`
}

type ClamAVConfig struct {
	Address        string `mapstructure:"address"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
}

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	URL      string `mapstructure:"url"`
	Password string `mapstructure:"password"`
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

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)

	viper.SetDefault("auth.jwt.access_token_expiry", 3600)   // 1 hour
	viper.SetDefault("auth.jwt.refresh_token_expiry", 86400) // 24 hours
	viper.SetDefault("auth.jwt.secret", "default-secret-change-me")
	viper.SetDefault("auth.jwt.issuer", "serenibase")

	viper.SetDefault("redis.enabled", false)
	viper.SetDefault("redis.url", "redis://localhost:6379")
	viper.SetDefault("auth.mode", "dev")
	viper.SetDefault("server.scheme", "https")
	// viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
