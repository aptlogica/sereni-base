package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/handlers"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type testRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func TestNewHandlerUtil(t *testing.T) {
	hu := handlers.NewHandlerUtil()
	assert.NotNil(t, hu)
}

func TestGetSchemaFromContext(t *testing.T) {
	hu := handlers.NewHandlerUtil()

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedSchema string
		expectedOk     bool
	}{
		{
			name: "schema exists in context",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "test_schema")
			},
			expectedSchema: "test_schema",
			expectedOk:     true,
		},
		{
			name: "schema not in context",
			setupContext: func(c *gin.Context) {
				// Don't set schema
			},
			expectedSchema: "",
			expectedOk:     false,
		},
		{
			name: "schema is empty string",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "")
			},
			expectedSchema: "",
			expectedOk:     false,
		},
		{
			name: "schema is non-string type",
			setupContext: func(c *gin.Context) {
				c.Set("schema", 123)
			},
			expectedSchema: "",
			expectedOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupContext(c)

			schema, ok := hu.GetSchemaFromContext(c)
			assert.Equal(t, tt.expectedSchema, schema)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	hu := handlers.NewHandlerUtil()

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedUserID string
		expectedOk     bool
	}{
		{
			name: "user_id exists in context",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
			},
			expectedUserID: "user123",
			expectedOk:     true,
		},
		{
			name: "user_id not in context",
			setupContext: func(c *gin.Context) {
				// Don't set user_id
			},
			expectedUserID: "",
			expectedOk:     false,
		},
		{
			name: "user_id is empty string",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "")
			},
			expectedUserID: "",
			expectedOk:     false,
		},
		{
			name: "user_id is non-string type",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", 456)
			},
			expectedUserID: "",
			expectedOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupContext(c)

			userID, ok := hu.GetUserIDFromContext(c)
			assert.Equal(t, tt.expectedUserID, userID)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}

func TestGetBothFromContext(t *testing.T) {
	hu := handlers.NewHandlerUtil()

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedSchema string
		expectedUserID string
		expectedOk     bool
	}{
		{
			name: "both schema and user_id exist",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "test_schema")
				c.Set("user_id", "user123")
			},
			expectedSchema: "test_schema",
			expectedUserID: "user123",
			expectedOk:     true,
		},
		{
			name: "only schema exists",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "test_schema")
			},
			expectedSchema: "",
			expectedUserID: "",
			expectedOk:     false,
		},
		{
			name: "only user_id exists",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
			},
			expectedSchema: "",
			expectedUserID: "",
			expectedOk:     false,
		},
		{
			name: "neither exists",
			setupContext: func(c *gin.Context) {
				// Don't set anything
			},
			expectedSchema: "",
			expectedUserID: "",
			expectedOk:     false,
		},
		{
			name: "schema is empty",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "")
				c.Set("user_id", "user123")
			},
			expectedSchema: "",
			expectedUserID: "user123",
			expectedOk:     false,
		},
		{
			name: "user_id is empty",
			setupContext: func(c *gin.Context) {
				c.Set("schema", "test_schema")
				c.Set("user_id", "")
			},
			expectedSchema: "test_schema",
			expectedUserID: "",
			expectedOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupContext(c)

			schema, userID, ok := hu.GetBothFromContext(c)
			assert.Equal(t, tt.expectedSchema, schema)
			assert.Equal(t, tt.expectedUserID, userID)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}

func TestSendSuccessResponse(t *testing.T) {
	hu := handlers.NewHandlerUtil()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"message": "success"}
	hu.SendSuccessResponse(c, responseConst.AuthSuccess.UserLogin, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, true, response["success"])
}

func TestSendErrorResponse(t *testing.T) {
	hu := handlers.NewHandlerUtil()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := errors.New("test error")
	hu.SendErrorResponse(c, err)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestSendInvalidPayloadError(t *testing.T) {
	hu := handlers.NewHandlerUtil()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	hu.SendInvalidPayloadError(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
}

func TestSendUnauthorizedError(t *testing.T) {
	hu := handlers.NewHandlerUtil()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	hu.SendUnauthorizedError(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, false, response["success"])
}

func TestBindAndValidateJSON_Success(t *testing.T) {
	hu := handlers.NewHandlerUtil()

	req := testRequest{
		Name:  "Test User",
		Email: "test@example.com",
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	var receivedReq testRequest
	result := hu.BindAndValidateJSON(c, &receivedReq, func(fe validator.FieldError) responseConst.ResponseCode {
		return responseConst.Error.ValidationFailed
	})

	assert.True(t, result)
	assert.Equal(t, req.Name, receivedReq.Name)
	assert.Equal(t, req.Email, receivedReq.Email)
}

func TestBindAndValidateJSON_InvalidJSON(t *testing.T) {
	hu := handlers.NewHandlerUtil()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	var req testRequest
	result := hu.BindAndValidateJSON(c, &req, func(fe validator.FieldError) responseConst.ResponseCode {
		return responseConst.Error.ValidationFailed
	})

	assert.False(t, result)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBindAndValidateJSON_ValidationError(t *testing.T) {
	hu := handlers.NewHandlerUtil()

	req := testRequest{
		Name:  "Test User",
		Email: "invalid-email", // Invalid email format
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	var receivedReq testRequest
	result := hu.BindAndValidateJSON(c, &receivedReq, func(fe validator.FieldError) responseConst.ResponseCode {
		return responseConst.Error.ValidationFailed
	})

	assert.False(t, result)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBindAndValidateJSON_MissingRequiredField(t *testing.T) {
	hu := handlers.NewHandlerUtil()

	req := map[string]string{
		"name": "Test User",
		// Email is missing
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	var receivedReq testRequest
	result := hu.BindAndValidateJSON(c, &receivedReq, func(fe validator.FieldError) responseConst.ResponseCode {
		return responseConst.Error.ValidationFailed
	})

	assert.False(t, result)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
