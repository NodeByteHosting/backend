package database

import (
	"context"
)

// DiscordWebhookInput represents input for creating/updating webhooks
type DiscordWebhookInput struct {
	Name        string `json:"name"`
	WebhookUrl  string `json:"webhook_url"`
	WebhookURL  string `json:"webhookUrl"`
	Type        string `json:"type"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// GetDiscordWebhooks retrieves all Discord webhooks
func (db *DB) GetDiscordWebhooks(ctx context.Context) ([]interface{}, error) {
	rows, err := db.Pool.Query(ctx, "SELECT id, \"webhookUrl\", name FROM \"discord_webhooks\" WHERE \"deletedAt\" IS NULL ORDER BY \"createdAt\" DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []interface{}
	for rows.Next() {
		var id, webhookUrl, name string
		if err := rows.Scan(&id, &webhookUrl, &name); err != nil {
			return nil, err
		}
		webhooks = append(webhooks, map[string]interface{}{
			"id":   id,
			"url":  webhookUrl,
			"name": name,
		})
	}
	return webhooks, rows.Err()
}

// CreateDiscordWebhook creates a new Discord webhook
func (db *DB) CreateDiscordWebhook(ctx context.Context, input DiscordWebhookInput) (map[string]interface{}, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		"INSERT INTO \"discord_webhooks\" (id, name, \"webhookUrl\", type, scope, enabled, \"createdAt\", \"updatedAt\") VALUES (gen_random_uuid(), $1, $2, $3, $4, true, NOW(), NOW()) RETURNING id",
		input.Name, input.WebhookUrl, input.Type, input.Scope).Scan(&id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":   id,
		"name": input.Name,
		"url":  input.WebhookUrl,
	}, nil
}

// UpdateDiscordWebhook updates a Discord webhook
func (db *DB) UpdateDiscordWebhook(ctx context.Context, id string, input DiscordWebhookInput) (map[string]interface{}, error) {
	_, err := db.Pool.Exec(ctx,
		"UPDATE \"discord_webhooks\" SET name = $1, \"webhookUrl\" = $2, type = $3, scope = $4, \"updatedAt\" = NOW() WHERE id = $5",
		input.Name, input.WebhookUrl, input.Type, input.Scope, id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":   id,
		"name": input.Name,
		"url":  input.WebhookUrl,
	}, nil
}

// GetDiscordWebhookByID retrieves a single Discord webhook
func (db *DB) GetDiscordWebhookByID(ctx context.Context, id string) (map[string]interface{}, error) {
	var webhookUrl, name string
	err := db.Pool.QueryRow(ctx,
		"SELECT \"webhookUrl\", name FROM \"discord_webhooks\" WHERE id = $1 AND \"deletedAt\" IS NULL", id).
		Scan(&webhookUrl, &name)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":   id,
		"url":  webhookUrl,
		"name": name,
	}, nil
}

// UpdateDiscordWebhookTestTime updates the last test time for a webhook
func (db *DB) UpdateDiscordWebhookTestTime(ctx context.Context, id string) (map[string]interface{}, error) {
	var webhookUrl, name string
	err := db.Pool.QueryRow(ctx,
		"UPDATE \"discord_webhooks\" SET \"testSuccessAt\" = NOW(), \"updatedAt\" = NOW() WHERE id = $1 RETURNING \"webhookUrl\", name",
		id).Scan(&webhookUrl, &name)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":   id,
		"name": name,
		"url":  webhookUrl,
	}, nil
}

// DeleteDiscordWebhook soft-deletes a Discord webhook
func (db *DB) DeleteDiscordWebhook(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx,
		"UPDATE \"discord_webhooks\" SET \"deletedAt\" = NOW(), \"updatedAt\" = NOW() WHERE id = $1",
		id)
	return err
}
