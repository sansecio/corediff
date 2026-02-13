package main

import (
	"fmt"

	"github.com/gwillem/corediff/internal/hashdb"
)

type dbMergeArg struct {
	Path struct {
		Path []string `positional-arg-name:"<db-file>" required:"1"`
	} `positional-args:"yes" required:"true"`
}

func (m *dbMergeArg) Execute(_ []string) error {
	dbPath := dbCommand.Database
	out, err := hashdb.OpenReadWrite(dbPath)
	if err != nil {
		out = hashdb.New()
	}

	totalInput := out.Len()
	for _, p := range m.Path.Path {
		db, err := hashdb.OpenReadOnly(p)
		if err != nil {
			return fmt.Errorf("loading %s: %w", p, err)
		}
		fmt.Printf("Merging %s with %d entries ..\n", p, db.Len())
		totalInput += db.Len()
		out.Merge(db)
		db.Close()
	}

	out.Compact()
	dupes := totalInput - out.Len()
	fmt.Printf("Saving %s with %d entries (%d duplicates removed).\n", dbPath, out.Len(), dupes)
	return out.Save(dbPath)
}
