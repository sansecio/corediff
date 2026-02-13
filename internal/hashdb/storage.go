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

// OpenReadOnly opens a hash database for querying only. Add() will panic.
func OpenReadOnly(path string) (*HashDB, error) {
	data, err := readDB(path)
	if err != nil {
		return nil, err
	}
	return &HashDB{main: data, readOnly: true}, nil
}

// OpenReadWrite opens a hash database for querying and mutation.
func OpenReadWrite(path string) (*HashDB, error) {
	data, err := readDB(path)
	if err != nil {
		return nil, err
	}
	return &HashDB{main: data}, nil
}

// Save writes the database to path atomically in the CDDB format.
func (db *HashDB) Save(path string) error {
	db.Compact()

	f, err := os.CreateTemp(filepath.Dir(path), "corediff_temp_db")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	hdr := dbHeader{
		Version: dbVersion,
		Count:   uint64(len(db.main)),
	}
	copy(hdr.Magic[:], dbMagic)

	if err := binary.Write(f, binary.LittleEndian, &hdr); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, db.main); err != nil {
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

	if size < headerSize {
		return nil, fmt.Errorf("file too small for CDDB header")
	}

	var hdr dbHeader
	if err := binary.Read(f, binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}
	if string(hdr.Magic[:]) != dbMagic {
		return nil, fmt.Errorf("not a CDDB file (bad magic)")
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
