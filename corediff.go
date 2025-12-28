/*
   corediff: quickly find unauthorized code changes to common applications

   Copyright (C) 2020-2026 Sansec BV contributors <info@sansec.io>

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gwillem/go-buildversion"
	"github.com/gwillem/go-selfupdate"
)

var (
	selfUpdateURL   = fmt.Sprintf("https://sansec.io/downloads/%s-%s/corediff", runtime.GOOS, runtime.GOARCH)
	placeholder     = struct{}{}
	corediffVersion = buildversion.String()
)

func loadDB(path string) hashDB {
	// get file size of path to pre allocate proper map size
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		// creating new db?
		return make(hashDB, 0)
	} else if err != nil {
		log.Fatal(err)
	}
	size := fi.Size()
	// fatal if not multiple of 8
	if size%8 != 0 {
		log.Fatal("Invalid database size, corrupt?")
	}

	// create a map of size
	m := make(hashDB, size/8)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return m
	} else if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	var b uint64
	for {
		err = binary.Read(reader, binary.LittleEndian, &b)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		m[b] = placeholder
	}
	return m
}

func saveDB(path string, db hashDB) {
	f, err := os.CreateTemp(filepath.Dir(path), "corediff_temp_db")
	if err != nil {
		log.Fatal(err)
	}
	// defer executed in reverse order
	defer os.Remove(f.Name())
	defer f.Close()
	for k := range db {
		if err := binary.Write(f, binary.LittleEndian, k); err != nil {
			log.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		log.Fatal(err)
	}
}

func parseFile(path string, db hashDB, updateDB bool) (hits []int, lines [][]byte) {
	fh, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		fmt.Println(warn("file does not exist: " + path))
		return nil, nil
	} else if err != nil {
		log.Fatal("open error on", path, err)
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	buf = buf[:0]
	scanner.Buffer(buf, maxTokenSize)
	for i := 0; scanner.Scan(); i++ {
		x := scanner.Bytes()
		h := hash(normalizeLine(x))
		if _, ok := db[h]; !ok {
			hits = append(hits, i)
			lines = append(lines, x) // to show specific line number
			if updateDB {
				db[h] = placeholder
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
		if args.PathFilter != "" && !strings.HasPrefix(relPath, args.PathFilter) {
			return nil
		}

		stats.totalFiles++

		if (!args.AllValidText && !hasValidExt(path)) || (args.AllValidText && !isValidUtf8(path)) {
			stats.filesNoCode++
			return nil
		}

		// Only do path checking for non-root elts
		if path != root && !args.IgnorePaths {

			_, foundInDb := db[pathHash(relPath)]
			shouldExclude := pathIsExcluded(relPath)

			if !foundInDb || shouldExclude {
				stats.filesCustomCode++
				logVerbose(grey(" ? ", relPath))
				return nil
			}
		}

		hits, lines := parseFile(path, db, false)

		if args.SuspectOnly {
			hitsFiltered := []int{}
			linesFiltered := [][]byte{}
			for i, lineNo := range hits {
				if shouldHighlight(lines[i]) {
					hitsFiltered = append(hitsFiltered, lineNo)
					linesFiltered = append(linesFiltered, lines[i])
				}
			}
			hits = hitsFiltered
			lines = linesFiltered
		}

		if len(hits) > 0 {
			stats.filesWithChanges++
			hasSuspectLines := false
			fmt.Println(boldred("\n X " + relPath))
			for i, lineNo := range hits {
				// fmt.Println(string(lines[idx]))
				if shouldHighlight(lines[i]) {
					hasSuspectLines = true
					fmt.Println("  ", grey(fmt.Sprintf("%-5d", lineNo)), alarm(string(lines[i])))
					// fmt.Printf("%s %s\n", grey(fmt.Sprintf("%-5d", idx)), alarm(string(lines[idx])))
				} else if !args.SuspectOnly {
					fmt.Println("  ", grey(fmt.Sprintf("%-5d", lineNo)), string(lines[i]))
					// fmt.Printf("%s %s\n", grey(fmt.Sprintf("%-5d", idx)), string(lines[idx]))
				}
			}
			if hasSuspectLines {
				stats.filesWithSuspectLines++
			}
			fmt.Println()
		} else {
			stats.filesWithoutChanges++
			if args.Verbose {
				stats.undetectedPaths = append(stats.undetectedPaths, path)
			}
			logVerbose(green(" V " + relPath))
		}

		return nil
	})
	if err != nil {
		log.Fatalln("error walking the path", root, err)
	}
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

		if !args.AllValidText && !hasValidExt(path) {
			logVerbose(grey(" - ", relPath, " (no code)"))
			return nil
		} else if !isValidUtf8(path) {
			logVerbose(grey(" - ", relPath, " (invalid utf8)"))
			return nil
		}

		// If relPath has valid ext, add hash of "path:<relPath>" to db
		// Never add root path (possibly file)
		if !args.IgnorePaths && path != root && !pathIsExcluded(relPath) {
			db[pathHash(relPath)] = placeholder
		}

		hits, _ := parseFile(path, db, true)
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

func main() {
	if restarted, err := selfupdate.UpdateRestart(selfUpdateURL); restarted || err != nil {
		logVerbose("Restarted new version", restarted, "with error:", err)
	}

	args := setup()
	db := loadDB(args.Database)

	fmt.Println(boldwhite("Corediff ", corediffVersion, " loaded ", len(db), " precomputed hashes. (C) 2020-2024 labs@sansec.io"))
	fmt.Println("Using database:", args.Database)

	if args.Merge {
		for _, p := range args.Path.Path {
			db2 := loadDB(p)
			fmt.Println("Merging", filepath.Base(p), "with", len(db2), "entries ..")
			for k := range db2 {
				db[k] = placeholder
			}
		}
		fmt.Println("Saving", args.Database, "with a total of", len(db), "entries.")
		saveDB(args.Database, db)
	} else if args.Add {
		oldSize := len(db)
		for _, path := range args.Path.Path {
			fmt.Println("Calculating checksums for", path)
			addPath(path, db, args)
			fmt.Println()
		}
		if len(db) != oldSize {
			fmt.Println("Computed", len(db)-oldSize, "new hashes, saving to", args.Database, "..")
			saveDB(args.Database, db)
		} else {
			fmt.Println("Found no new code hashes...")
		}
	} else {
		without := "code"
		if args.AllValidText {
			without = "text"
		}
		for _, path := range args.Path.Path {
			stats := checkPath(path, db, args)
			fmt.Println("\n===============================================================================")
			fmt.Println(" Corediff completed scanning", stats.totalFiles, "files in", path)
			fmt.Println(" - Files with unrecognized lines      :", boldred(fmt.Sprintf("%7d", stats.filesWithChanges)), grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesWithChanges))))
			fmt.Println(" - Files with suspect lines           :", warn(fmt.Sprintf("%7d", stats.filesWithSuspectLines)), grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesWithSuspectLines))))
			fmt.Println(" - Files with only recognized lines   :", green(fmt.Sprintf("%7d", stats.filesWithoutChanges)), grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesWithoutChanges))))
			fmt.Println(" - Files with custom code             :", fmt.Sprintf("%7d", stats.filesCustomCode), grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesCustomCode))))
			fmt.Println(" - Files without", without, "                :", fmt.Sprintf("%7d", stats.filesNoCode), grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesNoCode))))
			logVerbose("Undetected paths:")
			for _, p := range stats.undetectedPaths {
				logVerbose("  ", p)
			}
		}
	}
}
