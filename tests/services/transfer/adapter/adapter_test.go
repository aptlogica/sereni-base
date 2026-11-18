package adapter_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/sbase"
	"github.com/aptlogica/sereni-base/internal/services/transfer/formats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

func TestRegistryFindsAdaptersByName(t *testing.T) {
	r := adapter.NewRegistry()
	_, ok := r.Get("sbase")
	assert.False(t, ok, "an empty registry knows nothing")

	r.Register(sbase.New())
	got, ok := r.Get("sbase")
	require.True(t, ok)
	assert.Equal(t, "sbase", got.Name())
	assert.Equal(t, []string{"sbase"}, r.Names())

	_, ok = r.Get("csv")
	assert.False(t, ok, "an unknown format is not found")
	assert.Panics(t, func() { r.Register(sbase.New()) }, "registering a name twice is a mistake")
}

func TestApplicationFormats(t *testing.T) {
	r := formats.Registry()
	a, ok := r.Get(sbase.Name)
	require.True(t, ok, "the application imports and exports the SereniBase format")
	caps := a.Capabilities()
	assert.True(t, caps.Structure && caps.Rows && caps.Links && caps.Lookups && caps.Views && caps.Constraints && caps.Files)
}

// An adapter reads and writes a format; it never reaches the database or a service
func TestAdaptersImportNoDatabaseOrService(t *testing.T) {
	for _, dir := range []string{"../../../../internal/services/transfer/adapter", "../../../../internal/services/transfer/adapter/sbase"} {
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, e.Name()), nil, parser.ImportsOnly)
			require.NoError(t, err)
			for _, imp := range f.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				assert.NotContains(t, path, "go-postgres-rest", "%s imports the database layer", e.Name())
				assert.NotContains(t, path, "/internal/models", "%s imports models", e.Name())
				assert.NotContains(t, path, "/providers/", "%s imports a provider", e.Name())
				if strings.Contains(path, "/internal/services/") {
					assert.Contains(t, path, "/internal/services/transfer/adapter", "%s imports a service: %s", e.Name(), path)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// sbase: Describe
// ---------------------------------------------------------------------------

func TestSbaseDescribe(t *testing.T) {
	a := sbase.New()
	// an export is always a zip, with or without rows
	name, ctype := a.Describe(adapter.ExportOptions{}, "Sales CRM")
	assert.Equal(t, "Sales_CRM.sbase.zip", name)
	assert.Equal(t, "application/zip", ctype)

	name, ctype = a.Describe(adapter.ExportOptions{IncludeData: true}, "Sales CRM")
	assert.Equal(t, "Sales_CRM.sbase.zip", name)
	assert.Equal(t, "application/zip", ctype)

	// characters that are not safe in a file name, and a title with nothing usable in it
	name, _ = a.Describe(adapter.ExportOptions{}, `a/b\c:"d"`)
	assert.Equal(t, "a_b_c_d.sbase.zip", name)
	name, _ = a.Describe(adapter.ExportOptions{IncludeData: true}, "///")
	assert.Equal(t, "base.sbase.zip", name)
}

// ---------------------------------------------------------------------------
// sbase: Export
// ---------------------------------------------------------------------------

func model() *dto.BaseExportFile {
	return &dto.BaseExportFile{
		Base:   dto.BaseExportBase{Title: "CRM", Description: "amount >= 0 & b < c"},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{{Key: "t_a", Title: "A", Fields: []dto.BaseExportField{}}}},
	}
}

func withFiles() *dto.BaseExportFile {
	m := model()
	m.Base.Cover = &dto.BaseExportCover{Filename: "cover.png", ContentType: "image/png", Path: "attachments/cover_cover.png", StoragePath: "covers/1"}
	m.Data = &dto.BaseExportData{
		RowCounts: map[string]int{},
		Tables:    map[string]dto.BaseExportTableRow{},
		Attachments: []dto.BaseExportAttachment{
			{Ref: "a1", Filename: "note.txt", ContentType: "text/plain", Path: "attachments/a1_note.txt", StoragePath: "files/note"},
		},
	}
	return m
}

