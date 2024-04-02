package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cespare/xxhash/v2"
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

func hash(b []byte) uint64 {
	return xxhash.Sum64(b)
}

func pathHash(p string) uint64 {
	return hash([]byte("path:" + p))
}

func pathIsExcluded(p string) bool {
	// Does p match any of excludePaths ?
	for _, xx := range excludePaths {
		if xx.Match(p) {
			return true
		}
	}
	return false
}
