package platform

import (
	"os"
	"path/filepath"

	"github.com/gobwas/glob"
)

// Platform describes a web application platform with its detection rules,
// exclude patterns, and optional file validation.
type Platform struct {
	Name          string
	SentinelPaths []string    // relative paths that identify this platform
	ExcludePaths  []glob.Glob // compiled globs for paths to skip during scanning
	DefaultDBURL  string      // URL for the default hash database (empty = none)

	// ValidateFile is called for files that would otherwise be skipped
	// (not in DB, or excluded). Returns handled=true if it took
	// responsibility for the file, with flagged line numbers and content.
	// nil means no special validation.
	ValidateFile func(relPath, absPath string, scanBuf []byte) (handled bool, hits []int, lines [][]byte)
}

// IsExcluded reports whether relPath matches any of the platform's exclude patterns.
func (p *Platform) IsExcluded(relPath string) bool {
	for _, g := range p.ExcludePaths {
		if g.Match(relPath) {
			return true
		}
	}
	return false
}

var (
	Magento2 = &Platform{
		Name: "magento2",
		SentinelPaths: []string{
			"app/etc/env.php",
			"lib/internal/Magento",
			"app/design/frontend/Magento",
		},
		ExcludePaths: []glob.Glob{
			glob.MustCompile("var/**"),
			glob.MustCompile("vendor/composer/autoload_*.php"),
		},
		DefaultDBURL: "https://sansec.io/downloads/corediff-db/m2.db3",
		ValidateFile: validateMagentoGenerated,
	}

	Magento1 = &Platform{
		Name: "magento1",
		SentinelPaths: []string{
			"app/etc/local.xml",
		},
	}

	WordPress = &Platform{
		Name: "wordpress",
		SentinelPaths: []string{
			"wp-config.php",
		},
	}

	// platforms is ordered by detection priority (most specific first).
	platforms = []*Platform{Magento2, Magento1, WordPress}
)

// Detect identifies the platform at root by checking for sentinel paths.
// Returns nil if no known platform is detected.
func Detect(root string) *Platform {
	for _, p := range platforms {
		for _, sentinel := range p.SentinelPaths {
			if exists(filepath.Join(root, sentinel)) {
				return p
			}
		}
	}
	return nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
