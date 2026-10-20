package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"
)

// dispatcherCmd represents the dispatcher command
var dispatcherCmd = &cobra.Command{
	Use:   "dispatcher",
	Short: "Starts the background dispatcher for processing collection events.",
	Long: `The dispatcher processes collection events (e.g., item created, updated,
deleted): it queues the runs of the agents subscribed to them and the deliveries
of the webhooks.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		runDispatcher(configPath)
	},
}

func init() {
	dispatcherCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	rootCmd.AddCommand(dispatcherCmd)
}

// runDispatcher initializes and starts the dispatcher server.
func runDispatcher(cfgPath string) {
	cfg, driver, db, err := backgroundSetup(cfgPath)
	if err != nil {
		log.Fatalf("Cannot start the dispatcher: %v", err)
	}
	client := openQueueClient(cfg, driver, db)
	defer client.Close()
	mux := queue.NewMux()
	dispatcherHandlers(client, mux)
	concurrency := cfg.Queue.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	srv := newQueueServer(cfg, driver, db, concurrency, dispatcherQueues)

	ctx, stop := signalContext()
	defer stop()
	logger.Log.WithField("queue", driver).Info("Dispatcher is ready and listening for collection events...")
	if err := srv.Run(ctx, mux); err != nil {
		logger.Log.WithError(err).Fatal("Could not run the dispatcher")
	}
	logger.Log.Info("Dispatcher stopped")
}
