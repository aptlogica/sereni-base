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

// SeedAPIBases turns the public API on for every active base of the workspaces with these slugs
// (comma-separated, PUBLIC_API_SEED_WORKSPACES) and prints each base's API address. Bases keep their
// public ID across runs.
func SeedAPIBases(repo *pkg.DatabaseService, slugs string) {
	if repo == nil || repo.DB == nil {
		return
	}
	svc := platform.NewBaseAPIService(repo)
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
		rows, err := repo.DB.Query(fmt.Sprintf(
			`SELECT id::text, COALESCE(title, '') FROM %s WHERE workspace_id::text = $1 AND COALESCE(status, 'active') = 'active' ORDER BY created_time, id`,
			tenant.Base{}.TableName(appConstant.MasterDatabase)), workspaceID)
		if err != nil {
			fmt.Printf("⚠ Warning: public API seed: could not list the bases of workspace %q: %v\n", slug, err)
			continue
		}
		type seedBase struct{ id, title string }
		var bases []seedBase
		for rows.Next() {
			var b seedBase
			if err := rows.Scan(&b.id, &b.title); err != nil {
				fmt.Printf("⚠ Warning: public API seed: could not read a base of workspace %q: %v\n", slug, err)
				continue
			}
			bases = append(bases, b)
		}
		rows.Close()
		for _, b := range bases {
			out, err := svc.Enable(context.Background(), appConstant.MasterDatabase, b.id)
			if err != nil {
				fmt.Printf("⚠ Warning: public API seed: could not enable base %q: %v\n", b.title, err)
				continue
			}
			fmt.Printf("Public API enabled for base %q (workspace %q): %s (%s)\n", b.title, slug, out.PublicID, out.BasePath)
		}
	}
}
