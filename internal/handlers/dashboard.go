package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nodebyte/backend/internal/database"
	"github.com/nodebyte/backend/internal/queue"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// DashboardHandler handles dashboard API requests
type DashboardHandler struct {
	db           *database.DB
	queueManager *queue.Manager
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(db *database.DB, queueManager *queue.Manager) *DashboardHandler {
	return &DashboardHandler{db: db, queueManager: queueManager}
}

// GetDashboardStats retrieves user-specific dashboard statistics
// @Summary Get dashboard stats
// @Description Retrieves statistics for the user's dashboard including server counts and recent servers
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse "Dashboard stats retrieved"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/dashboard/stats [get]
func (h *DashboardHandler) GetDashboardStats(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get user ID from auth context
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}

	// Get server counts for this user
	var totalServers, onlineServers, offlineServers, suspendedServers int
	h.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM servers WHERE "ownerId" = $1`, userID).Scan(&totalServers)
	h.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM servers WHERE "ownerId" = $1 AND status = 'RUNNING'`, userID).Scan(&onlineServers)
	h.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM servers WHERE "ownerId" = $1 AND status = 'OFFLINE'`, userID).Scan(&offlineServers)
	h.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM servers WHERE "ownerId" = $1 AND "isSuspended" = true`, userID).Scan(&suspendedServers)

	// Get recent servers
	rows, err := h.db.Pool.Query(ctx, `
		SELECT 
			s.id, s.uuid, s.name, s.status,
			n.name as node_name,
			e.name as egg_name,
			COALESCE((SELECT value FROM server_properties WHERE "serverId" = s.id AND key = 'memory'), '0') as memory_limit,
			COALESCE((SELECT value FROM server_properties WHERE "serverId" = s.id AND key = 'cpu'), '100') as cpu_limit,
			COALESCE((SELECT value FROM server_properties WHERE "serverId" = s.id AND key = 'disk'), '0') as disk_limit,
			COALESCE((SELECT ip FROM allocations WHERE "serverId" = s.id AND "isAssigned" = true LIMIT 1), '0.0.0.0') as ip,
			COALESCE((SELECT port FROM allocations WHERE "serverId" = s.id AND "isAssigned" = true LIMIT 1), 0) as port
		FROM servers s
		LEFT JOIN nodes n ON s."nodeId" = n.id
		LEFT JOIN eggs e ON s."eggId" = e.id
		WHERE s."ownerId" = $1
		ORDER BY s."updatedAt" DESC
		LIMIT 6
	`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch recent servers",
		})
	}
	defer rows.Close()

	type RecentServer struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		Game      string `json:"game"`
		Node      string `json:"node"`
		Resources struct {
			Memory struct {
				Used  int `json:"used"`
				Limit int `json:"limit"`
			} `json:"memory"`
			CPU struct {
				Used  int `json:"used"`
				Limit int `json:"limit"`
			} `json:"cpu"`
			Disk struct {
				Used  int `json:"used"`
				Limit int `json:"limit"`
			} `json:"disk"`
		} `json:"resources"`
	}

	recentServers := []RecentServer{}
	for rows.Next() {
		var server RecentServer
		var memoryLimit, cpuLimit, diskLimit, ip string
		var port int
		err := rows.Scan(
			&server.ID, &server.ID, &server.Name, &server.Status,
			&server.Node, &server.Game,
			&memoryLimit, &cpuLimit, &diskLimit, &ip, &port,
		)
		if err != nil {
			continue
		}

		// Parse resource limits
		var memLimit, cpuLim, diskLim int
		fmt.Sscanf(memoryLimit, "%d", &memLimit)
		fmt.Sscanf(cpuLimit, "%d", &cpuLim)
		fmt.Sscanf(diskLimit, "%d", &diskLim)

		server.Resources.Memory.Limit = memLimit
		server.Resources.CPU.Limit = cpuLim
		server.Resources.Disk.Limit = diskLim
		server.Resources.Memory.Used = 0 // Would come from real-time API
		server.Resources.CPU.Used = 0
		server.Resources.Disk.Used = 0

		recentServers = append(recentServers, server)
	}

	// Get user account balance
	var accountBalance float64
	h.db.Pool.QueryRow(ctx,
		`SELECT COALESCE("accountBalance", 0) FROM users WHERE id = $1`, userID).Scan(&accountBalance)

	// Get open tickets count
	var openTickets int
	h.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM support_tickets 
		WHERE "userId" = $1 AND status IN ('open', 'pending', 'in_progress')
	`, userID).Scan(&openTickets)

	return c.JSON(SuccessResponse{
		Success: true,
		Data: fiber.Map{
			"servers": fiber.Map{
				"total":     totalServers,
				"online":    onlineServers,
				"offline":   offlineServers,
				"suspended": suspendedServers,
			},
			"recentServers":  recentServers,
			"accountBalance": accountBalance,
			"openTickets":    openTickets,
		},
	})
}

// GetUserServers retrieves paginated server list for the authenticated user
// @Summary Get user servers
// @Description Retrieves paginated list of servers owned by the authenticated user with search and filtering
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(12)
// @Param search query string false "Search query"
// @Param status query string false "Status filter"
// @Success 200 {object} SuccessResponse "Servers retrieved"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/dashboard/servers [get]
func (h *DashboardHandler) GetUserServers(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get user ID from auth context
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}

	// Parse query parameters
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	perPage := c.QueryInt("per_page", 12)
	if perPage < 1 || perPage > 50 {
		perPage = 12
	}
	search := c.Query("search", "")
	statusFilter := c.Query("status", "")
	viewAll := c.QueryBool("view_all", false)
	isAdmin, _ := c.Locals("isAdmin").(bool)

	// Build WHERE clause
	var whereClause string
	var args []interface{}
	argIndex := 1

	if viewAll && isAdmin {
		// Admin viewing all servers — no owner filter
		whereClause = "TRUE"
	} else {
		whereClause = `"ownerId" = $1`
		args = append(args, userID)
		argIndex = 2
	}

	if search != "" {
		whereClause += ` AND (s.name ILIKE $` + fmt.Sprintf("%d", argIndex) + ` OR s.description ILIKE $` + fmt.Sprintf("%d", argIndex) + `)`
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if statusFilter != "" && statusFilter != "all" {
		statusMap := map[string]string{
			"running":    "online",
			"online":     "online",
			"offline":    "offline",
			"starting":   "starting",
			"stopping":   "stopping",
			"suspended":  "online", // handled via isSuspended
			"installing": "installing",
		}
		if statusFilter == "suspended" {
			whereClause += ` AND s."isSuspended" = true`
		} else if mappedStatus, ok := statusMap[statusFilter]; ok {
			whereClause += ` AND s.status = $` + fmt.Sprintf("%d", argIndex)
			args = append(args, mappedStatus)
			argIndex++
		}
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM servers s WHERE ` + whereClause
	h.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)

	// Calculate pagination
	offset := (page - 1) * perPage
	totalPages := (total + perPage - 1) / perPage

	// Get servers — always LEFT JOIN users so owner info is available
	query := `
		SELECT 
			s.id, s.uuid, s.name, s.description, s.status, s."isSuspended",
			n.name as node_name,
			e.name as egg_name,
			s.memory, s.disk, s.cpu,
			COALESCE((SELECT ip FROM allocations WHERE "serverId" = s.id AND "isAssigned" = true LIMIT 1), '0.0.0.0') as ip,
			COALESCE((SELECT port FROM allocations WHERE "serverId" = s.id AND "isAssigned" = true LIMIT 1), 0) as port,
			s."createdAt",
			u.id as owner_id, u.username as owner_username, u.email as owner_email
		FROM servers s
		LEFT JOIN nodes n ON s."nodeId" = n.id
		LEFT JOIN eggs e ON s."eggId" = e.id
		LEFT JOIN users u ON s."ownerId" = u.id
		WHERE ` + whereClause + `
		ORDER BY s."updatedAt" DESC
		LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)

	args = append(args, perPage, offset)
	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch servers",
		})
	}
	defer rows.Close()

	type ServerOwner struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	type Server struct {
		ID          string       `json:"id"`
		UUID        string       `json:"uuid"`
		Name        string       `json:"name"`
		Description string       `json:"description"`
		Status      string       `json:"status"`
		IsSuspended bool         `json:"isSuspended"`
		Game        string       `json:"game"`
		Node        string       `json:"node"`
		IP          string       `json:"ip"`
		Port        int          `json:"port"`
		Owner       *ServerOwner `json:"owner,omitempty"`
		Resources   struct {
			Memory struct {
				Used  int `json:"used"`
				Limit int `json:"limit"`
			} `json:"memory"`
			CPU struct {
				Used  int `json:"used"`
				Limit int `json:"limit"`
			} `json:"cpu"`
			Disk struct {
				Used  int `json:"used"`
				Limit int `json:"limit"`
			} `json:"disk"`
		} `json:"resources"`
		CreatedAt string `json:"createdAt"`
	}

	servers := []Server{}
	for rows.Next() {
		var server Server
		var description *string
		var memory, disk, cpu int
		var ownerID, ownerUsername, ownerEmail *string
		err := rows.Scan(
			&server.ID, &server.UUID, &server.Name, &description, &server.Status, &server.IsSuspended,
			&server.Node, &server.Game,
			&memory, &disk, &cpu,
			&server.IP, &server.Port, &server.CreatedAt,
			&ownerID, &ownerUsername, &ownerEmail,
		)
		if err != nil {
			continue
		}

		if description != nil {
			server.Description = *description
		}
		if ownerID != nil {
			owner := &ServerOwner{ID: *ownerID}
			if ownerUsername != nil {
				owner.Username = *ownerUsername
			}
			if ownerEmail != nil {
				owner.Email = *ownerEmail
			}
			server.Owner = owner
		}

		// Resource limits come directly from the servers table columns
		server.Resources.Memory.Limit = memory
		server.Resources.CPU.Limit = cpu
		server.Resources.Disk.Limit = disk
		server.Resources.Memory.Used = 0 // Would come from real-time metrics API
		server.Resources.CPU.Used = 0
		server.Resources.Disk.Used = 0

		servers = append(servers, server)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    servers,
		"meta": fiber.Map{
			"total":      total,
			"page":       page,
			"perPage":    perPage,
			"totalPages": totalPages,
		},
	})
}

// GetUserAccount retrieves the authenticated user's account information
// @Summary Get user account
// @Description Retrieves account information for the authenticated user
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse "Account info retrieved"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/dashboard/account [get]
func (h *DashboardHandler) GetUserAccount(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get user ID from auth context
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}

	// Fetch user account data
	var user struct {
		ID             string   `json:"id"`
		Username       *string  `json:"username"`
		Email          string   `json:"email"`
		FirstName      *string  `json:"firstName"`
		LastName       *string  `json:"lastName"`
		PhoneNumber    *string  `json:"phoneNumber"`
		CompanyName    *string  `json:"companyName"`
		BillingEmail   *string  `json:"billingEmail"`
		AvatarURL      *string  `json:"avatarUrl"`
		AccountBalance float64  `json:"accountBalance"`
		CreatedAt      string   `json:"createdAt"`
		EmailVerified  bool     `json:"emailVerified"`
		LastLoginAt    *string  `json:"lastLoginAt"`
		Roles          []string `json:"roles"`
	}

	err := h.db.Pool.QueryRow(ctx, `
		SELECT id, username, email, "firstName", "lastName",
		       "phoneNumber", "companyName", "billingEmail",
		       "avatarUrl", COALESCE("accountBalance", 0), "createdAt"::TEXT,
		       "emailVerified" IS NOT NULL, "lastLoginAt"::TEXT, COALESCE(roles, '{}')
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName,
		&user.PhoneNumber, &user.CompanyName, &user.BillingEmail,
		&user.AvatarURL, &user.AccountBalance, &user.CreatedAt,
		&user.EmailVerified, &user.LastLoginAt, &user.Roles,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to fetch account",
		})
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data:    user,
	})
}

