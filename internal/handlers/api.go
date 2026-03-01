// Package handlers provides HTTP handlers for the NodeByte API.
//
// @title NodeByte Backend API
// @version 0.2.0
// @description Comprehensive API for managing game server infrastructure with Pterodactyl panel integration
// @host localhost:8080
// @basePath /
// @schemes http https
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @securityDefinitions.http BearerAuth
// @scheme bearer
// @bearerFormat JWT
package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/queue"
)

// SyncAPIHandler handles sync-related API requests
type SyncAPIHandler struct {
	db           *database.DB
	syncRepo     *database.SyncRepository
	queueManager *queue.Manager
}

// NewSyncAPIHandler creates a new sync API handler
func NewSyncAPIHandler(db *database.DB, queueManager *queue.Manager) *SyncAPIHandler {
	return &SyncAPIHandler{
		db:           db,
		syncRepo:     database.NewSyncRepository(db),
		queueManager: queueManager,
	}
}

// TriggerFullSyncRequest represents a full sync request
type TriggerFullSyncRequest struct {
	SkipUsers   bool   `json:"skip_users"`
	RequestedBy string `json:"requested_by"`
}

// TriggerFullSync triggers a full sync operation
// @Summary Trigger full sync
// @Description Initiates a complete synchronization of all resources (locations, nodes, allocations, nests, servers, databases, users)
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param payload body TriggerFullSyncRequest true "Sync request parameters"
// @Success 202 {object} SuccessResponse "Sync queued successfully"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/full [post]
func (h *SyncAPIHandler) TriggerFullSync(c *fiber.Ctx) error {
	var req TriggerFullSyncRequest
	if err := c.BodyParser(&req); err != nil {
		// Ignore parse errors, use defaults
	}

	// Create sync log
	syncLog, err := h.syncRepo.CreateSyncLog(c.Context(), "full", "PENDING", map[string]interface{}{
		"requested_by": req.RequestedBy,
		"skip_users":   req.SkipUsers,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create sync log")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to create sync log",
		})
	}

	// Enqueue the sync task
	taskInfo, err := h.queueManager.EnqueueSyncFull(queue.SyncFullPayload{
		SyncLogID:   syncLog.ID,
		RequestedBy: req.RequestedBy,
		SkipUsers:   req.SkipUsers,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to enqueue sync task")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to enqueue sync task",
		})
	}

	log.Info().
		Str("sync_log_id", syncLog.ID).
		Str("task_id", taskInfo.ID).
		Msg("Full sync triggered")

	return c.Status(fiber.StatusAccepted).JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"sync_log_id": syncLog.ID,
			"task_id":     taskInfo.ID,
			"status":      "PENDING",
		},
		Message: "Full sync has been queued",
	})
}

// TriggerLocationsSync triggers a locations-only sync
// @Summary Trigger locations sync
// @Description Synchronizes only location data from Pterodactyl panel
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 202 {object} SuccessResponse "Sync queued successfully"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/locations [post]
func (h *SyncAPIHandler) TriggerLocationsSync(c *fiber.Ctx) error {
	return h.triggerPartialSync(c, "locations", queue.TypeSyncLocations)
}

// TriggerNodesSync triggers a nodes-only sync
// @Summary Trigger nodes sync
// @Description Synchronizes only node data from Pterodactyl panel
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 202 {object} SuccessResponse "Sync queued successfully"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/nodes [post]
func (h *SyncAPIHandler) TriggerNodesSync(c *fiber.Ctx) error {
	return h.triggerPartialSync(c, "nodes", queue.TypeSyncNodes)
}

// TriggerServersSync triggers a servers-only sync
// @Summary Trigger servers sync
// @Description Synchronizes only server data from Pterodactyl panel
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 202 {object} SuccessResponse "Sync queued successfully"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/servers [post]
func (h *SyncAPIHandler) TriggerServersSync(c *fiber.Ctx) error {
	return h.triggerPartialSync(c, "servers", queue.TypeSyncServers)
}

