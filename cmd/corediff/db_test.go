package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sansecio/corediff/internal/hashdb"
	"github.com/sansecio/corediff/internal/indexer"
	"github.com/sansecio/corediff/internal/manifest"
	"github.com/sansecio/corediff/internal/normalize"
	"github.com/sansecio/corediff/internal/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBAdd(t *testing.T) {
	db := hashdb.New()
	addPath("../../fixture/docroot", db, false, false, platform.Magento2)

	// Should have line hashes
	assert.Greater(t, db.Len(), 0)

	// Should also have path hashes (ignorePaths=false, platform!=nil)
	// Check that at least one known path hash is present
	assert.True(t, db.Contains(normalize.PathHash("test.php")))
	assert.True(t, db.Contains(normalize.PathHash("highlight.php")))
}

func TestDBAddNoPlatform(t *testing.T) {
	dbWith := hashdb.New()
	addPath("../../fixture/docroot", dbWith, false, false, platform.Magento2)

	dbWithout := hashdb.New()
	addPath("../../fixture/docroot", dbWithout, false, false, nil)

	// plat=nil should have fewer hashes (no path hashes)
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
	dbCommand.Database = outPath
	mergeArg := dbMergeArg{}
	mergeArg.Path.Path = []string{db1Path, db2Path}
	require.NoError(t, mergeArg.Execute(nil))

	// Verify merged DB
	merged, err := hashdb.Open(outPath)
	require.NoError(t, err)


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
	addPath("../../fixture/docroot", db, false, false, nil)
	require.NoError(t, db.Save(dbPath))

	// Verify it can be reopened
	loaded, err := hashdb.Open(dbPath)
	require.NoError(t, err)

	assert.Greater(t, loaded.Len(), 0)
}

func TestDBAdd_PackagistValidation(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	dbCommand.Database = dbPath

	t.Run("packagist without args", func(t *testing.T) {
		arg := dbIndexArg{Packagist: true}
		err := arg.Execute(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--packagist requires at least one package name")
	})

	t.Run("composer+packagist", func(t *testing.T) {
		arg := dbIndexArg{Packagist: true, Composer: "/some/composer.json"}
		arg.Path.Path = []string{"vendor/pkg"}
		err := arg.Execute(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot combine")
	})

	t.Run("composer+path", func(t *testing.T) {
		arg := dbIndexArg{Composer: "/some/composer.json"}
		arg.Path.Path = []string{"/some/path"}
		err := arg.Execute(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot combine")
	})

	t.Run("neither provided", func(t *testing.T) {
		arg := dbIndexArg{}
		err := arg.Execute(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provide --packagist, --composer, --update, or at least one <path>")
	})

	t.Run("composer missing file", func(t *testing.T) {
		arg := dbIndexArg{Composer: filepath.Join(tmp, "nonexistent.json")}
		err := arg.Execute(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "composer.json")
	})
}

func TestBuildHTTPClient_AppliesAuth(t *testing.T) {
	// Set up auth.json in a temp dir so FindAuthConfig finds it
	tmp := t.TempDir()
	composerDir := filepath.Join(tmp, ".composer")
	require.NoError(t, os.MkdirAll(composerDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(composerDir, "auth.json"), []byte(`{
		"http-basic": {"repo.magento.com": {"username": "user", "password": "pass"}}
	}`), 0o644))

	// Run from the temp dir so FindAuthConfig walks up and finds it
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(origDir) })

	arg := dbIndexArg{}
	opts := indexer.IndexOptions{}
	client, err := arg.buildHTTPClient(&opts)
	require.NoError(t, err)
	require.NotNil(t, client, "buildHTTPClient should return non-nil client when auth is found")
	assert.Same(t, client, opts.HTTP, "opts.HTTP should be set to the returned client")

	// Verify auth header is applied for repo.magento.com
	req, _ := http.NewRequest("GET", "https://repo.magento.com/archives/test.zip", nil)
	client.Transport.RoundTrip(req) //nolint: we just want to check the header mutation
	assert.NotEmpty(t, req.Header.Get("Authorization"), "auth header should be set for repo.magento.com")
	assert.Contains(t, req.Header.Get("Authorization"), "Basic ")
}

func TestDBAdd_ParsePackagistVersion(t *testing.T) {
	tests := []struct {
		input   string
		wantPkg string
		wantPin string
	}{
		{"psr/log", "psr/log", ""},
		{"psr/log:3.0.0", "psr/log", "3.0.0"},
		{"psr/log@3.0.0", "psr/log", "3.0.0"},
		{"magento/framework:103.0.7-p3", "magento/framework", "103.0.7-p3"},
		{"magento/framework@103.0.7-p3", "magento/framework", "103.0.7-p3"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := tt.input
			var pin string
			if idx := strings.LastIndexAny(pkg, ":@"); idx > 0 {
				pkg, pin = pkg[:idx], pkg[idx+1:]
			}
			assert.Equal(t, tt.wantPkg, pkg)
			assert.Equal(t, tt.wantPin, pin)
		})
	}
}

func TestDBAdd_DatabaseOnParent(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	dbCommand.Database = dbPath

	arg := dbIndexArg{NoPlatform: true}
	arg.Path.Path = []string{"../../fixture/docroot"}
	require.NoError(t, arg.Execute(nil))

	// Verify database was created using parent's Database path
	db, err := hashdb.Open(dbPath)
	require.NoError(t, err)

	assert.Greater(t, db.Len(), 0)
}

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://github.com/magento/magento2.git", true},
		{"http://github.com/magento/magento2.git", true},
		{"git://github.com/magento/magento2.git", true},
		{"git@github.com:magento/magento2.git", true},
		{"ssh://git@github.com/magento/magento2.git", true},
		{"/some/local/path", false},
		{"./relative/path", false},
		{"relative/path", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isGitURL(tt.input))
		})
	}
}

