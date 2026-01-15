package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/hytale"
	"github.com/nodebyte/backend/internal/sentry"
)

// HytaleLogPersister handles fetching logs from Hytale and storing them persistently
type HytaleLogPersister struct {
	db          *database.DB
	logsRepo    *database.HytaleServerLogsRepository
	oauthRepo   *database.HytaleOAuthRepository
	oauthClient *hytale.OAuthClient
	useStaging  bool
}

// NewHytaleLogPersister creates a new log persister instance
func NewHytaleLogPersister(db *database.DB, useStaging bool) *HytaleLogPersister {
	oauthClient := hytale.NewOAuthClient(&hytale.OAuthClientConfig{
		ClientID:   "hytale-server",
		UseStaging: useStaging,
	})

	return &HytaleLogPersister{
		db:          db,
		logsRepo:    database.NewHytaleServerLogsRepository(db),
		oauthRepo:   database.NewHytaleOAuthRepository(db),
		oauthClient: oauthClient,
		useStaging:  useStaging,
	}
}

// PersistGameServerLogs is currently disabled - logs are sent directly from Wings via POST endpoint
// This function is kept for backward compatibility but is not called by the scheduler
func (p *HytaleLogPersister) PersistGameServerLogs(ctx context.Context) error {
	log.Debug().Msg("Hytale log persistence is now handled by Wings daemon posting to /api/v1/hytale/server-logs endpoint")
	return nil
}

// persistSessionLogs is now disabled - logs are sent directly from Wings
func (p *HytaleLogPersister) persistSessionLogs(ctx context.Context, session *database.HytaleGameSession) error {
	// No longer needed - Wings handles log submission
	return nil
}

// getLogsFromHytale is no longer used - Hytale doesn't provide a logs API
// Logs are now submitted directly from Wings daemon via POST /api/v1/hytale/server-logs
func (p *HytaleLogPersister) getLogsFromHytale(ctx context.Context, session *database.HytaleGameSession, since time.Time) ([]string, error) {
	return []string{}, fmt.Errorf("log fetching from Hytale API is not supported - use Wings log submission instead")
}

// CleanupOldLogs removes game server logs older than the retention period
// Called by scheduler daily
func (p *HytaleLogPersister) CleanupOldLogs(ctx context.Context, retentionDays int) error {
	tx := sentry.StartBackgroundTransaction(ctx, "worker.cleanup_hytale_logs")
	defer tx.Finish()
	ctx = tx.Context()

	log.Debug().Int("retention_days", retentionDays).Msg("Starting Hytale log cleanup")

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := p.logsRepo.DeleteOldLogs(ctx, cutoffTime)
	if err != nil {
		sentry.CaptureExceptionWithContext(ctx, err, "cleanup_logs")
		return err
	}

	log.Info().
		Int64("deleted_count", deleted).
		Time("older_than", cutoffTime).
		Msg("Hytale log cleanup completed")

	return nil
}
