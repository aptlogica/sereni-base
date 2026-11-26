// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"strings"
)

// csvSniffBytes is how much of a file is read to tell a text file from a renamed binary one
const csvSniffBytes = 8192

// errCSVFilesTooLarge is the problem bundleCSVFiles reports when the files together are over the limit
const errCSVFilesTooLarge = "the files are too large"

// importedCSVName is the name the new base gets when several CSV files are imported
const importedCSVName = "Imported CSV"

// csvFileProblem says why an uploaded file is not a usable CSV file, or "" when it is. The same rules are applied by the
// import dialog before it uploads anything; this is the check that is trusted.
func csvFileProblem(name string, size int64, head []byte) string {
	base := path.Base(strings.ReplaceAll(name, "\\", "/"))
	lower := strings.ToLower(base)
	if !strings.HasSuffix(lower, ".csv") {
		return fmt.Sprintf("%s is not a CSV file. Only .csv files or a .zip are supported.", base)
	}
	if size == 0 {
		return fmt.Sprintf("%s is empty.", base)
	}
	// a name starting with a dot is skipped when the files are read, so it would be silently left out
	if strings.HasPrefix(base, ".") {
		return fmt.Sprintf("%s is not a valid CSV file name.", base)
	}
	if bytes.HasPrefix(head, []byte("PK")) || bytes.IndexByte(head, 0) >= 0 {
		return fmt.Sprintf("%s is not a valid CSV file.", base)
	}
	return ""
}

// csvUploadKind sorts the files of a csv import: zips are a zip import, .csv files are loose CSV files. It returns the
// problem when the choice is not allowed: a mix, or more than one zip.
func csvUploadKind(files []*multipart.FileHeader) (loose bool, problem string) {
	zips, others := 0, 0
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f.Filename), ".zip") {
			zips++
		} else {
			others++
		}
	}
	switch {
	case zips > 1:
		return false, "Choose only one .zip."
	case zips == 1 && others > 0:
		return false, "Choose either CSV files or one .zip, not both."
	}
	return zips == 0, ""
}

// bundleCSVFiles checks every loose CSV file and collects them in one temporary file in the container the csv adapter
// reads, so a set of CSV files goes through exactly the flow of a zip of CSV files. It returns the open file (the caller
// closes and removes it), its size and the name to import it under: the name of the file for one CSV file, "Imported
// CSV" for several. A problem is returned for the first file that is not a CSV, and nothing is imported then.
func bundleCSVFiles(files []*multipart.FileHeader, limit int64) (tmp *os.File, size int64, name, problem string, err error) {
	var total int64
	seen := map[string]bool{}
	for _, f := range files {
		total += f.Size
		base := path.Base(strings.ReplaceAll(f.Filename, "\\", "/"))
		key := strings.ToLower(base)
		if seen[key] {
			return nil, 0, "", fmt.Sprintf("%s was added twice.", base), nil
		}
		seen[key] = true
	}
	if total > limit {
		return nil, 0, "", errCSVFilesTooLarge, nil
	}

	tmp, err = os.CreateTemp("", "sbase-import-csv-*.zip")
	if err != nil {
		return nil, 0, "", "", err
	}
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}
	zw := zip.NewWriter(tmp)
	for _, f := range files {
		src, openErr := f.Open()
		if openErr != nil {
			cleanup()
			return nil, 0, "", "", openErr
		}
		head := make([]byte, csvSniffBytes)
		n, _ := io.ReadFull(src, head)
		head = head[:n]
		if p := csvFileProblem(f.Filename, f.Size, head); p != "" {
			_ = src.Close()
			cleanup()
			return nil, 0, "", p, nil
		}
		w, createErr := zw.Create(path.Base(strings.ReplaceAll(f.Filename, "\\", "/")))
		if createErr == nil {
			if _, createErr = w.Write(head); createErr == nil {
				_, createErr = io.Copy(w, src)
			}
		}
		_ = src.Close()
		if createErr != nil {
			cleanup()
			return nil, 0, "", "", createErr
		}
	}
	if err = zw.Close(); err != nil {
		cleanup()
		return nil, 0, "", "", err
	}
	info, err := tmp.Stat()
	if err != nil {
		cleanup()
		return nil, 0, "", "", err
	}
	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, 0, "", "", err
	}

	name = importedCSVName + ".zip"
	if len(files) == 1 {
		one := path.Base(strings.ReplaceAll(files[0].Filename, "\\", "/"))
		name = one[:len(one)-len(".csv")] + ".zip"
	}
	return tmp, info.Size(), name, "", nil
}
