// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package scripts

import (
	"context"
	"fmt"
	"strings"

	"github.com/aptlogica/go-postgres-rest/pkg"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/platform"
)

// SeedAPIWorkspaces turns the public API on for the workspaces with these slugs (comma-separated,
// PUBLIC_API_SEED_WORKSPACES) and prints each one's API address. Workspaces keep their public ID across runs.
func SeedAPIWorkspaces(repo *pkg.DatabaseService, slugs string) {
	if repo == nil || repo.DB == nil {
		return
	}
	svc := platform.NewWorkspaceAPIService(repo)
	for _, slug := range strings.Split(slugs, ",") {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		var workspaceID string
		err := repo.DB.QueryRow(fmt.Sprintf(`SELECT id::text FROM %s WHERE slug = $1`,
			tenant.Workspace{}.TableName(appConstant.MasterDatabase)), slug).Scan(&workspaceID)
		if err != nil {
			fmt.Printf("⚠ Warning: public API seed: no workspace with slug %q\n", slug)
			continue
		}
		out, err := svc.Enable(context.Background(), appConstant.MasterDatabase, workspaceID)
		if err != nil {
			fmt.Printf("⚠ Warning: public API seed: could not enable workspace %q: %v\n", slug, err)
			continue
		}
		fmt.Printf("Public API enabled for workspace %q: %s (%s)\n", slug, out.PublicID, out.BasePath)
	}
}
