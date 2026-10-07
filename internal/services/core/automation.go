// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
)

// functionHeaderRegexp matches the start of an optional trigger function placed before the trigger:
// CREATE [OR REPLACE] FUNCTION name() RETURNS trigger [LANGUAGE plpgsql] AS $tag$
var functionHeaderRegexp = regexp.MustCompile(`(?is)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+((?:"?\w+"?\.)?"?\w+"?)\s*\(\s*\)\s*RETURNS\s+trigger\s+(?:LANGUAGE\s+plpgsql\s+)?AS\s+(\$\w*\$)`)

// functionFooterRegexp matches what may follow the function body: [LANGUAGE plpgsql] followed by ; or the end
var functionFooterRegexp = regexp.MustCompile(`(?is)^\s*(?:LANGUAGE\s+plpgsql\s*)?(?:;|$)`)

// createFunctionRegexp matches the start of a CREATE [OR REPLACE] FUNCTION query
var createFunctionRegexp = regexp.MustCompile(`(?is)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION`)

// triggerQueryRegexp captures the trigger name and target table of a CREATE TRIGGER query
var triggerQueryRegexp = regexp.MustCompile(`(?is)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?TRIGGER\s+("?\w+"?)\s.*?\sON\s+((?:"?\w+"?\.)?"?\w+"?)\s`)

type automationService struct {
	repo         *pkg.DatabaseService
	modelService interfaces.ModelService
}

func NewAutomationService(repo *pkg.DatabaseService, modelService interfaces.ModelService) interfaces.AutomationService {
	return &automationService{repo: repo, modelService: modelService}
}

// Create stores the automation; for a trigger or function it also runs the query stored in context
func (s *automationService) Create(ctx context.Context, schemaName string, req dto.CreateAutomationRequest) (tenant.Automation, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return tenant.Automation{}, err
	}

	query := strings.TrimSuffix(strings.TrimSpace(req.Context), ";")
	sql, err := prepareQuery(req.Type, query, model.Alias)
	if err != nil {
		return tenant.Automation{}, err
	}
	event, err := normalizeEvent(req.Type, req.Event)
	if err != nil {
		return tenant.Automation{}, err
	}

	automation := tenant.Automation{
		ID:      uuid.New(),
		ModelID: req.ModelID,
		Title:   req.Title,
		Type:    req.Type,
		Context: query,
	}
	label, err := labelSQL(automation, query, model.Alias)
	if err != nil {
		return tenant.Automation{}, err
	}
	if err := s.execInTx(sql, label); err != nil {
		return tenant.Automation{}, err
	}

	tableName := tenant.Automation{}.TableName(schemaName)
	inserted, err := s.repo.TableService.CreateRecord(tableName, map[string]interface{}{
		"id":               automation.ID,
		"model_id":         automation.ModelID,
		"title":            automation.Title,
		"type":             automation.Type,
		"context":          automation.Context,
		"event":            event,
		"created_by":       req.CreatedBy,
		"last_modified_by": req.CreatedBy,
	})
	if err != nil {
		switch automation.Type {
		case tenant.AutomationTypeTrigger:
			s.dropTrigger(query, model.Alias)
		case tenant.AutomationTypeFunction:
			s.dropFunction(query)
		}
		return tenant.Automation{}, app_errors.LogDatabaseError(err, "failed to create automation")
	}

	var out tenant.Automation
	if err := helpers.MapToStruct(inserted, &out); err != nil {
		return tenant.Automation{}, app_errors.ErrMapToStruct
	}
	return out, nil
}

