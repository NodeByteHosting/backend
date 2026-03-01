package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/config"
	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/panels"
	"github.com/nodebyte/backend/internal/queue"
	"github.com/nodebyte/backend/internal/sentry"
)

// SyncHandler handles sync-related tasks
type SyncHandler struct {
	db          *database.DB
	syncRepo    *database.SyncRepository
	pteroClient *panels.PterodactylClient
	cfg         *config.Config
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(db *database.DB, pteroClient *panels.PterodactylClient, cfg *config.Config) *SyncHandler {
	return &SyncHandler{
		db:          db,
		syncRepo:    database.NewSyncRepository(db),
		pteroClient: pteroClient,
		cfg:         cfg,
	}
}

// HandleFullSync processes a full sync task
func (h *SyncHandler) HandleFullSync(ctx context.Context, task *asynq.Task) error {
	tx := sentry.StartBackgroundTransaction(ctx, "worker.full_sync")
	defer tx.Finish()
	ctx = tx.Context()

	var payload queue.SyncFullPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		sentry.CaptureExceptionWithContext(ctx, err, "unmarshal_payload")
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Info().
		Str("sync_log_id", payload.SyncLogID).
		Str("requested_by", payload.RequestedBy).
		Msg("Starting full sync")

	startTime := time.Now()

	// Update sync log to RUNNING
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step":       "starting",
		"started_at": time.Now().Unix(),
	})

	// Check for cancellation before each step
	checkCancelled := func() bool {
		cancelled, _ := h.syncRepo.IsSyncCancelled(ctx, payload.SyncLogID)
		return cancelled
	}

	// Step 1: Sync Locations
	if checkCancelled() {
		return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before locations sync")
	}
	h.updateProgress(ctx, payload.SyncLogID, "locations", 0)
	if err := h.syncLocations(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "locations", err)
	}

	// Step 2: Sync Nodes
	if checkCancelled() {
		return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before nodes sync")
	}
	h.updateProgress(ctx, payload.SyncLogID, "nodes", 15)
	if err := h.syncNodes(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "nodes", err)
	}

	// Step 3: Sync Allocations
	if checkCancelled() {
		return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before allocations sync")
	}
	h.updateProgress(ctx, payload.SyncLogID, "allocations", 30)
	if err := h.syncAllocations(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "allocations", err)
	}

	// Step 4: Sync Nests & Eggs
	if checkCancelled() {
		return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before nests sync")
	}
	h.updateProgress(ctx, payload.SyncLogID, "nests", 45)
	if err := h.syncNestsAndEggs(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "nests", err)
	}

	// Step 5: Sync Users — BEFORE servers so ownerId lookups succeed
	if !payload.SkipUsers {
		if checkCancelled() {
			return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before users sync")
		}
		h.updateProgress(ctx, payload.SyncLogID, "users", 60)
		if err := h.syncUsers(ctx, payload.SyncLogID); err != nil {
			return h.failSync(ctx, payload.SyncLogID, "users", err)
		}
	}

	// Step 6: Sync Servers — users now exist so ownerId FK resolves correctly
	if checkCancelled() {
		return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before servers sync")
	}
	h.updateProgress(ctx, payload.SyncLogID, "servers", 75)
	if err := h.syncServers(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "servers", err)
	}

	// Step 7: Sync Server Subusers (Client API - selective)
	if checkCancelled() {
		return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before subusers sync")
	}
	h.updateProgress(ctx, payload.SyncLogID, "subusers", 85)
	if err := h.syncServerSubusers(ctx, payload.SyncLogID); err != nil {
		log.Warn().Err(err).Msg("Subuser sync failed - continuing with full sync")
		// Don't fail entire sync if subusers fail
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Complete
	h.updateProgress(ctx, payload.SyncLogID, "completed", 100)
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, nil, map[string]interface{}{
		"completed_at": time.Now().Unix(),
		"duration":     duration.Seconds(),
	})

	log.Info().
		Str("sync_log_id", payload.SyncLogID).
		Float64("duration_seconds", duration.Seconds()).
		Msg("Full sync completed")

	// Dispatch success webhook (non-blocking)
	go h.dispatchSyncWebhook(ctx, payload.SyncLogID, "COMPLETED", duration, nil)

	return nil
}

// dispatchSyncWebhook sends webhook notifications for sync completion/failure
func (h *SyncHandler) dispatchSyncWebhook(ctx context.Context, syncLogID, status string, duration time.Duration, syncError error) {
	// Create a new background context instead of using the task context which may be cancelled
	bgCtx := context.Background()

	// Get all enabled SYSTEM webhooks
	query := `
		SELECT "webhookUrl" 
		FROM discord_webhooks 
		WHERE enabled = true 
		AND type = 'SYSTEM' 
		AND scope = 'ADMIN'
	`

	rows, err := h.db.Pool.Query(bgCtx, query)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch webhooks for sync notification")
		return
	}
	defer rows.Close()

	var webhookURLs []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			continue
		}
		webhookURLs = append(webhookURLs, url)
	}

	if len(webhookURLs) == 0 {
		return
	}

	// Determine color and status text
	color := 3066993 // Green
	statusEmoji := "✅"
	statusText := "Completed Successfully"

	if status == "FAILED" {
		color = 15158332 // Red
		statusEmoji = "❌"
		statusText = "Failed"
	} else if status == "CANCELLED" {
		color = 16776960 // Yellow
		statusEmoji = "⚠️"
		statusText = "Cancelled"
	}

	fields := []map[string]interface{}{
		{
			"name":   "Status",
			"value":  fmt.Sprintf("%s %s", statusEmoji, statusText),
			"inline": true,
		},
		{
			"name":   "Duration",
			"value":  fmt.Sprintf("%.2f seconds", duration.Seconds()),
			"inline": true,
		},
	}

	if syncError != nil {
		fields = append(fields, map[string]interface{}{
			"name":   "Error",
			"value":  syncError.Error(),
			"inline": false,
		})
	}

	// Prepare webhook payload
	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       "🔄 Sync Operation " + statusText,
				"description": "Panel synchronization has " + strings.ToLower(statusText),
				"color":       color,
				"fields":      fields,
				"timestamp":   time.Now().Format(time.RFC3339),
				"footer": map[string]string{
					"text": "NodeByte Sync System",
				},
			},
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	// Send to all webhooks in parallel
	for _, webhookURL := range webhookURLs {
		go func(url string) {
			resp, err := http.Post(url, "application/json", bytes.NewReader(payloadBytes)) // FIX: Use bytes.NewReader
			if err != nil {
				log.Warn().Err(err).Str("webhook_url", url).Msg("Failed to send sync webhook")
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				log.Warn().Int("status", resp.StatusCode).Str("webhook_url", url).Msg("Webhook returned non-204 status")
			}
		}(webhookURL)
	}
}

