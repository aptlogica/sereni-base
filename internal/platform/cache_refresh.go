// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aptlogica/sereni-base/internal/providers/logger"
)

const (
	// refreshDebounce lets several schema writes in a row cause one rebuild
	refreshDebounce = time.Second
	// refreshSweep is how often cached bases are compared with their schema_version, which also covers
	// changes made in the dashboard and by other server instances
	refreshSweep = 30 * time.Second
)

// cacheRefresher rebuilds a base's schema snapshot and OpenAPI spec in the background, so the request after
// a schema change, and the docs page, find them warm. The request path still checks schema_version, so a
// slow or missed refresh is never wrong, only cold.
type cacheRefresher struct {
	api *API

	mu      sync.Mutex
	pending map[string]baseRef // by base ID: queued, waiting for the debounce
	wake    chan struct{}
}

func newCacheRefresher(a *API) *cacheRefresher {
	return &cacheRefresher{api: a, pending: map[string]baseRef{}, wake: make(chan struct{}, 1)}
}

// enqueue asks for a rebuild of the base's caches
func (r *cacheRefresher) enqueue(base baseRef) {
	r.mu.Lock()
	r.pending[base.ID] = base
	r.mu.Unlock()
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// run is the worker: it rebuilds queued bases after the debounce and sweeps now and then. It returns when
// ctx is done.
func (r *cacheRefresher) run(ctx context.Context) {
	sweep := time.NewTicker(refreshSweep)
	defer sweep.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.wake:
			select {
			case <-ctx.Done():
				return
			case <-time.After(refreshDebounce):
			}
			r.drain(ctx)
		case <-sweep.C:
			r.sweepStale(ctx)
		}
	}
}

func (r *cacheRefresher) drain(ctx context.Context) {
	r.mu.Lock()
	batch := r.pending
	r.pending = map[string]baseRef{}
	r.mu.Unlock()
	for _, base := range batch {
		r.refresh(ctx, base.PublicID)
	}
}

// refresh reads the base as it is now and warms its snapshot and spec
func (r *cacheRefresher) refresh(ctx context.Context, publicID string) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	base, found, err := r.api.readBase(ctx, publicID)
	if err != nil || !found {
		return
	}
	if _, err := r.api.spec(ctx, base); err != nil {
		logger.Get().Warn().Err(err).Str("base", publicID).Msg("background cache refresh failed")
	}
}

// sweepStale refreshes the cached bases whose schema_version moved on
func (r *cacheRefresher) sweepStale(ctx context.Context) {
	r.api.schemas.mu.Lock()
	cached := make(map[string]int64, len(r.api.schemas.byBase))
	for id, snap := range r.api.schemas.byBase {
		cached[id] = snap.version
	}
	r.api.schemas.mu.Unlock()
	if len(cached) == 0 {
		return
	}
	rows, err := r.api.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT base_id::text, public_id, schema_version FROM %s.base_api WHERE enabled = true`, r.api.schemaIdent()))
	if err != nil {
		return
	}
	var stale []string
	for rows.Next() {
		var id, publicID string
		var version int64
		if err := rows.Scan(&id, &publicID, &version); err != nil {
			continue
		}
		if have, ok := cached[id]; ok && have != version {
			stale = append(stale, publicID)
		}
	}
	rows.Close()
	for _, publicID := range stale {
		r.refresh(ctx, publicID)
	}
}
