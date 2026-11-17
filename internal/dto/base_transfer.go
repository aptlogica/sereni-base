// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"encoding/json"
	"io"
	"time"
)

const (
	// BaseExportRefPrefix marks an id inside a meta object that was replaced by a file key, e.g. "@f_name"
	BaseExportRefPrefix = "@"
)

// BaseExportFile is the content of a .sbase.json file. It holds no database ids: tables, fields and views are
// named by keys, rows by _rid, and ids inside meta are written as "@key".
type BaseExportFile struct {
	Base   BaseExportBase   `json:"base"`
	Schema BaseExportSchema `json:"schema"`
	Data   *BaseExportData  `json:"data"`
	// Skipped lists what export could not write (unsupported field types, files over the size limit, ...). It is
	// only logged by the server: it is never written to the file and never returned to the caller.
	Skipped []string `json:"-"`
}

type BaseExportBase struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	// Cover is the cover image; its file is in the zip under Path
	Cover *BaseExportCover `json:"cover,omitempty"`
}

// BaseExportCover describes the cover image file in the zip.
type BaseExportCover struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256,omitempty"`
	// StoragePath is where export reads the file from; it is never written to the file
	StoragePath string `json:"-"`
}

type BaseExportSchema struct {
	Tables []BaseExportTable `json:"tables"`
	// Views is the list of format 1.0.0, where each view names its table. Import still reads it; export writes
	// views inside their table instead.
	Views []BaseExportView `json:"views,omitempty"`
}

