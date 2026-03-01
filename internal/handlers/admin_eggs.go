package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/nodebyte/backend/internal/database"
)

// AdminEggHandler handles admin egg/nest operations
type AdminEggHandler struct {
	db *database.DB
}

// NewAdminEggHandler creates a new admin egg handler
func NewAdminEggHandler(db *database.DB) *AdminEggHandler {
	return &AdminEggHandler{db: db}
}

// AdminNestResponse represents a nest for admin view
type AdminNestResponse struct {
	ID          int    `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	EggCount    int    `json:"eggCount"`
	ServerCount int    `json:"serverCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// AdminEggResponse represents an egg for admin view
type AdminEggResponse struct {
	ID          int    `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	NestID      int    `json:"nestId"`
	NestName    string `json:"nestName"`
	ServerCount int    `json:"serverCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// GetNests returns all nests with egg counts and server counts
func (h *AdminEggHandler) GetNests(c *fiber.Ctx) error {
	search := c.Query("search", "")
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
		where += fmt.Sprintf(` AND n.name ILIKE $%d`, len(args))
	}

	var total int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM nests n `+where, args...,
	).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count nests"})
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	lp := fmt.Sprintf("$%d", len(args)-1)
	op := fmt.Sprintf("$%d", len(args))

	query := `
		SELECT
			n.id, n.uuid, n.name, COALESCE(n.description,''), COALESCE(n.author,''),
			(SELECT COUNT(*) FROM eggs e WHERE e."nestId" = n.id) AS egg_count,
			(SELECT COUNT(*) FROM servers s WHERE s."eggId" IN (SELECT id FROM eggs WHERE "nestId" = n.id)) AS server_count,
			n."createdAt", n."updatedAt"
		FROM nests n
		` + where + `
		ORDER BY n.name ASC
		LIMIT ` + lp + ` OFFSET ` + op

	rows, err := h.db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch nests: " + err.Error()})
	}
	defer rows.Close()

	nests := []AdminNestResponse{}
	for rows.Next() {
		var n AdminNestResponse
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&n.ID, &n.UUID, &n.Name, &n.Description, &n.Author,
			&n.EggCount, &n.ServerCount,
			&createdAt, &updatedAt,
		); err != nil {
			log.Warn().Err(err).Msg("Failed to scan nest row")
			continue
		}
		n.CreatedAt = createdAt.Format(time.RFC3339)
		n.UpdatedAt = updatedAt.Format(time.RFC3339)
		nests = append(nests, n)
	}

	totalPages := (total + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true,
		"nests":   nests,
		"pagination": fiber.Map{
			"page": page, "pageSize": pageSize,
			"total": total, "totalPages": totalPages,
		},
	})
}

// GetEggs returns paginated list of eggs with nest name and server count
func (h *AdminEggHandler) GetEggs(c *fiber.Ctx) error {
	search := c.Query("search", "")
	nestID := c.Query("nestId", "")
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
		where += fmt.Sprintf(` AND (e.name ILIKE $%d OR e.description ILIKE $%d)`, len(args), len(args))
	}
	if nestID != "" {
		args = append(args, nestID)
		where += fmt.Sprintf(` AND e."nestId" = $%d`, len(args))
	}

	var total int
	if err := h.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM eggs e `+where, args...,
	).Scan(&total); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count eggs"})
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	lp := fmt.Sprintf("$%d", len(args)-1)
	op := fmt.Sprintf("$%d", len(args))

	query := `
		SELECT
			e.id, e.uuid, e.name, COALESCE(e.description,''), COALESCE(e.author,''),
			e."nestId", COALESCE(n.name,''),
			(SELECT COUNT(*) FROM servers s WHERE s."eggId" = e.id) AS server_count,
			e."createdAt", e."updatedAt"
		FROM eggs e
		LEFT JOIN nests n ON n.id = e."nestId"
		` + where + `
		ORDER BY n.name ASC, e.name ASC
		LIMIT ` + lp + ` OFFSET ` + op

	rows, err := h.db.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch eggs: " + err.Error()})
	}
	defer rows.Close()

	eggs := []AdminEggResponse{}
	for rows.Next() {
		var eg AdminEggResponse
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&eg.ID, &eg.UUID, &eg.Name, &eg.Description, &eg.Author,
			&eg.NestID, &eg.NestName,
			&eg.ServerCount,
			&createdAt, &updatedAt,
		); err != nil {
			log.Warn().Err(err).Msg("Failed to scan egg row")
			continue
		}
		eg.CreatedAt = createdAt.Format(time.RFC3339)
		eg.UpdatedAt = updatedAt.Format(time.RFC3339)
		eggs = append(eggs, eg)
	}

	totalPages := (total + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true,
		"eggs":    eggs,
		"pagination": fiber.Map{
			"page": page, "pageSize": pageSize,
			"total": total, "totalPages": totalPages,
		},
	})
}
