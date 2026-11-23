// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package csv

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
)

// Sheet is one table of text read from a source that is not a CSV file but looks like one (a sheet of a workbook): a
// name, and the records of the sheet as text. The first record is the header.
type Sheet struct {
	// Name becomes the title of the table
	Name string
	// Records are the rows of the sheet as text, header first
	Records [][]string
	// Lines are the line of each record in the sheet (the header is line 1); it must have one entry per record
	Lines []int
}

// BuildModel turns sheets of text into a base model with one table per sheet, in the way an import of CSV files
// does: the types are inferred from the values, empty rows are dropped, and the source of every row is recorded for
// the error log. A column whose cells are all paths of files in attachments/ of zipFiles (by name; may be nil) becomes an
// attachment field, and the files it names are listed in the model's Data.Attachments.
func BuildModel(title string, sheets []Sheet, zipFiles map[string]*zip.File) (*dto.BaseExportFile, *dto.ImportSource) {
	model := &dto.BaseExportFile{
		Base:   dto.BaseExportBase{Title: title},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{}},
		Data:   &dto.BaseExportData{RowCounts: map[string]int{}, Tables: map[string]dto.BaseExportTableRow{}},
	}
	refs := newAttachmentIndex(model.Data, zipFiles)
	usedKeys := map[string]bool{}
	source := &dto.ImportSource{Rows: map[string]dto.ImportRowSource{}}
	for i, sh := range sheets {
		tableTitle := uniqueSheetTitle(i, sh.Name, usedKeys)
		table, rows := buildTitledTable(i, sh.Name, tableTitle, sh.Records, sh.Lines, usedKeys, refs, source)
		model.Schema.Tables = append(model.Schema.Tables, table)
		model.Data.Tables[table.Key] = dto.BaseExportTableRow{Rows: rows}
		model.Data.RowCounts[table.Key] = len(rows)
	}
	model.Data.Attachments = refs.files
	return model, source
}

// uniqueSheetTitle is the title of the table of a sheet: the name of the sheet, made unique when two sheets only
// differ in case
func uniqueSheetTitle(index int, name string, usedKeys map[string]bool) string {
	title := strings.TrimSpace(name)
	if title == "" {
		title = fmt.Sprintf("Table %d", index+1)
	}
	if runes := []rune(title); len(runes) > 240 {
		title = string(runes[:240])
	}
	for n, base := 2, title; usedKeys["title:"+strings.ToLower(title)]; n++ {
		title = fmt.Sprintf("%s (%d)", base, n)
	}
	usedKeys["title:"+strings.ToLower(title)] = true
	return title
}
