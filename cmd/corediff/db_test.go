package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBAdd(t *testing.T) {
	db := hashdb.New()
	addPath("../../fixture/docroot", db, false, false, false)
	db.Compact()

	// Should have line hashes
	assert.Greater(t, db.Len(), 0)

	// Should also have path hashes (ignorePaths=false, noPlatform=false)
	// Check that at least one known path hash is present
	assert.True(t, db.Contains(normalize.PathHash("test.php")))
	assert.True(t, db.Contains(normalize.PathHash("highlight.php")))
}

func TestDBAddNoPlatform(t *testing.T) {
	dbWith := hashdb.New()
	addPath("../../fixture/docroot", dbWith, false, false, false)
	dbWith.Compact()

	dbWithout := hashdb.New()
	addPath("../../fixture/docroot", dbWithout, false, false, true)
	dbWithout.Compact()

	// noPlatform=true should have fewer hashes (no path hashes)
	assert.Greater(t, dbWith.Len(), dbWithout.Len())

	// Path hashes should NOT be present in noPlatform DB
	assert.False(t, dbWithout.Contains(normalize.PathHash("test.php")))
	assert.False(t, dbWithout.Contains(normalize.PathHash("highlight.php")))
}

func TestDBMerge(t *testing.T) {
	tmp := t.TempDir()
	db1Path := filepath.Join(tmp, "db1.db")
	db2Path := filepath.Join(tmp, "db2.db")
	outPath := filepath.Join(tmp, "merged.db")

	// Create db1 with some hashes
	db1 := hashdb.New()
	db1.Add(100)
	db1.Add(200)
	db1.Add(300)
	require.NoError(t, db1.Save(db1Path))

	// Create db2 with overlapping and new hashes
	db2 := hashdb.New()
	db2.Add(200)
	db2.Add(300)
	db2.Add(400)
	require.NoError(t, db2.Save(db2Path))

	// Merge
	mergeArg := dbMergeArg{Database: outPath}
	mergeArg.Path.Path = []string{db1Path, db2Path}
	require.NoError(t, mergeArg.Execute(nil))

	// Verify merged DB
	merged, err := hashdb.OpenReadOnly(outPath)
	require.NoError(t, err)
	defer merged.Close()

	assert.Equal(t, 4, merged.Len()) // 100, 200, 300, 400 (deduped)
	assert.True(t, merged.Contains(100))
	assert.True(t, merged.Contains(200))
	assert.True(t, merged.Contains(300))
	assert.True(t, merged.Contains(400))
}

func TestDBSaveAndReopen(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "sample.db")

	db := hashdb.New()
	addPath("../../fixture/docroot", db, false, false, true)
	require.NoError(t, db.Save(dbPath))

	// Verify it can be reopened
	loaded, err := hashdb.OpenReadOnly(dbPath)
	require.NoError(t, err)
	defer loaded.Close()
	assert.Greater(t, loaded.Len(), 0)
}

func TestDBInfo(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	// Create a DB with known contents
	db := hashdb.New()
	db.Add(1)
	db.Add(2)
	db.Add(3)
	require.NoError(t, db.Save(dbPath))

	// Verify info command doesn't error
	infoArg := dbInfoArg{}
	infoArg.Path.Path = dbPath
	require.NoError(t, infoArg.Execute(nil))

	// Verify the file has correct size: 16 header + 3*8 data = 40
	fi, err := os.Stat(dbPath)
	require.NoError(t, err)
	assert.Equal(t, int64(40), fi.Size())
}
