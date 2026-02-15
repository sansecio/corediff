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

	git "github.com/go-git/go-git/v5"
	"github.com/gwillem/corediff/internal/composer"
	"github.com/gwillem/corediff/internal/gitindex"
	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/manifest"
	"github.com/gwillem/corediff/internal/normalize"
	"github.com/gwillem/corediff/internal/packagist"
	cdpath "github.com/gwillem/corediff/internal/path"
)

type dbAddArg struct {
	Packagist    string `short:"p" long:"packagist" description:"Index Packagist package (vendor/package)"`
	Composer     string `long:"composer" description:"Index all packages from composer.json + lock"`
	Update       bool   `short:"u" long:"update" description:"Re-check all previously indexed packages for new versions"`
	IgnorePaths  bool   `short:"i" long:"ignore-paths" description:"Don't store file paths in DB."`
	AllValidText bool   `short:"t" long:"text" description:"Scan all valid UTF-8 text files."`
	NoPlatform   bool   `long:"no-platform" description:"Don't check for app root."`
	Path         struct {
		Path []string `positional-arg-name:"<path>"`
	} `positional-args:"yes"`
}

func (a *dbAddArg) Execute(_ []string) error {
	// Mutual exclusion: only one of --packagist, --composer, --update, or <path>
	modes := 0
	if a.Packagist != "" {
		modes++
	}
	if a.Composer != "" {
		modes++
	}
	if a.Update {
		modes++
	}
	if len(a.Path.Path) > 0 {
		modes++
	}
	if modes > 1 {
		return fmt.Errorf("cannot combine --packagist, --composer, --update, and <path>; use only one")
	}
	if modes == 0 {
		return fmt.Errorf("please provide --packagist, --composer, --update, or at least one <path> argument")
	}

	applyVerbose()

	dbPath := dbCommand.Database
	db, err := hashdb.Open(dbPath)
	if os.IsNotExist(err) {
		db = hashdb.New()
		err = nil
	}
	if err != nil {
		return err
	}

	if a.Packagist != "" || a.Composer != "" || a.Update || (len(a.Path.Path) == 1 && isGitURL(a.Path.Path[0])) {
		mf, mfErr := manifest.Load(manifest.PathFromDB(dbPath))
		if mfErr != nil {
			return fmt.Errorf("loading manifest: %w", mfErr)
		}
		defer mf.Close()

		if len(a.Path.Path) == 1 && isGitURL(a.Path.Path[0]) {
			return a.executeGitURL(a.Path.Path[0], db, dbPath, mf)
		}
		if a.Update {
			return a.executeUpdate(db, dbPath, mf)
		}
		if a.Packagist != "" {
			return a.executePackagist(db, dbPath, mf)
		}
		return a.executeComposer(db, dbPath, mf)
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

	opts.Verbose = len(globalOpts.Verbose)

	if len(globalOpts.Verbose) >= 2 {
		logf := func(format string, args ...any) {
			fmt.Println(fmt.Sprintf(format, args...))
		}
		transport = &loggingTransport{base: transport, logf: logf}
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

// indexVersions tries git clone for all versions, falling back to zip per version.
// OnVersionDone (if set in opts) is called after each successfully indexed version.
// Returns the list of packages declared in composer.json "replace" sections (if any).
func (a *dbAddArg) indexVersions(pkg string, versions []packagist.Version, db *hashdb.HashDB, opts gitindex.IndexOptions) []string {
	if len(versions) == 0 {
		return nil
	}

	// Try git source if available, fall back to zip
	useZip := true
	if versions[0].Source.Type == "git" {
		repoURL := versions[0].Source.URL
		refs := make(map[string]string, len(versions))
		for _, v := range versions {
			if v.Source.Reference != "" {
				refs[v.Version] = v.Source.Reference
			}
		}

		var result *gitindex.IndexResult
		var gitErr error
		if dbCommand.CacheDir != "" {
			cloneDir := filepath.Join(dbCommand.CacheDir, "git", sanitizePath(pkg))
			if err := os.MkdirAll(cloneDir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "warning: creating cache dir for %s: %v\n", pkg, err)
			} else {
				result, gitErr = gitindex.CloneAndIndexWithDir(repoURL, cloneDir, refs, db, opts)
			}
		} else {
			result, gitErr = gitindex.CloneAndIndex(repoURL, refs, db, opts)
		}
		if gitErr != nil {
			fmt.Fprintf(os.Stderr, "warning: git clone failed for %s: %v, falling back to zip\n", pkg, gitErr)
		} else if result != nil {
			return result.Replaces
		}
	}

	if useZip {
		for _, v := range versions {
			if v.Dist.URL == "" {
				continue
			}
			logVerbose(fmt.Sprintf("  downloading %s (%s)", v.Version, v.Dist.URL))
			if err := gitindex.IndexZip(v.Dist.URL, db, opts); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping %s %s: %v\n", pkg, v.Version, err)
			} else if opts.OnVersionDone != nil {
				opts.OnVersionDone(v.Version)
			}
		}
	}
	return nil
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
	a.indexVersions(pkg, versions, db, opts)
	return nil
}

func (a *dbAddArg) executePackagist(db *hashdb.HashDB, dbPath string, mf *manifest.Manifest) error {
	// Parse optional version pin: "vendor/pkg:1.2.3" or "vendor/pkg@1.2.3"
	pkg := a.Packagist
	var pinVersion string
	if idx := strings.LastIndexAny(pkg, ":@"); idx > 0 {
		pkg, pinVersion = pkg[:idx], pkg[idx+1:]
	}

	// Track bare (unpinned) packages for automatic updates
	if pinVersion == "" {
		if err := mf.MarkTracked(pkg); err != nil {
			return fmt.Errorf("marking tracked: %w", err)
		}
	}

	opts := gitindex.IndexOptions{
		NoPlatform:   a.NoPlatform,
		AllValidText: a.AllValidText,
		CacheDir:     dbCommand.CacheDir,
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

	// Filter out already-indexed versions
	total := len(versions)
	var newVersions []packagist.Version
	for _, v := range versions {
		if !mf.IsIndexed(pkg, v.Version) {
			newVersions = append(newVersions, v)
		}
	}
	versions = newVersions

	if skipped := total - len(versions); skipped > 0 {
		fmt.Printf("Skipping %d already-indexed versions for %s\n", skipped, pkg)
	}
	if len(versions) == 0 {
		fmt.Printf("All %d versions of %s already indexed\n", total, pkg)
		return nil
	}

	logVerbose(fmt.Sprintf("Indexing %d new versions for %s", len(versions), pkg))

	oldSize := db.Len()
	opts.PathPrefix = "vendor/" + pkg + "/"
	opts.OnVersionDone = func(version string) {
		if err := mf.MarkIndexed(pkg, version); err != nil {
			fmt.Fprintf(os.Stderr, "warning: manifest write: %v\n", err)
		}
	}

	replaces := a.indexVersions(pkg, versions, db, opts)
	for _, r := range replaces {
		if err := mf.MarkReplaced(r); err != nil {
			fmt.Fprintf(os.Stderr, "warning: manifest write: %v\n", err)
		}
	}
	if len(replaces) > 0 {
		fmt.Printf("Recorded %d replaced packages in manifest\n", len(replaces))
	}

	newHashes := db.Len() - oldSize
	if newHashes > 0 {
		fmt.Printf("Computed %d new hashes, saving to %s ..\n", newHashes, dbPath)
		return db.Save(dbPath)
	}
	fmt.Println("Found no new code hashes...")
	return nil
}

func (a *dbAddArg) executeComposer(db *hashdb.HashDB, dbPath string, mf *manifest.Manifest) error {
	proj, err := composer.ParseProject(a.Composer)
	if err != nil {
		return err
	}

	opts := gitindex.IndexOptions{
		NoPlatform:   a.NoPlatform,
		AllValidText: a.AllValidText,
		CacheDir:     dbCommand.CacheDir,
	}

	httpClient, err := a.buildHTTPClient(&opts)
	if err != nil {
		return err
	}

	// Merge repositories from global ~/.composer/config.json
	configRepos, err := composer.FindConfigRepos()
	if err != nil {
		return fmt.Errorf("loading composer config: %w", err)
	}
	if len(configRepos) > 0 {
		var urls []string
		for _, r := range configRepos {
			urls = append(urls, r.URL)
		}
		fmt.Printf("Loaded composer config repos: %s\n", strings.Join(urls, ", "))
		proj.Repos = append(proj.Repos, configRepos...)
	}

	// Filter out already-indexed and replaced packages
	var newPkgs []composer.LockPackage
	var skipped, replaced int
	for _, pkg := range proj.Packages {
		if mf.IsIndexed(pkg.Name, pkg.Version) {
			skipped++
		} else if mf.IsReplaced(pkg.Name) {
			replaced++
		} else {
			newPkgs = append(newPkgs, pkg)
		}
	}

	fmt.Printf("Found %d packages across %d repositories", len(proj.Packages), len(proj.Repos))
	if skipped > 0 || replaced > 0 {
		fmt.Printf(" (")
		parts := []string{}
		if skipped > 0 {
			parts = append(parts, fmt.Sprintf("%d already indexed", skipped))
		}
		if replaced > 0 {
			parts = append(parts, fmt.Sprintf("%d replaced by monorepo", replaced))
		}
		fmt.Printf("%s)", strings.Join(parts, ", "))
	}
	fmt.Println()

	if len(newPkgs) == 0 {
		fmt.Println("All packages already indexed")
		return nil
	}

	// Install go-git HTTP transport once before concurrent operations
	opts.InstallHTTPTransport()

	oldSize := db.Len()

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, runtime.GOMAXPROCS(0))
	)

	for _, pkg := range newPkgs {
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

			if err := mf.MarkIndexed(pkg.Name, pkg.Version); err != nil {
				fmt.Fprintf(os.Stderr, "warning: manifest write: %v\n", err)
			}
		}()
	}

	wg.Wait()

	newHashes := db.Len() - oldSize
	if newHashes > 0 {
		fmt.Printf("Computed %d new hashes, saving to %s ..\n", newHashes, dbPath)
		return db.Save(dbPath)
	}
	fmt.Println("Found no new code hashes...")
	return nil
}

