package middleware

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ResponseWrapper() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Retrieve the response and status from context
		response, exists := c.Get("response")
		if !exists {
			return
		}

		status, _ := c.Get("status")
		statusCode := http.StatusOK
		if status != nil {
			statusCode = status.(int)
		}

		// Retrieve the meta from context
		meta, _ := c.Get("meta")

		// Format the response
		var formattedResponse gin.H
		if statusCode >= 400 { // Error response
			details, _ := c.Get("details")
			formattedResponse = gin.H{
				"error": gin.H{
					"status":  statusCode,
					"name":    getErrorName(statusCode),
					"message": response,
					"details": details,
				},
			}
		} else { // Success response
			formattedResponse = gin.H{
				"data": response,
				"meta": meta,
			}
		}

		// Send the JSON response
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
