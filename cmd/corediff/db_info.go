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
	fmt.Printf("Format:    CDDB v1\n")
	fmt.Printf("File size: %d bytes\n", fi.Size())
	fmt.Printf("Hashes:    %d\n", db.Len())
	return nil
}