// Update saves the title and query. Postgres is not changed until the entry is run.
func (s *automationService) Update(ctx context.Context, schemaName, id string, req dto.UpdateAutomationRequest) (tenant.Automation, error) {
	automation, err := s.getByID(schemaName, id)
	if err != nil {
		return tenant.Automation{}, err
	}
	model, err := s.modelService.GetModelByID(ctx, schemaName, automation.ModelID)
	if err != nil {
		return tenant.Automation{}, err
	}

	query := strings.TrimSuffix(strings.TrimSpace(req.Context), ";")
	// Only the format is checked here; Run applies it
	if _, err := prepareQuery(automation.Type, query, model.Alias); err != nil {
		return tenant.Automation{}, err
	}
	event, err := normalizeEvent(automation.Type, req.Event)
	if err != nil {
		return tenant.Automation{}, err
	}
	// Entries saved before labels existed: label what is installed now, while the saved query still names it
	s.ensureLabel(automation, model.Alias)

	updated, err := s.repo.TableService.UpdateRecord(tenant.Automation{}.TableName(schemaName), id, map[string]interface{}{
		"title":              req.Title,
		"context":            query,
		"event":              event,
		"last_modified_by":   req.UpdatedBy,
		"last_modified_time": time.Now(),
	})
	if err != nil {
		return tenant.Automation{}, app_errors.LogDatabaseError(err, "failed to update automation")
	}

	var out tenant.Automation
	if err := helpers.MapToStruct(updated, &out); err != nil {
		return tenant.Automation{}, app_errors.ErrMapToStruct
	}
	return out, nil
}

// Run applies the saved query in Postgres. The trigger or function installed for this entry is
// found in Postgres by its label and replaced in one transaction, so if the query fails it keeps working.
func (s *automationService) Run(ctx context.Context, schemaName, id string) (tenant.Automation, error) {
	automation, err := s.getByID(schemaName, id)
	if err != nil {
		return tenant.Automation{}, err
	}
	model, err := s.modelService.GetModelByID(ctx, schemaName, automation.ModelID)
	if err != nil {
		return tenant.Automation{}, err
	}

	query := automation.Context
	sql, err := prepareQuery(automation.Type, query, model.Alias)
	if err != nil {
		return tenant.Automation{}, err
	}
	label, err := labelSQL(automation, query, model.Alias)
	if err != nil {
		return tenant.Automation{}, err
	}

	switch automation.Type {
	case tenant.AutomationTypeTrigger:
		installed, table, err := s.installedTrigger(automation, model.Alias)
		if err != nil {
			return tenant.Automation{}, err
		}
		err = s.execInTx(fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", installed, table), sql, label)
		if err != nil {
			return tenant.Automation{}, err
		}
	case tenant.AutomationTypeFunction:
		installed, err := s.installedFunction(automation)
		if err != nil {
			return tenant.Automation{}, err
		}
		_, newName, _, _ := splitTriggerContext(query)
		if sameIdentifier(installed, newName) {
			// Same function: replace its body, triggers using it keep working
			err = s.execInTx(createFunctionRegexp.ReplaceAllString(sql, "CREATE OR REPLACE FUNCTION"), label)
		} else {
			// Renamed: Postgres refuses to drop the old one while a trigger still uses it
			err = s.execInTx(sql, fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", installed), label)
		}
		if err != nil {
			return tenant.Automation{}, err
		}
	}
	return automation, nil
}

// List returns the automations of one table
func (s *automationService) List(ctx context.Context, schemaName, modelID string) ([]tenant.Automation, error) {
	return s.fetch(schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{{Column: "model_id", Operator: "eq", Value: modelID}},
		OrderBy: []string{"created_time DESC"},
	})
}

// Delete removes the automation and the trigger or function installed for it in Postgres
func (s *automationService) Delete(ctx context.Context, schemaName, id string) error {
	automation, err := s.getByID(schemaName, id)
	if err != nil {
		return err
	}

	if automation.Type == tenant.AutomationTypeFunction {
		shared, err := s.functionSavedElsewhere(schemaName, automation)
		if err != nil {
			return err
		}
		// Another entry defines the same Postgres function, so it stays
		if !shared {
			installed, err := s.installedFunction(automation)
			if err != nil {
				return err
			}
			if _, err := s.repo.DB.Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", installed)); err != nil {
				logger.Get().Error().Err(err).Str("function", installed).Msg("failed to drop function")
				return fmt.Errorf("%w: %v", app_errors.TriggerFailed, err)
			}
		}
	}

	if automation.Type == tenant.AutomationTypeTrigger {
		model, err := s.modelService.GetModelByID(ctx, schemaName, automation.ModelID)
		if err != nil {
			return err
		}
		installed, table, err := s.installedTrigger(automation, model.Alias)
		if err != nil {
			return err
		}
		if _, err := s.repo.DB.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", installed, table)); err != nil {
			logger.Get().Error().Err(err).Msg("failed to drop trigger")
			return fmt.Errorf("%w: %v", app_errors.TriggerFailed, err)
		}
		// Triggers saved with their own function (before functions had their own tab) drop it too
		if _, functionName, _, _ := splitTriggerContext(automation.Context); functionName != "" {
			if _, err := s.repo.DB.Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName)); err != nil {
				logger.Get().Warn().Err(err).Str("function", functionName).Msg("trigger function not dropped")
			}
		}
	}

	if err := s.repo.TableService.DeleteRecord(tenant.Automation{}.TableName(schemaName), id); err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete automation")
	}
	return nil
}

