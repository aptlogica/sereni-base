// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package xlsx

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/csv"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/internal/ziputil"
)

// unzipLimit bounds what a workbook may unpack to, against a zip bomb
const unzipLimit = 1 << 30

// Import reads an export of this format into a model: a workbook (.xlsx), or a zip (.zip) that holds one workbook and,
// optionally, the files of an attachments/ folder. One table is made per visible sheet that has data, named after the
// sheet. The first row of a sheet holds the field titles. The cells are read as the text Excel shows (a number as its
// digits, a boolean as TRUE or FALSE, a date as the format of the cell shows it), and the types are guessed from the
// values in the way an import of CSV files does; a column whose cells are all paths of files in attachments/ becomes
// an attachment field. A formula is read as the value it last calculated; it is never calculated or kept. Hidden
// sheets and sheets with no cell are left out. The new base is named after the file. For a zip, the input must stay
// open until the import is done, because the attachment files are read from it.
func (xlsxAdapter) Import(_ context.Context, in io.Reader, size int64, opt adapter.ImportOptions) (*adapter.ImportResult, error) {
	lower := strings.ToLower(opt.Filename)
	isZip := strings.HasSuffix(lower, ".zip")
	if !isZip && !strings.HasSuffix(lower, Extension) {
		return nil, errors.New("only .xlsx or .zip files are supported")
	}
	limit := opt.MaxBytes
	if limit <= 0 {
		limit = adapter.DefaultMaxImportBytes
	}

	var workbook []byte
	var zipFiles map[string]*zip.File
	var zr *zip.Reader
	if isZip {
		var err error
		if zr, workbook, zipFiles, err = openZip(in, size, limit); err != nil {
			return nil, err
		}
	} else {
		raw, err := io.ReadAll(io.LimitReader(in, limit+1))
		if err != nil {
			return nil, errors.New("the file could not be read")
		}
		if int64(len(raw)) > limit {
			return nil, errors.New("the file is too large")
		}
		workbook = raw
	}

	sheets, err := readSheets(workbook)
	if err != nil {
		return nil, err
	}
	model, source := csv.BuildModel(baseTitleFromFilename(opt.Filename), sheets, zipFiles)
	result := &adapter.ImportResult{File: model, Source: source}
	if zr != nil && len(model.Data.Attachments) > 0 {
		result.Attachments = ziputil.NewZipSource(zr)
	}
	return result, nil
}

// openZip finds the one workbook of a zip and the files of attachments/. A zip with no workbook, or with more than one,
// is refused.
func openZip(in io.Reader, size, limit int64) (*zip.Reader, []byte, map[string]*zip.File, error) {
	ra, ok := in.(io.ReaderAt)
	if !ok {
		raw, err := io.ReadAll(io.LimitReader(in, limit+1))
		if err != nil {
			return nil, nil, nil, errors.New("the file could not be read")
		}
		if int64(len(raw)) > limit {
			return nil, nil, nil, errors.New("the file is too large")
		}
		ra, size = bytes.NewReader(raw), int64(len(raw))
	}
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, nil, nil, errors.New("the file is not a valid zip")
	}
	zipFiles := map[string]*zip.File{}
	var books []*zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		zipFiles[f.Name] = f
		if isWorkbook(f.Name) {
			books = append(books, f)
		}
	}
	switch {
	case len(books) == 0:
		return nil, nil, nil, errors.New("the zip has no .xlsx file")
	case len(books) > 1:
		return nil, nil, nil, errors.New("the zip has more than one .xlsx file")
	}
	book := books[0]
	if int64(book.UncompressedSize64) > limit {
		return nil, nil, nil, fmt.Errorf("%s in the zip is too large", book.Name)
	}
	rc, err := book.Open()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s could not be read", book.Name)
	}
	defer rc.Close()
	workbook, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil || int64(len(workbook)) > limit {
		return nil, nil, nil, fmt.Errorf("%s could not be read", book.Name)
	}
	return zr, workbook, zipFiles, nil
}

// isWorkbook reports whether a zip entry is a workbook: an .xlsx that is not inside attachments/
func isWorkbook(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, AttachmentDir) || strings.HasPrefix(lower, "__macosx/") || strings.HasPrefix(path.Base(name), ".") {
		return false
	}
	return strings.HasSuffix(lower, Extension)
}

// readSheets reads the visible sheets that have data
func readSheets(workbook []byte) ([]csv.Sheet, error) {
	f, err := excelize.OpenReader(bytes.NewReader(workbook), excelize.Options{UnzipSizeLimit: unzipLimit})
	if err != nil {
		if errors.Is(err, excelize.ErrWorkbookPassword) || errors.Is(err, excelize.ErrWorkbookFileFormat) {
			return nil, errors.New("a password-protected workbook cannot be imported")
		}
		return nil, errNotWorkbook
	}
	defer f.Close()

	var sheets []csv.Sheet
	for _, name := range f.GetSheetList() {
		if visible, err := f.GetSheetVisible(name); err != nil || !visible {
			continue
		}
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, fmt.Errorf("sheet %s could not be read: %v", name, err)
		}
		if isEmpty(rows) {
			continue
		}
		lines := make([]int, len(rows))
		for i := range rows {
			lines[i] = i + 1
		}
		sheets = append(sheets, csv.Sheet{Name: name, Records: rows, Lines: lines})
	}
	if len(sheets) == 0 {
		return nil, errors.New("the workbook has no sheet with data")
	}
	return sheets, nil
}

// isEmpty reports whether a sheet has no cell with a value
func isEmpty(rows [][]string) bool {
	for _, row := range rows {
		for _, c := range row {
			if strings.TrimSpace(c) != "" {
				return false
			}
		}
	}
	return true
}

// baseTitleFromFilename names the new base after the uploaded file: "My Base.xlsx.zip" and "My Base.xlsx" give "My Base"
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
		return "Imported Excel"
	}
	return name
}
