package services

import (
	"fmt"
	"strconv"
	"strings"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
)

// metaDefaultValueKey is the column meta key holding the user-supplied default.
const metaDefaultValueKey = "defaultValue"

// defaultValueSQL converts a meta default into a safe SQL literal for the given
// database type. The second return is false when no default should be applied.
func defaultValueSQL(dt string, raw interface{}) (string, bool, error) {
	if raw == nil {
		return "", false, nil
	}
	var str string
	switch v := raw.(type) {
	case string:
		str = strings.TrimSpace(v)
	case bool:
		str = strconv.FormatBool(v)
	case float64:
		str = strconv.FormatFloat(v, 'f', -1, 64)
	case int, int64:
		str = fmt.Sprint(v)
	default:
		return "", false, app_errors.InvalidPayload
	}
	if str == "" {
		return "", false, nil
	}

	switch strings.ToUpper(dt) {
	case "BOOLEAN", "BOOL":
		b, err := strconv.ParseBool(str)
		if err != nil {
			return "", false, app_errors.InvalidPayload
		}
		return strconv.FormatBool(b), true, nil
	case "INTEGER", "INT", "BIGINT", "SMALLINT", "NUMERIC", "DECIMAL", "REAL", "DOUBLE PRECISION", "FLOAT":
		if _, err := strconv.ParseFloat(str, 64); err != nil {
			return "", false, app_errors.InvalidPayload
		}
		return str, true, nil
	default:
		return "'" + strings.ReplaceAll(str, "'", "''") + "'", true, nil
	}
}

// applyColumnDefault alters the column so its DB default matches meta.defaultValue.
// An empty/absent value drops any existing default.
func (s tableManagementService) applyColumnDefault(schemaName, tableName, columnName, dt string, meta map[string]interface{}) error {
	literal, ok, err := defaultValueSQL(dt, meta[metaDefaultValueKey])
	if err != nil {
		return err
	}

	modify := dbModels.ModifyColumnRequest{ColumnName: fmt.Sprintf(QuotedColumnFormat, columnName)}
	if ok {
		modify.SetDefault = &literal
	} else {
		modify.DropDefault = true
	}

	req := dbModels.AlterTableRequest{Action: "modify_column", Data: modify}
	if err := s.repo.TableService.AlterTable(fmt.Sprintf(SchemaTableFormat, schemaName, tableName), req); err != nil {
		return app_errors.LogDatabaseError(err, "failed to set column default in DB")
	}
	return nil
}