type memoryFiles map[string]string

func (m memoryFiles) OpenFile(_ context.Context, path string) (io.ReadCloser, error) {
	data, ok := m[path]
	if !ok {
		return nil, errors.New("no such file " + path)
	}
	return io.NopCloser(strings.NewReader(data)), nil
}

// zipEntry returns the text of one file of a zip.
func zipEntry(t *testing.T, data []byte, name string) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err, "an export is always a zip")
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			require.NoError(t, err)
			defer rc.Close()
			raw, err := io.ReadAll(rc)
			require.NoError(t, err)
			return string(raw)
		}
	}
	t.Fatalf("the zip has no %s", name)
	return ""
}

// Without data the export is still a zip: it holds base.sbase.json, readable and indented, and no other file.
func TestSbaseExportWithoutDataIsAZipWithReadableJSON(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, sbase.New().Export(context.Background(), model(), adapter.ExportOptions{}, &out))

	text := zipEntry(t, out.Bytes(), "base.sbase.json")
	assert.True(t, strings.HasPrefix(text, "{\n  \"base\": {\n    \"title\": \"CRM\""), "indented by two spaces: %s", text)
	assert.True(t, strings.HasSuffix(text, "}\n"))
	assert.Contains(t, text, "amount >= 0 & b < c", "SQL and conditions stay readable, not escaped")
	assert.NotContains(t, text, "\\"+"u003e", "> must not be written as an escape")
}

func TestSbaseExportWithDataIsAZipWithTheFilesAndTheIndexMatchesThem(t *testing.T) {
	files := memoryFiles{"files/note": "hello note", "covers/1": "PNGDATA"}
	m := withFiles()
	var out bytes.Buffer
	require.NoError(t, sbase.New().Export(context.Background(), m, adapter.ExportOptions{IncludeData: true, OpenFile: files.OpenFile}, &out))

	zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	require.NoError(t, err)
	names := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		names[f.Name] = string(b)
	}
	assert.Equal(t, "hello note", names["attachments/a1_note.txt"])
	assert.Equal(t, "PNGDATA", names["attachments/cover_cover.png"])
	require.Contains(t, names, "base.sbase.json")

	// the sizes and checksums in the index are those of the bytes that were really packed
	sum := sha256.Sum256([]byte("hello note"))
	assert.Equal(t, int64(len("hello note")), m.Data.Attachments[0].Size)
	assert.Equal(t, "sha256:"+hex.EncodeToString(sum[:]), m.Data.Attachments[0].SHA256)
	assert.Equal(t, int64(len("PNGDATA")), m.Base.Cover.Size)
	assert.Contains(t, names["base.sbase.json"], `"sha256": "sha256:`+hex.EncodeToString(sum[:])+`"`)
	assert.NotContains(t, names["base.sbase.json"], "storage_path", "where a file is read from is never written")
}

