package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/gwillem/corediff/internal/composer"
	"github.com/gwillem/corediff/internal/gitindex"
	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
	"github.com/gwillem/corediff/internal/packagist"
	cdpath "github.com/gwillem/corediff/internal/path"
)

type dbAddArg struct {
	Packagist    string `short:"p" long:"packagist" description:"Index Packagist package (vendor/package)"`
	Composer     string `long:"composer" description:"Index all packages from composer.json + lock"`
	IgnorePaths  bool   `short:"i" long:"ignore-paths" description:"Don't store file paths in DB."`
	AllValidText bool   `short:"t" long:"text" description:"Scan all valid UTF-8 text files."`
	NoPlatform   bool   `long:"no-platform" description:"Don't check for app root."`
	Path         struct {
		Path []string `positional-arg-name:"<path>"`
	} `positional-args:"yes"`
}

func (a *dbAddArg) Execute(_ []string) error {
	// Mutual exclusion: only one of --packagist, --composer, or <path>
	modes := 0
	if a.Packagist != "" {
		modes++
	}
	if a.Composer != "" {
		modes++
	}
	if len(a.Path.Path) > 0 {
		modes++
	}
	if modes > 1 {
		return fmt.Errorf("cannot combine --packagist, --composer, and <path>; use only one")
	}
	if modes == 0 {
		return fmt.Errorf("please provide --packagist, --composer, or at least one <path> argument")
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
	if a.Composer != "" {
		return a.executeComposer(db, dbPath)
	}

	return a.executeLocalPaths(db, dbPath)
}

// authTransport wraps an http.RoundTripper and applies Composer auth headers.
type authTransport struct {
	base http.RoundTripper
	auth *composer.AuthConfig
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.auth.ApplyAuth(req)
	return t.base.RoundTrip(req)
}

// buildHTTPClient constructs an HTTP client with the auth + logging transport chain.
// It also populates opts.Logf / opts.LineLogf based on verbosity.
func (a *dbAddArg) buildHTTPClient(opts *gitindex.IndexOptions) (*http.Client, error) {
	var transport http.RoundTripper = http.DefaultTransport

	if len(globalOpts.Verbose) >= 2 {
		logf := func(format string, args ...any) {
			fmt.Println(fmt.Sprintf(format, args...))
		}
		transport = &loggingTransport{base: transport, logf: logf}
		if len(globalOpts.Verbose) >= 3 {
			opts.Logf = logf
		}
		if len(globalOpts.Verbose) >= 4 {
			opts.LineLogf = logf
		}
	}

	authCfg, err := composer.FindAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("loading composer auth: %w", err)
	}
	if authCfg != nil {
		if hosts := authCfg.Hosts(); len(hosts) > 0 {
			fmt.Printf("Loaded composer auth for: %s\n", strings.Join(hosts, ", "))
		}
		transport = &authTransport{base: transport, auth: authCfg}
	} else {
		fmt.Println("No composer auth.json found")
	}

	if transport != http.DefaultTransport {
		c := &http.Client{Transport: transport}
		opts.HTTP = c
		return c, nil
	}
	return nil, nil
}

// indexPackage fetches versions for pkg from repoURL and indexes them into db.
func (a *dbAddArg) indexPackage(pkg, repoURL string, httpClient *http.Client, db *hashdb.HashDB, opts gitindex.IndexOptions) error {
	c := &packagist.Client{BaseURL: repoURL, HTTP: httpClient}

	versions, err := c.Versions(pkg)
	if err != nil {
		return fmt.Errorf("fetching versions for %s: %w", pkg, err)
	}

	logVerbose(fmt.Sprintf("Found %d versions for %s", len(versions), pkg))

	opts.PathPrefix = "vendor/" + pkg + "/"

	// Try git source if available, fall back to zip
	useZip := true
	if len(versions) > 0 && versions[0].Source.Type == "git" {
		repoURL := versions[0].Source.URL
		refs := make(map[string]string, len(versions))
		for _, v := range versions {
			if v.Source.Reference != "" {
				refs[v.Version] = v.Source.Reference
			}
		}

		logVerbose(fmt.Sprintf("  cloning %s", repoURL))

		var gitErr error
		if dbCommand.CacheDir != "" {
			cloneDir := filepath.Join(dbCommand.CacheDir, "git", sanitizePath(pkg))
			if err := os.MkdirAll(cloneDir, 0o755); err != nil {
				return fmt.Errorf("creating cache dir: %w", err)
			}
			gitErr = gitindex.CloneAndIndexWithDir(repoURL, cloneDir, refs, db, opts)
		} else {
			gitErr = gitindex.CloneAndIndex(repoURL, refs, db, opts)
		}
		if gitErr != nil {
			fmt.Fprintf(os.Stderr, "warning: git clone failed: %v, falling back to zip\n", gitErr)
		} else {
			useZip = false
		}
	}

	if useZip {
		for _, v := range versions {
			if v.Dist.URL == "" {
				continue
			}
			logVerbose(fmt.Sprintf("  downloading %s (%s)", v.Version, v.Dist.URL))
			if err := gitindex.IndexZip(v.Dist.URL, db, opts); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", v.Version, err)
			}
		}
	}

	return nil
}

