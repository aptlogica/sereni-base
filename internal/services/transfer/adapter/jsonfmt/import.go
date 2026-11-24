// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package jsonfmt

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/csv"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/internal/ziputil"
)

// sbaseExtension is the extension of the other JSON format, which this one must not be mixed up with
const sbaseExtension = ".sbase.json"

// Import reads an export of this format into a model: a zip (.zip) with one .json file per table and, optionally,
// the files of an attachments/ folder. One table is made per .json file, named after the file. A file is an array of
// rows; each row is an object keyed by the field title. The fields of a table are the keys of its rows, in the order
// they first appear; a key a row does not have is an empty cell. The values are read as text (a list is joined with
// ",", an object is kept as JSON text) and the types are guessed from them in the way an import of CSV files does; a
// column whose cells are all paths of files in attachments/ becomes an attachment field. The new base is named after
// the zip. The input must stay open until the import is done, because the attachment files are read from it.
//
// A single .json file is read too: it is one table (an array of rows), named after the file, and it has no
// attachment files. The new base is named after the file.
func (jsonAdapter) Import(_ context.Context, in io.Reader, size int64, opt adapter.ImportOptions) (*adapter.ImportResult, error) {
	lowerName := strings.ToLower(opt.Filename)
	isZip := strings.HasSuffix(lowerName, ".zip")
	if !isZip && !strings.HasSuffix(lowerName, Extension) {
		return nil, errors.New("only .zip or .json files are supported")
	}
	limit := opt.MaxBytes
	if limit <= 0 {
		limit = adapter.DefaultMaxImportBytes
	}
	if !isZip {
		return importSingleFile(in, opt.Filename, limit)
	}

	zr, err := openZip(in, size, limit)
	if err != nil {
		return nil, err
	}
	zipFiles, tables := collectFiles(zr)
	if len(tables) == 0 {
		return nil, errors.New("the zip has no .json file")
	}
	sheets, err := readSheets(tables, limit)
	if err != nil {
		return nil, err
	}

	model, source := csv.BuildModel(baseTitleFromFilename(opt.Filename), sheets, zipFiles)
	result := &adapter.ImportResult{File: model, Source: source}
	if len(model.Data.Attachments) > 0 {
		result.Attachments = ziputil.NewZipSource(zr)
	}
	return result, nil
}

// importSingleFile reads one .json file as one table
func importSingleFile(in io.Reader, filename string, limit int64) (*adapter.ImportResult, error) {
	name := path.Base(strings.ReplaceAll(filename, "\\", "/"))
	sheet, err := readTableDoc(name, in, limit)
	if err != nil {
		return nil, err
	}
	model, source := csv.BuildModel(baseTitleFromFilename(filename), []csv.Sheet{sheet}, map[string]*zip.File{})
	return &adapter.ImportResult{File: model, Source: source}, nil
}

// openZip opens the input as a zip; an input that cannot seek is read into memory first, within the limit
func openZip(in io.Reader, size, limit int64) (*zip.Reader, error) {
	ra, ok := in.(io.ReaderAt)
	if !ok {
		raw, err := io.ReadAll(io.LimitReader(in, limit+1))
		if err != nil {
			return nil, errors.New("the file could not be read")
		}
		if int64(len(raw)) > limit {
			return nil, errors.New("the file is too large")
		}
		ra, size = bytes.NewReader(raw), int64(len(raw))
	}
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, errors.New("the file is not a valid zip")
	}
	return zr, nil
}

// collectFiles lists the files of the zip by name, and the ones that are the file of a table
func collectFiles(zr *zip.Reader) (map[string]*zip.File, []*zip.File) {
	zipFiles := map[string]*zip.File{}
	var tables []*zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		zipFiles[f.Name] = f
		if isTableFile(f.Name) {
			tables = append(tables, f)
		}
	}
	return zipFiles, tables
}

// readSheets reads every table file; the files of all the tables together stay within the limit, as one csv zip does
func readSheets(tables []*zip.File, limit int64) ([]csv.Sheet, error) {
	var total int64
	sheets := make([]csv.Sheet, 0, len(tables))
	for _, f := range tables {
		total += int64(f.UncompressedSize64)
		if int64(f.UncompressedSize64) > limit || total > limit {
			return nil, fmt.Errorf("%s in the zip is too large", f.Name)
		}
		sheet, err := readTable(f, limit)
		if err != nil {
			return nil, err
		}
		sheets = append(sheets, sheet)
	}
	return sheets, nil
}

// isTableFile reports whether a zip entry is the file of a table: a .json that is not inside attachments/ and not a
// hidden or system file
func isTableFile(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, AttachmentDir) || strings.HasPrefix(lower, "__macosx/") || strings.HasPrefix(path.Base(name), ".") {
		return false
	}
	return strings.HasSuffix(lower, Extension)
}

// tableTitle is the title of the table of a file: the file name without its extension
func tableTitle(name string) string {
	base := path.Base(strings.ReplaceAll(name, "\\", "/"))
	return strings.TrimSpace(base[:len(base)-len(path.Ext(base))])
}

// readTable reads one table file: an array of rows. It reads the file token by token, so that the order of the keys
// (the order of the fields) is kept, and with json.Number, so that a number keeps its digits. The header of the
// sheet holds the keys in order of first appearance, and every row is as wide as the header.
func readTable(f *zip.File, limit int64) (csv.Sheet, error) {
	rc, err := f.Open()
	if err != nil {
		return csv.Sheet{}, fmt.Errorf("%s could not be read", f.Name)
	}
	defer rc.Close()
	return readTableDoc(f.Name, rc, limit)
}