// TriggerUsersSync triggers a users-only sync
// @Summary Trigger users sync
// @Description Synchronizes only user data from Pterodactyl panel
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 202 {object} SuccessResponse "Sync queued successfully"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/users [post]
func (h *SyncAPIHandler) TriggerUsersSync(c *fiber.Ctx) error {
	return h.triggerPartialSync(c, "users", queue.TypeSyncUsers)
}

func (h *SyncAPIHandler) triggerPartialSync(c *fiber.Ctx, syncType, taskType string) error {
	syncLog, err := h.syncRepo.CreateSyncLog(c.Context(), syncType, "PENDING", nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to create sync log",
		})
	}

	payload := queue.SyncPayload{SyncLogID: syncLog.ID}
	var taskInfo *asynq.TaskInfo

	switch taskType {
	case queue.TypeSyncLocations:
		info, err := h.queueManager.EnqueueSyncLocations(payload)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Success: false,
				Error:   "Failed to enqueue task",
			})
		}
		taskInfo = info
	case queue.TypeSyncNodes:
		info, err := h.queueManager.EnqueueSyncNodes(payload)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Success: false,
				Error:   "Failed to enqueue task",
			})
		}
		taskInfo = info
	case queue.TypeSyncServers:
		info, err := h.queueManager.EnqueueSyncServers(payload)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Success: false,
				Error:   "Failed to enqueue task",
			})
		}
		taskInfo = info
	case queue.TypeSyncUsers:
		info, err := h.queueManager.EnqueueSyncUsers(payload)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Success: false,
				Error:   "Failed to enqueue task",
			})
		}
		taskInfo = info
	}

	return c.Status(fiber.StatusAccepted).JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"sync_log_id": syncLog.ID,
			"task_id":     taskInfo.ID,
			"status":      "PENDING",
		},
		Message: syncType + " sync has been queued",
	})
}

// CancelSync cancels a running sync operation
// @Summary Cancel sync operation
// @Description Requests cancellation of a running sync by ID
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Sync log ID"
// @Success 200 {object} SuccessResponse "Cancellation requested"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/cancel/{id} [post]
func (h *SyncAPIHandler) CancelSync(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Sync ID is required",
		})
	}

	// Update sync log to cancelling status
	err := h.syncRepo.UpdateSyncLog(c.Context(), id, "cancelling", nil, nil, nil, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to request cancellation",
		})
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Message: "Cancellation requested",
	})
}

// GetSyncStatus gets the status of a sync operation
// @Summary Get sync status
// @Description Retrieves detailed status and metadata of a specific sync operation
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Sync log ID"
// @Success 200 {object} SuccessResponse "Sync status retrieved"
// @Failure 404 {object} ErrorResponse "Sync not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/status/{id} [get]
func (h *SyncAPIHandler) GetSyncStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Sync ID is required",
		})
	}

	query := `
		SELECT id, type, status, "itemsTotal", "itemsSynced", "itemsFailed", error, metadata, "startedAt", "completedAt"
		FROM "sync_logs" WHERE id = $1
	`

	var log database.SyncLog
	err := h.db.Pool.QueryRow(c.Context(), query, id).Scan(
		&log.ID, &log.Type, &log.Status, &log.ItemsTotal, &log.ItemsSynced, &log.ItemsFailed, &log.Error, &log.Metadata,
		&log.StartedAt, &log.CompletedAt,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Success: false,
			Error:   "Sync log not found",
		})
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data:    log,
	})
}

