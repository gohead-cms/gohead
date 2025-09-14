package cmd

import (
	"log"

	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	gormlogger "gorm.io/gorm/logger"

	"github.com/gohead-cms/gohead/internal/agent/dispatcher"
	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
)

// dispatcherCmd represents the dispatcher command
var dispatcherCmd = &cobra.Command{
	Use:   "dispatcher",
	Short: "Starts the background dispatcher for processing collection events.",
	Long: `The dispatcher connects to Redis and listens for generic collection events 
(e.g., item created, updated, deleted). It then finds agents subscribed to these 
events and enqueues specific jobs for the agent worker to process.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		runDispatcher(configPath)
	},
}

func init() {
	dispatcherCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	rootCmd.AddCommand(dispatcherCmd)
}

// initializeDispatcher sets up all the components needed for the dispatcher to run.
func initializeDispatcher(cfgPath string) (*asynq.Server, *asynq.ServeMux, error) {
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	redisOpt := asynq.RedisClientOpt{Addr: cfg.Redis.Address}

	// The Asynq server for listening to events.
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 5,
			Logger:      &logger.AsynqLoggerAdapter{},
		},
	)

	// An Asynq client for the dispatcher to enqueue new jobs.
	client := asynq.NewClient(redisOpt)

	// Create the handler (dispatcher), passing the client it needs.
	eventDispatcher := dispatcher.NewEventDispatcher(client)

	mux := asynq.NewServeMux()
	mux.HandleFunc(events.TaskTypeCollectionEvent, eventDispatcher.HandleCollectionEvent)

	return srv, mux, nil
}

// runDispatcher initializes and starts the dispatcher server.
func runDispatcher(cfgPath string) {
	srv, mux, err := initializeDispatcher(cfgPath)
	if err != nil {
		log.Fatalf("Failed to initialize dispatcher: %v", err)
	}

	logger.Log.Info("Starting event dispatcher...")
	logger.Log.Info("Dispatcher is ready and listening for collection events...")

	if err := srv.Run(mux); err != nil {
		logger.Log.WithError(err).Fatal("Could not run dispatcher server")
	}
}
