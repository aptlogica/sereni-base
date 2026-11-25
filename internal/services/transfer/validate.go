// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/aptlogica/sereni-base/internal/dto"
)

const (
	maxProblems    = 50
	uidtAttachment = "attachment"
	// maxTitleLength and maxRIDLength are the limits of the schema
	maxTitleLength = 255
	maxRIDLength   = 128
	// maxDBNameLength is the longest name Postgres allows for a table or a column. The name of a table or a column in the
	// database is made from its title (spaces become _, other symbols are dropped, a name that does not start with a
	// letter gets "table_" in front) and then dbNameSuffixLength characters are added to make it unique: "_", the Unix
	// time (10 digits), "_" and 4 random characters.
	maxDBNameLength    = 63
	dbNameSuffixLength = 16
)

const (
	// the keys of the meta of a link or lookup field, and the message several checks share
	metaRelationID   = "relation_id"
	metaRelation     = "relation"
	msgTitleRequired = "%s: title is required"
)

var dbNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// dbNameLength is how long the name in the database of a table or a field with this title will be: the same rule the
// table service uses when it creates one
func dbNameLength(title string) int {
	slug := dbNameUnsafe.ReplaceAllString(strings.ReplaceAll(title, " ", "_"), "")
	if slug == "" || (slug[0] >= '0' && slug[0] <= '9') {
		slug = "table_" + slug
	}
	return len(slug) + dbNameSuffixLength
}

// dbNameProblem is the problem for a title whose name in the database would be over the limit of 63 characters, or ""
// when it fits. The database refuses such a table or column, and the import would fail with only "database error".
func dbNameProblem(where, what, title string) string {
	n := dbNameLength(title)
	if n <= maxDBNameLength {
		return ""
	}
	return fmt.Sprintf("%s: the %s %q exceeds the database limit of %d characters: its database name would be %d characters (the letters, digits and spaces of the name, plus %d that the app adds). Shorten the name by at least %d characters",
		where, what, shortTitle(title), maxDBNameLength, n, dbNameSuffixLength, n-maxDBNameLength)
}

// importPlan is a checked file with the lookups import needs.
type importPlan struct {
	file *dto.BaseExportFile
	// field key -> table key
	fieldTable map[string]string
	// field key -> field
	fields map[string]dto.BaseExportField
	// relation key -> its source field key and (if any) target field key
	relSource map[string]string
	relTarget map[string]string
	// table key -> order
	tableIndex map[string]int
	// every view of the file, those inside tables first, then those in the top-level list of format 1.0.0
	views []planView
	// function key -> table key, for the trigger functions of the file
	functionKeys map[string]string
}

// planView is a view with the table it belongs to.
type planView struct {
	view  dto.BaseExportView
	table string
	where string
}

// ValidateBaseFile checks a file as a whole, before anything is created, and returns every problem found.
func ValidateBaseFile(file *dto.BaseExportFile) []string {
	_, problems := buildPlan(file)
	return problems
}

// checkCategories are the labels the app accepts for a check constraint; no label means custom
var checkCategories = map[string]bool{"number_date": true, "text": true, "compare": true, "custom": true}

var sha256Pattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

// longer reports whether s has more than max characters.
func longer(s string, max int) bool { return utf8.RuneCountInString(s) > max }

// tooLongProblem is the problem for a name over the limit: where it is, what it is (a table name, a field name ...),
// the start of it, how long it is and what the limit is, so the user can find it and knows how much to cut
func tooLongProblem(where, what, title string) string {
	return fmt.Sprintf("%s: the %s %q is too long: it has %d characters and the limit is %d", where, what, shortTitle(title), utf8.RuneCountInString(title), maxTitleLength)
}

// shortTitle is the start of a long name, for a message
func shortTitle(title string) string {
	const shown = 40
	if runes := []rune(title); len(runes) > shown {
		return string(runes[:shown]) + "..."
	}
	return title
}

type problemList []string

func (p *problemList) add(format string, args ...interface{}) {
	if len(*p) < maxProblems {
		*p = append(*p, fmt.Sprintf(format, args...))
	}
}

