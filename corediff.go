package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"log"

	"github.com/fatih/color"
	"github.com/gwillem/urlfilecache"
	"github.com/jessevdk/go-flags"
)

type hashDB map[[16]byte]bool

const (
	hashDBURL = "https://sansec.io/ext/files/corediff.bin"
)

var (
	boldred   = color.New(color.FgHiRed, color.Bold).SprintFunc()
	grey      = color.New(color.FgHiBlack).SprintFunc()
	boldwhite = color.New(color.FgHiWhite).SprintFunc()
	alarm     = color.New(color.FgHiWhite, color.BgHiRed, color.Bold).SprintFunc()

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

type Args struct {
	Path struct {
		Path string `positional-arg-name:"<path>"`
	} `positional-args:"yes" description:"Scan file or dir" required:"true"`
	Full    bool `short:"f" long:"full" description:"Scan everything, not just core paths."`
	Verbose bool `short:"v" long:"verbose" description:"Show what is going on"`
}

func hash(b []byte) [16]byte {
	return md5.Sum(b)
}

func normalizeLine(b []byte) []byte {
	// Also strip slashes comments etc
	b = bytes.TrimSpace(b)
	for _, prefix := range skipLines {
		if bytes.HasPrefix(b, prefix) {
			return []byte{}
		}
	}
	return b
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func loadDB() hashDB {
	f, err := os.Open(urlfilecache.ToPath(hashDBURL))
	check(err)
	defer f.Close()
	reader := bufio.NewReader(f)
	m := make(hashDB)
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

func shouldHighlight(b []byte) bool {
	for _, p := range highlightPatterns {
		m, _ := regexp.Match(p, b)
		if m {
			return true
		}
	}
	return false
}

func parseFile(path string, db hashDB) {
	fh, err := os.Open(path)
	check(err)
	defer fh.Close()

	hits := []int{}
	lines := [][]byte{}

	maxTokenSize := 1024 * 1024 * 10 // 10MB
	scanner := bufio.NewScanner(fh)
	buf := make([]byte, 0, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)
	for i := 0; scanner.Scan(); i++ {
		x := scanner.Bytes()
		l := make([]byte, len(x))
		copy(l, x)
		lines = append(lines, l)
		h := hash(normalizeLine(l))
		if !db[h] {
			hits = append(hits, i)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if len(hits) > 0 {
		fmt.Println(boldred("\n>>>> " + path))
		for _, idx := range hits {
			// fmt.Println(string(lines[idx]))
			if shouldHighlight(lines[idx]) {
				fmt.Printf("%s %s\n", grey(fmt.Sprintf("%-5d", idx)), alarm(string(lines[idx])))

			} else {
				fmt.Printf("%s %s\n", grey(fmt.Sprintf("%-5d", idx)), string(lines[idx]))
			}
		}
		fmt.Println()
	}
}

func hasValidExt(path string) bool {
	got := strings.TrimLeft(filepath.Ext(path), ".")
	for _, want := range scanExts {
		if got == want {
			return true
		}
	}
	return false
}

func isRelPathInDB(relPath string, db hashDB) bool {
	key := "path:" + relPath
	return db[hash([]byte(key))]
}

func walk(root string, db hashDB, args *Args) {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("failure accessing a path %q: %v\n", path, err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !hasValidExt(path) {
			return nil
		}

		// Only do path checking for non-root elts
		if !args.Full {
			relPath := path[len(root)+1:]
			if !isRelPathInDB(relPath, db) {
				if args.Verbose {
					fmt.Println("Skipping:", relPath)
				}
				return nil
			}
		}

		parseFile(path, db)
		return nil
	})
	check(err)
}

func isCmsRoot(root string) bool {
	for _, testPath := range cmsPaths {
		full := root + testPath
		if pathExists(full) {
			return true
		}
	}
	return false
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
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

	// postprocessing
	args.Path.Path, err = filepath.Abs(args.Path.Path)
	check(err)
	args.Path.Path, err = filepath.EvalSymlinks(args.Path.Path)
	check(err)

	if !pathExists(args.Path.Path) {
		fmt.Println("Path", args.Path.Path, "does not exist?")
		os.Exit(1)
	}

	// Enforce scanning is target is just a single file
	if !isDir(args.Path.Path) {
		args.Full = true
	}

	// Basic check for application root?
	if !args.Full && !isCmsRoot(args.Path.Path) {
		fmt.Println("Path does not seem to be an application root path, so we cannot check official root paths.")
		fmt.Println("Try again with proper root path, or do a full scan with --full")
		os.Exit(1)
	}

	return args
}

func main() {

	args := setup()
	db := loadDB()

	fmt.Print(boldwhite(fmt.Sprintln("\nMagento Corediff loaded", len(db),
		"precomputed hashes. (C) 2020 info@sansec.io")))

	walk(args.Path.Path, db, args)
}
