package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func loadDB(path string) hashDB {

	m := make(hashDB)

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return m
	}
	check(err)
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		b := make([]byte, 16)
		n, err := reader.Read(b)
		if n == 0 {
			break
		}
		check(err)
		var b2 [16]byte
		copy(b2[:], b) // need to convert to array first
		m[b2] = true
	}
	return m
}

func saveDB(path string, db hashDB) {
	f, err := os.Create(path)
	defer f.Close()
	check(err)
	for k := range db {
		n, err := f.Write(k[:])
		check(err)
		if n != 16 {
			log.Fatal("Wrote unexpected number of bytes?")
		}
	}
}

func parseFile(path, relPath string, db hashDB, updateDB bool) (hits []int, lines [][]byte) {
	fh, err := os.Open(path)
	check(err)
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	buf = buf[:0]
	scanner.Buffer(buf, maxTokenSize)
	for i := 0; scanner.Scan(); i++ {
		x := scanner.Bytes()
		l := make([]byte, len(x))
		copy(l, x)
		lines = append(lines, l)
		h := hash(normalizeLine(l))
		if !db[h] {
			hits = append(hits, i)
			if updateDB {
				db[h] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return hits, lines
}

func checkPath(root string, db hashDB, args *baseArgs) *walkStats {
	stats := &walkStats{}
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

		stats.totalFiles++

		if !hasValidExt(path) {
			stats.filesNoCode++
			// logVerbose(grey(" ? ", relPath))
			return nil
		}

		// Only do path checking for non-root elts
		if path != root && !args.IgnorePaths {
			if !db[pathHash(relPath)] {
				stats.filesCustomCode++
				logVerbose(grey(" ? ", relPath))
				return nil
			}
		}

		hits, lines := parseFile(path, relPath, db, false)
		if len(hits) > 0 {
			stats.filesWithChanges++
			logInfo(boldred("\n X " + relPath))
			for _, idx := range hits {
				// fmt.Println(string(lines[idx]))
				if shouldHighlight(lines[idx]) {
					logInfo("  ", grey(fmt.Sprintf("%-5d", idx)), alarm(string(lines[idx])))
					// fmt.Printf("%s %s\n", grey(fmt.Sprintf("%-5d", idx)), alarm(string(lines[idx])))

				} else {
					logInfo("  ", grey(fmt.Sprintf("%-5d", idx)), string(lines[idx]))
					// fmt.Printf("%s %s\n", grey(fmt.Sprintf("%-5d", idx)), string(lines[idx]))
				}
			}
			logInfo()
		} else {
			stats.filesWithoutChanges++
			logVerbose(green(" V " + relPath))
		}

		return nil
	})
	check(err)
	return stats
}

func addPath(root string, db hashDB, args *baseArgs) {
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

		if !hasValidExt(path) {
			logVerbose(grey(" - ", relPath, " (no code)"))
			return nil
		}

		// If relPath has valid ext, add hash of "path:<relPath>" to db
		// Never add root path (possibly file)
		if !args.IgnorePaths && path != root && !pathIsExcluded(relPath) {
			db[pathHash(relPath)] = true
		}

		hits, _ := parseFile(path, relPath, db, true)
		if len(hits) > 0 {
			logVerbose(green(" U " + relPath))
		} else {
			logVerbose(grey(" - " + relPath))
		}

		return nil
	})
	check(err)

}

func main() {

	args := setup()
	db := loadDB(args.Database)

	logInfo(boldwhite("\nMagento Corediff loaded ", len(db), " precomputed hashes. (C) 2020 info@sansec.io"))
	logInfo("Using database:", args.Database, "\n")

	if args.Merge {
		for _, p := range args.Path.Path {
			db2 := loadDB(p)
			logInfo("Merging", filepath.Base(p), "with", len(db2), "entries ..")
			for k := range db2 {
				db[k] = true
			}
		}
		logInfo("Saving", args.Database, "with a total of", len(db), "entries.")
		saveDB(args.Database, db)
	} else if args.Add {
		oldSize := len(db)
		for _, path := range args.Path.Path {
			logInfo("Calculating checksums for", path, "\n")
			addPath(path, db, args)
			logInfo()
		}
		if len(db) != oldSize {
			logInfo("Computed", len(db)-oldSize, "new hashes, saving to", args.Database, "..")
			saveDB(args.Database, db)
		} else {
			logInfo("Found no new code hashes...")
		}
	} else {
		for _, path := range args.Path.Path {
			stats := checkPath(path, db, args)
			logInfo("\n===============================================================================")
			logInfo(" Corediff completed scanning", stats.totalFiles, "files in", path)
			logInfo(" - Files with unrecognized lines   :", boldred(stats.filesWithChanges))
			logInfo(" - Files with only recognized lines:", green(stats.filesWithoutChanges))
			logInfo(" - Files with custom code          :", stats.filesCustomCode)
			logInfo(" - Files without code              :", stats.filesNoCode)
		}
	}
	logInfo()
}
