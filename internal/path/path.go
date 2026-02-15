package path

import (
	"os"

	"github.com/gobwas/glob"
)

var (
	appRootPaths = []string{
		"/app/etc/local.xml",
		"/app/etc/env.php",
		"/wp-config.php",
		"/lib/internal/Magento",
		"/app/design/frontend/Magento",
	}

	excludePaths = []glob.Glob{
		glob.MustCompile("vendor/aws/aws-sdk-php/src/data/**"),       // aws data
		glob.MustCompile("vendor/symfony/intl/Resources/data/**"),     // emoji locales
		glob.MustCompile("vendor/composer/autoload_*.php"),
		glob.MustCompile("generated/**"),
		glob.MustCompile("var/**"),
	}
)

// IsAppRoot reports whether root looks like an application root directory.
func IsAppRoot(root string) bool {
	for _, testPath := range appRootPaths {
		if Exists(root + testPath) {
			return true
		}
	}
	return false
}

// Exists reports whether p exists on the filesystem.
func Exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// IsExcluded reports whether p matches any exclude patterns.
func IsExcluded(p string) bool {
	for _, xx := range excludePaths {
		if xx.Match(p) {
			return true
		}
	}
	return false
}