type BaseExportTable struct {
	Key         string            `json:"key"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	OrderIndex  float64           `json:"order_index"`
	Fields      []BaseExportField `json:"fields"`
	// Views are the saved views of this table (format 1.1.0 and later)
	Views []BaseExportView `json:"views,omitempty"`
	// PrimaryKey lists the key fields in key order; left out when the table uses the default Id key
	PrimaryKey  []string               `json:"primary_key,omitempty"`
	Constraints []BaseExportConstraint `json:"constraints,omitempty"`
	Functions   []BaseExportFunction   `json:"functions,omitempty"`
	Triggers    []BaseExportTrigger    `json:"triggers,omitempty"`
}

// BaseExportFunction is a trigger function ("CREATE FUNCTION name() RETURNS trigger ..."). The SQL is written with
// placeholders instead of names that differ in a copy: {t_key} for a table, {f_key} for a column, {fn_key} for a
// function (including this one), {schema} for the schema. Key is how triggers and functions refer to this function.
type BaseExportFunction struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Context string `json:"context"`
}

// BaseExportTrigger is a trigger ("CREATE TRIGGER name ... ON table ... EXECUTE FUNCTION f()") with the same placeholders.
// Event is the timing and operations, e.g. "BEFORE INSERT, BEFORE UPDATE".
type BaseExportTrigger struct {
	Title   string `json:"title"`
	Event   string `json:"event,omitempty"`
	Context string `json:"context"`
}

// BaseExportConstraint is a unique or check constraint of a table. A check condition is SQL in which the
// column names are written as {field_key}.
type BaseExportConstraint struct {
	Type      string   `json:"type"`
	Title     string   `json:"title"`
	FieldKeys []string `json:"field_keys,omitempty"`
	Category  string   `json:"category,omitempty"`
	Condition string   `json:"condition,omitempty"`
}

// BaseExportField is a column. System fields (id, created time, ...) are listed with their column name so ids
// in meta that point at them can be resolved; import does not create them, the new table already has them.
type BaseExportField struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	UIDT        string `json:"uidt"`
	// DT is the database column type the field had when it was exported (TEXT, NUMERIC, JSONB[], ...). It is
	// information only: import never reads it, because the app derives it from UIDT.
	DT         string                 `json:"dt,omitempty"`
	OrderIndex float64                `json:"order_index,omitempty"`
	System     bool                   `json:"system,omitempty"`
	ColumnName string                 `json:"column_name,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

type BaseExportView struct {
	Key string `json:"key"`
	// Table is only used in the top-level list of format 1.0.0; a view inside a table does not name it
	Table       string                 `json:"table,omitempty"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"`
	OrderIndex  float64                `json:"order_index,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

type BaseExportData struct {
	RowCounts map[string]int                `json:"row_counts"`
	Tables    map[string]BaseExportTableRow `json:"tables"`
	// Attachments lists the files used by attachment cells, each once. A cell holds the ref of its files.
	Attachments []BaseExportAttachment `json:"attachments,omitempty"`
}

// BaseExportAttachment is one file in the zip, under Path.
type BaseExportAttachment struct {
	Ref         string `json:"ref"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256,omitempty"`
	// StoragePath is where export reads the file from; it is never written to the file
	StoragePath string `json:"-"`
}

// AttachmentSource gives import the files of a zip, by path.
type AttachmentSource interface {
	Has(path string) bool
	Open(path string) (io.ReadCloser, error)
}

type BaseExportTableRow struct {
	Rows []BaseExportRow `json:"rows"`
}

// BaseExportRow is one record. Values are keyed by field key; a link cell is an array of _rid.
type BaseExportRow struct {
	RID    string                 `json:"_rid"`
	Values map[string]interface{} `json:"values"`
	// order is the order in which Values are written (see OrderBy); it is not part of the file
	order []string
}

// ExportBaseOptions are the options of a base export
type ExportBaseOptions struct {
	IncludeData bool
	// IncludeAttachments lists the attachment files in the file; the export adapter packs them. Needs IncludeData.
	IncludeAttachments bool
	// IncludeAutomations writes the trigger functions and triggers of the tables. They hold SQL, so the handler sets
	// it only for a user who has the automations permission.
	IncludeAutomations bool
	// Tables limits the export to specific tables by ID, title, or key. If empty, all tables are exported.
	Tables []string
}

// ImportBaseOptions are the options of a base import
type ImportBaseOptions struct {
	// Title replaces the title in the file when set
	Title string
	// IncludeData is ignored when the file has no data
	IncludeData bool
	// Attachments holds the files of a zip upload; nil for a plain JSON upload
	Attachments AttachmentSource
	// IncludeAttachments imports the attachment files of a zip and the cover image. When false they are left out and the attachment cells stay empty.
	IncludeAttachments bool
	// IncludeAutomations creates the trigger functions and triggers of the file in Postgres. They run SQL from the
	// file, so this is off by default and needs the automations permission.
	IncludeAutomations bool
}

// ImportBaseResult is what an import reports
type ImportBaseResult struct {
	BaseID      string   `json:"base_id,omitempty"`
	Title       string   `json:"title"`
	Tables      int      `json:"tables"`
	Fields      int      `json:"fields"`
	Views       int      `json:"views"`
	Rows        int      `json:"rows"`
	Files       int      `json:"attachments,omitempty"`
	Cover       bool     `json:"cover_image,omitempty"`
	Constraints int      `json:"constraints,omitempty"`
	Functions   int      `json:"functions,omitempty"`
	Triggers    int      `json:"triggers,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

// ImportValidationError lists everything wrong with a file, so the user can fix it in one pass
type ImportValidationError struct {
	Problems []string
}

func (e *ImportValidationError) Error() string {
	b, _ := json.Marshal(e.Problems)
	return "invalid base file: " + string(b)
}

// An export is completed when its zip can be downloaded (DownloadURL, FileURL) until ExpiresAt. An import is completed when its new base exists (BaseID, Result).
type TransferJobResponse struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	BaseID      string          `json:"base_id,omitempty"`
	BaseTitle   string          `json:"base_title,omitempty"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	Status      string          `json:"status"`
	Progress    int             `json:"progress"`
	FileName    string          `json:"file_name,omitempty"`
	SizeBytes   int64           `json:"size_bytes,omitempty"`
	Error       string          `json:"error,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	Reused      bool            `json:"reused,omitempty"`
	DownloadURL string          `json:"download_url,omitempty"`
	FileURL     string          `json:"file_url,omitempty"`
}

// Kinds of ExportDependency
const (
	ExportDependencyLink    = "link"
	ExportDependencyLookup  = "lookup"
	ExportDependencyTrigger = "trigger"
)

// ExportTableInfo is a table that can be picked for an export
type ExportTableInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ExportDependency says that a table needs another table of the same base: one of its link fields points at it, a
// lookup or rollup reads through such a link, or a trigger or function uses it. Exporting the table without the other
// table leaves that item out.
type ExportDependency struct {
	TableID        string `json:"table_id"`
	TableTitle     string `json:"table_title"`
	DependsOnID    string `json:"depends_on_id"`
	DependsOnTitle string `json:"depends_on_title"`
	// Kind is one of the ExportDependency* constants
	Kind string `json:"kind"`
	// Item is the title of the field, trigger or function
	Item string `json:"item"`
}

// ExportDependencies lists the tables of a base and what depends on what, so a client can warn before an export that
// leaves a table out
type ExportDependencies struct {
	Tables       []ExportTableInfo  `json:"tables"`
	Dependencies []ExportDependency `json:"dependencies"`
}
