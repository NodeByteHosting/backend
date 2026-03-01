package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
)

// AdminNodeHandler handles admin node operations
type AdminNodeHandler struct {
	db *database.DB
}

// NewAdminNodeHandler creates a new admin node handler
func NewAdminNodeHandler(db *database.DB) *AdminNodeHandler {
	return &AdminNodeHandler{db: db}
}

// AdminNodeResponse represents a node for admin view
type AdminNodeResponse struct {
	ID                 int    `json:"id"`
	UUID               string `json:"uuid"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	FQDN               string `json:"fqdn"`
	Scheme             string `json:"scheme"`
	BehindProxy        bool   `json:"behindProxy"`
	IsPublic           bool   `json:"isPublic"`
	IsMaintenanceMode  bool   `json:"isMaintenanceMode"`
	Memory             int64  `json:"memory"`
	MemoryOverallocate int    `json:"memoryOverallocate"`
	Disk               int64  `json:"disk"`
	DiskOverallocate   int    `json:"diskOverallocate"`
	DaemonListenPort   int    `json:"daemonListenPort"`
	DaemonSftpPort     int    `json:"daemonSftpPort"`
	LocationID         int    `json:"locationId"`
	LocationCode       string `json:"locationCode"`
	ServerCount        int    `json:"serverCount"`
	AllocationCount    int    `json:"allocationCount"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

// GetNodes returns paginated list of all nodes
// @Summary List all nodes
// @Description Returns a paginated list of all server nodes with location info, server counts, and allocation counts
// @Tags Admin Nodes
// @Produce json
// @Security Bearer
// @Param search query string false "Search by name or FQDN"
// @Param maintenance query string false "Filter by maintenance mode" Enums(all, yes, no) default(all)
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(25)
// @Success 200 {object} object "Nodes list with pagination"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal server error"
// @Router /api/admin/nodes [get]
func (h *AdminNodeHandler) GetNodes(c *fiber.Ctx) error {
	search := c.Query("search", "")
	maintenance := c.Query("maintenance", "all") // all, yes, no
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 25)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	args := []interface{}{}
	where := `WHERE 1=1`

	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(` AND (n.name ILIKE $%d OR n.fqdn ILIKE $%d)`, len(args), len(args))
	}
	switch maintenance {
	case "yes":
		where += ` AND n."isMaintenanceMode" = true`
	case "no":
		where += ` AND n."isMaintenanceMode" = false`
	}

	var total int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM nodes n `+where, args...,
	).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count nodes"})
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	lp := fmt.Sprintf("$%d", len(args)-1)
	op := fmt.Sprintf("$%d", len(args))

	query := `
		SELECT
			n.id, n.uuid, n.name, COALESCE(n.description,''), n.fqdn,
			n.scheme, n."behindProxy", n."isPublic", n."isMaintenanceMode",
			n.memory, n."memoryOverallocate", n.disk, n."diskOverallocate",
			n."daemonListenPort", n."daemonSftpPort",
			n."locationId", COALESCE(l."shortCode",''),
			(SELECT COUNT(*) FROM servers s WHERE s."nodeId" = n.id) AS server_count,
			(SELECT COUNT(*) FROM allocations a WHERE a."nodeId" = n.id) AS alloc_count,
			n."createdAt", n."updatedAt"
		FROM nodes n
		LEFT JOIN locations l ON l.id = n."locationId"
		` + where + `
		ORDER BY n.name ASC
		LIMIT ` + lp + ` OFFSET ` + op

	rows, err := h.db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch nodes: " + err.Error()})
	}
	defer rows.Close()

	nodes := []AdminNodeResponse{}
	for rows.Next() {
		var nd AdminNodeResponse
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&nd.ID, &nd.UUID, &nd.Name, &nd.Description, &nd.FQDN,
			&nd.Scheme, &nd.BehindProxy, &nd.IsPublic, &nd.IsMaintenanceMode,
			&nd.Memory, &nd.MemoryOverallocate, &nd.Disk, &nd.DiskOverallocate,
			&nd.DaemonListenPort, &nd.DaemonSftpPort,
			&nd.LocationID, &nd.LocationCode,
			&nd.ServerCount, &nd.AllocationCount,
			&createdAt, &updatedAt,
		); err != nil {
			log.Warn().Err(err).Msg("Failed to scan node row")
			continue
		}
		nd.CreatedAt = createdAt.Format(time.RFC3339)
		nd.UpdatedAt = updatedAt.Format(time.RFC3339)
		nodes = append(nodes, nd)
	}

	totalPages := (total + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true,
		"nodes":   nodes,
		"pagination": fiber.Map{
			"page": page, "pageSize": pageSize,
			"total": total, "totalPages": totalPages,
		},
	})
}

// GetNodeAllocations returns allocations for a specific node
// @Summary Get node allocations
// @Description Returns paginated allocations for a specific node with optional assigned filter
// @Tags Admin Nodes
// @Produce json
// @Security Bearer
// @Param id path string true "Node ID"
// @Param assigned query string false "Filter by assignment status" Enums(all, yes, no) default(all)
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(50)
// @Success 200 {object} object "Allocations list with pagination"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal server error"
// @Router /api/admin/nodes/{id}/allocations [get]
func (h *AdminNodeHandler) GetNodeAllocations(c *fiber.Ctx) error {
	nodeID := c.Params("id")
	assigned := c.Query("assigned", "all") // all, yes, no
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 50)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	where := `WHERE a."nodeId" = $1`
	args := []interface{}{nodeID}

	switch assigned {
	case "yes":
		where += ` AND a."isAssigned" = true`
	case "no":
		where += ` AND a."isAssigned" = false`
	}

	var total int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM allocations a `+where, args...,
	).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count allocations"})
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	lp := fmt.Sprintf("$%d", len(args)-1)
	op := fmt.Sprintf("$%d", len(args))

	query := `
		SELECT
			a.id, a.ip, a.port, COALESCE(a.alias,''), a."isAssigned",
			s."pterodactylId", s.name
		FROM allocations a
		LEFT JOIN servers s ON s.id = a."serverId"
		` + where + `
		ORDER BY a.ip ASC, a.port ASC
		LIMIT ` + lp + ` OFFSET ` + op

	rows, err := h.db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch allocations: " + err.Error()})
	}
	defer rows.Close()

	type AllocRow struct {
		ID            int     `json:"id"`
		IP            string  `json:"ip"`
		Port          int     `json:"port"`
		Alias         string  `json:"alias"`
		IsAssigned    bool    `json:"isAssigned"`
		ServerPteroID *int    `json:"serverPterodactylId"`
		ServerName    *string `json:"serverName"`
	}

	allocs := []AllocRow{}
	for rows.Next() {
		var a AllocRow
		if err := rows.Scan(&a.ID, &a.IP, &a.Port, &a.Alias, &a.IsAssigned, &a.ServerPteroID, &a.ServerName); err != nil {
			log.Warn().Err(err).Msg("Failed to scan allocation row")
			continue
		}
		allocs = append(allocs, a)
	}

	totalPages := (total + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success":     true,
		"allocations": allocs,
		"pagination": fiber.Map{
			"page": page, "pageSize": pageSize,
			"total": total, "totalPages": totalPages,
		},
	})
}