func buildPlan(file *dto.BaseExportFile) (*importPlan, []string) {
	var problems problemList
	if file == nil {
		return nil, []string{"file is empty"}
	}
	if strings.TrimSpace(file.Base.Title) == "" && len(file.Schema.Tables) == 0 {
		// Nothing identifies the file, so a stranger's JSON ends up here
		return nil, []string{"this is not a base export file: it has no base title and no tables"}
	}
	if strings.TrimSpace(file.Base.Title) == "" {
		problems.add("base.title is required")
	} else if longer(file.Base.Title, maxTitleLength) {
		problems.add("%s", tooLongProblem("base.title", "base name", file.Base.Title))
	}

	plan := &importPlan{
		file:       file,
		fieldTable: map[string]string{},
		fields:     map[string]dto.BaseExportField{},
		relSource:  map[string]string{},
		relTarget:  map[string]string{},
		tableIndex: map[string]int{},
	}
	checkKey := (&keyChecker{keys: map[string]string{}, problems: &problems}).check

	if len(file.Schema.Tables) == 0 {
		problems.add("schema.tables has no tables")
	}
	for i, t := range file.Schema.Tables {
		validateTable(plan, i, t, checkKey, &problems)
	}

	collectViews(plan, &problems)
	validateCover(file.Base.Cover, &problems)
	validateTableExtras(plan, &problems)
	validateRefs(plan, &problems)
	validateRelations(plan, &problems)
	validateViews(plan, checkKey, &problems)
	validateAutomations(plan, checkKey, &problems)
	validateData(plan, &problems)
	return plan, problems
}

// keyChecker checks the keys of the file: each must be well formed and used once. keys is every key in the file ->
// what it names.
type keyChecker struct {
	keys     map[string]string
	problems *problemList
}

func (k *keyChecker) check(key, what, where string) bool {
	if !keyPattern.MatchString(key) {
		k.problems.add("%s: %s %q is not a valid key", where, what, key)
		return false
	}
	if prev, dup := k.keys[key]; dup {
		k.problems.add("%s: key %q is already used by a %s", where, key, prev)
		return false
	}
	k.keys[key] = what
	return true
}

// checkName checks the title of a table or a field: it is there (when required), it is not over the limit of the
// schema, and the name the database would get for it fits (when checkDB)
func checkName(where, what, title string, required, checkDB bool, problems *problemList) {
	switch {
	case required && strings.TrimSpace(title) == "":
		problems.add(msgTitleRequired, where)
	case longer(title, maxTitleLength):
		problems.add("%s", tooLongProblem(where, what, title))
	case checkDB:
		if msg := dbNameProblem(where, what, title); msg != "" {
			problems.add("%s", msg)
		}
	}
}

// validateTable checks one table of the schema and its fields, and adds them to the plan
func validateTable(plan *importPlan, i int, t dto.BaseExportTable, checkKey func(key, what, where string) bool, problems *problemList) {
	where := fmt.Sprintf("schema.tables[%d]", i)
	if checkKey(t.Key, "table", where) {
		plan.tableIndex[t.Key] = i
	}
	checkName(where, "table name", t.Title, true, true, problems)
	for j, f := range t.Fields {
		validateField(plan, t.Key, fmt.Sprintf("%s.fields[%d]", where, j), f, checkKey, problems)
	}
}

// validateField checks one field of a table and adds it to the plan
func validateField(plan *importPlan, tableKey, where string, f dto.BaseExportField, checkKey func(key, what, where string) bool, problems *problemList) {
	if !checkKey(f.Key, "field", where) {
		return
	}
	plan.fieldTable[f.Key] = tableKey
	plan.fields[f.Key] = f
	checkName(where, "field name", f.Title, !f.System, !f.System, problems)
	if f.System {
		if f.ColumnName == "" {
			problems.add("%s: system field needs column_name", where)
		}
		return
	}
	if !isSupportedUIDT(f.UIDT) {
		problems.add("%s: field type %q is not supported", where, f.UIDT)
	}
}