// GetSyncLogs gets sync logs with pagination
// @Summary Get sync logs
// @Description Retrieves paginated list of sync operation logs
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Limit results (default 20)" Default(20) Minimum(1) Maximum(100)
// @Param offset query int false "Offset for pagination (default 0)" Default(0) Minimum(0)
// @Param type query string false "Filter by sync type (full, locations, nodes, servers, users)"
// @Success 200 {object} SuccessResponse "Sync logs retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/logs [get]
func (h *SyncAPIHandler) GetSyncLogs(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	syncType := c.Query("type")

	query := `
		SELECT id, type, status, "itemsTotal", "itemsSynced", "itemsFailed", error, metadata, "startedAt", "completedAt"
		FROM "sync_logs"
	`
	args := []interface{}{}

	if syncType != "" {
		query += " WHERE type = $1"
		args = append(args, syncType)
	}

	paramCount := len(args) + 1
	query += " ORDER BY \"startedAt\" DESC LIMIT $" + strconv.Itoa(paramCount) + " OFFSET $" + strconv.Itoa(paramCount+1)
	args = append(args, limit, offset)

	rows, err := h.db.Pool.Query(c.Context(), query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch sync logs",
		})
	}
	defer rows.Close()

	var logs []database.SyncLog
	for rows.Next() {
		var log database.SyncLog
		err := rows.Scan(
			&log.ID, &log.Type, &log.Status, &log.ItemsTotal, &log.ItemsSynced, &log.ItemsFailed, &log.Error, &log.Metadata,
			&log.StartedAt, &log.CompletedAt,
		)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"logs":   logs,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetLatestSync gets the latest sync for each type
// @Summary Get latest syncs
// @Description Retrieves the most recent sync operation for each sync type
// @Tags Sync
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse "Latest syncs retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/sync/latest [get]
func (h *SyncAPIHandler) GetLatestSync(c *fiber.Ctx) error {
	query := `
		SELECT DISTINCT ON (type) id, type, status, "itemsTotal", "itemsSynced", "itemsFailed", error, metadata, "startedAt", "completedAt"
		FROM sync_logs
		ORDER BY type, "startedAt" DESC
	`

	rows, err := h.db.Pool.Query(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch latest syncs",
		})
	}
	defer rows.Close()

	latest := make(map[string]database.SyncLog)
	for rows.Next() {
		var log database.SyncLog
		err := rows.Scan(
			&log.ID, &log.Type, &log.Status, &log.ItemsTotal, &log.ItemsSynced, &log.ItemsFailed, &log.Error, &log.Metadata,
			&log.StartedAt, &log.CompletedAt,
		)
		if err != nil {
			continue
		}
		latest[log.Type] = log
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data:    latest,
	})
}

// StatsHandler handles statistics API requests
type StatsHandler struct {
	db *database.DB
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(db *database.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

// GetOverview returns an overview of system statistics
// @Summary Get system overview
// @Description Retrieves aggregate statistics for all resources in the system
// @Tags Stats
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse "System statistics retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/stats/overview [get]
func (h *StatsHandler) GetOverview(c *fiber.Ctx) error {
	ctx := c.Context()

	// Run count queries in parallel
	type countResult struct {
		name  string
		count int
		err   error
	}

	queries := map[string]string{
		"users":       "SELECT COUNT(*) FROM users",
		"servers":     "SELECT COUNT(*) FROM servers",
		"nodes":       "SELECT COUNT(*) FROM nodes",
		"locations":   "SELECT COUNT(*) FROM locations",
		"eggs":        "SELECT COUNT(*) FROM eggs",
		"allocations": "SELECT COUNT(*) FROM allocations",
	}

	results := make(map[string]int)
	for name, query := range queries {
		var count int
		err := h.db.Pool.QueryRow(ctx, query).Scan(&count)
		if err != nil {
			log.Warn().Err(err).Str("query", name).Msg("Failed to get count")
			count = 0
		}
		results[name] = count
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data:    results,
	})
}

// GetServerStats returns server-specific statistics
// @Summary Get server statistics
// @Description Retrieves server count statistics grouped by status and node
// @Tags Stats
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse "Server statistics retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/stats/servers [get]
func (h *StatsHandler) GetServerStats(c *fiber.Ctx) error {
	ctx := c.Context()

	// Servers by status
	statusQuery := `
		SELECT status, COUNT(*) as count 
		FROM servers 
		GROUP BY status
	`
	rows, err := h.db.Pool.Query(ctx, statusQuery)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch server stats",
		})
	}
	defer rows.Close()

	byStatus := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		byStatus[status] = count
	}

	// Servers by node
	nodeQuery := `
		SELECT n.name, COUNT(s.id) as count 
		FROM nodes n
		LEFT JOIN servers s ON s.node_id = n.id
		GROUP BY n.id, n.name
	`
	nodeRows, err := h.db.Pool.Query(ctx, nodeQuery)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch server stats by node",
		})
	}
	defer nodeRows.Close()

	byNode := make(map[string]int)
	for nodeRows.Next() {
		var name string
		var count int
		if err := nodeRows.Scan(&name, &count); err != nil {
			continue
		}
		byNode[name] = count
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"by_status": byStatus,
			"by_node":   byNode,
		},
	})
}