// dropTrigger drops the trigger in context and, for triggers saved with their own function, that function
func (s *automationService) dropTrigger(query, tableAlias string) error {
	_, functionName, triggerSQL, err := splitTriggerContext(query)
	if err != nil {
		return err
	}
	name, table, err := parseTriggerQuery(triggerSQL, tableAlias)
	if err != nil {
		return err
	}
	if _, err := s.repo.DB.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", name, table)); err != nil {
		logger.Get().Error().Err(err).Msg("failed to drop trigger")
		return fmt.Errorf("%w: %v", app_errors.TriggerFailed, err)
	}
	if functionName != "" {
		// Kept when another trigger still uses it
		if _, err := s.repo.DB.Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName)); err != nil {
			logger.Get().Warn().Err(err).Str("function", functionName).Msg("trigger function not dropped")
		}
	}
	return nil
}

// dropFunction drops the function in context; it fails while a trigger still uses it
func (s *automationService) dropFunction(query string) error {
	_, functionName, _, err := splitTriggerContext(query)
	if err != nil || functionName == "" {
		return app_errors.InvalidFunctionQuery
	}
	if _, err := s.repo.DB.Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName)); err != nil {
		logger.Get().Error().Err(err).Str("function", functionName).Msg("failed to drop function")
		return fmt.Errorf("%w: %v", app_errors.TriggerFailed, err)
	}
	return nil
}

// functionSavedElsewhere reports whether another function entry defines the same Postgres function
func (s *automationService) functionSavedElsewhere(schemaName string, automation tenant.Automation) (bool, error) {
	_, name, _, _ := splitTriggerContext(automation.Context)
	functions, err := s.fetch(schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{{Column: "type", Operator: "eq", Value: tenant.AutomationTypeFunction}},
	})
	if err != nil {
		return false, err
	}
	for _, other := range functions {
		if other.ID == automation.ID {
			continue
		}
		if _, otherName, _, _ := splitTriggerContext(other.Context); sameIdentifier(otherName, name) {
			return true, nil
		}
	}
	return false, nil
}

// automationLabel is stored as a Postgres comment on the trigger or function an entry installed,
// so Run and Delete can find it in Postgres even after the entry was renamed
func automationLabel(automation tenant.Automation) string {
	return "automation:" + automation.ID.String()
}

// labelSQL returns the COMMENT statement that labels the trigger or function defined by query
func labelSQL(automation tenant.Automation, query, tableAlias string) (string, error) {
	switch automation.Type {
	case tenant.AutomationTypeTrigger:
		name, table, err := parseTriggerQuery(query, tableAlias)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("COMMENT ON TRIGGER %s ON %s IS '%s'", name, table, automationLabel(automation)), nil
	case tenant.AutomationTypeFunction:
		_, name, _, _ := splitTriggerContext(query)
		if name == "" {
			return "", app_errors.InvalidFunctionQuery
		}
		return fmt.Sprintf("COMMENT ON FUNCTION %s() IS '%s'", name, automationLabel(automation)), nil
	}
	return "", nil
}

// labelledTrigger finds the trigger on table labelled with the entry's ID
func (s *automationService) labelledTrigger(automation tenant.Automation, table string) (string, bool) {
	var name string
	err := s.repo.DB.QueryRow(
		`SELECT quote_ident(t.tgname) FROM pg_trigger t
		 JOIN pg_description d ON d.objoid = t.oid AND d.classoid = 'pg_trigger'::regclass
		 WHERE t.tgrelid = $1::regclass AND NOT t.tgisinternal AND d.description = $2`,
		table, automationLabel(automation),
	).Scan(&name)
	return name, err == nil
}

