// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"regexp"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/sbase"
)

const (
	// MaxAttachmentBytes is the largest file export packs and import accepts
	MaxAttachmentBytes = adapter.MaxAttachmentBytes
	// MaxAttachmentsTotalBytes is the most all files of one export may add up to
	MaxAttachmentsTotalBytes = 500 << 20

	zipAttachmentDir = sbase.AttachmentDir
	// uploadMemory is how much of an uploaded file is kept in memory while it is handed to the asset service
	uploadMemory = 32 << 20
)

var (
	attachmentRefPattern  = regexp.MustCompile(`^a[A-Za-z0-9_-]{0,62}$`)
	attachmentPathPattern = regexp.MustCompile(`^attachments/[A-Za-z0-9_.-]{1,200}$`)
	unsafeNameChars       = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
)

// attachmentCollector gives each attachment of an export a ref and applies the size limits.
type attachmentCollector struct {
	byAsset map[string]string // asset id -> ref
	files   []dto.BaseExportAttachment
	total   int64
}

func newAttachmentCollector() *attachmentCollector {
	return &attachmentCollector{byAsset: map[string]string{}}
}

// add registers the asset of an attachment cell item and returns its ref; ok is false (with a reason) when the
// file cannot travel.
func (c *attachmentCollector) add(item map[string]interface{}) (ref string, reason string) {
	assetID := idString(item["id"])
	storagePath := idString(item["base_path"])
	if assetID == "" || storagePath == "" {
		return "", "the cell has no stored file"
	}
	if ref, ok := c.byAsset[assetID]; ok {
		return ref, ""
	}
	size := int64(numericID(item["size"]))
	name := idString(item["title"])
	if name == "" {
		name = assetID
	}
	if size > MaxAttachmentBytes {
		return "", fmt.Sprintf("%s is larger than %d MB", name, MaxAttachmentBytes>>20)
	}
	if c.total+size > MaxAttachmentsTotalBytes {
		return "", fmt.Sprintf("%s would take the attachments over %d MB", name, MaxAttachmentsTotalBytes>>20)
	}
	ref = fmt.Sprintf("a%d", len(c.files)+1)
	c.byAsset[assetID] = ref
	c.total += size
	c.files = append(c.files, dto.BaseExportAttachment{
		Ref:         ref,
		Filename:    name,
		ContentType: idString(item["mime_type"]),
		Size:        size,
		Path:        zipAttachmentDir + ref + "_" + safeFileName(name),
		StoragePath: storagePath,
	})
	return ref, ""
}

func safeFileName(name string) string {
	clean := strings.Trim(unsafeNameChars.ReplaceAllString(name, "_"), "_.")
	if clean == "" {
		clean = "file"
	}
	if len(clean) > 100 {
		clean = clean[len(clean)-100:]
	}
	return clean
}

// attachmentItems reads an attachment cell: an array of asset objects.
func attachmentItems(cell interface{}) []map[string]interface{} {
	var out []map[string]interface{}
	for _, item := range toList(cell) {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

// OpenFile opens a stored file (an attachment or the cover image) for an export adapter, which packs it into the
// file it writes. It implements adapter.FileSource.
func (s *baseTransferService) OpenFile(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	if s.storage == nil {
		return nil, errors.New("files cannot be exported: no storage provider is configured")
	}
	return s.storage.Download(ctx, storagePath)
}

// NewZipSource wraps an opened zip as the attachment source of an import.
func NewZipSource(zr *zip.Reader) dto.AttachmentSource {
	return sbase.NewZipSource(zr)
}

// readFile reads one file of the zip, checking its size and checksum against the index.
func readFile(src dto.AttachmentSource, name, path, checksum string) ([]byte, error) {
	rc, err := src.Open(path)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, MaxAttachmentBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxAttachmentBytes {
		return nil, fmt.Errorf("%q is larger than %d MB", name, MaxAttachmentBytes>>20)
	}
	if checksum != "" {
		sum := sha256.Sum256(data)
		if "sha256:"+hex.EncodeToString(sum[:]) != checksum {
			return nil, fmt.Errorf("%q does not match its checksum; the file is damaged", name)
		}
	}
	return data, nil
}

// fileHeaderFor wraps bytes as an uploaded file, so they go through the asset service like any upload
// (antivirus scan, storage, thumbnail).
func fileHeaderFor(name, contentType string, data []byte) (*multipart.FileHeader, func(), error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename=%q`, name))
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	part, err := mw.CreatePart(h)
	if err != nil {
		return nil, nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, nil, err
	}
	form, err := multipart.NewReader(&body, mw.Boundary()).ReadForm(uploadMemory)
	if err != nil {
		return nil, nil, err
	}
	headers := form.File["files"]
	if len(headers) != 1 {
		_ = form.RemoveAll()
		return nil, nil, errors.New("could not prepare the attachment for upload")
	}
	return headers[0], func() { _ = form.RemoveAll() }, nil
}

// uploadAttachment stores one file as a new asset and returns it.
func (r *importRun) uploadAttachment(a dto.BaseExportAttachment) (tenant.Assets, error) {
	if r.s.assets == nil {
		return tenant.Assets{}, errors.New("attachments cannot be imported: the asset service is not configured")
	}
	data, err := readFile(r.attachments, a.Filename, a.Path, a.SHA256)
	if err != nil {
		return tenant.Assets{}, err
	}
	header, cleanup, err := fileHeaderFor(a.Filename, a.ContentType, data)
	if err != nil {
		return tenant.Assets{}, err
	}
	defer cleanup()
	uploaded, err := r.s.assets.Upload(r.ctx, dto.UploadAssetRequest{Files: []*multipart.FileHeader{header}}, r.schema)
	if err != nil {
		return tenant.Assets{}, fmt.Errorf("upload attachment %q: %w", a.Filename, err)
	}
	if len(uploaded) != 1 {
		return tenant.Assets{}, fmt.Errorf("upload attachment %q: expected one asset back, got %d", a.Filename, len(uploaded))
	}
	r.assetIDs = append(r.assetIDs, uploaded[0].ID.String())
	return uploaded[0], nil
}
