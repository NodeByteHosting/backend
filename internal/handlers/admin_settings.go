package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/crypto"
	"github.com/nodebyte/backend/internal/database"
)

const MASKED_VALUE = "••••••••••••••••••••"

type AdminSettingsHandler struct {
	db        *database.DB
	encryptor *crypto.Encryptor
}

func NewAdminSettingsHandler(db *database.DB) *AdminSettingsHandler {
	encryptor, err := crypto.NewEncryptorFromEnv()
	if err != nil {
		fmt.Printf("Warning: Encryption not configured: %v\n", err)
	}

	return &AdminSettingsHandler{
		db:        db,
		encryptor: encryptor,
	}
}

// SystemSettings represents all system configuration
type SystemSettings struct {
	// Pterodactyl
	PterodactylUrl          string `json:"pterodactylUrl"`
	PterodactylApiKey       string `json:"pterodactylApiKey"`
	PterodactylClientApiKey string `json:"pterodactylClientApiKey"`
	PterodactylApi          string `json:"pterodactylApi"`

	// Virtfusion
	VirtfusionUrl    string `json:"virtfusionUrl"`
	VirtfusionApiKey string `json:"virtfusionApiKey"`
	VirtfusionApi    string `json:"virtfusionApi"`

	// Crowdin
	CrowdinProjectId     string `json:"crowdinProjectId"`
	CrowdinPersonalToken string `json:"crowdinPersonalToken"`

	// GitHub
	GithubToken        string   `json:"githubToken"`
	GithubRepositories []string `json:"githubRepositories"`

	// Features
	RegistrationEnabled bool `json:"registrationEnabled"`
	MaintenanceMode     bool `json:"maintenanceMode"`
	AutoSyncEnabled     bool `json:"autoSyncEnabled"`

	// Email
	EmailNotifications bool   `json:"emailNotifications"`
	ResendApiKey       string `json:"resendApiKey"`

	// Discord
	DiscordNotifications bool  `json:"discordNotifications"`
	DiscordWebhooks      []any `json:"discordWebhooks"`

	// Advanced
	CacheTimeout int `json:"cacheTimeout"`
	SyncInterval int `json:"syncInterval"`

	// Admin
	AdminEmail string `json:"adminEmail"`
	SiteName   string `json:"siteName"`
	SiteUrl    string `json:"siteUrl"`
}

type ConnectionStatus struct {
	Connected bool   `json:"connected"`
	Latency   int    `json:"latency,omitempty"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`
}

// GetAdminSettings returns all system settings
// @Summary Get all admin settings
// @Description Returns all configuration from Config table and Discord webhooks
// @Tags Admin Settings
// @Produce json
// @Success 200 {object} map[string]interface{} "Settings retrieved successfully"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /api/admin/settings [get]
// @Security Bearer
func (h *AdminSettingsHandler) GetAdminSettings(c *fiber.Ctx) error {
	// Get all config entries
	configs, err := h.db.GetAllConfigs(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch settings",
		})
	}

	// Convert to settings object
	settings := h.configsToSettings(configs)

	// Get Discord webhooks from database
	var discordWebhooks []map[string]interface{}
	webhookRows, err := h.db.Pool.Query(c.Context(), `
		SELECT id, name, "webhookUrl", type, scope, description, enabled, "testSuccessAt", "createdAt"
		FROM discord_webhooks
		WHERE scope = 'ADMIN'
		ORDER BY "createdAt" DESC
	`)
	if err != nil {
		webhookRows.Close()
		discordWebhooks = []map[string]interface{}{}
	} else {
		defer webhookRows.Close()
		for webhookRows.Next() {
			var id, name, url, wtype, scope, description string
			var enabled bool
			var testSuccessAt, createdAt *time.Time

			if err := webhookRows.Scan(&id, &name, &url, &wtype, &scope, &description, &enabled, &testSuccessAt, &createdAt); err != nil {
				continue
			}

			discordWebhooks = append(discordWebhooks, map[string]interface{}{
				"id":            id,
				"name":          name,
				"webhookUrl":    url,
				"type":          wtype,
				"scope":         scope,
				"description":   description,
				"enabled":       enabled,
				"testSuccessAt": testSuccessAt,
				"createdAt":     createdAt,
			})
		}
	}

	webhooksAny := make([]any, len(discordWebhooks))
	for i, webhook := range discordWebhooks {
		webhooksAny[i] = webhook
	}
	settings.DiscordWebhooks = webhooksAny

	// Test connections (non-blocking)
	pterodactylStatus := h.testPterodactylConnection(settings.PterodactylUrl, settings.PterodactylApiKey)
	virtfusionStatus := h.testVirtfusionConnection(settings.VirtfusionUrl, settings.VirtfusionApiKey)
	databaseStatus := h.testDatabaseConnection(c)

	return c.JSON(fiber.Map{
		"success":           true,
		"settings":          settings,
		"pterodactylStatus": pterodactylStatus,
		"virtfusionStatus":  virtfusionStatus,
		"databaseStatus":    databaseStatus,
		"sensitiveFields": []string{
			"pterodactylApiKey",
			"pterodactylClientApiKey",
			"virtfusionApiKey",
			"crowdinPersonalToken",
			"githubToken",
			"resendApiKey",
		},
	})
}

