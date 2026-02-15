package gitindex

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
)

const maxZipSize = 100 * 1024 * 1024 // 100 MB

// IndexZip downloads a zip from zipURL (or reads from cache) and indexes its contents into db.
func IndexZip(zipURL string, db *hashdb.HashDB, opts IndexOptions) error {
	data, err := fetchZip(zipURL, opts)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}

	prefix := commonRootPrefix(zr.File)

	scanBuf := normalize.NewScanBuf()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := strings.TrimPrefix(f.Name, prefix)
		if name == "" {
			continue
		}

		if !opts.AllValidText && !normalize.HasValidExt(name) {
			opts.log(3, "skip %s (no valid ext)", name)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}

		// Check UTF-8 validity
		buf := make([]byte, 8*1024)
		n, readErr := rc.Read(buf)
		if readErr != nil && readErr != io.EOF {
			rc.Close()
			continue
		}
		if !utf8.Valid(buf[:n]) {
			opts.log(3, "skip %s (invalid utf8)", name)
			rc.Close()
			continue
		}

		// Re-open for full reading
		rc.Close()
		rc, err = f.Open()
		if err != nil {
			continue
		}

		if !opts.NoPlatform {
			storedPath := opts.PathPrefix + name
			db.Add(normalize.PathHash(storedPath))
			opts.log(3, "hash %s", storedPath)
		} else {
			opts.log(3, "hash %s", name)
		}

		normalize.HashReader(rc, func(h uint64, rawLine []byte) {
			if !db.Contains(h) {
				db.Add(h)
			}
			if opts.Verbose >= 4 {
				fmt.Printf("      %016x %s\n", h, rawLine)
			}
		}, scanBuf)
		rc.Close()
	}

	return nil
}

// fetchZip returns the zip data for zipURL, using the cache dir if configured.
func fetchZip(zipURL string, opts IndexOptions) ([]byte, error) {
	if opts.CacheDir != "" {
		cachePath := zipCachePath(opts.CacheDir, zipURL)
		if data, err := os.ReadFile(cachePath); err == nil {
			opts.log(3, "cache hit %s", zipURL)
			return data, nil
		}

		data, err := downloadZip(zipURL, opts)
		if err != nil {
			return nil, err
		}

		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
			return data, nil // download succeeded, cache write failed — continue
		}
		if err := os.WriteFile(cachePath, data, 0o644); err != nil {
			opts.log(1, "warning: caching zip: %v", err)
		}
		return data, nil
	}

	return downloadZip(zipURL, opts)
}

func downloadZip(zipURL string, opts IndexOptions) ([]byte, error) {
	resp, err := opts.httpClient().Get(zipURL)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", zipURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("downloading %s: HTTP %d", zipURL, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxZipSize))
	if err != nil {
		return nil, fmt.Errorf("reading zip: %w", err)
	}
	return data, nil
}

// zipCachePath returns a deterministic cache file path for a zip URL.
func zipCachePath(cacheDir, zipURL string) string {
	h := sha256.Sum256([]byte(zipURL))
	return filepath.Join(cacheDir, "zip", hex.EncodeToString(h[:12])+".zip")
}

// commonRootPrefix finds a shared directory prefix across all zip entries.
// GitHub zipballs typically have a "repo-hash/" prefix.
func commonRootPrefix(files []*zip.File) string {
	if len(files) == 0 {
		return ""
	}

	// Find first directory entry or derive from first file
	var prefix string
	for _, f := range files {
		idx := strings.IndexByte(f.Name, '/')
		if idx < 0 {
			return "" // top-level file, no common prefix
		}
		candidate := f.Name[:idx+1]
		if prefix == "" {
			prefix = candidate
		} else if candidate != prefix {
			return "" // different prefixes
		}
	}
	return prefix
}
