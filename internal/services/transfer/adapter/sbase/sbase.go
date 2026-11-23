// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package sbase is the adapter of the SereniBase file format: base.sbase.json, or a .sbase.zip that holds it with
// the attachment files and the cover image.
package sbase

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/internal/ziputil"
)

const (
	// Name is the name of this format
	Name = "sbase"
	// BaseFileInZip is the file of the zip that holds the model
	BaseFileInZip = "base.sbase.json"
	// AttachmentDir is the folder of the zip that holds the files
	AttachmentDir = "attachments/"
)

var unsafeFileNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type sbaseAdapter struct{}

// New returns the SereniBase adapter
func New() adapter.Adapter { return sbaseAdapter{} }

func (sbaseAdapter) Name() string { return Name }

func (sbaseAdapter) Capabilities() adapter.Capabilities {
	return adapter.Capabilities{Structure: true, Rows: true, Links: true, Lookups: true, Views: true, Constraints: true, Files: true}
}

// Describe: an export is always a .sbase.zip (with or without rows and files); import still reads a plain .sbase.json
func (sbaseAdapter) Describe(opt adapter.ExportOptions, title string) (string, string) {
	name := strings.Trim(unsafeFileNameChars.ReplaceAllString(title, "_"), "_")
	if name == "" {
		name = "base"
	}
	return name + ".sbase.zip", "application/zip"
}

// Export writes the model as a zip: base.sbase.json, plus the files when the model lists any. File sizes and
// checksums are taken from what is actually read, so the index in the JSON always matches the zip.
func (sbaseAdapter) Export(ctx context.Context, file *dto.BaseExportFile, opt adapter.ExportOptions, out io.Writer) error {
	zw := zip.NewWriter(out)
	hasFiles := file.Base.Cover != nil || (file.Data != nil && len(file.Data.Attachments) > 0)
	if hasFiles && opt.OpenFile == nil {
		return errors.New("files cannot be exported: no storage provider is configured")
	}
	if file.Data != nil {
		for i := range file.Data.Attachments {
			a := &file.Data.Attachments[i]
			size, sum, err := ziputil.PackFile(ctx, zw, opt.OpenFile, a.Filename, a.Path, a.StoragePath)
			if err != nil {
				return err
			}
			a.Size, a.SHA256 = size, sum
		}
	}
	if c := file.Base.Cover; c != nil {
		size, sum, err := ziputil.PackFile(ctx, zw, opt.OpenFile, c.Filename, c.Path, c.StoragePath)
		if err != nil {
			return err
		}
		c.Size, c.SHA256 = size, sum
	}
	jw, err := zw.Create(BaseFileInZip)
	if err != nil {
		return err
	}
	if err := encodeJSON(jw, file); err != nil {
		return err
	}
	return zw.Close()
}

// encodeJSON writes indented JSON without escaping <, > and &, so SQL in the file stays readable
func encodeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Import reads a .sbase.json, or a .zip that holds base.sbase.json (the extension of opt.Filename tells which).
// For a zip the input must be an io.ReaderAt (an uploaded file is) and must stay open until the import is done.
func (sbaseAdapter) Import(_ context.Context, in io.Reader, size int64, opt adapter.ImportOptions) (*adapter.ImportResult, error) {
	limit := opt.MaxBytes
	if limit <= 0 {
		limit = adapter.DefaultMaxImportBytes
	}

	reader := in
	var source *dto.Attachments
	if strings.HasSuffix(strings.ToLower(opt.Filename), ".zip") {
		ra, ok := in.(io.ReaderAt)
		if !ok {
			return nil, errors.New("the file could not be read")
		}
		zr, err := zip.NewReader(ra, size)
		if err != nil {
			return nil, errors.New("the file is not a valid zip")
		}
		entry := findZipEntry(zr)
		if entry == nil {
			return nil, fmt.Errorf("the zip has no %s", BaseFileInZip)
		}
		if int64(entry.UncompressedSize64) > limit {
			return nil, fmt.Errorf("%s in the zip is too large", BaseFileInZip)
		}
		rc, err := entry.Open()
		if err != nil {
			return nil, errors.New("the zip could not be read")
		}
		defer rc.Close()
		reader = rc
		source = NewZipSource(zr)
	}

	var file dto.BaseExportFile
	dec := json.NewDecoder(io.LimitReader(reader, limit+1))
	// The schema does not allow properties it does not define, so a typo such as "primary_keys" is an error here
	// instead of a setting that is silently dropped.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("the file is not valid JSON: %v", err)
	}
	return &adapter.ImportResult{File: &file, Attachments: source}, nil
}

func findZipEntry(zr *zip.Reader) *zip.File {
	for _, e := range zr.File {
		if e.Name == BaseFileInZip {
			return e
		}
	}
	return nil
}

// NewZipSource wraps an opened zip as the attachments of an import.
func NewZipSource(zr *zip.Reader) *dto.Attachments { return ziputil.NewZipSource(zr) }