// UpdateUserAccountRequest represents account update request
type UpdateUserAccountRequest struct {
	Username     *string `json:"username"`
	Email        *string `json:"email"`
	FirstName    *string `json:"firstName"`
	LastName     *string `json:"lastName"`
	PhoneNumber  *string `json:"phoneNumber"`
	CompanyName  *string `json:"companyName"`
	BillingEmail *string `json:"billingEmail"`
}

// UpdateUserAccount updates the authenticated user's account information
// @Summary Update user account
// @Description Updates account information for the authenticated user
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body UpdateUserAccountRequest true "Account update data"
// @Success 200 {object} SuccessResponse "Account updated"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/dashboard/account [put]
func (h *DashboardHandler) UpdateUserAccount(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get user ID from auth context
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}

	var req UpdateUserAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Username != nil && *req.Username != "" {
		updates = append(updates, fmt.Sprintf(`username = $%d`, argIndex))
		args = append(args, *req.Username)
		argIndex++
	}
	if req.Email != nil && *req.Email != "" {
		updates = append(updates, fmt.Sprintf(`email = $%d`, argIndex))
		args = append(args, *req.Email)
		argIndex++
		// Reset email verification since email changed
		updates = append(updates, `"emailVerified" = NULL`)
	}
	if req.FirstName != nil {
		updates = append(updates, fmt.Sprintf(`"firstName" = $%d`, argIndex))
		args = append(args, *req.FirstName)
		argIndex++
	}
	if req.LastName != nil {
		updates = append(updates, fmt.Sprintf(`"lastName" = $%d`, argIndex))
		args = append(args, *req.LastName)
		argIndex++
	}
	if req.PhoneNumber != nil {
		updates = append(updates, fmt.Sprintf(`"phoneNumber" = $%d`, argIndex))
		args = append(args, *req.PhoneNumber)
		argIndex++
	}
	if req.CompanyName != nil {
		updates = append(updates, fmt.Sprintf(`"companyName" = $%d`, argIndex))
		args = append(args, *req.CompanyName)
		argIndex++
	}
	if req.BillingEmail != nil {
		updates = append(updates, fmt.Sprintf(`"billingEmail" = $%d`, argIndex))
		args = append(args, *req.BillingEmail)
		argIndex++
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "No fields to update",
		})
	}

	// Add updated timestamp
	updates = append(updates, `"updatedAt" = NOW()`)

	// Add user ID for WHERE clause
	args = append(args, userID)

	query := `UPDATE users SET ` + strings.Join(updates, ", ") + ` WHERE id = $` + fmt.Sprintf("%d", argIndex)
	_, err := h.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to update account",
		})
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Message: "Account updated successfully",
	})
}

