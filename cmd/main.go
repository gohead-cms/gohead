// cmd/main.go
package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ginlogrus "github.com/toorop/gin-logrus"
	"gitlab.com/sudo.bngz/gohead/internal/api/handlers"
	"gitlab.com/sudo.bngz/gohead/internal/api/middleware"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/metrics"
	"gitlab.com/sudo.bngz/gohead/pkg/tracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	// Initialize the logger
	logger.InitLogger()

	// Initialize metrics
	metrics.InitMetrics()

	// Initialize tracing
	tracerProvider, err := tracing.InitTracer()
	if err != nil {
		logger.Log.Fatal("Failed to initialize tracer:", err)
	}
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			logger.Log.Error("Error shutting down tracer provider:", err)
		}
	}()

	// Initialize the database
	if err := database.InitDatabase(); err != nil {
		logger.Log.Fatal("Failed to connect to database!", err)
	}

	router := gin.New()
	router.Use(ginlogrus.Logger(logger.Log), gin.Recovery(), middleware.MetricsMiddleware())
	router.Use(otelgin.Middleware("gohead"))

	// Monitoring
	router.GET("/_metrics", gin.WrapH(promhttp.Handler()))

	// healthcheck
	router.GET("/_health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public routes
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", handlers.Register)
		authRoutes.POST("/login", handlers.Login)
	}

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/content-types", middleware.AuthorizeRole("admin"), handlers.CreateContentType)
		protected.Any("/:contentType", handlers.DynamicContentHandler)
		protected.Any("/:contentType/:id", handlers.DynamicContentHandler)
	}

	// Start the server
	logger.Log.Info("Starting server on port 8080")
	router.Run(":8080")
}
