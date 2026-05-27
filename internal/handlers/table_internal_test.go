package handlers

import (
	"errors"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/stretchr/testify/assert"
)

func TestIsTableNotFound(t *testing.T) {
	assert.False(t, isTableNotFound(nil))
	assert.True(t, isTableNotFound(app_errors.TableNotFound))
	assert.True(t, isTableNotFound(errors.New("table not found")))
	assert.False(t, isTableNotFound(errors.New("other error")))
}