// ToggleNodeMaintenance toggles maintenance mode on a node
// @Summary Toggle node maintenance mode
// @Description Toggles the maintenance mode flag on a specific node
// @Tags Admin Nodes
// @Produce json
// @Security Bearer
// @Param id path string true "Node ID"
// @Success 200 {object} object "Maintenance mode toggled"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal server error"
// @Router /api/admin/nodes/{id}/maintenance [post]
func (h *AdminNodeHandler) ToggleNodeMaintenance(c *fiber.Ctx) error {
	nodeID := c.Params("id")

	var enabled bool
	if err := h.db.Pool.QueryRow(context.Background(),
		`UPDATE nodes SET "isMaintenanceMode" = NOT "isMaintenanceMode", "updatedAt" = NOW()
		 WHERE id = $1 RETURNING "isMaintenanceMode"`,
		nodeID,
	).Scan(&enabled); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to toggle maintenance mode"})
	}

	status := "disabled"
	if enabled {
		status = "enabled"
	}
	return c.JSON(fiber.Map{
		"success":         true,
		"maintenanceMode": enabled,
		"message":         "Maintenance mode " + status,
	})
}

// GetLocations returns all locations (simple list, no pagination needed)
// @Summary List all locations
// @Description Returns all Pterodactyl panel locations with their node counts
// @Tags Admin Nodes
// @Produce json
// @Security Bearer
// @Success 200 {object} object "Locations list"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal server error"
// @Router /api/admin/locations [get]
func (h *AdminNodeHandler) GetLocations(c *fiber.Ctx) error {
	rows, err := h.db.Pool.Query(context.Background(),
		`SELECT l.id, l."shortCode", COALESCE(l.description,''),
		        (SELECT COUNT(*) FROM nodes n WHERE n."locationId" = l.id) AS node_count
		 FROM locations l
		 ORDER BY l."shortCode" ASC`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch locations"})
	}
	defer rows.Close()

	type LocRow struct {
		ID        int    `json:"id"`
		ShortCode string `json:"shortCode"`
		Desc      string `json:"description"`
		NodeCount int    `json:"nodeCount"`
	}

	locs := []LocRow{}
	for rows.Next() {
		var l LocRow
		if err := rows.Scan(&l.ID, &l.ShortCode, &l.Desc, &l.NodeCount); err != nil {
			continue
		}
		locs = append(locs, l)
	}

	return c.JSON(fiber.Map{"success": true, "locations": locs})
}

