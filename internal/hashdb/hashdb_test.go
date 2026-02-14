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

	assert.True(t, db.Contains(42))
	assert.True(t, db.Contains(100))
	assert.True(t, db.Contains(1))
	assert.False(t, db.Contains(99))
}

func TestDeduplication(t *testing.T) {
	db := New()
	db.Add(5)
	db.Add(3)
	db.Add(5) // duplicate
	db.Add(1)
	db.Add(3) // duplicate

	assert.Equal(t, 3, db.Len())
	assert.True(t, db.Contains(1))
	assert.True(t, db.Contains(3))
	assert.True(t, db.Contains(5))
}

func TestMerge(t *testing.T) {
	db1 := New()
	db1.Add(1)
	db1.Add(2)

	db2 := New()
	db2.Add(2) // overlap
	db2.Add(3)

	db1.Merge(db2)
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

	loaded, err := Open(path)
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

	loaded, err := Open(path)
	require.NoError(t, err)
	assert.Equal(t, 0, loaded.Len())
	assert.False(t, loaded.Contains(42))
}

func TestSaveDeduplication(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db := New()
	for range 100 {
		db.Add(42)
	}
	require.NoError(t, db.Save(path))

	loaded, err := Open(path)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Len())
}

func TestCorruptFile(t *testing.T) {
	dir := t.TempDir()

	// File size not a multiple of 8 (invalid for both formats)
	path := filepath.Join(dir, "corrupt.db")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))
	_, err := Open(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a multiple of 8")
}

func TestLegacyFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")

	// Write raw sequential uint64s (no header)
	f, err := os.Create(path)
	require.NoError(t, err)
	hashes := []uint64{42, 100, 1}
	require.NoError(t, binary.Write(f, binary.LittleEndian, hashes))
	require.NoError(t, f.Close())

	db, err := Open(path)
	require.NoError(t, err)
	assert.Equal(t, 3, db.Len())
	assert.True(t, db.Contains(42))
	assert.True(t, db.Contains(100))
	assert.True(t, db.Contains(1))
	assert.False(t, db.Contains(99))
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

	_, err = Open(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database version")
}

func TestFileNotFound(t *testing.T) {
	_, err := Open("/nonexistent/path.db")
	assert.Error(t, err)
}

func TestOpenAndMutate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db := New()
	db.Add(1)
	db.Add(2)
	require.NoError(t, db.Save(path))

	loaded, err := Open(path)
	require.NoError(t, err)
	loaded.Add(3) // should not panic
	assert.Equal(t, 3, loaded.Len())
}

func createBenchDB(b *testing.B, n int) string {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.db")
	db := New()
	for i := range n {
		db.Add(uint64(i) * 31)
	}
	if err := db.Save(path); err != nil {
		b.Fatal(err)
	}
	return path
}

func BenchmarkOpen(b *testing.B) {
	path := createBenchDB(b, 100_000)
	b.ResetTimer()
	for range b.N {
		db, err := Open(path)
		if err != nil {
			b.Fatal(err)
		}
		_ = db
	}
}

func BenchmarkContains(b *testing.B) {
	path := createBenchDB(b, 100_000)
	db, err := Open(path)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := range b.N {
		db.Contains(uint64(i) * 31)
	}
}

// BenchmarkContains20M benchmarks map lookup with 20M entries.
func BenchmarkContains20M(b *testing.B) {
	const n = 20_000_000
	rng := newSplitMix64(12345)

	db := New()
	keys := make([]uint64, n)
	for i := range n {
		keys[i] = rng.next()
		db.Add(keys[i])
	}

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