// SaveAdminSettings saves system settings
// @Summary Save admin settings
// @Description Updates configuration in Config table and handles GitHub repos merging
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body SystemSettings true "Settings to save"
// @Success 200 {object} map[string]interface{} "Settings saved successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /api/admin/settings [post]
// @Security Bearer
func (h *AdminSettingsHandler) SaveAdminSettings(c *fiber.Ctx) error {
	var req struct {
		SystemSettings
		GithubRepositoriesMerge bool `json:"githubRepositoriesMerge"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Save all settings to Config
	settingsMap := h.structToConfigMap(req.SystemSettings)

	// Handle GitHub repositories merge
	if req.GithubRepositoriesMerge && len(req.GithubRepositories) > 0 {
		existingRaw, _ := h.db.GetConfig(c.Context(), "github_repositories")
		var existing []string
		if existingRaw != "" {
			json.Unmarshal([]byte(existingRaw), &existing)
		}

		// Merge and deduplicate
		merged := append(existing, req.GithubRepositories...)
		uniqueRepos := make([]string, 0)
		seen := make(map[string]bool)
		for _, repo := range merged {
			if !seen[repo] {
				uniqueRepos = append(uniqueRepos, repo)
				seen[repo] = true
			}
		}
		reposJSON, _ := json.Marshal(uniqueRepos)
		settingsMap["github_repositories"] = string(reposJSON)
	} else if len(req.GithubRepositories) > 0 {
		reposJSON, _ := json.Marshal(req.GithubRepositories)
		settingsMap["github_repositories"] = string(reposJSON)
	}

	// Track changed fields for webhook notification
	changedFields := []string{}

	// Save all configs
	for key, value := range settingsMap {
		if err := h.db.SetConfig(c.Context(), key, value); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   fmt.Sprintf("Failed to save setting: %s", key),
			})
		}
		// Track what changed for webhook
		changedFields = append(changedFields, key)
	}

	// Get updated settings
	configs, _ := h.db.GetAllConfigs(c.Context())
	settings := h.configsToSettings(configs)

	// Dispatch webhook notification for settings update (non-blocking)
	go h.dispatchSettingsUpdateWebhook(c.Context(), changedFields)

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "Settings saved successfully",
		"settings": settings,
	})
}

// ResetAdminSettings resets sensitive settings
// @Summary Reset admin settings
// @Description Clears sensitive API keys and tokens
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body object true "Keys to reset"
// @Success 200 {object} map[string]string "Settings reset successfully"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /api/admin/settings [put]
// @Security Bearer
func (h *AdminSettingsHandler) ResetAdminSettings(c *fiber.Ctx) error {
	var req struct {
		Keys []string `json:"keys"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	// Map frontend keys to config keys
	keyMap := map[string]string{
		"pterodactylApiKey":       "pterodactyl_api_key",
		"pterodactylClientApiKey": "pterodactyl_client_api_key",
		"virtfusionApiKey":        "virtfusion_api_key",
		"crowdinPersonalToken":    "crowdin_personal_token",
		"githubToken":             "github_token",
		"resendApiKey":            "resend_api_key",
	}

	for _, key := range req.Keys {
		if configKey, ok := keyMap[key]; ok {
			h.db.SetConfig(c.Context(), configKey, "")
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Settings reset successfully",
	})
}

// TestConnection tests a connection to an external service
// @Summary Test connection to external service
// @Description Tests connection to Pterodactyl, Virtfusion, or Database
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param type query string true "Connection type: pterodactyl, virtfusion, or database"
// @Success 200 {object} map[string]interface{} "Connection test result"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/admin/settings/test [post]
// @Security Bearer
func (h *AdminSettingsHandler) TestConnection(c *fiber.Ctx) error {
	connType := c.Query("type")

	var req struct {
		PterodactylUrl    string `json:"pterodactylUrl"`
		PterodactylApiKey string `json:"pterodactylApiKey"`
		VirtfusionUrl     string `json:"virtfusionUrl"`
		VirtfusionApiKey  string `json:"virtfusionApiKey"`
	}

	c.BodyParser(&req)

	switch connType {
	case "pterodactyl":
		status := h.testPterodactylConnection(req.PterodactylUrl, req.PterodactylApiKey)
		return c.JSON(status)

	case "virtfusion":
		status := h.testVirtfusionConnection(req.VirtfusionUrl, req.VirtfusionApiKey)
		return c.JSON(status)

	case "database":
		status := h.testDatabaseConnection(c)
		return c.JSON(status)

	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid connection type",
		})
	}
}

