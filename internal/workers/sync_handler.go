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
	var payload queue.SyncFullPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Info().
		Str("sync_log_id", payload.SyncLogID).
		Str("requested_by", payload.RequestedBy).
		Msg("Starting full sync")

	startTime := time.Now()

	// Update sync log to RUNNING
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "RUNNING", nil, nil, map[string]interface{}{
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

	// Step 5: Sync Servers
	if checkCancelled() {
		return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before servers sync")
	}
	h.updateProgress(ctx, payload.SyncLogID, "servers", 60)
	if err := h.syncServers(ctx, payload.SyncLogID); err != nil {
		return h.failSync(ctx, payload.SyncLogID, "servers", err)
	}

	// Step 6: Sync Users (optional)
	if !payload.SkipUsers {
		if checkCancelled() {
			return h.cancelSync(ctx, payload.SyncLogID, "Cancelled before users sync")
		}
		h.updateProgress(ctx, payload.SyncLogID, "users", 80)
		if err := h.syncUsers(ctx, payload.SyncLogID); err != nil {
			return h.failSync(ctx, payload.SyncLogID, "users", err)
		}
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Complete
	h.updateProgress(ctx, payload.SyncLogID, "completed", 100)
	h.syncRepo.UpdateSyncLog(ctx, payload.SyncLogID, "COMPLETED", nil, nil, map[string]interface{}{
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
	// Get all enabled SYSTEM webhooks
	query := `
		SELECT "webhookUrl" 
		FROM discord_webhooks 
		WHERE enabled = true 
		AND type = 'SYSTEM' 
		AND scope = 'ADMIN'
	`

	rows, err := h.db.Pool.Query(ctx, query)
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

	return h.syncLocations(ctx, payload.SyncLogID)
}

// HandleSyncNodes syncs only nodes
func (h *SyncHandler) HandleSyncNodes(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return h.syncNodes(ctx, payload.SyncLogID)
}

// HandleSyncAllocations syncs only allocations
func (h *SyncHandler) HandleSyncAllocations(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return h.syncAllocations(ctx, payload.SyncLogID)
}

// HandleSyncNests syncs nests and eggs
func (h *SyncHandler) HandleSyncNests(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return h.syncNestsAndEggs(ctx, payload.SyncLogID)
}

// HandleSyncServers syncs servers
func (h *SyncHandler) HandleSyncServers(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return h.syncServers(ctx, payload.SyncLogID)
}

// HandleSyncDatabases syncs server databases
func (h *SyncHandler) HandleSyncDatabases(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return h.syncDatabases(ctx, payload.SyncLogID)
}

// HandleSyncUsers syncs users
func (h *SyncHandler) HandleSyncUsers(ctx context.Context, task *asynq.Task) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return h.syncUsers(ctx, payload.SyncLogID)
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

	query := `DELETE FROM sync_logs WHERE created_at < $1`
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

	for _, loc := range locations {
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
	}

	log.Info().Int("count", len(locations)).Msg("Synced locations")
	return nil
}

func (h *SyncHandler) syncNodes(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing nodes")

	nodes, err := h.pteroClient.GetAllNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch nodes: %w", err)
	}

	for _, node := range nodes {
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
	}

	log.Info().Int("count", len(nodes)).Msg("Synced nodes")
	return nil
}

func (h *SyncHandler) syncAllocations(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing allocations")

	// Get all nodes first
	nodes, err := h.pteroClient.GetAllNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch nodes for allocations: %w", err)
	}

	totalAllocations := 0
	for _, node := range nodes {
		allocations, err := h.pteroClient.GetAllAllocationsForNode(ctx, node.Attributes.ID)
		if err != nil {
			log.Warn().Err(err).Int("node_id", node.Attributes.ID).Msg("Failed to fetch allocations")
			continue
		}

		for _, alloc := range allocations {
			query := `
				INSERT INTO allocations (id, ip, port, alias, notes, "isAssigned", "nodeId", "createdAt", "updatedAt")
				VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
				ON CONFLICT (id) DO UPDATE SET
					ip = EXCLUDED.ip,
					port = EXCLUDED.port,
					alias = EXCLUDED.alias,
					notes = EXCLUDED.notes,
					"isAssigned" = EXCLUDED."isAssigned",
					"nodeId" = EXCLUDED."nodeId",
					"updatedAt" = NOW()
			`
			_, err := h.db.Pool.Exec(ctx, query,
				alloc.Attributes.ID,
				alloc.Attributes.IP,
				alloc.Attributes.Port,
				alloc.Attributes.Alias,
				alloc.Attributes.Notes,
				alloc.Attributes.Assigned,
				node.Attributes.ID,
			)
			if err != nil {
				log.Warn().Err(err).Int("allocation_id", alloc.Attributes.ID).Msg("Failed to upsert allocation")
			}
			totalAllocations++
		}
	}

	log.Info().Int("count", totalAllocations).Msg("Synced allocations")
	return nil
}

