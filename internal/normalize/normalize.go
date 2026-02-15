package normalize

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gwillem/corediff/internal/chunker"
	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/zeebo/xxh3"
)

const minSize = 10 // skip shorter lines

var (
	normalizeRx = []*regexp.Regexp{
		regexp.MustCompile(`'reference' => '[a-f0-9]{40}',`),
	}

	// rxGuard is a cheap prefix check: only run the regex if the line
	// contains this literal substring. Avoids regex overhead on 99%+ of lines.
	rxGuard = []byte("'reference' =>")

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
	if len(b) < minSize {
		return b
	}
	for i := range skipLines {
		if bytes.HasPrefix(b, skipLines[i]) {
			return []byte{}
		}
	}
	if bytes.Contains(b, rxGuard) {
		for _, rx := range normalizeRx {
			b = rx.ReplaceAllLiteral(b, nil)
		}
	}
	return b
}

func hash(b []byte) uint64 {
	return xxh3.Hash(b)
}

// HashLine normalizes a line, then hashes it (chunking if minified).
// Calls fn for each hash produced. fn returns true to continue, false to stop.
// Empty/comment lines produce no calls to fn.
func HashLine(raw []byte, fn func(uint64) bool) {
	if len(raw) < minSize {
		return
	}
	norm := Line(raw)
	if len(norm) < minSize {
		return
	}
	// Fast path: lines within chunk threshold (vast majority) produce a
	// single hash without going through ChunkLine.
	if len(norm) <= chunker.ChunkThreshold {
		fn(xxh3.Hash(norm))
		return
	}
	for _, c := range chunker.ChunkLine(norm) {
		if !fn(xxh3.Hash(c)) {
			return
		}
	}
}

// PathHash returns the hash for a path entry (prefixed with "path:").
func PathHash(p string) uint64 {
	return xxh3.Hash([]byte("path:" + p))
}

// HasValidExt reports whether path has a recognized code file extension.
func HasValidExt(path string) bool {
	return slices.Contains(ScanExts, strings.TrimLeft(filepath.Ext(path), "."))
}

const MaxTokenSize = 1024 * 1024 * 10 // 10 MB

// NewScanBuf allocates a reusable scanner buffer. Pass it to HashReader
// to avoid a 10 MB allocation per call. Safe to reuse across sequential calls.
func NewScanBuf() []byte {
	return make([]byte, MaxTokenSize)
}

// HashReader scans lines from r, normalizes and hashes each line,
// and adds new hashes to db. Returns (new hashes added, total hashes processed).
// If logf is non-nil, each hash is logged as "HASH line".
// If buf is non-nil, it is used as the scanner buffer (see NewScanBuf).
func HashReader(r io.Reader, db *hashdb.HashDB, logf func(string, ...any), buf []byte) (int, int) {
	scanner := bufio.NewScanner(r)
	if buf == nil {
		buf = make([]byte, MaxTokenSize)
	}
	scanner.Buffer(buf, MaxTokenSize)

	var added, total int
	for scanner.Scan() {
		line := scanner.Bytes()
		HashLine(line, func(h uint64) bool {
			total++
			if !db.Contains(h) {
				db.Add(h)
				added++
			}
			if logf != nil {
				logf("      %016x %s", h, line)
			}
			return true
		})
	}
	return added, total
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
