package hashdb

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContains(t *testing.T) {
	db := New()
	db.Add(42)
	db.Add(100)
	db.Add(1)

	// Before compact: linear scan on buf
	assert.True(t, db.Contains(42))
	assert.True(t, db.Contains(100))
	assert.True(t, db.Contains(1))
	assert.False(t, db.Contains(99))

	// After compact: binary search on main
	db.Compact()
	assert.True(t, db.Contains(42))
	assert.True(t, db.Contains(100))
	assert.True(t, db.Contains(1))
	assert.False(t, db.Contains(99))
}

func TestCompact(t *testing.T) {
	db := New()
	db.Add(5)
	db.Add(3)
	db.Add(5) // duplicate
	db.Add(1)
	db.Add(3) // duplicate

	db.Compact()
	assert.Equal(t, 3, db.Len())
	assert.True(t, db.Contains(1))
	assert.True(t, db.Contains(3))
	assert.True(t, db.Contains(5))
}

func TestCompactEmpty(t *testing.T) {
	db := New()
	db.Compact() // should not panic
	assert.Equal(t, 0, db.Len())
}

func TestMerge(t *testing.T) {
	db1 := New()
	db1.Add(1)
	db1.Add(2)
	db1.Compact()

	db2 := New()
	db2.Add(2) // overlap
	db2.Add(3)
	db2.Compact()

	db1.Merge(db2)
	db1.Compact()
	assert.Equal(t, 3, db1.Len()) // 1, 2, 3 deduped
	assert.True(t, db1.Contains(1))
	assert.True(t, db1.Contains(2))
	assert.True(t, db1.Contains(3))
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db := New()
	db.Add(42)
	db.Add(100)
	db.Add(1)
	require.NoError(t, db.Save(path))

	loaded, err := OpenReadOnly(path)
	require.NoError(t, err)
	assert.Equal(t, 3, loaded.Len())
	assert.True(t, loaded.Contains(42))
	assert.True(t, loaded.Contains(100))
	assert.True(t, loaded.Contains(1))
	assert.False(t, loaded.Contains(99))
}

func TestEmptyDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.db")

	db := New()
	require.NoError(t, db.Save(path))

	loaded, err := OpenReadOnly(path)
	require.NoError(t, err)
	assert.Equal(t, 0, loaded.Len())
	assert.False(t, loaded.Contains(42))
}

func TestDeduplication(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db := New()
	for i := 0; i < 100; i++ {
		db.Add(42)
	}
	require.NoError(t, db.Save(path))

	loaded, err := OpenReadOnly(path)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Len())
}

func TestReadOnlyPanics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db := New()
	db.Add(1)
	require.NoError(t, db.Save(path))

	ro, err := OpenReadOnly(path)
	require.NoError(t, err)
	assert.Panics(t, func() { ro.Add(42) })
}

func TestOpenReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db := New()
	db.Add(1)
	db.Add(2)
	require.NoError(t, db.Save(path))

	rw, err := OpenReadWrite(path)
	require.NoError(t, err)
	rw.Add(3) // should not panic
	rw.Compact()
	assert.Equal(t, 3, rw.Len())
}

func TestCorruptFile(t *testing.T) {
	dir := t.TempDir()

	// File too small for header
	path := filepath.Join(dir, "corrupt.db")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))
	_, err := OpenReadOnly(path)
	assert.Error(t, err)

	// File with bad magic
	path2 := filepath.Join(dir, "badmagic.db")
	require.NoError(t, os.WriteFile(path2, make([]byte, 24), 0644))
	_, err = OpenReadOnly(path2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad magic")
}

func TestVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "future.db")

	f, err := os.Create(path)
	require.NoError(t, err)
	hdr := dbHeader{Version: 99, Count: 0}
	copy(hdr.Magic[:], dbMagic)
	require.NoError(t, binary.Write(f, binary.LittleEndian, &hdr))
	require.NoError(t, f.Close())

	_, err = OpenReadOnly(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database version")
}