// GetUserStats returns user-specific statistics
// @Summary Get user statistics
// @Description Retrieves user statistics including active, migrated, and admin counts
// @Tags Stats
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse "User statistics retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/stats/users [get]
func (h *StatsHandler) GetUserStats(c *fiber.Ctx) error {
	ctx := c.Context()

	stats := make(map[string]interface{})

	// Total users
	var totalUsers int
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)
	stats["total"] = totalUsers

	// Active users
	var activeUsers int
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE \"isActive\" = true").Scan(&activeUsers)
	stats["active"] = activeUsers

	// Migrated users
	var migratedUsers int
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE \"isMigrated\" = true").Scan(&migratedUsers)
	stats["migrated"] = migratedUsers

	// Admin users
	var adminUsers int
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE \"isPterodactylAdmin\" = true OR \"isSystemAdmin\" = true").Scan(&adminUsers)
	stats["admins"] = adminUsers

	// Users registered in last 7 days
	var recentUsers int
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE \"createdAt\" > NOW() - INTERVAL '7 days'").Scan(&recentUsers)
	stats["recent_7_days"] = recentUsers

	return c.JSON(SuccessResponse{
		Success: true,
		Data:    stats,
	})
}

// EmailAPIHandler handles email API requests
type EmailAPIHandler struct {
	queueManager *queue.Manager
}

// NewEmailAPIHandler creates a new email API handler
func NewEmailAPIHandler(queueManager *queue.Manager) *EmailAPIHandler {
	return &EmailAPIHandler{queueManager: queueManager}
}

// QueueEmailRequest represents an email queue request
type QueueEmailRequest struct {
	To       string            `json:"to"`
	Subject  string            `json:"subject"`
	Template string            `json:"template"`
	Data     map[string]string `json:"data"`
}

// QueueEmail queues an email for sending
// @Summary Queue email
// @Description Queues an email message for asynchronous sending
// @Tags Email
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param payload body QueueEmailRequest true "Email request parameters"
// @Success 202 {object} SuccessResponse "Email queued successfully"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/email/queue [post]
func (h *EmailAPIHandler) QueueEmail(c *fiber.Ctx) error {
	var req QueueEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.To == "" || req.Subject == "" || req.Template == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "to, subject, and template are required",
		})
	}

	taskInfo, err := h.queueManager.EnqueueEmail(queue.EmailPayload{
		To:       req.To,
		Subject:  req.Subject,
		Template: req.Template,
		Data:     req.Data,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to queue email",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"task_id": taskInfo.ID,
		},
		Message: "Email has been queued",
	})
}

// WebhookAPIHandler handles webhook API requests
type WebhookAPIHandler struct {
	db           *database.DB
	queueManager *queue.Manager
}

// NewWebhookAPIHandler creates a new webhook API handler
func NewWebhookAPIHandler(db *database.DB, queueManager *queue.Manager) *WebhookAPIHandler {
	return &WebhookAPIHandler{
		db:           db,
		queueManager: queueManager,
	}
}

// DispatchWebhookRequest represents a webhook dispatch request
type DispatchWebhookRequest struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

