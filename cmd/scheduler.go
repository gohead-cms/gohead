package cmd

import (
	"log"
	"time"

	"github.com/spf13/cobra"

	"github.com/gohead-cms/gohead/pkg/logger"
)

// reloadInterval is how often the scheduler re-reads cron agents.
const reloadInterval = time.Minute

var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Starts the background scheduler for cron-based agent jobs.",
	Long: `The scheduler is a long-running process that connects to the database to find
agents with cron triggers and enqueues jobs for them at the specified times.
It is recommended to run only one instance of the scheduler in a production environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		runSchedulerCmd(configPath)
	},
}

func init() {
	schedulerCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	rootCmd.AddCommand(schedulerCmd)
}

func runSchedulerCmd(cfgPath string) {
	cfg, driver, db, err := backgroundSetup(cfgPath)
	if err != nil {
		log.Fatalf("Cannot start the scheduler: %v", err)
	}
	client := openQueueClient(cfg, driver, db)
	defer client.Close()

	ctx, stop := signalContext()
	defer stop()
	logger.Log.WithField("queue", driver).Info("Scheduler started successfully. Waiting for jobs...")
	runScheduler(ctx, client)
	logger.Log.Info("Shutting down scheduler...")
}
