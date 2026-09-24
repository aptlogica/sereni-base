// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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

// triggerQueryRegexp captures the trigger name and target table of a CREATE TRIGGER query
var triggerQueryRegexp = regexp.MustCompile(`(?is)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?TRIGGER\s+("?\w+"?)\s.*?\sON\s+((?:"?\w+"?\.)?"?\w+"?)\s`)

type automationService struct {
	repo         *pkg.DatabaseService
	modelService interfaces.ModelService
}

func NewAutomationService(repo *pkg.DatabaseService, modelService interfaces.ModelService) interfaces.AutomationService {
	return &automationService{repo: repo, modelService: modelService}
}

// Create stores the automation; for a trigger it also runs the query stored in context
func (s *automationService) Create(ctx context.Context, schemaName string, req dto.CreateAutomationRequest) (tenant.Automation, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return tenant.Automation{}, err
	}

	query := strings.TrimSuffix(strings.TrimSpace(req.Context), ";")
	if req.Type == tenant.AutomationTypeFunction {
		functionSQL, _, rest, err := splitTriggerContext(query)
		if err != nil || functionSQL == "" || rest != "" {
			return tenant.Automation{}, app_errors.InvalidFunctionQuery
		}
		if err := s.execInTx(functionSQL); err != nil {
			return tenant.Automation{}, err
		}
	}
	if req.Type == tenant.AutomationTypeTrigger {
		functionSQL, _, triggerSQL, err := splitTriggerContext(query)
		if err != nil {
			return tenant.Automation{}, err
		}
		if _, _, err := parseTriggerQuery(triggerSQL, model.Alias); err != nil {
			return tenant.Automation{}, err
		}
		// Function and trigger are created together, or not at all
		if err := s.execInTx(functionSQL, triggerSQL); err != nil {
			return tenant.Automation{}, err
		}
	}

	automation := tenant.Automation{
		ID:      uuid.New(),
		ModelID: req.ModelID,
		Title:   req.Title,
		Type:    req.Type,
		Context: query,
	}
	tableName := tenant.Automation{}.TableName(schemaName)
	inserted, err := s.repo.TableService.CreateRecord(tableName, map[string]interface{}{
		"id":       automation.ID,
		"model_id": automation.ModelID,
		"title":    automation.Title,
		"type":     automation.Type,
		"context":  automation.Context,
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

// List returns automations, optionally only those of one table and/or one type
func (s *automationService) List(ctx context.Context, schemaName, modelID, automationType string) ([]tenant.Automation, error) {
	filters := []dbModels.QueryFilter{}
	if modelID != "" {
		filters = append(filters, dbModels.QueryFilter{Column: "model_id", Operator: "eq", Value: modelID})
	}
	if automationType != "" {
		filters = append(filters, dbModels.QueryFilter{Column: "type", Operator: "eq", Value: automationType})
	}
	return s.fetch(schemaName, dbModels.QueryParams{Filters: filters})
}

// Delete removes the automation; for a trigger it also drops the trigger from the table
func (s *automationService) Delete(ctx context.Context, schemaName, id string) error {
	limit := 1
	automations, err := s.fetch(schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{{Column: "id", Operator: "eq", Value: id}},
		Limit:   &limit,
	})
	if err != nil {
		return err
	}
	if len(automations) == 0 {
		return app_errors.AutomationNotFound
	}
	automation := automations[0]

	if automation.Type == tenant.AutomationTypeFunction {
		if err := s.dropFunction(automation.Context); err != nil {
			return err
		}
	}

	if automation.Type == tenant.AutomationTypeTrigger {
		model, err := s.modelService.GetModelByID(ctx, schemaName, automation.ModelID)
		if err != nil {
			return err
		}
		if err := s.dropTrigger(automation.Context, model.Alias); err != nil {
			return err
		}
	}

	if err := s.repo.TableService.DeleteRecord(tenant.Automation{}.TableName(schemaName), id); err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete automation")
	}
	return nil
}

// dropTrigger drops the trigger in context and, if context created one, its function
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
