package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nodebyte/backend/internal/database"
	"github.com/rs/zerolog/log"
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
	ID            string     `json:"id"`
	ServerType    string     `json:"serverType"`
	PterodactylID int        `json:"pterodactylId"`
	UUID          string     `json:"uuid"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Status        string     `json:"status"`
	IsSuspended   bool       `json:"isSuspended"`
	PanelType     string     `json:"panelType"`
	Owner         *OwnerInfo `json:"owner"`
	Node          *NodeInfo  `json:"node"`
	Egg           *EggInfo   `json:"egg"`
	Memory        int        `json:"memory"`
	Disk          int        `json:"disk"`
	CPU           int        `json:"cpu"`
	CreatedAt     string     `json:"createdAt"`
	UpdatedAt     string     `json:"updatedAt"`
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
	Search     string `query:"search"`
	Status     string `query:"status"`     // all, online, offline, suspended, installing
	ServerType string `query:"serverType"` // all, game_server, vps, email, web_hosting
	Sort       string `query:"sort"`       // name, created, status
	Order      string `query:"order"`      // asc, desc
	Page       int    `query:"page"`
	PageSize   int    `query:"pageSize"`
}

// GetServers returns paginated list of all servers with filtering
func (h *AdminServerHandler) GetServers(c *fiber.Ctx) error {
	// Parse query parameters
	req := GetServersRequest{
		Search:     c.Query("search", ""),
		Status:     c.Query("status", "all"),
		ServerType: c.Query("serverType", "all"),
		Sort:       c.Query("sort", "created"),
		Order:      c.Query("order", "desc"),
		Page:       c.QueryInt("page", 1),
		PageSize:   c.QueryInt("pageSize", 25),
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 25
	}

	whereClause := `WHERE 1=1`
	args := []interface{}{}

	// Apply search filter
	if req.Search != "" {
		args = append(args, "%"+req.Search+"%")
		whereClause += fmt.Sprintf(` AND (s.name ILIKE $%d OR s.description ILIKE $%d)`, len(args), len(args))
	}

	// Apply status filter — match actual DB enum values
	switch req.Status {
	case "online", "running":
		whereClause += ` AND s.status = 'online' AND s."isSuspended" = false`
	case "offline":
		whereClause += ` AND s.status = 'offline' AND s."isSuspended" = false`
	case "suspended":
		whereClause += ` AND s."isSuspended" = true`
	case "installing":
		whereClause += ` AND s.status = 'installing'`
	}

	// Apply server type filter
	if req.ServerType != "" && req.ServerType != "all" {
		args = append(args, req.ServerType)
		whereClause += fmt.Sprintf(` AND s."serverType" = $%d`, len(args))
	}

	// Apply sorting
	sortField := `s."createdAt"`
	if req.Sort == "name" {
		sortField = "s.name"
	} else if req.Sort == "status" {
		sortField = "s.status"
	}
	sortOrder := "DESC"
	if strings.ToLower(req.Order) == "asc" {
		sortOrder = "ASC"
	}

	// Count
	var totalCount int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM servers s `+whereClause,
		args...,
	).Scan(&totalCount); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to count servers: " + err.Error(),
		})
	}

	// Pagination args
	offset := (req.Page - 1) * req.PageSize
	args = append(args, req.PageSize, offset)
	limitPlaceholder := fmt.Sprintf("$%d", len(args)-1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args))

	query := `
		SELECT
			s.id, COALESCE(s."serverType", 'game_server'), s."pterodactylId", COALESCE(s.uuid, ''), s.name,
			COALESCE(s.description, ''), s.status, s."isSuspended",
			s.memory, s.disk, s.cpu, COALESCE(s."panelType", 'pterodactyl'),
			s."createdAt", s."updatedAt",
			u.id, u.email, u.username,
			n.id, n.name, n.fqdn,
			e.id, e.name, nest.name
		FROM servers s
		LEFT JOIN users u ON s."ownerId" = u.id
		LEFT JOIN nodes n ON s."nodeId" = n.id
		LEFT JOIN eggs e ON s."eggId" = e.id
		LEFT JOIN nests nest ON e."nestId" = nest.id
		` + whereClause + `
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder

	rows, err := h.db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch servers: " + err.Error(),
		})
	}
	defer rows.Close()

	servers := []AdminServerResponse{}
	for rows.Next() {
		var server AdminServerResponse
		var pterodactylId *int
		var uuid, ownerID, ownerEmail, ownerUsername *string
		var nodeID *int
		var nodeName, nodeFQDN *string
		var eggID *int
		var eggName, nestName *string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&server.ID, &server.ServerType, &pterodactylId, &uuid, &server.Name,
			&server.Description, &server.Status, &server.IsSuspended,
			&server.Memory, &server.Disk, &server.CPU, &server.PanelType,
			&createdAt, &updatedAt,
			&ownerID, &ownerEmail, &ownerUsername,
			&nodeID, &nodeName, &nodeFQDN,
			&eggID, &eggName, &nestName,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to scan server row")
			continue
		}

		if pterodactylId != nil {
			server.PterodactylID = *pterodactylId
		}
		if uuid != nil {
			server.UUID = *uuid
		}
		server.CreatedAt = createdAt.Format(time.RFC3339)
		server.UpdatedAt = updatedAt.Format(time.RFC3339)

		if ownerID != nil {
			server.Owner = &OwnerInfo{
				ID:       *ownerID,
				Email:    *ownerEmail,
				Username: *ownerUsername,
			}
		}
		if nodeID != nil {
			server.Node = &NodeInfo{
				ID:   *nodeID,
				Name: *nodeName,
				FQDN: *nodeFQDN,
			}
		}
		if eggID != nil {
			nest := ""
			if nestName != nil {
				nest = *nestName
			}
			server.Egg = &EggInfo{
				ID:   *eggID,
				Name: *eggName,
				Nest: nest,
			}
		}

		servers = append(servers, server)
	}

	totalPages := (totalCount + req.PageSize - 1) / req.PageSize
	return c.JSON(fiber.Map{
		"success": true,
		"servers": servers,
		"pagination": fiber.Map{
			"page":       req.Page,
			"pageSize":   req.PageSize,
			"total":      totalCount,
			"totalPages": totalPages,
		},
	})
}
