package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/nodebyte/backend/internal/config"
	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/middleware"
	"github.com/nodebyte/backend/internal/queue"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, db *database.DB, queueManager *queue.Manager, apiKeyMiddleware *APIKeyMiddleware, cfg *config.Config) {
	// Health check route (public - no authentication required)
	app.Get("/health", healthCheck(db, queueManager))

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

	// Hytale OAuth routes (public - no authentication required)
	// Apply rate limiting to OAuth endpoints
	hytaleOAuthHandler := NewHytaleOAuthHandler(db, cfg.HytaleUseStaging)

	deviceCodeLimiter := middleware.NewRateLimiter(middleware.DeviceCodeRateLimit)
	tokenPollLimiter := middleware.NewRateLimiter(middleware.TokenPollRateLimit)
	tokenRefreshLimiter := middleware.NewRateLimiter(middleware.TokenRefreshRateLimit)
	gameSessionLimiter := middleware.NewRateLimiter(middleware.GameSessionRateLimit)

	app.Post("/api/v1/hytale/oauth/device-code", deviceCodeLimiter.Middleware(), hytaleOAuthHandler.RequestDeviceCode)
	app.Post("/api/v1/hytale/oauth/token", tokenPollLimiter.Middleware(), hytaleOAuthHandler.PollToken)
	app.Post("/api/v1/hytale/oauth/refresh", tokenRefreshLimiter.Middleware(), hytaleOAuthHandler.RefreshAccessToken)
	app.Post("/api/v1/hytale/oauth/profiles", gameSessionLimiter.Middleware(), hytaleOAuthHandler.GetProfiles)
	app.Post("/api/v1/hytale/oauth/select-profile", gameSessionLimiter.Middleware(), hytaleOAuthHandler.SelectProfile)
	app.Post("/api/v1/hytale/oauth/game-session/new", gameSessionLimiter.Middleware(), hytaleOAuthHandler.CreateGameSession)
	app.Post("/api/v1/hytale/oauth/game-session/refresh", gameSessionLimiter.Middleware(), hytaleOAuthHandler.RefreshGameSession)
	app.Post("/api/v1/hytale/oauth/game-session/delete", gameSessionLimiter.Middleware(), hytaleOAuthHandler.TerminateGameSession)

	hytaleLogsHandler := NewHytaleLogsHandler(db)
	app.Get("/api/v1/hytale/logs", hytaleLogsHandler.GetHytaleLogs)

	hytaleServerLogsHandler := NewHytaleServerLogsHandler(db)
	app.Get("/api/v1/hytale/server-logs", hytaleServerLogsHandler.GetHytaleServerLogs)
	app.Post("/api/v1/hytale/server-logs", hytaleServerLogsHandler.CreateServerLogs)
	app.Get("/api/v1/hytale/server-logs/count", hytaleServerLogsHandler.GetHytaleServerLogsCount)

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

	// Swagger documentation routes (public)
	app.Get("/docs/swagger.json", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		return c.SendFile("./docs/swagger.json")
	})

	// Swagger UI
	app.Get("/swagger", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>NodeByte API Documentation (Swagger UI)</title>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<link href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui.css" rel="stylesheet">
			<link rel="icon" type="image/png" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/favicon-32x32.png" sizes="32x32">
			<link rel="icon" type="image/png" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/favicon-16x16.png" sizes="16x16">
			<style>
				html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
				*, *:before, *:after { box-sizing: inherit; }
				body { margin: 0; background: #fafafa; }
				.swagger-ui .topbar { background-color: #1f1f1f; }
				.swagger-ui .info .title { color: #3b82f6; }
			</style>
		</head>
		<body>
			<div id="swagger-ui"></div>
			<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui-bundle.js"></script>
			<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui-standalone-preset.js"></script>
			<script>
				const ui = SwaggerUIBundle({
					url: "/docs/swagger.json",
					dom_id: '#swagger-ui',
					deepLinking: true,
					presets: [
						SwaggerUIBundle.presets.apis,
						SwaggerUIStandalonePreset
					],
					plugins: [
						SwaggerUIBundle.plugins.DownloadUrl
					],
					layout: "StandaloneLayout",
					defaultModelsExpandDepth: 1,
					defaultModelExpandDepth: 1,
					onComplete: function() {
						console.log("Swagger UI loaded successfully");
					}
				});
				window.ui = ui;
			</script>
		</body>
		</html>
		`
		return c.SendString(html)
	})

	// ReDoc UI
	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>NodeByte API Documentation (ReDoc)</title>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
			<style>
				body { margin: 0; padding: 0; font-family: Roboto, sans-serif; }
				redoc { display: block; }
			</style>
		</head>
		<body>
			<redoc 
				spec-url="/docs/swagger.json"
				expand-single-schema="true"
				sort-props-alphabetically="true"
				show-extensions="true"
				native-scrollbars="true"
				path-in-middle-panel="true">
			</redoc>
			<script src="https://cdn.jsdelivr.net/npm/redoc/bundles/redoc.standalone.js"></script>
		</body>
		</html>
		`
		return c.SendString(html)
	})
}

// healthCheck returns a health check handler with worker monitoring
func healthCheck(db *database.DB, queueManager *queue.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check database connection
		dbErr := db.HealthCheck(c.Context())
		dbHealthy := dbErr == nil

		// Check queue/worker status
		queueHealthy := true
		var queueStats fiber.Map
		if queueManager != nil {
			// Try to get queue stats - this indicates if workers are running
			queueStats = fiber.Map{
				"status": "running",
			}
		} else {
			queueHealthy = false
			queueStats = fiber.Map{
				"status": "unavailable",
			}
		}

		// If any critical service is down, return unhealthy
		if !dbHealthy || !queueHealthy {
			statusCode := fiber.StatusServiceUnavailable
			checks := fiber.Map{
				"database": "error",
				"workers":  "error",
			}
			if dbHealthy {
				checks["database"] = "ok"
			}
			if queueHealthy {
				checks["workers"] = "ok"
			}

			errorMsg := ""
			if !dbHealthy {
				errorMsg = dbErr.Error()
			}

			return c.Status(statusCode).JSON(fiber.Map{
				"status":    "unhealthy",
				"database":  map[string]interface{}{"status": "disconnected", "error": errorMsg},
				"workers":   queueStats,
				"service":   "nodebyte-backend",
				"version":   "1.0.0",
				"timestamp": c.Get("Date"),
				"checks":    checks,
			})
		}

		// All services healthy
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"database":  fiber.Map{"status": "connected"},
			"workers":   queueStats,
			"service":   "nodebyte-backend",
			"version":   "1.0.0",
			"timestamp": c.Get("Date"),
			"checks": fiber.Map{
				"database": "ok",
				"workers":  "ok",
				"api":      "ok",
			},
		})
	}
}