// DispatchWebhook dispatches a webhook to all applicable webhooks
// @Summary Dispatch webhook
// @Description Dispatches a webhook event to all configured webhooks that handle the event type
// @Tags Webhooks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param payload body DispatchWebhookRequest true "Webhook dispatch parameters"
// @Success 202 {object} SuccessResponse "Webhooks queued"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/webhook/dispatch [post]
func (h *WebhookAPIHandler) DispatchWebhook(c *fiber.Ctx) error {
	var req DispatchWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.Event == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "event is required",
		})
	}

	// Get all enabled webhooks
	query := `SELECT id FROM "discord_webhooks" WHERE enabled = true`
	rows, err := h.db.Pool.Query(c.Context(), query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch webhooks",
		})
	}
	defer rows.Close()

	var taskIDs []string
	for rows.Next() {
		var webhookID string
		if err := rows.Scan(&webhookID); err != nil {
			continue
		}

		taskInfo, err := h.queueManager.EnqueueWebhook(queue.WebhookPayload{
			WebhookID: webhookID,
			Event:     req.Event,
			Data:      req.Data,
		})
		if err != nil {
			log.Warn().Err(err).Str("webhook_id", webhookID).Msg("Failed to queue webhook")
			continue
		}
		taskIDs = append(taskIDs, taskInfo.ID)
	}

	return c.Status(fiber.StatusAccepted).JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"dispatched": len(taskIDs),
			"task_ids":   taskIDs,
		},
		Message: "Webhooks have been queued",
	})
}

// QueueHandler handles queue inspection requests
type QueueHandler struct{}

// NewQueueHandler creates a new queue handler
func NewQueueHandler() *QueueHandler {
	return &QueueHandler{}
}

// GetStats returns queue statistics
// @Summary Get queue statistics
// @Description Retrieves statistics about job queues
// @Tags Queues
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse "Queue statistics retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/queues/stats [get]
func (h *QueueHandler) GetStats(c *fiber.Ctx) error {
	// TODO: Implement queue stats using Asynq inspector
	// This would show pending/active/completed/failed task counts

	return c.JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"queues": []string{queue.QueueCritical, queue.QueueDefault, queue.QueueLow},
			"note":   "Detailed stats coming soon",
		},
	})
}

// Helper to generate unique IDs
func generateID() string {
	return uuid.New().String()
}

// Helper to get current time
func now() time.Time {
	return time.Now()
}

// ============================================================================
// PHASE 1: PUBLIC STATS HANDLERS (No Auth Required)
// ============================================================================

// GetPublicStats handles GET /api/stats (public endpoint)
// @Summary Get public statistics
// @Description Retrieves publicly available system statistics (no authentication required)
// @Tags Public
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse "Public statistics retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/stats [get]
func (h *StatsHandler) GetPublicStats(c *fiber.Ctx) error {
	ctx := c.Context()

	var totalServers, totalUsers, totalAllocations, activeUsers int

	// Get counts
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM servers").Scan(&totalServers)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM allocations").Scan(&totalAllocations)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE last_login_at IS NOT NULL").Scan(&activeUsers)

	return c.JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"totalServers":     totalServers,
			"totalUsers":       totalUsers,
			"activeUsers":      activeUsers,
			"totalAllocations": totalAllocations,
		},
	})
}

// GetPanelCounts handles GET /api/panel/counts (public endpoint)
// @Summary Get panel resource counts
// @Description Retrieves counts of all manageable resources for the panel (no authentication required)
// @Tags Public
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse "Panel counts retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/panel/counts [get]
func (h *StatsHandler) GetPanelCounts(c *fiber.Ctx) error {
	ctx := c.Context()

	var nodeCount, serverCount, userCount, allocationCount, nestCount int

	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM nodes").Scan(&nodeCount)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM servers").Scan(&serverCount)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM allocations WHERE \"isAssigned\" = true").Scan(&allocationCount)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM nests").Scan(&nestCount)

	return c.JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"nodes":       nodeCount,
			"servers":     serverCount,
			"users":       userCount,
			"allocations": allocationCount,
			"nests":       nestCount,
		},
	})
}

// ============================================================================
// PHASE 2: ADMIN SYNC CONTROL HANDLERS (Bearer Token Auth)
// ============================================================================

