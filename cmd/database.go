package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var placeholder = struct{}{}

func newDB() hashDB {
	return make(hashDB, 1024*1024)
}

func loadDB(path string) (hashDB, error) {
	// get file size of path to pre allocate proper map size
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	size := fi.Size()
	// fatal if not multiple of 8
	if size%8 != 0 {
		return nil, fmt.Errorf("Invalid database size, corrupt?")
	}

	// create a map of size
	m := make(hashDB, size/8)
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
		m[b] = placeholder
	}
	return m, nil
}

func saveDB(path string, db hashDB) error {
	f, err := os.CreateTemp(filepath.Dir(path), "corediff_temp_db")
	if err != nil {
		return err
	}
	// defer executed in reverse order
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
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	return nil
}
