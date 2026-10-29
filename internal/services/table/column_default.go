package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/lib/pq"
)

// metaDefaultValueKey is the column meta key holding the user-supplied default.
const metaDefaultValueKey = "defaultValue"

// defaultValueSQL converts a meta default into a safe SQL literal for the given
// database type. The second return is false when no default should be applied.
func defaultValueSQL(uidt, dt string, raw interface{}) (string, bool, error) {
	if raw == nil {
		return "", false, nil
	}
	if arr, isArr := raw.([]interface{}); isArr {
		return arrayDefaultSQL(uidt, dt, arr)
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

// arrayDefaultSQL converts a multiSelect list default into a Postgres array literal.
// List defaults are only valid for multiSelect fields.
func arrayDefaultSQL(uidt, dt string, items []interface{}) (string, bool, error) {
	if uidt != "multiSelect" || !strings.HasSuffix(strings.TrimSpace(dt), "[]") {
		return "", false, app_errors.InvalidPayload
	}
	if len(items) == 0 {
		return "", false, nil
	}
	elems := make([]string, 0, len(items))
	for _, it := range items {
		var e string
		switch v := it.(type) {
		case string:
			e = v
		case bool:
			e = strconv.FormatBool(v)
		case float64:
			e = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			return "", false, app_errors.InvalidPayload
		}
		e = strings.ReplaceAll(e, `\`, `\\`)
		e = strings.ReplaceAll(e, `"`, `\"`)
		elems = append(elems, `"`+e+`"`)
	}
	lit := "{" + strings.Join(elems, ",") + "}"
	return "'" + strings.ReplaceAll(lit, "'", "''") + "'", true, nil
}

// applyColumnDefault alters the column so its DB default matches meta.defaultValue.
// An empty/absent value drops any existing default.
func (s tableManagementService) applyColumnDefault(schemaName, tableName, columnName, uidt, dt string, meta map[string]interface{}) error {
	literal, ok, err := defaultValueSQL(uidt, dt, meta[metaDefaultValueKey])
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

// fillNewColumnDefault gives a newly created column its default in the records that already
// exist. A DB default only applies to records inserted afterwards, so without this the
// existing records would stay empty. Later changes to a default don't touch existing records.
func (s tableManagementService) fillNewColumnDefault(ctx context.Context, schemaName, tableName, columnName, uidt, dt string, meta map[string]interface{}) error {
	literal, ok, err := defaultValueSQL(uidt, dt, meta[metaDefaultValueKey])
	if err != nil || !ok {
		return err
	}
	col := pq.QuoteIdentifier(columnName)
	query := fmt.Sprintf(`UPDATE %s SET %s = %s WHERE %s IS NULL`, fmt.Sprintf(SchemaTableFormat, schemaName, tableName), col, literal, col)
	if _, err := s.repo.DB.ExecContext(ctx, query); err != nil {
		return app_errors.LogDatabaseError(err, "failed to fill existing records with the column default")
	}
	return nil
}
