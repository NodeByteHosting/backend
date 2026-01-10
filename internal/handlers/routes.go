package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/queue"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, db *database.DB, queueManager *queue.Manager, apiKeyMiddleware *APIKeyMiddleware) {
	// Public routes (no authentication required)
	statsHandler := NewStatsHandler(db)
	app.Get("/api/stats", statsHandler.GetPublicStats)
	app.Get("/api/panel/counts", statsHandler.GetPanelCounts)

	// Auth routes (public - no authentication required)
	authHandler := NewAuthHandler(db, queueManager)
	app.Post("/api/v1/auth/login", authHandler.AuthenticateUser)
	app.Post("/api/v1/auth/register", authHandler.RegisterUser)
	app.Post("/api/v1/auth/validate", authHandler.ValidateCredentials)
	app.Post("/api/v1/auth/verify-email", authHandler.VerifyEmail)
	app.Post("/api/v1/auth/forgot-password", authHandler.ForgotPassword)
	app.Post("/api/v1/auth/reset-password", authHandler.ResetPassword)
	app.Post("/api/v1/auth/magic-link", authHandler.RequestMagicLink)
	app.Post("/api/v1/auth/magic-link/verify", authHandler.VerifyMagicLink)
	app.Get("/api/v1/auth/check-email", authHandler.CheckEmailExists)
	app.Get("/api/v1/auth/users/:id", authHandler.GetUserByID)

	// Admin settings routes (require bearer token auth) - MUST BE BEFORE /api group
	bearerAuth := NewBearerAuthMiddleware(db)
	adminGroup := app.Group("/api/admin", bearerAuth.Handler())

	// Settings routes
	settingsHandler := NewAdminSettingsHandler(db)
	adminGroup.Get("/settings", settingsHandler.GetAdminSettings)
	adminGroup.Post("/settings", settingsHandler.SaveAdminSettings)
	adminGroup.Put("/settings", settingsHandler.ResetAdminSettings)
	adminGroup.Post("/settings/test", settingsHandler.TestConnection)

	// GitHub repositories routes
	adminGroup.Get("/settings/repos", settingsHandler.GetRepositories)
	adminGroup.Post("/settings/repos", settingsHandler.AddRepository)
	adminGroup.Put("/settings/repos", settingsHandler.UpdateRepository)
	adminGroup.Delete("/settings/repos", settingsHandler.DeleteRepository)

	// Webhooks routes
	webhooksHandler := NewAdminWebhooksHandler(db)
	adminGroup.Get("/settings/webhooks", webhooksHandler.GetWebhooks)
	adminGroup.Post("/settings/webhooks", webhooksHandler.CreateWebhook)
	adminGroup.Put("/settings/webhooks", webhooksHandler.UpdateWebhook)
	adminGroup.Patch("/settings/webhooks", webhooksHandler.TestWebhook)
	adminGroup.Delete("/settings/webhooks", webhooksHandler.DeleteWebhook)

	// Admin sync routes
	adminSyncHandler := NewAdminSyncHandler(db, queueManager)
	adminGroup.Get("/sync", adminSyncHandler.GetSyncStatusAdmin)
	adminGroup.Post("/sync", adminSyncHandler.TriggerSyncAdmin)
	adminGroup.Post("/sync/cancel", adminSyncHandler.CancelSyncAdmin)
	adminGroup.Get("/sync/logs", adminSyncHandler.GetSyncLogs)
	adminGroup.Get("/sync/settings", adminSyncHandler.GetSyncSettingsAdmin)
	adminGroup.Post("/sync/settings", adminSyncHandler.UpdateSyncSettingsAdmin)

	// Protected routes (require API key or bearer token) - AFTER admin routes
	protected := app.Group("/api", apiKeyMiddleware.Handler())

	// Sync routes
	syncHandler := NewSyncAPIHandler(db, queueManager)
	protected.Post("/v1/sync/full", syncHandler.TriggerFullSync)
	protected.Post("/v1/sync/locations", syncHandler.TriggerLocationsSync)
	protected.Post("/v1/sync/nodes", syncHandler.TriggerNodesSync)
	protected.Post("/v1/sync/servers", syncHandler.TriggerServersSync)
	protected.Post("/v1/sync/users", syncHandler.TriggerUsersSync)
	protected.Post("/v1/sync/cancel/:id", syncHandler.CancelSync)
	protected.Get("/v1/sync/status/:id", syncHandler.GetSyncStatus)
	protected.Get("/v1/sync/logs", syncHandler.GetSyncLogs)
	protected.Get("/v1/sync/latest", syncHandler.GetLatestSync)

	// Stats routes
	protected.Get("/v1/stats/overview", statsHandler.GetOverview)
	protected.Get("/v1/stats/servers", statsHandler.GetServerStats)
	protected.Get("/v1/stats/users", statsHandler.GetUserStats)
	protected.Get("/v1/stats/admin", statsHandler.GetAdminStats)

	// Email routes
	emailHandler := NewEmailAPIHandler(queueManager)
	protected.Post("/v1/email/queue", emailHandler.QueueEmail)

	// Webhook routes
	webhookHandler := NewWebhookAPIHandler(db, queueManager)
	protected.Post("/v1/webhook/dispatch", webhookHandler.DispatchWebhook)

	// Queue routes
	queueHandler := NewQueueHandler()
	protected.Get("/v1/queues/stats", queueHandler.GetStats)
}

// healthCheck returns a health check handler
func healthCheck(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check database connection
		if err := db.HealthCheck(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":   "unhealthy",
				"database": "disconnected",
				"error":    err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status":   "healthy",
			"database": "connected",
			"service":  "nodebyte-backend",
			"version":  "1.0.0",
		})
	}
}
