package types

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
