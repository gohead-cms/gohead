package main

import (
	"context"
	"flag"
	"log"
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
	"gitlab.com/sudo.bngz/gohead/pkg/migrations"
	"gitlab.com/sudo.bngz/gohead/pkg/tracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// InitializeServer initializes the Gin server and all dependencies
func InitializeServer(cfgPath string) (*gin.Engine, error) {
	// Load configuration
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	// Initialize the logger with the configured log level
	logger.InitLogger(cfg.LogLevel)

	// Initialize metrics
	metrics.InitMetrics()

	if cfg.TelemetryEnabled {
		tracerProvider, err := tracing.InitTracer()
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := tracerProvider.Shutdown(context.Background()); err != nil {
				logger.Log.Error("Error shutting down tracer provider:", err)
			}
		}()
	}

	// Initialize the database
	db, err := database.InitDatabase(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	// Apply migrations
	if err := migrations.MigrateDatabase(db); err != nil {
		return nil, err
	}

	// Initialize JWT with the secret from config
	auth.InitializeJWT(cfg.JWTSecret)

	// Create the router
	router := gin.New()
	router.Use(ginlogrus.Logger(logger.Log))
	router.Use(gin.Recovery())
	router.Use(middleware.MetricsMiddleware())
	router.Use(otelgin.Middleware("gohead"))

	// Monitoring
	router.GET("/_metrics", gin.WrapH(promhttp.Handler()))

	// Healthcheck
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

	return router, nil
}

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Initialize the server
	router, err := InitializeServer(*configPath)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Start the server
	cfg, _ := config.LoadConfig(*configPath) // Reload config to get ServerPort
	logger.Log.Infof("Starting server on port %s", cfg.ServerPort)
	router.Run(":" + cfg.ServerPort)
}
