package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Starts the background worker for processing agent jobs.",
	Long: `The worker runs queued jobs: agent runs and webhook deliveries. Jobs are
queued in Redis (queue.driver: redis) or in the database (queue.driver: database).`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		runWorker(configPath)
	},
}

func init() {
	workerCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	rootCmd.AddCommand(workerCmd)
}

func runWorker(cfgPath string) {
	cfg, driver, db, err := backgroundSetup(cfgPath)
	if err != nil {
		log.Fatalf("Cannot start the worker: %v", err)
	}
	mux := queue.NewMux()
	workerHandlers(cfg, mux)
	concurrency := cfg.Queue.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}
	srv := newQueueServer(cfg, driver, db, concurrency, workerQueues)

	ctx, stop := signalContext()
	defer stop()
	logger.Log.WithField("queue", driver).Info("Worker is ready and listening for jobs...")
	if err := srv.Run(ctx, mux); err != nil {
		logger.Log.WithError(err).Fatal("Could not run the worker")
	}
	logger.Log.Info("Worker stopped")
}
