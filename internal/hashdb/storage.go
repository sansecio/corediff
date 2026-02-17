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

// Open opens a hash database from path into memory (read-only).
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

// OpenForWrite opens a hash database for incremental appending.
// If the file exists, existing hashes are loaded into memory and the file is
// reopened for append. If it doesn't exist, a new file with a valid header is
// created. Trailing bytes beyond headerSize + count*8 are truncated (from
// interrupted writes).
func OpenForWrite(path string) (*HashDB, error) {
	var set map[uint64]struct{}
	var version uint32 = dbVersion

	if _, err := os.Stat(path); err == nil {
		// File exists — load existing hashes
		data, ver, readErr := readDB(path)
		if readErr != nil {
			return nil, readErr
		}
		set = make(map[uint64]struct{}, len(data))
		for _, h := range data {
			set[h] = struct{}{}
		}
		version = ver

		// Reopen for read-write and truncate trailing garbage
		f, err := os.OpenFile(path, os.O_RDWR, 0o644)
		if err != nil {
			return nil, err
		}
		validSize := int64(headerSize) + int64(len(data))*8
		if err := f.Truncate(validSize); err != nil {
			f.Close()
			return nil, fmt.Errorf("truncating trailing bytes: %w", err)
		}
		if _, err := f.Seek(0, 2); err != nil { // seek to end
			f.Close()
			return nil, err
		}
		return &HashDB{set: set, Version: version, file: f}, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// File doesn't exist — create with header
	set = make(map[uint64]struct{})
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	hdr := dbHeader{Version: dbVersion, Count: 0}
	copy(hdr.Magic[:], dbMagic)
	if err := binary.Write(f, binary.LittleEndian, &hdr); err != nil {
		f.Close()
		return nil, fmt.Errorf("writing header: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	return &HashDB{set: set, Version: dbVersion, file: f}, nil
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
	dataEntries := dataSize / 8

	// Use header count as the authoritative entry count.
	// Extra trailing bytes (from a crash mid-append) are ignored.
	if dataEntries < int64(hdr.Count) {
		return nil, 0, fmt.Errorf("header count %d but only %d entries in file (truncated?)", hdr.Count, dataEntries)
	}

	count := int64(hdr.Count)
	data := make([]uint64, count)
	if err := binary.Read(f, binary.LittleEndian, data); err != nil {
		return nil, 0, fmt.Errorf("reading data: %w", err)
	}
	return data, hdr.Version, nil
}
