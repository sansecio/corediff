package gitindex

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
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

	_, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)

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

	_, err := CloneAndIndex(repoPath, refs, db, IndexOptions{PathPrefix: "vendor/acme/pkg/"})
	require.NoError(t, err)

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

	_, err := CloneAndIndex(repoPath, refs, db, IndexOptions{NoPlatform: true})
	require.NoError(t, err)

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
	_, err := CloneAndIndex(repoPath, map[string]string{"v1": commitHash}, db1, IndexOptions{})
	require.NoError(t, err)

	// With AllValidText, readme.txt should be included
	db2 := hashdb.New()
	_, err = CloneAndIndex(repoPath, map[string]string{"v1": commitHash}, db2, IndexOptions{AllValidText: true})
	require.NoError(t, err)

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
	_, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
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

	_, err = CloneAndIndex(versions[0].Source.URL, refs, db, IndexOptions{AllValidText: true, NoPlatform: true})
	require.NoError(t, err)

	t.Logf("Indexed %d hashes from psr/log", db.Len())
	assert.Greater(t, db.Len(), 10, "expected at least 10 hashes from psr/log")
}

// createTestRepoMultiVersion creates a git repo with multiple commits.
// Each entry in versions is a map of files for that commit.
// Returns repo path and a map of version→commit hash.
func createTestRepoMultiVersion(t *testing.T, versions map[string]map[string]string) (string, map[string]string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	refs := make(map[string]string)
	for version, files := range versions {
		for name, content := range files {
			path := dir + "/" + name
			require.NoError(t, os.MkdirAll(dir+"/"+dirOf(name), 0o755))
			require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
			_, err := wt.Add(name)
			require.NoError(t, err)
		}
		hash, err := wt.Commit("version "+version, &git.CommitOptions{
			Author: &object.Signature{
				Name:  "test",
				Email: "test@test.com",
				When:  time.Now(),
			},
		})
		require.NoError(t, err)
		refs[version] = hash.String()
	}
	return dir, refs
}

func TestCloneAndIndex_BlobDedup(t *testing.T) {
	// Create two versions: v2 changes only one file, the rest are identical.
	versions := map[string]map[string]string{
		"1.0.0": {
			"index.php":      "<?php\necho 'hello';\n",
			"lib/helper.php": "<?php\nfunction foo() { return 1; }\n",
			"lib/utils.php":  "<?php\nfunction bar() { return 2; }\n",
		},
		"2.0.0": {
			"index.php":      "<?php\necho 'hello world';\n", // changed
			"lib/helper.php": "<?php\nfunction foo() { return 1; }\n", // unchanged
			"lib/utils.php":  "<?php\nfunction bar() { return 2; }\n", // unchanged
		},
	}
	repoPath, refs := createTestRepoMultiVersion(t, versions)

	// Index both versions — blob dedup should skip unchanged files
	db := hashdb.New()
	_, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)
	assert.Greater(t, db.Len(), 0)

	// All line hashes from both versions should be present
	for _, h := range normalize.HashLine([]byte("echo 'hello';")) {
		assert.True(t, db.Contains(h))
	}
	for _, h := range normalize.HashLine([]byte("echo 'hello world';")) {
		assert.True(t, db.Contains(h))
	}
	for _, h := range normalize.HashLine([]byte("function foo() { return 1; }")) {
		assert.True(t, db.Contains(h))
	}

	// Verify by comparing against indexing each version separately (no dedup)
	dbSeparate := hashdb.New()
	_, err = CloneAndIndex(repoPath, map[string]string{"1.0.0": refs["1.0.0"]}, dbSeparate, IndexOptions{})
	require.NoError(t, err)
	_, err = CloneAndIndex(repoPath, map[string]string{"2.0.0": refs["2.0.0"]}, dbSeparate, IndexOptions{})
	require.NoError(t, err)

	// Both approaches should produce the same hashes
	assert.Equal(t, dbSeparate.Len(), db.Len(), "blob dedup should produce same result as separate indexing")
}

func TestCloneAndIndex_ReturnsReplaces(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'hello';\n",
		"composer.json": `{
			"name": "magento/magento2ce",
			"replace": {
				"magento/module-catalog": "*",
				"magento/module-checkout": "*"
			}
		}`,
	}
	repoPath, commitHash := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	replaces, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)

	slices.Sort(replaces)
	assert.Equal(t, []string{"magento/module-catalog", "magento/module-checkout"}, replaces)
}

