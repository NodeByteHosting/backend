package workers

import (
	"context"
	"strconv"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/config"
	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/queue"
)

// Scheduler handles scheduled/cron jobs
type Scheduler struct {
	cron        *cron.Cron
	asynqClient *asynq.Client
	cfg         *config.Config
	db          *database.DB
}

// NewScheduler creates a new scheduler
func NewScheduler(db *database.DB, redisOpt asynq.RedisClientOpt, cfg *config.Config) *Scheduler {
	asynqClient := asynq.NewClient(redisOpt)

	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()),
		asynqClient: asynqClient,
		cfg:         cfg,
		db:          db,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	log.Info().Msg("Starting scheduler")

	queueManager := queue.NewManager(s.asynqClient)
	hytaleRefresher := NewHytaleRefresher(s.db, s.cfg.HytaleUseStaging)

	// Auto-sync job (if enabled)
	if s.cfg.AutoSyncEnabled {
		interval := s.cfg.AutoSyncInterval
		if interval < 1 {
			interval = 1 // Minimum 1 second
		}

		// Config stores interval in seconds (e.g. 60 = 60 seconds, 3600 = 1 hour, 86400 = 24 hours)
		cronSpec := "@every " + strconv.Itoa(interval) + "s"
		_, err := s.cron.AddFunc(cronSpec, func() {
			log.Info().Msg("Triggering scheduled auto-sync")

			// Create sync log and enqueue task
			// Note: In production, this should create a sync log first
			_, err := queueManager.EnqueueSyncFull(queue.SyncFullPayload{
				SyncLogID:   "auto-" + strconv.Itoa(interval) + "s",
				RequestedBy: "scheduler",
			})
			if err != nil {
				log.Error().Err(err).Msg("Failed to enqueue auto-sync")
			}
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to schedule auto-sync job")
		} else {
			log.Info().Int("interval_seconds", interval).Msg("Scheduled auto-sync job")
		}
	}

	// OAuth token refresh every 5 minutes
	_, err := s.cron.AddFunc("@every 5m", func() {
		log.Debug().Msg("Running OAuth token refresh")
		if err := hytaleRefresher.RefreshOAuthTokens(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to refresh OAuth tokens")
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule OAuth token refresh")
	} else {
		log.Info().Msg("Scheduled OAuth token refresh (every 5 minutes)")
	}

	// Game session refresh every 10 minutes
	_, err = s.cron.AddFunc("@every 10m", func() {
		log.Debug().Msg("Running game session refresh")
		if err := hytaleRefresher.RefreshGameSessions(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to refresh game sessions")
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule game session refresh")
	} else {
		log.Info().Msg("Scheduled game session refresh (every 10 minutes)")
	}

	// Game session cleanup daily at 2 AM
	_, err = s.cron.AddFunc("0 0 2 * * *", func() {
		log.Debug().Msg("Running game session cleanup")
		if err := hytaleRefresher.CleanupExpiredSessions(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to cleanup expired game sessions")
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule game session cleanup")
	} else {
		log.Info().Msg("Scheduled game session cleanup (daily at 2 AM)")
	}

	// Daily log cleanup at 3 AM
	_, err = s.cron.AddFunc("0 0 3 * * *", func() {
		log.Info().Msg("Triggering daily log cleanup")
		_, err := queueManager.EnqueueCleanupLogs(30) // Keep 30 days
		if err != nil {
			log.Error().Err(err).Msg("Failed to enqueue log cleanup")
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule log cleanup job")
	}

	// Health check every minute (for monitoring)
	_, err = s.cron.AddFunc("@every 1m", func() {
		log.Debug().Msg("Scheduler health check")
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule health check")
	}

	s.cron.Start()
	log.Info().Int("jobs", len(s.cron.Entries())).Msg("Scheduler started")

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Info().Msg("Stopping scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.asynqClient.Close()
}
