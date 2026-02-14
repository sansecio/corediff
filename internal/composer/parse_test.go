package composer

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
}

func pkgNames(pkgs []LockPackage) []string {
	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		names[i] = p.Name
	}
	return names
}

func TestParseProject(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"composer.json": `{
			"repositories": {
				"private": {"type": "composer", "url": "https://packages.example.com"},
				"packagist": {"type": "composer", "url": "https://repo.packagist.org"}
			}
		}`,
		"composer.lock": `{
			"packages": [
				{"name": "vendor/alpha", "type": "library"},
				{"name": "vendor/beta", "type": "library"}
			]
		}`,
	})

	proj, err := ParseProject(filepath.Join(dir, "composer.json"))
	require.NoError(t, err)
	assert.Len(t, proj.Repos, 2)
	assert.Equal(t, []string{"vendor/alpha", "vendor/beta"}, pkgNames(proj.Packages))
}

func TestParseProject_ArrayRepos(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"composer.json": `{
			"repositories": [
				{"type": "composer", "url": "https://packages.example.com"},
				{"type": "vcs", "url": "https://github.com/foo/bar"}
			]
		}`,
		"composer.lock": `{
			"packages": [
				{"name": "foo/bar", "type": "library"}
			]
		}`,
	})

	proj, err := ParseProject(filepath.Join(dir, "composer.json"))
	require.NoError(t, err)

	// Only composer-type repos, plus implicit packagist
	var urls []string
	for _, r := range proj.Repos {
		urls = append(urls, r.URL)
	}
	assert.Contains(t, urls, "https://packages.example.com")
	assert.Contains(t, urls, "https://repo.packagist.org")
	assert.NotContains(t, urls, "https://github.com/foo/bar")
}

func TestParseProject_SkipsMetapackages(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"composer.json": `{"repositories": []}`,
		"composer.lock": `{
			"packages": [
				{"name": "vendor/real", "type": "library"},
				{"name": "vendor/meta", "type": "metapackage"}
			]
		}`,
	})

	proj, err := ParseProject(filepath.Join(dir, "composer.json"))
	require.NoError(t, err)
	assert.Equal(t, []string{"vendor/real"}, pkgNames(proj.Packages))
}

func TestParseProject_SkipsPlatform(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"composer.json": `{"repositories": []}`,
		"composer.lock": `{
			"packages": [
				{"name": "vendor/real", "type": "library"},
				{"name": "php", "type": ""},
				{"name": "ext-json", "type": ""}
			]
		}`,
	})

	proj, err := ParseProject(filepath.Join(dir, "composer.json"))
	require.NoError(t, err)
	assert.Equal(t, []string{"vendor/real"}, pkgNames(proj.Packages))
}

func TestParseProject_ImplicitPackagist(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"composer.json": `{
			"repositories": {
				"private": {"type": "composer", "url": "https://packages.example.com"}
			}
		}`,
		"composer.lock": `{"packages": []}`,
	})

	proj, err := ParseProject(filepath.Join(dir, "composer.json"))
	require.NoError(t, err)

	var urls []string
	for _, r := range proj.Repos {
		urls = append(urls, r.URL)
	}
	assert.Contains(t, urls, "https://repo.packagist.org")
}

func TestParseProject_PreservesSourceDist(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"composer.json": `{"repositories": []}`,
		"composer.lock": `{
			"packages": [{
				"name": "vendor/alpha",
				"type": "library",
				"source": {"type": "git", "url": "https://github.com/vendor/alpha.git", "reference": "abc123"},
				"dist": {"type": "zip", "url": "https://api.github.com/repos/vendor/alpha/zipball/abc123", "reference": "abc123"}
			}]
		}`,
	})

	proj, err := ParseProject(filepath.Join(dir, "composer.json"))
	require.NoError(t, err)
	require.Len(t, proj.Packages, 1)
	pkg := proj.Packages[0]
	assert.Equal(t, "git", pkg.Source.Type)
	assert.Equal(t, "https://github.com/vendor/alpha.git", pkg.Source.URL)
	assert.Equal(t, "abc123", pkg.Source.Reference)
	assert.Equal(t, "https://api.github.com/repos/vendor/alpha/zipball/abc123", pkg.Dist.URL)
}

func TestParseProject_NoLock(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"composer.json": `{"repositories": []}`,
	})

	_, err := ParseProject(filepath.Join(dir, "composer.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "composer.lock")
}

func TestParseReplace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name: "magento2 style",
			input: `{
				"name": "magento/magento2ce",
				"replace": {
					"magento/module-catalog": "*",
					"magento/module-checkout": "*",
					"magento/module-sales": "*"
				}
			}`,
			want: []string{"magento/module-catalog", "magento/module-checkout", "magento/module-sales"},
		},
		{
			name:  "empty replace",
			input: `{"name": "vendor/pkg", "replace": {}}`,
			want:  nil,
		},
		{
			name:  "no replace section",
			input: `{"name": "vendor/pkg"}`,
			want:  nil,
		},
		{
			name: "filters non-package entries",
			input: `{
				"replace": {
					"magento/module-catalog": "*",
					"not-a-package": "*"
				}
			}`,
			want: []string{"magento/module-catalog"},
		},
		{
			name: "self.version constraint",
			input: `{
				"replace": {
					"symfony/polyfill-php80": "self.version"
				}
			}`,
			want: []string{"symfony/polyfill-php80"},
		},
		{
			name:    "invalid json",
			input:   `{not valid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReplace([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			slices.Sort(got)
			slices.Sort(tt.want)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard package",
			input: `{"name": "magento/magento2ce", "version": "2.4.7"}`,
			want:  "magento/magento2ce",
		},
		{
			name:  "no name field",
			input: `{"version": "1.0.0"}`,
			want:  "",
		},
		{
			name:  "empty name",
			input: `{"name": ""}`,
			want:  "",
		},
		{
			name:  "invalid json",
			input: `{not valid`,
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseName([]byte(tt.input)))
		})
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://repo.magento.com/", "https://repo.magento.com"},
		{"https://ci.swissuplabs.com/api/packages.json", "https://ci.swissuplabs.com/api"},
		{"https://repo.packagist.org", "https://repo.packagist.org"},
		{"https://example.com/repo/packages.json/", "https://example.com/repo/packages.json"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeRepoURL(tt.input))
		})
	}
}
