package panels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
			StartupCommand string                 `json:"startup_command"`
			Image          string                 `json:"image"`
			Installed      int                    `json:"installed"`
			Environment    map[string]interface{} `json:"environment"`
		} `json:"container"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"attributes"`
	Relationships struct {
		Allocations struct {
			Object string            `json:"object"`
			Data   []PteroAllocation `json:"data"`
		} `json:"allocations"`
	} `json:"relationships"`
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

// ClientServer represents a server from Client API perspective
type ClientServer struct {
	Object     string `json:"object"`
	Attributes struct {
		ServerOwner bool     `json:"server_owner"`
		Identifier  string   `json:"identifier"`
		UUID        string   `json:"uuid"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Status      string   `json:"status"`
		InternalID  int      `json:"internal_id"`
		Permissions []string `json:"permissions"`
	} `json:"attributes"`
}

// ClientSubuser represents a subuser from Client API
type ClientSubuser struct {
	Object     string `json:"object"`
	Attributes struct {
		UUID             string   `json:"uuid"`
		Username         string   `json:"username"`
		Email            string   `json:"email"`
		Image            string   `json:"image"`
		TwoFactorEnabled bool     `json:"2fa_enabled"`
		Permissions      []string `json:"permissions"`
		CreatedAt        string   `json:"created_at"`
	} `json:"attributes"`
}

// doRequest performs an HTTP request to the Pterodactyl API using the application API key
func (c *PterodactylClient) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/application%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if c.apiKey == "" {
		// Log warning if API key is empty
		fmt.Printf("Warning: Application API key is empty\n")
	} else {
		// Log the length and first/last chars for debugging (never log full key)
		keyLen := len(c.apiKey)
		keyPreview := "***"
		if keyLen > 8 {
			keyPreview = c.apiKey[:4] + "..." + c.apiKey[keyLen-4:]
		}
		fmt.Printf("DEBUG: Sending request with API key (length: %d, preview: %s)\n", keyLen, keyPreview)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	// Add Cloudflare Access headers if configured
	if c.cfAccessClientID != "" {
		req.Header.Set("CF-Access-Client-Id", c.cfAccessClientID)
		req.Header.Set("CF-Access-Client-Secret", c.cfAccessSecret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("ERROR: Pterodactyl API returned %d for %s %s: %s\n", resp.StatusCode, method, url, string(body))
		// Return a synthetic response with the status so callers can handle it
		resp.Body = io.NopCloser(strings.NewReader(string(body)))
	}
	return resp, nil
}

// doClientRequest performs an HTTP request to the Pterodactyl Client API using the client API key
func (c *PterodactylClient) doClientRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if c.clientAPIKey == "" {
		// Fall back to application API if client key not available
		return c.doRequest(ctx, method, path, body)
	}

	url := fmt.Sprintf("%s/api/client%s", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.clientAPIKey))

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

// GetServerResources fetches live resource usage for a specific server (requires client API key)
// Returns CPU, memory, disk, and network usage data
func (c *PterodactylClient) GetServerResources(ctx context.Context, serverUUID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/servers/%s/resources", serverUUID)
	resp, err := c.doClientRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetServerDetailWithIncludes fetches detailed server info with specific includes (allocations, variables, etc)
func (c *PterodactylClient) GetServerDetailWithIncludes(ctx context.Context, serverID int, includes []string) (*PteroServer, error) {
	path := fmt.Sprintf("/servers/%d", serverID)
	if len(includes) > 0 {
		path += "?include=" + strings.Join(includes, ",")
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Object     string      `json:"object"`
		Attributes PteroServer `json:"attributes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Attributes, nil
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
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response body: %w", readErr)
		}
		if err := json.Unmarshal(body, &paginated); err != nil {
			fmt.Printf("ERROR: failed to unmarshal paginated response from %s: %v\nBody: %s\n", fullPath, err, string(body))
			return nil, err
		}

		// Log pagination info on first page
		if page == 1 {
			fmt.Printf("DEBUG: Pagination for %s — total=%d pages=%d\n",
				fullPath, paginated.Meta.Pagination.Total, paginated.Meta.Pagination.TotalPages)
		}

		// Unmarshal data array
		var dataItems []json.RawMessage
		if err := json.Unmarshal(paginated.Data, &dataItems); err != nil {
			return nil, err
		}

		for _, item := range dataItems {
			unmarshaled, err := unmarshal(item)
			if err != nil {
				fmt.Printf("WARN: failed to unmarshal item from %s: %v\nItem: %s\n", fullPath, err, string(item))
				continue
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

// UpdateServerEnvironment updates environment variables for a server
func (c *PterodactylClient) UpdateServerEnvironment(ctx context.Context, serverUUID string, envVars map[string]string) error {
	path := fmt.Sprintf("/api/client/servers/%s/settings/environment", serverUUID)
	body := map[string]map[string]string{"variables": envVars}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.doClientRequest(ctx, "POST", path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to update server environment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update server environment: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetClientServers fetches servers accessible to the client API user
func (c *PterodactylClient) GetClientServers(ctx context.Context) ([]ClientServer, error) {
	if c.clientAPIKey == "" {
		return nil, fmt.Errorf("client API key not configured")
	}

	resp, err := c.doClientRequest(ctx, "GET", "/servers", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []ClientServer `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetServerSubusers fetches subusers for a specific server (requires owner or admin)
func (c *PterodactylClient) GetServerSubusers(ctx context.Context, serverUUID string) ([]ClientSubuser, error) {
	if c.clientAPIKey == "" {
		return nil, fmt.Errorf("client API key not configured")
	}

	path := fmt.Sprintf("/servers/%s/users", serverUUID)
	resp, err := c.doClientRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []ClientSubuser `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
