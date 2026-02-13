package hashdb

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// HashDB stores precomputed hashes for fast lookup.
type HashDB map[uint64]struct{}

// New creates an empty HashDB with pre-allocated capacity.
func New() HashDB {
	return make(HashDB, 1024*1024)
}

// Contains reports whether h is in the database.
func (db HashDB) Contains(h uint64) bool {
	_, ok := db[h]
	return ok
}

// Add inserts h into the database.
func (db HashDB) Add(h uint64) {
	db[h] = struct{}{}
}

// Load reads a hash database from a binary file of little-endian uint64 values.
func Load(path string) (HashDB, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size%8 != 0 {
		return nil, fmt.Errorf("invalid database size, corrupt?")
	}

	m := make(HashDB, size/8)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return m, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	var b uint64
	for {
		err = binary.Read(reader, binary.LittleEndian, &b)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		m[b] = struct{}{}
	}
	return m, nil
}

// Save writes the hash database to a binary file atomically.
func Save(path string, db HashDB) error {
	f, err := os.CreateTemp(filepath.Dir(path), "corediff_temp_db")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	for k := range db {
		if err := binary.Write(f, binary.LittleEndian, k); err != nil {
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
