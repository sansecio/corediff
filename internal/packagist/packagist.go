package packagist

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Version holds metadata for a single package version from Packagist.
type Version struct {
	Version string `json:"version"`
	Source  struct {
		URL       string `json:"url"`
		Type      string `json:"type"`
		Reference string `json:"reference"`
	} `json:"source"`
	Dist struct {
		URL       string `json:"url"`
		Type      string `json:"type"`
		Reference string `json:"reference"`
	} `json:"dist"`
}

// Client talks to the Packagist API.
type Client struct {
	BaseURL string       // default "https://repo.packagist.org"
	HTTP    *http.Client // injectable for tests
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return "https://repo.packagist.org"
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// Versions fetches all versions for a Packagist package.
// pkg must be in "vendor/package" format.
func (c *Client) Versions(pkg string) ([]Version, error) {
	parts := strings.SplitN(pkg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid package name %q: must be vendor/package", pkg)
	}

	url := fmt.Sprintf("%s/p2/%s/%s.json", c.baseURL(), parts[0], parts[1])
	resp, err := c.httpClient().Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("packagist returned %d for %s", resp.StatusCode, pkg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result struct {
		Packages map[string][]Version `json:"packages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	versions, ok := result.Packages[pkg]
	if !ok {
		return nil, fmt.Errorf("package %q not found in response", pkg)
	}

	return versions, nil
}
