package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
)

// AdminWebhooksHandler handles admin webhook management
type AdminWebhooksHandler struct {
	db *database.DB
}

// NewAdminWebhooksHandler creates a new admin webhooks handler
func NewAdminWebhooksHandler(db *database.DB) *AdminWebhooksHandler {
	return &AdminWebhooksHandler{db: db}
}

// DiscordWebhookDTO represents a Discord webhook
type DiscordWebhookDTO struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	WebhookURL    string     `json:"webhookUrl"`
	Type          string     `json:"type"`
	Scope         string     `json:"scope"`
	Description   string     `json:"description"`
	Enabled       bool       `json:"enabled"`
	TestSuccessAt *time.Time `json:"testSuccessAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// GetWebhooks returns all discord webhooks
// @Summary Get webhooks
// @Description Returns list of configured Discord webhooks
// @Tags Admin Settings
// @Produce json
// @Success 200 {object} map[string]interface{} "Webhooks retrieved"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /api/admin/settings/webhooks [get]
// @Security Bearer
func (h *AdminWebhooksHandler) GetWebhooks(c *fiber.Ctx) error {
	query := `
		SELECT id, name, "webhookUrl", type, scope, description, enabled, "testSuccessAt", "createdAt"
		FROM discord_webhooks
		ORDER BY "createdAt" DESC
	`

	rows, err := h.db.Pool.Query(c.Context(), query)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch webhooks",
		})
	}
	defer rows.Close()

	var webhooks []DiscordWebhookDTO
	for rows.Next() {
		var wh DiscordWebhookDTO
		if err := rows.Scan(&wh.ID, &wh.Name, &wh.WebhookURL, &wh.Type, &wh.Scope, &wh.Description, &wh.Enabled, &wh.TestSuccessAt, &wh.CreatedAt); err != nil {
			continue
		}
		webhooks = append(webhooks, wh)
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"webhooks": webhooks,
	})
}

// CreateWebhook creates a new Discord webhook
// @Summary Create webhook
// @Description Creates a new Discord webhook for notifications
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body object true "Webhook info"
// @Success 200 {object} map[string]interface{} "Webhook created"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /api/admin/settings/webhooks [post]
// @Security Bearer
func (h *AdminWebhooksHandler) CreateWebhook(c *fiber.Ctx) error {
	var req struct {
		Name        string `json:"name"`
		WebhookURL  string `json:"webhookUrl"`
		Type        string `json:"type"`
		Scope       string `json:"scope"`
		Description string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.Name == "" || req.WebhookURL == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "name and webhookUrl are required",
		})
	}

	// Validate webhook URL
	if !isValidDiscordWebhookURL(req.WebhookURL) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid Discord webhook URL",
		})
	}

	// Set defaults
	if req.Type == "" {
		req.Type = "SYSTEM"
	}
	if req.Scope == "" {
		req.Scope = "ADMIN"
	}

	webhookID := uuid.New().String()
	query := `
		INSERT INTO discord_webhooks (id, name, "webhookUrl", type, scope, description, enabled, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), NOW())
	`

	_, err := h.db.Pool.Exec(c.Context(), query,
		webhookID, req.Name, req.WebhookURL, req.Type, req.Scope, req.Description,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create webhook")
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to create webhook",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"webhook": fiber.Map{
			"id":         webhookID,
			"name":       req.Name,
			"webhookUrl": req.WebhookURL,
			"type":       req.Type,
			"scope":      req.Scope,
		},
	})
}

// UpdateWebhook updates a Discord webhook
// @Summary Update webhook
// @Description Updates an existing Discord webhook
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body object true "Webhook info"
// @Success 200 {object} map[string]interface{} "Webhook updated"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /api/admin/settings/webhooks [put]
// @Security Bearer
func (h *AdminWebhooksHandler) UpdateWebhook(c *fiber.Ctx) error {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		WebhookURL  string `json:"webhookUrl"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Enabled     *bool  `json:"enabled"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.ID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "id is required",
		})
	}

	// Build dynamic update query
	query := `UPDATE discord_webhooks SET "updatedAt" = NOW()`
	args := []interface{}{}
	paramCount := 1

	if req.Name != "" {
		paramCount++
		query += `, name = $` + fmt.Sprintf("%d", paramCount)
		args = append(args, req.Name)
	}

	if req.WebhookURL != "" {
		if !isValidDiscordWebhookURL(req.WebhookURL) {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid Discord webhook URL",
			})
		}
		paramCount++
		query += `, "webhookUrl" = $` + fmt.Sprintf("%d", paramCount)
		args = append(args, req.WebhookURL)
	}

	if req.Type != "" {
		paramCount++
		query += `, type = $` + fmt.Sprintf("%d", paramCount)
		args = append(args, req.Type)
	}

	if req.Description != "" {
		paramCount++
		query += `, description = $` + fmt.Sprintf("%d", paramCount)
		args = append(args, req.Description)
	}

	if req.Enabled != nil {
		paramCount++
		query += `, enabled = $` + fmt.Sprintf("%d", paramCount)
		args = append(args, *req.Enabled)
	}

	paramCount++
	query += ` WHERE id = $` + fmt.Sprintf("%d", paramCount)
	args = append(args, req.ID)

	_, err := h.db.Pool.Exec(c.Context(), query, args...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update webhook")
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to update webhook",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Webhook updated",
	})
}

// DeleteWebhook deletes a Discord webhook
// @Summary Delete webhook
// @Description Deletes a Discord webhook
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body object true "Webhook id"
// @Success 200 {object} map[string]interface{} "Webhook deleted"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /api/admin/settings/webhooks [delete]
// @Security Bearer
func (h *AdminWebhooksHandler) DeleteWebhook(c *fiber.Ctx) error {
	var req struct {
		ID string `json:"id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.ID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "id is required",
		})
	}

	_, err := h.db.Pool.Exec(c.Context(), `DELETE FROM discord_webhooks WHERE id = $1`, req.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete webhook")
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to delete webhook",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Webhook deleted",
	})
}