// ChangePasswordRequest represents password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePassword changes the authenticated user's password
// @Summary Change user password
// @Description Changes password for the authenticated user
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body ChangePasswordRequest true "Password change data"
// @Success 200 {object} SuccessResponse "Password changed"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized or wrong password"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/dashboard/account/password [put]
func (h *DashboardHandler) ChangePassword(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get user ID from auth context
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Success: false,
			Error:   "Current and new passwords are required",
		})
	}

	// Get current user with password
	user, err := h.db.QueryUserByID(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to verify user",
		})
	}

	// Verify current password
	if !user.VerifyPassword(req.CurrentPassword) {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Success: false,
			Error:   "Current password is incorrect",
		})
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to hash new password",
		})
	}

	// Update password
	_, err = h.db.Pool.Exec(ctx, `
		UPDATE users 
		SET password = $1, "updatedAt" = NOW()
		WHERE id = $2
	`, newHash, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Success: false,
			Error:   "Failed to update password",
		})
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Message: "Password changed successfully",
	})
}

// ResendVerificationEmail resends the email verification email for the authenticated user
// @Summary Resend verification email
// @Description Resends the email verification email for the authenticated user. Fails if email is already verified.
// @Tags Dashboard
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse "Verification email sent"
// @Failure 400 {object} ErrorResponse "Email already verified"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/dashboard/account/resend-verification [post]
func (h *DashboardHandler) ResendVerificationEmail(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Success: false, Error: "User not authenticated"})
	}

	// Fetch user
	var email string
	var firstName *string
	var alreadyVerified bool
	err := h.db.Pool.QueryRow(ctx,
		`SELECT email, "firstName", "emailVerified" IS NOT NULL FROM users WHERE id = $1`, userID,
	).Scan(&email, &firstName, &alreadyVerified)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Success: false, Error: "Failed to fetch user"})
	}

	if alreadyVerified {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Success: false, Error: "Email already verified"})
	}

	// Generate a fresh verification token
	token, err := h.db.StoreVerificationToken(ctx, userID, database.VerificationTokenType, database.TokenExpiration)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("Failed to generate verification token for resend")
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Success: false, Error: "Failed to generate verification token"})
	}

	// Queue the verification email
	if h.queueManager != nil {
		name := ""
		if firstName != nil {
			name = *firstName
		}
		_, _ = h.queueManager.EnqueueEmail(queue.EmailPayload{
			To:       email,
			Subject:  "Verify your email",
			Template: "verify-email",
			Data: map[string]string{
				"name":  name,
				"token": token,
				"email": email,
			},
		})
	}

	return c.JSON(SuccessResponse{Success: true, Message: "Verification email sent"})
}