func (a *dbAddArg) executeUpdate(db *hashdb.HashDB, dbPath string, mf *manifest.Manifest) error {
	pkgs := mf.TrackedPackages()
	if len(pkgs) == 0 {
		return fmt.Errorf("no tracked packages — nothing to update. Add packages with --packagist or a git URL first")
	}

	// Partition into git URLs and packagist package names.
	// Skip replaced packages — they're provided by a monorepo.
	var gitURLs, packagistPkgs []string
	var replaced int
	for _, pkg := range pkgs {
		if isGitURL(pkg) {
			gitURLs = append(gitURLs, pkg)
		} else if mf.IsReplaced(pkg) {
			replaced++
		} else {
			packagistPkgs = append(packagistPkgs, pkg)
		}
	}

	fmt.Printf("Checking %d packages for new versions", len(packagistPkgs)+len(gitURLs))
	if len(gitURLs) > 0 || replaced > 0 {
		parts := []string{}
		if len(packagistPkgs) > 0 {
			parts = append(parts, fmt.Sprintf("%d packagist", len(packagistPkgs)))
		}
		if len(gitURLs) > 0 {
			parts = append(parts, fmt.Sprintf("%d git", len(gitURLs)))
		}
		if replaced > 0 {
			parts = append(parts, fmt.Sprintf("%d replaced, skipped", replaced))
		}
		fmt.Printf(" (%s)", strings.Join(parts, ", "))
	}
	fmt.Println("...")

	opts := gitindex.IndexOptions{
		NoPlatform:   a.NoPlatform,
		AllValidText: a.AllValidText,
		CacheDir:     dbCommand.CacheDir,
	}

	httpClient, err := a.buildHTTPClient(&opts)
	if err != nil {
		return err
	}

	// Install go-git HTTP transport once before concurrent operations
	opts.InstallHTTPTransport()

	oldSize := db.Len()

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, runtime.GOMAXPROCS(0))
	)

	for _, pkg := range packagistPkgs {
		sem <- struct{}{} // acquire slot
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			c := &packagist.Client{HTTP: httpClient}
			versions, err := c.Versions(pkg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: fetching versions for %s: %v\n", pkg, err)
				return
			}

			// Filter to only new versions
			var newVersions []packagist.Version
			for _, v := range versions {
				if !mf.IsIndexed(pkg, v.Version) {
					newVersions = append(newVersions, v)
				}
			}

			if len(newVersions) == 0 {
				logVerbose(fmt.Sprintf("  %s: up to date", pkg))
				return
			}

			fmt.Printf("  %s: %d new versions\n", pkg, len(newVersions))

			pkgOpts := opts
			pkgOpts.PathPrefix = "vendor/" + pkg + "/"
			pkgOpts.OnVersionDone = func(version string) {
				if err := mf.MarkIndexed(pkg, version); err != nil {
					fmt.Fprintf(os.Stderr, "warning: manifest write: %v\n", err)
				}
			}

			pkgDB := hashdb.New()
			replaces := a.indexVersions(pkg, newVersions, pkgDB, pkgOpts)
			for _, r := range replaces {
				if err := mf.MarkReplaced(r); err != nil {
					fmt.Fprintf(os.Stderr, "warning: manifest write: %v\n", err)
				}
			}

			mu.Lock()
			db.Merge(pkgDB)
			mu.Unlock()
		}()
	}

	for _, url := range gitURLs {
		sem <- struct{}{} // acquire slot
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release slot

			pkgDB := hashdb.New()
			a.updateGitURLEntry(url, pkgDB, mf, opts)

			mu.Lock()
			db.Merge(pkgDB)
			mu.Unlock()
		}()
	}

	wg.Wait()

	newHashes := db.Len() - oldSize
	if newHashes > 0 {
		fmt.Printf("Computed %d new hashes, saving to %s ..\n", newHashes, dbPath)
		return db.Save(dbPath)
	}
	fmt.Println("All packages up to date, no new hashes")
	return nil
}