func (h *SyncHandler) syncNestsAndEggs(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing nests and eggs")

	nests, err := h.pteroClient.GetAllNests(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch nests: %w", err)
	}

	totalEggs := 0
	for _, nest := range nests {
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
	}

	log.Info().Int("nests", len(nests)).Int("eggs", totalEggs).Msg("Synced nests and eggs")
	return nil
}

func (h *SyncHandler) syncServers(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing servers")

	servers, err := h.pteroClient.GetAllServers(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to fetch servers: %w", err)
	}

	for _, server := range servers {
		// Map status
		status := "online"
		if server.Attributes.Status != "" {
			status = server.Attributes.Status
		}
		if server.Attributes.Suspended {
			status = "suspended"
		}

		query := `
			INSERT INTO servers (
				id, "pterodactylId", uuid, "uuidShort", "externalId", "panelType",
				name, description, status, "isSuspended",
				"ownerId", "nodeId", "eggId", memory, disk, cpu,
				"createdAt", "updatedAt"
			) VALUES (
				gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9,
				(SELECT id FROM users WHERE "pterodactylId" = $10 LIMIT 1),
				$11, $12, $13, $14, $15, NOW(), NOW()
			)
			ON CONFLICT ("pterodactylId") DO UPDATE SET
				uuid = EXCLUDED.uuid,
				"uuidShort" = EXCLUDED."uuidShort",
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				status = EXCLUDED.status,
				"isSuspended" = EXCLUDED."isSuspended",
				"ownerId" = EXCLUDED."ownerId",
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
			server.Attributes.User,
			server.Attributes.Node,
			server.Attributes.Egg,
			server.Attributes.Limits.Memory,
			server.Attributes.Limits.Disk,
			server.Attributes.Limits.CPU,
		)
		if err != nil {
			log.Warn().Err(err).Int("server_id", server.Attributes.ID).Msg("Failed to upsert server")
		}
	}

	log.Info().Int("count", len(servers)).Msg("Synced servers")
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

	totalDatabases := 0
	for rows.Next() {
		var serverID string
		var pteroID int
		if err := rows.Scan(&serverID, &pteroID); err != nil {
			continue
		}

		databases, err := h.pteroClient.GetServerDatabasesWithHost(ctx, pteroID)
		if err != nil {
			log.Warn().Err(err).Int("pterodactyl_id", pteroID).Msg("Failed to fetch server databases")
			continue
		}

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
				serverID,
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
	return nil
}

func (h *SyncHandler) syncUsers(ctx context.Context, syncLogID string) error {
	log.Debug().Str("sync_log_id", syncLogID).Msg("Syncing users")

	totalUsers := 0
	page := 1

	for {
		resp, err := h.pteroClient.GetUsers(ctx, page)
		if err != nil {
			return fmt.Errorf("failed to fetch users page %d: %w", page, err)
		}

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

		if page >= resp.Meta.Pagination.TotalPages {
			break
		}
		page++
	}

	log.Info().Int("count", totalUsers).Msg("Synced users")
	return nil
}

// Helper methods

func (h *SyncHandler) updateProgress(ctx context.Context, syncLogID, step string, progress int) {
	h.syncRepo.UpdateSyncLog(ctx, syncLogID, "in_progress", nil, nil, map[string]interface{}{
		"step":     step,
		"progress": progress,
	})
}

func (h *SyncHandler) failSync(ctx context.Context, syncLogID, step string, err error) error {
	duration := time.Duration(0)
	h.syncRepo.UpdateSyncLog(ctx, syncLogID, "FAILED", nil, nil, map[string]interface{}{
		"failed_step": step,
		"error":       err.Error(),
	})
	// Dispatch failure webhook (non-blocking)
	go h.dispatchSyncWebhook(ctx, syncLogID, "FAILED", duration, err)
	return err
}

func (h *SyncHandler) cancelSync(ctx context.Context, syncLogID, reason string) error {
	h.syncRepo.UpdateSyncLog(ctx, syncLogID, "CANCELLED", nil, nil, nil)
	return fmt.Errorf("sync cancelled: %s", reason)
}
