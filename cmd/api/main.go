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
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

	_ "github.com/nodebyte/backend/docs"
	"github.com/nodebyte/backend/internal/config"
	"github.com/nodebyte/backend/internal/crypto"
	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/handlers"
	"github.com/nodebyte/backend/internal/queue"
	"github.com/nodebyte/backend/internal/workers"
)

func main() {
	// Load .env file from current directory
	if err := godotenv.Load(".env"); err != nil {
		log.Warn().Err(err).Msg(".env file not found, using environment variables")
	}

	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().Str("env", cfg.Env).Msg("Starting NodeByte Backend Service")

	// Initialize database connection
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	log.Info().Msg("Connected to PostgreSQL database")

	// Create encryptor if encryption key is configured
	var encryptor *crypto.Encryptor
	encryptor, err = crypto.NewEncryptorFromEnv()
	if err != nil {
		log.Warn().Err(err).Msg("Encryption not configured; sensitive values stored unencrypted")
	}

	// Load system settings from database to override/populate config
	if err := cfg.MergeFromDB(db, encryptor); err != nil {
		log.Warn().Err(err).Msg("Failed to load settings from database; using env values only")
	} else {
		log.Info().Msg("Loaded system settings from database")
	}

	// Debug: Log configuration state
	log.Debug().
		Str("pterodactyl_url", cfg.PterodactylURL).
		Int("pterodactyl_api_key_len", len(cfg.PterodactylAPIKey)).
		Int("pterodactyl_client_api_key_len", len(cfg.PterodactylClientAPIKey)).
		Msg("Configuration initialized")

	// Parse Redis URL and create Asynq client
	// REDIS_URL format: redis://user:pass@host:port/db or host:port
	redisOpt, err := parseRedisURL(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse Redis URL")
	}

	log.Info().
		Str("redis_addr", redisOpt.Addr).
		Int("redis_db", redisOpt.DB).
		Bool("redis_has_password", redisOpt.Password != "").
		Msg("Redis connection configured")

	// Initialize Asynq client (for enqueuing tasks)
	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	// Initialize queue manager
	queueManager := queue.NewManager(asynqClient)

	log.Info().Str("redis", cfg.RedisURL).Msg("Connected to Redis")

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "NodeByte Backend v1.0.0",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-API-Key",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// API key middleware for protected routes
	apiKeyMiddleware := handlers.NewAPIKeyMiddleware(cfg.APIKey)

	// Setup routes
	handlers.SetupRoutes(app, db, queueManager, apiKeyMiddleware, cfg)

	// Start Asynq worker server in a goroutine
	workerServer := workers.NewServer(redisOpt, db, cfg)
	go func() {
		if err := workerServer.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start worker server")
		}
	}()

	// Start scheduler for cron jobs
	scheduler := workers.NewScheduler(db, redisOpt, cfg)
	go func() {
		if err := scheduler.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start scheduler")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("Shutting down server...")

		// Stop scheduler
		scheduler.Stop()

		// Stop worker server
		workerServer.Stop()

		// Shutdown Fiber with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(ctx); err != nil {
			log.Error().Err(err).Msg("Server forced to shutdown")
		}
	}()

	// Start HTTP server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Info().Str("port", port).Msg("Starting HTTP server")
	if err := app.Listen(":" + port); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

// parseRedisURL parses a Redis connection string (redis://user:pass@host:port/db)
// and returns an Asynq RedisClientOpt
func parseRedisURL(redisURL string) (asynq.RedisClientOpt, error) {
	// Handle simple host:port format
	if !strings.Contains(redisURL, "://") {
		parts := strings.Split(redisURL, ":")
		if len(parts) == 2 {
			return asynq.RedisClientOpt{Addr: redisURL}, nil
		}
	}

	// Parse full redis:// URL
	u, err := url.Parse(redisURL)
	if err != nil {
		return asynq.RedisClientOpt{}, err
	}

	// Get host and port
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "6379"
	}
	addr := host + ":" + port

	// Get credentials
	var password string
	if u.User != nil {
		password, _ = u.User.Password()
	}

	// Get database number
	db := 0
	if u.Path != "" {
		path := strings.TrimPrefix(u.Path, "/")
		if path != "" {
			if dbNum, err := strconv.Atoi(path); err == nil {
				db = dbNum
			}
		}
	}

	return asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
		DB:       db,
	}, nil
}
