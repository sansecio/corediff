package hashdb

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const (
	dbMagic    = "CDDB"
	dbVersion  = 2  // current version written by Save (v2 = XXH3)
	headerSize = 16 // 4 magic + 4 version + 8 count
)

type dbHeader struct {
	Magic   [4]byte
	Version uint32
	Count   uint64
}

// Open opens a hash database from path into memory.
// Only CDDB v2 (XXH3) databases are supported.
func Open(path string) (*HashDB, error) {
	data, version, err := readDB(path)
	if err != nil {
		return nil, err
	}
	set := make(map[uint64]struct{}, len(data))
	for _, h := range data {
		set[h] = struct{}{}
	}
	return &HashDB{set: set, Version: version}, nil
}

// Save writes the database to path atomically in the CDDB format.
func (db *HashDB) Save(path string) error {
	f, err := os.CreateTemp(filepath.Dir(path), "corediff_temp_db")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	hashes := make([]uint64, 0, len(db.set))
	for h := range db.set {
		hashes = append(hashes, h)
	}

	hdr := dbHeader{
		Version: dbVersion,
		Count:   uint64(len(hashes)),
	}
	copy(hdr.Magic[:], dbMagic)

	if err := binary.Write(f, binary.LittleEndian, &hdr); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, hashes); err != nil {
		return fmt.Errorf("writing data: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func readDB(path string) ([]uint64, uint32, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	size := fi.Size()
	if size == 0 {
		return nil, dbVersion, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	if size < headerSize {
		return nil, 0, fmt.Errorf("database too small; legacy xxhash64 databases are no longer supported — please re-index")
	}

	var hdr dbHeader
	if err := binary.Read(f, binary.LittleEndian, &hdr); err != nil {
		return nil, 0, fmt.Errorf("reading header: %w", err)
	}

	if string(hdr.Magic[:]) != dbMagic {
		return nil, 0, fmt.Errorf("not a CDDB database; legacy xxhash64 databases are no longer supported — please re-index")
	}
	if hdr.Version < dbVersion {
		return nil, 0, fmt.Errorf("database version %d uses xxhash64 which is no longer supported — please re-index", hdr.Version)
	}
	if hdr.Version > dbVersion {
		return nil, 0, fmt.Errorf("database version %d is newer than supported (v%d) — please run `corediff update`", hdr.Version, dbVersion)
	}

	dataSize := size - headerSize
	if dataSize%8 != 0 {
		return nil, 0, fmt.Errorf("invalid database size, corrupt?")
	}
	count := dataSize / 8
	if count != int64(hdr.Count) {
		return nil, 0, fmt.Errorf("header count %d doesn't match data (%d entries)", hdr.Count, count)
	}
	data := make([]uint64, count)
	if err := binary.Read(f, binary.LittleEndian, data); err != nil {
		return nil, 0, fmt.Errorf("reading data: %w", err)
	}
	return data, hdr.Version, nil
}
