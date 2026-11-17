// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/router"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// withoutAutomations holds every permission except the one on automations.
type withoutAutomations struct{ regularMemberAccess }

func (withoutAutomations) CheckUserPermission(_ context.Context, _, _, _ string, _ *string, resource, _ string) (bool, error) {
	return resource != constant.ResourceCodes.Automations, nil
}

const ownBaseID = "33cc3ba3-9fac-43d8-866f-0bd1e44548eb"

type stubTransfer struct {
	interfaces.BaseTransferService
	exports, imports int
	// exported are the options of the last export
	exported dto.ExportBaseOptions
}

func (s *stubTransfer) ExportBase(_ context.Context, _, _ string, opts dto.ExportBaseOptions) (*dto.BaseExportFile, error) {
	s.exports++
	s.exported = opts
	return &dto.BaseExportFile{Base: dto.BaseExportBase{Title: "CRM"}}, nil
}

func (s *stubTransfer) ImportBase(context.Context, string, string, string, *dto.BaseExportFile, dto.ImportBaseOptions) (dto.ImportBaseResult, error) {
	s.imports++
	return dto.ImportBaseResult{Title: "CRM"}, nil
}

func transferRouter(access interfaces.AccessMemberService) (*gin.Engine, *stubTransfer) {
	gin.SetMode(gin.TestMode)
	stub := &stubTransfer{}
	middlewares := createMockMiddlewares()
	middlewares.AuthMiddleware = func() gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set("user_id", "user-b")
			c.Set("schema", "tenant")
			c.Next()
		}
	}
	middlewares.AccessMemberService = access
	// the export route needs a real UUID for the base; it belongs to the user's own workspace
	lookups := testScopeLookups()
	known := lookups.BaseWorkspace
	lookups.BaseWorkspace = func(ctx context.Context, schema, id string) (string, error) {
		if id == ownBaseID {
			return "ws-attacker", nil
		}
		return known(ctx, schema, id)
	}
	middlewares.ScopeLookups = lookups
	h := createMockHandlers()
	h.BaseTransfer = handlers.NewBaseTransferHandler(stub)
	return router.Setup(&config.Config{}, h, middlewares), stub
}

func importUpload(t *testing.T, fields map[string]string) *http.Request {
	t.Helper()
	return importUploadAt(t, "", fields)
}

// importUploadAt is importUpload with a query string such as "?include_automations=true".
func importUploadAt(t *testing.T, query string, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	part, _ := mw.CreateFormFile("file", "x.sbase.zip")
	zw := zip.NewWriter(part)
	entry, _ := zw.Create("base.sbase.json")
	_, _ = io.WriteString(entry, `{"base":{"title":"X"},"schema":{"tables":[]},"data":null}`)
	_ = zw.Close()
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/ws-attacker/import-base"+query, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// Triggers and functions run SQL, so asking for them needs the automations permission on top of the usual ones.
func TestTriggersAndFunctionsNeedTheAutomationsPermission(t *testing.T) {
	r, stub := transferRouter(withoutAutomations{})
	serve := func(req *http.Request) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	export := func(query string) *http.Request {
		return httptest.NewRequest(http.MethodGet, "/api/v1/base/"+ownBaseID+"/export"+query, nil)
	}

	// an export is always allowed; a user who lacks the permission gets it without triggers and functions
	assert.Equal(t, http.StatusOK, serve(export("?include_data=true")).Code)
	assert.Equal(t, 1, stub.exports)
	assert.False(t, stub.exported.IncludeAutomations, "the SQL must stay out of the file for a user without the permission")
	// the removed option cannot be used to ask for them
	assert.Equal(t, http.StatusOK, serve(export("?include_automations=true")).Code)
	assert.Equal(t, 2, stub.exports)
	assert.False(t, stub.exported.IncludeAutomations)

	assert.Equal(t, http.StatusOK, serve(importUpload(t, nil)).Code)
	assert.Equal(t, 1, stub.imports)
	refused := serve(importUpload(t, map[string]string{"include_automations": "true"}))
	assert.NotEqual(t, http.StatusOK, refused.Code)
	assert.Contains(t, refused.Body.String(), "ERR_0002")
	assert.Equal(t, 1, stub.imports, "the service must not run for a refused request")
}

func TestTriggersAndFunctionsAreAllowedWithThePermission(t *testing.T) {
	r, stub := transferRouter(regularMemberAccess{}) // holds every permission
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/base/"+ownBaseID+"/export?include_data=true", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, stub.exported.IncludeAutomations, "a user with the permission always gets the triggers and functions")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, importUpload(t, map[string]string{"include_automations": "true"}))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, stub.exports)
	assert.Equal(t, 1, stub.imports)
}

// The option can be sent in the URL as well as in the form; the permission is checked for either.
func TestAutomationsInTheURLNeedThePermissionToo(t *testing.T) {
	r, stub := transferRouter(withoutAutomations{})
	serve := func(req *http.Request) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	refused := serve(importUploadAt(t, "?include_automations=true", nil))
	assert.NotEqual(t, http.StatusOK, refused.Code)
	assert.Contains(t, refused.Body.String(), "ERR_0002")
	assert.Equal(t, 0, stub.imports, "the service must not run for a refused request")

	// a blank form field falls back to the URL, so it cannot be used to slip past the check
	refused = serve(importUploadAt(t, "?include_automations=true", map[string]string{"include_automations": " "}))
	assert.NotEqual(t, http.StatusOK, refused.Code)
	assert.Equal(t, 0, stub.imports)

	// the form field wins, so a false form field is an honest "no"
	assert.Equal(t, http.StatusOK, serve(importUploadAt(t, "?include_automations=true", map[string]string{"include_automations": "false"})).Code)
	assert.Equal(t, 1, stub.imports)

	allowed, stub2 := transferRouter(regularMemberAccess{})
	w := httptest.NewRecorder()
	allowed.ServeHTTP(w, importUploadAt(t, "?include_automations=true", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, stub2.imports)
}
