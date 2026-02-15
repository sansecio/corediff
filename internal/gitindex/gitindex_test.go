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
	"github.com/go-git/go-git/v5/plumbing"
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

	result, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Greater(t, db.Len(), 0)

	// Path hashes should be present (NoPlatform=false)
	assert.True(t, db.Contains(normalize.PathHash("index.php")))
	assert.True(t, db.Contains(normalize.PathHash("lib/helper.php")))

	// Line hashes for PHP content should be present
	normalize.HashLine([]byte("echo 'hello';"), func(h uint64, _ []byte) bool {
		assert.True(t, db.Contains(h), "missing hash for 'echo hello'")
		return true
	})
}

func TestCloneAndIndex_PathPrefix(t *testing.T) {
	files := map[string]string{
		"src/Foo.php": "<?php\nclass Foo {}\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	result, err := CloneAndIndex(repoPath, refs, db, IndexOptions{PathPrefix: "vendor/acme/pkg/"})
	require.NoError(t, err)
	require.NotNil(t, result)

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

	result, err := CloneAndIndex(repoPath, refs, db, IndexOptions{NoPlatform: true})
	require.NoError(t, err)
	require.NotNil(t, result)

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
	result1, err := CloneAndIndex(repoPath, map[string]string{"v1": commitHash}, db1, IndexOptions{})
	require.NoError(t, err)
	require.NotNil(t, result1)

	// With AllValidText, readme.txt should be included
	db2 := hashdb.New()
	result2, err := CloneAndIndex(repoPath, map[string]string{"v1": commitHash}, db2, IndexOptions{AllValidText: true})
	require.NoError(t, err)
	require.NotNil(t, result2)

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
	normalize.HashLine([]byte("echo 'hello';"), func(h uint64, _ []byte) bool {
		assert.True(t, db.Contains(h))
		return true
	})

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

func TestIndexZip_Cache(t *testing.T) {
	zipData := createTestZip(t, map[string]string{
		"index.php": "<?php\necho 'cached';\n",
	})

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write(zipData)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()

	// First call: downloads and caches
	db := hashdb.New()
	err := IndexZip(srv.URL+"/test.zip", db, IndexOptions{CacheDir: cacheDir})
	require.NoError(t, err)
	assert.Equal(t, 1, hits)
	assert.Greater(t, db.Len(), 0)

	// Second call: should use cache, no extra HTTP request
	db2 := hashdb.New()
	err = IndexZip(srv.URL+"/test.zip", db2, IndexOptions{CacheDir: cacheDir})
	require.NoError(t, err)
	assert.Equal(t, 1, hits) // still 1 — cache hit
	assert.Equal(t, db.Len(), db2.Len())
}

func TestCloneAndIndex_UnreachableRef(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'test';\n",
	}
	repoPath, _ := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v999": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}

	// Should not error, just skip unreachable ref
	result, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
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

	result, err := CloneAndIndex(versions[0].Source.URL, refs, db, IndexOptions{AllValidText: true, NoPlatform: true})
	require.NoError(t, err)
	require.NotNil(t, result)

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
	result, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, db.Len(), 0)

	// All line hashes from both versions should be present
	assertHashed := func(line string) {
		normalize.HashLine([]byte(line), func(h uint64, _ []byte) bool {
			assert.True(t, db.Contains(h))
			return true
		})
	}
	assertHashed("echo 'hello';")
	assertHashed("echo 'hello world';")
	assertHashed("function foo() { return 1; }")

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

	result, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)

	slices.Sort(result.Replaces)
	assert.Equal(t, []string{"magento/module-catalog", "magento/module-checkout"}, result.Replaces)
}

func TestCloneAndIndex_NoComposerJson(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'hello';\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	result, err := CloneAndIndex(repoPath, refs, db, IndexOptions{})
	require.NoError(t, err)
	assert.Empty(t, result.Replaces)
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

	result, err := IndexRepo(repo, refs, db, IndexOptions{NoPlatform: true})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, db.Len(), 0)
}

func TestFindSubPackages(t *testing.T) {
	files := map[string]string{
		"composer.json": `{"name": "magento/magento2ce", "version": "2.4.7"}`,
		"index.php":     "<?php\n",
		"app/code/Magento/Catalog/composer.json":  `{"name": "magento/module-catalog", "version": "104.0.7"}`,
		"app/code/Magento/Catalog/Block/Product.php": "<?php\nclass Product {}\n",
		"app/code/Magento/Sales/composer.json":    `{"name": "magento/module-sales", "version": "103.0.7"}`,
		"app/code/Magento/Sales/Model/Order.php":  "<?php\nclass Order {}\n",
		"lib/internal/README.md":                  "no composer.json here",
	}
	repoPath, commitHash := createTestRepo(t, files)

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	commit, err := repo.CommitObject(plumbing.NewHash(commitHash))
	require.NoError(t, err)
	tree, err := commit.Tree()
	require.NoError(t, err)

	subPkgs := findSubPackages(tree)

	assert.Len(t, subPkgs, 2)

	byName := make(map[string]subPackage)
	for _, sp := range subPkgs {
		byName[sp.Name] = sp
	}

	catalog := byName["magento/module-catalog"]
	assert.Equal(t, "104.0.7", catalog.Version)
	assert.Equal(t, "app/code/Magento/Catalog/", catalog.Dir)

	sales := byName["magento/module-sales"]
	assert.Equal(t, "103.0.7", sales.Version)
	assert.Equal(t, "app/code/Magento/Sales/", sales.Dir)
}