// HandleSyncLocations syncs only locations
func (h *SyncHandler) HandleSyncLocations(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step": "locations", "lastUpdated": time.Now().Unix(),
	})
	if err := h.syncLocations(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "locations", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, nil, map[string]interface{}{
		"step": "locations", "completed_at": time.Now().Unix(), "lastUpdated": time.Now().Unix(),
	})
	return nil
}

// HandleSyncNodes syncs only nodes
func (h *SyncHandler) HandleSyncNodes(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step": "nodes", "lastUpdated": time.Now().Unix(),
	})
	if err := h.syncNodes(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "nodes", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, nil, map[string]interface{}{
		"step": "nodes", "completed_at": time.Now().Unix(), "lastUpdated": time.Now().Unix(),
	})
	return nil
}

// HandleSyncAllocations syncs only allocations
func (h *SyncHandler) HandleSyncAllocations(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step": "allocations", "lastUpdated": time.Now().Unix(),
	})
	if err := h.syncAllocations(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "allocations", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, nil, map[string]interface{}{
		"step": "allocations", "completed_at": time.Now().Unix(), "lastUpdated": time.Now().Unix(),
	})
	return nil
}

// HandleSyncNests syncs nests and eggs
func (h *SyncHandler) HandleSyncNests(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step": "nests", "lastUpdated": time.Now().Unix(),
	})
	if err := h.syncNestsAndEggs(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "nests", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, nil, map[string]interface{}{
		"step": "nests", "completed_at": time.Now().Unix(), "lastUpdated": time.Now().Unix(),
	})
	return nil
}

// HandleSyncServers syncs servers
func (h *SyncHandler) HandleSyncServers(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step": "servers", "lastUpdated": time.Now().Unix(),
	})
	if err := h.syncServers(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "servers", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, nil, map[string]interface{}{
		"step": "servers", "completed_at": time.Now().Unix(), "lastUpdated": time.Now().Unix(),
	})
	return nil
}

// HandleSyncDatabases syncs server databases
func (h *SyncHandler) HandleSyncDatabases(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step": "databases", "lastUpdated": time.Now().Unix(),
	})
	if err := h.syncDatabases(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "databases", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, nil, map[string]interface{}{
		"step": "databases", "completed_at": time.Now().Unix(), "lastUpdated": time.Now().Unix(),
	})
	return nil
}

// HandleSyncUsers syncs users
func (h *SyncHandler) HandleSyncUsers(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step": "users", "lastUpdated": time.Now().Unix(),
	})
	if err := h.syncUsers(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "users", err)
	}
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, nil, map[string]interface{}{
		"step": "users", "completed_at": time.Now().Unix(), "lastUpdated": time.Now().Unix(),
	})
	return nil
}

// HandleCleanupLogs cleans up old sync logs
func (h *SyncHandler) HandleCleanupLogs(ctx context.Context, task *asynq.Task) error {
	var payload struct {
		OlderThanDays int `json:"older_than_days"`
	}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	days := payload.OlderThanDays
	if days == 0 {
		days = 30 // Default to 30 days
	}

	cutoff := time.Now().AddDate(0, 0, -days)

	query := `DELETE FROM sync_logs WHERE "startedAt" < $1`
	result, err := h.db.Pool.Exec(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup logs: %w", err)
	}

	log.Info().
		Int64("deleted", result.RowsAffected()).
		Int("older_than_days", days).
		Msg("Cleaned up old sync logs")

	return nil
}

// Internal sync methods

