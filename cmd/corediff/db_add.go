package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
	cdpath "github.com/gwillem/corediff/internal/path"
)

type dbAddArg struct {
	Database     string `short:"d" long:"database" description:"Hash database path" required:"true"`
	IgnorePaths  bool   `short:"i" long:"ignore-paths" description:"Don't store file paths in DB."`
	AllValidText bool   `short:"t" long:"text" description:"Scan all valid UTF-8 text files."`
	NoPlatform   bool   `long:"no-platform" description:"Don't check for app root."`
	Verbose      bool   `short:"v" long:"verbose" description:"Show what is going on"`
	Path         struct {
		Path []string `positional-arg-name:"<path>" required:"1"`
	} `positional-args:"yes" required:"true"`
}

func (a *dbAddArg) Execute(_ []string) error {
	if a.Verbose {
		logLevel = 3
	}

	db, err := hashdb.Load(a.Database)
	if os.IsNotExist(err) {
		db = hashdb.New()
		err = nil
	}
	if err != nil {
		return err
	}

	oldSize := len(db)
	for _, p := range a.Path.Path {
		fmt.Println("Calculating checksums for", p)
		addPath(p, db, a.IgnorePaths, a.AllValidText)
		fmt.Println()
	}

	if len(db) != oldSize {
		fmt.Println("Computed", len(db)-oldSize, "new hashes, saving to", a.Database, "..")
		return hashdb.Save(a.Database, db)
	}
	fmt.Println("Found no new code hashes...")
	return nil
}

func addPath(root string, db hashdb.HashDB, ignorePaths bool, allValidText bool) {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		var relPath string
		if path == root {
			relPath = root
		} else {
			relPath = path[len(root)+1:]
		}

		if err != nil {
			fmt.Printf("failure accessing a path %q: %v\n", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}

		if !allValidText && !normalize.HasValidExt(path) {
			logVerbose(grey(" - ", relPath, " (no code)"))
			return nil
		} else if !normalize.IsValidUtf8(path) {
			logVerbose(grey(" - ", relPath, " (invalid utf8)"))
			return nil
		}

		if !ignorePaths && path != root && !cdpath.IsExcluded(relPath) {
			db.Add(normalize.PathHash(relPath))
		}

		hits, _ := parseFileWithDB(path, db, true)
		if len(hits) > 0 {
			logVerbose(green(" U " + relPath))
		} else {
			logVerbose(grey(" - " + relPath))
		}

		return nil
	})
	if err != nil {
		log.Fatalln("error walking the path", root, err)
	}
}