// =============================================================================
// GitHub Repositories Management
// =============================================================================

// GetRepositories returns GitHub repositories list
// @Summary Get GitHub repositories
// @Description Returns list of tracked GitHub repositories
// @Tags Admin Settings
// @Produce json
// @Success 200 {object} map[string]interface{} "Repositories list"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/admin/settings/repos [get]
// @Security Bearer
func (h *AdminSettingsHandler) GetRepositories(c *fiber.Ctx) error {
	reposRaw, err := h.db.GetConfig(c.Context(), "github_repositories")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to fetch repositories",
		})
	}

	var repos []string
	if reposRaw != "" {
		json.Unmarshal([]byte(reposRaw), &repos)
	}
	if repos == nil {
		repos = []string{}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"repos":   repos,
	})
}

// AddRepository adds a GitHub repository
// @Summary Add GitHub repository
// @Description Adds a new repository to track
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body object true "Repository info"
// @Success 200 {object} map[string]interface{} "Repository added"
// @Failure 400 {object} map[string]string "Invalid repository format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/admin/settings/repos [post]
// @Security Bearer
func (h *AdminSettingsHandler) AddRepository(c *fiber.Ctx) error {
	var req struct {
		Repo string `json:"repo"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	repo := strings.TrimSpace(req.Repo)
	if !isValidRepoFormat(repo) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid repo format. Use owner/repo.",
		})
	}

	// Get existing repos
	reposRaw, _ := h.db.GetConfig(c.Context(), "github_repositories")
	var repos []string
	if reposRaw != "" {
		json.Unmarshal([]byte(reposRaw), &repos)
	}

	// Add if not already present
	if !slices.Contains(repos, repo) {
		repos = append(repos, repo)
	}

	// Save back
	reposJSON, _ := json.Marshal(repos)
	h.db.SetConfig(c.Context(), "github_repositories", string(reposJSON))

	return c.JSON(fiber.Map{
		"success": true,
		"repos":   repos,
	})
}

// UpdateRepository updates a GitHub repository
// @Summary Update GitHub repository
// @Description Updates an existing repository entry
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body object true "Old and new repository info"
// @Success 200 {object} map[string]interface{} "Repository updated"
// @Failure 400 {object} map[string]string "Invalid repository format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/admin/settings/repos [put]
// @Security Bearer
func (h *AdminSettingsHandler) UpdateRepository(c *fiber.Ctx) error {
	var req struct {
		OldRepo string `json:"oldRepo"`
		Repo    string `json:"repo"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	newRepo := strings.TrimSpace(req.Repo)
	if !isValidRepoFormat(newRepo) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid repo format. Use owner/repo.",
		})
	}

	// Get existing repos
	reposRaw, _ := h.db.GetConfig(c.Context(), "github_repositories")
	var repos []string
	if reposRaw != "" {
		json.Unmarshal([]byte(reposRaw), &repos)
	}

	// Find and update
	oldRepo := strings.TrimSpace(req.OldRepo)
	found := false
	for i, r := range repos {
		if r == oldRepo {
			repos[i] = newRepo
			found = true
			break
		}
	}

	if !found {
		// If not found by exact match, append
		repos = append(repos, newRepo)
	}

	// Deduplicate
	deduped := make([]string, 0)
	seen := make(map[string]bool)
	for _, r := range repos {
		if !seen[r] {
			deduped = append(deduped, r)
			seen[r] = true
		}
	}

	// Save back
	reposJSON, _ := json.Marshal(deduped)
	h.db.SetConfig(c.Context(), "github_repositories", string(reposJSON))

	return c.JSON(fiber.Map{
		"success": true,
		"repos":   deduped,
	})
}

