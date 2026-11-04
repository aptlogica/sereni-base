// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package scripts

import (
	"fmt"

	"github.com/aptlogica/go-postgres-rest/pkg"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
)

// InstallSchemaVersionTriggers (re)installs the triggers that bump workspace_api.schema_version whenever a base,
// table or field changes. Safe to run on every start.
func InstallSchemaVersionTriggers(repo *pkg.DatabaseService, schemaName string) error {
	if repo == nil || repo.DB == nil {
		return nil
	}
	if _, err := repo.DB.Exec(fmt.Sprintf(appConstant.InstallSchemaVersionTriggersQuery, quoteIdent(schemaName))); err != nil {
		return fmt.Errorf("install schema version triggers: %w", err)
	}
	return nil
}