func TestResolveStoredPath(t *testing.T) {
	subPkgs := []subPackage{
		{Name: "magento/module-catalog", Dir: "app/code/Magento/Catalog/"},
		{Name: "magento/module-sales", Dir: "app/code/Magento/Sales/"},
	}

	tests := []struct {
		filePath string
		want     string
	}{
		{"app/code/Magento/Catalog/Block/Product.php", "vendor/magento/module-catalog/Block/Product.php"},
		{"app/code/Magento/Sales/Model/Order.php", "vendor/magento/module-sales/Model/Order.php"},
		{"index.php", "vendor/magento/magento2ce/index.php"},
		{"pub/index.php", "vendor/magento/magento2ce/pub/index.php"},
	}
	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			got := resolveStoredPath(tt.filePath, subPkgs, "vendor/magento/magento2ce/")
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIndexRef_SubPackagePaths(t *testing.T) {
	// Verify that files inside sub-packages get stored with canonical vendor paths
	files := map[string]string{
		"composer.json": `{"name": "magento/magento2ce", "version": "2.4.7"}`,
		"index.php":     "<?php\necho 'root';\n",
		"app/code/Magento/Catalog/composer.json":     `{"name": "magento/module-catalog", "version": "104.0.7"}`,
		"app/code/Magento/Catalog/Block/Product.php": "<?php\nclass Product {}\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	db := hashdb.New()
	opts := IndexOptions{PathPrefix: "vendor/magento/magento2ce/"}
	_, _, err = indexRef(repo, "2.4.7", commitHash, db, opts, nil)
	require.NoError(t, err)

	// Root file should be stored under the root package prefix
	assert.True(t, db.Contains(normalize.PathHash("vendor/magento/magento2ce/index.php")),
		"root file should use root package prefix")

	// Sub-package file should be stored under canonical vendor path
	assert.True(t, db.Contains(normalize.PathHash("vendor/magento/module-catalog/Block/Product.php")),
		"sub-package file should use canonical vendor path")

	// Should NOT be stored under the monorepo path
	assert.False(t, db.Contains(normalize.PathHash("vendor/magento/magento2ce/app/code/Magento/Catalog/Block/Product.php")),
		"sub-package file should NOT use monorepo path")
}

func TestIndexRef_SubPackageCallback(t *testing.T) {
	files := map[string]string{
		"composer.json": `{"name": "magento/magento2ce", "version": "2.4.7"}`,
		"app/code/Magento/Catalog/composer.json": `{"name": "magento/module-catalog", "version": "104.0.7"}`,
		"app/code/Magento/Catalog/Block/Product.php": "<?php\nclass Product {}\n",
	}
	repoPath, commitHash := createTestRepo(t, files)

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	db := hashdb.New()
	var recorded []string
	opts := IndexOptions{
		PathPrefix: "vendor/magento/magento2ce/",
		OnSubPackage: func(name, version string) {
			recorded = append(recorded, name+"@"+version)
		},
	}

	_, _, err = indexRef(repo, "2.4.7", commitHash, db, opts, nil)
	require.NoError(t, err)

	assert.Contains(t, recorded, "magento/module-catalog@104.0.7")
}

func TestIndexRefs_CollectsLockDeps(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'hello';\n",
		"composer.json": `{"name": "myvendor/myapp"}`,
		"composer.lock": `{
			"packages": [
				{"name": "monolog/monolog", "type": "library", "version": "3.5.0",
				 "source": {"type": "git", "url": "https://github.com/Seldaek/monolog.git", "reference": "aaa"}},
				{"name": "psr/log", "type": "library", "version": "3.0.0"},
				{"name": "php", "type": ""},
				{"name": "ext-json", "type": ""},
				{"name": "vendor/meta", "type": "metapackage"}
			]
		}`,
	}
	repoPath, commitHash := createTestRepo(t, files)

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	result := indexRefs(repo, refs, db, IndexOptions{CollectLockDeps: true})

	// Should have monolog and psr/log, not php/ext-json/metapackage
	depNames := make(map[string]bool)
	for _, dep := range result.LockDeps {
		depNames[dep.Name] = true
	}
	assert.True(t, depNames["monolog/monolog"], "should contain monolog/monolog")
	assert.True(t, depNames["psr/log"], "should contain psr/log")
	assert.False(t, depNames["php"], "should not contain php")
	assert.False(t, depNames["ext-json"], "should not contain ext-json")
	assert.False(t, depNames["vendor/meta"], "should not contain metapackage")

	// Verify source info is preserved
	for _, dep := range result.LockDeps {
		if dep.Name == "monolog/monolog" {
			assert.Equal(t, "3.5.0", dep.Version)
			assert.Equal(t, "git", dep.Source.Type)
			assert.Equal(t, "https://github.com/Seldaek/monolog.git", dep.Source.URL)
		}
	}
}

func TestIndexRefs_LockDepsExcludesReplaced(t *testing.T) {
	files := map[string]string{
		"index.php": "<?php\necho 'hello';\n",
		"composer.json": `{
			"name": "magento/magento2ce",
			"replace": {"magento/module-catalog": "*"}
		}`,
		"composer.lock": `{
			"packages": [
				{"name": "magento/module-catalog", "type": "library", "version": "104.0.7"},
				{"name": "monolog/monolog", "type": "library", "version": "3.5.0"}
			]
		}`,
	}
	repoPath, commitHash := createTestRepo(t, files)

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	result := indexRefs(repo, refs, db, IndexOptions{CollectLockDeps: true})

	depNames := make(map[string]bool)
	for _, dep := range result.LockDeps {
		depNames[dep.Name] = true
	}
	assert.False(t, depNames["magento/module-catalog"], "replaced package should be excluded from lock deps")
	assert.True(t, depNames["monolog/monolog"], "non-replaced package should be included")
}

func TestIndexRefs_LockDepsDedup(t *testing.T) {
	// Two versions with overlapping lock deps → each dep appears only once.
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// v1.0.0 with monolog 3.5.0 and psr/log 3.0.0
	files1 := map[string]string{
		"index.php":     "<?php\necho 'v1';\n",
		"composer.json": `{"name": "myvendor/myapp"}`,
		"composer.lock": `{"packages": [
			{"name": "monolog/monolog", "type": "library", "version": "3.5.0"},
			{"name": "psr/log", "type": "library", "version": "3.0.0"}
		]}`,
	}
	for name, content := range files1 {
		path := dir + "/" + name
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		_, err = wt.Add(name)
		require.NoError(t, err)
	}
	h1, err := wt.Commit("v1", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)

	// v2.0.0 with monolog 3.5.0 (same) and psr/log 3.0.1 (different)
	files2 := map[string]string{
		"index.php": "<?php\necho 'v2';\n",
		"composer.lock": `{"packages": [
			{"name": "monolog/monolog", "type": "library", "version": "3.5.0"},
			{"name": "psr/log", "type": "library", "version": "3.0.1"}
		]}`,
	}
	for name, content := range files2 {
		path := dir + "/" + name
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		_, err = wt.Add(name)
		require.NoError(t, err)
	}
	h2, err := wt.Commit("v2", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)

	db := hashdb.New()
	refs := map[string]string{
		"v1.0.0": h1.String(),
		"v2.0.0": h2.String(),
	}

	result := indexRefs(repo, refs, db, IndexOptions{CollectLockDeps: true})

	// Count occurrences of each dep
	depCounts := make(map[string]int)
	for _, dep := range result.LockDeps {
		depCounts[dep.Name+"@"+dep.Version]++
	}

	// monolog 3.5.0 should appear once (deduped across versions)
	assert.Equal(t, 1, depCounts["monolog/monolog@3.5.0"])
	// psr/log should have both versions
	assert.Equal(t, 1, depCounts["psr/log@3.0.0"])
	assert.Equal(t, 1, depCounts["psr/log@3.0.1"])
}

func TestIndexRefs_NoLockDepsWhenDisabled(t *testing.T) {
	files := map[string]string{
		"index.php":     "<?php\necho 'hello';\n",
		"composer.json": `{"name": "myvendor/myapp"}`,
		"composer.lock": `{"packages": [
			{"name": "monolog/monolog", "type": "library", "version": "3.5.0"}
		]}`,
	}
	repoPath, commitHash := createTestRepo(t, files)

	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	db := hashdb.New()
	refs := map[string]string{"v1.0.0": commitHash}

	// CollectLockDeps=false (default)
	result := indexRefs(repo, refs, db, IndexOptions{})
	assert.Empty(t, result.LockDeps, "LockDeps should be empty when CollectLockDeps is false")
}

func TestCmpVersionDesc(t *testing.T) {
	versions := []string{"1.0.0", "3.255.8", "3.49.0", "3.356.10", "v2.1.0", "3.103.2-p3", "3.103.2"}
	slices.SortFunc(versions, cmpVersionDesc)
	assert.Equal(t, []string{"3.356.10", "3.255.8", "3.103.2-p3", "3.103.2", "3.49.0", "v2.1.0", "1.0.0"}, versions)
}
