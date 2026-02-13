package normalize

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cespare/xxhash/v2"
	"github.com/gwillem/corediff/internal/chunker"
	"github.com/gwillem/corediff/internal/hashdb"
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

// HashLine normalizes a line, then chunks it if it's long (minified code).
// Returns one or more hashes. Empty/comment lines return nil.
func HashLine(raw []byte) []uint64 {
	norm := Line(raw)
	if len(norm) == 0 {
		return nil
	}
	chunks := chunker.ChunkLine(norm)
	hashes := make([]uint64, len(chunks))
	for i, c := range chunks {
		hashes[i] = Hash(c)
	}
	return hashes
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

const maxTokenSize = 1024 * 1024 * 10 // 10 MB

// HashReader scans lines from r, normalizes and hashes each line,
// and adds new hashes to db. Returns the count of new hashes added.
// If logf is non-nil, each hash is logged as "HASH line".
func HashReader(r io.Reader, db *hashdb.HashDB, logf func(string, ...any)) int {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)

	added := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		hashes := HashLine(line)
		for _, h := range hashes {
			if !db.Contains(h) {
				db.Add(h)
				added++
			}
			if logf != nil {
				logf("      %016x %s", h, line)
			}
		}
	}
	return added
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
