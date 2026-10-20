package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
		if !cmd.Flags().Changed("data") {
			if env := os.Getenv("GOHEAD_DATA_DIR"); env != "" {
				dataDir = env
			}
		}
		runServer(configPath, ServerOptions{Standalone: true, DataDir: dataDir})
	},
}

func init() {
	standaloneCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file (optional)")
	standaloneCmd.Flags().StringP("data", "d", "data", "Directory of the SQLite database, uploads and generated secrets (or GOHEAD_DATA_DIR)")
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
	if info, err := os.Stat(dir); err == nil && info.Mode().Perm()&0o007 != 0 && runtime.GOOS != "windows" {
		logger.Log.WithField("dir", dir).Warn("The data directory is readable by other users of this machine; it holds the database and the JWT secret (chmod 750 it)")
	}
	if !explicitlySet("database_url") {
		dbFile := filepath.Join(dir, "gohead.db")
		cfg.DatabaseURL = "sqlite://" + dbFile
		// Created private: it holds password hashes and webhook secrets
		// (SQLite gives its -wal and -shm files the same mode).
		if f, err := os.OpenFile(dbFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600); err == nil {
			f.Close()
		}
	}
	if !explicitlySet("media.local.path") {
		cfg.Media.Local.Path = filepath.Join(dir, "uploads")
	}
	return filepath.Join(dir, "jwt_secret"), nil
}

// startBackground runs the worker, the dispatcher and the scheduler in this
// process until ctx ends. Events have their own small pool, so long agent
// runs never hold up dispatching and webhook deliveries.
func startBackground(ctx context.Context, wg *sync.WaitGroup, app *Server) {
	cfg := *app.Config
	concurrency := cfg.Queue.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	workMux := queue.NewMux()
	workerHandlers(cfg, workMux)
	eventMux := queue.NewMux()
	dispatcherHandlers(app.Queue, eventMux)
	servers := []struct {
		srv queue.Server
		mux *queue.Mux
	}{
		{newQueueServer(cfg, app.QueueDriver, app.DB, concurrency, workerQueues), workMux},
		{newQueueServer(cfg, app.QueueDriver, app.DB, 2, dispatcherQueues), eventMux},
	}
	for _, s := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.srv.Run(ctx, s.mux); err != nil {
				logger.Log.WithError(err).Fatal("Could not run background jobs")
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		runScheduler(ctx, app.Queue)
	}()
	logger.Log.WithField("queue", app.QueueDriver).WithField("concurrency", concurrency).
		Info("Standalone: worker, dispatcher and scheduler running in this process")
}
