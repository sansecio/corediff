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

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/highlight"
	"github.com/gwillem/corediff/internal/normalize"
	cdpath "github.com/gwillem/corediff/internal/path"
	buildversion "github.com/gwillem/go-buildversion"
	selfupdate "github.com/gwillem/go-selfupdate"
	"github.com/gwillem/urlfilecache"
)

type walkStats struct {
	totalFiles            int
	filesWithSuspectLines int
	filesWithChanges      int
	filesWithoutChanges   int
	filesNoCode           int
	filesCustomCode       int
	undetectedPaths       []string
}

type scanArg struct {
	Path struct {
		Path []string `positional-arg-name:"<path>" required:"1"`
	} `positional-args:"yes" description:"Scan file or dir" required:"true"`
	Database     string `short:"d" long:"database" description:"Hash database path (default: download Willem de Groot database)"`
	IgnorePaths  bool   `short:"i" long:"ignore-paths" description:"Scan everything, not just core paths."`
	SuspectOnly  bool   `short:"s" long:"suspect" description:"Show suspect code lines only."`
	AllValidText bool   `short:"t" long:"text" description:"Scan all valid UTF-8 text files, instead of just files with valid prefixes."`
	NoPlatform   bool   `long:"no-platform" description:"Don't check for app root when adding hashes. Do add file paths."`
	PathFilter   string `short:"f" long:"path-filter" description:"Applies a path filter prior to diffing (e.g. vendor/magento)"`
}

var (
	scanCmd scanArg

	selfUpdateURL   = fmt.Sprintf("https://sansec.io/downloads/%s-%s/corediff", runtime.GOOS, runtime.GOARCH)
	corediffVersion = buildversion.String()
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

	db, err := hashdb.Open(s.Database)
	if err != nil {
		log.Fatal("Error loading database:", err)
	}

	fmt.Println(boldwhite("Corediff ", corediffVersion, " loaded ", db.Len(), " precomputed hashes. (C) 2023-2026 Willem de Groot"))
	fmt.Println("Using database:", s.Database)

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
	applyVerbose()

	if s.Database == "" {
		s.Database = urlfilecache.ToPath(defaultHashDBURL)
	}

	for i, path := range s.Path.Path {
		if !cdpath.Exists(path) {
			return fmt.Errorf("path %q does not exist", path)
		}

		path, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("error getting absolute path: %w", err)
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("error eval'ing symlinks for %q: %w", path, err)
		}

		// Skip app root check for single files
		fi, fiErr := os.Stat(path)
		if fiErr != nil {
			return fmt.Errorf("error stat'ing %q: %w", path, fiErr)
		}
		if !fi.IsDir() {
			s.Path.Path[i] = path
			continue
		}

		if !s.IgnorePaths && !s.NoPlatform && !cdpath.IsAppRoot(path) {
			return fmt.Errorf("path %q does not seem to be an application root path, so we cannot check official root paths. Try again with proper root path, or do a full scan with --ignore-paths", path)
		}

		s.Path.Path[i] = path
	}
	return nil
}

func parseFile(path string, lineCB func([]byte)) error {
	fh, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
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

// addFileHashes opens path and adds all line hashes to db using HashReader.
// Returns the number of new hashes added.
func addFileHashes(path string, db *hashdb.HashDB) int {
	fh, err := os.Open(path)
	if err != nil {
		log.Println("err: ", err)
		return 0
	}
	defer fh.Close()
	return normalize.HashReader(fh, db, nil)
}

// scanFileWithDB scans path and returns line numbers and content of lines
// whose hashes are not in db.
func scanFileWithDB(path string, db *hashdb.HashDB) (hits []int, lines [][]byte) {
	c := 0
	err := parseFile(path, func(line []byte) {
		c++
		hashes := normalize.HashLine(line)
		if len(hashes) == 0 {
			return // empty/comment line
		}
		for _, h := range hashes {
			if !db.Contains(h) {
				hits = append(hits, c)
				lines = append(lines, line)
				return
			}
		}
	})
	if err != nil {
		log.Println("err: ", err)
	}
	return hits, lines
}

func walkPath(root string, db *hashdb.HashDB, args *scanArg) *walkStats {
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

		if (!args.AllValidText && !normalize.HasValidExt(path)) || (args.AllValidText && !normalize.IsValidUtf8(path)) {
			stats.filesNoCode++
			return nil
		}

		// Only do path checking for non-root elements
		if path != root && !args.IgnorePaths {
			foundInDb := db.Contains(normalize.PathHash(relPath))
			shouldExclude := cdpath.IsExcluded(relPath)

			if !foundInDb || shouldExclude {
				stats.filesCustomCode++
				logVerbose(grey(" ? ", relPath))
				return nil
			}
		}

		hits, lines := scanFileWithDB(path, db)

		if args.SuspectOnly {
			hitsFiltered := []int{}
			linesFiltered := [][]byte{}
			for i, lineNo := range hits {
				if highlight.ShouldHighlight(lines[i]) {
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
				if highlight.ShouldHighlight(lines[i]) {
					hasSuspectLines = true
					fmt.Println("  ", grey(fmt.Sprintf("%-5d", lineNo)), alarm(string(lines[i])))
				} else if !args.SuspectOnly {
					fmt.Println("  ", grey(fmt.Sprintf("%-5d", lineNo)), string(lines[i]))
				}
			}
			if hasSuspectLines {
				stats.filesWithSuspectLines++
			}
			fmt.Println()
		} else {
			stats.filesWithoutChanges++
			if len(globalOpts.Verbose) >= 1 {
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