// knownRef reports whether "@key" names a table, field or relation of the file.
func (p *importPlan) knownRef(ref string) bool {
	key := strings.TrimPrefix(ref, dto.BaseExportRefPrefix)
	if _, ok := p.tableIndex[key]; ok {
		return true
	}
	if _, ok := p.fields[key]; ok {
		return true
	}
	_, ok := p.relSource[key]
	return ok
}

// collectRelations finds the relations from the link fields: each relation has a source field and at most one target.
func collectRelations(plan *importPlan, problems *problemList) {
	for _, t := range plan.file.Schema.Tables {
		for _, f := range t.Fields {
			if f.UIDT != uidtLinks || f.System {
				continue
			}
			collectRelation(plan, f, problems)
		}
	}
}

// collectRelation records the relation of one link field as the source or the target of its relation
func collectRelation(plan *importPlan, f dto.BaseExportField, problems *problemList) {
	rel := strings.TrimPrefix(idString(f.Meta[metaRelationID]), dto.BaseExportRefPrefix)
	role := idString(f.Meta["entity_role"])
	where := fmt.Sprintf("field %s", f.Key)
	if rel == "" || !refPattern.MatchString(dto.BaseExportRefPrefix+rel) {
		problems.add("%s: meta.relation_id must be a @key", where)
		return
	}
	switch role {
	case entityRoleSource:
		if prev, dup := plan.relSource[rel]; dup {
			problems.add("%s: relation @%s already has source field %s", where, rel, prev)
		}
		plan.relSource[rel] = f.Key
	case entityRoleTarget:
		if prev, dup := plan.relTarget[rel]; dup {
			problems.add("%s: relation @%s already has target field %s", where, rel, prev)
		}
		plan.relTarget[rel] = f.Key
	default:
		problems.add("%s: meta.entity_role must be source or target", where)
	}
}

func validateRelations(plan *importPlan, problems *problemList) {
	for rel, target := range plan.relTarget {
		if _, ok := plan.relSource[rel]; !ok {
			problems.add("field %s: relation @%s has no source field", target, rel)
		}
	}
	for rel, source := range plan.relSource {
		f := plan.fields[source]
		with := strings.TrimPrefix(nestedString(f.Meta, metaRelation, "with"), dto.BaseExportRefPrefix)
		if _, ok := plan.tableIndex[with]; !ok {
			problems.add("field %s: meta.relation.with must name a table of the file", source)
		}
		switch nestedString(f.Meta, metaRelation, "type") {
		case "one-to-one", "has-many", "many-to-many":
		default:
			problems.add("field %s: meta.relation.type must be one-to-one, has-many or many-to-many", source)
		}
		if t, ok := plan.relTarget[rel]; ok && plan.fieldTable[t] != with {
			problems.add("field %s: relation @%s target field is not in table @%s", t, rel, with)
		}
	}
}

// validateRefs checks that every @key inside a meta points at something in the file, and that a lookup is complete.
func validateRefs(plan *importPlan, problems *problemList) {
	collectRelations(plan, problems)
	for _, t := range plan.file.Schema.Tables {
		for _, f := range t.Fields {
			if f.System {
				continue
			}
			validateFieldRefs(plan, f, problems)
		}
	}
}

// validateFieldRefs checks the @keys in the meta of one field, and that a lookup is complete
func validateFieldRefs(plan *importPlan, f dto.BaseExportField, problems *problemList) {
	refs := map[string]struct{}{}
	collectRefs(f.Meta, refs)
	for ref := range refs {
		if !plan.knownRef(ref) {
			problems.add("field %s: meta refers to %s, which is not in the file", f.Key, ref)
		}
	}
	if f.UIDT == uidtLookup {
		validateLookupField(plan, f, problems)
	}
}

