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

// API serves the base public API at /api/v1/{bs}
type API struct {
	db                 pgInterfaces.DB
	schema             string // schema holding the metadata and the records tables
	records            RecordWriter
	rateLimitPerMinute int
	bases              *baseCache
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
		bases:              &baseCache{entries: map[string]baseEntry{}},
		schemas:            &schemaCache{byBase: map[string]*schemaSnapshot{}},
		now:                time.Now,
	}
}

func (a *API) schemaIdent() string { return pq.QuoteIdentifier(a.schema) }

// Register adds the /:bs routes to the /api/v1 group. Existing /api/v1/<name> routes always win; any other
// first segment is looked up as a base public ID.
func (a *API) Register(api *gin.RouterGroup) {
	bs := api.Group("/:bs", a.resolveBase(), authPlaceholder(), a.rateLimit())

	// The base's own spec and its docs pages, as FastAPI serves them
	bs.GET("/openapi.json", a.openAPISpec)
	bs.GET("/docs", a.swaggerUI)
	bs.GET("/redoc", a.redoc)

	bs.GET("/tables", a.listTables)
	bs.GET("/tables/:table", a.getTable)
	bs.GET("/tables/:table/columns", a.listColumns)

	records := bs.Group("/tables/:table/records")
	records.GET("", a.listRecords)
	records.GET("/:record", a.getRecord)
	records.POST("", a.idempotent(), a.createRecords)
	records.PATCH("/:record", a.updateRecord)
	records.PATCH("", a.updateRecords)
	records.DELETE("/:record", a.deleteRecord)
	records.POST("/delete", a.idempotent(), a.deleteRecords)
	// The HTTP QUERY method (a read with a body) does the same as POST; clients use whichever they support
	records.POST("/query", a.queryRecords)
	records.Handle(MethodQuery, "/query", a.queryRecords)
	records.POST("/aggregate", a.aggregateRecords)
	records.Handle(MethodQuery, "/aggregate", a.aggregateRecords)
}

// MethodQuery is the HTTP QUERY method: safe and idempotent like GET, with a body like POST
const MethodQuery = "QUERY"

// NotFound answers unknown /api/v1/bs_... paths in the API's envelope
func NotFound(c *gin.Context) {
	respondError(c, notFound("No such endpoint. See the API docs for the available paths."))
}

// ---- Pipeline: resolve base, auth placeholder, rate limit ----

const ctxBase = "platform.base"

var publicIDPattern = regexp.MustCompile(`^bs_[0-9a-z]{10}$`)

// resolveBase turns the bs path segment into the base; unknown, disabled and archived bases are 404
func (a *API) resolveBase() gin.HandlerFunc {
	return func(c *gin.Context) {
		publicID := c.Param("bs")
		if !publicIDPattern.MatchString(publicID) {
			respondError(c, notFound("No such base or endpoint."))
			return
		}
		ref, found, err := a.lookupBase(c, publicID)
		if err != nil {
			respondError(c, err)
			return
		}
		if !found {
			respondError(c, notFound(fmt.Sprintf("No base with ID %q, or its API is turned off.", publicID)))
			return
		}
		c.Set(ctxBase, ref)
		c.Set(middleware.PublicBaseIDKey, ref.PublicID)
		// For respondError: did a missing column or table come from a schema change during this request?
		c.Set(ctxSchemaCheck, func() bool {
			now, found, err := a.readBase(c.Request.Context(), publicID)
			return err == nil && (!found || now.SchemaVersion != ref.SchemaVersion)
		})
		c.Next()
	}
}

func currentBase(c *gin.Context) baseRef {
	ref, _ := c.Get(ctxBase)
	base, _ := ref.(baseRef)
	return base
}

// authPlaceholder is where API keys plug in during the security stage. The routes are only registered
// when PUBLIC_API_OPEN_ACCESS is on, which the server refuses in production.
func authPlaceholder() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

// rateLimit allows rateLimitPerMinute requests per base per minute
func (a *API) rateLimit() gin.HandlerFunc {
	return middleware.RateLimiterBy(
		func(c *gin.Context) string { return currentBase(c).ID },
		a.rateLimitPerMinute,
		func(c *gin.Context, retryAfter time.Duration) {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			respondError(c, NewError(CodeRateLimited, fmt.Sprintf(
				"Too many requests: this base allows %d per minute. Try again after the time in Retry-After.", a.rateLimitPerMinute)))
		})
}

// ---- Path resolution shared by the handlers ----

// tableContext is the base, table and fields a records request addresses
type tableContext struct {
	base   baseRef
	table  *tableInfo
	fields []*field
	key    *recordKey // the table's primary key, read when a record is addressed
}

func (a *API) resolveTablePath(c *gin.Context) (*tableContext, error) {
	base := currentBase(c)
	table, err := a.resolveTable(c, base, c.Param("table"))
	if err != nil {
		return nil, err
	}
	fields, err := a.fields(c, base, table)
	if err != nil {
		return nil, err
	}
	return &tableContext{base: base, table: table, fields: fields}, nil
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
