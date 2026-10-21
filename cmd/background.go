package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gohead-cms/gohead/internal/agent/dispatcher"
	"github.com/gohead-cms/gohead/internal/agent/events"
	"github.com/gohead-cms/gohead/internal/agent/jobs"
	runner "github.com/gohead-cms/gohead/internal/agent/runners"
	"github.com/gohead-cms/gohead/internal/agent/triggers"
	"github.com/gohead-cms/gohead/internal/webhooks"
	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/llm"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/queue"
	"github.com/gohead-cms/gohead/pkg/storage"

	"github.com/spf13/viper"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Queues of the background processes, with their priority.
var (
	workerQueues     = map[string]int{jobs.QueueAgents: 10, webhooks.Queue: 5}
	dispatcherQueues = map[string]int{events.QueueEvents: 10}
)

// queueDriver returns the configured job queue driver.
func queueDriver(cfg config.Config, standalone bool) (string, error) {
	switch d := strings.ToLower(strings.TrimSpace(cfg.Queue.Driver)); d {
	case "":
		if standalone {
			return queue.DriverDatabase, nil
		}
		return queue.DriverRedis, nil
	case queue.DriverRedis, queue.DriverDatabase:
		return d, nil
	default:
		return "", fmt.Errorf("unknown queue.driver %q (redis or database)", cfg.Queue.Driver)
	}
}

// useRedis tells whether this process should connect to Redis: always with
// the redis queue driver; otherwise only when an address was configured
// explicitly (for shared rate limits and schema changes between replicas).
func useRedis(cfg config.Config, driver string) bool {
	if driver == queue.DriverRedis {
		return true
	}
	return cfg.Redis.Address != "" && explicitlySet("redis.address")
}

// explicitlySet tells whether a setting comes from the configuration file
// or a GOHEAD_* environment variable, rather than from its default.
func explicitlySet(key string) bool {
	if viper.InConfig(key) {
		return true
	}
	_, ok := os.LookupEnv("GOHEAD_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_")))
	return ok
}

// openQueueClient returns the client that queues background jobs.
func openQueueClient(cfg config.Config, driver string, db *gorm.DB) queue.Client {
	if driver == queue.DriverDatabase {
		return queue.NewDatabaseClient(db)
	}
	return queue.NewRedisClient(cfg.RedisClientOpt())
}

// newQueueServer returns a server processing the given queues.
func newQueueServer(cfg config.Config, driver string, db *gorm.DB, concurrency int, queues map[string]int) queue.Server {
	sc := queue.ServerConfig{Concurrency: concurrency, Queues: queues}
	if driver == queue.DriverDatabase {
		return queue.NewDatabaseServer(db, queue.DatabaseServerConfig{ServerConfig: sc, PollInterval: cfg.Queue.PollInterval})
	}
	return queue.NewRedisServer(cfg.RedisClientOpt(), sc)
}

// workerHandlers registers the handlers of the worker: agent runs and
// webhook deliveries.
func workerHandlers(cfg config.Config, mux *queue.Mux) {
	agentRunner := runner.NewAgentRunner()
	if cfg.Agents.RunsKept > 0 {
		runner.RunsKept = cfg.Agents.RunsKept
	}
	mux.Handle(jobs.TaskTypeAgentRun, agentRunner.HandleAgentJob)
	mux.Handle(webhooks.TaskTypeDeliver, webhooks.HandleDelivery)
}

// dispatcherHandlers registers the handler of the dispatcher: collection
// events, turned into agent runs and webhook deliveries.
func dispatcherHandlers(client queue.Client, mux *queue.Mux) {
	mux.Handle(events.TaskTypeCollectionEvent, dispatcher.NewEventDispatcher(client).HandleCollectionEvent)
}

// runScheduler enqueues the jobs of cron triggers until ctx ends, reloading
// the cron agents every reloadInterval.
func runScheduler(ctx context.Context, client queue.Client) {
	triggers.InitQueue(client)
	triggers.StartScheduler()
	reload := time.NewTicker(reloadInterval)
	defer reload.Stop()
	for {
		select {
		case <-reload.C:
			triggers.StartScheduler()
		case <-ctx.Done():
			if triggers.Scheduler != nil {
				triggers.Scheduler.Stop()
			}
			return
		}
	}
}

// gormLogLevel maps GoHead's log level to GORM's.
func gormLogLevel(level string) gormlogger.LogLevel {
	switch level {
	case "debug":
		return gormlogger.Info
	case "info", "warn", "warning":
		return gormlogger.Warn
	case "error":
		return gormlogger.Error
	default:
		return gormlogger.Silent
	}
}

// backgroundSetup loads the configuration and opens the database for the
// worker, dispatcher and scheduler commands.
func backgroundSetup(cfgPath string) (config.Config, string, *gorm.DB, error) {
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return cfg, "", nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	logger.InitLogger(cfg.LogLevel)
	webhooks.Configure(webhooks.Config{
		AllowPrivateNetworks: cfg.Webhooks.AllowPrivateNetworks,
		Timeout:              cfg.Webhooks.Timeout,
		MaxRetries:           cfg.Webhooks.MaxRetries,
	})
	configureApps(cfg)
	configureSecrets(cfg.JWTSecret, cfg.JWTSecret, "")
	llm.SetDefaultConfig(cfg.LLM)
	storage.RevisionLimit = max(cfg.Content.RevisionsKept, 0)
	driver, err := queueDriver(cfg, false)
	if err != nil {
		return cfg, "", nil, err
	}
	db, err := database.InitDatabase(cfg.DatabaseURL, gormLogLevel(cfg.LogLevel))
	if err != nil {
		return cfg, "", nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	if driver == queue.DriverDatabase && !db.Migrator().HasTable(&queue.Job{}) {
		return cfg, "", nil, fmt.Errorf("the jobs table does not exist: start the API (or run gohead migrate) first")
	}
	return cfg, driver, db, nil
}

// signalContext ends on SIGINT or SIGTERM.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