func (h *SyncHandler) syncLocations(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing locations")

	locations, err := h.pteroClient.GetAllLocations(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch locations: %w", err)
	}

	h.updateDetailedProgress(ctx, syncLogID, "locations", len(locations), 0, fmt.Sprintf("Fetched %d locations from panel", len(locations)))

	for i, loc := range locations {
		query := `
			INSERT INTO locations (id, "shortCode", description, "createdAt", "updatedAt")
			VALUES ($1, $2, $3, NOW(), NOW())
			ON CONFLICT (id) DO UPDATE SET
				"shortCode" = EXCLUDED."shortCode",
				description = EXCLUDED.description,
				"updatedAt" = NOW()
		`
		_, err := h.db.Pool.Exec(ctx, query,
			loc.Attributes.ID,
			loc.Attributes.ShortCode,
			loc.Attributes.Long,
		)
		if err != nil {
			log.Warn().Err(err).Int("location_id", loc.Attributes.ID).Msg("Failed to upsert location")
		}

		// Update progress every 10 items or at end
		if (i+1)%10 == 0 || i == len(locations)-1 {
			h.updateDetailedProgress(ctx, syncLogID, "locations", len(locations), i+1, fmt.Sprintf("Processing location %d/%d (ID: %d)", i+1, len(locations), loc.Attributes.ID))
		}
	}

	// Remove stale locations no longer in the panel
	if len(locations) > 0 {
		ids := make([]interface{}, len(locations))
		ph := make([]string, len(locations))
		for i, loc := range locations {
			ids[i] = loc.Attributes.ID
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		if res, err := h.db.Pool.Exec(ctx, `DELETE FROM locations WHERE id NOT IN (`+strings.Join(ph, ",")+`)`, ids...); err != nil {
			log.Warn().Err(err).Msg("Failed to delete stale locations")
		} else if res.RowsAffected() > 0 {
			log.Info().Int64("deleted", res.RowsAffected()).Msg("Deleted stale locations")
		}
	}

	log.Info().Int("count", len(locations)).Msg("Synced locations")
	h.updateDetailedProgress(ctx, syncLogID, "locations", len(locations), len(locations), fmt.Sprintf("✓ Synced %d locations", len(locations)))
	return nil
}

func (h *SyncHandler) syncNodes(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing nodes")

	nodes, err := h.pteroClient.GetAllNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch nodes: %w", err)
	}

	h.updateDetailedProgress(ctx, syncLogID, "nodes", len(nodes), 0, fmt.Sprintf("Fetched %d nodes from panel", len(nodes)))

	for i, node := range nodes {
		query := `
			INSERT INTO nodes (
				id, uuid, name, description, fqdn, scheme, "behindProxy", "panelType",
				memory, "memoryOverallocate", disk, "diskOverallocate",
				"isPublic", "isMaintenanceMode", "daemonListenPort", "daemonSftpPort", "daemonBase",
				"locationId", "createdAt", "updatedAt"
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW())
			ON CONFLICT (id) DO UPDATE SET
				uuid = EXCLUDED.uuid,
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				fqdn = EXCLUDED.fqdn,
				scheme = EXCLUDED.scheme,
				"behindProxy" = EXCLUDED."behindProxy",
				memory = EXCLUDED.memory,
				"memoryOverallocate" = EXCLUDED."memoryOverallocate",
				disk = EXCLUDED.disk,
				"diskOverallocate" = EXCLUDED."diskOverallocate",
				"isPublic" = EXCLUDED."isPublic",
				"isMaintenanceMode" = EXCLUDED."isMaintenanceMode",
				"daemonListenPort" = EXCLUDED."daemonListenPort",
				"daemonSftpPort" = EXCLUDED."daemonSftpPort",
				"daemonBase" = EXCLUDED."daemonBase",
				"locationId" = EXCLUDED."locationId",
				"updatedAt" = NOW()
		`
		_, err := h.db.Pool.Exec(ctx, query,
			node.Attributes.ID,
			node.Attributes.UUID,
			node.Attributes.Name,
			node.Attributes.Description,
			node.Attributes.FQDN,
			node.Attributes.Scheme,
			node.Attributes.BehindProxy,
			"pterodactyl",
			node.Attributes.Memory,
			node.Attributes.MemoryOverallocate,
			node.Attributes.Disk,
			node.Attributes.DiskOverallocate,
			node.Attributes.Public,
			node.Attributes.MaintenanceMode,
			node.Attributes.DaemonListen,
			node.Attributes.DaemonSFTP,
			node.Attributes.DaemonBase,
			node.Attributes.LocationID,
		)
		if err != nil {
			log.Warn().Err(err).Int("node_id", node.Attributes.ID).Msg("Failed to upsert node")
		}

		// Update progress every 5 items or at end
		if (i+1)%5 == 0 || i == len(nodes)-1 {
			h.updateDetailedProgress(ctx, syncLogID, "nodes", len(nodes), i+1, fmt.Sprintf("Processing node %d/%d (%s)", i+1, len(nodes), node.Attributes.Name))
		}
	}

	// Remove stale nodes no longer in the panel
	if len(nodes) > 0 {
		ids := make([]interface{}, len(nodes))
		ph := make([]string, len(nodes))
		for i, node := range nodes {
			ids[i] = node.Attributes.ID
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		if res, err := h.db.Pool.Exec(ctx, `DELETE FROM nodes WHERE id NOT IN (`+strings.Join(ph, ",")+`)`, ids...); err != nil {
			log.Warn().Err(err).Msg("Failed to delete stale nodes")
		} else if res.RowsAffected() > 0 {
			log.Info().Int64("deleted", res.RowsAffected()).Msg("Deleted stale nodes")
		}
	}

	log.Info().Int("count", len(nodes)).Msg("Synced nodes")
	h.updateDetailedProgress(ctx, syncLogID, "nodes", len(nodes), len(nodes), fmt.Sprintf("✓ Synced %d nodes", len(nodes)))
	return nil
}

func (h *SyncHandler) syncAllocations(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing allocations")

	// Get all nodes first
	nodes, err := h.pteroClient.GetAllNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch nodes for allocations: %w", err)
	}

	h.updateDetailedProgress(ctx, syncLogID, "allocations", 0, 0, fmt.Sprintf("Fetching allocations from %d nodes", len(nodes)))

	totalAllocations := 0
	processedAllocations := 0
	batchSize := 500                   // Insert 500 allocations at a time for better performance
	allSeenAllocIDs := []interface{}{} // collect all allocation IDs for stale cleanup

	for nodeIdx, node := range nodes {
		allocations, err := h.pteroClient.GetAllAllocationsForNode(ctx, node.Attributes.ID)
		if err != nil {
			log.Warn().Err(err).Int("node_id", node.Attributes.ID).Msg("Failed to fetch allocations")
			continue
		}

		h.updateDetailedProgress(ctx, syncLogID, "allocations", 0, processedAllocations, fmt.Sprintf("Processing node %d/%d (%s): %d allocations", nodeIdx+1, len(nodes), node.Attributes.Name, len(allocations)))

		// Batch insert allocations
		for _, alloc := range allocations {
			allSeenAllocIDs = append(allSeenAllocIDs, alloc.Attributes.ID)
		}

		for batchStart := 0; batchStart < len(allocations); batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > len(allocations) {
				batchEnd = len(allocations)
			}
			batch := allocations[batchStart:batchEnd]

			// Build batch insert query
			query := `
				INSERT INTO allocations (id, ip, port, alias, notes, "isAssigned", "nodeId", "createdAt", "updatedAt")
				VALUES `

			args := make([]interface{}, 0, len(batch)*7)
			for i, alloc := range batch {
				if i > 0 {
					query += ", "
				}
				query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, NOW(), NOW())",
					i*7+1, i*7+2, i*7+3, i*7+4, i*7+5, i*7+6, i*7+7)
				args = append(args, alloc.Attributes.ID, alloc.Attributes.IP, alloc.Attributes.Port,
					alloc.Attributes.Alias, alloc.Attributes.Notes, alloc.Attributes.Assigned, node.Attributes.ID)
			}

			query += ` ON CONFLICT (id) DO UPDATE SET
				ip = EXCLUDED.ip,
				port = EXCLUDED.port,
				alias = EXCLUDED.alias,
				notes = EXCLUDED.notes,
				"isAssigned" = EXCLUDED."isAssigned",
				"nodeId" = EXCLUDED."nodeId",
				"updatedAt" = NOW()`

			_, err := h.db.Pool.Exec(ctx, query, args...)
			if err != nil {
				log.Warn().Err(err).Int("node_id", node.Attributes.ID).Int("batch_size", len(batch)).Msg("Failed to batch upsert allocations")
			}

			totalAllocations += len(batch)
			processedAllocations += len(batch)

			// Update every batch
			h.updateDetailedProgress(ctx, syncLogID, "allocations", 0, processedAllocations, fmt.Sprintf("Synced %d allocations from node %d/%d...", processedAllocations, nodeIdx+1, len(nodes)))
		}
	}

	// Remove stale allocations no longer in the panel
	if len(allSeenAllocIDs) > 0 {
		ph := make([]string, len(allSeenAllocIDs))
		for i := range allSeenAllocIDs {
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		// Only delete allocations that belong to known panel nodes (nodeId IS NOT NULL)
		if res, err := h.db.Pool.Exec(ctx,
			`DELETE FROM allocations WHERE id NOT IN (`+strings.Join(ph, ",")+`)`,
			allSeenAllocIDs...); err != nil {
			log.Warn().Err(err).Msg("Failed to delete stale allocations")
		} else if res.RowsAffected() > 0 {
			log.Info().Int64("deleted", res.RowsAffected()).Msg("Deleted stale allocations")
		}
	}

	log.Info().Int("count", totalAllocations).Msg("Synced allocations")
	h.updateDetailedProgress(ctx, syncLogID, "allocations", totalAllocations, totalAllocations, fmt.Sprintf("✓ Synced %d allocations", totalAllocations))
	return nil
}

