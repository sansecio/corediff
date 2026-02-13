package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gwillem/corediff/internal/gitindex"
	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
	"github.com/gwillem/corediff/internal/packagist"
	cdpath "github.com/gwillem/corediff/internal/path"
)

type dbAddArg struct {
	Packagist    string `short:"p" long:"packagist" description:"Index Packagist package (vendor/package)"`
	IgnorePaths  bool   `short:"i" long:"ignore-paths" description:"Don't store file paths in DB."`
	AllValidText bool   `short:"t" long:"text" description:"Scan all valid UTF-8 text files."`
	NoPlatform   bool   `long:"no-platform" description:"Don't check for app root."`
	Path         struct {
		Path []string `positional-arg-name:"<path>"`
	} `positional-args:"yes"`
}

func (a *dbAddArg) Execute(_ []string) error {
	if a.Packagist != "" && len(a.Path.Path) > 0 {
		return fmt.Errorf("cannot use --packagist and <path> together")
	}
	if a.Packagist == "" && len(a.Path.Path) == 0 {
		return fmt.Errorf("please provide --packagist or at least one <path> argument")
	}

	applyVerbose()

	dbPath := dbCommand.Database
	db, err := hashdb.OpenReadWrite(dbPath)
	if os.IsNotExist(err) {
		db = hashdb.New()
		err = nil
	}
	if err != nil {
		return err
	}

	if a.Packagist != "" {
		return a.executePackagist(db, dbPath)
	}

	return a.executeLocalPaths(db, dbPath)
}

func (a *dbAddArg) executePackagist(db *hashdb.HashDB, dbPath string) error {
	c := &packagist.Client{}

	// Parse optional version pin: "vendor/pkg:1.2.3" or "vendor/pkg@1.2.3"
	pkg := a.Packagist
	var pinVersion string
	if idx := strings.LastIndexAny(pkg, ":@"); idx > 0 {
		pkg, pinVersion = pkg[:idx], pkg[idx+1:]
	}

	opts := gitindex.IndexOptions{
		NoPlatform:   a.NoPlatform,
		AllValidText: a.AllValidText,
		PathPrefix:   "vendor/" + pkg + "/",
	}
	if len(globalOpts.Verbose) >= 2 {
		logf := func(format string, args ...any) {
			fmt.Println(fmt.Sprintf(format, args...))
		}
		opts.Logf = logf
		if len(globalOpts.Verbose) >= 3 {
			opts.LineLogf = logf
		}
		httpClient := newLoggingHTTPClient(logf)
		opts.HTTP = httpClient
		c.HTTP = httpClient
	}

	versions, err := c.Versions(pkg)
	if err != nil {
		return fmt.Errorf("fetching versions for %s: %w", pkg, err)
	}

	if pinVersion != "" {
		var filtered []packagist.Version
		for _, v := range versions {
			if v.Version == pinVersion {
				filtered = append(filtered, v)
				break
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("version %q not found for %s", pinVersion, pkg)
		}
		versions = filtered
	}

	fmt.Printf("Found %d versions for %s\n", len(versions), pkg)

	oldSize := db.Len()

	// Try git source if available
	if len(versions) > 0 && versions[0].Source.Type == "git" {
		repoURL := versions[0].Source.URL
		refs := make(map[string]string, len(versions))
		for _, v := range versions {
			if v.Source.Reference != "" {
				refs[v.Version] = v.Source.Reference
			}
		}

		fmt.Printf("Cloning %s ...\n", repoURL)

		if dbCommand.CacheDir != "" {
			// Use persistent cache directory
			cloneDir := filepath.Join(dbCommand.CacheDir, "git", sanitizePath(pkg))
			if err := os.MkdirAll(cloneDir, 0o755); err != nil {
				return fmt.Errorf("creating cache dir: %w", err)
			}
			err = gitindex.CloneAndIndexWithDir(repoURL, cloneDir, refs, db, opts)
		} else {
			err = gitindex.CloneAndIndex(repoURL, refs, db, opts)
		}
		if err != nil {
			return fmt.Errorf("git indexing %s: %w", repoURL, err)
		}
	} else {
		// Fallback to zip for each version
		for _, v := range versions {
			if v.Dist.URL == "" {
				continue
			}
			fmt.Printf("  downloading %s (%s)\n", v.Version, v.Dist.URL)
			if err := gitindex.IndexZip(v.Dist.URL, db, opts); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", v.Version, err)
			}
		}
	}

	db.Compact()
	newHashes := db.Len() - oldSize
	if newHashes > 0 {
		fmt.Printf("Computed %d new hashes, saving to %s ..\n", newHashes, dbPath)
		return db.Save(dbPath)
	}
	fmt.Println("Found no new code hashes...")
	return nil
}

func (a *dbAddArg) executeLocalPaths(db *hashdb.HashDB, dbPath string) error {
	for _, p := range a.Path.Path {
		fi, fiErr := os.Stat(p)
		if fiErr != nil {
			return fmt.Errorf("error stat'ing %q: %w", p, fiErr)
		}
		if fi.IsDir() && !a.NoPlatform && !a.IgnorePaths && !cdpath.IsAppRoot(p) {
			return fmt.Errorf("path %q does not seem to be an application root path. Try again with proper root path, or use --no-platform", p)
		}
	}

	oldSize := db.Len()
	for _, p := range a.Path.Path {
		fmt.Println("Calculating checksums for", p)
		addPath(p, db, a.IgnorePaths, a.AllValidText, a.NoPlatform)
		fmt.Println()
	}

	db.Compact()
	if db.Len() != oldSize {
		fmt.Println("Computed", db.Len()-oldSize, "new hashes, saving to", dbPath, "..")
		return db.Save(dbPath)
	}
	fmt.Println("Found no new code hashes...")
	return nil
}

// sanitizePath replaces slashes with dashes for safe directory names.
func sanitizePath(s string) string {
	out := make([]byte, len(s))
	for i := range len(s) {
		if s[i] == '/' {
			out[i] = '-'
		} else {
			out[i] = s[i]
		}
	}
	return string(out)
}

func addPath(root string, db *hashdb.HashDB, ignorePaths bool, allValidText bool, noPlatform bool) {
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

		if !ignorePaths && !noPlatform && path != root && !cdpath.IsExcluded(relPath) {
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
