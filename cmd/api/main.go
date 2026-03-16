// @title NodeByte API
// @version 0.2.0
// @description Comprehensive REST API for managing game server infrastructure with panel integration, async job queue management, and real-time sync operations
// @termsOfService https://nodebyte.co.uk/legal/terms
// @contact.name Contact Support
// @contact.url https://discord.gg/wN58bTzzpW
// @license.name AGPL 3.0
// @license.url https://www.gnu.org/licenses/agpl-3.0.en.html
// @host core.nodebyte.host
// @basePath /
// @schemes http https
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description Backend-to-backend API authentication
// @securityDefinitions.http BearerAuth
// @scheme bearer
// @bearerFormat JWT
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	_ "github.com/nodebyte/backend/docs"
	"github.com/nodebyte/backend/internal/cli/api"
	"github.com/nodebyte/backend/internal/config"
	"github.com/nodebyte/backend/internal/crypto"
	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/handlers"
	"github.com/nodebyte/backend/internal/queue"
	"github.com/nodebyte/backend/internal/sentry"
	"github.com/nodebyte/backend/internal/workers"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "api",
		Short: "NodeByte Backend API Server",
		Long:  "Start the NodeByte backend API server with worker and scheduler services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer()
		},
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runServer initializes and starts the API server.
func runServer() error {
	// Initialize logging and configuration
	initLogging()
	cfg := loadConfig()

	log.Info().Str("env", cfg.Env).Msg("Starting NodeByte Backend Service")

	// Initialize database
	db := connectDatabase(cfg)
	defer db.Close()

	// Initialize encryption
	encryptor := initEncryption()

	// Load database settings
	if err := cfg.MergeFromDB(db, encryptor); err != nil {
		log.Warn().Err(err).Msg("Failed to load settings from database; using env values only")
	} else {
		log.Info().Msg("Loaded system settings from database")
	}

	log.Debug().
		Str("pterodactyl_url", cfg.PterodactylURL).
		Int("pterodactyl_api_key_len", len(cfg.PterodactylAPIKey)).
		Int("pterodactyl_client_api_key_len", len(cfg.PterodactylClientAPIKey)).
		Msg("Configuration initialized")

	// Initialize Redis and queue
	_, queueMgr := initQueue(cfg)
	// ensure the underlying Asynq client is closed when runServer exits
	defer func() {
		if err := queueMgr.Close(); err != nil {
			log.Error().Err(err).Msg("error closing queue manager client")
		}
	}()

	// Initialize Sentry
	sentryHandler := initSentry(cfg)

	// Setup and start HTTP server
	return startServer(cfg, db, queueMgr, sentryHandler)
}

// initLogging configures the logging system.
func initLogging() {
	if err := godotenv.Load(".env"); err != nil {
		log.Warn().Err(err).Msg(".env file not found, using environment variables")
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

// loadConfig loads and returns the application configuration.
func loadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	return cfg
}

// connectDatabase establishes a database connection.
func connectDatabase(cfg *config.Config) *database.DB {
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	log.Info().Msg("Connected to PostgreSQL database")
	return db
}

// initEncryption initializes the encryption system.
func initEncryption() *crypto.Encryptor {
	encryptor, err := crypto.NewEncryptorFromEnv()
	if err != nil {
		log.Warn().Err(err).Msg("Encryption not configured; sensitive values stored unencrypted")
	}
	return encryptor
}

// initQueue initializes Redis configuration.
func initQueue(cfg *config.Config) (asynq.RedisClientOpt, *queue.Manager) {
	redisConfig, err := api.ParseRedisURL(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse Redis URL")
	}

	redisOpt := redisConfig.ToAsynqOpt()

	log.Info().
		Str("redis_addr", redisOpt.Addr).
		Int("redis_db", redisOpt.DB).
		Bool("redis_has_password", redisOpt.Password != "").
		Msg("Redis connection configured")

	// Create a client for the queue manager
	asynqClient := asynq.NewClient(redisOpt)
	log.Info().Msg("Connected to Redis")

	return redisOpt, queue.NewManager(asynqClient)
}

// initSentry initializes the Sentry error tracking system.
func initSentry(cfg *config.Config) fiber.Handler {
	sentryHandler, err := sentry.InitSentry(cfg.SentryDSN, cfg.Env, "0.2.1")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Sentry")
	}
	return sentryHandler
}

// startServer initializes and starts the HTTP server.
func startServer(cfg *config.Config, db *database.DB, queueMgr *queue.Manager, sentryHandler fiber.Handler) error {
	app := fiber.New(fiber.Config{
		AppName:      "NodeByte Backend v1.0.0",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Setup middleware
	setupMiddleware(app, sentryHandler, cfg)

	// Setup routes
	apiKeyMiddleware := handlers.NewAPIKeyMiddleware(cfg.APIKey)
	handlers.SetupRoutes(app, db, queueMgr, apiKeyMiddleware, cfg)

	// Start background services
	redisConfig, _ := api.ParseRedisURL(cfg.RedisURL)
	redisOpt := redisConfig.ToAsynqOpt()

	workerServer := workers.NewServer(redisOpt, db, cfg)
	scheduler := workers.NewScheduler(db, redisOpt, cfg)

	go startWorkerServer(workerServer)
	go startScheduler(scheduler)

	// Setup graceful shutdown (including queue client cleanup)
	setupGracefulShutdown(app, scheduler, workerServer, queueMgr)

	// Start server
	port := getPort(cfg)
	log.Info().Str("port", port).Msg("Starting HTTP server")
	return app.Listen(":" + port)
}

// setupMiddleware configures HTTP middleware.
func setupMiddleware(app *fiber.App, sentryHandler fiber.Handler, cfg *config.Config) {
	app.Use(recover.New())
	if sentryHandler != nil {
		app.Use(sentryHandler)
	}
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-API-Key",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		AllowCredentials: true,
	}))
}

// startWorkerServer starts the Asynq worker server.
func startWorkerServer(workerServer *workers.Server) {
	if err := workerServer.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start worker server")
	}
}

// startScheduler starts the task scheduler.
func startScheduler(scheduler *workers.Scheduler) {
	if err := scheduler.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start scheduler")
	}
}

// setupGracefulShutdown configures graceful server shutdown.
func setupGracefulShutdown(app *fiber.App, scheduler *workers.Scheduler, workerServer *workers.Server, queueMgr *queue.Manager) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("Shutting down server...")

		sentry.Flush(5 * time.Second)

		scheduler.Stop()
		workerServer.Stop()
		// make sure we close the queue client so Redis connections are released
		if queueMgr != nil {
			if err := queueMgr.Close(); err != nil {
				log.Error().Err(err).Msg("error closing queue manager client during shutdown")
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(ctx); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		}
	}()
}

// getPort returns the server port from config or defaults to 8080.
func getPort(cfg *config.Config) string {
	if cfg.Port != "" {
		return cfg.Port
	}
	return "8080"
}
