package cmd

import (
	"context"
	"log"
	"net/http"

	"github.com/gohead-cms/gohead/internal/agent/triggers"
	"github.com/gohead-cms/gohead/internal/api/handlers"
	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/graphql"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/metrics"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/seed"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/tracing"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	ginlogrus "github.com/toorop/gin-logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	gormlogger "gorm.io/gorm/logger"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start GoHead API server",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")

		// Initialize and start the server
		router, err := InitializeServer(configPath)
		if err != nil {
			// Use the initialized logger if available, otherwise fallback to standard log
			if logger.Log != nil {
				logger.Log.Fatalf("Cannot initialize server: %v", err)
			}
			log.Fatalf("Cannot initialize server: %v", err)
		}

		// Load config just to get the port, as logger is now initialized inside InitializeServer
		cfg, _ := config.LoadConfig(configPath)
		logger.Log.Infof("Starting server on port %s", cfg.ServerPort)
		if err := router.Run(":" + cfg.ServerPort); err != nil {
			logger.Log.Fatalf("Cannot start server: %v", err)
		}
	},
}

func init() {
	startCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
}

func InitializeServer(cfgPath string) (*gin.Engine, error) {
	// --- Core Dependencies ---
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	logger.InitLogger(cfg.LogLevel)

	var gormLogLevel gormlogger.LogLevel
	switch cfg.LogLevel {
	case "debug":
		gormLogLevel = gormlogger.Info
	case "info", "warn", "warning":
		gormLogLevel = gormlogger.Warn
	case "error":
		gormLogLevel = gormlogger.Error
	default:
		gormLogLevel = gormlogger.Silent
	}

	db, err := database.InitDatabase(cfg.DatabaseURL, gormLogLevel)
	if err != nil {
		return nil, err
	}
	if err := migrations.MigrateDatabase(db); err != nil {
		return nil, err
	}

	// Initialize the GraphQL Schema
	if err := graphql.InitializeGraphQLSchema(); err != nil {
		logger.Log.WithError(err).Error("Failed to initialize GraphQL schema")
		return nil, err
	}

	// --- Application Services ---
	seed.SeedRoles()
	auth.InitializeJWT(cfg.JWTSecret)
	metrics.InitMetrics()

	// --- Asynq Client Initialization for Producers ---
	// The API server acts as a producer, enqueuing jobs for workers.
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.Address})
	storage.InitAsynqClient(asynqClient)
	triggers.InitAsynqClient(asynqClient)

	// --- Telemetry (Optional) ---
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

	// --- Gin Router Setup ---
	switch cfg.Mode {
	case "development":
		gin.SetMode(gin.DebugMode)
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(ginlogrus.Logger(logger.Log), gin.Recovery())
	router.Use(middleware.CORSMiddleware(cfg))
	router.Use(middleware.MetricsMiddleware())
	router.Use(otelgin.Middleware("gohead"))
	router.Use(middleware.ResponseWrapper())

	// --- Routes ---
	setupRoutes(router)

	return router, nil
}

func setupRoutes(router *gin.Engine) {
	// Monitoring & Healthcheck
	router.GET("/_metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/_health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public routes
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", handlers.Register)
		authRoutes.POST("/login", handlers.Login)
	}

	// Agent Webhook Trigger (Public, authenticates with a token)
	router.POST("/agents/webhook/:id", handlers.HandleWebhook)

	// ADMIN routes (schema/definition)
	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		// Collections
		admin.POST("/collections", handlers.CreateCollection)
		admin.GET("/collections", handlers.GetCollections)
		admin.GET("/collections/:name", handlers.GetCollection)
		admin.PUT("/collections/:name", handlers.UpdateCollection)
		admin.DELETE("/collections/:name", handlers.DeleteCollection)

		// Singletons
		admin.POST("/singleton", handlers.CreateOrUpdateSingleton)
		admin.GET("/singleton/:name", handlers.GetSingleton)
		admin.PUT("/singleton/:name", handlers.CreateOrUpdateSingleton)
		admin.DELETE("/singleton/:name", handlers.DeleteSingleton)

		// Components
		admin.POST("/components", handlers.CreateComponent)
		admin.GET("/components/:name", handlers.GetComponent)
		admin.PUT("/components/:name", handlers.UpdateComponent)
		admin.DELETE("/components/:name", handlers.DeleteComponent)

		// Agents
		agents := admin.Group("/agents")
		{
			agents.POST("/", handlers.CreateAgent)
			agents.GET("", handlers.GetAgents)
			agents.GET("/:name", handlers.GetAgent)
			agents.PUT("/:name", handlers.UpdateAgent)
			agents.DELETE("/:name", handlers.DeleteAgent)
		}
	}

	// CONTENT routes (actual data/items)
	content := router.Group("/api")
	content.Use(middleware.AuthMiddleware())
	{
		content.POST("/graphql", handlers.GraphQLHandler)

		// Dynamic Handlers
		content.Any("/collections/:collection", handlers.DynamicCollectionHandler)
		content.Any("/collections/:collection/:id", handlers.DynamicCollectionHandler)
		content.GET("/singleton/:name", handlers.GetSingleItem)
		content.POST("/singleton/:name", handlers.CreateOrUpdateSingletonItem)
		content.PUT("/singleton/:name", handlers.CreateOrUpdateSingletonItem)
	}
}
