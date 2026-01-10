package database

import (
	"database/sql"
	"time"
)

// User represents a user in the system
type User struct {
	ID                 string
	Email              string
	Password           sql.NullString
	Username           sql.NullString
	FirstName          sql.NullString
	LastName           sql.NullString
	Roles              []string
	IsPterodactylAdmin bool
	IsVirtfusionAdmin  bool
	IsSystemAdmin      bool
	PterodactylID      sql.NullInt64
	VirtfusionID       sql.NullInt64
	IsMigrated         bool
	EmailVerified      sql.NullTime
	IsActive           bool
	AvatarURL          sql.NullString
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastLoginAt        sql.NullTime
	LastSyncedAt       sql.NullTime
}

// Location represents a data center location
type Location struct {
	ID          int
	ShortCode   string
	Description sql.NullString
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Node represents a game server host node
type Node struct {
	ID                 int
	UUID               string
	Name               string
	Description        sql.NullString
	FQDN               string
	Scheme             string
	BehindProxy        bool
	PanelType          string
	Memory             int64
	MemoryOverallocate int
	Disk               int64
	DiskOverallocate   int
	IsPublic           bool
	IsMaintenanceMode  bool
	DaemonListenPort   int
	DaemonSFTPPort     int
	DaemonBase         string
	LocationID         int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Allocation represents an IP:Port allocation on a node
type Allocation struct {
	ID         int
	IP         string
	Port       int
	Alias      sql.NullString
	Notes      sql.NullString
	IsAssigned bool
	NodeID     int
	ServerID   sql.NullString
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Nest represents a category of eggs (e.g., "Minecraft")
type Nest struct {
	ID          int
	UUID        string
	Name        string
	Description sql.NullString
	Author      sql.NullString
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Egg represents a server type template
type Egg struct {
	ID          int
	UUID        string
	Name        string
	Description sql.NullString
	Author      sql.NullString
	PanelType   string
	NestID      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EggVariable represents configuration options for an egg
type EggVariable struct {
	ID           int
	EggID        int
	Name         string
	Description  sql.NullString
	EnvVariable  string
	DefaultValue sql.NullString
	UserViewable bool
	UserEditable bool
	Rules        sql.NullString
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Server represents a game server instance
type Server struct {
	ID            string
	PterodactylID sql.NullInt64
	VirtfusionID  sql.NullInt64
	UUID          string
	UUIDShort     sql.NullString
	ExternalID    sql.NullString
	PanelType     string
	Name          string
	Description   sql.NullString
	Status        string
	IsSuspended   bool
	OwnerID       sql.NullString
	NodeID        sql.NullInt64
	EggID         sql.NullInt64
	Memory        int64
	Disk          int64
	CPU           int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ServerDatabase represents a database attached to a server
type ServerDatabase struct {
	ID             int
	PterodactylID  int
	ServerID       string
	DatabaseName   string
	Username       string
	HostID         sql.NullInt64
	MaxConnections int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SyncLog represents a sync operation log
type SyncLog struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	ItemsTotal  int       `json:"itemsTotal"`
	ItemsSynced int       `json:"itemsSynced"`
	ItemsFailed int       `json:"itemsFailed"`
	Error       *string   `json:"error"`
	Metadata    string    `json:"metadata"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

// Config represents a system configuration key-value pair
type Config struct {
	ID        string
	Key       string
	Value     string
	Type      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DiscordWebhook represents a Discord webhook configuration
type DiscordWebhook struct {
	ID                 string
	Name               string
	URL                string
	Enabled            bool
	NotificationTypes  []string
	NotificationScopes []string
	CreatorID          sql.NullString
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SetupStatus represents system setup state
type SetupStatus struct {
	ID        string
	Step      string
	Completed bool
	Metadata  sql.NullString // JSON
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PteroUser represents a Pterodactyl user from API
type PteroUser struct {
	Attributes struct {
		ID         int    `json:"id"`
		Email      string `json:"email"`
		Username   string `json:"username"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		RootAdmin  bool   `json:"root_admin"`
	} `json:"attributes"`
}
