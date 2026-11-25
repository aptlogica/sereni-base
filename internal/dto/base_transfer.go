// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"encoding/json"
	"io"
	"strings"
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

// Attachments gives import the files a file format carried, by path. It is a plain struct (not an interface) so
// that the adapter layer has one interface only: adapter.Adapter. A nil *Attachments means the file carried none.
type Attachments struct {
	Has  func(path string) bool
	Open func(path string) (io.ReadCloser, error)
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
	// IncludeSystemColumns writes the source (system) columns of every row: id, created and modified time, created
	// and modified by. Needs IncludeData.
	IncludeSystemColumns bool
	// LinksAsText exports every link field (and the lookups and rollups built on it) as long text, whether or not the
	// linked table is exported. Set for formats that cannot hold links (csv).
	LinksAsText bool
	// Format is the file format of the export: "sbase" (the default, also when empty), "csv" or "xlsx".
	Format string
	// Delimiter is the field separator of a delimited-text format (csv): ',', ';', '\t' or '|'. Zero means ','.
	Delimiter rune
	// Tables limits the export to specific tables by ID, title, or key. If empty, all tables are exported.
	Tables []string
	// Filters keeps only the rows that match, per table (by id, title or alias, like Tables). A table with no entry
	// exports all its rows. Only used with IncludeData.
	Filters map[string]ExportTableFilter
}

// ExportFilterCondition is one condition of a row filter. It is the condition the filter builder of a table view
// makes. Column is the id of the field.
type ExportFilterCondition struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	// Logic joins this condition to the ones before it: "AND" (default) or "OR". The first condition has none.
	Logic string `json:"logic,omitempty"`
}

// ExportTableFilter is the filter of one table: its conditions, combined from first to last
type ExportTableFilter struct {
	Conditions []ExportFilterCondition `json:"conditions"`
}

// ExportBaseRequest is the body of POST /base/:id/export: the options of the GET request, and the row filters
type ExportBaseRequest struct {
	IncludeData        *bool `json:"include_data"`
	IncludeAttachments *bool `json:"include_attachments"`
	IncludeAutomations *bool `json:"include_automations"`
	// IncludeSystemColumns adds the source columns (id, created / modified time and user) to the rows. Needs include_data.
	IncludeSystemColumns *bool `json:"include_system_columns"`
	// Format is "sbase" (default), "csv" or "xlsx"; Delimiter is "comma" (default), "semicolon", "tab" or "pipe" and is
	// only used by csv.
	Format    string                       `json:"format"`
	Delimiter string                       `json:"delimiter"`
	Tables    []string                     `json:"tables"`
	Filters   map[string]ExportTableFilter `json:"filters"`
}

// ExportFilterError says why the filters of an export are not valid, in words for the user
type ExportFilterError struct {
	Problems []string
}

func (e *ExportFilterError) Error() string {
	return "the export filters are not valid: " + strings.Join(e.Problems, "; ")
}

// ImportUpload is the uploaded file of a base import: its content, name and size in bytes
type ImportUpload struct {
	File     io.Reader
	Filename string
	Size     int64
}

// ImportBaseOptions are the options of a base import
type ImportBaseOptions struct {
	// Title replaces the title in the file when set
	Title string
	// IncludeData is ignored when the file has no data
	IncludeData bool
	// Attachments holds the files of a zip upload; nil for a plain JSON upload
	Attachments *Attachments
	// IncludeAttachments imports the attachment files of a zip and the cover image. When false they are left out and the attachment cells stay empty.
	IncludeAttachments bool
	// IncludeAutomations creates the trigger functions and triggers of the file in Postgres. They run SQL from the
	// file, so this is off by default and needs the automations permission.
	IncludeAutomations bool
	// IncludeSystemColumns restores the source columns of the rows from the file: id, created / modified time and
	// created / modified by. When false the rows get a new id, the time of the import and the importing user.
	IncludeSystemColumns bool
	// Format is the file format of the upload: "sbase" (the default, also when empty), "csv" (a zip of .csv
	// files, which becomes a new base with one table per file) or "xlsx" (a workbook, one table per sheet).
	Format string
	// SkipFailedRows imports the rows the database accepts and leaves out the ones it refuses, instead of failing the
	// whole import on the first one. The rows left out are listed in ImportBaseResult.ErrorLog. Set for a csv or xlsx, which
	// has no schema that could have promised the values would fit.
	SkipFailedRows bool
	// Source says where each row came from in the file (by _rid), for the error log of a skipped row; it is not part
	// of the file format, so an adapter that has no line numbers leaves it nil.
	Source *ImportSource
	// FileName is the name of the uploaded file; it is only written in the header of the error log
	FileName string
}

// ImportSource says where the rows of an import came from, so a row that is refused can be named by its file and
// line and shown as it was written
type ImportSource struct {
	// Rows is by _rid
	Rows map[string]ImportRowSource
	// EmptyRows and DuplicateRows count the rows of the files that were blank, or the same as an earlier row. They only
	// go in the report: blank rows were never imported, and duplicate rows are imported as they are.
	EmptyRows, DuplicateRows int
}

// ImportRowSource is one row as it was in its file
type ImportRowSource struct {
	// File and Table name the file and the table made from it
	File, Table string
	// Line is the line of the row in the file; the header is line 1
	Line int
	// Headers and Cells are the header row and this row, as text
	Headers, Cells []string
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
	// SkippedRows counts the rows the database refused and that were left out (only with SkipFailedRows); ErrorLog
	// is the text of the report that names each one and says why. EmptyRows and DuplicateRows are counted from the
	// files for the report.
	SkippedRows   int    `json:"skipped_rows,omitempty"`
	EmptyRows     int    `json:"empty_rows,omitempty"`
	DuplicateRows int    `json:"duplicate_rows,omitempty"`
	ErrorLog      string `json:"error_log,omitempty"`
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
	// HasLog is true for an import that has an error log to download (GET /import-job/:id/log): rows were left out,
	// there are warnings, or the import failed. SkippedRows and WarningCount are the counts behind it.
	HasLog       bool `json:"has_log,omitempty"`
	SkippedRows  int  `json:"skipped_rows,omitempty"`
	WarningCount int  `json:"warning_count,omitempty"`
}

// Kinds of ExportDependency
const (
	ExportDependencyLink    = "link"
	ExportDependencyLookup  = "lookup"
	ExportDependencyTrigger = "trigger"
)

// ExportFieldInfo is a field of a table that a row filter can use
type ExportFieldInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	UIDT  string `json:"uidt"`
	// Options are the choices of a select or multi-select field
	Options []string `json:"options,omitempty"`
}

// ExportTableInfo is a table that can be picked for an export
type ExportTableInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Fields are the fields a row filter can use
	Fields []ExportFieldInfo `json:"fields"`
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
