package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appErrors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/utils/response"
)

func TestCheckAndSendError_DetailedError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("sends the mapped code with the error's own message", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		response.CheckAndSendError(c, appErrors.WithMessage(appErrors.UniqueKeyViolation, "A record with Email = a@x.com already exists"))

		var body response.StandardResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, "TCN_1012", body.Error.Code)
		assert.Equal(t, "A record with Email = a@x.com already exists", body.Error.Message)
	})

	t.Run("still sends the static message for plain errors", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		response.CheckAndSendError(c, appErrors.ConstraintNotFound)

		var body response.StandardResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "Unique key not found", body.Error.Message)
	})
}
