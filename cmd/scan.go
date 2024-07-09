package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gobwas/glob"
	"github.com/gwillem/go-buildversion"
	"github.com/gwillem/go-selfupdate"
	"github.com/gwillem/urlfilecache"
)

type (
	hashDB map[uint64]struct{}

	walkStats struct {
		totalFiles            int
		filesWithSuspectLines int
		filesWithChanges      int
		filesWithoutChanges   int
		filesNoCode           int
		filesCustomCode       int
		undetectedPaths       []string
	}

	scanArg struct {
		Path struct {
			Path []string `positional-arg-name:"<path>" required:"1"`
		} `positional-args:"yes" description:"Scan file or dir" required:"true"`
		Database string `short:"d" long:"database" description:"Hash database path (default: download Sansec database)"`
		// Add          bool   `short:"a" long:"add" description:"Add new hashes to DB, do not check"`
		// Merge        bool   `short:"m" long:"merge" description:"Merge databases"`
		IgnorePaths  bool   `short:"i" long:"ignore-paths" description:"Scan everything, not just core paths."`
		SuspectOnly  bool   `short:"s" long:"suspect" description:"Show suspect code lines only."`
		AllValidText bool   `short:"t" long:"text" description:"Scan all valid UTF-8 text files, instead of just files with valid prefixes."`
		NoCMS        bool   `long:"no-cms" description:"Don't check for CMS root when adding hashes. Do add file paths."`
		Verbose      bool   `short:"v" long:"verbose" description:"Show what is going on"`
		PathFilter   string `short:"f" long:"path-filter" description:"Applies a path filter prior to diffing (e.g. vendor/magento)"`
	}
)

var (
	scanCmd scanArg

	selfUpdateURL   = fmt.Sprintf("https://sansec.io/downloads/%s-%s/corediff", runtime.GOOS, runtime.GOARCH)
	corediffVersion = buildversion.String()

	scanExts  = []string{"php", "phtml", "js", "htaccess", "sh"}
	skipLines = [][]byte{
		[]byte("*"),
		[]byte("/*"),
		[]byte("//"),
		[]byte("#"),
	}

	cmsPaths = []string{
		"/app/etc/local.xml",
		"/app/etc/env.php",
		"/wp-config.php",
		"/lib/internal/Magento",
		"/app/design/frontend/Magento",
	}

	// They vary often, so add these to core paths when adding signatures
	// However, do process their contents, so files can be inspected with
	// corediff --ignore-paths
	excludePaths = []glob.Glob{
		// "vendor/composer/**",
		glob.MustCompile("vendor/composer/autoload_*.php"),
		glob.MustCompile("generated/**"),
		glob.MustCompile("var/**"),
	}
)

const (
	defaultHashDBURL = "https://sansec.io/downloads/corediff-db/corediff.bin"
	maxTokenSize     = 1024 * 1024 * 10 // 10 MB
)

func (s *scanArg) Execute(_ []string) error {
	if restarted, err := selfupdate.UpdateRestart(selfUpdateURL); restarted || err != nil {
		logVerbose("Restarted new version", restarted, "with error:", err)
	}
	s.validate()

	db, err := loadDB(s.Database)
	if err != nil {
		log.Fatal("Error loading database:", err)
	}

	fmt.Println(boldwhite("Corediff ", corediffVersion, " loaded ", len(db), " precomputed hashes. (C) 2020-2024 labs@sansec.io"))
	fmt.Println("Using database:", s.Database)

	// if s.Merge {
	// 	for _, p := range s.Path.Path {
	// 		db2 := loadDB(p)
	// 		fmt.Println("Merging", filepath.Base(p), "with", len(db2), "entries ..")
	// 		for k := range db2 {
	// 			db[k] = placeholder
	// 		}
	// 	}
	// 	fmt.Println("Saving", s.Database, "with a total of", len(db), "entries.")
	// 	saveDB(s.Database, db)
	// } else if s.Add {
	// 	oldSize := len(db)
	// 	for _, path := range s.Path.Path {
	// 		fmt.Println("Calculating checksums for", path)
	// 		addPath(path, db, s)
	// 		fmt.Println()
	// 	}
	// 	if len(db) != oldSize {
	// 		fmt.Println("Computed", len(db)-oldSize, "new hashes, saving to", s.Database, "..")
	// 		saveDB(s.Database, db)
	// 	} else {
	// 		fmt.Println("Found no new code hashes...")
	// 	}
	// } else {
	without := "code"
	if s.AllValidText {
		without = "text"
	}
	for _, path := range s.Path.Path {
		stats := walkPath(path, db, s)
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

	return nil
}

func init() {
	cli.AddCommand("scan", "Scan file or dir", "Scan file or dir", &scanCmd)
}

func (stats *walkStats) percentage(of int) float64 {
	return float64(of) / float64(stats.totalFiles) * 100
}

func (s *scanArg) validate() error {
	var err error
	if s.Verbose {
		logLevel = 3
	}

	if s.Database == "" {
		s.Database = urlfilecache.ToPath(defaultHashDBURL)
	}

	for i, path := range s.Path.Path {
		if !pathExists(path) {
			return fmt.Errorf("Path %q does not exist", path)
		}

		path, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("Error getting absolute path: %w", err)
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("Error eval'ing symlinks for %q: %w", path, err)
		}

		if !s.IgnorePaths && !s.NoCMS && !isCmsRoot(path) {
			return fmt.Errorf("Path %q does not seem to be an application root path, so we cannot check official root paths. Try again with proper root path, or do a full scan with --ignore-paths", path)
		}

		s.Path.Path[i] = path
	}
	return nil
}

func parseFile(path string, lineCB func([]byte)) error {
	fh, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: " + path)
	} else if err != nil {
		return fmt.Errorf("open error on %s: %s", path, err)
	}
	defer fh.Close()
	return parseFH(fh, lineCB)
}

func parseFH(r io.Reader, lineCB func([]byte)) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)
	for i := 0; scanner.Scan(); i++ {
		lineCB(scanner.Bytes())
	}
	return scanner.Err()
}

func parseFileWithDB(path string, db hashDB, updateDB bool) (hits []int, lines [][]byte) {
	c := 0
	err := parseFile(path, func(line []byte) {
		c++
		h := hash(normalizeLine(line))
		if _, ok := db[h]; !ok {
			hits = append(hits, c)
			lines = append(lines, line) // to show specific line number
			if updateDB {
				db[h] = placeholder
			}
		}
	})
	if err != nil {
		log.Println("err: ", err)
	}

	return hits, lines
}

func walkPath(root string, db hashDB, args *scanArg) *walkStats {
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

		hits, lines := parseFileWithDB(path, db, false)

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

func addPath(root string, db hashDB, args *scanArg) {
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
