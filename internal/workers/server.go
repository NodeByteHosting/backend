package workers

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/config"
	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/panels"
	"github.com/nodebyte/backend/internal/queue"
)

// Server is the Asynq worker server
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewServer creates a new worker server
func NewServer(redisOpt asynq.RedisClientOpt, db *database.DB, cfg *config.Config) *Server {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Queue priorities
			Queues: map[string]int{
				queue.QueueCritical: 6,
				queue.QueueDefault:  3,
				queue.QueueLow:      1,
			},
			// Error handler
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Error().
					Err(err).
					Str("task_type", task.Type()).
					Bytes("payload", task.Payload()).
					Msg("Task failed")
			}),
			// Logger
			Logger: &asynqLogger{},
		},
	)

	// Create handlers
	pteroClient := panels.NewPterodactylClientWithClientKey(
		cfg.PterodactylURL,
		cfg.PterodactylAPIKey,
		cfg.PterodactylClientAPIKey,
		cfg.CFAccessClientID,
		cfg.CFAccessClientSecret,
	)

	syncHandler := NewSyncHandler(db, pteroClient, cfg)
	emailHandler := NewEmailHandler(cfg)
	webhookHandler := NewWebhookHandler(db)

	// Setup task handlers
	mux := asynq.NewServeMux()

	// Sync tasks
	mux.HandleFunc(queue.TypeSyncFull, syncHandler.HandleFullSync)
	mux.HandleFunc(queue.TypeSyncLocations, syncHandler.HandleSyncLocations)
	mux.HandleFunc(queue.TypeSyncNodes, syncHandler.HandleSyncNodes)
	mux.HandleFunc(queue.TypeSyncAllocations, syncHandler.HandleSyncAllocations)
	mux.HandleFunc(queue.TypeSyncNests, syncHandler.HandleSyncNests)
	mux.HandleFunc(queue.TypeSyncServers, syncHandler.HandleSyncServers)
	mux.HandleFunc(queue.TypeSyncDatabases, syncHandler.HandleSyncDatabases)
	mux.HandleFunc(queue.TypeSyncUsers, syncHandler.HandleSyncUsers)

	// Email tasks
	mux.HandleFunc(queue.TypeEmailSend, emailHandler.HandleSendEmail)

	// Webhook tasks
	mux.HandleFunc(queue.TypeWebhookDiscord, webhookHandler.HandleDiscordWebhook)

	// Cleanup tasks
	mux.HandleFunc(queue.TypeCleanupLogs, syncHandler.HandleCleanupLogs)

	return &Server{
		server: server,
		mux:    mux,
	}
}

// Start starts the worker server
func (s *Server) Start() error {
	log.Info().Msg("Starting Asynq worker server")
	return s.server.Run(s.mux)
}

// Stop gracefully stops the worker server
func (s *Server) Stop() {
	log.Info().Msg("Stopping Asynq worker server")
	s.server.Shutdown()
}

// asynqLogger implements asynq.Logger interface
type asynqLogger struct{}

func (l *asynqLogger) Debug(args ...interface{}) {
	log.Debug().Msgf("%v", args)
}

func (l *asynqLogger) Info(args ...interface{}) {
	log.Info().Msgf("%v", args)
}

func (l *asynqLogger) Warn(args ...interface{}) {
	log.Warn().Msgf("%v", args)
}

func (l *asynqLogger) Error(args ...interface{}) {
	log.Error().Msgf("%v", args)
}

func (l *asynqLogger) Fatal(args ...interface{}) {
	log.Fatal().Msgf("%v", args)
}