// validateLookupField checks that a lookup names the field it reads and the relation it goes through, and that the
// relation has a link field
func validateLookupField(plan *importPlan, f dto.BaseExportField, problems *problemList) {
	for _, k := range []string{"lookup_column_id", metaRelationID} {
		if !refPattern.MatchString(idString(f.Meta[k])) {
			problems.add("field %s: lookup meta.%s must be a @key", f.Key, k)
		}
	}
	if rel := strings.TrimPrefix(idString(f.Meta[metaRelationID]), dto.BaseExportRefPrefix); rel != "" {
		if _, ok := plan.relSource[rel]; !ok {
			problems.add("field %s: lookup uses relation @%s, which has no link field", f.Key, rel)
		}
	}
	validateLookupTarget(plan, f, problems)
}

// collectViews gathers the views of every table, then the views of the top-level list (format 1.0.0). A view
// inside a table may leave out `table`, but if it names one it must be its own.
func collectViews(plan *importPlan, problems *problemList) {
	for i, tb := range plan.file.Schema.Tables {
		for j, v := range tb.Views {
			where := fmt.Sprintf("schema.tables[%d].views[%d]", i, j)
			if v.Table != "" && v.Table != tb.Key {
				problems.add("%s: table %q does not match the table it is in (%s)", where, v.Table, tb.Key)
			}
			plan.views = append(plan.views, planView{view: v, table: tb.Key, where: where})
		}
	}
	for i, v := range plan.file.Schema.Views {
		plan.views = append(plan.views, planView{view: v, table: v.Table, where: fmt.Sprintf("schema.views[%d]", i)})
	}
}

func validateViews(plan *importPlan, checkKey func(key, what, where string) bool, problems *problemList) {
	for _, pv := range plan.views {
		v, where := pv.view, pv.where
		checkKey(v.Key, "view", where)
		if _, ok := plan.tableIndex[pv.table]; !ok {
			problems.add("%s: table %q is not in the file", where, pv.table)
		}
		if strings.TrimSpace(v.Title) == "" || strings.TrimSpace(v.Type) == "" {
			problems.add("%s: title and type are required", where)
		} else if longer(v.Title, maxTitleLength) {
			problems.add("%s", tooLongProblem(where, "view name", v.Title))
		}
		refs := map[string]struct{}{}
		collectRefs(v.Meta, refs)
		for ref := range refs {
			if !plan.knownRef(ref) {
				problems.add("%s: meta refers to %s, which is not in the file", where, ref)
			}
		}
	}
}

func validateData(plan *importPlan, problems *problemList) {
	data := plan.file.Data
	if data == nil {
		return
	}
	rids := collectRowIDs(plan, data, problems)
	cells := &cellChecker{
		plan:     plan,
		rids:     rids,
		refs:     validateAttachmentIndex(data, problems),
		usedIDs:  map[string]struct{}{},
		problems: problems,
	}
	for tkey, rows := range data.Tables {
		for i, r := range rows.Rows {
			for fkey, val := range r.Values {
				cells.check(tkey, i, fkey, val)
			}
		}
	}
}

// collectRowIDs checks the tables and the _rid of the rows of the data, and returns every _rid -> its table key
func collectRowIDs(plan *importPlan, data *dto.BaseExportData, problems *problemList) map[string]string {
	rids := map[string]string{} // _rid -> table key
	for tkey, rows := range data.Tables {
		if _, ok := plan.tableIndex[tkey]; !ok {
			problems.add("data.tables: table %q is not in the schema", tkey)
			continue
		}
		if want, ok := data.RowCounts[tkey]; ok && want != len(rows.Rows) {
			problems.add("data.row_counts[%s] is %d but the table has %d rows", tkey, want, len(rows.Rows))
		}
		for i, r := range rows.Rows {
			if r.RID == "" {
				problems.add("data.tables.%s.rows[%d]: _rid is required", tkey, i)
				continue
			}
			if longer(r.RID, maxRIDLength) {
				problems.add("data.tables.%s.rows[%d]: _rid is longer than %d characters", tkey, i, maxRIDLength)
				continue
			}
			if prev, dup := rids[r.RID]; dup {
				problems.add("data.tables.%s.rows[%d]: _rid %q is already used in table %s", tkey, i, r.RID, prev)
			}
			rids[r.RID] = tkey
		}
	}
	return rids
}

