package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"
)

// AuditLogType represents the type of audit log event
type AuditLogType string

const (
	AuditTokenCreated     AuditLogType = "TOKEN_CREATED"
	AuditTokenRefreshed   AuditLogType = "TOKEN_REFRESHED"
	AuditTokenDeleted     AuditLogType = "TOKEN_DELETED"
	AuditAuthFailed       AuditLogType = "AUTH_FAILED"
	AuditSessionCreated   AuditLogType = "SESSION_CREATED"
	AuditSessionRefreshed AuditLogType = "SESSION_REFRESHED"
	AuditSessionDeleted   AuditLogType = "SESSION_DELETED"
	AuditProfileSelected  AuditLogType = "PROFILE_SELECTED"
)

// HytaleAuditLog represents an audit log entry for Hytale operations
type HytaleAuditLog struct {
	ID        string
	AccountID string  // Hytale account UUID
	ProfileID *string // Game profile UUID (optional)
	EventType AuditLogType
	Details   *string // JSON details about the event
	IPAddress *string // IP address of the request origin
	UserAgent *string // User agent string
	CreatedAt time.Time
}

// HytaleAuditLogRepository handles audit log database operations
type HytaleAuditLogRepository struct {
	db *DB
}

// NewHytaleAuditLogRepository creates a new audit log repository
func NewHytaleAuditLogRepository(db *DB) *HytaleAuditLogRepository {
	return &HytaleAuditLogRepository{db: db}
}

