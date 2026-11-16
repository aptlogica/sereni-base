// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package formats lists the file formats the application can import and export.
package formats

import (
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/sbase"
)

// Registry returns a registry with every format registered. Adding a format is one more Register line.
func Registry() *adapter.Registry {
	r := adapter.NewRegistry()
	r.Register(sbase.New())
	return r
}