// updateGitURLEntry fetches new tags from a git URL and indexes any versions
// not yet in the manifest. Used by executeUpdate for git URL manifest entries.
func (a *dbAddArg) updateGitURLEntry(url string, db *hashdb.HashDB, mf *manifest.Manifest, opts gitindex.IndexOptions) {
	var cloneDir string
	if dbCommand.CacheDir != "" {
		cloneDir = filepath.Join(dbCommand.CacheDir, "git", sanitizePath(url))
		if err := os.MkdirAll(cloneDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "warning: creating cache dir for %s: %v\n", url, err)
			return
		}
	} else {
		tmp, err := os.MkdirTemp("", "corediff-git-*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: creating temp dir for %s: %v\n", url, err)
			return
		}
		defer os.RemoveAll(tmp)
		cloneDir = tmp
	}

	repo, refs, err := gitindex.RefsFromTags(url, cloneDir, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: fetching tags for %s: %v\n", url, err)
		return
	}

	// Filter out already-indexed versions.
	total := len(refs)
	for version := range refs {
		if mf.IsIndexed(url, version) {
			delete(refs, version)
		}
	}

	if len(refs) == 0 {
		logVerbose(fmt.Sprintf("  %s: up to date (%d versions)", url, total))
		return
	}

	fmt.Printf("  %s: %d new versions\n", url, len(refs))

	// Detect composer package name for path prefix.
	pkgOpts := opts
	if !opts.NoPlatform && opts.PathPrefix == "" {
		pkgOpts.PathPrefix = readComposerPathPrefix(repo)
	}

	pkgOpts.OnVersionDone = func(version string) {
		if err := mf.MarkIndexed(url, version); err != nil {
			fmt.Fprintf(os.Stderr, "warning: manifest write: %v\n", err)
		}
	}
	subPkgSet := make(map[string]struct{})
	pkgOpts.OnSubPackage = func(name, version string) {
		if version != "" {
			subPkgSet[name+"@"+version] = struct{}{}
		}
	}

	pkgOpts.CollectLockDeps = true

	result, err := gitindex.IndexRepo(repo, refs, db, pkgOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: indexing %s: %v\n", url, err)
		return
	}

	if len(subPkgSet) > 0 {
		fmt.Printf("Indexed %d embedded packages\n", len(subPkgSet))
	}

	// Also write replaces found across all indexed versions (may overlap with HEAD).
	for _, r := range result.Replaces {
		if err := mf.MarkReplaced(r); err != nil {
			fmt.Fprintf(os.Stderr, "warning: manifest write: %v\n", err)
		}
	}

	// Index dependencies from composer.lock files across all versions.
	if len(result.LockDeps) > 0 {
		// Group deps by package name for efficient indexing (single clone per package).
		depsByPkg := make(map[string][]composer.LockPackage)
		for _, dep := range result.LockDeps {
			if !mf.IsIndexed(dep.Name, dep.Version) && !mf.IsReplaced(dep.Name) {
				depsByPkg[dep.Name] = append(depsByPkg[dep.Name], dep)
			}
		}

		if len(depsByPkg) > 0 {
			totalVersions := 0
			for _, deps := range depsByPkg {
				totalVersions += len(deps)
			}
			fmt.Printf("Found %d dependency packages (%d versions) from composer.lock files\n",
				len(depsByPkg), totalVersions)

			for pkgName, deps := range depsByPkg {
				versions := make([]packagist.Version, 0, len(deps))
				for _, dep := range deps {
					versions = append(versions, lockToVersion(dep))
				}

				depOpts := opts // copy base opts (no callbacks from parent)
				depOpts.PathPrefix = "vendor/" + pkgName + "/"
				depOpts.CollectLockDeps = false // don't recurse
				depOpts.OnVersionDone = func(version string) {
					if err := mf.MarkIndexed(pkgName, version); err != nil {
						fmt.Fprintf(os.Stderr, "warning: manifest write: %v\n", err)
					}
				}
				depOpts.OnSubPackage = nil

				a.indexVersions(pkgName, versions, db, depOpts)
			}
		}
	}
}