// GetAllAllocations returns all allocations across all nodes with filtering
// @Summary List all allocations
// @Description Returns paginated allocations across all nodes with filtering by assignment status, node, and IP/alias search
// @Tags Admin Nodes
// @Produce json
// @Security Bearer
// @Param assigned query string false "Filter by assignment status" Enums(all, yes, no) default(all)
// @Param nodeId query string false "Filter by node ID"
// @Param search query string false "Search by IP, alias, or port"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(50)
// @Success 200 {object} object "Allocations list with pagination"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal server error"
// @Router /api/admin/allocations [get]
func (h *AdminNodeHandler) GetAllAllocations(c *fiber.Ctx) error {
	assigned := c.Query("assigned", "all") // all, yes, no
	nodeID := c.Query("nodeId", "")
	search := c.Query("search", "") // IP or alias search
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 50)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	where := `WHERE 1=1`
	args := []interface{}{}

	if nodeID != "" {
		args = append(args, nodeID)
		where += fmt.Sprintf(` AND a."nodeId" = $%d`, len(args))
	}

	switch assigned {
	case "yes":
		where += ` AND a."isAssigned" = true`
	case "no":
		where += ` AND a."isAssigned" = false`
	}

	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(` AND (a.ip ILIKE $%d OR COALESCE(a.alias,'') ILIKE $%d OR a.port::text ILIKE $%d)`, len(args), len(args), len(args))
	}

	var total int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM allocations a `+where, args...,
	).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count allocations"})
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	lp := fmt.Sprintf("$%d", len(args)-1)
	op := fmt.Sprintf("$%d", len(args))

	query := `
		SELECT
			a.id, a.ip, a.port, COALESCE(a.alias,''), a."isAssigned",
			a."nodeId", COALESCE(n.name,''), COALESCE(n.fqdn,''),
			s.id, s."pterodactylId", s.name
		FROM allocations a
		LEFT JOIN nodes n ON n.id = a."nodeId"
		LEFT JOIN servers s ON s.id = a."serverId"
		` + where + `
		ORDER BY n.name ASC, a.ip ASC, a.port ASC
		LIMIT ` + lp + ` OFFSET ` + op

	rows, err := h.db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch allocations: " + err.Error()})
	}
	defer rows.Close()

	type AllocRow struct {
		ID            int     `json:"id"`
		IP            string  `json:"ip"`
		Port          int     `json:"port"`
		Alias         string  `json:"alias"`
		IsAssigned    bool    `json:"isAssigned"`
		NodeID        int     `json:"nodeId"`
		NodeName      string  `json:"nodeName"`
		NodeFQDN      string  `json:"nodeFqdn"`
		ServerID      *string `json:"serverId"`
		ServerPteroID *int    `json:"serverPterodactylId"`
		ServerName    *string `json:"serverName"`
	}

	allocs := []AllocRow{}
	for rows.Next() {
		var a AllocRow
		if err := rows.Scan(
			&a.ID, &a.IP, &a.Port, &a.Alias, &a.IsAssigned,
			&a.NodeID, &a.NodeName, &a.NodeFQDN,
			&a.ServerID, &a.ServerPteroID, &a.ServerName,
		); err != nil {
			log.Warn().Err(err).Msg("Failed to scan allocation row")
			continue
		}
		allocs = append(allocs, a)
	}

	totalPages := (total + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success":     true,
		"allocations": allocs,
		"pagination": fiber.Map{
			"page": page, "pageSize": pageSize,
			"total": total, "totalPages": totalPages,
		},
	})
}
