package main

import (
	"testing"

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/stretchr/testify/assert"
)

func Test_parseFile(t *testing.T) {
	hdb := hashdb.HashDB{}
	updateDB := true
	hits, lines := parseFileWithDB("../../fixture/docroot/odd-encoding.js", hdb, updateDB)
	assert.Equal(t, 220, len(hdb))
	assert.Equal(t, 220, len(hits))
	assert.Equal(t, 220, len(lines))
}
