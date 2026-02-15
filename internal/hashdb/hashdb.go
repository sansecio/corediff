package hashdb

// HashDB stores precomputed hashes using a map for O(1) lookups.
type HashDB struct {
	set     map[uint64]struct{}
	Version uint32 // CDDB version (2 = xxh3)
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

// Add inserts h into the database.
func (db *HashDB) Add(h uint64) {
	db.set[h] = struct{}{}
}

// Merge adds all entries from other into this database.
func (db *HashDB) Merge(other *HashDB) {
	for h := range other.set {
		db.set[h] = struct{}{}
	}
}

// Len returns the number of unique entries.
func (db *HashDB) Len() int {
	return len(db.set)
}
