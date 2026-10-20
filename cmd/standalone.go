package cmd

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"
)

var standaloneCmd = &cobra.Command{
	Use:   "standalone",
	Short: "Run GoHead in one process: API, admin UI and background jobs",
	Long: `Runs the API, the admin UI (when the binary includes it), the worker, the
dispatcher and the scheduler in one process, without Redis: jobs are queued in
the database.

Unless configured otherwise, data goes to the --data directory: the SQLite
database (gohead.db), uploaded files (uploads/) and a generated JWT secret
(jwt_secret). Made for small machines such as a Raspberry Pi; run a single
standalone process per database.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		dataDir, _ := cmd.Flags().GetString("data")
		runServer(configPath, ServerOptions{Standalone: true, DataDir: dataDir})
	},
}

func init() {
	standaloneCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file (optional)")
	standaloneCmd.Flags().StringP("data", "d", "data", "Directory of the SQLite database, uploads and generated secrets")
	rootCmd.AddCommand(standaloneCmd)
}

// applyDataDir points the database, the uploads and the JWT secret file to
// dir, for the settings not configured explicitly. It returns the file that
// keeps a generated JWT secret.
func applyDataDir(cfg *config.Config, dir string) (string, error) {
	if dir == "" {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("data directory: %w", err)
	}
	if !explicitlySet("database_url") {
		cfg.DatabaseURL = "sqlite://" + filepath.Join(dir, "gohead.db")
	}
	if !explicitlySet("media.local.path") {
		cfg.Media.Local.Path = filepath.Join(dir, "uploads")
	}
	return filepath.Join(dir, "jwt_secret"), nil
}

// startBackground runs the worker, the dispatcher and the scheduler in this
// process until ctx ends.
func startBackground(ctx context.Context, wg *sync.WaitGroup, app *Server) {
	cfg := *app.Config
	mux := queue.NewMux()
	workerHandlers(cfg, mux)
	dispatcherHandlers(app.Queue, mux)
	queues := maps.Clone(workerQueues)
	maps.Copy(queues, dispatcherQueues)
	concurrency := cfg.Queue.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	srv := newQueueServer(cfg, app.QueueDriver, app.DB, concurrency, queues)

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := srv.Run(ctx, mux); err != nil {
			logger.Log.WithError(err).Fatal("Could not run background jobs")
		}
	}()
	go func() {
		defer wg.Done()
		runScheduler(ctx, app.Queue)
	}()
	logger.Log.WithField("queue", app.QueueDriver).WithField("concurrency", concurrency).
		Info("Standalone: worker, dispatcher and scheduler running in this process")
}