func TestUpdateGitURLEntry(t *testing.T) {
	// Create a git repo with two version tags
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// v1.0.0
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.php"), []byte("<?php\necho 'v1';\n"), 0o644))
	_, err = wt.Add("index.php")
	require.NoError(t, err)
	h1, err := wt.Commit("v1", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)
	_, err = repo.CreateTag("v1.0.0", h1, nil)
	require.NoError(t, err)

	// v2.0.0
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.php"), []byte("<?php\necho 'v2';\n"), 0o644))
	_, err = wt.Add("index.php")
	require.NoError(t, err)
	h2, err := wt.Commit("v2", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)
	_, err = repo.CreateTag("v2.0.0", h2, nil)
	require.NoError(t, err)

	// Set up manifest with v1.0.0 already indexed
	tmp := t.TempDir()
	mfPath := filepath.Join(tmp, "test.manifest")
	require.NoError(t, os.WriteFile(mfPath, []byte(dir+"@v1.0.0\n"), 0o644))

	mf, err := manifest.Load(mfPath)
	require.NoError(t, err)
	defer mf.Close()

	db := hashdb.New()
	opts := indexer.IndexOptions{NoPlatform: true}

	arg := &dbIndexArg{NoPlatform: true}
	arg.updateGitURLEntry(dir, db, mf, opts)

	// Should have indexed v2.0.0 but not re-indexed v1.0.0
	assert.True(t, mf.IsIndexed(dir, "v2.0.0"))
	assert.True(t, mf.IsIndexed(dir, "v1.0.0"))
	assert.Greater(t, db.Len(), 0)
}

func TestUpdateGitURLEntry_WritesSubPackagesToManifest(t *testing.T) {
	// Create a monorepo with sub-package composer.json files (like magento2)
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "app/code/Magento/Catalog"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "app/code/Magento/Sales"), 0o755))

	files := map[string]string{
		"composer.json": `{
			"name": "magento/magento2ce",
			"replace": {
				"magento/module-catalog": "*",
				"magento/module-sales": "*"
			}
		}`,
		"index.php": "<?php\necho 'hello';\n",
		"app/code/Magento/Catalog/composer.json":     `{"name": "magento/module-catalog", "version": "104.0.7"}`,
		"app/code/Magento/Catalog/Block/Product.php": "<?php\nclass Product {}\n",
		"app/code/Magento/Sales/composer.json":       `{"name": "magento/module-sales", "version": "103.0.7"}`,
		"app/code/Magento/Sales/Model/Order.php":     "<?php\nclass Order {}\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		_, err = wt.Add(name)
		require.NoError(t, err)
	}

	h1, err := wt.Commit("v1", &git.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t", When: time.Now()},
	})
	require.NoError(t, err)
	_, err = repo.CreateTag("v1.0.0", h1, nil)
	require.NoError(t, err)

	// Set up empty manifest
	tmp := t.TempDir()
	mfPath := filepath.Join(tmp, "test.manifest")
	mf, err := manifest.Load(mfPath)
	require.NoError(t, err)
	defer mf.Close()

	db := hashdb.New()
	opts := indexer.IndexOptions{}

	arg := &dbIndexArg{}
	arg.updateGitURLEntry(dir, db, mf, opts)

	// Verify monorepo version was indexed
	assert.True(t, mf.IsIndexed(dir, "v1.0.0"))

	// Sub-packages should NOT have individual @version entries — replace: covers them
	assert.False(t, mf.IsIndexed("magento/module-catalog", "104.0.7"),
		"replaced sub-packages should not have @version manifest entries")
	assert.False(t, mf.IsIndexed("magento/module-sales", "103.0.7"),
		"replaced sub-packages should not have @version manifest entries")

	// Verify path hashes use canonical vendor paths
	assert.True(t, db.Contains(normalize.PathHash("vendor/magento/module-catalog/Block/Product.php")),
		"sub-package file should use canonical vendor path")
	assert.True(t, db.Contains(normalize.PathHash("vendor/magento/module-sales/Model/Order.php")),
		"sub-package file should use canonical vendor path")

	// Replace entries should also be in manifest
	assert.True(t, mf.IsReplaced("magento/module-catalog"))
	assert.True(t, mf.IsReplaced("magento/module-sales"))
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
	dbCommand.Database = dbPath
	infoArg := dbInfoArg{}
	require.NoError(t, infoArg.Execute(nil))

	// Verify the file has correct size: 16 header + 3*8 data = 40
	fi, err := os.Stat(dbPath)
	require.NoError(t, err)
	assert.Equal(t, int64(40), fi.Size())
}
