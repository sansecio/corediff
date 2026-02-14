package manifest

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestPath(t *testing.T) {
	tests := []struct {
		dbPath   string
		expected string
	}{
		{"corediff.db", "corediff.manifest"},
		{"/path/to/data.db", "/path/to/data.manifest"},
		{"nodb", "nodb.manifest"},
		{"foo.bar.db", "foo.bar.manifest"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, PathFromDB(tt.dbPath))
	}
}

func TestLoadCreatesNew(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	m, err := Load(path)
	require.NoError(t, err)
	defer m.Close()

	assert.False(t, m.IsIndexed("vendor/pkg", "1.0.0"))
	assert.Empty(t, m.Packages())
}

func TestLoadExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	err := os.WriteFile(path, []byte("vendor/pkg@1.0.0\nvendor/pkg@2.0.0\n"), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)
	defer m.Close()

	assert.True(t, m.IsIndexed("vendor/pkg", "1.0.0"))
	assert.True(t, m.IsIndexed("vendor/pkg", "2.0.0"))
	assert.False(t, m.IsIndexed("vendor/pkg", "3.0.0"))
	assert.False(t, m.IsIndexed("other/pkg", "1.0.0"))
}

func TestMarkIndexed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	m, err := Load(path)
	require.NoError(t, err)

	assert.False(t, m.IsIndexed("vendor/pkg", "1.0.0"))

	err = m.MarkIndexed("vendor/pkg", "1.0.0")
	require.NoError(t, err)

	assert.True(t, m.IsIndexed("vendor/pkg", "1.0.0"))

	// Mark same entry again — should be idempotent
	err = m.MarkIndexed("vendor/pkg", "1.0.0")
	require.NoError(t, err)

	m.Close()

	// Verify persistence: reload and check
	m2, err := Load(path)
	require.NoError(t, err)
	defer m2.Close()

	assert.True(t, m2.IsIndexed("vendor/pkg", "1.0.0"))
}

func TestPackages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	content := "vendor/a@1.0.0\nvendor/a@2.0.0\nvendor/b@1.0.0\n"
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)
	defer m.Close()

	pkgs := m.Packages()
	assert.Len(t, pkgs, 2)
	assert.Contains(t, pkgs, "vendor/a")
	assert.Contains(t, pkgs, "vendor/b")
}

func TestConcurrentMarkIndexed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	m, err := Load(path)
	require.NoError(t, err)
	defer m.Close()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			pkg := "vendor/pkg"
			version := "1.0." + string(rune('0'+i%10))
			_ = m.MarkIndexed(pkg, version)
		})
	}
	wg.Wait()

	// All entries should be present
	for i := range 10 {
		version := "1.0." + string(rune('0'+i))
		assert.True(t, m.IsIndexed("vendor/pkg", version))
	}
}

func TestLoadSkipsEmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	content := "vendor/a@1.0.0\n\n\nvendor/b@2.0.0\n"
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)
	defer m.Close()

	assert.True(t, m.IsIndexed("vendor/a", "1.0.0"))
	assert.True(t, m.IsIndexed("vendor/b", "2.0.0"))
}

func TestLoadSkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	content := "vendor/a@1.0.0\nno-at-sign\nvendor/b@2.0.0\n"
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)
	defer m.Close()

	assert.True(t, m.IsIndexed("vendor/a", "1.0.0"))
	assert.True(t, m.IsIndexed("vendor/b", "2.0.0"))
}

func TestMarkReplaced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	m, err := Load(path)
	require.NoError(t, err)

	assert.False(t, m.IsReplaced("magento/module-catalog"))

	err = m.MarkReplaced("magento/module-catalog")
	require.NoError(t, err)

	assert.True(t, m.IsReplaced("magento/module-catalog"))
	assert.False(t, m.IsReplaced("magento/module-sales"))

	// Idempotent
	err = m.MarkReplaced("magento/module-catalog")
	require.NoError(t, err)

	m.Close()

	// Verify persistence: reload and check
	m2, err := Load(path)
	require.NoError(t, err)
	defer m2.Close()

	assert.True(t, m2.IsReplaced("magento/module-catalog"))
	assert.False(t, m2.IsReplaced("magento/module-sales"))
}

func TestLoadWithReplaceEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	content := "vendor/a@1.0.0\nreplace:magento/module-catalog\nvendor/b@2.0.0\nreplace:magento/module-sales\n"
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := Load(path)
	require.NoError(t, err)
	defer m.Close()

	// Indexed entries still work
	assert.True(t, m.IsIndexed("vendor/a", "1.0.0"))
	assert.True(t, m.IsIndexed("vendor/b", "2.0.0"))

	// Replace entries are loaded
	assert.True(t, m.IsReplaced("magento/module-catalog"))
	assert.True(t, m.IsReplaced("magento/module-sales"))

	// Packages() should not include replace entries
	pkgs := m.Packages()
	assert.Contains(t, pkgs, "vendor/a")
	assert.Contains(t, pkgs, "vendor/b")
	assert.NotContains(t, pkgs, "magento/module-catalog")
	assert.NotContains(t, pkgs, "magento/module-sales")
}

func TestReplacedIdempotentFileWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	m, err := Load(path)
	require.NoError(t, err)

	require.NoError(t, m.MarkReplaced("magento/module-catalog"))
	require.NoError(t, m.MarkReplaced("magento/module-catalog")) // duplicate
	m.Close()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "replace:magento/module-catalog\n", string(data))
}

func TestIdempotentFileWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest")

	m, err := Load(path)
	require.NoError(t, err)

	require.NoError(t, m.MarkIndexed("vendor/pkg", "1.0.0"))
	require.NoError(t, m.MarkIndexed("vendor/pkg", "1.0.0")) // duplicate
	m.Close()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// Should only have one line (no duplicates written)
	assert.Equal(t, "vendor/pkg@1.0.0\n", string(data))
}
