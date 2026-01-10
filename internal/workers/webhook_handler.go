package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/queue"
)

// WebhookHandler handles webhook dispatch tasks
type WebhookHandler struct {
	db         *database.DB
	httpClient *http.Client
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(db *database.DB) *WebhookHandler {
	return &WebhookHandler{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DiscordWebhookPayload represents a Discord webhook message
type DiscordWebhookPayload struct {
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Content   string         `json:"content,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	URL         string              `json:"url,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Author      *DiscordEmbedAuthor `json:"author,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
}

// DiscordEmbedFooter represents a Discord embed footer
type DiscordEmbedFooter struct {
	Text    string `json:"text,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedAuthor represents a Discord embed author
type DiscordEmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedField represents a Discord embed field
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// HandleDiscordWebhook processes a Discord webhook task
func (h *WebhookHandler) HandleDiscordWebhook(ctx context.Context, task *asynq.Task) error {
	var payload queue.WebhookPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Info().
		Str("webhook_id", payload.WebhookID).
		Str("event", payload.Event).
		Msg("Processing Discord webhook")

	// Get webhook URL from database
	var webhookURL string
	var enabled bool
	query := `SELECT url, enabled FROM discord_webhooks WHERE id = $1`
	err := h.db.Pool.QueryRow(ctx, query, payload.WebhookID).Scan(&webhookURL, &enabled)
	if err != nil {
		return fmt.Errorf("failed to get webhook: %w", err)
	}

	if !enabled {
		log.Warn().Str("webhook_id", payload.WebhookID).Msg("Webhook is disabled, skipping")
		return nil
	}

	// Build Discord message based on event type
	message := h.buildDiscordMessage(payload.Event, payload.Data)

	// Send to Discord
	jsonBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Discord rate limiting - retry later
		if resp.StatusCode == 429 {
			return fmt.Errorf("rate limited by Discord")
		}
		return fmt.Errorf("Discord returned status %d", resp.StatusCode)
	}

	log.Info().
		Str("webhook_id", payload.WebhookID).
		Str("event", payload.Event).
		Int("status", resp.StatusCode).
		Msg("Discord webhook sent successfully")

	return nil
}

// buildDiscordMessage creates a Discord message based on event type
func (h *WebhookHandler) buildDiscordMessage(event string, data map[string]interface{}) DiscordWebhookPayload {
	message := DiscordWebhookPayload{
		Username:  "NodeByte",
		AvatarURL: "https://nodebyte.host/logo.png",
	}

	var embed DiscordEmbed
	embed.Timestamp = time.Now().UTC().Format(time.RFC3339)
	embed.Footer = &DiscordEmbedFooter{
		Text: "NodeByte Notifications",
	}

	switch event {
	case "sync.started":
		embed.Title = "🔄 Sync Started"
		embed.Description = "A synchronization operation has started."
		embed.Color = 0x3B82F6 // Blue
		if syncType, ok := data["type"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Type",
				Value:  syncType,
				Inline: true,
			})
		}

	case "sync.completed":
		embed.Title = "✅ Sync Completed"
		embed.Description = "Synchronization completed successfully."
		embed.Color = 0x22C55E // Green
		if syncType, ok := data["type"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Type",
				Value:  syncType,
				Inline: true,
			})
		}
		if duration, ok := data["duration"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Duration",
				Value:  duration,
				Inline: true,
			})
		}

	case "sync.failed":
		embed.Title = "❌ Sync Failed"
		embed.Description = "A synchronization operation has failed."
		embed.Color = 0xEF4444 // Red
		if errorMsg, ok := data["error"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:  "Error",
				Value: errorMsg,
			})
		}

	case "user.registered":
		embed.Title = "👤 New User Registered"
		embed.Description = "A new user has registered on NodeByte."
		embed.Color = 0x8B5CF6 // Purple
		if email, ok := data["email"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Email",
				Value:  email,
				Inline: true,
			})
		}

	case "server.created":
		embed.Title = "🖥️ Server Created"
		embed.Description = "A new server has been created."
		embed.Color = 0x6366F1 // Indigo
		if name, ok := data["name"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Name",
				Value:  name,
				Inline: true,
			})
		}
		if owner, ok := data["owner"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Owner",
				Value:  owner,
				Inline: true,
			})
		}

	case "server.suspended":
		embed.Title = "⚠️ Server Suspended"
		embed.Description = "A server has been suspended."
		embed.Color = 0xF59E0B // Amber
		if name, ok := data["name"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Name",
				Value:  name,
				Inline: true,
			})
		}
		if reason, ok := data["reason"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:  "Reason",
				Value: reason,
			})
		}

	case "support.ticket_created":
		embed.Title = "🎫 New Support Ticket"
		embed.Description = "A new support ticket has been created."
		embed.Color = 0x0EA5E9 // Sky
		if subject, ok := data["subject"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:  "Subject",
				Value: subject,
			})
		}
		if user, ok := data["user"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "User",
				Value:  user,
				Inline: true,
			})
		}
		if priority, ok := data["priority"].(string); ok {
			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   "Priority",
				Value:  priority,
				Inline: true,
			})
		}

	default:
		embed.Title = "📢 Notification"
		embed.Description = fmt.Sprintf("Event: %s", event)
		embed.Color = 0x6B7280 // Gray
	}

	message.Embeds = []DiscordEmbed{embed}
	return message
}
