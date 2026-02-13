package main

import (
	"fmt"

	"github.com/gwillem/corediff/internal/hashdb"
)

type dbMergeArg struct {
	Database string `short:"d" long:"database" description:"Output database path" required:"true"`
	Path     struct {
		Path []string `positional-arg-name:"<db-file>" required:"1"`
	} `positional-args:"yes" required:"true"`
}

func (m *dbMergeArg) Execute(_ []string) error {
	out, err := hashdb.Load(m.Database)
	if err != nil {
		out = hashdb.New()
	}

	for _, p := range m.Path.Path {
		db, err := hashdb.Load(p)
		if err != nil {
			return fmt.Errorf("loading %s: %w", p, err)
		}
		fmt.Printf("Merging %s with %d entries ..\n", p, len(db))
		for k := range db {
			out.Add(k)
		}
	}

	fmt.Printf("Saving %s with a total of %d entries.\n", m.Database, len(out))
	return hashdb.Save(m.Database, out)
}