func TestSbaseExportWithFilesNeedsAnOpenFile(t *testing.T) {
	err := sbase.New().Export(context.Background(), withFiles(), adapter.ExportOptions{IncludeData: true}, io.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no storage provider")

	// a model without files needs none
	require.NoError(t, sbase.New().Export(context.Background(), model(), adapter.ExportOptions{IncludeData: true}, io.Discard))
}

func TestSbaseExportReportsAFileThatCannotBeRead(t *testing.T) {
	err := sbase.New().Export(context.Background(), withFiles(), adapter.ExportOptions{IncludeData: true, OpenFile: memoryFiles{}.OpenFile}, io.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `read "note.txt"`)
}

// ---------------------------------------------------------------------------
// sbase: Import
// ---------------------------------------------------------------------------

func TestSbaseImportReadsWhatExportWrote(t *testing.T) {
	a := sbase.New()
	files := memoryFiles{"files/note": "hello note", "covers/1": "PNGDATA"}

	// plain JSON (an export is a zip, but import still reads a bare .sbase.json)
	var structure bytes.Buffer
	require.NoError(t, a.Export(context.Background(), model(), adapter.ExportOptions{}, &structure))
	js := bytes.NewBufferString(zipEntry(t, structure.Bytes(), "base.sbase.json"))
	got, err := a.Import(context.Background(), bytes.NewReader(js.Bytes()), int64(js.Len()), adapter.ImportOptions{Filename: "crm.sbase.json"})
	require.NoError(t, err)
	assert.Equal(t, "CRM", got.File.Base.Title)
	assert.Nil(t, got.Attachments, "plain JSON carries no files")

	// zip with files
	var zp bytes.Buffer
	require.NoError(t, a.Export(context.Background(), withFiles(), adapter.ExportOptions{IncludeData: true, OpenFile: files.OpenFile}, &zp))
	got, err = a.Import(context.Background(), bytes.NewReader(zp.Bytes()), int64(zp.Len()), adapter.ImportOptions{Filename: "CRM.SBASE.ZIP"})
	require.NoError(t, err)
	assert.Equal(t, "CRM", got.File.Base.Title)
	require.NotNil(t, got.Attachments)
	assert.True(t, got.Attachments.Has("attachments/a1_note.txt"))
	rc, err := got.Attachments.Open("attachments/a1_note.txt")
	require.NoError(t, err)
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	assert.Equal(t, "hello note", string(data))
}

func TestSbaseImportSaysWhatIsWrongWithTheFile(t *testing.T) {
	a := sbase.New()
	run := func(name string, data []byte, opt adapter.ImportOptions) error {
		opt.Filename = name
		_, err := a.Import(context.Background(), bytes.NewReader(data), int64(len(data)), opt)
		return err
	}

	err := run("x.sbase.json", []byte("{not json"), adapter.ImportOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "the file is not valid JSON")

	// a typo in a property is an error, not a setting that is dropped without a word
	err = run("x.sbase.json", []byte(`{"base":{"title":"X"},"schema":{"tables":[{"key":"t_a","title":"A","primary_keys":["f"]}]}}`), adapter.ImportOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "primary_keys")

	err = run("x.zip", []byte("this is not a zip"), adapter.ImportOptions{})
	assert.EqualError(t, err, "the file is not a valid zip")

	// a zip without base.sbase.json
	var zp bytes.Buffer
	zw := zip.NewWriter(&zp)
	w, _ := zw.Create("other.txt")
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	err = run("x.zip", zp.Bytes(), adapter.ImportOptions{})
	assert.EqualError(t, err, "the zip has no base.sbase.json")

	// a model over the size limit is refused before it is read
	var big bytes.Buffer
	zw = zip.NewWriter(&big)
	w, _ = zw.Create("base.sbase.json")
	_, _ = w.Write([]byte(`{"base":{"title":"` + strings.Repeat("x", 5000) + `"},"schema":{"tables":[]},"data":null}`))
	_ = zw.Close()
	err = run("x.zip", big.Bytes(), adapter.ImportOptions{MaxBytes: 1000})
	assert.EqualError(t, err, "base.sbase.json in the zip is too large")
}

// a zip is read by position, so the input must support it (an uploaded file does)
func TestSbaseImportOfAZipNeedsARandomAccessInput(t *testing.T) {
	_, err := sbase.New().Import(context.Background(), io.MultiReader(strings.NewReader("PK")), 2, adapter.ImportOptions{Filename: "x.zip"})
	assert.EqualError(t, err, "the file could not be read")
}

func TestSbaseImportRefusesTheFormatBlockOfOlderFiles(t *testing.T) {
	old := `{"format":{"name":"serenibase.base","version":"1.1.0"},"base":{"title":"Old"},"schema":{"tables":[]},"data":null}`
	_, err := sbase.New().Import(context.Background(), strings.NewReader(old), int64(len(old)), adapter.ImportOptions{Filename: "old.sbase.json"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "format"`)
}