// AdminSyncHandler handles admin sync control endpoints
type AdminSyncHandler struct {
	db           *database.DB
	syncRepo     *database.SyncRepository
	queueManager *queue.Manager
}

// NewAdminSyncHandler creates a new admin sync handler
func NewAdminSyncHandler(db *database.DB, queueManager *queue.Manager) *AdminSyncHandler {
	return &AdminSyncHandler{
		db:           db,
		syncRepo:     database.NewSyncRepository(db),
		queueManager: queueManager,
	}
}

// GetSyncLogs handles GET /api/admin/sync/logs
// @Summary Get sync logs (admin)
// @Description Retrieves paginated sync operation logs with full details
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit results (default 20)" Default(20) Minimum(1) Maximum(100)
// @Param offset query int false "Offset for pagination (default 0)" Default(0) Minimum(0)
// @Param type query string false "Filter by sync type"
// @Success 200 {object} SuccessResponse "Sync logs retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/admin/sync/logs [get]
func (h *AdminSyncHandler) GetSyncLogs(c *fiber.Ctx) error {
	ctx := c.Context()
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	syncType := c.Query("type", "")

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	logs, err := h.syncRepo.GetSyncLogs(ctx, limit, offset, syncType)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch sync logs")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch sync logs",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"logs":    logs,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetSyncStatusAdmin handles GET /api/admin/sync
// @Summary Get sync status (admin)
// @Description Retrieves current sync status and recent stats
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse "Sync status retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/admin/sync [get]
func (h *AdminSyncHandler) GetSyncStatusAdmin(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get latest sync log
	logs, err := h.syncRepo.GetSyncLogs(ctx, 1, 0, "")
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch sync logs")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch sync status",
		})
	}

	var latestSync interface{} = nil
	if len(logs) > 0 {
		latestSync = logs[0]
	}

	// Get stats - all counts needed
	var (
		totalUsers, migratedUsers, totalServers, totalNodes, totalLocations int
		totalAllocations, totalNests, totalEggs, totalEggVariables          int
		totalServerDatabases                                                int
	)

	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE \"isMigrated\" = true").Scan(&migratedUsers)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM servers").Scan(&totalServers)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM nodes").Scan(&totalNodes)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM locations").Scan(&totalLocations)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM allocations").Scan(&totalAllocations)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM nests").Scan(&totalNests)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM eggs").Scan(&totalEggs)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM egg_variables").Scan(&totalEggVariables)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM server_databases").Scan(&totalServerDatabases)

	return c.JSON(fiber.Map{
		"success": true,
		"status": fiber.Map{
			"lastSync":  latestSync,
			"isSyncing": false,
		},
		"counts": fiber.Map{
			"users":           totalUsers,
			"migratedUsers":   migratedUsers,
			"servers":         totalServers,
			"nodes":           totalNodes,
			"locations":       totalLocations,
			"allocations":     totalAllocations,
			"nests":           totalNests,
			"eggs":            totalEggs,
			"eggVariables":    totalEggVariables,
			"serverDatabases": totalServerDatabases,
		},
		"availableTargets": []string{"full", "locations", "nodes", "servers", "users"},
	})
}

// TriggerSyncAdminRequest represents a sync trigger request from admin
type TriggerSyncAdminRequest struct {
	Type string `json:"type"`
}

