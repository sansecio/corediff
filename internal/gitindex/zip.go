package gitindex

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
)

const maxZipSize = 100 * 1024 * 1024 // 100 MB

// IndexZip downloads a zip from zipURL and indexes its contents into db.
func IndexZip(zipURL string, db *hashdb.HashDB, opts IndexOptions) error {
	resp, err := opts.httpClient().Get(zipURL)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", zipURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("downloading %s: HTTP %d", zipURL, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxZipSize))
	if err != nil {
		return fmt.Errorf("reading zip: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}

	prefix := commonRootPrefix(zr.File)

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := strings.TrimPrefix(f.Name, prefix)
		if name == "" {
			continue
		}

		if !opts.AllValidText && !normalize.HasValidExt(name) {
			opts.logf("    skip %s (no valid ext)", name)
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
			opts.logf("    skip %s (invalid utf8)", name)
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
			opts.logf("    hash %s", storedPath)
		} else {
			opts.logf("    hash %s", name)
		}

		normalize.HashReader(rc, db, opts.LineLogf)
		rc.Close()
	}

	return nil
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