func TestFileNotFound(t *testing.T) {
	_, err := OpenReadOnly("/nonexistent/path.db")
	assert.Error(t, err)
}

func TestMmapMatchesReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db := New()
	for i := uint64(0); i < 1000; i++ {
		db.Add(i * 7)
	}
	require.NoError(t, db.Save(path))

	ro, err := OpenReadOnly(path)
	require.NoError(t, err)
	defer ro.Close()

	rw, err := OpenReadWrite(path)
	require.NoError(t, err)

	assert.Equal(t, rw.Len(), ro.Len())
	for i := uint64(0); i < 1000; i++ {
		h := i * 7
		assert.Equal(t, rw.Contains(h), ro.Contains(h), "mismatch for %d", h)
	}
	// Check some values not in the DB
	assert.False(t, ro.Contains(1))
	assert.False(t, ro.Contains(9999))
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db := New()
	db.Add(1)
	require.NoError(t, db.Save(path))

	ro, err := OpenReadOnly(path)
	require.NoError(t, err)
	assert.True(t, ro.Contains(1))
	require.NoError(t, ro.Close())

	// Close on non-mmap DB is a no-op
	rw := New()
	require.NoError(t, rw.Close())
}

func createBenchDB(b *testing.B, n int) string {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.db")
	db := New()
	for i := 0; i < n; i++ {
		db.Add(uint64(i) * 31)
	}
	if err := db.Save(path); err != nil {
		b.Fatal(err)
	}
	return path
}

func BenchmarkOpenReadOnly(b *testing.B) {
	path := createBenchDB(b, 100_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := OpenReadOnly(path)
		if err != nil {
			b.Fatal(err)
		}
		db.Close()
	}
}

func BenchmarkOpenReadWrite(b *testing.B) {
	path := createBenchDB(b, 100_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, err := OpenReadWrite(path)
		if err != nil {
			b.Fatal(err)
		}
		_ = db
	}
}

func BenchmarkContains(b *testing.B) {
	path := createBenchDB(b, 100_000)
	db, err := OpenReadOnly(path)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Contains(uint64(i) * 31)
	}
}

// BenchmarkContains20M_Slice benchmarks sorted-slice lookup with 20M entries.
func BenchmarkContains20M_Slice(b *testing.B) {
	const n = 20_000_000
	rng := newSplitMix64(12345)

	db := New()
	keys := make([]uint64, n)
	for i := range n {
		keys[i] = rng.next()
		db.Add(keys[i])
	}
	db.Compact()

	// Build lookup keys: 50% hits, 50% misses
	lookup := make([]uint64, n)
	for i := range n {
		if i%2 == 0 {
			lookup[i] = keys[rng.next()%uint64(n)]
		} else {
			lookup[i] = rng.next()
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := range b.N {
		db.Contains(lookup[i%n])
	}
}

// BenchmarkContains20M_Map benchmarks map[uint64]struct{} lookup with 20M entries.
func BenchmarkContains20M_Map(b *testing.B) {
	const n = 20_000_000
	rng := newSplitMix64(12345)

	m := make(map[uint64]struct{}, n)
	keys := make([]uint64, n)
	for i := range n {
		keys[i] = rng.next()
		m[keys[i]] = struct{}{}
	}

	// Build lookup keys: 50% hits, 50% misses (same seed pattern as slice bench)
	lookup := make([]uint64, n)
	for i := range n {
		if i%2 == 0 {
			lookup[i] = keys[rng.next()%uint64(n)]
		} else {
			lookup[i] = rng.next()
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := range b.N {
		_ = func() bool { _, ok := m[lookup[i%n]]; return ok }()
	}
}

// splitMix64 is a simple, fast PRNG for deterministic benchmark data.
type splitMix64 struct{ state uint64 }

func newSplitMix64(seed uint64) splitMix64 { return splitMix64{state: seed} }

func (s *splitMix64) next() uint64 {
	s.state += 0x9e3779b97f4a7c15
	z := s.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}
