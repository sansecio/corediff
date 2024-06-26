package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/gobwas/glob"
	"github.com/gwillem/urlfilecache"
	"github.com/jessevdk/go-flags"
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

	baseArgs struct {
		Path struct {
			Path []string `positional-arg-name:"<path>" required:"1"`
		} `positional-args:"yes" description:"Scan file or dir" required:"true"`
		Database     string `short:"d" long:"database" description:"Hash database path (default: download Sansec database)"`
		Add          bool   `short:"a" long:"add" description:"Add new hashes to DB, do not check"`
		Merge        bool   `short:"m" long:"merge" description:"Merge databases"`
		IgnorePaths  bool   `short:"i" long:"ignore-paths" description:"Scan everything, not just core paths."`
		SuspectOnly  bool   `short:"s" long:"suspect" description:"Show suspect code lines only."`
		AllValidText bool   `short:"t" long:"text" description:"Scan all valid UTF-8 text files, instead of just files with valid prefixes."`
		NoCMS        bool   `long:"no-cms" description:"Don't check for CMS root when adding hashes. Do add file paths."`
		Verbose      bool   `short:"v" long:"verbose" description:"Show what is going on"`
		PathFilter   string `short:"f" long:"path-filter" description:"Applies a path filter prior to diffing (e.g. vendor/magento)"`
	}
)

func (stats *walkStats) percentage(of int) float64 {
	return float64(of) / float64(stats.totalFiles) * 100
}

const (
	hashDBURL    = "https://sansec.io/downloads/corediff-db/corediff.bin"
	maxTokenSize = 1024 * 1024 * 10 // 10 MB
)

var (
	boldred   = color.New(color.FgHiRed, color.Bold).SprintFunc()
	grey      = color.New(color.FgHiBlack).SprintFunc()
	boldwhite = color.New(color.FgHiWhite).SprintFunc()
	warn      = color.New(color.FgYellow, color.Bold).SprintFunc()
	alarm     = color.New(color.FgHiWhite, color.BgHiRed, color.Bold).SprintFunc()
	green     = color.New(color.FgGreen).SprintFunc()

	logLevel = 1

	buf = make([]byte, 0, maxTokenSize)

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

func setup() *baseArgs {
	var err error
	color.NoColor = false

	args := &baseArgs{}
	argParser := flags.NewParser(args, flags.HelpFlag|flags.PrintErrors|flags.PassDoubleDash)
	if _, err := argParser.Parse(); err != nil {
		os.Exit(1)
	}

	if args.Verbose {
		logLevel = 3
	}

	if args.Database == "" {
		if args.Merge {
			fmt.Println("Can't merge without given --database file")
			os.Exit(1)
		}
		// fmt.Println("Using default hash database from", hashDBURL)
		args.Database = urlfilecache.ToPath(hashDBURL)
	}

	for i, path := range args.Path.Path {
		if !pathExists(path) {
			fmt.Println("Path", path, "does not exist?")
			os.Exit(1)
		}

		path, err = filepath.Abs(path)
		if err != nil {
			log.Fatal("Error getting absolute path:", err)
		}
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			log.Fatal("Error eval'ing symlinks for", path, err)
		}

		if !args.Merge && !args.IgnorePaths && !args.NoCMS && !isCmsRoot(path) {
			fmt.Println("!!!", path)
			fmt.Println("Path does not seem to be an application root path, so we cannot check official root paths.")
			fmt.Println("Try again with proper root path, or do a full scan with --ignore-paths")
			os.Exit(1)
		}

		args.Path.Path[i] = path
	}

	return args
}
