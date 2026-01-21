package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nodebyte/backend/internal/database"
)

// AdminServerHandler handles admin server operations
type AdminServerHandler struct {
	db *database.DB
}

// NewAdminServerHandler creates a new admin server handler
func NewAdminServerHandler(db *database.DB) *AdminServerHandler {
	return &AdminServerHandler{db: db}
}

// AdminServerResponse represents a server for admin view
type AdminServerResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	IsSuspended bool       `json:"isSuspended"`
	Owner       *OwnerInfo `json:"owner"`
	Node        *NodeInfo  `json:"node"`
	Egg         *EggInfo   `json:"egg"`
	Memory      int64      `json:"memory"`
	Disk        int64      `json:"disk"`
	CPU         int        `json:"cpu"`
	CreatedAt   string     `json:"createdAt"`
	UpdatedAt   string     `json:"updatedAt"`
}

// OwnerInfo represents server owner information
type OwnerInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// NodeInfo represents node information
type NodeInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	FQDN string `json:"fqdn"`
}

// EggInfo represents egg information
type EggInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Nest string `json:"nest"`
}

// GetServersRequest represents pagination and filter parameters
type GetServersRequest struct {
	Search   string `query:"search"`
	Status   string `query:"status"` // all, online, offline, suspended, installing
	Sort     string `query:"sort"`   // name, created, status
	Order    string `query:"order"`  // asc, desc
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
}

// GetServers returns paginated list of all servers with filtering
func (h *AdminServerHandler) GetServers(c *fiber.Ctx) error {
	// Parse query parameters
	req := GetServersRequest{
		Search:   c.Query("search", ""),
		Status:   c.Query("status", "all"),
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

	// Build query
	query := `
		SELECT 
			s.id, s.name, s.description, s.status, s.is_suspended,
			u.id, u.email, u.username,
			n.id, n.name, n.fqdn,
			e.id, e.name, nest.name,
			s.memory, s.disk, s.cpu, s.created_at, s.updated_at
		FROM servers s
		LEFT JOIN users u ON s.owner_id = u.id
		LEFT JOIN nodes n ON s.node_id = n.id
		LEFT JOIN eggs e ON s.egg_id = e.id
		LEFT JOIN nests nest ON e.nest_id = nest.id
		WHERE 1=1
	`

	args := []interface{}{}

	// Apply search filter
	if req.Search != "" {
		args = append(args, "%"+req.Search+"%")
		query += fmt.Sprintf(` AND (s.name ILIKE $%d OR s.description ILIKE $%d)`, len(args), len(args))
	}

	// Apply status filter
	switch req.Status {
	case "online":
		args = append(args, "running")
		query += fmt.Sprintf(` AND s.status = $%d AND s.is_suspended = false`, len(args))
	case "offline":
		args = append(args, "stopped")
		query += fmt.Sprintf(` AND s.status = $%d AND s.is_suspended = false`, len(args))
	case "suspended":
		query += ` AND s.is_suspended = true`
	case "installing":
		args = append(args, "installing")
		query += fmt.Sprintf(` AND s.status = $%d`, len(args))
		// default: "all" - no additional filter
	}

	// Apply sorting
	sortField := "s.created_at"
	if req.Sort == "name" {
		sortField = "s.name"
	} else if req.Sort == "status" {
		sortField = "s.status"
	}
	sortOrder := "DESC"
	if strings.ToLower(req.Order) == "asc" {
		sortOrder = "ASC"
	}
	query += fmt.Sprintf(` ORDER BY %s %s`, sortField, sortOrder)

	// Get total count for pagination
	countQuery := `
		SELECT COUNT(*)
		FROM servers s
		WHERE 1=1
	`

	// Apply same filters to count query
	if req.Search != "" {
		countQuery += ` AND (s.name ILIKE $1 OR s.description ILIKE $1)`
	}

	switch req.Status {
	case "online":
		countQuery += ` AND s.status = 'running' AND s.is_suspended = false`
	case "offline":
		countQuery += ` AND s.status = 'stopped' AND s.is_suspended = false`
	case "suspended":
		countQuery += ` AND s.is_suspended = true`
	case "installing":
		countQuery += ` AND s.status = 'installing'`
	}

	var totalCount int
	countArgs := []interface{}{}
	if req.Search != "" {
		countArgs = append(countArgs, "%"+req.Search+"%")
	}

	err := h.db.Pool.QueryRow(context.Background(), countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count servers",
		})
	}

	// Apply pagination
	offset := (req.Page - 1) * req.PageSize
	args = append(args, req.PageSize, offset)
	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	// Execute query
	rows, err := h.db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch servers",
		})
	}
	defer rows.Close()

	servers := []AdminServerResponse{}
	for rows.Next() {
		var server AdminServerResponse
		var ownerID, ownerEmail, ownerUsername interface{}
		var nodeID interface{}
		var nodeName, nodeFQDN interface{}
		var eggID interface{}
		var eggName, nestName interface{}

		err := rows.Scan(
			&server.ID,
			&server.Name,
			&server.Description,
			&server.Status,
			&server.IsSuspended,
			&ownerID,
			&ownerEmail,
			&ownerUsername,
			&nodeID,
			&nodeName,
			&nodeFQDN,
			&eggID,
			&eggName,
			&nestName,
			&server.Memory,
			&server.Disk,
			&server.CPU,
			&server.CreatedAt,
			&server.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Map owner info
		if ownerID != nil {
			server.Owner = &OwnerInfo{
				ID:       ownerID.(string),
				Email:    ownerEmail.(string),
				Username: ownerUsername.(string),
			}
		}

		// Map node info
		if nodeID != nil {
			server.Node = &NodeInfo{
				ID:   nodeID.(int),
				Name: nodeName.(string),
				FQDN: nodeFQDN.(string),
			}
		}

		// Map egg info
		if eggID != nil {
			server.Egg = &EggInfo{
				ID:   eggID.(int),
				Name: eggName.(string),
				Nest: nestName.(string),
			}
		}

		servers = append(servers, server)
	}

	// Calculate pagination info
	totalPages := (totalCount + req.PageSize - 1) / req.PageSize

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"servers": servers,
			"pagination": fiber.Map{
				"page":       req.Page,
				"pageSize":   req.PageSize,
				"total":      totalCount,
				"totalPages": totalPages,
			},
		},
	})
}
