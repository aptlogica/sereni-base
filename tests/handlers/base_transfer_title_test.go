// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/gin-gonic/gin"
)

type capturingTransfer struct {
	interfaces.BaseTransferService
	got    dto.ImportBaseOptions
	file   *dto.BaseExportFile
	called bool
	// exported are the options of the last export
	exported dto.ExportBaseOptions
}

func (c *capturingTransfer) ImportBase(_ context.Context, _, _, _ string, _ *dto.BaseExportFile, opts dto.ImportBaseOptions) (dto.ImportBaseResult, error) {
	c.got = opts
	c.called = true
	return dto.ImportBaseResult{Title: opts.Title}, nil
}

const minimalBaseFile = `{"base":{"title":"From file"},"schema":{"tables":[]},"data":null}`

func importRequest(t *testing.T, fields map[string]string) *capturingTransfer {
	t.Helper()
	svc, _ := importBody(t, fields, minimalBaseFile)
	return svc
}

// importBody uploads body as the file and returns the service that was called and the recorded response.
func importBody(t *testing.T, fields map[string]string, body string) (*capturingTransfer, *httptest.ResponseRecorder) {
	t.Helper()
	return importBodyAt(t, "/", fields, body)
}

// importBodyAt is importBody with a URL, so a parameter can be sent in the query string.
func importBodyAt(t *testing.T, url string, fields map[string]string, body string) (*capturingTransfer, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := &capturingTransfer{}
	h := handlers.NewBaseTransferHandler(svc)

	var payload bytes.Buffer
	mw := multipart.NewWriter(&payload)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	part, _ := mw.CreateFormFile("file", "x.sbase.json")
	_, _ = io.WriteString(part, body)
	_ = mw.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, url, &payload)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "ws"}}
	c.Set("schema", "s")
	c.Set("user_id", "u")
	h.ImportBase(c)
	return svc, w
}

func TestImportBaseReadsTitleFormField(t *testing.T) {
	if got := importRequest(t, map[string]string{"title": "My new name"}).got.Title; got != "My new name" {
		t.Fatalf("title = %q", got)
	}
	// only the form field named "title" is read; other names are ignored and the file's title is used
	for _, key := range []string{"name", "base_name", "baseName", "Title"} {
		if got := importRequest(t, map[string]string{key: "My new name"}).got.Title; got != "" {
			t.Fatalf("field %q must not be read, got %q", key, got)
		}
	}
}

func (c *capturingTransfer) ExportBase(_ context.Context, _ string, _ string, opts dto.ExportBaseOptions) (*dto.BaseExportFile, error) {
	c.exported = opts
	return c.file, nil
}

func exportRequest(t *testing.T, query string, skipped []string) *httptest.ResponseRecorder {
	t.Helper()
	_, w := exportWith(t, query, skipped)
	return w
}

func exportWith(t *testing.T, query string, skipped []string) (*capturingTransfer, *httptest.ResponseRecorder) {
	t.Helper()
	return exportAs(t, query, skipped, false)
}

// exportAs is exportWith for a user who has (or has not) the automations permission, as the router records it.
func exportAs(t *testing.T, query string, skipped []string, automationsAllowed bool) (*capturingTransfer, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := &capturingTransfer{file: &dto.BaseExportFile{
		Base:    dto.BaseExportBase{Title: "CRM"},
		Skipped: skipped,
	}}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	c.Params = gin.Params{{Key: "id", Value: "33cc3ba3-9fac-43d8-866f-0bd1e44548eb"}}
	c.Set("schema", "s")
	c.Set(handlers.AutomationsAllowedKey, automationsAllowed)
	handlers.NewBaseTransferHandler(svc).ExportBase(c)
	return svc, w
}

// include_data is the only export option: without it the download is the structure as .sbase.json, with it a
// .sbase.zip that also carries the rows, the attachment files and the cover image.
func TestExportBaseDownloadsAZipWhenDataIsIncluded(t *testing.T) {
	plain := exportRequest(t, "", nil)
	if plain.Code != http.StatusOK || !strings.HasPrefix(plain.Header().Get("Content-Type"), "application/json") ||
		!strings.Contains(plain.Header().Get("Content-Disposition"), ".sbase.json") {
		t.Fatalf("structure export: status %d, headers %v", plain.Code, plain.Header())
	}
	svc, zipped := exportWith(t, "include_data=true", nil)
	if zipped.Code != http.StatusOK || zipped.Header().Get("Content-Type") != "application/zip" ||
		!strings.Contains(zipped.Header().Get("Content-Disposition"), ".sbase.zip") || !isSbaseZip(zipped.Body.Bytes(), "CRM") {
		t.Fatalf("data export: status %d, headers %v", zipped.Code, zipped.Header())
	}
	if !svc.exported.IncludeData || !svc.exported.IncludeAttachments {
		t.Fatalf("include_data must also include the attachment files and the cover: %+v", svc.exported)
	}
	if svc, _ = exportWith(t, "", nil); svc.exported.IncludeData || svc.exported.IncludeAttachments {
		t.Fatalf("without include_data nothing but the structure is exported: %+v", svc.exported)
	}
}

// The old options are gone: they are ignored, so they cannot switch anything on or off.
func TestExportBaseIgnoresTheRemovedOptions(t *testing.T) {
	svc, w := exportWith(t, "include_attachments=true&include_automations=true", nil)
	if w.Code != http.StatusOK || svc.exported.IncludeAttachments || svc.exported.IncludeAutomations {
		t.Fatalf("status %d, options %+v", w.Code, svc.exported)
	}
}

