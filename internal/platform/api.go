// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	pgInterfaces "github.com/aptlogica/go-postgres-rest/pkg/database/interfaces"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// RecordWriter writes records for the API: the table service's API write methods and its link writes
type RecordWriter interface {
	interfaces.APIRecordService
	UpdateRawDataForLinks(ctx context.Context, schemaName string, req dto.UpdateRowDataLinksRequest) (dto.RecordResponse, error)
}

// API serves the workspace public API at /api/v1/{ws}
type API struct {
	db                 pgInterfaces.DB
	schema             string // schema holding the metadata and the records tables
	records            RecordWriter
	rateLimitPerMinute int
	workspaces         *workspaceCache
	schemas            *schemaCache
	now                func() time.Time
}

// NewAPI returns the public API. schema is the schema of the metadata tables (constant.MasterDatabase).
func NewAPI(db pgInterfaces.DB, schema string, records RecordWriter, rateLimitPerMinute int) *API {
	if rateLimitPerMinute <= 0 {
		rateLimitPerMinute = 600
	}
	return &API{
		db:                 db,
		schema:             schema,
		records:            records,
		rateLimitPerMinute: rateLimitPerMinute,
		workspaces:         &workspaceCache{entries: map[string]workspaceEntry{}},
		schemas:            &schemaCache{byWorkspace: map[string]*schemaSnapshot{}},
		now:                time.Now,
	}
}

func (a *API) schemaIdent() string { return pq.QuoteIdentifier(a.schema) }

// Register adds the /:ws routes to the /api/v1 group. Existing /api/v1/<name> routes always win; any other
// first segment is looked up as a workspace public ID.
func (a *API) Register(api *gin.RouterGroup) {
	ws := api.Group("/:ws", a.resolveWorkspace(), authPlaceholder(), a.rateLimit())

	ws.GET("/bases", a.listBases)
	ws.GET("/bases/:base/tables", a.listTables)
	ws.GET("/bases/:base/tables/:table", a.getTable)

	records := ws.Group("/bases/:base/tables/:table/records")
	records.GET("", a.listRecords)
	records.GET("/:record", a.getRecord)
	records.POST("", a.idempotent(), a.createRecords)
	records.PATCH("/:record", a.updateRecord)
	records.PATCH("", a.updateRecords)
	records.DELETE("/:record", a.deleteRecord)
	records.POST("/delete", a.idempotent(), a.deleteRecords)
}

// NotFound answers unknown /api/v1/ws_... paths in the API's envelope
func NotFound(c *gin.Context) {
	respondError(c, notFound("No such endpoint. See the API docs for the available paths."))
}

// ---- Pipeline: resolve workspace, auth placeholder, rate limit ----

const ctxWorkspace = "platform.workspace"

var publicIDPattern = regexp.MustCompile(`^ws_[0-9a-z]{10}$`)

// resolveWorkspace turns the ws path segment into the workspace; unknown and disabled workspaces are 404
func (a *API) resolveWorkspace() gin.HandlerFunc {
	return func(c *gin.Context) {
		publicID := c.Param("ws")
		if !publicIDPattern.MatchString(publicID) {
			respondError(c, notFound("No such workspace or endpoint."))
			return
		}
		ref, found, err := a.lookupWorkspace(c, publicID)
		if err != nil {
			respondError(c, err)
			return
		}
		if !found {
			respondError(c, notFound(fmt.Sprintf("No workspace with ID %q, or its API is turned off.", publicID)))
			return
		}
		c.Set(ctxWorkspace, ref)
		c.Set(middleware.PublicWorkspaceIDKey, ref.PublicID)
		c.Next()
	}
}

func currentWorkspace(c *gin.Context) workspaceRef {
	ref, _ := c.Get(ctxWorkspace)
	ws, _ := ref.(workspaceRef)
	return ws
}

// authPlaceholder is where API keys plug in during the security stage. The routes are only registered
// when PUBLIC_API_OPEN_ACCESS is on, which the server refuses in production.
func authPlaceholder() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

// rateLimit allows rateLimitPerMinute requests per workspace per minute
func (a *API) rateLimit() gin.HandlerFunc {
	return middleware.RateLimiterBy(
		func(c *gin.Context) string { return currentWorkspace(c).ID },
		a.rateLimitPerMinute,
		func(c *gin.Context, retryAfter time.Duration) {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			respondError(c, NewError(CodeRateLimited, fmt.Sprintf(
				"Too many requests: this workspace allows %d per minute. Try again after the time in Retry-After.", a.rateLimitPerMinute)))
		})
}

// ---- Path resolution shared by the handlers ----

// tableContext is the workspace, base, table and fields a records request addresses
type tableContext struct {
	ws     workspaceRef
	base   *baseInfo
	table  *tableInfo
	fields []*field
}

func (a *API) resolveTablePath(c *gin.Context) (*tableContext, error) {
	ws := currentWorkspace(c)
	base, err := a.resolveBase(c, ws, c.Param("base"))
	if err != nil {
		return nil, err
	}
	table, err := a.resolveTable(c, ws, base, c.Param("table"))
	if err != nil {
		return nil, err
	}
	fields, err := a.fields(c, ws, table)
	if err != nil {
		return nil, err
	}
	return &tableContext{ws: ws, base: base, table: table, fields: fields}, nil
}

// recordsTable is the quoted physical table of the records
func (a *API) recordsTable(t *tableInfo) string {
	return a.schemaIdent() + "." + pq.QuoteIdentifier(t.Alias)
}

// statusFor is 201 for created records, else 200
func statusFor(created bool) int {
	if created {
		return http.StatusCreated
	}
	return http.StatusOK
}

// splitList returns the comma-separated values of a query parameter, without blanks
func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
