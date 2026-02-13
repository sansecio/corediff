package composer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Repository represents a single entry in the composer.json "repositories" section.
type Repository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// SourceRef holds git source metadata from composer.lock.
type SourceRef struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
}

// DistRef holds distribution (zip) metadata from composer.lock.
type DistRef struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
}

// LockPackage represents a single entry in the composer.lock "packages" array.
type LockPackage struct {
	Name    string    `json:"name"`
	Version string    `json:"version"`
	Type    string    `json:"type"`
	Source  SourceRef `json:"source"`
	Dist    DistRef   `json:"dist"`
}

// ComposerProject holds the parsed result of a composer.json + composer.lock pair.
type ComposerProject struct {
	Repos    []Repository  // composer repositories (type=composer only)
	Packages []LockPackage // filtered packages from lock file (with source/dist info)
}

// ParseProject reads a composer.json and its sibling composer.lock, returning
// the list of composer repositories and non-meta, non-platform package names.
func ParseProject(jsonPath string) (*ComposerProject, error) {
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("reading composer.json: %w", err)
	}

	lockPath := filepath.Join(filepath.Dir(jsonPath), "composer.lock")
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("reading composer.lock: %w", err)
	}

	repos, err := parseRepos(jsonData)
	if err != nil {
		return nil, fmt.Errorf("parsing repositories: %w", err)
	}

	pkgs, err := parseLockPackages(lockData)
	if err != nil {
		return nil, fmt.Errorf("parsing lock packages: %w", err)
	}

	// Ensure packagist.org is present as implicit fallback
	hasPackagist := false
	for _, r := range repos {
		if strings.Contains(r.URL, "packagist.org") {
			hasPackagist = true
			break
		}
	}
	if !hasPackagist {
		repos = append(repos, Repository{
			Type: "composer",
			URL:  "https://repo.packagist.org",
		})
	}

	return &ComposerProject{
		Repos:    repos,
		Packages: pkgs,
	}, nil
}

// parseRepos extracts composer-type repositories from the JSON data.
// Handles both object format {"name": {type, url}} and array format [{type, url}].
func parseRepos(data []byte) ([]Repository, error) {
	var raw struct {
		Repositories json.RawMessage `json:"repositories"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.Repositories == nil {
		return nil, nil
	}

	// Try object format first (keyed by name)
	var objRepos map[string]Repository
	if err := json.Unmarshal(raw.Repositories, &objRepos); err == nil {
		var repos []Repository
		for _, r := range objRepos {
			if r.Type == "composer" {
				r.URL = NormalizeRepoURL(r.URL)
				repos = append(repos, r)
			}
		}
		return repos, nil
	}

	// Fall back to array format
	var arrRepos []Repository
	if err := json.Unmarshal(raw.Repositories, &arrRepos); err != nil {
		return nil, fmt.Errorf("repositories is neither object nor array: %w", err)
	}
	var repos []Repository
	for _, r := range arrRepos {
		if r.Type == "composer" {
			r.URL = NormalizeRepoURL(r.URL)
			repos = append(repos, r)
		}
	}
	return repos, nil
}

// NormalizeRepoURL strips trailing /packages.json and trailing slashes from a repo URL.
func NormalizeRepoURL(u string) string {
	u = strings.TrimSuffix(u, "/packages.json")
	u = strings.TrimRight(u, "/")
	return u
}

// parseLockPackages extracts non-meta, non-platform packages from composer.lock.
// Only "packages" is read; "packages-dev" is skipped.
func parseLockPackages(data []byte) ([]LockPackage, error) {
	var lock struct {
		Packages []LockPackage `json:"packages"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}

	var pkgs []LockPackage
	for _, p := range lock.Packages {
		if isPlatformPackage(p.Name) {
			continue
		}
		if p.Type == "metapackage" {
			continue
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// isPlatformPackage returns true for "php" and "ext-*" entries.
func isPlatformPackage(name string) bool {
	return name == "php" || strings.HasPrefix(name, "ext-")
}
