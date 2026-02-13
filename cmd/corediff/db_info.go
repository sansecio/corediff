package main

import (
	"fmt"
	"os"

	"github.com/gwillem/corediff/internal/hashdb"
)

type dbInfoArg struct {
	Path struct {
		Path string `positional-arg-name:"<db-file>" required:"true"`
	} `positional-args:"yes" required:"true"`
}

func (a *dbInfoArg) Execute(_ []string) error {
	fi, err := os.Stat(a.Path.Path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", a.Path.Path, err)
	}

	db, err := hashdb.OpenReadOnly(a.Path.Path)
	if err != nil {
		return fmt.Errorf("open %s: %w", a.Path.Path, err)
	}
	defer db.Close()

	fmt.Printf("Database:  %s\n", a.Path.Path)
	fmt.Printf("Format:    CDDB v1\n")
	fmt.Printf("File size: %d bytes\n", fi.Size())
	fmt.Printf("Hashes:    %d\n", db.Len())
	return nil
}
