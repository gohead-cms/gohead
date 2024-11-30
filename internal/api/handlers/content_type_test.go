// internal/api/handlers/type_test.go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

func TestCreateContentTypeHandler(t *testing.T) {
	logger.InitLogger("debug")
	// Load test configuration
	// Create the Gin router
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Register the handler
	router.POST("/content-types", CreateContentType)

	// Define test cases
	testCases := []struct {
		name           string
		inputData      map[string]interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid Content Type",
			inputData: map[string]interface{}{
				"name": "articles",
				"fields": []map[string]interface{}{
					{
						"name":     "title",
						"type":     "string",
						"required": true,
					},
					{
						"name":     "content",
						"type":     "richtext",
						"required": true,
					},
				},
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"name":"articles"`,
		},
		{
			name: "Missing Name Field",
			inputData: map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"name":     "title",
						"type":     "string",
						"required": true,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"error":"missing required field: name"`,
		},
		{
			name: "Empty Fields Array",
			inputData: map[string]interface{}{
				"name":   "users",
				"fields": []map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"error":"fields array cannot be empty"`,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare request body
			body, _ := json.Marshal(tc.inputData)
			req, _ := http.NewRequest(http.MethodPost, "/content-types", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the HTTP request
			router.ServeHTTP(rr, req)

			// Assert the response status
			assert.Equal(t, tc.expectedStatus, rr.Code)

			// Assert the response body contains expected data
			assert.Contains(t, rr.Body.String(), tc.expectedBody)
		})
	}
}
