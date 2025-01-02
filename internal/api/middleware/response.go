package middleware

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ResponseWrapper() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request
		c.Next()

		// Get the response data and status
		response, exists := c.Get("response")
		if !exists {
			return // If no response is set, do nothing
		}
		status, _ := c.Get("status")
		statusCode := http.StatusOK
		if status != nil {
			statusCode = status.(int)
		}

		// Normalize the response
		var formattedResponse gin.H
		if statusCode >= 400 { // Error response
			formattedResponse = gin.H{
				"error": gin.H{
					"status":  statusCode,
					"name":    getErrorName(statusCode),
					"message": response,
					"details": nil,
				},
			}
		} else { // Success response
			formattedResponse = gin.H{
				"data": response,
				"meta": nil, // Add metadata if needed
			}
		}

		// Write the formatted response
		c.JSON(statusCode, formattedResponse)
	}
}

type responseRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func getErrorName(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "ValidationError"
	case http.StatusUnauthorized:
		return "UnauthorizedError"
	case http.StatusForbidden:
		return "ForbiddenError"
	case http.StatusNotFound:
		return "NotFoundError"
	default:
		return "ServerError"
	}
}
