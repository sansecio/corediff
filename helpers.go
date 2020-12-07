package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

func isCmsRoot(root string) bool {
	for _, testPath := range cmsPaths {
		full := root + testPath
		if pathExists(full) {
			return true
		}
	}
	return false
}

func shouldHighlight(b []byte) bool {
	for _, p := range highlightPatterns {
		m, _ := regexp.Match(p, b)
		if m {
			return true
		}
	}
	return false
}

func normalizeLine(b []byte) []byte {
	// Also strip slashes comments etc
	b = bytes.TrimSpace(b)
	for _, prefix := range skipLines {
		if bytes.HasPrefix(b, prefix) {
			return []byte{}
		}
	}
	return b
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func hasValidExt(path string) bool {
	got := strings.TrimLeft(filepath.Ext(path), ".")
	for _, want := range scanExts {
		if got == want {
			return true
		}
	}
	return false
}

func logVerbose(a ...interface{}) {
	if logLevel >= 3 {
		fmt.Println(a...)
	}
}
func logInfo(a ...interface{}) {
	fmt.Println(a...)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func hash(b []byte) [16]byte {
	return md5.Sum(b)
}

func pathHash(p string) [16]byte {
	return hash([]byte("path:" + p))
}

func pathIsExcluded(p string) bool {
	// Does p match any of excludePaths ?
	for _, xx := range excludePaths {
		// TODO: optim with precompile
		if glob.MustCompile(xx).Match(p) {
			return true
		}
	}
	return false
}
