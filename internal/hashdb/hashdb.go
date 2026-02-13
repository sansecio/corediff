package hashdb

import (
	"sort"
	"syscall"
)

// HashDB stores precomputed hashes using a map for O(1) lookups.
// The main slice holds sorted, deduped data for file I/O.
// The set map is the primary lookup structure.
type HashDB struct {
	main     []uint64               // sorted, deduped — used for Save
	set      map[uint64]struct{}    // primary lookup structure
	readOnly bool
	mmapData []byte // non-nil when backed by mmap
}

// New creates an empty read-write HashDB.
func New() *HashDB {
	return &HashDB{set: make(map[uint64]struct{})}
}

// Contains reports whether h is in the database.
func (db *HashDB) Contains(h uint64) bool {
	_, ok := db.set[h]
	return ok
}

// Add inserts h into the database. Panics on read-only databases.
func (db *HashDB) Add(h uint64) {
	if db.readOnly {
		panic("hashdb: cannot add to read-only database")
	}
	db.set[h] = struct{}{}
}

// Merge adds all entries from other into this database.
func (db *HashDB) Merge(other *HashDB) {
	for h := range other.set {
		db.set[h] = struct{}{}
	}
}

// Compact rebuilds the sorted main slice from the set.
func (db *HashDB) Compact() {
	db.main = make([]uint64, 0, len(db.set))
	for h := range db.set {
		db.main = append(db.main, h)
	}
	sort.Slice(db.main, func(i, j int) bool { return db.main[i] < db.main[j] })
}

// Len returns the number of unique entries.
func (db *HashDB) Len() int {
	return len(db.set)
}

// Close releases resources. Must be called on mmap-backed databases.
func (db *HashDB) Close() error {
	if db.mmapData != nil {
		err := syscall.Munmap(db.mmapData)
		db.mmapData = nil
		db.main = nil
		db.set = nil
		return err
	}
	return nil
}
