// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// Swagger UI's files are the ones swaggo/files embeds, which the router already serves at /swagger/
var swaggerUIPage = template.Must(template.New("docs").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<link rel="stylesheet" href="/swagger/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="/swagger/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({url: {{.SpecURL}}, dom_id: "#swagger-ui", deepLinking: true, tryItOutEnabled: true,
  presets: [SwaggerUIBundle.presets.apis]});
</script>
</body>
</html>
`))

// ReDoc comes from jsDelivr, pinned, as FastAPI does
var redocPage = template.Must(template.New("redoc").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>body { margin: 0; padding: 0; }</style>
</head>
<body>
<redoc spec-url="{{.SpecURL}}"></redoc>
<script src="https://cdn.jsdelivr.net/npm/redoc@2.1.5/bundles/redoc.standalone.js"></script>
</body>
</html>
`))

// GET /docs: Swagger UI for the workspace's spec
func (a *API) swaggerUI(c *gin.Context) { docsPage(c, swaggerUIPage, "API docs") }

// GET /redoc: ReDoc for the workspace's spec
func (a *API) redoc(c *gin.Context) { docsPage(c, redocPage, "API reference") }

// docsPage renders a page that loads ./openapi.json, passing on ?base=
func docsPage(c *gin.Context, page *template.Template, title string) {
	ws := currentWorkspace(c)
	specURL := "openapi.json"
	if base := c.Query("base"); base != "" {
		specURL += "?base=" + url.QueryEscape(base)
	}
	var html bytes.Buffer
	if err := page.Execute(&html, map[string]string{"Title": ws.PublicID + " " + title, "SpecURL": specURL}); err != nil {
		respondError(c, err)
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", html.Bytes())
}