// TriggerSyncAdmin handles POST /api/admin/sync
// @Summary Trigger sync (admin)
// @Description Triggers a synchronization operation with specified type
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body TriggerSyncAdminRequest true "Sync trigger parameters"
// @Success 202 {object} SuccessResponse "Sync queued successfully"
// @Failure 400 {object} ErrorResponse "Invalid sync type"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/admin/sync [post]
func (h *AdminSyncHandler) TriggerSyncAdmin(c *fiber.Ctx) error {
	var req TriggerSyncAdminRequest
	if err := c.BodyParser(&req); err != nil {
		req.Type = "full"
	}

	// Default to full sync
	syncType := req.Type
	if syncType == "" {
		syncType = "full"
	}

	// Validate and map sync type
	validTypes := map[string]bool{
		"full":        true,
		"locations":   true,
		"nodes":       true,
		"allocations": true,
		"nests":       true,
		"servers":     true,
		"databases":   true,
		"users":       true,
	}

	if !validTypes[syncType] {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Invalid sync type. Valid types: full, locations, nodes, allocations, nests, servers, databases, users",
		})
	}

	// Create sync log
	syncLog, err := h.syncRepo.CreateSyncLog(c.Context(), syncType, "PENDING", map[string]interface{}{
		"requested_by": "admin",
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create sync log")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to create sync log",
		})
	}

	// Create appropriate payload based on sync type
	var taskInfo *asynq.TaskInfo

	switch syncType {
	case "full":
		payload := queue.SyncFullPayload{SyncLogID: syncLog.ID, RequestedBy: "admin"}
		taskInfo, err = h.queueManager.EnqueueSyncFull(payload)
	case "locations":
		payload := queue.SyncPayload{SyncLogID: syncLog.ID}
		taskInfo, err = h.queueManager.EnqueueSyncLocations(payload)
	case "nodes":
		payload := queue.SyncPayload{SyncLogID: syncLog.ID}
		taskInfo, err = h.queueManager.EnqueueSyncNodes(payload)
	case "allocations":
		payload := queue.SyncPayload{SyncLogID: syncLog.ID}
		taskInfo, err = h.queueManager.EnqueueSyncAllocations(payload)
	case "nests":
		payload := queue.SyncPayload{SyncLogID: syncLog.ID}
		taskInfo, err = h.queueManager.EnqueueSyncNests(payload)
	case "servers":
		payload := queue.SyncPayload{SyncLogID: syncLog.ID}
		taskInfo, err = h.queueManager.EnqueueSyncServers(payload)
	case "databases":
		payload := queue.SyncPayload{SyncLogID: syncLog.ID}
		taskInfo, err = h.queueManager.EnqueueSyncDatabases(payload)
	case "users":
		payload := queue.SyncPayload{SyncLogID: syncLog.ID}
		taskInfo, err = h.queueManager.EnqueueSyncUsers(payload)
	}

	if err != nil {
		log.Error().Err(err).Str("type", syncType).Msg("Failed to enqueue sync")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to enqueue sync",
		})
	}

	log.Info().Str("sync_log_id", syncLog.ID).Str("type", syncType).Str("task_id", taskInfo.ID).Msg("Sync enqueued from admin")

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success":     true,
		"sync_log_id": syncLog.ID,
		"task_id":     taskInfo.ID,
		"status":      "PENDING",
		"message":     "Sync has been queued",
	})
}

// CancelSyncAdmin handles POST /api/admin/sync/cancel
// @Summary Cancel sync (admin)
// @Description Requests cancellation of the currently running sync operation
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse "Cancellation requested"
// @Failure 404 {object} ErrorResponse "No active sync found"
// @Failure 400 {object} ErrorResponse "Sync cannot be cancelled"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/admin/sync/cancel [post]
func (h *AdminSyncHandler) CancelSyncAdmin(c *fiber.Ctx) error {
	// Get the latest sync that is in progress
	logs, err := h.syncRepo.GetSyncLogs(c.Context(), 1, 0, "")
	if err != nil || len(logs) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Success: false,
			Error:   "No sync found",
		})
	}

	latestLog := logs[0]
	if latestLog.Status != "PENDING" && latestLog.Status != "RUNNING" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Cannot cancel a sync that is not running",
		})
	}

	// Mark sync for cancellation - worker will check this and stop
	err = h.syncRepo.MarkSyncCancelled(c.Context(), latestLog.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to cancel sync")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to cancel sync",
		})
	}

	log.Info().Str("sync_log_id", latestLog.ID).Msg("Sync cancellation requested by admin")

	return c.JSON(fiber.Map{
		"success":     true,
		"sync_log_id": latestLog.ID,
		"message":     "Sync cancellation requested",
	})
}

