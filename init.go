package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/gwillem/urlfilecache"
	"github.com/jessevdk/go-flags"
)

type (
	hashDB map[[16]byte]bool

	walkStats struct {
		totalFiles          int
		filesWithChanges    int
		filesWithoutChanges int
	}

	Args struct {
		Path struct {
			Path []string `positional-arg-name:"<path>"`
		} `positional-args:"yes" description:"Scan file or dir" required:"true"`
		Database string `short:"d" long:"database" description:"Hash database path (default: download Sansec database)"`
		Add      bool   `short:"a" long:"add" description:"Add new hashes to DB, do not check"`
		Full     bool   `short:"f" long:"full" description:"Scan everything, not just core paths."`
		Verbose  bool   `short:"v" long:"verbose" description:"Show what is going on"`
	}
)

const (
	hashDBURL = "https://sansec.io/ext/files/corediff.bin"
)

var (
	boldred   = color.New(color.FgHiRed, color.Bold).SprintFunc()
	grey      = color.New(color.FgHiBlack).SprintFunc()
	boldwhite = color.New(color.FgHiWhite).SprintFunc()
	alarm     = color.New(color.FgHiWhite, color.BgHiRed, color.Bold).SprintFunc()
	green     = color.New(color.FgGreen).SprintFunc()

	logLevel = 1

	globalDB hashDB

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
	}

	highlightPatterns = []string{
		`\$_[A-Z]`,
		`["']\s*\.\s*['"]`,
		`die\(`,
		`base64_`,
		`@(unlink|include|mysql)`,
		`../../..`,
		`hex2bin`,
		`fopen`,
		`file_put_contents`,
		`file_get_contents`,
	}
)

func setup() *Args {

	var err error
	color.NoColor = false

	args := &Args{}
	argParser := flags.NewParser(args, flags.HelpFlag|flags.PrintErrors|flags.PassDoubleDash)
	if _, err := argParser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrRequired {
		} else {
			// log.Fatal(err)
			// fmt.Println("Config parse error:", err)
		}
		os.Exit(1)
	}

	if args.Verbose {
		logLevel = 3
	}

	if args.Database == "" {
		args.Database = urlfilecache.ToPath(hashDBURL)
	}

	for i, path := range args.Path.Path {
		path, err = filepath.Abs(path)
		check(err)
		path, err = filepath.EvalSymlinks(path)
		check(err)

		if !pathExists(path) {
			fmt.Println("Path", path, "does not exist?")
			os.Exit(1)
		}

		if !args.Full && !isCmsRoot(path) {
			fmt.Println("!!!", path)
			fmt.Println("Path does not seem to be an application root path, so we cannot check official root paths.")
			fmt.Println("Try again with proper root path, or do a full scan with --full")
			os.Exit(1)
		}

		args.Path.Path[i] = path
	}

	// postprocessing

	// Basic check for application root?

	return args
}
