package database

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// SyncRepository handles sync log database operations
type SyncRepository struct {
	db *DB
}

// NewSyncRepository creates a new sync repository
func NewSyncRepository(db *DB) *SyncRepository {
	return &SyncRepository{db: db}
}

// CreateSyncLog creates a new sync log entry
func (r *SyncRepository) CreateSyncLog(ctx context.Context, syncType, status string, metadata map[string]interface{}) (*SyncLog, error) {
	var metadataJSON []byte
	if metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
	}

	syncLog := &SyncLog{
		ID:        uuid.New().String(),
		Type:      syncType,
		Status:    status,
		StartedAt: time.Now(),
	}

	query := `
		INSERT INTO sync_logs (id, type, status, metadata, "startedAt")
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Pool.Exec(ctx, query, syncLog.ID, syncLog.Type, syncLog.Status, string(metadataJSON), syncLog.StartedAt)
	if err != nil {
		return nil, err
	}

	return syncLog, nil
}

// UpdateSyncLog updates a sync log entry
func (r *SyncRepository) UpdateSyncLog(ctx context.Context, syncLogID, status string, itemsTotal, itemsSynced, itemsFailed *int, metadata map[string]interface{}) error {
	var metadataJSON []byte
	if metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return err
		}
	}

	query := `UPDATE sync_logs SET status = $2`
	args := []interface{}{syncLogID, status}

	if itemsTotal != nil {
		query += `, "itemsTotal" = $` + strconv.Itoa(len(args)+1)
		args = append(args, *itemsTotal)
	}

	if itemsSynced != nil {
		query += `, "itemsSynced" = $` + strconv.Itoa(len(args)+1)
		args = append(args, *itemsSynced)
	}

	if itemsFailed != nil {
		query += `, "itemsFailed" = $` + strconv.Itoa(len(args)+1)
		args = append(args, *itemsFailed)
	}

	if len(metadataJSON) > 0 {
		query += `, metadata = $` + strconv.Itoa(len(args)+1)
		args = append(args, string(metadataJSON))
	}

	query += ` WHERE id = $1`

	_, err := r.db.Pool.Exec(ctx, query, args...)
	return err
}

// GetSyncLogs retrieves sync logs with pagination
func (r *SyncRepository) GetSyncLogs(ctx context.Context, limit int, offset int, syncType string) ([]SyncLog, error) {
	query := `SELECT id, type, status, "itemsTotal", "itemsSynced", "itemsFailed", error, metadata, "startedAt", "completedAt" FROM sync_logs`
	args := []interface{}{}

	if syncType != "" {
		query += ` WHERE type = $1`
		args = append(args, syncType)
	}

	paramCount := len(args) + 1
	query += ` ORDER BY "startedAt" DESC LIMIT $` + strconv.Itoa(paramCount) + ` OFFSET $` + strconv.Itoa(paramCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []SyncLog
	for rows.Next() {
		var log SyncLog
		err := rows.Scan(&log.ID, &log.Type, &log.Status, &log.ItemsTotal, &log.ItemsSynced, &log.ItemsFailed, &log.Error, &log.Metadata, &log.StartedAt, &log.CompletedAt)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetSyncLog retrieves a specific sync log by ID
func (r *SyncRepository) GetSyncLog(ctx context.Context, syncLogID string) (*SyncLog, error) {
	var log SyncLog
	query := `SELECT id, type, status, "itemsTotal", "itemsSynced", "itemsFailed", error, metadata, "startedAt", "completedAt" FROM sync_logs WHERE id = $1`

	err := r.db.Pool.QueryRow(ctx, query, syncLogID).Scan(
		&log.ID, &log.Type, &log.Status, &log.ItemsTotal, &log.ItemsSynced, &log.ItemsFailed, &log.Error, &log.Metadata, &log.StartedAt, &log.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	return &log, nil
}

// IsSyncCancelled checks if a sync has been marked for cancellation
func (r *SyncRepository) IsSyncCancelled(ctx context.Context, syncLogID string) (bool, error) {
	var cancelledAt *time.Time
	query := `SELECT "cancelledAt" FROM sync_logs WHERE id = $1`

	err := r.db.Pool.QueryRow(ctx, query, syncLogID).Scan(&cancelledAt)
	if err != nil {
		return false, err
	}

	return cancelledAt != nil, nil
}

// MarkSyncCancelled marks a sync for cancellation
func (r *SyncRepository) MarkSyncCancelled(ctx context.Context, syncLogID string) error {
	query := `UPDATE sync_logs SET "cancelledAt" = $1 WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, time.Now(), syncLogID)
	return err
}
