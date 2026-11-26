// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/gin-gonic/gin"
)

// schemaLockWait is how long a schema write waits for another one on the same base before answering 503
const schemaLockWait = 30 * time.Second

// SchemaServices are the app services the base management routes call. They are the same services the
// dashboard uses, so every rule the app enforces applies here too.
type SchemaServices struct {
	Tables      interfaces.TableManagementService
	Constraints interfaces.TableConstraintService
	PrimaryKeys interfaces.PrimaryKeyService
}

// WithSchemaServices turns on the base management routes
func (a *API) WithSchemaServices(s SchemaServices) *API {
	a.schemaSvc = &s
	a.refresher = newCacheRefresher(a)
	return a
}

// StartBackground starts the cache refresh worker and sweep; they stop with ctx
func (a *API) StartBackground(ctx context.Context) {
	if a.refresher != nil {
		go a.refresher.run(ctx)
	}
}

// ---- Locks: one schema write per base at a time ----

type schemaLocks struct {
	mu    sync.Mutex
	byKey map[string]chan struct{}
}

func (l *schemaLocks) chanFor(key string) chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.byKey == nil {
		l.byKey = map[string]chan struct{}{}
	}
	ch, ok := l.byKey[key]
	if !ok {
		ch = make(chan struct{}, 1)
		l.byKey[key] = ch
	}
	return ch
}

// acquire waits up to wait for the base's lock; the returned func releases it
func (l *schemaLocks) acquire(ctx context.Context, key string, wait time.Duration) (func(), bool) {
	ch := l.chanFor(key)
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, true
	case <-timer.C:
		return nil, false
	case <-ctx.Done():
		return nil, false
	}
}

// ---- Middleware ----

// schemaWrite guards every base management write route: the per-base switch, and a per-base lock so two
// schema changes never interleave. After a successful write it queues a cache refresh.
//
// There is no secret key check yet: like every route of the base API today, these routes run without
// authentication (the API only exists with PUBLIC_API_OPEN_ACCESS=true, which prod refuses). The security
// task adds the secret key check (and its error message) in front of requireFeature.
func (a *API) schemaWrite() []gin.HandlerFunc {
	return []gin.HandlerFunc{a.requireFeature(featureBaseManagement), a.lockSchema()}
}

// requireFeature answers 403 feature_disabled while the feature is off for the base
func (a *API) requireFeature(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !currentBase(c).Settings.enabled(feature) {
			respondError(c, NewError(CodeFeatureDisabled,
				"Base management is off for this base. Turn it on in the Developer settings."))
			return
		}
		c.Next()
	}
}

func (a *API) lockSchema() gin.HandlerFunc {
	return func(c *gin.Context) {
		base := currentBase(c)
		release, ok := a.locks.acquire(c.Request.Context(), base.ID, schemaLockWait)
		if !ok {
			c.Header("Retry-After", "5")
			respondError(c, NewError(CodeUnavailable, "Another schema change on this base is still running. Try again shortly."))
			return
		}
		defer release()
		c.Next()
		if !c.IsAborted() && c.Writer.Status() < http.StatusBadRequest && a.refresher != nil {
			a.refresher.enqueue(base)
		}
	}
}

// actor is the user schema writes are made in the name of: the user who turned base management on, or the
// base owner when that user has left the base or was deleted
func (a *API) actor(ctx context.Context, base baseRef) string {
	enabledBy := base.Settings.BaseManagement.EnabledBy
	if enabledBy != "" && a.userIsActive(base, enabledBy) {
		return enabledBy
	}
	logger.Get().Warn().Str("base_id", base.ID).Str("enabled_by", enabledBy).Str("owner", base.Owner).
		Msg("base management actor is gone; schema write falls back to the base owner")
	return base.Owner
}

func (a *API) userIsActive(base baseRef, userID string) bool {
	var one int
	err := a.db.QueryRow(fmt.Sprintf(
		`SELECT 1 FROM %[1]s.users u WHERE u.id::text = $1 AND COALESCE(u.is_deleted, false) = false
		   AND EXISTS (SELECT 1 FROM %[1]s.workspace_members m WHERE m.user_id::text = $1 AND m.workspace_id::text = $2)`,
		a.schemaIdent()), userID, base.WorkspaceID).Scan(&one)
	return err == nil
}

// guardedError turns an error of the app's services into an API error that says what is wrong; errors
// that are not recognised are returned as they are (and become internal_error)
func guardedError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*Error); ok {
		return err
	}
	if stopped := stoppedWriteError(err); stopped != nil {
		return stopped
	}
	if m := userMessage(err); m != cantSaveMessage {
		return NewError(CodeValidationFailed, m)
	}
	return err
}

// schemaTableOut is a table as base management returns it
type schemaTableOut struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	BaseID      string `json:"base_id"`
}

func schemaTableOf(base baseRef, m dto.ModelResponse) schemaTableOut {
	return schemaTableOut{
		ID: EncodeUUID(PrefixTable, m.ID.String()), Name: m.Title, Description: m.Description,
		BaseID: EncodeUUID(PrefixBase, base.ID),
	}
}
