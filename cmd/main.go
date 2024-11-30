// cmd/main.go
package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ginlogrus "github.com/toorop/gin-logrus"
	"gitlab.com/sudo.bngz/gohead/internal/api/handlers"
	"gitlab.com/sudo.bngz/gohead/internal/api/middleware"
	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/config"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/metrics"
	"gitlab.com/sudo.bngz/gohead/pkg/tracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		panic(err)
	}
	// Initialize the logger with the configured log level
	logger.InitLogger(cfg.LogLevel)

	// Initialize metrics
	metrics.InitMetrics()

	if cfg.TelemetryEnabled {
		tracerProvider, err := tracing.InitTracer()
		if err != nil {
			logger.Log.Fatal("Failed to initialize tracer:", err)
		}
		defer func() {
			if err := tracerProvider.Shutdown(context.Background()); err != nil {
				logger.Log.Error("Error shutting down tracer provider:", err)
			}
		}()
	}

	// Initialize the database with the configured database URL
	if err := database.InitDatabase(cfg.DatabaseURL); err != nil {
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
	// Initialize JWT with the secret from config
	auth.InitializeJWT(cfg.JWTSecret)

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/content-types", middleware.AuthorizeRole("admin"), handlers.CreateContentType)
		protected.Any("/:contentType", handlers.DynamicContentHandler)
		protected.Any("/:contentType/:id", handlers.DynamicContentHandler)
	}

	// Start the server
	logger.Log.Infof("Starting server on port %s", cfg.ServerPort)
	router.Run(":" + cfg.ServerPort)
}