// cellChecker checks the cells of the rows: rids is every _rid -> its table, refs the refs of the attachment files, and
// usedIDs the ids the rows of a source column have taken so far
type cellChecker struct {
	plan     *importPlan
	rids     map[string]string
	refs     map[string]struct{}
	usedIDs  map[string]struct{}
	problems *problemList
}

// check checks the value of one field of one row
func (c *cellChecker) check(tkey string, i int, fkey string, val interface{}) {
	f, ok := c.plan.fields[fkey]
	where := fmt.Sprintf("data.tables.%s.rows[%d].values.%s", tkey, i, fkey)
	switch {
	case !ok || c.plan.fieldTable[fkey] != tkey:
		c.problems.add("%s: field is not in table %s", where, tkey)
	case f.System && isSourceColumn(f.ColumnName):
		checkSourceValue(f, val, where, c.problems)
		if f.ColumnName == "id" {
			c.checkUniqueID(tkey, val, where)
		}
	case f.System || f.UIDT == uidtLookup:
		c.problems.add("%s: %s fields are calculated and cannot hold values", where, describeField(f))
	case f.UIDT == uidtLinks:
		checkLinkCell(f, val, c.rids, where, c.problems)
	case f.UIDT == uidtAttachment:
		checkAttachmentCell(val, c.refs, where, c.problems)
	}
}

// checkUniqueID checks that two rows of a table do not get the same id
func (c *cellChecker) checkUniqueID(tkey string, val interface{}, where string) {
	k := tkey + "|" + cellKey(val)
	if _, dup := c.usedIDs[k]; dup {
		c.problems.add("%s: the id %s is used by another row of table %s", where, cellKey(val), tkey)
	}
	c.usedIDs[k] = struct{}{}
}

// checkSourceValue checks the value of a source column (id, created / modified time and user) of a row.
func checkSourceValue(f dto.BaseExportField, val interface{}, where string, problems *problemList) {
	switch f.ColumnName {
	case "id":
		if n := numericID(val); n < 1 || n != math.Trunc(n) {
			problems.add("%s: the id must be a whole number above 0", where)
		}
	case "created_time", "last_modified_time":
		text, _ := val.(string)
		if _, ok := parseSourceTime(text); !ok {
			problems.add("%s: a time must be text like 2025-01-31T10:00:00Z", where)
		}
	default:
		if _, ok := val.(string); !ok {
			problems.add("%s: a user must be text", where)
		}
	}
}

func describeField(f dto.BaseExportField) string {
	if f.System {
		return "system"
	}
	return f.UIDT
}

func checkLinkCell(f dto.BaseExportField, val interface{}, rids map[string]string, where string, problems *problemList) {
	list, ok := val.([]interface{})
	if !ok {
		problems.add("%s: a link cell must be an array of _rid", where)
		return
	}
	// Both sides of a link name the table on the other side in meta.relation.with
	with := strings.TrimPrefix(nestedString(f.Meta, metaRelation, "with"), dto.BaseExportRefPrefix)
	for _, item := range list {
		rid, isStr := item.(string)
		if !isStr {
			problems.add("%s: link cell items must be _rid strings", where)
			continue
		}
		if table, known := rids[rid]; !known {
			problems.add("%s: _rid %q is not in the file", where, rid)
		} else if with != "" && table != with {
			problems.add("%s: _rid %q is a row of table %s, not %s", where, rid, table, with)
		}
	}
}

