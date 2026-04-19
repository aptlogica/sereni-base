/*

Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
This file is part of software developed by Aptlogica Technologies Private Limited.
Licensed under the Apache License, Version 2.0. See the LICENSE file in the project root
for full license information.
Websites:
https://www.aptlogica.com
https://www.serenibase.com
Support:
support@aptlogica.com
support@serenibase.com
*/

package main

import (
	"log"
	"os"
	"strings"

	"github.com/aptlogica/sereni-base/internal/app"
	"github.com/aptlogica/sereni-base/internal/config"
)

var version = "0.1.0-beta"

// validateSecrets checks that critical secrets are not using insecure default values
func validateSecrets(cfg *config.Config) error {
	insecureValues := []string{
		"changeme", "password", "secret", "admin123", "123456",
		"your-secure", "example", "test", "default",
	}

	// Check JWT secret
	jwtSecret := strings.ToLower(cfg.Auth.JWT.Secret)
	for _, insecure := range insecureValues {
		if strings.Contains(jwtSecret, insecure) {
			log.Printf("WARNING: JWT secret appears to use an insecure value")
			break
		}
	}

	// Check minimum JWT secret length
	if len(cfg.Auth.JWT.Secret) < 32 {
		log.Printf("WARNING: JWT secret should be at least 32 characters long")
	}

	// Check database password
	dbPassword := strings.ToLower(cfg.Database.Password)
	for _, insecure := range insecureValues {
		if strings.Contains(dbPassword, insecure) {
			log.Printf("WARNING: Database password appears to use an insecure value")
			break
		}
	}

	// In production, these warnings should be fatal
	if cfg.Server.Env == "prod" || cfg.Server.Env == "production" {
		if len(cfg.Auth.JWT.Secret) < 32 {
			return &SecretValidationError{msg: "JWT secret must be at least 32 characters in production"}
		}
	}

	return nil
}

// SecretValidationError represents a secret validation failure
type SecretValidationError struct {
	msg string
}

func (e *SecretValidationError) Error() string {
	return e.msg
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	cfg.Server.Version = version

	// Validate secrets before starting
	if err := validateSecrets(cfg); err != nil {
		log.Fatal("Secret validation failed:", err)
		os.Exit(1)
	}

	application, err := app.New(cfg)
	config.AppConfig = cfg
	if err != nil {
		log.Fatal("Failed to create application:", err)
	}

	if err := application.Run(); err != nil {
		log.Fatal("Failed to run application:", err)
	}
}
