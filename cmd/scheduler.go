package cmd

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gohead-cms/gohead/internal/agent/triggers"
	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	gormlogger "gorm.io/gorm/logger"
)

// schedulerCmd represents the scheduler command
var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Starts the background scheduler for cron-based agent jobs.",
	Long: `The scheduler is a long-running process that connects to the database to find
agents with cron triggers and enqueues jobs for them at the specified times.
It is recommended to run only one instance of the scheduler in a production environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		runScheduler(configPath)
	},
}

func init() {
	schedulerCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	rootCmd.AddCommand(schedulerCmd)
}

// runScheduler initializes all necessary components and starts the cron scheduler.
func runScheduler(cfgPath string) {
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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

	if _, err := database.InitDatabase(cfg.DatabaseURL, gormLogLevel); err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize database")
	}

	// The scheduler needs an Asynq client to enqueue jobs.
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.Address})
	triggers.InitAsynqClient(asynqClient)

	// Start the actual cron scheduler.
	triggers.StartScheduler()

	logger.Log.Info("Scheduler started successfully. Waiting for jobs...")

	// Wait indefinitely until a termination signal is received.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down scheduler...")
}