func (a *dbAddArg) executePackagist(db *hashdb.HashDB, dbPath string) error {
	// Parse optional version pin: "vendor/pkg:1.2.3" or "vendor/pkg@1.2.3"
	pkg := a.Packagist
	var pinVersion string
	if idx := strings.LastIndexAny(pkg, ":@"); idx > 0 {
		pkg, pinVersion = pkg[:idx], pkg[idx+1:]
	}

	opts := gitindex.IndexOptions{
		NoPlatform:   a.NoPlatform,
		AllValidText: a.AllValidText,
	}

	httpClient, err := a.buildHTTPClient(&opts)
	if err != nil {
		return err
	}

	c := &packagist.Client{HTTP: httpClient}

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

	logVerbose(fmt.Sprintf("Found %d versions for %s", len(versions), pkg))

	oldSize := db.Len()
	opts.PathPrefix = "vendor/" + pkg + "/"

	// Try git source if available, fall back to zip
	useZip := true
	if len(versions) > 0 && versions[0].Source.Type == "git" {
		repoURL := versions[0].Source.URL
		refs := make(map[string]string, len(versions))
		for _, v := range versions {
			if v.Source.Reference != "" {
				refs[v.Version] = v.Source.Reference
			}
		}

		logVerbose(fmt.Sprintf("  cloning %s", repoURL))

		var gitErr error
		if dbCommand.CacheDir != "" {
			cloneDir := filepath.Join(dbCommand.CacheDir, "git", sanitizePath(pkg))
			if err := os.MkdirAll(cloneDir, 0o755); err != nil {
				return fmt.Errorf("creating cache dir: %w", err)
			}
			gitErr = gitindex.CloneAndIndexWithDir(repoURL, cloneDir, refs, db, opts)
		} else {
			gitErr = gitindex.CloneAndIndex(repoURL, refs, db, opts)
		}
		if gitErr != nil {
			fmt.Fprintf(os.Stderr, "warning: git clone failed: %v, falling back to zip\n", gitErr)
		} else {
			useZip = false
		}
	}

	if useZip {
		for _, v := range versions {
			if v.Dist.URL == "" {
				continue
			}
			logVerbose(fmt.Sprintf("  downloading %s (%s)", v.Version, v.Dist.URL))
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

func (a *dbAddArg) executeComposer(db *hashdb.HashDB, dbPath string) error {
	proj, err := composer.ParseProject(a.Composer)
	if err != nil {
		return err
	}

	opts := gitindex.IndexOptions{
		NoPlatform:   a.NoPlatform,
		AllValidText: a.AllValidText,
	}

	httpClient, err := a.buildHTTPClient(&opts)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d packages across %d repositories\n", len(proj.Packages), len(proj.Repos))

	// Install go-git HTTP transport once before concurrent operations
	opts.InstallHTTPTransport()

	oldSize := db.Len()

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, runtime.GOMAXPROCS(0))
	)

	for _, pkg := range proj.Packages {
		sem <- struct{}{} // acquire slot
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			pkgDB := hashdb.New()
			a.indexComposerPackage(pkg, proj.Repos, httpClient, pkgDB, opts)

			mu.Lock()
			db.Merge(pkgDB)
			mu.Unlock()
		}()
	}

	wg.Wait()

	db.Compact()
	newHashes := db.Len() - oldSize
	if newHashes > 0 {
		fmt.Printf("Computed %d new hashes, saving to %s ..\n", newHashes, dbPath)
		return db.Save(dbPath)
	}
	fmt.Println("Found no new code hashes...")
	return nil
}

// indexComposerPackage indexes a single lock file package using source/dist/repo fallback.
func (a *dbAddArg) indexComposerPackage(pkg composer.LockPackage, repos []composer.Repository, httpClient *http.Client, db *hashdb.HashDB, opts gitindex.IndexOptions) {
	fmt.Printf("Indexing %s (%s)\n", pkg.Name, pkg.Version)
	opts.PathPrefix = "vendor/" + pkg.Name + "/"

	// Use source/dist from lock file directly — no need to query repos
	if pkg.Source.Type == "git" && pkg.Source.URL != "" {
		refs := map[string]string{pkg.Name: pkg.Source.Reference}

		logVerbose(fmt.Sprintf("  cloning %s (%s)", pkg.Name, pkg.Source.URL))

		var gitErr error
		if dbCommand.CacheDir != "" {
			cloneDir := filepath.Join(dbCommand.CacheDir, "git", sanitizePath(pkg.Name))
			if err := os.MkdirAll(cloneDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "warning: creating cache dir for %s: %v\n", pkg.Name, err)
			} else {
				gitErr = gitindex.CloneAndIndexWithDir(pkg.Source.URL, cloneDir, refs, db, opts)
			}
		} else {
			gitErr = gitindex.CloneAndIndex(pkg.Source.URL, refs, db, opts)
		}
		if gitErr != nil {
			fmt.Fprintf(os.Stderr, "warning: git clone failed for %s: %v, trying zip\n", pkg.Name, gitErr)
		} else {
			return
		}
	}

	// Fall back to dist zip from lock file
	if pkg.Dist.URL != "" {
		logVerbose(fmt.Sprintf("  downloading %s (%s)", pkg.Name, pkg.Dist.URL))
		if err := gitindex.IndexZip(pkg.Dist.URL, db, opts); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", pkg.Name, err)
		}
		return
	}

	// No source/dist in lock — fall back to repo API lookup
	for _, repo := range repos {
		if a.indexPackage(pkg.Name, repo.URL, httpClient, db, opts) == nil {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "warning: package %s not found in any repository\n", pkg.Name)
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

		if n := addFileHashes(path, db); n > 0 {
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