// TestWebhook tests a Discord webhook
// @Summary Test webhook
// @Description Tests webhook connectivity
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body object true "Webhook id"
// @Success 200 {object} map[string]interface{} "Test result"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/admin/settings/webhooks [patch]
// @Security Bearer
func (h *AdminWebhooksHandler) TestWebhook(c *fiber.Ctx) error {
	var req struct {
		ID string `json:"id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.ID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "id is required",
		})
	}

	// Get webhook
	var webhookURL string
	err := h.db.Pool.QueryRow(c.Context(), `SELECT "webhookUrl" FROM discord_webhooks WHERE id = $1`, req.ID).Scan(&webhookURL)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Webhook not found",
		})
	}

	// Test webhook with proper payload
	testPayload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       "🧪 Webhook Test",
				"description": "This is a test message to verify webhook connectivity",
				"color":       3066993, // Green
				"fields": []map[string]interface{}{
					{
						"name":   "Status",
						"value":  "✅ Webhook is working correctly",
						"inline": true,
					},
					{
						"name":   "Test Time",
						"value":  time.Now().Format(time.RFC3339),
						"inline": true,
					},
				},
				"timestamp": time.Now().Format(time.RFC3339),
				"footer": map[string]string{
					"text": "NodeByte Webhook Test",
				},
			},
		},
	}

	payloadBytes, _ := json.Marshal(testPayload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return c.JSON(fiber.Map{
			"success": false,
			"message": "Failed to reach Discord webhook endpoint",
		})
	}
	defer resp.Body.Close()

	success := resp.StatusCode == http.StatusNoContent
	if success {
		// Update testSuccessAt timestamp
		h.db.Pool.Exec(c.Context(), `UPDATE discord_webhooks SET "testSuccessAt" = NOW() WHERE id = $1`, req.ID)
	}

	return c.JSON(fiber.Map{
		"success": success,
		"message": map[bool]string{
			true:  "Webhook test successful",
			false: fmt.Sprintf("Webhook test failed with status: %d", resp.StatusCode),
		}[success],
	})
}

// Helper functions

// isValidDiscordWebhookURL validates if a URL is a valid Discord webhook URL.
// Accepts discord.com directly or proxy URLs (e.g. gateway.nodebyte.host/proxy/discord/webhooks/...)
// as long as the URL is HTTPS and the path contains "/webhooks/".
func isValidDiscordWebhookURL(webhookURL string) bool {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return false
	}
	if parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return false
	}
	// Allow discord.com directly or any proxy that forwards to the webhooks endpoint
	return parsedURL.Host == "discord.com" || strings.Contains(parsedURL.Path, "/webhooks/")
}