// readTableDoc reads the rows of one table from the content of a file called name
func readTableDoc(name string, rc io.Reader, limit int64) (csv.Sheet, error) {
	if strings.HasSuffix(strings.ToLower(name), sbaseExtension) {
		return csv.Sheet{}, errors.New("this is a Sereni Base file: import it with the sbase format")
	}
	doc, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil || int64(len(doc)) > limit {
		return csv.Sheet{}, fmt.Errorf("%s could not be read", name)
	}
	doc = bytes.TrimPrefix(doc, []byte("\xef\xbb\xbf"))

	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return csv.Sheet{}, invalidJSON(name, err)
	}
	cols := &columnSet{index: map[string]int{}}
	switch d, _ := tok.(json.Delim); d {
	case '[':
		return readRows(dec, name, cols)
	case '{':
		return readSingleObject(dec, name, cols)
	}
	return csv.Sheet{}, fmt.Errorf("%s must be an array of objects or one object", name)
}

// readRows reads the rest of a file that is an array of rows, after its opening [
func readRows(dec *json.Decoder, name string, cols *columnSet) (csv.Sheet, error) {
	var rows []map[int]string
	for dec.More() {
		row, err := readRow(dec, name, len(rows)+1, cols)
		if err != nil {
			return csv.Sheet{}, err
		}
		rows = append(rows, row)
	}
	if _, err := dec.Token(); err != nil { // the closing ]
		return csv.Sheet{}, invalidJSON(name, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return csv.Sheet{}, fmt.Errorf("%s has data after the array of rows", name)
	}
	return buildSheet(tableTitle(name), cols.names, rows), nil
}

// readSingleObject reads the rest of a file that is one object, after its opening {: the file is a table of one row.
// An object with no key has no row and no field, so it is refused.
func readSingleObject(dec *json.Decoder, name string, cols *columnSet) (csv.Sheet, error) {
	if !dec.More() {
		return csv.Sheet{}, fmt.Errorf("%s is empty: it has no rows", name)
	}
	row, err := readObjectBody(dec, name, cols)
	if err != nil {
		return csv.Sheet{}, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return csv.Sheet{}, fmt.Errorf("%s has data after the object", name)
	}
	return buildSheet(tableTitle(name), cols.names, []map[int]string{row}), nil
}

// columnSet holds the keys seen so far, in order of first appearance
type columnSet struct {
	names []string
	index map[string]int
}

func (c *columnSet) position(key string) int {
	i, ok := c.index[key]
	if !ok {
		i = len(c.names)
		c.index[key] = i
		c.names = append(c.names, key)
	}
	return i
}

// readRow reads one row: an object whose keys are added to the columns
func readRow(dec *json.Decoder, name string, number int, cols *columnSet) (map[int]string, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, invalidJSON(name, err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("row %d of %s must be an object", number, name)
	}
	return readObjectBody(dec, name, cols)
}

// readObjectBody reads the keys and values of an object, after its opening {, up to and including its closing }
func readObjectBody(dec *json.Decoder, name string, cols *columnSet) (map[int]string, error) {
	row := map[int]string{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, invalidJSON(name, err)
		}
		key, _ := keyTok.(string)
		var v interface{}
		if err := dec.Decode(&v); err != nil {
			return nil, invalidJSON(name, err)
		}
		row[cols.position(key)] = cellText(v)
	}
	if _, err := dec.Token(); err != nil { // the closing }
		return nil, invalidJSON(name, err)
	}
	return row, nil
}

// buildSheet makes the sheet of a table: the header, then every row as wide as the header. A table with no field at
// all is made with the default field of an empty file.
func buildSheet(name string, columns []string, rows []map[int]string) csv.Sheet {
	if len(columns) == 0 {
		return csv.Sheet{Name: name}
	}
	records := make([][]string, 0, len(rows)+1)
	lines := make([]int, 0, len(rows)+1)
	records = append(records, columns)
	lines = append(lines, 1)
	for i, row := range rows {
		rec := make([]string, len(columns))
		for c, text := range row {
			rec[c] = text
		}
		records = append(records, rec)
		lines = append(lines, i+2)
	}
	return csv.Sheet{Name: name, Records: records, Lines: lines}
}

// cellText is the text of a value read from the file: a string as it is, a number with its own digits, a list joined
// with ",", an object as compact JSON, null as nothing
func cellText(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "true"
		}
		return "false"
	case []interface{}:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			if s := cellText(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, listJoiner(parts))
	default:
		return string(marshal(x))
	}
}

// listJoiner is what joins the items of a list in one cell: a comma, except for a list of the paths of attachment files,
// which the import of the files reads as items joined with a pipe
func listJoiner(items []string) string {
	if len(items) == 0 {
		return ","
	}
	for _, item := range items {
		if !strings.HasPrefix(strings.ToLower(item), AttachmentDir) {
			return ","
		}
	}
	return joiner
}

// invalidJSON is what the user sees for a file that is not valid JSON
func invalidJSON(name string, err error) error {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return fmt.Errorf("%s is not valid JSON (error at byte %d)", name, syntax.Offset)
	}
	return fmt.Errorf("%s is not valid JSON", name)
}

// baseTitleFromFilename names the new base after the uploaded file: "My Base.json.zip", "My Base.zip" and "My Base.json" give "My Base"
func baseTitleFromFilename(filename string) string {
	name := path.Base(strings.ReplaceAll(filename, "\\", "/"))
	lower := strings.ToLower(name)
	for _, suffix := range []string{ZipExtension, ".zip", Extension} {
		if strings.HasSuffix(lower, suffix) {
			name = name[:len(name)-len(suffix)]
			break
		}
	}
	if name = strings.TrimSpace(name); name == "" || name == "." {
		return "Imported JSON"
	}
	return name
}
