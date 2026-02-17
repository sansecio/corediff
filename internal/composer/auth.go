package composer

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// BasicAuth holds username/password for http-basic auth.
type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthConfig represents a Composer auth.json file.
type AuthConfig struct {
	HTTPBasic   map[string]BasicAuth `json:"http-basic"`
	Bearer      map[string]string    `json:"bearer"`
	GithubOAuth map[string]string    `json:"github-oauth"`
}

// FindAuthConfig searches for a Composer auth.json file by walking from
// the current working directory up to /, then checking $HOME/.composer/auth.json.
// Returns (nil, nil) if no auth.json is found.
func FindAuthConfig() (*AuthConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	ac, err := findAuthConfigFrom(cwd, "/")
	if err != nil || ac != nil {
		return ac, err
	}

	// Fall back to $HOME/.composer/auth.json
	if home != "" {
		return tryLoadAuth(filepath.Join(home, ".composer", "auth.json"))
	}
	return nil, nil
}

// findAuthConfigFrom walks from startDir up to stopDir (inclusive), looking for
// .composer/auth.json in each directory. Exported for testing.
func findAuthConfigFrom(startDir, stopDir string) (*AuthConfig, error) {
	dir := startDir
	for {
		ac, err := tryLoadAuth(filepath.Join(dir, ".composer", "auth.json"))
		if err != nil {
			return nil, err
		}
		if ac != nil {
			return ac, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		if dir == stopDir {
			break
		}
		dir = parent
	}
	return nil, nil
}

// tryLoadAuth reads and parses a single auth.json path.
// Returns (nil, nil) if the file does not exist.
func tryLoadAuth(path string) (*AuthConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseAuthConfig(data)
}

func parseAuthConfig(data []byte) (*AuthConfig, error) {
	var ac AuthConfig
	if err := json.Unmarshal(data, &ac); err != nil {
		return nil, err
	}
	return &ac, nil
}

// Hosts returns a summary of configured auth hosts, e.g. ["http-basic: repo.magento.com", "bearer: pkg.example.com"].
// Safe to call on a nil receiver (returns nil).
func (a *AuthConfig) Hosts() []string {
	if a == nil {
		return nil
	}
	var hosts []string
	for h := range a.HTTPBasic {
		hosts = append(hosts, "http-basic: "+h)
	}
	for h := range a.Bearer {
		hosts = append(hosts, "bearer: "+h)
	}
	for h := range a.GithubOAuth {
		hosts = append(hosts, "github-oauth: "+h)
	}
	return hosts
}

// ApplyAuth sets the Authorization header on req based on the request hostname.
// Safe to call on a nil receiver (no-op).
func (a *AuthConfig) ApplyAuth(req *http.Request) {
	if a == nil {
		return
	}
	host := req.URL.Host

	if cred, ok := a.HTTPBasic[host]; ok {
		encoded := base64.StdEncoding.EncodeToString([]byte(cred.Username + ":" + cred.Password))
		req.Header.Set("Authorization", "Basic "+encoded)
		return
	}
	if token, ok := a.Bearer[host]; ok {
		req.Header.Set("Authorization", "Bearer "+token)
		return
	}
	if token, ok := a.GithubOAuth[host]; ok {
		req.Header.Set("Authorization", "token "+token)
		return
	}
}