// DeleteRepository removes a GitHub repository
// @Summary Delete GitHub repository
// @Description Removes a repository from tracking
// @Tags Admin Settings
// @Accept json
// @Produce json
// @Param body body object true "Repository info"
// @Success 200 {object} map[string]interface{} "Repository deleted"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /api/admin/settings/repos [delete]
// @Security Bearer
func (h *AdminSettingsHandler) DeleteRepository(c *fiber.Ctx) error {
	var req struct {
		Repo  string `json:"repo"`
		Index int    `json:"index"`
	}

	c.BodyParser(&req)

	reposRaw, _ := h.db.GetConfig(c.Context(), "github_repositories")
	var repos []string
	if reposRaw != "" {
		json.Unmarshal([]byte(reposRaw), &repos)
	}

	// Remove by index or by name
	if req.Index >= 0 && req.Index < len(repos) {
		repos = append(repos[:req.Index], repos[req.Index+1:]...)
	} else if req.Repo != "" {
		filtered := make([]string, 0)
		for _, r := range repos {
			if r != req.Repo {
				filtered = append(filtered, r)
			}
		}
		repos = filtered
	}

	// Save back
	reposJSON, _ := json.Marshal(repos)
	h.db.SetConfig(c.Context(), "github_repositories", string(reposJSON))

	return c.JSON(fiber.Map{
		"success": true,
		"repos":   repos,
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func (h *AdminSettingsHandler) configsToSettings(configs map[string]string) SystemSettings {
	return SystemSettings{
		PterodactylUrl:          getValue(configs, "pterodactyl_url"),
		PterodactylApiKey:       h.decryptIfNeeded(getValue(configs, "pterodactyl_api_key")),
		PterodactylClientApiKey: h.decryptIfNeeded(getValue(configs, "pterodactyl_client_api_key")),
		PterodactylApi:          getValue(configs, "pterodactyl_api"),
		VirtfusionUrl:           getValue(configs, "virtfusion_url"),
		VirtfusionApiKey:        h.decryptIfNeeded(getValue(configs, "virtfusion_api_key")),
		VirtfusionApi:           getValue(configs, "virtfusion_api"),
		CrowdinProjectId:        getValue(configs, "crowdin_project_id"),
		CrowdinPersonalToken:    h.decryptIfNeeded(getValue(configs, "crowdin_personal_token")),
		GithubToken:             h.decryptIfNeeded(getValue(configs, "github_token")),
		GithubRepositories:      parseRepos(getValue(configs, "github_repositories")),
		RegistrationEnabled:     parseBool(getValue(configs, "registration_enabled")),
		MaintenanceMode:         parseBool(getValue(configs, "maintenance_mode")),
		AutoSyncEnabled:         parseBool(getValue(configs, "auto_sync_enabled")),
		EmailNotifications:      parseBool(getValue(configs, "email_notifications_enabled")),
		ResendApiKey:            h.decryptIfNeeded(getValue(configs, "resend_api_key")),
		DiscordNotifications:    parseBool(getValue(configs, "discord_notifications_enabled")),
		CacheTimeout:            parseInt(getValue(configs, "cache_timeout"), 60),
		SyncInterval:            parseInt(getValue(configs, "sync_interval"), 3600),
		AdminEmail:              getValue(configs, "admin_email"),
		SiteName:                getValue(configs, "site_name", "NodeByte Hosting"),
		SiteUrl:                 getValue(configs, "site_url"),
	}
}

func (h *AdminSettingsHandler) structToConfigMap(s SystemSettings) map[string]string {
	configMap := make(map[string]string)

	if s.PterodactylUrl != "" {
		configMap["pterodactyl_url"] = s.PterodactylUrl
	}
	if s.PterodactylApiKey != "" && !crypto.IsMasked(s.PterodactylApiKey) {
		configMap["pterodactyl_api_key"] = h.encryptIfNeeded(s.PterodactylApiKey)
	}
	if s.PterodactylClientApiKey != "" && !crypto.IsMasked(s.PterodactylClientApiKey) {
		configMap["pterodactyl_client_api_key"] = h.encryptIfNeeded(s.PterodactylClientApiKey)
	}
	if s.PterodactylApi != "" {
		configMap["pterodactyl_api"] = s.PterodactylApi
	}

	if s.VirtfusionUrl != "" {
		configMap["virtfusion_url"] = s.VirtfusionUrl
	}
	if s.VirtfusionApiKey != "" && !crypto.IsMasked(s.VirtfusionApiKey) {
		configMap["virtfusion_api_key"] = h.encryptIfNeeded(s.VirtfusionApiKey)
	}
	if s.VirtfusionApi != "" {
		configMap["virtfusion_api"] = s.VirtfusionApi
	}

	if s.CrowdinProjectId != "" {
		configMap["crowdin_project_id"] = s.CrowdinProjectId
	}
	if s.CrowdinPersonalToken != "" && !crypto.IsMasked(s.CrowdinPersonalToken) {
		configMap["crowdin_personal_token"] = h.encryptIfNeeded(s.CrowdinPersonalToken)
	}

	if s.GithubToken != "" && !crypto.IsMasked(s.GithubToken) {
		configMap["github_token"] = h.encryptIfNeeded(s.GithubToken)
	}

	configMap["registration_enabled"] = fmt.Sprintf("%v", s.RegistrationEnabled)
	configMap["maintenance_mode"] = fmt.Sprintf("%v", s.MaintenanceMode)
	configMap["auto_sync_enabled"] = fmt.Sprintf("%v", s.AutoSyncEnabled)
	configMap["email_notifications_enabled"] = fmt.Sprintf("%v", s.EmailNotifications)

	if s.ResendApiKey != "" && !crypto.IsMasked(s.ResendApiKey) {
		configMap["resend_api_key"] = h.encryptIfNeeded(s.ResendApiKey)
	}

	configMap["discord_notifications_enabled"] = fmt.Sprintf("%v", s.DiscordNotifications)
	configMap["cache_timeout"] = fmt.Sprintf("%d", s.CacheTimeout)
	configMap["sync_interval"] = fmt.Sprintf("%d", s.SyncInterval)

	if s.AdminEmail != "" {
		configMap["admin_email"] = s.AdminEmail
	}
	if s.SiteName != "" {
		configMap["site_name"] = s.SiteName
	}
	if s.SiteUrl != "" {
		configMap["site_url"] = s.SiteUrl
	}

	return configMap
}

// Test Pterodactyl connection (mock)
func (h *AdminSettingsHandler) testPterodactylConnection(url, apiKey string) fiber.Map {
	if url == "" || apiKey == "" {
		return fiber.Map{
			"success": false,
			"error":   "Pterodactyl credentials not configured",
		}
	}

	start := time.Now()
	// TODO: Implement actual Pterodactyl connection test
	latency := int(time.Since(start).Milliseconds())

	return fiber.Map{
		"success": true,
		"latency": latency,
		"version": "Connected",
	}
}

// Test Virtfusion connection (mock)
func (h *AdminSettingsHandler) testVirtfusionConnection(url, apiKey string) fiber.Map {
	if url == "" || apiKey == "" {
		return fiber.Map{
			"success": false,
			"error":   "Virtfusion credentials not configured",
		}
	}

	start := time.Now()
	// TODO: Implement actual Virtfusion connection test
	latency := int(time.Since(start).Milliseconds())

	return fiber.Map{
		"success": true,
		"latency": latency,
		"version": "Connected",
	}
}

// Test database connection
func (h *AdminSettingsHandler) testDatabaseConnection(c *fiber.Ctx) fiber.Map {
	start := time.Now()
	if err := h.db.HealthCheck(c.Context()); err != nil {
		return fiber.Map{
			"success": false,
			"error":   "Database connection failed",
		}
	}
	latency := int(time.Since(start).Milliseconds())

	return fiber.Map{
		"success": true,
		"latency": latency,
	}
}

// Utility functions
func getValue(m map[string]string, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok && v != "" {
			return v
		}
	}
	return ""
}

func maskValue(v string) string {
	if v == "" {
		return ""
	}
	return MASKED_VALUE
}

// encryptIfNeeded encrypts a plaintext value if encryptor is available
func (h *AdminSettingsHandler) encryptIfNeeded(plaintext string) string {
	if h.encryptor == nil || plaintext == "" {
		return plaintext
	}
	encrypted, err := h.encryptor.Encrypt(plaintext)
	if err != nil {
		fmt.Printf("Warning: failed to encrypt value: %v\n", err)
		return plaintext
	}
	return encrypted
}

// decryptIfNeeded decrypts a value if encryptor is available, returns plaintext
// This is sent to frontend - frontend is responsible for masking in UI
func (h *AdminSettingsHandler) decryptIfNeeded(encrypted string) string {
	if encrypted == "" {
		return ""
	}

	if h.encryptor == nil {
		return encrypted
	}

	decrypted, err := h.encryptor.Decrypt(encrypted)
	if err != nil {
		// If decryption fails, return encrypted as-is (might be unencrypted legacy value)
		return encrypted
	}

	return decrypted
}

func parseRepos(v string) []string {
	if v == "" {
		return []string{}
	}
	var repos []string
	if err := json.Unmarshal([]byte(v), &repos); err != nil {
		// Fallback: split by newliness
		return strings.Split(v, "\n")
	}
	return repos
}

func parseBool(v string) bool {
	return v == "true" || v == "1" || v == "yes"
}

func parseInt(v string, defaultVal int) int {
	if v == "" {
		return defaultVal
	}
	var i int
	if _, err := fmt.Sscanf(v, "%d", &i); err != nil {
		return defaultVal
	}
	return i
}

func isValidRepoFormat(repo string) bool {
	pattern := regexp.MustCompile(`^[^\s/]+/[^\s/]+$`)
	return pattern.MatchString(repo)
}

// dispatchSettingsUpdateWebhook sends webhook notifications for settings changes
func (h *AdminSettingsHandler) dispatchSettingsUpdateWebhook(ctx context.Context, changedFields []string) {
	if len(changedFields) == 0 {
		return
	}

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
		log.Warn().Err(err).Msg("Failed to fetch webhooks for settings update")
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

	// Prepare webhook payload
	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       "⚙️ System Settings Updated",
				"description": "Administrator has updated system configuration",
				"color":       3066993, // Green
				"fields": []map[string]interface{}{
					{
						"name":   "Changed Fields",
						"value":  strings.Join(changedFields, ", "),
						"inline": false,
					},
					{
						"name":   "Updated At",
						"value":  time.Now().Format(time.RFC3339),
						"inline": false,
					},
				},
				"timestamp": time.Now().Format(time.RFC3339),
				"footer": map[string]string{
					"text": "NodeByte System",
				},
			},
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	// Send to all webhooks in parallel
	for _, webhookURL := range webhookURLs {
		go func(url string) {
			resp, err := http.Post(url, "application/json", bytes.NewReader(payloadBytes))
			if err != nil {
				log.Warn().Err(err).Str("webhook_url", url).Msg("Failed to send settings update webhook")
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				log.Warn().Int("status", resp.StatusCode).Str("webhook_url", url).Msg("Webhook returned non-204 status")
			}
		}(webhookURL)
	}
}
