package tests

import (
	"errors"
	"fmt"
	"testing"

	appErrors "github.com/aptlogica/sereni-base/internal/app-errors"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestUniqueViolation(t *testing.T) {
	t.Run("reads the constraint and detail of a Postgres duplicate, even when wrapped", func(t *testing.T) {
		err := fmt.Errorf("insert failed: %w", &pq.Error{Code: "23505", Constraint: "uq_emp_1", Detail: "Key (email)=(a@x.com) already exists."})
		name, detail, ok := appErrors.UniqueViolation(err)
		assert.True(t, ok)
		assert.Equal(t, "uq_emp_1", name)
		assert.Equal(t, "Key (email)=(a@x.com) already exists.", detail)
	})

	t.Run("ignores other Postgres errors", func(t *testing.T) {
		_, _, ok := appErrors.UniqueViolation(&pq.Error{Code: "23502", Message: "null value"})
		assert.False(t, ok)
	})

	t.Run("falls back to the error text", func(t *testing.T) {
		name, detail, ok := appErrors.UniqueViolation(errors.New(`pq: duplicate key value violates unique constraint "uq_emp_2"`))
		assert.True(t, ok)
		assert.Equal(t, "uq_emp_2", name)
		assert.Empty(t, detail)
	})

	t.Run("is false for nil and unrelated errors", func(t *testing.T) {
		_, _, ok := appErrors.UniqueViolation(nil)
		assert.False(t, ok)
		_, _, ok = appErrors.UniqueViolation(errors.New("connection refused"))
		assert.False(t, ok)
	})
}

func TestPostgresMessage(t *testing.T) {
	assert.Equal(t, "syntax error", appErrors.PostgresMessage(fmt.Errorf("wrapped: %w", &pq.Error{Message: "syntax error"})))
	assert.Equal(t, "relation does not exist", appErrors.PostgresMessage(errors.New("pq: relation does not exist")))
}

func TestDetailedError(t *testing.T) {
	err := appErrors.WithMessage(appErrors.ConstraintNotFound, "Unique key k1 is gone")
	assert.ErrorIs(t, err, appErrors.ConstraintNotFound)
	assert.Equal(t, "constraint not found: Unique key k1 is gone", err.Error())

	var detailed *appErrors.DetailedError
	assert.ErrorAs(t, fmt.Errorf("outer: %w", err), &detailed)
	assert.Equal(t, "Unique key k1 is gone", detailed.Message)
}
