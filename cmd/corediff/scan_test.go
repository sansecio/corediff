package main

import (
	"testing"

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
	"github.com/stretchr/testify/assert"
)

func Test_addFileHashes(t *testing.T) {
	hdb := hashdb.New()
	buf := normalize.NewScanBuf()
	n := addFileHashes("../../fixture/docroot/odd-encoding.js", hdb, buf)
	assert.Equal(t, 203, hdb.Len())
	assert.Equal(t, 203, n)
}

func Test_scanFileWithDB(t *testing.T) {
	// First populate the DB
	hdb := hashdb.New()
	buf := normalize.NewScanBuf()
	addFileHashes("../../fixture/docroot/odd-encoding.js", hdb, buf)

	// Scanning the same file should find zero unrecognized lines
	hits, lines := scanFileWithDB("../../fixture/docroot/odd-encoding.js", hdb, buf)
	assert.Equal(t, 0, len(hits))
	assert.Equal(t, 0, len(lines))
}
