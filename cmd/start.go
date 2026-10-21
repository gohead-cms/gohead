package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gohead-cms/gohead/pkg/llm"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gohead-cms/gohead/internal/accounts"
	"github.com/gohead-cms/gohead/internal/agent/triggers"
	"github.com/gohead-cms/gohead/internal/api/handlers"
	"github.com/gohead-cms/gohead/internal/api/middleware"
	"github.com/gohead-cms/gohead/internal/graphql"
	"github.com/gohead-cms/gohead/internal/webhooks"
	"github.com/gohead-cms/gohead/internal/webui"
	"github.com/gohead-cms/gohead/pkg/auth"
	"github.com/gohead-cms/gohead/pkg/cluster"
	"github.com/gohead-cms/gohead/pkg/config"
	"github.com/gohead-cms/gohead/pkg/database"
	"github.com/gohead-cms/gohead/pkg/logger"
	"github.com/gohead-cms/gohead/pkg/media"
	"github.com/gohead-cms/gohead/pkg/metrics"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/queue"
	"github.com/gohead-cms/gohead/pkg/ratelimit"
	"github.com/gohead-cms/gohead/pkg/seed"
	"github.com/gohead-cms/gohead/pkg/storage"
	"github.com/gohead-cms/gohead/pkg/tracing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	ginlogrus "github.com/toorop/gin-logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/gorm"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start GoHead API server",
	Long: `Starts the API (and the admin UI, when the binary includes it). Background
jobs run in separate processes: gohead worker, dispatcher and scheduler. To run
everything in one process, use gohead standalone.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		runServer(configPath, ServerOptions{})
	},
}

// ServerOptions changes how the server runs.
type ServerOptions struct {
	// Standalone runs the worker, dispatcher and scheduler in the API
	// process, with the database job queue unless configured otherwise.
	Standalone bool
	// DataDir holds the SQLite database, the uploads and a generated JWT
	// secret, unless they are configured otherwise.
	DataDir string
}

// Server is an initialized GoHead.
type Server struct {
	Router *gin.Engine
	Config *config.Config
	// Queue driver and client of background jobs.
	QueueDriver string
	Queue       queue.Client
	DB          *gorm.DB
}

// runServer serves the API until SIGINT or SIGTERM.
func runServer(configPath string, opts ServerOptions) {
	app, cleanup, err := InitializeServer(configPath, opts)
	if err != nil {
		if logger.Log != nil {
			logger.Log.Fatalf("Cannot initialize server: %v", err)
		}
		log.Fatalf("Cannot initialize server: %v", err)
	}
	defer cleanup()
	cfg := app.Config

	srv := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           app.Router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signalContext()
	defer stop()

	// Standalone: background jobs in this process.
	bgCtx, stopBackground := context.WithCancel(context.Background())
	var background sync.WaitGroup
	if opts.Standalone {
		startBackground(bgCtx, &background, app)
	}

	go func() {
		logger.Log.Infof("Starting server on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatalf("Cannot start server: %v", err)
		}
	}()

	<-ctx.Done()
	logger.Log.Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.WithError(err).Error("Server forced to shut down")
	}
	stopBackground()
	background.Wait()
}

func init() {
	startCmd.Flags().StringP("config", "c", "config.yaml", "Path to the configuration file")
}

// InitializeServer builds the router and its dependencies. The returned
// cleanup function releases long-lived resources (queue client, tracer) and
// must be called when the server stops.
func InitializeServer(cfgPath string, opts ServerOptions) (*Server, func(), error) {
	var closers []func()
	cleanup := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
	}
	// --- Core Dependencies ---
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	logger.InitLogger(cfg.LogLevel)
	jwtSecretFile, err := applyDataDir(&cfg, opts.DataDir)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	llm.SetDefaultConfig(cfg.LLM)
	webhooks.Configure(webhooks.Config{
		AllowPrivateNetworks: cfg.Webhooks.AllowPrivateNetworks,
		Timeout:              cfg.Webhooks.Timeout,
		MaxRetries:           cfg.Webhooks.MaxRetries,
	})

	driver, err := queueDriver(cfg, opts.Standalone)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}

	db, err := database.InitDatabase(cfg.DatabaseURL, gormLogLevel(cfg.LogLevel))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	storage.RevisionLimit = max(cfg.Content.RevisionsKept, 0)
	if cfg.AutoMigrate {
		if _, err := migrations.Up(db); err != nil {
			cleanup()
			return nil, func() {}, err
		}
	} else if err := migrations.Check(db); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if err := storage.EnsureDefaultLocale(cfg.I18n.DefaultLocale); err != nil {
		cleanup()
		return nil, func() {}, err
	}

	// --- Media storage ---
	localMedia, err := initMediaStore(cfg)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}

	// Initialize the GraphQL Schema
	if err := graphql.InitializeGraphQLSchema(); err != nil {
		logger.Log.WithError(err).Error("Failed to initialize GraphQL schema")
		cleanup()
		return nil, func() {}, err
	}

	// --- Application Services ---
	seed.SeedRoles()
	if err := setupAccounts(cfg); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	auth.InitializeJWT(resolveJWTSecret(cfg.JWTSecret, jwtSecretFile))
	metrics.InitMetrics()

	// --- Job queue ---
	// The API queues jobs (collection events, agent runs) for the workers.
	queueClient := openQueueClient(cfg, driver, db)
	storage.InitQueue(queueClient)
	triggers.InitQueue(queueClient)
	handlers.InitQueue(queueClient) // agent webhook endpoint and manual runs
	closers = append(closers, func() {
		if err := queueClient.Close(); err != nil {
			logger.Log.WithError(err).Warn("Error closing queue client")
		}
	})
	logger.Log.WithField("driver", driver).Info("Job queue ready")

	// --- Shared state across replicas (Redis) ---
	var rdb *redis.Client
	if useRedis(cfg, driver) {
		rdb = connectRedis(cfg)
	} else {
		logger.Log.Info("Redis is not used: rate limits are kept in memory and schema changes are not shared, so run a single API process")
	}
	if rdb != nil {
		closers = append(closers, func() { _ = rdb.Close() })
	}

	// GraphQL schema: rebuild when another replica changes a collection.
	schemaSync := cluster.NewSchemaSync(rdb)
	if rdb != nil {
		handlers.SchemaNotifier = schemaSync
		syncCtx, stopSync := context.WithCancel(context.Background())
		go schemaSync.Run(syncCtx, graphql.InitializeGraphQLSchema)
		closers = append(closers, stopSync)
	}

	// Authentication rate limits, shared by all replicas.
	var authLimiter ratelimit.Counter
	if cfg.RateLimit.Enabled {
		authLimiter = ratelimit.New(rdb, "gohead:ratelimit:")
		if cfg.RateLimit.LoginFailures > 0 {
			handlers.LoginGuard = &handlers.LoginAttemptGuard{
				Counter:     authLimiter,
				MaxFailures: cfg.RateLimit.LoginFailures,
				Window:      cfg.RateLimit.LoginWindow,
			}
		}
	}

	// --- Telemetry (Optional) ---
	if cfg.TelemetryEnabled {
		tracerProvider, err := tracing.InitTracer()
		if err != nil {
			cleanup()
			return nil, func() {}, err
		}
		// Shut down (and flush spans) when the server stops, not when this
		// function returns.
		closers = append(closers, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracerProvider.Shutdown(ctx); err != nil {
				logger.Log.WithError(err).Error("Error shutting down tracer provider")
			}
		})
	}

	// --- Gin Router Setup ---
	switch cfg.Mode {
	case "development":
		gin.SetMode(gin.DebugMode)
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	// Only trust X-Forwarded-For from configured proxies; otherwise clients
	// could spoof their IP and dodge rate limits.
	var trusted []string
	if len(cfg.TrustedProxies) > 0 {
		trusted = cfg.TrustedProxies
	}
	if err := router.SetTrustedProxies(trusted); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("invalid trusted_proxies: %w", err)
	}
	router.Use(ginlogrus.Logger(logger.Log), gin.Recovery())
	router.Use(middleware.CORSMiddleware(cfg))
	router.Use(middleware.MetricsMiddleware())
	router.Use(otelgin.Middleware("gohead"))
	router.Use(middleware.ResponseWrapper())

	// --- Routes ---
	authLimit := middleware.RateLimit(authLimiter, "auth", 0, time.Minute)
	if cfg.RateLimit.Enabled {
		authLimit = middleware.RateLimit(authLimiter, "auth", cfg.RateLimit.AuthPerMinute, time.Minute)
	}
	// Requests without a token act with the public role, when it grants
	// anything; they have their own rate limit.
	var publicLimit middleware.Limiter
	if cfg.RateLimit.Enabled {
		publicLimit = middleware.RateLimitAllow(authLimiter, "public", cfg.RateLimit.PublicPerMinute, time.Minute)
	}
	handlers.APIVersion = version
	setupRoutes(router, authLimit, publicLimit, localMedia, cfg.OpenAPI)
	if cfg.UI.Enabled {
		if webui.Register(router) {
			logger.Log.Info("Admin UI served at /")
		}
	}

	return &Server{Router: router, Config: &cfg, QueueDriver: driver, Queue: queueClient, DB: db}, cleanup, nil
}

// initMediaStore configures uploads. A misconfigured S3 store stops the
// server; a local directory that cannot be created (e.g. a read-only
// filesystem) only disables uploads, with a warning. It returns the local
// store, whose files the API serves itself, if that driver is used.
func initMediaStore(cfg config.Config) (*media.Local, error) {
	if !cfg.Media.Enabled {
		logger.Log.Info("Media uploads are disabled (media.enabled=false)")
		return nil, nil
	}
	m := cfg.Media
	store, err := media.New(context.Background(), media.Config{
		Driver:    m.Driver,
		PublicURL: m.PublicURL,
		Local:     media.LocalConfig{Path: m.Local.Path},
		S3: media.S3Config{
			Endpoint: m.S3.Endpoint, Region: m.S3.Region, Bucket: m.S3.Bucket,
			AccessKey: m.S3.AccessKey, SecretKey: m.S3.SecretKey,
			UseSSL: m.S3.UseSSL, PathStyle: m.S3.PathStyle, Prefix: m.S3.Prefix,
			PresignExpiry: m.S3.PresignExpiry,
		},
	})
	if err != nil {
		if strings.EqualFold(m.Driver, "s3") {
			return nil, fmt.Errorf("media storage: %w", err)
		}
		logger.Log.WithError(err).Warn("Media uploads are disabled: the local media directory is not writable. " +
			"Set media.local.path (GOHEAD_MEDIA_LOCAL_PATH) to a writable volume, or use media.driver=s3.")
		return nil, nil
	}
	storage.InitMediaStore(store)
	handlers.MediaPolicy = storage.UploadPolicy{
		MaxSize:      int64(m.MaxSizeMB) << 20,
		AllowedTypes: m.AllowedTypes,
	}
	logger.Log.WithField("driver", store.Driver()).WithField("max_size_mb", m.MaxSizeMB).Info("Media uploads enabled")
	local, _ := store.(*media.Local)
	if local != nil {
		logger.Log.WithField("path", local.Root()).Info("Media files are stored on local disk; use the S3 driver to run several API replicas")
	}
	return local, nil
}

func setupRoutes(router *gin.Engine, authLimit gin.HandlerFunc, publicLimit middleware.Limiter, localMedia *media.Local, openAPI config.OpenAPIConfig) {
	// API description: the spec (public or behind a token) and Swagger UI.
	if openAPI.Enabled {
		if openAPI.Public {
			router.GET("/api/openapi.json", handlers.OpenAPISpec)
		}
		router.GET("/api/docs", handlers.APIDocs(openAPI.Public))
	}

	// Files of the local media driver (public, like a bucket's URLs).
	if localMedia != nil {
		router.GET(media.LocalPathPrefix+"/*key", handlers.ServeLocalMedia(localMedia))
		router.HEAD(media.LocalPathPrefix+"/*key", handlers.ServeLocalMedia(localMedia))
	}

	// Monitoring & Healthcheck
	router.GET("/_metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/_health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public routes
	authRoutes := router.Group("/auth", authLimit)
	{
		authRoutes.POST("/register", handlers.Register)
		authRoutes.POST("/login", handlers.Login)
	}
	// Not rate limited with login: the admin UI calls it on every page load.
	router.GET("/auth/me", middleware.AuthMiddleware(), handlers.Me)

	// Agent Webhook Trigger (Public, authenticates with a token)
	router.POST("/agents/webhook/:id", handlers.HandleWebhook)

	// ADMIN routes (schema/definition)
	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		// Collections
		admin.POST("/collections", handlers.CreateCollection)
		admin.GET("/collections", handlers.GetCollections)
		admin.GET("/collections/:name", handlers.GetCollection)
		admin.PUT("/collections/:name", handlers.UpdateCollection)
		admin.DELETE("/collections/:name", handlers.DeleteCollection)

		// Singletons
		admin.GET("/singletons", handlers.GetSingletons)
		admin.POST("/singleton", handlers.CreateOrUpdateSingleton)
		admin.GET("/singleton/:name", handlers.GetSingleton)
		admin.PUT("/singleton/:name", handlers.CreateOrUpdateSingleton)
		admin.DELETE("/singleton/:name", handlers.DeleteSingleton)

		// Components
		admin.POST("/components", handlers.CreateComponent)
		admin.GET("/components", handlers.GetComponents)
		admin.GET("/components/:name", handlers.GetComponent)
		admin.PUT("/components/:name", handlers.UpdateComponent)
		admin.DELETE("/components/:name", handlers.DeleteComponent)

		// Users
		// Credentials (users, API tokens) can only be managed from a user
		// session, so a leaked token cannot mint new credentials.
		creds := admin.Group("", middleware.UserSessionOnly())
		creds.POST("/users", handlers.CreateUser)
		creds.GET("/users", handlers.GetAllUsers)
		creds.GET("/users/:id", handlers.GetUser)
		creds.PUT("/users/:id", handlers.UpdateUser)
		creds.DELETE("/users/:id", handlers.DeleteUser)
		creds.POST("/tokens", handlers.CreateAPIToken)
		creds.GET("/tokens", handlers.ListAPITokens)
		creds.GET("/tokens/:id", handlers.GetAPIToken)
		creds.DELETE("/tokens/:id", handlers.RevokeAPIToken)
		creds.GET("/roles", handlers.ListRoles)
		creds.GET("/roles/:name", handlers.GetRole)
		creds.POST("/roles", handlers.CreateRole)
		creds.PUT("/roles/:name", handlers.UpdateRole)
		creds.DELETE("/roles/:name", handlers.DeleteRole)

		// Outbound webhooks
		admin.POST("/locales", handlers.CreateLocale)
		admin.PUT("/locales/:code", handlers.UpdateLocale)
		admin.DELETE("/locales/:code", handlers.DeleteLocale)

		admin.GET("/overview", handlers.GetOverview)
		admin.GET("/runs", handlers.ListAllAgentRuns)
		admin.GET("/agent-functions", handlers.AgentFunctionCatalog)

		admin.POST("/webhooks", handlers.CreateWebhook)
		admin.GET("/webhooks", handlers.ListWebhooks)
		admin.GET("/webhooks/:id", handlers.GetWebhook)
		admin.PUT("/webhooks/:id", handlers.UpdateWebhook)
		admin.DELETE("/webhooks/:id", handlers.DeleteWebhook)
		admin.GET("/webhooks/:id/deliveries", handlers.ListWebhookDeliveries)
		admin.POST("/webhooks/:id/test", handlers.TestWebhook)

		// Agents
		admin.GET("/agent-templates", handlers.ListAgentTemplates)
		admin.POST("/agent-templates/:id/render", handlers.RenderAgentTemplate)
		agents := admin.Group("/agents")
		{
			agents.POST("", handlers.CreateAgent)
			agents.GET("", handlers.GetAgents)
			agents.GET("/:name", handlers.GetAgent)
			agents.PUT("/:name", handlers.UpdateAgent)
			agents.DELETE("/:name", handlers.DeleteAgent)
			agents.GET("/:name/runs", handlers.ListAgentRuns)
			agents.GET("/:name/runs/:run", handlers.GetAgentRun)
			agents.GET("/:name/usage", handlers.GetAgentUsage)
			agents.POST("/:name/run", handlers.RunAgentNow)
			agents.POST("/:name/test", handlers.TestAgent)
			agents.POST("/:name/runs/:run/revert", handlers.RevertAgentRun)
		}
	}

	// CONTENT routes (actual data/items)
	content := router.Group("/api")
	content.Use(middleware.PublicOrAuth(publicLimit))
	signedIn := middleware.SignedInOnly()
	{
		content.POST("/graphql", handlers.GraphQLHandler)

		// The content model, read-only, for every signed-in user (editors
		// need it to edit content; changing it stays under /admin).
		content.GET("/schema/collections", signedIn, handlers.GetCollections)
		content.GET("/schema/collections/:name", signedIn, handlers.GetCollection)
		content.GET("/schema/singletons", signedIn, handlers.GetSingletons)
		content.GET("/schema/singletons/:name", signedIn, handlers.GetSingleton)
		content.GET("/schema/components", signedIn, handlers.GetComponents)
		content.GET("/schema/components/:name", signedIn, handlers.GetComponent)
		content.GET("/locales", handlers.ListLocales)
		if openAPI.Enabled && !openAPI.Public {
			content.GET("/openapi.json", signedIn, handlers.OpenAPISpec)
		}

		// Dynamic Handlers
		content.Any("/collections/:collection", handlers.DynamicCollectionHandler)
		content.Any("/collections/:collection/:id", handlers.DynamicCollectionHandler)
		content.POST("/collections/:collection/:id/publish", handlers.PublishItem(true))
		content.POST("/collections/:collection/:id/unpublish", handlers.PublishItem(false))
		// Content history (editors and admins)
		content.GET("/collections/:collection/:id/revisions", handlers.ListItemRevisions)
		content.GET("/collections/:collection/:id/revisions/:version", handlers.GetItemRevision)
		content.POST("/collections/:collection/:id/revisions/:version/restore", handlers.RestoreItemRevision)
		content.GET("/singleton/:name/revisions", handlers.ListSingletonRevisions)
		content.GET("/singleton/:name/revisions/:version", handlers.GetSingletonRevision)
		content.POST("/singleton/:name/revisions/:version/restore", handlers.RestoreSingletonRevision)
		content.GET("/revisions", handlers.ListRevisions)
		content.GET("/proposals", handlers.ListProposals)
		content.GET("/proposals/:id", handlers.GetProposal)
		content.POST("/proposals/:id/approve", handlers.ApproveProposal)
		content.POST("/proposals/:id/reject", handlers.RejectProposal)
		content.POST("/media", handlers.UploadMedia)
		content.GET("/media", handlers.ListMedia)
		content.GET("/media/:id", handlers.GetMedia)
		content.PUT("/media/:id", handlers.UpdateMedia)
		content.DELETE("/media/:id", handlers.DeleteMedia)
		content.GET("/singleton/:name", handlers.GetSingleItem)
		content.POST("/singleton/:name", handlers.CreateOrUpdateSingletonItem)
		content.PUT("/singleton/:name", handlers.CreateOrUpdateSingletonItem)
	}
}

// insecureJWTSecrets are well-known values (shipped in samples/defaults) that
// would let anyone forge admin tokens.
var insecureJWTSecrets = map[string]bool{
	"":                true,
	"your-secret-key": true,
	// docker-compose.yml's fallback when GOHEAD_JWT_SECRET is not set
	"change-me-to-a-long-random-string-0123456789": true,
}

// resolveJWTSecret returns the configured secret, or a random one when the
// configured value is missing or a published default. A random secret keeps
// quick-start setups working without being forgeable, at the cost of
// invalidating tokens on restart.
func resolveJWTSecret(configured, persistFile string) string {
	if !insecureJWTSecrets[configured] {
		if len(configured) < 32 {
			logger.Log.Warn("jwt_secret is shorter than 32 characters; use a long random value")
		}
		return configured
	}
	if persistFile != "" {
		if b, err := os.ReadFile(persistFile); err == nil && len(strings.TrimSpace(string(b))) >= 32 {
			return strings.TrimSpace(string(b))
		}
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		logger.Log.Fatalf("Cannot generate a JWT secret: %v", err)
	}
	secret := hex.EncodeToString(buf)
	if persistFile != "" {
		if err := os.WriteFile(persistFile, []byte(secret+"\n"), 0o600); err == nil {
			logger.Log.WithField("file", persistFile).Info("jwt_secret is not set: generated one and saved it, so sessions survive restarts")
			return secret
		} else {
			logger.Log.WithError(err).Warn("Could not save the generated JWT secret")
		}
	}
	logger.Log.Warn("jwt_secret is not set (or uses the sample default): using a random secret for this run. " +
		"Tokens will not survive a restart; set jwt_secret (or GOHEAD_JWT_SECRET) to a long random value.")
	return secret
}

// connectRedis returns a client for the configured Redis, or nil when it is
// not reachable at startup. Without Redis, schema changes are not broadcast
// (single replica only) and rate limits are per process.
func connectRedis(cfg config.Config) *redis.Client {
	rdb := redis.NewClient(cfg.RedisOptions())
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = rdb.Ping(ctx).Err()
		cancel()
		if err == nil {
			return rdb
		}
		time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
	}
	logger.Log.WithError(err).Warn("Redis unreachable: schema changes will not be shared between replicas and rate limits are per process. Run a single API replica or fix GOHEAD_REDIS_ADDRESS.")
	_ = rdb.Close()
	return nil
}

// setupAccounts applies the registration mode and creates the configured
// first administrator on an empty instance.
func setupAccounts(cfg config.Config) error {
	switch cfg.Auth.Registration {
	case handlers.RegistrationDisabled, handlers.RegistrationOpen:
		handlers.Registration = cfg.Auth.Registration
	default:
		return fmt.Errorf("auth.registration must be %q or %q, not %q",
			handlers.RegistrationDisabled, handlers.RegistrationOpen, cfg.Auth.Registration)
	}
	if _, err := accounts.BootstrapAdmin(cfg.Admin.Username, cfg.Admin.Email, cfg.Admin.Password); err != nil {
		return fmt.Errorf("could not create the first administrator: %w", err)
	}
	if count, err := storage.CountUsers(); err == nil && count == 0 {
		logger.Log.Warn("No account exists yet. Create an administrator with `gohead admin create`, " +
			"or set GOHEAD_ADMIN_USERNAME, GOHEAD_ADMIN_EMAIL and GOHEAD_ADMIN_PASSWORD and restart.")
	}
	return nil
}
