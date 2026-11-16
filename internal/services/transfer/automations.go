// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

// Trigger functions and triggers are SQL that Postgres runs. In a copy of a base the physical names differ (table
// aliases and column names are generated per base), and a function name is global in the schema, so the SQL is
// written with placeholders and rebuilt for the new base:
//
//	{t_key}   the physical table of table t_key
//	{f_key}   the column of field f_key
//	{fn_key}  the function fn_key (each copy gets its own name, so a copy never replaces the original's function)
//	{schema}  the schema, quoted
const (
	// maxAutomationContext is the longest SQL text of one trigger or function
	maxAutomationContext = 100_000
	// functionSuffixLength is the number of hex characters added to the name of an imported function
	functionSuffixLength = 8
)

var (
	functionHeader   = regexp.MustCompile(`(?is)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+((?:"?\w+"?\.)?"?\w+"?)\s*\(`)
	tableToken       = regexp.MustCompile(`\{(t_[a-z0-9_]{1,62})\}`)
	functionToken    = regexp.MustCompile(`\{(fn_[a-z0-9_]{1,61})\}`)
	generatedSuffix  = regexp.MustCompile(`_[0-9a-f]{8}$`)
	identifierInName = regexp.MustCompile(`[^a-z0-9_]+`)
)

const schemaToken = "{schema}"

// repeatedTiming finds "... OR BEFORE ..." (or AFTER, INSTEAD OF) in a trigger: the timing is written once
var repeatedTiming = regexp.MustCompile(`(?i)\bOR\s+(BEFORE|AFTER|INSTEAD\s+OF)\b`)

