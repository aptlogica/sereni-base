// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package scripts

import (
	"fmt"

	"github.com/aptlogica/go-postgres-rest/pkg"
)

// MigratePublicAPIToBase moves the public API from workspaces to bases. Public IDs now belong to a base
// (base_api, bs_...) so the old workspace_api table and the workspace-keyed api_idempotency_keys cannot be
// carried over: workspace_api is dropped, and api_idempotency_keys is dropped when it still has the
// workspace_id column (it only holds retry data kept for 24 hours). Both are recreated by the normal table
// creation afterwards. Safe to run on every start: with nothing old left it does nothing.
func MigratePublicAPIToBase(repo *pkg.DatabaseService, schemaName string) {
	if repo == nil || repo.DB == nil {
		return
	}
	quoted := quoteIdent(schemaName)
	if _, err := repo.DB.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s.workspace_api`, quoted)); err != nil {
		fmt.Printf("⚠ Warning: could not drop the old workspace_api table: %v\n", err)
	}
	// One statement, so the check and the drop cannot race another instance starting up
	drop := fmt.Sprintf(`DO $$ BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns
			WHERE table_schema = '%s' AND table_name = 'api_idempotency_keys' AND column_name = 'workspace_id') THEN
			DROP TABLE %s.api_idempotency_keys;
		END IF;
	END $$`, escapeLiteral(schemaName), quoted)
	if _, err := repo.DB.Exec(drop); err != nil {
		fmt.Printf("⚠ Warning: could not replace the old api_idempotency_keys table: %v\n", err)
	}
}