// validateAttachmentIndex checks the list of files and returns their refs.
func validateAttachmentIndex(data *dto.BaseExportData, problems *problemList) map[string]struct{} {
	refs := map[string]struct{}{}
	paths := map[string]struct{}{}
	var total int64
	for i, a := range data.Attachments {
		where := fmt.Sprintf("data.attachments[%d]", i)
		switch {
		case !attachmentRefPattern.MatchString(a.Ref):
			problems.add("%s: ref %q is not valid", where, a.Ref)
		case a.Filename == "":
			problems.add("%s: filename is required", where)
		case longer(a.Filename, maxTitleLength):
			problems.add("%s: filename is longer than %d characters", where, maxTitleLength)
		case a.SHA256 != "" && !sha256Pattern.MatchString(a.SHA256):
			problems.add("%s: sha256 must be \"sha256:\" followed by 64 hex characters", where)
		case !attachmentPathPattern.MatchString(a.Path):
			problems.add("%s: path %q must be a file directly in attachments/", where, a.Path)
		case a.Size < 0 || a.Size > MaxAttachmentBytes:
			problems.add("%s: size must be between 0 and %d MB", where, MaxAttachmentBytes>>20)
		default:
			if _, dup := refs[a.Ref]; dup {
				problems.add("%s: ref %q is used twice", where, a.Ref)
			}
			if _, dup := paths[a.Path]; dup {
				problems.add("%s: path %q is used twice", where, a.Path)
			}
			refs[a.Ref] = struct{}{}
			paths[a.Path] = struct{}{}
			total += a.Size
		}
	}
	if total > MaxAttachmentsTotalBytes {
		problems.add("data.attachments: files add up to more than %d MB", MaxAttachmentsTotalBytes>>20)
	}
	return refs
}

func checkAttachmentCell(val interface{}, refs map[string]struct{}, where string, problems *problemList) {
	list, ok := val.([]interface{})
	if !ok {
		problems.add("%s: an attachment cell must be an array of refs", where)
		return
	}
	for _, item := range list {
		ref, isStr := item.(string)
		if _, known := refs[ref]; !isStr || !known {
			problems.add("%s: attachment ref %v is not in data.attachments", where, item)
		}
	}
}

func validateCover(c *dto.BaseExportCover, problems *problemList) {
	if c == nil {
		return
	}
	switch {
	case strings.TrimSpace(c.Filename) == "":
		problems.add("base.cover: filename is required")
	case longer(c.Filename, maxTitleLength):
		problems.add("base.cover: filename is longer than %d characters", maxTitleLength)
	case c.SHA256 != "" && !sha256Pattern.MatchString(c.SHA256):
		problems.add("base.cover: sha256 must be \"sha256:\" followed by 64 hex characters")
	case !attachmentPathPattern.MatchString(c.Path):
		problems.add("base.cover: path %q must be a file directly in attachments/", c.Path)
	case c.Size < 0 || c.Size > MaxAttachmentBytes:
		problems.add("base.cover: size must be between 0 and %d MB", MaxAttachmentBytes>>20)
	}
}

// validateTableExtras checks the primary key, the constraints and the formula settings of each table.
func validateTableExtras(plan *importPlan, problems *problemList) {
	for i, t := range plan.file.Schema.Tables {
		where := fmt.Sprintf("schema.tables[%d]", i)
		validatePrimaryKey(plan, t, where, problems)
		for j, c := range t.Constraints {
			validateConstraint(plan, t.Key, fmt.Sprintf("%s.constraints[%d]", where, j), c, problems)
		}
		for j, f := range t.Fields {
			validateFormulaMeta(f, fmt.Sprintf("%s.fields[%d]", where, j), problems)
		}
	}
}

// validatePrimaryKey checks that the key of a table is made of its own plain fields, each listed once
func validatePrimaryKey(plan *importPlan, t dto.BaseExportTable, where string, problems *problemList) {
	seen := map[string]bool{}
	for _, key := range t.PrimaryKey {
		f, ok := plan.fields[key]
		switch {
		case !ok || plan.fieldTable[key] != t.Key:
			problems.add("%s.primary_key: %q is not a field of table %s", where, key, t.Key)
		case f.System:
			problems.add("%s.primary_key: %q is a system field and cannot be a key", where, key)
		case f.UIDT == uidtLinks || f.UIDT == uidtLookup || f.UIDT == uidtAttachment:
			problems.add("%s.primary_key: %q is a %s field and cannot be a key", where, key, f.UIDT)
		case seen[key]:
			problems.add("%s.primary_key: %q is listed twice", where, key)
		}
		seen[key] = true
	}
}