// GetSyncSettingsAdmin handles GET /api/admin/sync/settings
// @Summary Get sync settings (admin)
// @Description Retrieves current sync automation settings
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse "Settings retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/admin/sync/settings [get]
func (h *AdminSyncHandler) GetSyncSettingsAdmin(c *fiber.Ctx) error {
	ctx := c.Context()

	var autoSyncEnabled string
	var autoSyncInterval int

	// Get auto sync enabled
	err := h.db.Pool.QueryRow(ctx, `SELECT value FROM config WHERE key = 'auto_sync_enabled' LIMIT 1`).Scan(&autoSyncEnabled)
	if err != nil {
		autoSyncEnabled = "false"
	}

	// Get auto sync interval
	err = h.db.Pool.QueryRow(ctx, `SELECT value FROM config WHERE key = 'auto_sync_interval' LIMIT 1`).Scan(&autoSyncInterval)
	if err != nil {
		autoSyncInterval = 3600
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"autoSyncEnabled":  autoSyncEnabled == "true",
			"autoSyncInterval": autoSyncInterval,
		},
	})
}

// UpdateSyncSettingsAdminRequest represents a request to update sync settings
type UpdateSyncSettingsAdminRequest struct {
	AutoSyncEnabled  *bool `json:"autoSyncEnabled,omitempty"`
	AutoSyncInterval *int  `json:"autoSyncInterval,omitempty"`
}

// UpdateSyncSettingsAdmin handles POST /api/admin/sync/settings
// @Summary Update sync settings (admin)
// @Description Updates sync automation settings (enabled status and interval)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body UpdateSyncSettingsAdminRequest true "Settings update parameters"
// @Success 200 {object} SuccessResponse "Settings updated"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/admin/sync/settings [post]
func (h *AdminSyncHandler) UpdateSyncSettingsAdmin(c *fiber.Ctx) error {
	ctx := c.Context()
	var req UpdateSyncSettingsAdminRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.AutoSyncEnabled != nil {
		enabled := "false"
		if *req.AutoSyncEnabled {
			enabled = "true"
		}

		_, err := h.db.Pool.Exec(ctx, `
			INSERT INTO config (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
		`, "auto_sync_enabled", enabled)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update auto_sync_enabled")
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Success: false,
				Error:   "Failed to update settings",
			})
		}
	}

	if req.AutoSyncInterval != nil && *req.AutoSyncInterval > 0 {
		_, err := h.db.Pool.Exec(ctx, `
			INSERT INTO config (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
		`, "auto_sync_interval", *req.AutoSyncInterval)
		if err != nil {
			log.Error().Err(err).Msg("Failed to update auto_sync_interval")
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Success: false,
				Error:   "Failed to update settings",
			})
		}
	}

	log.Info().Msg("Sync settings updated by admin")

	return c.JSON(SuccessResponse{
		Success: true,
		Message: "Settings updated successfully",
	})
}

// GetAdminStats handles GET /api/admin/stats
// @Summary Get admin statistics
// @Description Retrieves comprehensive statistics for admin dashboard
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse "Admin statistics retrieved"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/admin/stats [get]
func (h *StatsHandler) GetAdminStats(c *fiber.Ctx) error {
	ctx := c.Context()

	var totalServers, totalUsers, totalNodes, suspendedServers, totalAllocations, usedAllocations int

	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM servers").Scan(&totalServers)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM nodes").Scan(&totalNodes)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM servers WHERE \"isSuspended\" = true").Scan(&suspendedServers)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM allocations").Scan(&totalAllocations)
	h.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM allocations WHERE \"isAssigned\" = true").Scan(&usedAllocations)

	return c.JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"totalServers":         totalServers,
			"suspendedServers":     suspendedServers,
			"totalUsers":           totalUsers,
			"totalNodes":           totalNodes,
			"totalAllocations":     totalAllocations,
			"usedAllocations":      usedAllocations,
			"availableAllocations": totalAllocations - usedAllocations,
		},
	})
}