// labelledFunction finds the function labelled with the entry's ID
func (s *automationService) labelledFunction(automation tenant.Automation) (string, bool) {
	var name string
	err := s.repo.DB.QueryRow(
		`SELECT quote_ident(p.proname) FROM pg_proc p
		 JOIN pg_description d ON d.objoid = p.oid AND d.classoid = 'pg_proc'::regclass
		 WHERE d.description = $1 LIMIT 1`,
		automationLabel(automation),
	).Scan(&name)
	return name, err == nil
}

// installedTrigger returns the name and table of the trigger installed for the entry: the one
// labelled with its ID, or (entries saved before labels existed) the one named in its query
func (s *automationService) installedTrigger(automation tenant.Automation, tableAlias string) (string, string, error) {
	_, _, triggerSQL, err := splitTriggerContext(automation.Context)
	if err != nil {
		return "", "", err
	}
	name, table, err := parseTriggerQuery(triggerSQL, tableAlias)
	if err != nil {
		return "", "", err
	}
	if labelled, ok := s.labelledTrigger(automation, table); ok {
		return labelled, table, nil
	}
	return name, table, nil
}

// installedFunction returns the name of the function installed for the entry: the one labelled
// with its ID, or (entries saved before labels existed) the one named in its query
func (s *automationService) installedFunction(automation tenant.Automation) (string, error) {
	if labelled, ok := s.labelledFunction(automation); ok {
		return labelled, nil
	}
	_, name, _, _ := splitTriggerContext(automation.Context)
	if name == "" {
		return "", app_errors.InvalidFunctionQuery
	}
	return name, nil
}

// ensureLabel labels the trigger or function an entry installed before labels existed.
// It runs before the saved query changes, while that query still names what is installed.
func (s *automationService) ensureLabel(automation tenant.Automation, tableAlias string) {
	switch automation.Type {
	case tenant.AutomationTypeTrigger:
		_, _, triggerSQL, err := splitTriggerContext(automation.Context)
		if err != nil {
			return
		}
		_, table, err := parseTriggerQuery(triggerSQL, tableAlias)
		if err != nil {
			return
		}
		if _, ok := s.labelledTrigger(automation, table); ok {
			return
		}
	case tenant.AutomationTypeFunction:
		if _, ok := s.labelledFunction(automation); ok {
			return
		}
	default:
		return
	}
	label, err := labelSQL(automation, automation.Context, tableAlias)
	if err != nil {
		return
	}
	if _, err := s.repo.DB.Exec(label); err != nil {
		logger.Get().Warn().Err(err).Str("automation", automation.ID.String()).Msg("could not label installed trigger or function")
	}
}

var (
	triggerTimings    = []string{"BEFORE", "AFTER", "INSTEAD OF"}
	triggerOperations = []string{"INSERT", "UPDATE", "DELETE", "TRUNCATE"}
)

// normalizeEvent checks the timing and events chosen for a trigger, e.g. "before insert, before update",
// and returns them in a fixed form: "BEFORE INSERT, BEFORE UPDATE". Other types have no event.
func normalizeEvent(automationType, event string) (string, error) {
	if automationType != tenant.AutomationTypeTrigger {
		return "", nil
	}
	timing := ""
	chosen := map[string]bool{}
	for _, part := range strings.Split(event, ",") {
		words := strings.Fields(strings.ToUpper(part))
		if len(words) < 2 {
			return "", app_errors.InvalidTriggerEvent
		}
		partTiming, operation := strings.Join(words[:len(words)-1], " "), words[len(words)-1]
		if !slices.Contains(triggerTimings, partTiming) || !slices.Contains(triggerOperations, operation) {
			return "", app_errors.InvalidTriggerEvent
		}
		// One trigger has one timing
		if timing != "" && partTiming != timing {
			return "", app_errors.InvalidTriggerEvent
		}
		timing = partTiming
		chosen[operation] = true
	}
	parts := make([]string, 0, len(chosen))
	for _, operation := range triggerOperations {
		if chosen[operation] {
			parts = append(parts, timing+" "+operation)
		}
	}
	return strings.Join(parts, ", "), nil
}

