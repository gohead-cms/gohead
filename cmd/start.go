package cmd

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	ginlogrus "github.com/toorop/gin-logrus"
	"gitlab.com/sudo.bngz/gohead/internal/api/handlers"
	"gitlab.com/sudo.bngz/gohead/internal/api/middleware"
	"gitlab.com/sudo.bngz/gohead/pkg/auth"
	"gitlab.com/sudo.bngz/gohead/pkg/config"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
	"gitlab.com/sudo.bngz/gohead/pkg/metrics"
	"gitlab.com/sudo.bngz/gohead/pkg/migrations"
	"gitlab.com/sudo.bngz/gohead/pkg/seed"
	"gitlab.com/sudo.bngz/gohead/pkg/tracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	gormlogger "gorm.io/gorm/logger"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start GoHead server",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")

		// Load configuration
		cfg, _ := config.LoadConfig(configPath)

		// Initialize and start the server
		router, err := InitializeServer(configPath)
		if err != nil {
			logger.Log.Errorf("Cannot start server on port %s: %v", cfg.ServerPort, err)
			return
		}

		logger.Log.Infof("Starting server on port %s", cfg.ServerPort)
		router.Run(":" + cfg.ServerPort)
	},
}

func init() {
	startCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
}

func InitializeServer(cfgPath string) (*gin.Engine, error) {
	// Load configuration
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	// Initialize logger
	logger.InitLogger(cfg.LogLevel)

	// Map to GORM log levels
	var gormLogLevel gormlogger.LogLevel
	switch cfg.LogLevel {
	case "debug":
		gormLogLevel = gormlogger.Info
	case "info":
		gormLogLevel = gormlogger.Warn
	case "warn", "warning":
		gormLogLevel = gormlogger.Warn
	case "error":
		gormLogLevel = gormlogger.Error
	default:
		gormLogLevel = gormlogger.Silent
	}

	// Initialize database
	db, err := database.InitDatabase(cfg.DatabaseURL, gormLogLevel)
	if err != nil {
		return nil, err
	}

	// Migrate
	if err := migrations.MigrateDatabase(db); err != nil {
		return nil, err
	}

	// Seed roles, init JWT, metrics
	seed.SeedRoles()
	auth.InitializeJWT(cfg.JWTSecret)
	metrics.InitMetrics()

	// Tracing
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

	// Gin mode
	switch cfg.Mode {
	case "development":
		gin.SetMode(gin.DebugMode)
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		log.Printf("Unknown Gin mode '%s', defaulting to 'release'", cfg.Mode)
		gin.SetMode(gin.ReleaseMode)
	}

	// Create the router
	router := gin.New()
	router.Use(ginlogrus.Logger(logger.Log))
	router.Use(gin.Recovery())
	router.Use(middleware.MetricsMiddleware())
	router.Use(otelgin.Middleware("gohead"))
	router.Use(middleware.ResponseWrapper())

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

	// ADMIN routes (schema/definition)
	// Only admin can manage content definitions (e.g., collections & single-types)
	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.AdminOnly())
	{
		admin.POST("/register", handlers.Register)

		// Collections definitions
		admin.POST("/collections", handlers.CreateCollection)
		admin.GET("/collections/:name", handlers.GetCollection)
		admin.PUT("/collections/:name", handlers.UpdateCollection)
		admin.DELETE("/collections/:name", handlers.DeleteCollection)

		// Single Types definitions
		admin.POST("/single-types", handlers.CreateOrUpdateSingleType)
		admin.GET("/single-types/:name", handlers.GetSingleType)
		admin.PUT("/single-types/:name", handlers.CreateOrUpdateSingleType)
		admin.DELETE("/single-types/:name", handlers.DeleteSingleType)
	}

	// CONTENT routes (actual data/items)
	content := router.Group("/api")
	content.Use(middleware.AuthMiddleware())
	{
		content.Any("/:collection", handlers.DynamicCollectionHandler)
		content.Any("/:collection/:id", handlers.DynamicCollectionHandler)

		content.GET("/single-types/:name", handlers.GetSingleItem)
		content.POST("/single-types/:name", handlers.CreateOrUpdateSingleTypeItem)
		content.PUT("/single-types/:name", handlers.CreateOrUpdateSingleTypeItem)
	}

	return router, nil
}

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, _ := config.LoadConfig(*configPath)

	// Initialize server
	router, err := InitializeServer(*configPath)
	if err != nil {
		logger.Log.Errorf("Cannot start server on port %s: %v", cfg.ServerPort, err)
		return
	}

	// Start the server
	logger.Log.Infof("Starting server on port %s", cfg.ServerPort)
	router.Run(":" + cfg.ServerPort)
}