// functionNameOf returns the name of the function a "CREATE FUNCTION" text defines, without schema or quotes.
func functionNameOf(context string) string {
	m := functionHeader.FindStringSubmatch(context)
	if m == nil {
		return ""
	}
	parts := strings.Split(m[1], ".")
	return strings.Trim(parts[len(parts)-1], `"`)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

type exportedAutomations struct {
	functions []dto.BaseExportFunction
	triggers  []dto.BaseExportTrigger
}

// exportAutomations reads the trigger functions and triggers of every table and writes their SQL with placeholders.
func (s *baseTransferService) exportAutomations(ctx context.Context, schemaName string, tables []*exportTable, skipped map[string]struct{}, warnings *[]string) (map[string]*exportedAutomations, error) {
	if s.automations == nil {
		*warnings = append(*warnings, "triggers and functions not exported: the automation service is not configured")
		return nil, nil
	}

	type entry struct {
		table *exportTable
		a     tenant.Automation
	}
	var functions, triggers []entry
	for _, t := range tables {
		list, err := s.automations.List(ctx, schemaName, t.model.ID.String())
		if err != nil {
			return nil, err
		}
		// List returns the newest first; keep the order they were made in
		sort.SliceStable(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
		for _, a := range list {
			switch a.Type {
			case tenant.AutomationTypeFunction:
				functions = append(functions, entry{t, a})
			case tenant.AutomationTypeTrigger:
				triggers = append(triggers, entry{t, a})
			}
			// a webhook calls out of the system and is not exported
		}
	}

	// Names that are replaced in every text: tables, functions, and (preferring the owner's) columns
	aliases := map[string]string{}
	for _, t := range tables {
		aliases[t.model.Alias] = t.key
	}
	fnAlloc := newKeyAllocator()
	fnKeys := map[string]string{} // function name -> key
	for _, f := range functions {
		name := functionNameOf(f.a.Context)
		if name == "" {
			continue
		}
		if _, dup := fnKeys[strings.ToLower(name)]; !dup {
			fnKeys[strings.ToLower(name)] = clipKey(fnAlloc.next("fn_", name))
		}
	}
	functionNames := map[string]string{}
	for _, f := range functions {
		if name := functionNameOf(f.a.Context); name != "" {
			functionNames[name] = fnKeys[strings.ToLower(name)]
			// an unquoted name written in another case still refers to the same function
			functionNames[strings.ToLower(name)] = fnKeys[strings.ToLower(name)]
		}
	}

	tokenize := func(owner *exportTable, text string) (string, string) {
		// A text that uses a field that is not exported cannot be rebuilt
		for _, t := range tables {
			for _, c := range t.columns {
				if _, gone := skipped[c.ID.String()]; gone && containsWord(text, c.ColumnName) {
					return "", fmt.Sprintf("it uses the field %s.%s, which is not exported", t.model.Title, c.Title)
				}
			}
		}
		columns := map[string]string{}
		for _, t := range tables {
			for _, c := range t.columns {
				if _, gone := skipped[c.ID.String()]; !gone {
					if _, taken := columns[c.ColumnName]; !taken {
						columns[c.ColumnName] = t.fieldKeys[c.ID.String()]
					}
				}
			}
		}
		for _, c := range owner.columns {
			if _, gone := skipped[c.ID.String()]; !gone {
				columns[c.ColumnName] = owner.fieldKeys[c.ID.String()]
			}
		}
		out := tokenizeCondition(text, aliases)
		out = tokenizeCondition(out, functionNames)
		out = tokenizeCondition(out, columns)
		return tokenizeSchema(out, schemaName), ""
	}

	result := map[string]*exportedAutomations{}
	get := func(t *exportTable) *exportedAutomations {
		if result[t.key] == nil {
			result[t.key] = &exportedAutomations{}
		}
		return result[t.key]
	}

	for _, f := range functions {
		name := functionNameOf(f.a.Context)
		if name == "" {
			*warnings = append(*warnings, fmt.Sprintf("function %q of table %s not exported: its name could not be read", f.a.Title, f.table.model.Title))
			continue
		}
		text, reason := tokenize(f.table, f.a.Context)
		if reason != "" {
			*warnings = append(*warnings, fmt.Sprintf("function %q of table %s not exported: %s", f.a.Title, f.table.model.Title, reason))
			continue
		}
		get(f.table).functions = append(get(f.table).functions, dto.BaseExportFunction{Key: fnKeys[strings.ToLower(name)], Title: f.a.Title, Context: text})
	}
	for _, tr := range triggers {
		text, reason := tokenize(tr.table, tr.a.Context)
		if reason != "" {
			*warnings = append(*warnings, fmt.Sprintf("trigger %q of table %s not exported: %s", tr.a.Title, tr.table.model.Title, reason))
			continue
		}
		get(tr.table).triggers = append(get(tr.table).triggers, dto.BaseExportTrigger{Title: tr.a.Title, Event: tr.a.Event, Context: text})
	}
	return result, nil
}

// containsWord reports whether text has name as a whole word.
func containsWord(text, name string) bool {
	if name == "" {
		return false
	}
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(text)
}

// tokenizeSchema writes the schema qualifier ("s". or s.) as {schema}.
func tokenizeSchema(text, schemaName string) string {
	if schemaName == "" {
		return text
	}
	q := regexp.QuoteMeta(schemaName)
	return regexp.MustCompile(`(?:"`+q+`"|\b`+q+`\b)\s*\.`).ReplaceAllString(text, schemaToken+".")
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// validateAutomations checks the trigger functions and triggers of the tables.
func validateAutomations(plan *importPlan, checkKey func(key, what, where string) bool, problems *problemList) {
	plan.functionKeys = map[string]string{}
	for i, t := range plan.file.Schema.Tables {
		for j, f := range t.Functions {
			where := fmt.Sprintf("schema.tables[%d].functions[%d]", i, j)
			if checkKey(f.Key, "function", where) {
				plan.functionKeys[f.Key] = t.Key
			}
		}
	}
	for i, t := range plan.file.Schema.Tables {
		titles := map[string]bool{}
		for j, f := range t.Functions {
			where := fmt.Sprintf("schema.tables[%d].functions[%d]", i, j)
			validateAutomationText(plan, where, "function", f.Title, f.Context, titles, problems)
			if functionNameOf(f.Context) == "" && !strings.Contains(f.Context, "{fn_") {
				problems.add("%s: context must start with CREATE FUNCTION", where)
			}
		}
		titles = map[string]bool{}
		for j, tr := range t.Triggers {
			where := fmt.Sprintf("schema.tables[%d].triggers[%d]", i, j)
			validateAutomationText(plan, where, "trigger", tr.Title, tr.Context, titles, problems)
			if strings.TrimSpace(tr.Event) == "" {
				problems.add("%s: event is required, e.g. \"BEFORE INSERT, BEFORE UPDATE\"", where)
			}
			if repeatedTiming.MatchString(tr.Context) {
				problems.add("%s: a trigger names its timing once, as in BEFORE INSERT OR UPDATE ON ...; Postgres rejects BEFORE INSERT OR BEFORE UPDATE", where)
			}
		}
	}
}

func validateAutomationText(plan *importPlan, where, kind, title, context string, titles map[string]bool, problems *problemList) {
	switch {
	case strings.TrimSpace(title) == "":
		problems.add("%s: title is required", where)
	case longer(title, maxTitleLength):
		problems.add("%s: title is longer than %d characters", where, maxTitleLength)
	case titles[strings.ToLower(strings.TrimSpace(title))]:
		problems.add("%s: another %s of the table already has the title %q", where, kind, title)
	}
	titles[strings.ToLower(strings.TrimSpace(title))] = true

	switch {
	case strings.TrimSpace(context) == "":
		problems.add("%s: context is required", where)
		return
	case longer(context, maxAutomationContext):
		problems.add("%s: context is longer than %d characters", where, maxAutomationContext)
		return
	}
	for _, m := range tableToken.FindAllStringSubmatch(context, -1) {
		if _, ok := plan.tableIndex[m[1]]; !ok {
			problems.add("%s: {%s} is not a table of the file", where, m[1])
		}
	}
	for _, key := range conditionKeys(context) {
		if _, ok := plan.fields[key]; !ok {
			problems.add("%s: {%s} is not a field of the file", where, key)
		}
	}
	for _, m := range functionToken.FindAllStringSubmatch(context, -1) {
		if _, ok := plan.functionKeys[m[1]]; !ok {
			problems.add("%s: {%s} is not a function of the file", where, m[1])
		}
	}
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

// functionName returns the name the function fn_key gets in the new base. Every import uses its own, so a copy made in
// the same database never replaces (or collides with) the function of the original.
func (r *importRun) functionName(key string) string {
	if name, ok := r.functionNames[key]; ok {
		return name
	}
	base := generatedSuffix.ReplaceAllString(strings.TrimPrefix(key, "fn_"), "")
	base = identifierInName.ReplaceAllString(strings.ToLower(base), "_")
	if base == "" || (base[0] >= '0' && base[0] <= '9') {
		base = "fn_" + base
	}
	sum := sha256.Sum256([]byte(r.baseID.String()))
	suffix := hex.EncodeToString(sum[:])[:functionSuffixLength]
	if len(base) > 63-1-functionSuffixLength {
		base = base[:63-1-functionSuffixLength]
	}
	name := base + "_" + suffix
	r.functionNames[key] = name
	return name
}

// automationSQL rebuilds the SQL of a function or trigger for the new base.
func (r *importRun) automationSQL(text string) string {
	text = tableToken.ReplaceAllStringFunc(text, func(tok string) string {
		if t, ok := r.tables[tok[1:len(tok)-1]]; ok {
			return t.Model.Alias
		}
		return tok
	})
	names := map[string]string{}
	for key, col := range r.columns {
		names[key] = col.ColumnName
	}
	text = detokenizeCondition(text, names)
	text = functionToken.ReplaceAllStringFunc(text, func(tok string) string { return r.functionName(tok[1 : len(tok)-1]) })
	return strings.ReplaceAll(text, schemaToken, `"`+r.schema+`"`)
}

// createAutomations creates the trigger functions, then the triggers, after all rows exist so that a trigger never
// fires while the base is being filled.
func (r *importRun) createAutomations() error {
	if !r.includeAutomations {
		return nil
	}
	if r.s.automations == nil {
		r.warn("triggers and functions were not created: the automation service is not configured")
		return nil
	}
	create := func(table dto.BaseExportTable, kind, title, event, text string) error {
		a, err := r.s.automations.Create(r.ctx, r.schema, dto.CreateAutomationRequest{
			ModelID:   r.tables[table.Key].Model.ID.String(),
			Title:     title,
			Type:      kind,
			Context:   r.automationSQL(text),
			Event:     event,
			CreatedBy: r.userID,
		})
		if err != nil {
			return fmt.Errorf("create %s %q on table %q: %w", kind, title, table.Title, err)
		}
		r.automationIDs = append(r.automationIDs, a.ID.String())
		return nil
	}
	for _, t := range r.plan.file.Schema.Tables {
		for _, f := range t.Functions {
			if err := create(t, tenant.AutomationTypeFunction, f.Title, "", f.Context); err != nil {
				return err
			}
			r.result.Functions++
		}
	}
	for _, t := range r.plan.file.Schema.Tables {
		for _, tr := range t.Triggers {
			if err := create(t, tenant.AutomationTypeTrigger, tr.Title, tr.Event, tr.Context); err != nil {
				return err
			}
			r.result.Triggers++
		}
	}
	return nil
}

// countAutomations returns how many functions and triggers a file holds.
func countAutomations(file *dto.BaseExportFile) (functions, triggers int) {
	for _, t := range file.Schema.Tables {
		functions += len(t.Functions)
		triggers += len(t.Triggers)
	}
	return functions, triggers
}
