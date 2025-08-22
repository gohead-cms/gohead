// internal/api/middleware/metrics.go
package middleware

import (
	"strconv"
	"time"

	"github.com/gohead-cms/gohead/pkg/metrics"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)
		statusCode := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		// Increment request count metric
		metrics.RequestCount.WithLabelValues(statusCode, method).Inc()

		// Record request duration metric
		metrics.RequestDuration.WithLabelValues(statusCode, method).Observe(duration.Seconds())
	}
}
