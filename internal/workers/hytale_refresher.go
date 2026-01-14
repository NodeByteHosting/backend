package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/hytale"
)

// HytaleRefresher handles Hytale token and session refresh operations
type HytaleRefresher struct {
	db          *database.DB
	oauthRepo   *database.HytaleOAuthRepository
	oauthClient *hytale.OAuthClient
}

// NewHytaleRefresher creates a new Hytale refresher
func NewHytaleRefresher(db *database.DB, useStaging bool) *HytaleRefresher {
	oauthClient := hytale.NewOAuthClient(&hytale.OAuthClientConfig{
		ClientID:   "hytale-server",
		UseStaging: useStaging,
	})

	return &HytaleRefresher{
		db:          db,
		oauthRepo:   database.NewHytaleOAuthRepository(db),
		oauthClient: oauthClient,
	}
}

// RefreshOAuthTokens refreshes all OAuth tokens that are expiring soon
// Called by scheduler every 5 minutes
func (r *HytaleRefresher) RefreshOAuthTokens(ctx context.Context) error {
	log.Debug().Msg("Starting OAuth token refresh check")

	// Get all tokens from database
	tokens, err := r.oauthRepo.GetAllOAuthTokens(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch OAuth tokens for refresh")
		return err
	}

	if len(tokens) == 0 {
		log.Debug().Msg("No OAuth tokens found to refresh")
		return nil
	}

	log.Debug().Int("token_count", len(tokens)).Msg("Checking OAuth tokens for refresh")

	// Refresh tokens expiring in next 5 minutes
	refreshThreshold := time.Now().Add(5 * time.Minute)

	for _, token := range tokens {
		if token.AccessTokenExpiry.Before(refreshThreshold) {
			log.Info().
				Str("account_id", token.AccountID).
				Time("expiry", token.AccessTokenExpiry).
				Msg("Refreshing OAuth token")

			if err := r.refreshSingleToken(ctx, token); err != nil {
				log.Error().
					Err(err).
					Str("account_id", token.AccountID).
					Msg("Failed to refresh OAuth token")
				// Continue refreshing other tokens
				continue
			}
		}
	}

	return nil
}

// refreshSingleToken refreshes a single OAuth token
func (r *HytaleRefresher) refreshSingleToken(ctx context.Context, token *database.HytaleOAuthToken) error {
	// Refresh token with Hytale
	tokenResp, err := r.oauthClient.RefreshToken(ctx, token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Check for errors
	if tokenResp.Error != "" {
		return fmt.Errorf("hytale returned error: %s - %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	// Update token in database
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	token.AccessToken = tokenResp.AccessToken
	token.RefreshToken = tokenResp.RefreshToken
	token.AccessTokenExpiry = expiresAt
	token.Scope = tokenResp.Scope

	if err := r.oauthRepo.SaveOAuthToken(ctx, token); err != nil {
		return fmt.Errorf("failed to save updated token: %w", err)
	}

	log.Info().
		Str("account_id", token.AccountID).
		Time("new_expiry", expiresAt).
		Msg("OAuth token refreshed successfully")

	return nil
}

// RefreshGameSessions refreshes all game sessions that are expiring soon
// Called by scheduler every 10 minutes
func (r *HytaleRefresher) RefreshGameSessions(ctx context.Context) error {
	log.Debug().Msg("Starting game session refresh check")

	// Get all game sessions from database
	sessions, err := r.oauthRepo.GetAllGameSessions(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch game sessions for refresh")
		return err
	}

	if len(sessions) == 0 {
		log.Debug().Msg("No game sessions found to refresh")
		return nil
	}

	log.Debug().Int("session_count", len(sessions)).Msg("Checking game sessions for refresh")

	// Refresh sessions expiring in next 5 minutes
	// Game sessions expire in 1 hour, so refresh at 55 minute mark
	now := time.Now()

	for _, session := range sessions {
		// Check if session is about to expire (created_at + 55 minutes >= now)
		createdAt := session.CreatedAt
		expiryTime := createdAt.Add(55 * time.Minute)

		if expiryTime.Before(now) || expiryTime.Equal(now) {
			log.Info().
				Str("account_id", session.AccountID).
				Str("profile_uuid", session.ProfileUUID).
				Msg("Refreshing game session")

			if err := r.refreshSingleSession(ctx, session); err != nil {
				log.Error().
					Err(err).
					Str("account_id", session.AccountID).
					Str("profile_uuid", session.ProfileUUID).
					Msg("Failed to refresh game session")
				// Continue refreshing other sessions
				continue
			}
		}
	}

	return nil
}

// refreshSingleSession refreshes a single game session
func (r *HytaleRefresher) refreshSingleSession(ctx context.Context, session *database.HytaleGameSession) error {
	// Refresh session with Hytale
	if err := r.oauthClient.RefreshGameSession(ctx, session.SessionToken); err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	// Update the refresh timestamp in database
	if err := r.oauthRepo.UpdateGameSessionRefresh(ctx, session.AccountID, session.ProfileUUID); err != nil {
		return fmt.Errorf("failed to update session refresh time: %w", err)
	}

	log.Info().
		Str("account_id", session.AccountID).
		Str("profile_uuid", session.ProfileUUID).
		Msg("Game session refreshed successfully")

	return nil
}

// CleanupExpiredSessions removes game sessions that have been inactive for 2 hours
// Called by scheduler daily at 2 AM
func (r *HytaleRefresher) CleanupExpiredSessions(ctx context.Context) error {
	log.Debug().Msg("Starting expired game session cleanup")

	// Get all game sessions
	sessions, err := r.oauthRepo.GetAllGameSessions(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch game sessions for cleanup")
		return err
	}

	if len(sessions) == 0 {
		log.Debug().Msg("No game sessions found for cleanup")
		return nil
	}

	now := time.Now()
	inactiveThreshold := 2 * time.Hour
	deletedCount := 0

	for _, session := range sessions {
		// Check if session hasn't been refreshed in 2 hours
		lastRefresh := session.UpdatedAt
		if now.Sub(lastRefresh) > inactiveThreshold {
			log.Info().
				Str("account_id", session.AccountID).
				Str("profile_uuid", session.ProfileUUID).
				Time("last_refresh", lastRefresh).
				Msg("Deleting inactive game session")

			// Try to terminate with Hytale first
			if err := r.oauthClient.TerminateGameSession(ctx, session.SessionToken); err != nil {
				log.Warn().
					Err(err).
					Str("account_id", session.AccountID).
					Str("profile_uuid", session.ProfileUUID).
					Msg("Failed to terminate session with Hytale, continuing with local deletion")
			}

			// Delete from database
			if err := r.oauthRepo.DeleteGameSession(ctx, session.AccountID, session.ProfileUUID); err != nil {
				log.Error().
					Err(err).
					Str("account_id", session.AccountID).
					Str("profile_uuid", session.ProfileUUID).
					Msg("Failed to delete inactive game session")
				continue
			}

			deletedCount++
		}
	}

	if deletedCount > 0 {
		log.Info().Int("deleted_count", deletedCount).Msg("Game session cleanup completed")
	} else {
		log.Debug().Msg("No inactive game sessions found for cleanup")
	}

	return nil
}
