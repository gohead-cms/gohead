package cmd

import (
	"log"

	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	gormlogger "gorm.io/gorm/logger"

	runner "github.com/gohead-cms/gohead/internal/agent/runners"
	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Starts the background worker for processing agent jobs.",
	Long:  `The worker connects to Redis and listens for asynchronous jobs, such as running agents based on cron schedules or webhooks.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Use the default config path, similar to your start command
		configPath, _ := cmd.Flags().GetString("config")
		runWorker(configPath)
	},
}

func init() {
	// Add the same config flag as the start command for consistency
	workerCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	rootCmd.AddCommand(workerCmd)
}

func runWorker(cfgPath string) {
	// 1. Load configuration using your LoadConfig function
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Initialize logger and database just like in start.go
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

	if _, err := database.InitDatabase(cfg.DatabaseURL, gormLogLevel); err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize database")
	}

	logger.Log.Info("Starting agent worker...")

	// 3. Create the Asynq server for consuming jobs
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Address},
		asynq.Config{
			Concurrency: 10,
			Logger:      &logger.AsynqLoggerAdapter{},
			Queues:      map[string]int{"agents": 10},
		},
	)

	// 4. Create an instance of your agent runner
	agentRunner := runner.NewAgentRunner()

	// 5. Create a new ServeMux to map task types to handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc("agent:run", agentRunner.HandleAgentJob)

	// 6. Start the server
	logger.Log.Info("Worker is ready and listening for jobs...")
	if err := srv.Run(mux); err != nil {
		logger.Log.WithError(err).Fatal("Could not run asynq server")
	}
}
