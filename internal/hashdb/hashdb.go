package hashdb

import (
	"encoding/binary"
	"os"
	"sync"
)

// HashDB stores precomputed hashes using a map for O(1) lookups.
type HashDB struct {
	set     map[uint64]struct{}
	Version uint32 // CDDB version (2 = xxh3)
	mu      sync.Mutex
	file    *os.File // write handle, nil if read-only
}

// New creates an empty HashDB.
func New() *HashDB {
	return &HashDB{set: make(map[uint64]struct{})}
}

// Contains reports whether h is in the database.
func (db *HashDB) Contains(h uint64) bool {
	_, ok := db.set[h]
	return ok
}

// Add inserts h into the database. If opened for writing, the hash is also
// appended to the file (without updating the header count yet).
func (db *HashDB) Add(h uint64) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, exists := db.set[h]; exists {
		return
	}
	db.set[h] = struct{}{}
	if db.file != nil {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], h)
		db.file.Write(buf[:]) //nolint:errcheck // best-effort append
	}
}

// Merge adds all entries from other into this database.
// If opened for writing, new hashes are appended and the header count is
// updated + synced for crash durability.
func (db *HashDB) Merge(other *HashDB) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Collect new hashes
	var newHashes []uint64
	for h := range other.set {
		if _, exists := db.set[h]; !exists {
			db.set[h] = struct{}{}
			newHashes = append(newHashes, h)
		}
	}

	if db.file == nil || len(newHashes) == 0 {
		return
	}

	// 1. Append hash bytes to end of file
	buf := make([]byte, 8*len(newHashes))
	for i, h := range newHashes {
		binary.LittleEndian.PutUint64(buf[i*8:], h)
	}
	db.file.Write(buf) //nolint:errcheck

	// 2. Sync data to disk
	db.file.Sync() //nolint:errcheck

	// 3. Update count in header
	db.writeCount()

	// 4. Sync header to disk
	db.file.Sync() //nolint:errcheck
}

// Flush updates the header count to match len(set) and syncs to disk.
// Safe to call multiple times.
func (db *HashDB) Flush() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.file == nil {
		return nil
	}
	db.writeCount()
	return db.file.Sync()
}

// Close flushes and closes the write file handle. Idempotent.
func (db *HashDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.file == nil {
		return nil
	}
	db.writeCount()
	db.file.Sync() //nolint:errcheck
	err := db.file.Close()
	db.file = nil
	return err
}

// writeCount writes len(set) into the header's count field at offset 8.
// Caller must hold db.mu.
func (db *HashDB) writeCount() {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(len(db.set)))
	db.file.WriteAt(buf[:], 8) //nolint:errcheck
}

// Len returns the number of unique entries.
func (db *HashDB) Len() int {
	return len(db.set)
}
