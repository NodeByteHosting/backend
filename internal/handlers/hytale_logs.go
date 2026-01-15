package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/types"
)

// HytaleLogsHandler handles Hytale audit log retrieval
type HytaleLogsHandler struct {
	db *database.DB
}

// NewHytaleLogsHandler creates a new Hytale logs handler
func NewHytaleLogsHandler(db *database.DB) *HytaleLogsHandler {
	return &HytaleLogsHandler{db: db}
}

// GetHytaleLogs retrieves audit logs for a Hytale account
// @Summary Get Hytale Audit Logs
// @Description Retrieves audit logs for a specific Hytale account
// @Tags Hytale Logs
// @Accept json
// @Produce json
// @Param account_id query string true "Account UUID"
// @Param limit query int false "Maximum number of logs (default: 100, max: 1000)"
// @Success 200 {object} types.GetHytaleLogsResponse
// @Failure 400 {object} types.ErrorResponse "Missing account_id or invalid parameters"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/hytale/logs [get]
func (h *HytaleLogsHandler) GetHytaleLogs(c *fiber.Ctx) error {
	accountID := c.Query("account_id")
	if accountID == "" {
		return c.Status(http.StatusBadRequest).JSON(types.ErrorResponse{
			Success: false,
			Error:   "account_id is required",
		})
	}

	limitStr := c.Query("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	auditRepo := database.NewHytaleAuditLogRepository(h.db)
	logs, err := auditRepo.GetAuditLogs(c.Context(), accountID, limit)
	if err != nil {
		log.Error().Err(err).Str("account_id", accountID).Msg("Failed to retrieve Hytale audit logs")
		return c.Status(http.StatusInternalServerError).JSON(types.ErrorResponse{
			Success: false,
			Error:   "Failed to retrieve logs",
		})
	}

	return c.JSON(types.GetHytaleLogsResponse{
		Success: true,
		Logs:    logs,
		Count:   len(logs),
	})
}
