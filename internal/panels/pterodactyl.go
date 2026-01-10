package panels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PterodactylClient handles communication with the Pterodactyl panel API
type PterodactylClient struct {
	baseURL          string
	apiKey           string
	clientAPIKey     string
	cfAccessClientID string
	cfAccessSecret   string
	httpClient       *http.Client
}

// NewPterodactylClient creates a new Pterodactyl API client
func NewPterodactylClient(baseURL, apiKey, cfClientID, cfSecret string) *PterodactylClient {
	return &PterodactylClient{
		baseURL:          baseURL,
		apiKey:           apiKey,
		clientAPIKey:     "",
		cfAccessClientID: cfClientID,
		cfAccessSecret:   cfSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewPterodactylClientWithClientKey creates a new Pterodactyl API client with both application and client API keys
func NewPterodactylClientWithClientKey(baseURL, apiKey, clientAPIKey, cfClientID, cfSecret string) *PterodactylClient {
	return &PterodactylClient{
		baseURL:          baseURL,
		apiKey:           apiKey,
		clientAPIKey:     clientAPIKey,
		cfAccessClientID: cfClientID,
		cfAccessSecret:   cfSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Object string          `json:"object"`
	Data   json.RawMessage `json:"data"`
	Meta   struct {
		Pagination struct {
			Total       int `json:"total"`
			Count       int `json:"count"`
			PerPage     int `json:"per_page"`
			CurrentPage int `json:"current_page"`
			TotalPages  int `json:"total_pages"`
		} `json:"pagination"`
	} `json:"meta"`
}

// PteroLocation represents a Pterodactyl location
type PteroLocation struct {
	Object     string `json:"object"`
	Attributes struct {
		ID        int    `json:"id"`
		ShortCode string `json:"short"`
		Long      string `json:"long"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
}

// PteroNode represents a Pterodactyl node
type PteroNode struct {
	Object     string `json:"object"`
	Attributes struct {
		ID                 int    `json:"id"`
		UUID               string `json:"uuid"`
		Public             bool   `json:"public"`
		Name               string `json:"name"`
		Description        string `json:"description"`
		LocationID         int    `json:"location_id"`
		FQDN               string `json:"fqdn"`
		Scheme             string `json:"scheme"`
		BehindProxy        bool   `json:"behind_proxy"`
		MaintenanceMode    bool   `json:"maintenance_mode"`
		Memory             int64  `json:"memory"`
		MemoryOverallocate int    `json:"memory_overallocate"`
		Disk               int64  `json:"disk"`
		DiskOverallocate   int    `json:"disk_overallocate"`
		UploadSize         int    `json:"upload_size"`
		DaemonListen       int    `json:"daemon_listen"`
		DaemonSFTP         int    `json:"daemon_sftp"`
		DaemonBase         string `json:"daemon_base"`
		CreatedAt          string `json:"created_at"`
		UpdatedAt          string `json:"updated_at"`
	} `json:"attributes"`
}

// PteroAllocation represents a Pterodactyl allocation
type PteroAllocation struct {
	Object     string `json:"object"`
	Attributes struct {
		ID       int    `json:"id"`
		IP       string `json:"ip"`
		Alias    string `json:"alias"`
		Port     int    `json:"port"`
		Notes    string `json:"notes"`
		Assigned bool   `json:"assigned"`
	} `json:"attributes"`
}

// PteroNest represents a Pterodactyl nest
type PteroNest struct {
	Object     string `json:"object"`
	Attributes struct {
		ID          int    `json:"id"`
		UUID        string `json:"uuid"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Author      string `json:"author"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
	} `json:"attributes"`
}

// PteroEgg represents a Pterodactyl egg
type PteroEgg struct {
	Object     string `json:"object"`
	Attributes struct {
		ID          int    `json:"id"`
		UUID        string `json:"uuid"`
		Name        string `json:"name"`
		Nest        int    `json:"nest"`
		Author      string `json:"author"`
		Description string `json:"description"`
		DockerImage string `json:"docker_image"`
		Startup     string `json:"startup"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
	} `json:"attributes"`
	Relationships struct {
		Variables struct {
			Object string             `json:"object"`
			Data   []PteroEggVariable `json:"data"`
		} `json:"variables"`
	} `json:"relationships"`
}

// PteroEggVariable represents an egg variable
type PteroEggVariable struct {
	Object     string `json:"object"`
	Attributes struct {
		ID           int    `json:"id"`
		EggID        int    `json:"egg_id"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		EnvVariable  string `json:"env_variable"`
		DefaultValue string `json:"default_value"`
		UserViewable bool   `json:"user_viewable"`
		UserEditable bool   `json:"user_editable"`
		Rules        string `json:"rules"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
	} `json:"attributes"`
}

// PteroServer represents a Pterodactyl server
type PteroServer struct {
	Object     string `json:"object"`
	Attributes struct {
		ID          int    `json:"id"`
		ExternalID  string `json:"external_id"`
		UUID        string `json:"uuid"`
		Identifier  string `json:"identifier"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Suspended   bool   `json:"suspended"`
		Limits      struct {
			Memory      int64  `json:"memory"`
			Swap        int64  `json:"swap"`
			Disk        int64  `json:"disk"`
			IO          int    `json:"io"`
			CPU         int    `json:"cpu"`
			Threads     string `json:"threads"`
			OOMDisabled bool   `json:"oom_disabled"`
		} `json:"limits"`
		FeatureLimits struct {
			Databases   int `json:"databases"`
			Allocations int `json:"allocations"`
			Backups     int `json:"backups"`
		} `json:"feature_limits"`
		User       int `json:"user"`
		Node       int `json:"node"`
		Allocation int `json:"allocation"`
		Nest       int `json:"nest"`
		Egg        int `json:"egg"`
		Container  struct {
			StartupCommand string            `json:"startup_command"`
			Image          string            `json:"image"`
			Installed      int               `json:"installed"`
			Environment    map[string]string `json:"environment"`
		} `json:"container"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
}

// PteroUser represents a Pterodactyl user
type PteroUser struct {
	Object     string `json:"object"`
	Attributes struct {
		ID         int    `json:"id"`
		ExternalID string `json:"external_id"`
		UUID       string `json:"uuid"`
		Username   string `json:"username"`
		Email      string `json:"email"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		Language   string `json:"language"`
		RootAdmin  bool   `json:"root_admin"`
		TwoFactor  bool   `json:"2fa"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at"`
	} `json:"attributes"`
}

// PteroDatabase represents a server database
type PteroDatabase struct {
	Object     string `json:"object"`
	Attributes struct {
		ID             int    `json:"id"`
		Server         int    `json:"server"`
		Host           int    `json:"host"`
		Database       string `json:"database"`
		Username       string `json:"username"`
		Remote         string `json:"remote"`
		MaxConnections int    `json:"max_connections"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	} `json:"attributes"`
}

// doRequest performs an HTTP request to the Pterodactyl API
func (c *PterodactylClient) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/application%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Add Cloudflare Access headers if configured
	if c.cfAccessClientID != "" {
		req.Header.Set("CF-Access-Client-Id", c.cfAccessClientID)
		req.Header.Set("CF-Access-Client-Secret", c.cfAccessSecret)
	}

	return c.httpClient.Do(req)
}

// GetLocations fetches all locations from Pterodactyl
func (c *PterodactylClient) GetLocations(ctx context.Context) ([]PteroLocation, error) {
	resp, err := c.doRequest(ctx, "GET", "/locations", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []PteroLocation `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetNodes fetches all nodes from Pterodactyl
func (c *PterodactylClient) GetNodes(ctx context.Context) ([]PteroNode, error) {
	resp, err := c.doRequest(ctx, "GET", "/nodes", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []PteroNode `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetNodeAllocations fetches allocations for a specific node
func (c *PterodactylClient) GetNodeAllocations(ctx context.Context, nodeID int, page int) (*PaginatedResponse, error) {
	path := fmt.Sprintf("/nodes/%d/allocations?page=%d", nodeID, page)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result PaginatedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetNests fetches all nests from Pterodactyl
func (c *PterodactylClient) GetNests(ctx context.Context) ([]PteroNest, error) {
	resp, err := c.doRequest(ctx, "GET", "/nests", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []PteroNest `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetNestEggs fetches all eggs for a specific nest
func (c *PterodactylClient) GetNestEggs(ctx context.Context, nestID int) ([]PteroEgg, error) {
	path := fmt.Sprintf("/nests/%d/eggs?include=variables", nestID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []PteroEgg `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetServers fetches servers with pagination
func (c *PterodactylClient) GetServers(ctx context.Context, page int) (*PaginatedResponse, error) {
	path := fmt.Sprintf("/servers?page=%d&per_page=50", page)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result PaginatedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetUsers fetches users with pagination
func (c *PterodactylClient) GetUsers(ctx context.Context, page int) (*PaginatedResponse, error) {
	path := fmt.Sprintf("/users?page=%d&per_page=50", page)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result PaginatedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetServerDatabases fetches databases for a specific server
func (c *PterodactylClient) GetServerDatabases(ctx context.Context, serverID int) ([]PteroDatabase, error) {
	path := fmt.Sprintf("/servers/%d/databases", serverID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []PteroDatabase `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// TestConnection verifies the API connection is working
func (c *PterodactylClient) TestConnection(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "GET", "/locations", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// SYNC-SPECIFIC METHODS (for full data synchronization)
// ============================================================================

// GetAllLocations fetches all locations with automatic pagination handling
func (c *PterodactylClient) GetAllLocations(ctx context.Context) ([]PteroLocation, error) {
	items, err := c.getAllWithPagination(ctx, "/locations", func(data json.RawMessage) (interface{}, error) {
		var loc PteroLocation
		if err := json.Unmarshal(data, &loc); err != nil {
			return nil, err
		}
		return loc, nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]PteroLocation, len(items))
	for i, item := range items {
		result[i] = item.(PteroLocation)
	}
	return result, nil
}

// GetAllNodes fetches all nodes with automatic pagination handling
func (c *PterodactylClient) GetAllNodes(ctx context.Context) ([]PteroNode, error) {
	items, err := c.getAllWithPagination(ctx, "/nodes", func(data json.RawMessage) (interface{}, error) {
		var node PteroNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, err
		}
		return node, nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]PteroNode, len(items))
	for i, item := range items {
		result[i] = item.(PteroNode)
	}
	return result, nil
}

// GetAllAllocationsForNode fetches all allocations for a specific node with pagination
func (c *PterodactylClient) GetAllAllocationsForNode(ctx context.Context, nodeID int) ([]PteroAllocation, error) {
	path := fmt.Sprintf("/nodes/%d/allocations", nodeID)
	items, err := c.getAllWithPagination(ctx, path, func(data json.RawMessage) (interface{}, error) {
		var alloc PteroAllocation
		if err := json.Unmarshal(data, &alloc); err != nil {
			return nil, err
		}
		return alloc, nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]PteroAllocation, len(items))
	for i, item := range items {
		result[i] = item.(PteroAllocation)
	}
	return result, nil
}

// GetAllNests fetches all nests
func (c *PterodactylClient) GetAllNests(ctx context.Context) ([]PteroNest, error) {
	items, err := c.getAllWithPagination(ctx, "/nests", func(data json.RawMessage) (interface{}, error) {
		var nest PteroNest
		if err := json.Unmarshal(data, &nest); err != nil {
			return nil, err
		}
		return nest, nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]PteroNest, len(items))
	for i, item := range items {
		result[i] = item.(PteroNest)
	}
	return result, nil
}

// GetEggsForNest fetches all eggs for a specific nest (with variables if includeVars is true)
func (c *PterodactylClient) GetEggsForNest(ctx context.Context, nestID int, includeVars bool) ([]PteroEgg, error) {
	path := fmt.Sprintf("/nests/%d/eggs", nestID)
	if includeVars {
		path += "?include=variables"
	}

	items, err := c.getAllWithPagination(ctx, path, func(data json.RawMessage) (interface{}, error) {
		var egg PteroEgg
		if err := json.Unmarshal(data, &egg); err != nil {
			return nil, err
		}
		return egg, nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]PteroEgg, len(items))
	for i, item := range items {
		result[i] = item.(PteroEgg)
	}
	return result, nil
}

// GetAllServers fetches all servers (optionally with allocations)
func (c *PterodactylClient) GetAllServers(ctx context.Context, includeAllocations bool) ([]PteroServer, error) {
	path := "/servers"
	if includeAllocations {
		path += "?include=allocations"
	}

	items, err := c.getAllWithPagination(ctx, path, func(data json.RawMessage) (interface{}, error) {
		var srv PteroServer
		if err := json.Unmarshal(data, &srv); err != nil {
			return nil, err
		}
		return srv, nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]PteroServer, len(items))
	for i, item := range items {
		result[i] = item.(PteroServer)
	}
	return result, nil
}

// GetServerDatabasesWithHost fetches databases for a specific server with host info
func (c *PterodactylClient) GetServerDatabasesWithHost(ctx context.Context, serverID int) ([]PteroDatabase, error) {
	path := fmt.Sprintf("/servers/%d/databases?include=host", serverID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Data []PteroDatabase `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// getAllWithPagination is a helper to fetch all pages and merge results
func (c *PterodactylClient) getAllWithPagination(ctx context.Context, path string, unmarshal func(json.RawMessage) (interface{}, error)) ([]interface{}, error) {
	var allItems []interface{}
	page := 1

	for {
		// Add page param
		separator := "?"
		if string(path[len(path)-1]) == "?" {
			separator = ""
		} else if containsQueryParams(path) {
			separator = "&"
		}

		fullPath := fmt.Sprintf("%s%spage=%d", path, separator, page)
		resp, err := c.doRequest(ctx, "GET", fullPath, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var paginated PaginatedResponse
		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &paginated); err != nil {
			return nil, err
		}

		// Unmarshal data array
		var dataItems []json.RawMessage
		if err := json.Unmarshal(paginated.Data, &dataItems); err != nil {
			return nil, err
		}

		for _, item := range dataItems {
			unmarshaled, err := unmarshal(item)
			if err != nil {
				continue // Skip items that fail to unmarshal
			}
			allItems = append(allItems, unmarshaled)
		}

		// Check if there are more pages
		if page >= paginated.Meta.Pagination.TotalPages {
			break
		}
		page++
	}

	return allItems, nil
}

// containsQueryParams checks if a path already has query parameters
func containsQueryParams(path string) bool {
	for _, c := range path {
		if c == '?' {
			return true
		}
	}
	return false
}
