package tests

import (
	"net/http"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/stretchr/testify/assert"
)

func TestTableConstraintCodes(t *testing.T) {
	errorStatus := map[constants.ResponseCode]int{
		constants.TableConstraintError.ConstraintNotFound:     http.StatusNotFound,
		constants.TableConstraintError.TitleRequired:          http.StatusBadRequest,
		constants.TableConstraintError.TitleTooLong:           http.StatusBadRequest,
		constants.TableConstraintError.ColumnNotFound:         http.StatusBadRequest,
		constants.TableConstraintError.ColumnNotAllowed:       http.StatusBadRequest,
		constants.TableConstraintError.RepeatedColumn:         http.StatusBadRequest,
		constants.TableConstraintError.AlreadyExists:          http.StatusConflict,
		constants.TableConstraintError.DuplicateValues:        http.StatusConflict,
		constants.TableConstraintError.ConstraintFailed:       http.StatusBadRequest,
		constants.TableConstraintError.ColumnUsedInConstraint: http.StatusConflict,
		constants.TableConstraintError.UniqueKeyViolation:     http.StatusConflict,
		constants.TableConstraintError.TypeRequired:           http.StatusBadRequest,
	}
	for code, status := range errorStatus {
		meta, ok := constants.ErrorCodes[code]
		assert.True(t, ok, "%s is not registered", code)
		assert.Equal(t, status, meta.HTTPStatus, code)
	}

	successStatus := map[constants.ResponseCode]int{
		constants.TableConstraintSuccess.ConstraintCreated:  http.StatusCreated,
		constants.TableConstraintSuccess.ConstraintsFetched: http.StatusOK,
		constants.TableConstraintSuccess.ConstraintUpdated:  http.StatusOK,
		constants.TableConstraintSuccess.ConstraintDeleted:  http.StatusOK,
		constants.TableConstraintSuccess.ConstraintChecked:  http.StatusOK,
	}
	for code, status := range successStatus {
		meta, ok := constants.SuccessCodes[code]
		assert.True(t, ok, "%s is not registered", code)
		assert.Equal(t, status, meta.HTTPStatus, code)
	}
}

func TestTableConstraintErrorMapping(t *testing.T) {
	cases := map[error]constants.ResponseCode{
		app_errors.ConstraintNotFound:         constants.TableConstraintError.ConstraintNotFound,
		app_errors.ConstraintColumnNotFound:   constants.TableConstraintError.ColumnNotFound,
		app_errors.ConstraintColumnNotAllowed: constants.TableConstraintError.ColumnNotAllowed,
		app_errors.ConstraintRepeatedColumn:   constants.TableConstraintError.RepeatedColumn,
		app_errors.ConstraintAlreadyExists:    constants.TableConstraintError.AlreadyExists,
		app_errors.ConstraintDuplicateValues:  constants.TableConstraintError.DuplicateValues,
		app_errors.ConstraintFailed:           constants.TableConstraintError.ConstraintFailed,
		app_errors.ColumnUsedInConstraint:     constants.TableConstraintError.ColumnUsedInConstraint,
		app_errors.UniqueKeyViolation:         constants.TableConstraintError.UniqueKeyViolation,
	}
	for err, code := range cases {
		assert.Equal(t, code, constants.MapError(err), err.Error())
		assert.Equal(t, code, constants.MapError(app_errors.WithMessage(err, "details")), "with a message: "+err.Error())
	}
}