// sameIdentifier compares Postgres names the way Postgres does for unquoted names
func sameIdentifier(a, b string) bool {
	return strings.EqualFold(strings.Trim(a, `"`), strings.Trim(b, `"`))
}

func (s *automationService) getByID(schemaName, id string) (tenant.Automation, error) {
	limit := 1
	automations, err := s.fetch(schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{{Column: "id", Operator: "eq", Value: id}},
		Limit:   &limit,
	})
	if err != nil {
		return tenant.Automation{}, err
	}
	if len(automations) == 0 {
		return tenant.Automation{}, app_errors.AutomationNotFound
	}
	return automations[0], nil
}

// prepareQuery checks the query of a trigger or function and returns the SQL to run (none for a webhook)
func prepareQuery(automationType, query, tableAlias string) (string, error) {
	switch automationType {
	case tenant.AutomationTypeFunction:
		functionSQL, _, rest, err := splitTriggerContext(query)
		if err != nil || functionSQL == "" || rest != "" {
			return "", app_errors.InvalidFunctionQuery
		}
		return functionSQL, nil
	case tenant.AutomationTypeTrigger:
		// The function a trigger calls is saved on its own, in the Functions tab
		if functionHeaderRegexp.MatchString(query) {
			return "", app_errors.TriggerHasFunction
		}
		if _, _, err := parseTriggerQuery(query, tableAlias); err != nil {
			return "", err
		}
		return query, nil
	}
	return "", nil
}

func (s *automationService) execInTx(statements ...string) error {
	tx, err := s.repo.DB.Begin()
	if err != nil {
		return fmt.Errorf("%w: %v", app_errors.TriggerFailed, err)
	}
	for _, stmt := range statements {
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			logger.Get().Error().Err(err).Msg("failed to run trigger query")
			return fmt.Errorf("%w: %v", app_errors.TriggerFailed, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", app_errors.TriggerFailed, err)
	}
	return nil
}

func (s *automationService) fetch(schemaName string, params dbModels.QueryParams) ([]tenant.Automation, error) {
	rows, err := s.repo.TableService.GetTableData(tenant.Automation{}.TableName(schemaName), params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch automations")
	}
	automations := make([]tenant.Automation, 0, len(rows))
	for _, row := range rows {
		var automation tenant.Automation
		if err := helpers.MapToStruct(row, &automation); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		automations = append(automations, automation)
	}
	return automations, nil
}

// splitTriggerContext splits context into an optional leading
// "CREATE FUNCTION name() RETURNS trigger ... AS $$ ... $$;" and the CREATE TRIGGER query.
func splitTriggerContext(query string) (functionSQL, functionName, triggerSQL string, err error) {
	header := functionHeaderRegexp.FindStringSubmatchIndex(query)
	if header == nil {
		return "", "", query, nil
	}
	functionName = query[header[2]:header[3]]
	tag := query[header[4]:header[5]]

	bodyEnd := strings.Index(query[header[1]:], tag)
	if bodyEnd < 0 {
		return "", "", "", app_errors.InvalidTriggerQuery
	}
	bodyEnd += header[1] + len(tag)

	footer := functionFooterRegexp.FindStringIndex(query[bodyEnd:])
	if footer == nil {
		return "", "", "", app_errors.InvalidTriggerQuery
	}
	functionSQL = strings.TrimSuffix(query[:bodyEnd+footer[1]], ";")
	triggerSQL = strings.TrimSuffix(strings.TrimSpace(query[bodyEnd+footer[1]:]), ";")
	return functionSQL, functionName, triggerSQL, nil
}

// parseTriggerQuery accepts only a single CREATE TRIGGER statement on the given table
// and returns the trigger name and table as written in the query.
func parseTriggerQuery(query, tableAlias string) (string, string, error) {
	if strings.Contains(query, ";") {
		return "", "", app_errors.InvalidTriggerQuery
	}
	match := triggerQueryRegexp.FindStringSubmatch(query + " ")
	if match == nil {
		return "", "", app_errors.InvalidTriggerQuery
	}
	table := match[2]
	parts := strings.Split(table, ".")
	if strings.Trim(parts[len(parts)-1], `"`) != tableAlias {
		return "", "", app_errors.InvalidTriggerQuery
	}
	return match[1], table, nil
}
