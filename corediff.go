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
	m := make(hashDB)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return m
	} else if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		var b uint64
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
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	for k := range db {
		if err := binary.Write(f, binary.LittleEndian, k); err != nil {
			log.Fatal(err)
		}
	}
}

func parseFile(path, relPath string, db hashDB, updateDB bool) (hits []int, lines [][]byte) {
	fh, err := os.Open(path)
	if os.IsNotExist(err) {
		logInfo(warn("file does not exist: " + path))
		return nil, nil
	}
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
		if _, ok := db[h]; !ok {
			hits = append(hits, i)
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

		hits, lines := parseFile(path, relPath, db, false)

		if args.SuspectOnly {
			hitsFiltered := []int{}
			for _, idx := range hits {
				if shouldHighlight(lines[idx]) {
					hitsFiltered = append(hitsFiltered, idx)
				}
			}
			hits = hitsFiltered
		}

		if len(hits) > 0 {
			stats.filesWithChanges++
			hasSuspectLines := false
			logInfo(boldred("\n X " + relPath))
			for _, idx := range hits {
				// fmt.Println(string(lines[idx]))
				if shouldHighlight(lines[idx]) {
					hasSuspectLines = true
					logInfo("  ", grey(fmt.Sprintf("%-5d", idx)), alarm(string(lines[idx])))
					// fmt.Printf("%s %s\n", grey(fmt.Sprintf("%-5d", idx)), alarm(string(lines[idx])))
				} else if !args.SuspectOnly {
					logInfo("  ", grey(fmt.Sprintf("%-5d", idx)), string(lines[idx]))
					// fmt.Printf("%s %s\n", grey(fmt.Sprintf("%-5d", idx)), string(lines[idx]))
				}
			}
			if hasSuspectLines {
				stats.filesWithSuspectLines++
			}
			logInfo()
		} else {
			stats.filesWithoutChanges++
			if args.Verbose {
				stats.undetectedPaths = append(stats.undetectedPaths, path)
			}
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
	if restarted, err := selfupdate.UpdateRestart(selfUpdateURL); restarted || err != nil {
		logVerbose("Restarted new version", restarted, "with error:", err)
	}

	args := setup()
	db := loadDB(args.Database)

	logInfo(boldwhite("Corediff ", corediffVersion, " loaded ", len(db), " precomputed hashes. (C) 2020-2024 labs@sansec.io"))
	logInfo("Using database:", args.Database, "\n")

	if args.Merge {
		for _, p := range args.Path.Path {
			db2 := loadDB(p)
			logInfo("Merging", filepath.Base(p), "with", len(db2), "entries ..")
			for k := range db2 {
				db[k] = placeholder
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
		without := "code"
		if args.AllValidText {
			without = "text"
		}
		for _, path := range args.Path.Path {
			stats := checkPath(path, db, args)
			logInfo("\n===============================================================================")
			logInfo(" Corediff completed scanning", stats.totalFiles, "files in", path)
			logInfo(" - Files with unrecognized lines      :", boldred(stats.filesWithChanges), grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesWithChanges))))
			logInfo(" - Files with suspect lines           :", warn(stats.filesWithSuspectLines), grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesWithSuspectLines))))
			logInfo(" - Files with only recognized lines   :", green(stats.filesWithoutChanges), grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesWithoutChanges))))
			logInfo(" - Files with custom code             :", stats.filesCustomCode, grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesCustomCode))))
			logInfo(" - Files without", without, "                :", stats.filesNoCode, grey(fmt.Sprintf("%8.2f%%", stats.percentage(stats.filesNoCode))))
			logVerbose("Undetected paths:")
			for _, p := range stats.undetectedPaths {
				logVerbose("  ", p)
			}
		}
	}
}
