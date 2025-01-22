package middleware

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

type responseRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func ResponseWrapper() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Retrieve the response and status from context
		response, exists := c.Get("response")
		if !exists {
			// No response to send (e.g., static file served directly), do nothing
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
		if statusCode >= 400 {
			// Error response
			details, _ := c.Get("details")
			c.JSON(statusCode, gin.H{
				"error": gin.H{
					"status":  statusCode,
					"name":    getErrorName(statusCode),
					"message": response,
					"details": details,
				},
			})
		} else {
			// Success response
			formattedResponse := gin.H{
				"data": response,
			}
			// Only add "meta" if it’s not nil
			if meta != nil {
				formattedResponse["meta"] = meta
			}

			c.JSON(statusCode, formattedResponse)
		}
	}
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
