package gitindex

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
	"github.com/gwillem/corediff/internal/packagist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRepo creates a non-bare git repo with a single commit containing
// the given files. Returns repo path and the commit hash string.
func createTestRepo(t *testing.T, files map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	for name, content := range files {
		path := dir + "/" + name
		require.NoError(t, os.MkdirAll(dir+"/"+dirOf(name), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		_, err := wt.Add(name)
		require.NoError(t, err)
	}

	hash, err := wt.Commit("test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	return dir, hash.String()
}

// dirOf returns the directory portion of a path, or "." for top-level files.
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}

func TestCloneAndIndex(t *testing.T) {
	files := map[string]string{
		"index.php":       "<?php\necho 'hello';\n",
		"lib/helper.php":  "<?php\nfunction foo() { return 1; }\n",
		"readme.txt":      "This is not a code file",
		"data/binary.dat": "\x00\x01\x02\x03",
	}
	repoPath, commitHash := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)

	db.Compact()
	assert.Greater(t, db.Len(), 0)

	// Path hashes should be present (NoPlatform=false)
	assert.True(t, db.Contains(normalize.PathHash("index.php")))
	assert.True(t, db.Contains(normalize.PathHash("lib/helper.php")))

	// Line hashes for PHP content should be present
	for _, h := range normalize.HashLine([]byte("echo 'hello';")) {
		assert.True(t, db.Contains(h), "missing hash for 'echo hello'")
	}
}

func TestCloneAndIndex_PathPrefix(t *testing.T) {
	files := map[string]string{
		"src/Foo.php": "<?php\nclass Foo {}\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	err := CloneAndIndex(repoPath, refs, db, IndexOptions{PathPrefix: "vendor/acme/pkg/"})
	require.NoError(t, err)

	db.Compact()
	// Path hash should use the prefix
	assert.True(t, db.Contains(normalize.PathHash("vendor/acme/pkg/src/Foo.php")))
	// Bare path should NOT be present
	assert.False(t, db.Contains(normalize.PathHash("src/Foo.php")))
}

func TestCloneAndIndex_NoPlatform(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'test';\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	err := CloneAndIndex(repoPath, refs, db, IndexOptions{NoPlatform: true})
	require.NoError(t, err)

	db.Compact()
	assert.Greater(t, db.Len(), 0)

	// With NoPlatform, path hashes should NOT be present
	assert.False(t, db.Contains(normalize.PathHash("index.php")))
}

func TestCloneAndIndex_AllValidText(t *testing.T) {
	files := map[string]string{
		"index.php":  "<?php\necho 'hello';\n",
		"readme.txt": "line one\nline two\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	// Without AllValidText, readme.txt should be skipped
	db1 := hashdb.New()
	err := CloneAndIndex(repoPath, map[string]string{"v1": commitHash}, db1, IndexOptions{})
	require.NoError(t, err)
	db1.Compact()

	// With AllValidText, readme.txt should be included
	db2 := hashdb.New()
	err = CloneAndIndex(repoPath, map[string]string{"v1": commitHash}, db2, IndexOptions{AllValidText: true})
	require.NoError(t, err)
	db2.Compact()

	assert.Greater(t, db2.Len(), db1.Len())
}

func createTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		// GitHub-style root prefix
		f, err := w.Create("repo-abc123/" + name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestIndexZip(t *testing.T) {
	zipData := createTestZip(t, map[string]string{
		"index.php":      "<?php\necho 'hello';\n",
		"lib/helper.php": "<?php\nfunction bar() { return 2; }\n",
		"readme.txt":     "Not a code file",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipData)
	}))
	defer srv.Close()

	db := hashdb.New()
	err := IndexZip(srv.URL+"/test.zip", db, IndexOptions{})
	require.NoError(t, err)

	db.Compact()
	assert.Greater(t, db.Len(), 0)

	// Path hashes should be present (without the root prefix)
	assert.True(t, db.Contains(normalize.PathHash("index.php")))
	assert.True(t, db.Contains(normalize.PathHash("lib/helper.php")))

	// Line hashes should be present
	for _, h := range normalize.HashLine([]byte("echo 'hello';")) {
		assert.True(t, db.Contains(h))
	}

	// readme.txt should be skipped (no valid ext, AllValidText=false)
	assert.False(t, db.Contains(normalize.PathHash("readme.txt")))
}

func TestIndexZip_NoPlatform(t *testing.T) {
	zipData := createTestZip(t, map[string]string{
		"index.php": "<?php\necho 'test';\n",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(zipData)
	}))
	defer srv.Close()

	db := hashdb.New()
	err := IndexZip(srv.URL+"/test.zip", db, IndexOptions{NoPlatform: true})
	require.NoError(t, err)

	db.Compact()
	assert.Greater(t, db.Len(), 0)
	assert.False(t, db.Contains(normalize.PathHash("index.php")))
}

func TestCloneAndIndex_UnreachableRef(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'test';\n",
	}
	repoPath, _ := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v999": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}

	// Should not error, just skip unreachable ref
	err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, db.Len())
}

func TestIntegration_PsrLog(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	// Fetch versions from Packagist
	c := &packagist.Client{}
	versions, err := c.Versions("psr/log")
	require.NoError(t, err)
	require.Greater(t, len(versions), 0)

	// Use only the first 2 versions to keep it fast
	refs := make(map[string]string)
	for i, v := range versions {
		if i >= 2 {
			break
		}
		if v.Source.Reference != "" {
			refs[v.Version] = v.Source.Reference
		}
	}

	db := hashdb.New()
	require.Equal(t, "git", versions[0].Source.Type)

	err = CloneAndIndex(versions[0].Source.URL, refs, db, IndexOptions{AllValidText: true, NoPlatform: true})
	require.NoError(t, err)

	db.Compact()
	t.Logf("Indexed %d hashes from psr/log", db.Len())
	assert.Greater(t, db.Len(), 10, "expected at least 10 hashes from psr/log")
}