func TestCloneAndIndex_NoComposerJson(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'hello';\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	replaces, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)
	assert.Empty(t, replaces)
}

func TestIsVersionTag(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"2.4.7", true},
		{"v1.0.0", true},
		{"v1.0.0-beta1", true},
		{"0.1.0", true},
		{"v2.4.7-p3", true},
		{"1.0", true},
		{"latest", false},
		{"release-2024", false},
		{"stable", false},
		{"", false},
		{"v", false},
		{"abc", false},
		{"nightly-20240101", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isVersionTag(tt.name), "isVersionTag(%q)", tt.name)
		})
	}
}

func TestRefsFromTags(t *testing.T) {
	// Create a bare repo with lightweight tags
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// Create a file and commit
	require.NoError(t, os.WriteFile(dir+"/index.php", []byte("<?php\n"), 0o644))
	_, err = wt.Add("index.php")
	require.NoError(t, err)

	hash, err := wt.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)

	// Create version tags
	_, err = repo.CreateTag("v1.0.0", hash, nil)
	require.NoError(t, err)
	_, err = repo.CreateTag("v2.0.0", hash, nil)
	require.NoError(t, err)
	// Create non-version tag
	_, err = repo.CreateTag("latest", hash, nil)
	require.NoError(t, err)

	cloneDir := t.TempDir()
	gotRepo, refs, err := RefsFromTags(dir, cloneDir, IndexOptions{})
	require.NoError(t, err)
	require.NotNil(t, gotRepo)

	assert.Contains(t, refs, "v1.0.0")
	assert.Contains(t, refs, "v2.0.0")
	assert.NotContains(t, refs, "latest")
	assert.Equal(t, hash.String(), refs["v1.0.0"])
}

func TestRefsFromTags_AnnotatedTags(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(dir+"/index.php", []byte("<?php\n"), 0o644))
	_, err = wt.Add("index.php")
	require.NoError(t, err)

	commitHash, err := wt.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)

	// Create annotated tag
	_, err = repo.CreateTag("v1.0.0", commitHash, &git.CreateTagOptions{
		Tagger:  &object.Signature{Name: "test", Email: "t@t", When: time.Now()},
		Message: "release v1.0.0",
	})
	require.NoError(t, err)

	cloneDir := t.TempDir()
	_, refs, err := RefsFromTags(dir, cloneDir, IndexOptions{})
	require.NoError(t, err)

	// Annotated tag should resolve to the commit hash, not the tag object hash
	assert.Equal(t, commitHash.String(), refs["v1.0.0"])
}

func TestRefsFromTags_FiltersNonVersionTags(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(dir+"/index.php", []byte("<?php\n"), 0o644))
	_, err = wt.Add("index.php")
	require.NoError(t, err)

	hash, err := wt.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)

	tags := []string{"v1.0.0", "2.4.7", "latest", "release-2024", "stable", "nightly"}
	for _, tag := range tags {
		_, err = repo.CreateTag(tag, hash, nil)
		require.NoError(t, err)
	}

	cloneDir := t.TempDir()
	_, refs, err := RefsFromTags(dir, cloneDir, IndexOptions{})
	require.NoError(t, err)

	assert.Contains(t, refs, "v1.0.0")
	assert.Contains(t, refs, "2.4.7")
	assert.NotContains(t, refs, "latest")
	assert.NotContains(t, refs, "release-2024")
	assert.NotContains(t, refs, "stable")
	assert.NotContains(t, refs, "nightly")
}

func TestIndexRepo(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'hello';\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	refs := map[string]string{"v1.0.0": commitHash}
	db := hashdb.New()

	_, err = IndexRepo(repo, refs, db, IndexOptions{NoPlatform: true})
	require.NoError(t, err)
	assert.Greater(t, db.Len(), 0)
}

func TestCmpVersionDesc(t *testing.T) {
	versions := []string{"1.0.0", "3.255.8", "3.49.0", "3.356.10", "v2.1.0", "3.103.2-p3", "3.103.2"}
	slices.SortFunc(versions, cmpVersionDesc)
	assert.Equal(t, []string{"3.356.10", "3.255.8", "3.103.2-p3", "3.103.2", "3.49.0", "v2.1.0", "1.0.0"}, versions)
}
