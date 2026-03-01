package handlers

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
)

// SyncStreamHandler streams live sync progress via Server-Sent Events.
type SyncStreamHandler struct {
	db       *database.DB
	syncRepo *database.SyncRepository
}

// NewSyncStreamHandler creates a new SyncStreamHandler.
func NewSyncStreamHandler(db *database.DB) *SyncStreamHandler {
	return &SyncStreamHandler{
		db:       db,
		syncRepo: database.NewSyncRepository(db),
	}
}

// validateQueryToken validates a JWT token supplied as a query parameter.
// Used by SSE endpoints because EventSource cannot send custom headers.
// Returns the userID on success.
func (h *SyncStreamHandler) validateQueryToken(db *database.DB, token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	decodedPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid token encoding")
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decodedPayload, &claims); err != nil {
		return "", fmt.Errorf("invalid token claims")
	}

	userID, ok := claims["id"].(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("missing user ID in token")
	}

	return userID, nil
}

// StreamSyncProgress streams sync log updates via Server-Sent Events.
// The token is supplied as a ?token= query parameter since browsers cannot
// set custom headers on EventSource connections.
//
// @Summary Stream sync progress (SSE)
// @Description Streams live sync log updates as Server-Sent Events until the sync reaches a terminal state
// @Tags Admin
// @Produce text/event-stream
// @Param id path string true "Sync log ID"
// @Param token query string true "Bearer JWT token"
// @Router /api/admin/sync/stream/{id} [get]
func (h *SyncStreamHandler) StreamSyncProgress(c *fiber.Ctx) error {
	// --- Auth via query param ---
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
	}

	userID, err := h.validateQueryToken(h.db, token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var isSystemAdmin bool
	if err := h.db.Pool.QueryRow(c.Context(),
		`SELECT "isSystemAdmin" FROM users WHERE id = $1 LIMIT 1`,
		userID,
	).Scan(&isSystemAdmin); err != nil || !isSystemAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}

	// --- Validate sync log ---
	syncLogID := c.Params("id")
	if syncLogID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "sync log ID required"})
	}

	if _, err := h.syncRepo.GetSyncLog(c.Context(), syncLogID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sync log not found"})
	}

	// --- SSE headers ---
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // disable nginx buffering

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()

		ctx := context.Background()
		var lastMetaUpdate int64
		heartbeat := 0

		// Initial connected event
		fmt.Fprintf(w, "event: connected\ndata: {\"syncLogId\":\"%s\"}\n\n", syncLogID)
		w.Flush()

		for range ticker.C {
			syncLog, err := h.syncRepo.GetSyncLog(ctx, syncLogID)
			if err != nil {
				log.Error().Err(err).Str("sync_log_id", syncLogID).Msg("SSE: failed to fetch sync log")
				fmt.Fprintf(w, "event: error\ndata: {\"error\":\"log not found\"}\n\n")
				w.Flush()
				return
			}

			// Parse metadata
			var meta map[string]interface{}
			if syncLog.Metadata != "" {
				_ = json.Unmarshal([]byte(syncLog.Metadata), &meta)
			}
			if meta == nil {
				meta = map[string]interface{}{}
			}

			// Determine if anything changed
			curUpdate := int64(0)
			if v, ok := meta["lastUpdated"].(float64); ok {
				curUpdate = int64(v)
			}

			if curUpdate == lastMetaUpdate && syncLog.Status == "PENDING" {
				// Send a heartbeat comment every ~2 s so the connection stays alive
				heartbeat++
				if heartbeat%5 == 0 {
					fmt.Fprintf(w, ": ping\n\n")
					w.Flush()
				}
				continue
			}

			lastMetaUpdate = curUpdate
			heartbeat = 0

			payload, _ := json.Marshal(map[string]interface{}{
				"id":          syncLog.ID,
				"status":      syncLog.Status,
				"itemsTotal":  syncLog.ItemsTotal,
				"itemsSynced": syncLog.ItemsSynced,
				"metadata":    meta,
			})

			fmt.Fprintf(w, "event: update\ndata: %s\n\n", payload)
			w.Flush()

			// Terminal state → send done event then close
			if syncLog.Status == "COMPLETED" || syncLog.Status == "FAILED" || syncLog.Status == "CANCELLED" {
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", payload)
				w.Flush()
				return
			}
		}
	})

	return nil
}