// readComposerPathPrefix reads the "name" field from HEAD's composer.json
// and returns a vendor path prefix (e.g. "vendor/magento/magento2ce/").
// Returns empty string if the name cannot be determined.
func readComposerPathPrefix(repo *git.Repository) string {
	head, err := repo.Head()
	if err != nil {
		return ""
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return ""
	}
	tree, err := commit.Tree()
	if err != nil {
		return ""
	}
	f, err := tree.File("composer.json")
	if err != nil {
		return ""
	}
	content, err := f.Contents()
	if err != nil {
		return ""
	}
	if name := composer.ParseName([]byte(content)); name != "" {
		return "vendor/" + name + "/"
	}
	return ""
}

// indexComposerPackage indexes a single lock file package using source/dist/repo fallback.
func (a *dbAddArg) indexComposerPackage(pkg composer.LockPackage, repos []composer.Repository, httpClient *http.Client, db *hashdb.HashDB, opts gitindex.IndexOptions) {
	fmt.Printf("Indexing %s (%s)\n", pkg.Name, pkg.Version)
	opts.PathPrefix = "vendor/" + pkg.Name + "/"

	// Convert lock file source/dist to packagist.Version for indexVersions
	if pkg.Source.URL != "" || pkg.Dist.URL != "" {
		v := lockToVersion(pkg)
		a.indexVersions(pkg.Name, []packagist.Version{v}, db, opts)
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

// lockToVersion converts a composer lock package to a packagist.Version.
func lockToVersion(pkg composer.LockPackage) packagist.Version {
	var v packagist.Version
	v.Version = pkg.Version
	v.Source.Type = pkg.Source.Type
	v.Source.URL = pkg.Source.URL
	v.Source.Reference = pkg.Source.Reference
	v.Dist.Type = pkg.Dist.Type
	v.Dist.URL = pkg.Dist.URL
	v.Dist.Reference = pkg.Dist.Reference
	return v
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

// isGitURL returns true if s looks like a git URL (contains "://" or starts with "git@").
func isGitURL(s string) bool {
	return strings.Contains(s, "://") || strings.HasPrefix(s, "git@")
}

func (a *dbAddArg) executeGitURL(url string, db *hashdb.HashDB, dbPath string, mf *manifest.Manifest) error {
	opts := gitindex.IndexOptions{
		NoPlatform:   a.NoPlatform,
		AllValidText: a.AllValidText,
		CacheDir:     dbCommand.CacheDir,
	}

	if _, err := a.buildHTTPClient(&opts); err != nil {
		return err
	}

	if err := mf.MarkTracked(url); err != nil {
		return fmt.Errorf("marking tracked: %w", err)
	}

	oldSize := db.Len()
	a.updateGitURLEntry(url, db, mf, opts)

	newHashes := db.Len() - oldSize
	if newHashes > 0 {
		fmt.Printf("Computed %d new hashes, saving to %s ..\n", newHashes, dbPath)
		return db.Save(dbPath)
	}
	fmt.Println("Found no new code hashes...")
	return nil
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
