package path

import "os"

// Exists reports whether p exists on the filesystem.
func Exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