func TestExportBaseReturnsNothingAboutWhatItSkipped(t *testing.T) {
	w := exportRequest(t, "include_data=true", []string{"field Orders.Margin (rollup) not exported"})
	for name := range w.Header() {
		if strings.HasPrefix(name, "X-Export") {
			t.Fatalf("unexpected header %s", name)
		}
	}
	if strings.Contains(w.Body.String(), "rollup") || strings.Contains(w.Body.String(), "warnings") {
		t.Fatalf("the response must not carry what was skipped: %s", w.Body.String())
	}
}

func TestImportRejectsPropertiesTheSchemaDoesNotDefine(t *testing.T) {
	// a typo in an optional property would otherwise be dropped without a word
	typo := `{"base":{"title":"X"},"schema":{"tables":[{"key":"t_a","title":"A","order_index":1,"fields":[],"primary_keys":["f_a"]}]},"data":null}`
	svc, w := importBody(t, nil, typo)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "primary_keys") {
		t.Fatalf("the message must name the unknown property: %s", w.Body.String())
	}
	if svc.got.Title != "" || svc.called {
		t.Fatal("nothing may be imported from a file with an unknown property")
	}
}

// The format block and base.type of older layouts are no longer part of the file, so the strict reader refuses them
func TestImportRejectsTheFormatBlockAndBaseTypeOfOlderFiles(t *testing.T) {
	for name, body := range map[string]string{
		"format block": `{"format":{"name":"serenibase.base","version":"1.1.0"},"base":{"title":"From file"},"schema":{"tables":[]},"data":null}`,
		"base.type":    `{"base":{"title":"From file","type":"default"},"schema":{"tables":[]},"data":null}`,
	} {
		svc, w := importBody(t, nil, body)
		if w.Code != http.StatusBadRequest || svc.called {
			t.Fatalf("%s: status = %d, called = %v, body = %s", name, w.Code, svc.called, w.Body.String())
		}
	}
}

func TestAutomationsAreExportedForAUserWhoMayReadThemAndImportedOnlyWhenAsked(t *testing.T) {
	svc, _ := exportAs(t, "include_data=true", nil, true)
	if !svc.exported.IncludeAutomations {
		t.Fatal("export must include triggers and functions for a user who may read them")
	}
	svc, w := exportAs(t, "include_data=true", nil, false)
	if w.Code != http.StatusOK || svc.exported.IncludeAutomations {
		t.Fatalf("a user without the automations permission still gets the export, without them: status %d, %+v", w.Code, svc.exported)
	}

	if importRequest(t, nil).got.IncludeAutomations {
		t.Fatal("import must not create triggers and functions unless asked")
	}
	if !importRequest(t, map[string]string{"include_automations": "true"}).got.IncludeAutomations {
		t.Fatal("include_automations=true must reach the service")
	}
	_, w = importBody(t, map[string]string{"include_automations": "maybe"}, minimalBaseFile)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("a value that is not a boolean must be rejected, got %d", w.Code)
	}
}

func TestExportedJSONKeepsSQLReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &capturingTransfer{file: &dto.BaseExportFile{
		Base: dto.BaseExportBase{Title: "CRM"},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{{
			Key: "t_a", Title: "A", Fields: []dto.BaseExportField{},
			Constraints: []dto.BaseExportConstraint{{Type: "check", Title: "Not negative", Condition: "{f_amount} >= 0 AND a < b & c"}},
		}}},
	}}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "33cc3ba3-9fac-43d8-866f-0bd1e44548eb"}}
	c.Set("schema", "s")
	handlers.NewBaseTransferHandler(svc).ExportBase(c)

	body := w.Body.String()
	if !strings.Contains(body, "{f_amount} >= 0 AND a < b & c") || strings.Contains(body, `\u003e`) {
		t.Fatalf("the downloaded file escapes characters of the SQL:\n%s", body)
	}
}

func TestImportTakesAutomationsFromTheFormOrTheURL(t *testing.T) {
	cases := []struct {
		name   string
		url    string
		fields map[string]string
		want   bool
	}{
		{"form field", "/", map[string]string{"include_automations": "true"}, true},
		{"URL parameter", "/?include_automations=true", nil, true},
		{"form field wins over the URL", "/?include_automations=true", map[string]string{"include_automations": "false"}, false},
		{"a blank form field falls back to the URL", "/?include_automations=true", map[string]string{"include_automations": " "}, true},
		{"nothing asked", "/", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, w := importBodyAt(t, c.url, c.fields, minimalBaseFile)
			if w.Code != http.StatusOK || svc.got.IncludeAutomations != c.want {
				t.Fatalf("status %d, IncludeAutomations %v, want %v", w.Code, svc.got.IncludeAutomations, c.want)
			}
		})
	}
	if _, w := importBodyAt(t, "/?include_automations=maybe", nil, minimalBaseFile); w.Code != http.StatusBadRequest {
		t.Fatalf("a value that is not a boolean must be rejected, got %d", w.Code)
	}
}

// isSbaseZip reports whether data is a SereniBase zip: base.sbase.json inside, for a base called title. The adapter
// wrote it; the handler no longer writes any file format itself.
func isSbaseZip(data []byte, title string) bool {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false
	}
	for _, f := range zr.File {
		if f.Name != "base.sbase.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return false
		}
		defer rc.Close()
		var doc struct {
			Base struct {
				Title string `json:"title"`
			} `json:"base"`
		}
		return json.NewDecoder(rc).Decode(&doc) == nil && doc.Base.Title == title
	}
	return false
}