// RequestEmailChange allows an authenticated user to request an email change.
// The new email is set immediately and emailVerified is cleared; a verification
// email is sent to the new address.
// @Summary Request email change
// @Description Changes the user's email address. Requires current password for verification. Clears email verification status and sends a verification email to the new address.
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body object true "Email change request" SchemaExample({"newEmail": "new@example.com", "currentPassword": "password123"})
// @Success 200 {object} SuccessResponse "Email updated"
// @Failure 400 {object} ErrorResponse "Missing required fields"
// @Failure 401 {object} ErrorResponse "Unauthorized or wrong password"
// @Failure 409 {object} ErrorResponse "Email already in use"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/dashboard/account/change-email [post]
func (h *DashboardHandler) RequestEmailChange(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Success: false, Error: "User not authenticated"})
	}

	var req struct {
		NewEmail        string `json:"newEmail"`
		CurrentPassword string `json:"currentPassword"`
	}
	if err := c.BodyParser(&req); err != nil || req.NewEmail == "" || req.CurrentPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Success: false, Error: "newEmail and currentPassword are required"})
	}

	// Verify current password
	user, err := h.db.QueryUserByID(ctx, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Success: false, Error: "Failed to fetch user"})
	}
	if !user.VerifyPassword(req.CurrentPassword) {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Success: false, Error: "Current password is incorrect"})
	}

	// Check new email not already in use
	var exists bool
	_ = h.db.Pool.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND id != $2)`, req.NewEmail, userID,
	).Scan(&exists)
	if exists {
		return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Success: false, Error: "Email already in use"})
	}

	// Update email and clear verification
	_, err = h.db.Pool.Exec(ctx,
		`UPDATE users SET email = $1, "emailVerified" = NULL, "updatedAt" = NOW() WHERE id = $2`,
		req.NewEmail, userID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Success: false, Error: "Failed to update email"})
	}

	// Send verification email to new address
	token, err := h.db.StoreVerificationToken(ctx, userID, database.VerificationTokenType, database.TokenExpiration)
	if err == nil && h.queueManager != nil {
		name := ""
		if user.FirstName.Valid {
			name = user.FirstName.String
		}
		_, _ = h.queueManager.EnqueueEmail(queue.EmailPayload{
			To:       req.NewEmail,
			Subject:  "Verify your new email address",
			Template: "verify-email",
			Data: map[string]string{
				"name":  name,
				"token": token,
				"email": req.NewEmail,
			},
		})
	}

	return c.JSON(SuccessResponse{Success: true, Message: "Email updated. Please check your new email to verify."})
}
