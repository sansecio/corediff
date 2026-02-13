package normalize

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cespare/xxhash/v2"
)

var (
	normalizeRx = []*regexp.Regexp{
		regexp.MustCompile(`'reference' => '[a-f0-9]{40}',`),
	}

	skipLines = [][]byte{
		[]byte("*"),
		[]byte("/*"),
		[]byte("//"),
		[]byte("#"),
	}

	// ScanExts lists file extensions to scan.
	ScanExts = []string{"php", "phtml", "js", "htaccess", "sh"}
)

// Line normalizes a line of code by trimming whitespace,
// stripping comments, and applying regex filters.
func Line(b []byte) []byte {
	b = bytes.TrimSpace(b)
	for _, prefix := range skipLines {
		if bytes.HasPrefix(b, prefix) {
			return []byte{}
		}
	}
	for _, rx := range normalizeRx {
		b = rx.ReplaceAllLiteral(b, nil)
	}
	return b
}

// Hash returns the xxhash64 of b.
func Hash(b []byte) uint64 {
	return xxhash.Sum64(b)
}

// PathHash returns the hash for a path entry (prefixed with "path:").
func PathHash(p string) uint64 {
	return Hash([]byte("path:" + p))
}

// HasValidExt reports whether path has a recognized code file extension.
func HasValidExt(path string) bool {
	got := strings.TrimLeft(filepath.Ext(path), ".")
	for _, want := range ScanExts {
		if got == want {
			return true
		}
	}
	return false
}

// IsValidUtf8 checks if the first 8KB of a file is valid UTF-8.
func IsValidUtf8(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 1024*8)
	if _, err := f.Read(buf); err != nil {
		return false
	}

	valid := utf8.Valid(buf)
	if !valid {
		fmt.Println("Invalid UTF-8:", path)
	}
	return valid
}
