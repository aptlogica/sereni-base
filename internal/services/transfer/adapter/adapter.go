// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package adapter turns the standard model of a base (dto.BaseExportFile) into a file and back. An adapter never
// touches the database or a service: the core service builds and creates bases, the adapter only reads and writes
// the file format.
//
//	export:  core.ExportBase (database -> model)  ->  adapter.Export (model -> file)
//	import:  adapter.Import (file -> model)       ->  core.ImportBase (validate, create)
package adapter

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/aptlogica/sereni-base/internal/dto"
)

const (
	// MaxAttachmentBytes is the largest file an export packs and an import accepts
	MaxAttachmentBytes = 100 << 20
	// DefaultMaxImportBytes caps the file an import reads when ImportOptions does not say
	DefaultMaxImportBytes = 100 << 20
)

// Capabilities says what a format can hold; what it cannot hold is lost (and reported) when a base goes into it.
type Capabilities struct {
	Structure, Rows, Links, Lookups, Views, Constraints, Files bool
}

// FileSource gives an export the bytes of the files (attachments, cover image) the model lists. The core service
// implements it, so the adapter needs no storage code.
type FileSource interface {
	// OpenFile opens a stored file by the StoragePath of the model entry that lists it
	OpenFile(ctx context.Context, storagePath string) (io.ReadCloser, error)
}

// ExportOptions are the options of one export
type ExportOptions struct {
	// IncludeData is true when the model has rows. For a format that can carry files, the result then also holds
	// the attachment files and the cover image.
	IncludeData bool
	// Files reads the files the model lists; it may be nil when the model lists none
	Files FileSource
}

// ImportOptions are the options of one import
type ImportOptions struct {
	// Filename is the name of the uploaded file; the extension tells a zip from plain JSON
	Filename string
	// MaxBytes caps what is read from the file (0: DefaultMaxImportBytes)
	MaxBytes int64
}

// ImportResult is a file read into the model
type ImportResult struct {
	File *dto.BaseExportFile
	// Attachments gives the files a zip carried, by path; nil when the file carried none. It reads from the input
	// given to Import, so the input must stay open until the import is done.
	Attachments dto.AttachmentSource
}

// Adapter reads and writes one file format
type Adapter interface {
	// Name is the format name, e.g. "sbase"
	Name() string
	Capabilities() Capabilities
	// Describe returns the name and content type a download of this export gets, for a base called title
	Describe(opt ExportOptions, title string) (filename, contentType string)
	// Export writes the model as a file
	Export(ctx context.Context, m *dto.BaseExportFile, opt ExportOptions, out io.Writer) error
	// Import reads a file into the model. The error message says what is wrong with the file, in words for the user.
	Import(ctx context.Context, in io.Reader, size int64, opt ImportOptions) (*ImportResult, error)
}

// Registry finds an adapter by format name
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry returns an empty registry
func NewRegistry() *Registry {
	return &Registry{adapters: map[string]Adapter{}}
}

// Register adds an adapter; registering a name twice is a programming error
func (r *Registry) Register(a Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.adapters[a.Name()]; dup {
		panic(fmt.Sprintf("adapter %q is registered twice", a.Name()))
	}
	r.adapters[a.Name()] = a
}

// Get returns the adapter of a format
func (r *Registry) Get(name string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[name]
	return a, ok
}

// Names lists the registered formats, sorted
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for n := range r.adapters {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
