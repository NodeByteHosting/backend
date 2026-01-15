package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/sentry"
	"github.com/nodebyte/backend/internal/types"
)

// HytaleServerLogsHandler handles Hytale game server logs API requests
type HytaleServerLogsHandler struct {
	db *database.DB
}

// NewHytaleServerLogsHandler creates a new Hytale server logs handler
func NewHytaleServerLogsHandler(db *database.DB) *HytaleServerLogsHandler {
	return &HytaleServerLogsHandler{
		db: db,
	}
}

// CreateServerLogs receives and stores game server logs from Wings/Panel
// @Summary Create game server logs
// @Description Stores console output logs sent from Wings daemon or Panel
// @Tags Hytale Logs
// @Accept json
// @Produce json
// @Param payload body types.CreateServerLogsRequest true "Logs to store"
// @Success 201 {object} types.SuccessResponse "Logs stored successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/server-logs [post]
func (h *HytaleServerLogsHandler) CreateServerLogs(c *fiber.Ctx) error {
	span := sentry.StartSpan(c.Context(), "create_server_logs", "http")
	defer span.Finish()
	ctx := span.Context()

	var req types.CreateServerLogsRequest
	if err := c.BodyParser(&req); err != nil {
		sentry.SetTag(c, "error_type", "invalid_request")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Invalid request format",
		})
	}

	// Validate required fields
	if req.ServerUUID == "" || req.AccountID == "" {
		sentry.SetTag(c, "error_type", "missing_field")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "server_uuid and account_id are required",
		})
	}

	if len(req.Logs) == 0 {
		sentry.SetTag(c, "error_type", "empty_logs")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "At least one log entry is required",
		})
	}

	log.Debug().
		Str("server_uuid", req.ServerUUID).
		Int("log_count", len(req.Logs)).
		Msg("Received logs from Wings")

	// Convert request logs to database format
	dbLogs := make([]string, 0, len(req.Logs))
	for _, logEntry := range req.Logs {
		dbLogs = append(dbLogs, logEntry.Line)
	}

	// Store logs in database
	logsRepo := database.NewHytaleServerLogsRepository(h.db)
	if err := logsRepo.SaveLogs(ctx, req.ServerUUID, req.AccountID, dbLogs); err != nil {
		log.Error().Err(err).
			Str("server_uuid", req.ServerUUID).
			Msg("Failed to save server logs")

		sentry.CaptureErrorWithContext(c, err, http.StatusInternalServerError, "save_server_logs")
		return c.Status(http.StatusInternalServerError).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to store logs",
		})
	}

	sentry.SetTag(c, "log_count", strconv.Itoa(len(dbLogs)))
	return c.Status(http.StatusCreated).JSON(types.SuccessResponse{
		Success: true,
		Data: map[string]interface{}{
			"saved_count": len(dbLogs),
			"server_uuid": req.ServerUUID,
		},
	})
}

// GetHytaleServerLogs retrieves persistent logs for a game server

// @Summary Get Hytale game server logs
// @Description Retrieves stored console output logs from a Hytale game server
// @Tags Hytale Logs
// @Accept json
// @Produce json
// @Param server_uuid query string true "Server UUID"
// @Param limit query int false "Maximum logs to return (default: 100, max: 1000)"
// @Param offset query int false "Offset for pagination (default: 0)"
// @Success 200 {object} types.SuccessResponse{data=[]types.ServerLog} "Server logs retrieved successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid query parameters"
// @Failure 404 {object} types.ErrorResponse "Server not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/server-logs [get]
func (h *HytaleServerLogsHandler) GetHytaleServerLogs(c *fiber.Ctx) error {
	span := sentry.StartSpan(c.Context(), "get_hytale_server_logs", "http")
	defer span.Finish()
	ctx := span.Context()

	// Get query parameters
	serverUUID := c.Query("server_uuid")
	limitStr := c.Query("limit", "100")
	offsetStr := c.Query("offset", "0")

	// Validate required parameters
	if serverUUID == "" {
		sentry.SetTag(c, "error_type", "missing_parameter")
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "server_uuid query parameter is required",
		})
	}

	// Parse and validate limit
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	// Parse and validate offset
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	log.Info().
		Str("server_uuid", serverUUID).
		Int("limit", limit).
		Int("offset", offset).
		Msg("Fetching Hytale server logs")

	// Get logs from database
	logsRepo := database.NewHytaleServerLogsRepository(h.db)
	serverLogs, err := logsRepo.GetLogsByServer(ctx, serverUUID, limit, offset)
	if err != nil {
		log.Error().Err(err).
			Str("server_uuid", serverUUID).
			Msg("Failed to fetch server logs from database")

		sentry.CaptureErrorWithContext(c, err, http.StatusInternalServerError, "fetch_server_logs")
		return c.Status(http.StatusInternalServerError).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to fetch server logs",
		})
	}

	// Convert to response format
	logs := make([]types.ServerLog, 0, len(serverLogs))
	for _, serverLog := range serverLogs {
		logs = append(logs, types.ServerLog{
			ID:        serverLog.ID,
			Line:      serverLog.LogLine,
			Timestamp: serverLog.CreatedAt,
		})
	}

	sentry.SetTag(c, "log_count", strconv.Itoa(len(logs)))
	return c.Status(http.StatusOK).JSON(types.SuccessResponse{
		Success: true,
		Data:    logs,
	})
}

// GetHytaleServerLogsCount retrieves the total count of logs for a server
// @Summary Get Hytale server logs count
// @Description Returns the total number of stored logs for a specific server
// @Tags Hytale Logs
// @Accept json
// @Produce json
// @Param server_uuid query string true "Server UUID"
// @Success 200 {object} types.SuccessResponse{data=map[string]interface{}} "Log count retrieved successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid parameters"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/server-logs/count [get]
func (h *HytaleServerLogsHandler) GetHytaleServerLogsCount(c *fiber.Ctx) error {
	span := sentry.StartSpan(c.Context(), "get_hytale_server_logs_count", "http")
	defer span.Finish()
	ctx := span.Context()

	serverUUID := c.Query("server_uuid")
	if serverUUID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "server_uuid query parameter is required",
		})
	}

	logsRepo := database.NewHytaleServerLogsRepository(h.db)
	count, err := logsRepo.CountLogsByServer(ctx, serverUUID)
	if err != nil {
		log.Error().Err(err).
			Str("server_uuid", serverUUID).
			Msg("Failed to count server logs")

		sentry.CaptureErrorWithContext(c, err, http.StatusInternalServerError, "count_server_logs")
		return c.Status(http.StatusInternalServerError).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to count server logs",
		})
	}

	return c.Status(http.StatusOK).JSON(types.SuccessResponse{
		Success: true,
		Data: map[string]interface{}{
			"server_uuid": serverUUID,
			"total_logs":  count,
		},
	})
}
