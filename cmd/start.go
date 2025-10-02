package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gohead-cms/gohead/internal/agent/triggers"
	"github.com/gohead-cms/gohead/internal/api/middleware" // <-- IMPORT aDDED
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/metrics"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/seed"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	ginlogrus "github.com/toorop/gin-logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	gormlogger "gorm.io/gorm/logger"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start GoHead API server",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")

		router, err := InitializeServer(configPath)
		if err != nil {
			// Fallback to standard log if logger fails to initialize
			log.Fatalf("Cannot initialize server: %v", err)
		}

		cfg, _ := config.LoadConfig(configPath)

		// --- Server startup with graceful shutdown ---
		srv := &http.Server{
			Addr:    ":" + cfg.ServerPort,
			Handler: router,
		}

		// Start the server in a goroutine so it doesn't block.
		go func() {
			logger.Log.Infof("Starting server on port %s", cfg.ServerPort)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Log.Fatalf("listen: %s\n", err)
			}
		}()

		// Wait for an interrupt signal to gracefully shut down the server.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logger.Log.Info("Shutting down server...")

		// The context is used to inform the server it has 5 seconds to finish
		// the requests it is currently handling.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Log.Fatal("Server forced to shutdown:", err)
		}

		logger.Log.Info("Server exiting gracefully")
	},
}

func init() {
	startCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	rootCmd.AddCommand(startCmd)
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

	// --- Initialize GraphQL Schema ---
	// This must be done after the database is initialized.
	//if err := graphql.InitializeGraphQLSchema(); err != nil {
	//	return nil, fmt.Errorf("failed to initialize GraphQL schema: %w", err)
	//}

	// --- Application Services ---
	seed.SeedRoles()
	auth.InitializeJWT(cfg.JWTSecret)
	metrics.InitMetrics()

	// --- Asynq Client Initialization ---
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.Address})
	storage.InitAsynqClient(asynqClient)
	triggers.InitAsynqClient(asynqClient)

	// --- Telemetry (Optional) ---
	if cfg.TelemetryEnabled {
		// ... (tracing setup)
	}

	// --- Gin Router Setup ---
	gin.SetMode(gin.ReleaseMode)
	if cfg.Mode == "development" {
		gin.SetMode(gin.DebugMode)
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
	// ... (your route setup remains the same)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
