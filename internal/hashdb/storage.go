package hashdb

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const (
	dbMagic    = "CDDB"
	dbVersion  = 1
	headerSize = 16 // 4 magic + 4 version + 8 count
)

type dbHeader struct {
	Magic   [4]byte
	Version uint32
	Count   uint64
}

// Open opens a hash database from path into memory.
func Open(path string) (*HashDB, error) {
	data, err := readDB(path)
	if err != nil {
		return nil, err
	}
	set := make(map[uint64]struct{}, len(data))
	for _, h := range data {
		set[h] = struct{}{}
	}
	return &HashDB{set: set}, nil
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

func readDB(path string) ([]uint64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	if size == 0 {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Detect format: check for CDDB magic header
	var magic [4]byte
	if size >= 4 {
		if _, err := f.Read(magic[:]); err != nil {
			return nil, fmt.Errorf("reading magic: %w", err)
		}
	}

	if string(magic[:]) == dbMagic {
		return readCDDB(f, size)
	}

	// Legacy format: raw sequential little-endian uint64s
	return readLegacy(f, size)
}

func readCDDB(f *os.File, size int64) ([]uint64, error) {
	if size < headerSize {
		return nil, fmt.Errorf("file too small for CDDB header")
	}

	// Seek back to start to read full header
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}

	var hdr dbHeader
	if err := binary.Read(f, binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	if hdr.Version != dbVersion {
		return nil, fmt.Errorf("unsupported database version %d (max supported: %d)", hdr.Version, dbVersion)
	}
	dataSize := size - headerSize
	if dataSize%8 != 0 {
		return nil, fmt.Errorf("invalid database size, corrupt?")
	}
	count := dataSize / 8
	if count != int64(hdr.Count) {
		return nil, fmt.Errorf("header count %d doesn't match data (%d entries)", hdr.Count, count)
	}
	data := make([]uint64, count)
	if err := binary.Read(f, binary.LittleEndian, data); err != nil {
		return nil, fmt.Errorf("reading data: %w", err)
	}
	return data, nil
}

func readLegacy(f *os.File, size int64) ([]uint64, error) {
	if size%8 != 0 {
		return nil, fmt.Errorf("invalid legacy database size (%d bytes, not a multiple of 8)", size)
	}

	// Seek back to start — we already read 4 bytes for magic detection
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}

	count := size / 8
	data := make([]uint64, count)
	if err := binary.Read(f, binary.LittleEndian, data); err != nil {
		return nil, fmt.Errorf("reading legacy data: %w", err)
	}
	return data, nil
}
