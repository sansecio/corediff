package main

import (
	"fmt"
	"os"

	"github.com/gwillem/corediff/internal/hashdb"
)

type dbInfoArg struct{}

func (a *dbInfoArg) Execute(_ []string) error {
	dbPath := dbCommand.Database
	fi, err := os.Stat(dbPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", dbPath, err)
	}

	db, err := hashdb.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", dbPath, err)
	}

	fmt.Printf("Database:  %s\n", dbPath)
	var format string
	switch db.Version {
	case 0:
		format = "legacy (xxhash64, no header)"
	case 1:
		format = "CDDB v1 (xxhash64)"
	case 2:
		format = "CDDB v2 (xxh3)"
	default:
		format = fmt.Sprintf("unknown (v%d)", db.Version)
	}
	fmt.Printf("Format:    %s\n", format)
	fmt.Printf("File size: %d bytes\n", fi.Size())
	fmt.Printf("Hashes:    %d\n", db.Len())
	return nil
}
