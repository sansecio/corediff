package hashdb

import (
	"sort"
	"syscall"
)

// HashDB stores precomputed hashes using sorted slices for memory efficiency.
// The main slice is sorted and deduped; buf is an unsorted append buffer.
type HashDB struct {
	main     []uint64
	buf      []uint64
	readOnly bool
	mmapData []byte // non-nil when backed by mmap
}

// New creates an empty read-write HashDB.
func New() *HashDB {
	return &HashDB{}
}

// Contains reports whether h is in the database.
// Uses binary search on the sorted main slice and linear scan on the buffer.
func (db *HashDB) Contains(h uint64) bool {
	i := sort.Search(len(db.main), func(i int) bool {
		return db.main[i] >= h
	})
	if i < len(db.main) && db.main[i] == h {
		return true
	}
	for _, v := range db.buf {
		if v == h {
			return true
		}
	}
	return false
}

// Add inserts h into the database. Panics on read-only databases.
func (db *HashDB) Add(h uint64) {
	if db.readOnly {
		panic("hashdb: cannot add to read-only database")
	}
	db.buf = append(db.buf, h)
}

// Merge adds all entries from other into this database.
func (db *HashDB) Merge(other *HashDB) {
	db.buf = append(db.buf, other.main...)
	db.buf = append(db.buf, other.buf...)
}

// Compact sorts the buffer, merges it with main, and deduplicates.
func (db *HashDB) Compact() {
	if len(db.buf) == 0 {
		return
	}
	all := make([]uint64, 0, len(db.main)+len(db.buf))
	all = append(all, db.main...)
	all = append(all, db.buf...)
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	// Dedup in place
	if len(all) > 0 {
		j := 0
		for i := 1; i < len(all); i++ {
			if all[i] != all[j] {
				j++
				all[j] = all[i]
			}
		}
		all = all[:j+1]
	}
	db.main = all
	db.buf = nil
}

// Len returns the total number of entries (main + buffer, may include dupes).
func (db *HashDB) Len() int {
	return len(db.main) + len(db.buf)
}

// Close releases resources. Must be called on mmap-backed databases.
func (db *HashDB) Close() error {
	if db.mmapData != nil {
		err := syscall.Munmap(db.mmapData)
		db.mmapData = nil
		db.main = nil
		return err
	}
	return nil
}
