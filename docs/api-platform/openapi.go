// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package apiplatform embeds the base API's generic OpenAPI contract, so the server builds each
// base's spec from this file itself. go:embed can't reach files outside its own package, so the embed
// lives next to the file.
package apiplatform

import _ "embed"

// OpenAPIYAML is openapi.yaml, the generic contract of the base API
//
//go:embed openapi.yaml
var OpenAPIYAML []byte
