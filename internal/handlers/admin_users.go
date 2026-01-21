package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	"github.com/nodebyte/backend/internal/database"
)

// AdminUserHandler handles admin user operations
type AdminUserHandler struct {
	db *database.DB
}

// NewAdminUserHandler creates a new admin user handler
func NewAdminUserHandler(db *database.DB) *AdminUserHandler {
	return &AdminUserHandler{db: db}
}

// AdminUserResponse represents a user for admin view
type AdminUserResponse struct {
	ID                 string   `json:"id"`
	Email              string   `json:"email"`
	Username           string   `json:"username"`
	FirstName          string   `json:"firstName"`
	LastName           string   `json:"lastName"`
	Roles              []string `json:"roles"`
	IsPterodactylAdmin bool     `json:"isPterodactylAdmin"`
	IsVirtfusionAdmin  bool     `json:"isVirtfusionAdmin"`
	IsSystemAdmin      bool     `json:"isSystemAdmin"`
	IsMigrated         bool     `json:"isMigrated"`
	IsActive           bool     `json:"isActive"`
	EmailVerified      bool     `json:"emailVerified"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
	LastLoginAt        string   `json:"lastLoginAt,omitempty"`
	ServerCount        int      `json:"serverCount"`
	SessionCount       int      `json:"sessionCount"`
}

// GetUsersRequest represents pagination and filter parameters
type GetUsersRequest struct {
	Search   string `query:"search"`
	Filter   string `query:"filter"` // all, migrated, active, admin, inactive
	Sort     string `query:"sort"`   // email, created
	Order    string `query:"order"`  // asc, desc
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
}

// GetUsers returns paginated list of all users with filtering
func (h *AdminUserHandler) GetUsers(c *fiber.Ctx) error {
	// Parse query parameters
	req := GetUsersRequest{
		Search:   c.Query("search", ""),
		Filter:   c.Query("filter", "all"),
		Sort:     c.Query("sort", "created"),
		Order:    c.Query("order", "desc"),
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("pageSize", 25),
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 25
	}

	// Build base query with WHERE clause first
	baseQuery := `WHERE 1=1`

	// Apply search filter
	if req.Search != "" {
		baseQuery += ` AND (u.email ILIKE $1 OR u.username ILIKE $1)`
	}

	// Apply status filter
	switch req.Filter {
	case "migrated":
		baseQuery += ` AND u."isMigrated" = true`
	case "active":
		baseQuery += ` AND u."isActive" = true`
	case "admin":
		baseQuery += ` AND (u."isSystemAdmin" = true OR u."isPterodactylAdmin" = true OR u."isVirtfusionAdmin" = true)`
	case "inactive":
		baseQuery += ` AND u."isActive" = false`
		// default: "all" - no additional filter
	}

	// Build main query using subqueries for counts
	query := `
		SELECT 
			u.id, u.email, u.username,
			u.roles, u."isPterodactylAdmin", u."isVirtfusionAdmin", 
			u."isSystemAdmin", u."isMigrated", u."isActive", u."emailVerified",
			u."createdAt", u."updatedAt", u."lastLoginAt",
			(SELECT COUNT(*) FROM servers WHERE "ownerId" = u.id) as server_count,
			(SELECT COUNT(*) FROM sessions WHERE "userId" = u.id) as session_count
		FROM users u
		` + baseQuery

	// Apply sorting
	sortField := "u.\"createdAt\""
	if req.Sort == "email" {
		sortField = "u.email"
	}
	sortOrder := "DESC"
	if strings.ToLower(req.Order) == "asc" {
		sortOrder = "ASC"
	}
	query += fmt.Sprintf(` ORDER BY %s %s`, sortField, sortOrder)

	// Build count query
	countQuery := `SELECT COUNT(*) FROM users u ` + baseQuery

	// Get total count for pagination
	var totalCount int
	args := []interface{}{}
	if req.Search != "" {
		args = append(args, "%"+req.Search+"%")
	}

	err := h.db.Pool.QueryRow(context.Background(), countQuery, args...).Scan(&totalCount)
	if err != nil {
		fmt.Println("DEBUG: Count query error:", err.Error())
		fmt.Println("DEBUG: Count query:", countQuery)
		fmt.Println("DEBUG: Args:", args)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count users: " + err.Error(),
		})
	}

	// Apply pagination
	offset := (req.Page - 1) * req.PageSize
	query += fmt.Sprintf(` LIMIT %d OFFSET %d`, req.PageSize, offset)

	// Execute query
	rows, err := h.db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		fmt.Println("DEBUG: Query error:", err.Error())
		fmt.Println("DEBUG: Query:", query)
		fmt.Println("DEBUG: Args:", args)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users: " + err.Error(),
		})
	}
	defer rows.Close()

	users := []AdminUserResponse{}
	for rows.Next() {
		var user AdminUserResponse
		var rolesArray pq.StringArray
		var lastLoginAt *time.Time
		var emailVerifiedTime *time.Time
		var createdAt time.Time
		var updatedAt time.Time

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&rolesArray,
			&user.IsPterodactylAdmin,
			&user.IsVirtfusionAdmin,
			&user.IsSystemAdmin,
			&user.IsMigrated,
			&user.IsActive,
			&emailVerifiedTime,
			&createdAt,
			&updatedAt,
			&lastLoginAt,
			&user.ServerCount,
			&user.SessionCount,
		)
		if err != nil {
			fmt.Printf("DEBUG: Scan error: %v\n", err)
			continue
		}

		// Parse roles array
		user.Roles = []string(rolesArray)
		if user.Roles == nil {
			user.Roles = []string{}
		}

		// Convert timestamps to ISO 8601 string format
		user.CreatedAt = createdAt.Format(time.RFC3339)
		user.UpdatedAt = updatedAt.Format(time.RFC3339)

		// Handle nullable lastLoginAt
		if lastLoginAt != nil {
			user.LastLoginAt = lastLoginAt.Format(time.RFC3339)
		}

		// Handle nullable emailVerified (TIMESTAMP) - if not null, user has verified email
		if emailVerifiedTime != nil {
			user.EmailVerified = true
		}

		users = append(users, user)
	}

	// Calculate pagination info
	totalPages := (totalCount + req.PageSize - 1) / req.PageSize

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"users": users,
			"pagination": fiber.Map{
				"page":       req.Page,
				"pageSize":   req.PageSize,
				"total":      totalCount,
				"totalPages": totalPages,
			},
		},
	})
}

// UpdateUserRolesRequest represents a request to update user roles
type UpdateUserRolesRequest struct {
	UserID string   `json:"userId"`
	Roles  []string `json:"roles"`
}

// UpdateUserRoles updates the roles for a user
func (h *AdminUserHandler) UpdateUserRoles(c *fiber.Ctx) error {
	var req UpdateUserRolesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "userId is required",
		})
	}

	// Validate roles
	validRoles := map[string]bool{
		"admin":             true,
		"moderator":         true,
		"supporter":         true,
		"pterodactyl_admin": true,
		"virtfusion_admin":  true,
		"system_admin":      true,
	}

	for _, role := range req.Roles {
		if !validRoles[role] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid role: %s", role),
			})
		}
	}

	// Update user roles in database
	roleStr := strings.Join(req.Roles, ",")
	err := h.db.Pool.QueryRow(context.Background(),
		`UPDATE users SET roles = $1, updated_at = NOW() WHERE id = $2 RETURNING id, roles`,
		roleStr, req.UserID,
	).Scan(nil, nil)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user roles",
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"userId": req.UserID,
			"roles":  req.Roles,
		},
	})
}
