package composer

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAuthConfig(t *testing.T) {
	data := []byte(`{
		"http-basic": {
			"repo.magento.com": {
				"username": "pub_key",
				"password": "priv_key"
			}
		},
		"bearer": {
			"packages.example.com": "tok_123"
		},
		"github-oauth": {
			"github.com": "ghp_abc"
		}
	}`)

	ac, err := parseAuthConfig(data)
	require.NoError(t, err)

	assert.Equal(t, "pub_key", ac.HTTPBasic["repo.magento.com"].Username)
	assert.Equal(t, "priv_key", ac.HTTPBasic["repo.magento.com"].Password)
	assert.Equal(t, "tok_123", ac.Bearer["packages.example.com"])
	assert.Equal(t, "ghp_abc", ac.GithubOAuth["github.com"])
}

func TestApplyAuthHTTPBasic(t *testing.T) {
	ac := &AuthConfig{
		HTTPBasic: map[string]BasicAuth{
			"repo.magento.com": {Username: "user", Password: "pass"},
		},
	}
	req, _ := http.NewRequest("GET", "https://repo.magento.com/archives/foo.zip", nil)
	ac.ApplyAuth(req)
	assert.Equal(t, "Basic dXNlcjpwYXNz", req.Header.Get("Authorization"))
}

func TestApplyAuthBearer(t *testing.T) {
	ac := &AuthConfig{
		Bearer: map[string]string{
			"packages.example.com": "tok_123",
		},
	}
	req, _ := http.NewRequest("GET", "https://packages.example.com/p2/foo.json", nil)
	ac.ApplyAuth(req)
	assert.Equal(t, "Bearer tok_123", req.Header.Get("Authorization"))
}

func TestApplyAuthGithubOAuth(t *testing.T) {
	ac := &AuthConfig{
		GithubOAuth: map[string]string{
			"github.com": "ghp_abc",
		},
	}
	req, _ := http.NewRequest("GET", "https://github.com/foo/bar.git/info/refs", nil)
	ac.ApplyAuth(req)
	assert.Equal(t, "token ghp_abc", req.Header.Get("Authorization"))
}

func TestApplyAuthNoMatch(t *testing.T) {
	ac := &AuthConfig{
		HTTPBasic: map[string]BasicAuth{
			"repo.magento.com": {Username: "user", Password: "pass"},
		},
	}
	req, _ := http.NewRequest("GET", "https://other.example.com/foo", nil)
	ac.ApplyAuth(req)
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestApplyAuthNilConfig(t *testing.T) {
	var ac *AuthConfig
	req, _ := http.NewRequest("GET", "https://example.com/foo", nil)
	ac.ApplyAuth(req)
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestFindAuthConfigFromParent(t *testing.T) {
	// Create temp dir structure: root/.composer/auth.json, root/sub/
	root := t.TempDir()
	composerDir := filepath.Join(root, ".composer")
	require.NoError(t, os.MkdirAll(composerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(composerDir, "auth.json"), []byte(`{
		"http-basic": {"repo.magento.com": {"username": "u", "password": "p"}}
	}`), 0o644))

	subDir := filepath.Join(root, "sub", "deep")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	ac, err := findAuthConfigFrom(subDir, root)
	require.NoError(t, err)
	require.NotNil(t, ac)
	assert.Equal(t, "u", ac.HTTPBasic["repo.magento.com"].Username)
}

func TestFindAuthConfigNotFound(t *testing.T) {
	root := t.TempDir()
	ac, err := findAuthConfigFrom(root, root)
	require.NoError(t, err)
	assert.Nil(t, ac)
}

func TestFindAuthConfigInCwd(t *testing.T) {
	root := t.TempDir()
	composerDir := filepath.Join(root, ".composer")
	require.NoError(t, os.MkdirAll(composerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(composerDir, "auth.json"), []byte(`{
		"bearer": {"pkg.example.com": "tok"}
	}`), 0o644))

	ac, err := findAuthConfigFrom(root, root)
	require.NoError(t, err)
	require.NotNil(t, ac)
	assert.Equal(t, "tok", ac.Bearer["pkg.example.com"])
}