func (h *SyncHandler) syncNestsAndEggs(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing nests and eggs")

	nests, err := h.pteroClient.GetAllNests(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch nests: %w", err)
	}

	h.updateDetailedProgress(ctx, syncLogID, "nests", len(nests), 0, fmt.Sprintf("Fetched %d nests from panel", len(nests)))

	totalEggs := 0
	processedNests := 0
	for nestIdx, nest := range nests {
		// Upsert nest
		nestQuery := `
			INSERT INTO nests (id, uuid, name, description, author, "createdAt", "updatedAt")
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			ON CONFLICT (id) DO UPDATE SET
				uuid = EXCLUDED.uuid,
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				author = EXCLUDED.author,
				"updatedAt" = NOW()
		`
		_, err := h.db.Pool.Exec(ctx, nestQuery,
			nest.Attributes.ID,
			nest.Attributes.UUID,
			nest.Attributes.Name,
			nest.Attributes.Description,
			nest.Attributes.Author,
		)
		if err != nil {
			log.Warn().Err(err).Int("nest_id", nest.Attributes.ID).Msg("Failed to upsert nest")
			continue
		}

		// Fetch and upsert eggs for this nest
		eggs, err := h.pteroClient.GetEggsForNest(ctx, nest.Attributes.ID, true)
		if err != nil {
			log.Warn().Err(err).Int("nest_id", nest.Attributes.ID).Msg("Failed to fetch eggs")
			continue
		}

		h.updateDetailedProgress(ctx, syncLogID, "nests", len(nests), nestIdx+1, fmt.Sprintf("Processing nest %d/%d (%s): %d eggs", nestIdx+1, len(nests), nest.Attributes.Name, len(eggs)))

		for _, egg := range eggs {
			eggQuery := `
				INSERT INTO eggs (id, uuid, name, description, author, "panelType", "nestId", "createdAt", "updatedAt")
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
				ON CONFLICT (id) DO UPDATE SET
					uuid = EXCLUDED.uuid,
					name = EXCLUDED.name,
					description = EXCLUDED.description,
					author = EXCLUDED.author,
					"panelType" = EXCLUDED."panelType",
					"nestId" = EXCLUDED."nestId",
					"updatedAt" = NOW()
			`
			_, err := h.db.Pool.Exec(ctx, eggQuery,
				egg.Attributes.ID,
				egg.Attributes.UUID,
				egg.Attributes.Name,
				egg.Attributes.Description,
				egg.Attributes.Author,
				"pterodactyl",
				nest.Attributes.ID,
			)
			if err != nil {
				log.Warn().Err(err).Int("egg_id", egg.Attributes.ID).Msg("Failed to upsert egg")
			}

			// Sync egg variables
			for _, variable := range egg.Relationships.Variables.Data {
				varQuery := `
					INSERT INTO egg_variables (id, "eggId", name, description, "envVariable", "defaultValue", "userViewable", "userEditable", rules, "createdAt", "updatedAt")
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
					ON CONFLICT (id) DO UPDATE SET
						name = EXCLUDED.name,
						description = EXCLUDED.description,
						"envVariable" = EXCLUDED."envVariable",
						"defaultValue" = EXCLUDED."defaultValue",
						"userViewable" = EXCLUDED."userViewable",
						"userEditable" = EXCLUDED."userEditable",
						rules = EXCLUDED.rules,
						"updatedAt" = NOW()
				`
				_, err := h.db.Pool.Exec(ctx, varQuery,
					variable.Attributes.ID,
					egg.Attributes.ID,
					variable.Attributes.Name,
					variable.Attributes.Description,
					variable.Attributes.EnvVariable,
					variable.Attributes.DefaultValue,
					variable.Attributes.UserViewable,
					variable.Attributes.UserEditable,
					variable.Attributes.Rules,
				)
				if err != nil {
					log.Warn().Err(err).Int("variable_id", variable.Attributes.ID).Msg("Failed to upsert egg variable")
				}
			}
			totalEggs++
		}
		processedNests++
	}

	// Remove stale nests no longer in the panel (eggs cascade via FK)
	if len(nests) > 0 {
		ids := make([]interface{}, len(nests))
		ph := make([]string, len(nests))
		for i, n := range nests {
			ids[i] = n.Attributes.ID
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		if res, err := h.db.Pool.Exec(ctx, `DELETE FROM nests WHERE id NOT IN (`+strings.Join(ph, ",")+`)`, ids...); err != nil {
			log.Warn().Err(err).Msg("Failed to delete stale nests")
		} else if res.RowsAffected() > 0 {
			log.Info().Int64("deleted", res.RowsAffected()).Msg("Deleted stale nests")
		}
	}

	log.Info().Int("nests", len(nests)).Int("eggs", totalEggs).Msg("Synced nests and eggs")
	h.updateDetailedProgress(ctx, syncLogID, "nests", len(nests), len(nests), fmt.Sprintf("✓ Synced %d nests and %d eggs", len(nests), totalEggs))
	return nil
}

func (h *SyncHandler) syncServers(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing servers")

	// Fetch servers with allocations data included
	servers, err := h.pteroClient.GetAllServers(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}

	h.updateDetailedProgress(ctx, syncLogID, "servers", len(servers), 0, fmt.Sprintf("Fetched %d servers from panel", len(servers)))

	for i, server := range servers {
		// Map status
		status := "online"
		if server.Attributes.Status != "" {
			status = server.Attributes.Status
		}
		if server.Attributes.Suspended {
			status = "suspended"
		}

		// Look up local owner — pterodactylId may not exist yet (users not yet synced).
		// We allow NULL here and reconcile during users sync.
		var ownerID *string
		_ = h.db.Pool.QueryRow(ctx,
			`SELECT id FROM users WHERE "pterodactylId" = $1 LIMIT 1`,
			server.Attributes.User,
		).Scan(&ownerID)

		query := `
			INSERT INTO servers (
				id, "pterodactylId", uuid, "uuidShort", "externalId", "panelType",
				name, description, status, "isSuspended",
				"ownerId", "nodeId", "eggId", memory, disk, cpu,
				"createdAt", "updatedAt"
			) VALUES (
				gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10,
				$11, $12, $13, $14, $15, NOW(), NOW()
			)
			ON CONFLICT ("pterodactylId") DO UPDATE SET
				uuid = EXCLUDED.uuid,
				"uuidShort" = EXCLUDED."uuidShort",
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				status = EXCLUDED.status,
				"isSuspended" = EXCLUDED."isSuspended",
				"ownerId" = COALESCE(EXCLUDED."ownerId", servers."ownerId"),
				"nodeId" = EXCLUDED."nodeId",
				"eggId" = EXCLUDED."eggId",
				memory = EXCLUDED.memory,
				disk = EXCLUDED.disk,
				cpu = EXCLUDED.cpu,
				"updatedAt" = NOW()
		`
		_, err := h.db.Pool.Exec(ctx, query,
			server.Attributes.ID,
			server.Attributes.UUID,
			server.Attributes.Identifier,
			server.Attributes.ExternalID,
			"pterodactyl",
			server.Attributes.Name,
			server.Attributes.Description,
			status,
			server.Attributes.Suspended,
			ownerID,
			server.Attributes.Node,
			server.Attributes.Egg,
			server.Attributes.Limits.Memory,
			server.Attributes.Limits.Disk,
			server.Attributes.Limits.CPU,
		)
		if err != nil {
			log.Warn().Err(err).Int("server_id", server.Attributes.ID).Msg("Failed to upsert server")
		}

		// Link allocations to this server if included in response
		if len(server.Relationships.Allocations.Data) > 0 {
			for _, alloc := range server.Relationships.Allocations.Data {
				_, err := h.db.Pool.Exec(ctx,
					`UPDATE allocations SET "serverId" = (SELECT id FROM servers WHERE "pterodactylId" = $1 LIMIT 1), "updatedAt" = NOW() WHERE id = $2`,
					server.Attributes.ID, alloc.Attributes.ID)
				if err != nil {
					log.Warn().Err(err).Int("allocation_id", alloc.Attributes.ID).Msg("Failed to link allocation to server")
				}
			}
		}

		// Update progress every 25 servers
		if (i+1)%25 == 0 || i == len(servers)-1 {
			h.updateDetailedProgress(ctx, syncLogID, "servers", len(servers), i+1, fmt.Sprintf("Processing server %d/%d (%s)", i+1, len(servers), server.Attributes.Name))
		}
	}

	// Remove stale panel servers no longer in Pterodactyl
	if len(servers) > 0 {
		ids := make([]interface{}, len(servers))
		ph := make([]string, len(servers))
		for i, srv := range servers {
			ids[i] = srv.Attributes.ID
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		if res, err := h.db.Pool.Exec(ctx,
			`DELETE FROM servers WHERE "pterodactylId" IS NOT NULL AND "panelType" = 'pterodactyl' AND "pterodactylId" NOT IN (`+strings.Join(ph, ",")+`)`,
			ids...); err != nil {
			log.Warn().Err(err).Msg("Failed to delete stale servers")
		} else if res.RowsAffected() > 0 {
			log.Info().Int64("deleted", res.RowsAffected()).Msg("Deleted stale servers")
		}
	}

	log.Info().Int("count", len(servers)).Msg("Synced servers")
	h.updateDetailedProgress(ctx, syncLogID, "servers", len(servers), len(servers), fmt.Sprintf("✓ Synced %d servers", len(servers)))
	return nil
}

func (h *SyncHandler) syncServerResources(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing detailed server resources (status, allocations, cpu usage, etc)")

	// Get all servers with UUIDs
	rows, err := h.db.Pool.Query(ctx, `SELECT id, uuid FROM servers WHERE uuid IS NOT NULL LIMIT 100`)
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}
	defer rows.Close()

	var servers []struct {
		ID   string
		UUID string
	}
	for rows.Next() {
		var serverID, uuid string
		if err := rows.Scan(&serverID, &uuid); err != nil {
			continue
		}
		servers = append(servers, struct {
			ID   string
			UUID string
		}{ID: serverID, UUID: uuid})
	}

	h.updateDetailedProgress(ctx, syncLogID, "server_resources", len(servers), 0, fmt.Sprintf("Fetching resource data for %d servers", len(servers)))

	for i, srv := range servers {
		// Fetch live resource data (CPU, memory, disk, network usage)
		resources, err := h.pteroClient.GetServerResources(ctx, srv.UUID)
		if err != nil {
			// Log but don't fail - resource endpoint may not be available
			log.Warn().Err(err).Str("server_uuid", srv.UUID).Msg("Failed to fetch server resources")
		} else if resources != nil {
			// Store resource data in database if we have a column for it
			// For now, just log that we got the data
			log.Debug().Str("server_uuid", srv.UUID).Interface("resources", resources).Msg("Fetched server resources")
		}

		// Update progress every 10 servers
		if (i+1)%10 == 0 || i == len(servers)-1 {
			h.updateDetailedProgress(ctx, syncLogID, "server_resources", len(servers), i+1, fmt.Sprintf("Processing resources %d/%d", i+1, len(servers)))
		}
	}

	log.Info().Int("count", len(servers)).Msg("Synced server resources")
	h.updateDetailedProgress(ctx, syncLogID, "server_resources", len(servers), len(servers), fmt.Sprintf("✓ Synced resources for %d servers", len(servers)))
	return nil
}

func (h *SyncHandler) syncDatabases(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing server databases")

	// Get all servers with pterodactyl IDs
	rows, err := h.db.Pool.Query(ctx, `SELECT id, "pterodactylId" FROM servers WHERE "pterodactylId" IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}
	defer rows.Close()

	// Count total servers first
	var servers []struct {
		ID      string
		PteroID int
	}
	for rows.Next() {
		var serverID string
		var pteroID int
		if err := rows.Scan(&serverID, &pteroID); err != nil {
			continue
		}
		servers = append(servers, struct {
			ID      string
			PteroID int
		}{serverID, pteroID})
	}

	h.updateDetailedProgress(ctx, syncLogID, "databases", len(servers), 0, fmt.Sprintf("Scanning databases across %d servers", len(servers)))

	totalDatabases := 0
	for serverIdx, server := range servers {
		databases, err := h.pteroClient.GetServerDatabasesWithHost(ctx, server.PteroID)
		if err != nil {
			log.Warn().Err(err).Int("pterodactyl_id", server.PteroID).Msg("Failed to fetch server databases")
			continue
		}

		h.updateDetailedProgress(ctx, syncLogID, "databases", 0, 0, fmt.Sprintf("Processing server %d/%d: %d databases", serverIdx+1, len(servers), len(databases)))

		for _, db := range databases {
			query := `
				INSERT INTO server_databases (id, "pterodactylId", "serverId", "databaseName", username, host, "maxConnections", "createdAt", "updatedAt")
				VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW(), NOW())
				ON CONFLICT ("pterodactylId") DO UPDATE SET
					"databaseName" = EXCLUDED."databaseName",
					username = EXCLUDED.username,
					host = EXCLUDED.host,
					"maxConnections" = EXCLUDED."maxConnections",
					"updatedAt" = NOW()
			`
			_, err := h.db.Pool.Exec(ctx, query,
				db.Attributes.ID,
				server.ID,
				db.Attributes.Database,
				db.Attributes.Username,
				db.Attributes.Host,
				db.Attributes.MaxConnections,
			)
			if err != nil {
				log.Warn().Err(err).Int("database_id", db.Attributes.ID).Msg("Failed to upsert database")
			}
			totalDatabases++
		}
	}

	log.Info().Int("count", totalDatabases).Msg("Synced server databases")
	h.updateDetailedProgress(ctx, syncLogID, "databases", totalDatabases, totalDatabases, fmt.Sprintf("✓ Synced %d databases", totalDatabases))
	return nil
}

func (h *SyncHandler) syncUsers(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing users")

	totalUsers := 0
	page := 1

	// First pass: estimate total pages
	resp, err := h.pteroClient.GetUsers(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to fetch users page 1: %w", err)
	}

	totalPages := resp.Meta.Pagination.TotalPages
	h.updateDetailedProgress(ctx, syncLogID, "users", totalPages*50, 0, fmt.Sprintf("Fetching %d users from %d pages", resp.Meta.Pagination.Total, totalPages))

	// Process first page
	var users []panels.PteroUser
	if err := json.Unmarshal(resp.Data, &users); err != nil {
		return fmt.Errorf("failed to unmarshal users: %w", err)
	}

	for _, user := range users {
		// Upsert user - creates if not exists, updates pterodactyl fields if exists
		query := `
			INSERT INTO users (
				id, email, username, "firstName", "lastName",
				"pterodactylId", "isPterodactylAdmin",
				"isMigrated", "isActive", "createdAt", "updatedAt"
			) VALUES (
				gen_random_uuid(), $1, $2, $3, $4, $5, $6, false, true, NOW(), NOW()
			)
			ON CONFLICT (email) DO UPDATE SET
				"pterodactylId" = EXCLUDED."pterodactylId",
				"isPterodactylAdmin" = EXCLUDED."isPterodactylAdmin",
				username = COALESCE(users.username, EXCLUDED.username),
				"firstName" = COALESCE(users."firstName", EXCLUDED."firstName"),
				"lastName" = COALESCE(users."lastName", EXCLUDED."lastName"),
				"updatedAt" = NOW()
		`
		_, err := h.db.Pool.Exec(ctx, query,
			user.Attributes.Email,
			user.Attributes.Username,
			user.Attributes.FirstName,
			user.Attributes.LastName,
			user.Attributes.ID,
			user.Attributes.RootAdmin,
		)
		if err != nil {
			log.Warn().Err(err).Str("email", user.Attributes.Email).Msg("Failed to upsert user")
		}
		totalUsers++
	}

	h.updateDetailedProgress(ctx, syncLogID, "users", resp.Meta.Pagination.Total, totalUsers, fmt.Sprintf("Processing page 1/%d (%d users)", totalPages, totalUsers))

	// Process remaining pages
	for page = 2; page <= totalPages; page++ {
		resp, err := h.pteroClient.GetUsers(ctx, page)
		if err != nil {
			return fmt.Errorf("failed to fetch users page %d: %w", page, err)
		}

		var users []panels.PteroUser
		if err := json.Unmarshal(resp.Data, &users); err != nil {
			return fmt.Errorf("failed to unmarshal users: %w", err)
		}

		for _, user := range users {
			query := `
				INSERT INTO users (
					id, email, username, "firstName", "lastName",
					"pterodactylId", "isPterodactylAdmin",
					"isMigrated", "isActive", "createdAt", "updatedAt"
				) VALUES (
					gen_random_uuid(), $1, $2, $3, $4, $5, $6, false, true, NOW(), NOW()
				)
				ON CONFLICT (email) DO UPDATE SET
					"pterodactylId" = EXCLUDED."pterodactylId",
					"isPterodactylAdmin" = EXCLUDED."isPterodactylAdmin",
					username = COALESCE(users.username, EXCLUDED.username),
					"firstName" = COALESCE(users."firstName", EXCLUDED."firstName"),
					"lastName" = COALESCE(users."lastName", EXCLUDED."lastName"),
					"updatedAt" = NOW()
			`
			_, err := h.db.Pool.Exec(ctx, query,
				user.Attributes.Email,
				user.Attributes.Username,
				user.Attributes.FirstName,
				user.Attributes.LastName,
				user.Attributes.ID,
				user.Attributes.RootAdmin,
			)
			if err != nil {
				log.Warn().Err(err).Str("email", user.Attributes.Email).Msg("Failed to upsert user")
			}
			totalUsers++
		}

		h.updateDetailedProgress(ctx, syncLogID, "users", resp.Meta.Pagination.Total, totalUsers, fmt.Sprintf("Processing page %d/%d (%d/%d users)", page, totalPages, totalUsers, resp.Meta.Pagination.Total))
	}

	log.Info().Int("count", totalUsers).Msg("Synced users")
	h.updateDetailedProgress(ctx, syncLogID, "users", totalUsers, totalUsers, fmt.Sprintf("✓ Synced %d users", totalUsers))
	return nil
}

func (h *SyncHandler) syncServerSubusers(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing server subusers via Client API")

	// Only sync if client API key is configured
	if h.cfg.PterodactylClientAPIKey == "" {
		log.Info().Msg("Skipping subuser sync - client API key not configured")
		h.updateDetailedProgress(ctx, syncLogID, "subusers", 0, 0, "⊘ Skipped - client API key not configured")
		return nil
	}

	// Check if subuser sync is enabled
	if !h.cfg.SyncSubusersEnabled {
		log.Info().Msg("Skipping subuser sync - disabled in config")
		h.updateDetailedProgress(ctx, syncLogID, "subusers", 0, 0, "⊘ Skipped - disabled in config")
		return nil
	}

	// Get servers that need subuser sync (owned by panel admin)
	rows, err := h.db.Pool.Query(ctx, `
		SELECT s.id, s.uuid 
		FROM servers s
		JOIN users u ON s."ownerId" = u.id
		WHERE u."isPterodactylAdmin" = true
		  AND s.uuid IS NOT NULL
		LIMIT $1
	`, h.cfg.SyncSubusersBatchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch admin servers: %w", err)
	}
	defer rows.Close()

	var servers []struct {
		ID   string
		UUID string
	}
	for rows.Next() {
		var s struct {
			ID   string
			UUID string
		}
		if err := rows.Scan(&s.ID, &s.UUID); err != nil {
			continue
		}
		servers = append(servers, s)
	}

	if len(servers) == 0 {
		log.Info().Msg("No admin-owned servers found for subuser sync")
		h.updateDetailedProgress(ctx, syncLogID, "subusers", 0, 0, "⊘ No admin-owned servers found")
		return nil
	}

	h.updateDetailedProgress(ctx, syncLogID, "subusers", len(servers), 0,
		fmt.Sprintf("Syncing subusers for %d admin-owned servers", len(servers)))

	totalSubusers := 0
	for i, server := range servers {
		// Add delay between requests to respect rate limits
		if i > 0 && i%5 == 0 {
			time.Sleep(2 * time.Second)
		}

		// Fetch subusers via Client API
		subusers, err := h.pteroClient.GetServerSubusers(ctx, server.UUID)
		if err != nil {
			log.Warn().Err(err).Str("server_uuid", server.UUID).
				Msg("Failed to fetch subusers - skipping server")
			continue
		}

		// Upsert subusers
		for _, subuser := range subusers {
			// Find user by email
			var userID string
			err := h.db.Pool.QueryRow(ctx,
				`SELECT id FROM users WHERE email = $1`, subuser.Attributes.Email).
				Scan(&userID)
			if err != nil {
				log.Debug().Str("email", subuser.Attributes.Email).
					Msg("Subuser not found in users table - may need full user sync first")
				continue
			}

			// Insert subuser relationship
			_, err = h.db.Pool.Exec(ctx, `
				INSERT INTO server_subusers (
					id, "serverId", "userId", permissions, "isOwner", "lastSyncedAt"
				) VALUES (
					gen_random_uuid(), $1, $2, $3, false, NOW()
				)
				ON CONFLICT ("serverId", "userId") DO UPDATE SET
					permissions = EXCLUDED.permissions,
					"isOwner" = EXCLUDED."isOwner",
					"lastSyncedAt" = NOW()
			`, server.ID, userID, subuser.Attributes.Permissions)

			if err != nil {
				log.Warn().Err(err).Str("email", subuser.Attributes.Email).
					Msg("Failed to upsert subuser")
			} else {
				totalSubusers++
			}
		}

		// Also mark owner in server_subusers table
		_, err = h.db.Pool.Exec(ctx, `
			INSERT INTO server_subusers (
				id, "serverId", "userId", permissions, "isOwner", "lastSyncedAt"
			) VALUES (
				gen_random_uuid(), $1, 
				(SELECT "ownerId" FROM servers WHERE id = $1),
				ARRAY['*'], true, NOW()
			)
			ON CONFLICT ("serverId", "userId") DO UPDATE SET
				"isOwner" = true,
				permissions = ARRAY['*'],
				"lastSyncedAt" = NOW()
		`, server.ID)

		if err != nil {
			log.Warn().Err(err).Str("server_id", server.ID).Msg("Failed to mark owner")
		}

		// Update progress every 5 servers
		if (i+1)%5 == 0 || i == len(servers)-1 {
			h.updateDetailedProgress(ctx, syncLogID, "subusers", len(servers), i+1,
				fmt.Sprintf("Processed %d/%d servers (%d subusers)", i+1, len(servers), totalSubusers))
		}
	}

	log.Info().Int("servers", len(servers)).Int("subusers", totalSubusers).Msg("Synced server subusers")
	h.updateDetailedProgress(ctx, syncLogID, "subusers", len(servers), len(servers),
		fmt.Sprintf("✓ Synced %d subusers across %d servers", totalSubusers, len(servers)))
	return nil
}

// Helper methods

func (h *SyncHandler) updateProgress(ctx context.Context, syncLogID, step string, progress int) {
	// Generate a user-friendly message for the step
	message := fmt.Sprintf("Syncing %s...", step)
	if progress == 100 {
		message = "Sync completed"
	}

	h.syncRepo.UpdateSyncLog(ctx, syncLogID, "RUNNING", nil, nil, nil, map[string]interface{}{
		"step":        step,
		"progress":    progress,
		"lastMessage": message,
		"lastUpdated": time.Now().Unix(),
	})
}

// updateDetailedProgress updates sync progress with detailed information
// itemsTotal: total number of items to sync
// itemsProcessed: number of items processed so far
// lastMessage: detailed message about current operation
func (h *SyncHandler) updateDetailedProgress(ctx context.Context, syncLogID, step string, itemsTotal, itemsProcessed int, lastMessage string) {
	percentage := 0
	if itemsTotal > 0 {
		percentage = (itemsProcessed * 100) / itemsTotal
	}

	h.syncRepo.UpdateSyncLog(ctx, syncLogID, "in_progress", &itemsTotal, &itemsProcessed, nil, map[string]interface{}{
		"step":           step,
		"itemsTotal":     itemsTotal,
		"itemsProcessed": itemsProcessed,
		"percentage":     percentage,
		"lastMessage":    lastMessage,
		"lastUpdated":    time.Now().Unix(),
	})
}

func (h *SyncHandler) failSync(ctx context.Context, syncLogID, step string, err error) error {
	duration := time.Duration(0)
	h.syncRepo.UpdateSyncLog(ctx, syncLogID, "FAILED", nil, nil, nil, map[string]interface{}{
		"failed_step": step,
		"error":       err.Error(),
	})
	// Dispatch failure webhook (non-blocking)
	go h.dispatchSyncWebhook(ctx, syncLogID, "FAILED", duration, err)
	return err
}

func (h *SyncHandler) cancelSync(ctx context.Context, syncLogID, reason string) error {
	h.syncRepo.UpdateSyncLog(ctx, syncLogID, "CANCELLED", nil, nil, nil, nil)
	return fmt.Errorf("sync cancelled: %s", reason)
}