// LogTokenCreated logs a token creation event
func (r *HytaleAuditLogRepository) LogTokenCreated(ctx context.Context, accountID string, profileID *string, ipAddress *string) error {
	query := `
		INSERT INTO hytale_audit_logs (account_id, profile_id, event_type, ip_address, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.Pool.Exec(ctx, query, accountID, profileID, string(AuditTokenCreated), ipAddress)
	if err != nil {
		log.Error().
			Err(err).
			Str("account_id", accountID).
			Str("event", string(AuditTokenCreated)).
			Msg("Failed to log token creation")
		return err
	}

	log.Info().
		Str("account_id", accountID).
		Str("event", string(AuditTokenCreated)).
		Msg("Token creation logged")

	return nil
}

// LogTokenRefreshed logs a token refresh event
func (r *HytaleAuditLogRepository) LogTokenRefreshed(ctx context.Context, accountID string, profileID *string, ipAddress *string) error {
	query := `
		INSERT INTO hytale_audit_logs (account_id, profile_id, event_type, ip_address, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.Pool.Exec(ctx, query, accountID, profileID, string(AuditTokenRefreshed), ipAddress)
	if err != nil {
		log.Error().
			Err(err).
			Str("account_id", accountID).
			Str("event", string(AuditTokenRefreshed)).
			Msg("Failed to log token refresh")
		return err
	}

	log.Debug().
		Str("account_id", accountID).
		Str("event", string(AuditTokenRefreshed)).
		Msg("Token refresh logged")

	return nil
}

// LogTokenDeleted logs a token deletion event
func (r *HytaleAuditLogRepository) LogTokenDeleted(ctx context.Context, accountID string, ipAddress *string) error {
	query := `
		INSERT INTO hytale_audit_logs (account_id, event_type, ip_address, created_at)
		VALUES ($1, $2, $3, NOW())
	`

	_, err := r.db.Pool.Exec(ctx, query, accountID, string(AuditTokenDeleted), ipAddress)
	if err != nil {
		log.Error().
			Err(err).
			Str("account_id", accountID).
			Str("event", string(AuditTokenDeleted)).
			Msg("Failed to log token deletion")
		return err
	}

	log.Info().
		Str("account_id", accountID).
		Str("event", string(AuditTokenDeleted)).
		Msg("Token deletion logged")

	return nil
}

// LogSessionCreated logs a game session creation event
func (r *HytaleAuditLogRepository) LogSessionCreated(ctx context.Context, accountID string, profileID string, ipAddress *string) error {
	query := `
		INSERT INTO hytale_audit_logs (account_id, profile_id, event_type, ip_address, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.Pool.Exec(ctx, query, accountID, profileID, string(AuditSessionCreated), ipAddress)
	if err != nil {
		log.Error().
			Err(err).
			Str("account_id", accountID).
			Str("profile_id", profileID).
			Str("event", string(AuditSessionCreated)).
			Msg("Failed to log session creation")
		return err
	}

	log.Info().
		Str("account_id", accountID).
		Str("profile_id", profileID).
		Str("event", string(AuditSessionCreated)).
		Msg("Session creation logged")

	return nil
}

// LogSessionRefreshed logs a game session refresh event
func (r *HytaleAuditLogRepository) LogSessionRefreshed(ctx context.Context, accountID string, profileID string, ipAddress *string) error {
	query := `
		INSERT INTO hytale_audit_logs (account_id, profile_id, event_type, ip_address, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.Pool.Exec(ctx, query, accountID, profileID, string(AuditSessionRefreshed), ipAddress)
	if err != nil {
		log.Error().
			Err(err).
			Str("account_id", accountID).
			Str("profile_id", profileID).
			Str("event", string(AuditSessionRefreshed)).
			Msg("Failed to log session refresh")
		return err
	}

	log.Debug().
		Str("account_id", accountID).
		Str("profile_id", profileID).
		Str("event", string(AuditSessionRefreshed)).
		Msg("Session refresh logged")

	return nil
}

// LogSessionDeleted logs a game session deletion event
func (r *HytaleAuditLogRepository) LogSessionDeleted(ctx context.Context, accountID string, profileID string, ipAddress *string) error {
	query := `
		INSERT INTO hytale_audit_logs (account_id, profile_id, event_type, ip_address, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.Pool.Exec(ctx, query, accountID, profileID, string(AuditSessionDeleted), ipAddress)
	if err != nil {
		log.Error().
			Err(err).
			Str("account_id", accountID).
			Str("profile_id", profileID).
			Str("event", string(AuditSessionDeleted)).
			Msg("Failed to log session deletion")
		return err
	}

	log.Info().
		Str("account_id", accountID).
		Str("profile_id", profileID).
		Str("event", string(AuditSessionDeleted)).
		Msg("Session deletion logged")

	return nil
}

// LogAuthFailure logs an authentication failure
func (r *HytaleAuditLogRepository) LogAuthFailure(ctx context.Context, accountID string, reason string, ipAddress *string) error {
	query := `
		INSERT INTO hytale_audit_logs (account_id, event_type, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.Pool.Exec(ctx, query, accountID, string(AuditAuthFailed), reason, ipAddress)
	if err != nil {
		log.Error().
			Err(err).
			Str("account_id", accountID).
			Str("reason", reason).
			Str("event", string(AuditAuthFailed)).
			Msg("Failed to log auth failure")
		return err
	}

	log.Warn().
		Str("account_id", accountID).
		Str("reason", reason).
		Str("event", string(AuditAuthFailed)).
		Msg("Auth failure logged")

	return nil
}

// GetAuditLogs retrieves audit logs for an account
func (r *HytaleAuditLogRepository) GetAuditLogs(ctx context.Context, accountID string, limit int) ([]HytaleAuditLog, error) {
	query := `
		SELECT id, account_id, profile_id, event_type, details, ip_address, user_agent, created_at
		FROM hytale_audit_logs
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, accountID, limit)
	if err != nil {
		log.Error().
			Err(err).
			Str("account_id", accountID).
			Msg("Failed to query audit logs")
		return nil, err
	}
	defer rows.Close()

	var logs []HytaleAuditLog
	for rows.Next() {
		var log HytaleAuditLog
		if err := rows.Scan(
			&log.ID,
			&log.AccountID,
			&log.ProfileID,
			&log.EventType,
			&log.Details,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetLatestAuditLog gets the most recent audit log for an account
func (r *HytaleAuditLogRepository) GetLatestAuditLog(ctx context.Context, accountID string) (*HytaleAuditLog, error) {
	query := `
		SELECT id, account_id, profile_id, event_type, details, ip_address, user_agent, created_at
		FROM hytale_audit_logs
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var log HytaleAuditLog
	err := r.db.Pool.QueryRow(ctx, query, accountID).Scan(
		&log.ID,
		&log.AccountID,
		&log.ProfileID,
		&log.EventType,
		&log.Details,
		&log.IPAddress,
		&log.UserAgent,
		&log.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &log, nil
}
