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

	"github.com/nodebyte/backend/internal/config"
	"github.com/nodebyte/backend/internal/queue"
)

// EmailHandler handles email-related tasks
type EmailHandler struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(cfg *config.Config) *EmailHandler {
	return &EmailHandler{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ResendEmailRequest represents the Resend API request body
type ResendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text,omitempty"`
}

// HandleSendEmail processes an email send task
func (h *EmailHandler) HandleSendEmail(ctx context.Context, task *asynq.Task) error {
	var payload queue.EmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Info().
		Str("to", payload.To).
		Str("subject", payload.Subject).
		Str("template", payload.Template).
		Msg("Sending email")

	// Build HTML content based on template
	htmlContent := h.buildEmailHTML(payload.Template, payload.Data)

	// Prepare Resend API request
	reqBody := ResendEmailRequest{
		From:    h.cfg.EmailFrom,
		To:      []string{payload.To},
		Subject: payload.Subject,
		HTML:    htmlContent,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Send request to Resend API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.cfg.ResendAPIKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("resend API returned status %d", resp.StatusCode)
	}

	log.Info().
		Str("to", payload.To).
		Int("status", resp.StatusCode).
		Msg("Email sent successfully")

	return nil
}

// buildEmailHTML builds HTML content for email templates
func (h *EmailHandler) buildEmailHTML(template string, data map[string]string) string {
	// Base email template
	baseStyle := `
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.header { text-align: center; padding: 20px 0; }
		.logo { font-size: 24px; font-weight: bold; color: #6366f1; }
		.content { padding: 20px; background: #f9fafb; border-radius: 8px; }
		.button { display: inline-block; padding: 12px 24px; background: #6366f1; color: white; text-decoration: none; border-radius: 6px; margin: 20px 0; }
		.footer { text-align: center; padding: 20px; font-size: 12px; color: #6b7280; }
	`

	var content string

	switch template {
	case "password-reset":
		content = fmt.Sprintf(`
			<div class="content">
				<h2>Reset Your Password</h2>
				<p>Hello %s,</p>
				<p>We received a request to reset your password. Click the button below to create a new password:</p>
				<a href="%s" class="button">Reset Password</a>
				<p>If you didn't request this, you can safely ignore this email.</p>
				<p>This link will expire in 1 hour.</p>
			</div>
		`, data["name"], data["resetUrl"])

	case "email-verification":
		content = fmt.Sprintf(`
			<div class="content">
				<h2>Verify Your Email</h2>
				<p>Hello %s,</p>
				<p>Thanks for signing up! Please verify your email address by clicking the button below:</p>
				<a href="%s" class="button">Verify Email</a>
				<p>If you didn't create an account, you can safely ignore this email.</p>
			</div>
		`, data["name"], data["verifyUrl"])

	case "magic-link":
		content = fmt.Sprintf(`
			<div class="content">
				<h2>Sign In to NodeByte</h2>
				<p>Hello,</p>
				<p>Click the button below to sign in to your account:</p>
				<a href="%s" class="button">Sign In</a>
				<p>This link will expire in 15 minutes.</p>
				<p>If you didn't request this, you can safely ignore this email.</p>
			</div>
		`, data["magicLinkUrl"])

	case "sync-complete":
		content = fmt.Sprintf(`
			<div class="content">
				<h2>Sync Completed</h2>
				<p>A sync operation has completed on NodeByte.</p>
				<p><strong>Type:</strong> %s</p>
				<p><strong>Status:</strong> %s</p>
				<p><strong>Duration:</strong> %s</p>
			</div>
		`, data["syncType"], data["status"], data["duration"])

	default:
		content = fmt.Sprintf(`
			<div class="content">
				<p>%s</p>
			</div>
		`, data["message"])
	}

	return fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<style>%s</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<div class="logo">NodeByte</div>
				</div>
				%s
				<div class="footer">
					<p>© %d NodeByte. All rights reserved.</p>
					<p>This is an automated message, please do not reply.</p>
				</div>
			</div>
		</body>
		</html>
	`, baseStyle, content, time.Now().Year())
}