// validateFormulaMeta checks that the formula of a formula field is text
func validateFormulaMeta(f dto.BaseExportField, where string, problems *problemList) {
	if f.UIDT != "formula" || f.System {
		return
	}
	if v, has := f.Meta["formula"]; has {
		if _, isString := v.(string); !isString {
			problems.add("%s: meta.formula must be text", where)
		}
	}
}

func validateConstraint(plan *importPlan, tableKey, where string, c dto.BaseExportConstraint, problems *problemList) {
	if strings.TrimSpace(c.Title) == "" {
		problems.add(msgTitleRequired, where)
	}
	inTable := func(key string) bool { return plan.fieldTable[key] == tableKey }
	for _, key := range c.FieldKeys {
		if !inTable(key) {
			problems.add("%s: field %q is not in table %s", where, key, tableKey)
		}
	}
	switch c.Type {
	case "unique":
		if len(c.FieldKeys) == 0 {
			problems.add("%s: a unique constraint needs field_keys", where)
		}
	case "check":
		validateCheckConstraint(c, inTable, where, tableKey, problems)
	default:
		problems.add("%s: type must be unique or check", where)
	}
}

// validateCheckConstraint checks the condition and the category of a check constraint, and that the condition only
// uses fields of its own table
func validateCheckConstraint(c dto.BaseExportConstraint, inTable func(key string) bool, where, tableKey string, problems *problemList) {
	if strings.TrimSpace(c.Condition) == "" {
		problems.add("%s: a check constraint needs a condition", where)
	}
	if c.Category != "" && !checkCategories[c.Category] {
		problems.add("%s: category %q is not one of number_date, text, compare, custom", where, c.Category)
	}
	for _, key := range conditionKeys(c.Condition) {
		if !inTable(key) {
			problems.add("%s: condition uses {%s}, which is not a field of table %s", where, key, tableKey)
		}
	}
}

// validateLookupTarget applies the rules the app checks when it creates a lookup, so a file that would fail halfway
// through the import is refused before anything is created: the field it reads must be a plain field (not a link,
// another lookup or a rollup) on the table the link points at.
func validateLookupTarget(plan *importPlan, f dto.BaseExportField, problems *problemList) {
	shown := strings.TrimPrefix(idString(f.Meta["lookup_column_id"]), dto.BaseExportRefPrefix)
	target, ok := plan.fields[shown]
	if !ok {
		return // an unknown @key is reported on its own
	}
	switch target.UIDT {
	case uidtLinks, uidtLookup, "rollup":
		problems.add("field %s: a lookup cannot read the %s field %s; it must read a plain field", f.Key, target.UIDT, shown)
		return
	}

	// the link it goes through is a link field of the lookup's own table with the same relation
	pointsAt := tablesLinkedFor(plan, f)
	if len(pointsAt) > 0 && !pointsAt[plan.fieldTable[shown]] {
		problems.add("field %s: the field it reads (%s) is not on the table its link points at", f.Key, shown)
	}
}

// tablesLinkedFor is the set of tables the link fields of a lookup's own table point at, for the relation (and the link
// field, when it names one) of the lookup
func tablesLinkedFor(plan *importPlan, f dto.BaseExportField) map[string]bool {
	rel := strings.TrimPrefix(idString(f.Meta[metaRelationID]), dto.BaseExportRefPrefix)
	through := strings.TrimPrefix(idString(f.Meta["link_column_id"]), dto.BaseExportRefPrefix)
	own := plan.fieldTable[f.Key]
	pointsAt := map[string]bool{}
	for key, candidate := range plan.fields {
		if plan.fieldTable[key] != own || candidate.UIDT != uidtLinks || candidate.System {
			continue
		}
		if strings.TrimPrefix(idString(candidate.Meta[metaRelationID]), dto.BaseExportRefPrefix) != rel || (through != "" && key != through) {
			continue
		}
		pointsAt[strings.TrimPrefix(nestedString(candidate.Meta, metaRelation, "with"), dto.BaseExportRefPrefix)] = true
	}
	return pointsAt
}
