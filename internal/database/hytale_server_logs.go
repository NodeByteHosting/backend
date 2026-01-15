package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// HytaleServerLog represents a stored game server log entry
type HytaleServerLog struct {
	ID           int64      `db:"id"`
	ServerUUID   string     `db:"server_uuid"`
	AccountID    string     `db:"account_id"`
	LogLine      string     `db:"log_line"`
	LogTimestamp *time.Time `db:"log_timestamp"`
	CreatedAt    time.Time  `db:"created_at"`
}

// HytaleLogSyncState tracks synchronization state for log persistence
type HytaleLogSyncState struct {
	ID           int       `db:"id"`
	ServerUUID   string    `db:"server_uuid"`
	LastSyncTime time.Time `db:"last_sync_time"`
	LastLineID   int64     `db:"last_line_id"`
	SyncStatus   string    `db:"sync_status"`
	ErrorMessage *string   `db:"error_message"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// HytaleServerLogsRepository provides database operations for Hytale server logs
type HytaleServerLogsRepository struct {
	db *DB
}

// NewHytaleServerLogsRepository creates a new repository instance
func NewHytaleServerLogsRepository(db *DB) *HytaleServerLogsRepository {
	return &HytaleServerLogsRepository{db: db}
}

// SaveLogs stores a batch of server logs
func (r *HytaleServerLogsRepository) SaveLogs(ctx context.Context, serverUUID, accountID string, logs []string) error {
	if len(logs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, log := range logs {
		batch.Queue(
			`INSERT INTO hytale_server_logs (server_uuid, account_id, log_line, log_timestamp)
			 VALUES ($1, $2, $3, CURRENT_TIMESTAMP)`,
			serverUUID, accountID, log,
		)
	}

	br := r.db.Pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to save log: %w", err)
		}
	}

	return nil
}

// GetLogsByServer retrieves logs for a specific server
func (r *HytaleServerLogsRepository) GetLogsByServer(ctx context.Context, serverUUID string, limit int, offset int) ([]*HytaleServerLog, error) {
	query := `
		SELECT id, server_uuid, account_id, log_line, log_timestamp, created_at
		FROM hytale_server_logs
		WHERE server_uuid = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, serverUUID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*HytaleServerLog
	for rows.Next() {
		log := &HytaleServerLog{}
		if err := rows.Scan(&log.ID, &log.ServerUUID, &log.AccountID, &log.LogLine, &log.LogTimestamp, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan log row: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning logs: %w", err)
	}

	return logs, nil
}

// GetLogsAfterTime retrieves logs created after a specific time
func (r *HytaleServerLogsRepository) GetLogsAfterTime(ctx context.Context, serverUUID string, after time.Time, limit int) ([]*HytaleServerLog, error) {
	query := `
		SELECT id, server_uuid, account_id, log_line, log_timestamp, created_at
		FROM hytale_server_logs
		WHERE server_uuid = $1 AND created_at > $2
		ORDER BY created_at ASC
		LIMIT $3
	`

	rows, err := r.db.Pool.Query(ctx, query, serverUUID, after, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*HytaleServerLog
	for rows.Next() {
		log := &HytaleServerLog{}
		if err := rows.Scan(&log.ID, &log.ServerUUID, &log.AccountID, &log.LogLine, &log.LogTimestamp, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan log row: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning logs: %w", err)
	}

	return logs, nil
}

// DeleteOldLogs removes logs older than the specified time
func (r *HytaleServerLogsRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.db.Pool.Exec(ctx,
		`DELETE FROM hytale_server_logs WHERE created_at < $1`, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", err)
	}

	return result.RowsAffected(), nil
}

// CountLogsByServer returns the total count of logs for a server
func (r *HytaleServerLogsRepository) CountLogsByServer(ctx context.Context, serverUUID string) (int64, error) {
	var count int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM hytale_server_logs WHERE server_uuid = $1`, serverUUID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count logs: %w", err)
	}
	return count, nil
}

// GetOrCreateSyncState retrieves or creates sync state for a server
func (r *HytaleServerLogsRepository) GetOrCreateSyncState(ctx context.Context, serverUUID string) (*HytaleLogSyncState, error) {
	state := &HytaleLogSyncState{}
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, server_uuid, last_sync_time, last_line_id, sync_status, error_message, updated_at
		 FROM hytale_log_sync_state WHERE server_uuid = $1`,
		serverUUID).Scan(&state.ID, &state.ServerUUID, &state.LastSyncTime, &state.LastLineID,
		&state.SyncStatus, &state.ErrorMessage, &state.UpdatedAt)

	if err == pgx.ErrNoRows {
		// Create new sync state
		err = r.db.Pool.QueryRow(ctx,
			`INSERT INTO hytale_log_sync_state (server_uuid, last_sync_time, sync_status)
			 VALUES ($1, CURRENT_TIMESTAMP, 'pending')
			 RETURNING id, server_uuid, last_sync_time, last_line_id, sync_status, error_message, updated_at`,
			serverUUID).Scan(&state.ID, &state.ServerUUID, &state.LastSyncTime, &state.LastLineID,
			&state.SyncStatus, &state.ErrorMessage, &state.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to create sync state: %w", err)
		}
		return state, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query sync state: %w", err)
	}

	return state, nil
}

// UpdateSyncState updates the sync state for a server
func (r *HytaleServerLogsRepository) UpdateSyncState(ctx context.Context, serverUUID string, status string, errorMsg *string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE hytale_log_sync_state 
		 SET last_sync_time = CURRENT_TIMESTAMP, sync_status = $1, error_message = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE server_uuid = $3`,
		status, errorMsg, serverUUID)

	if err != nil {
		return fmt.Errorf("failed to update sync state: %w", err)
	}

	return nil
}
