package gitindex

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitclient "github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/gwillem/corediff/internal/composer"
	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
)

// SubPackage represents a composer sub-package found inside a monorepo tree.
type SubPackage struct {
	Name    string // e.g. "magento/module-catalog"
	Version string // e.g. "104.0.7"
	Dir     string // directory within repo, e.g. "app/code/Magento/Catalog/"
}

// IndexOptions controls how files are indexed.
type IndexOptions struct {
	NoPlatform      bool
	AllValidText    bool
	PathPrefix      string            // prepended to file paths for path hashes (e.g. "vendor/psr/log/")
	Verbose         int               // verbosity level: 1=-v (versions), 3=-vvv (files), 4=-vvvv (lines)
	HTTP            *http.Client      // optional; defaults to http.DefaultClient
	OnVersionDone   func(version string)                // called after each version is indexed
	OnSubPackage    func(name, version string)          // called for each sub-package found in a version
}

// CloneAndIndex bare-clones repoURL, then for each version→ref pair,
// walks the git tree and hashes all eligible files into db.
// Returns the union of package names from composer.json "replace" sections
// across all indexed versions.
func CloneAndIndex(repoURL string, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) ([]string, error) {
	opts.InstallHTTPTransport()

	tmpDir, err := os.MkdirTemp("", "corediff-git-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := git.PlainClone(tmpDir, true, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning %s: %w", repoURL, err)
	}

	replaces := indexRefs(repo, refs, db, opts)
	return replaces, nil
}

// CloneAndIndexWithDir is like CloneAndIndex but uses an existing directory
// for the bare clone. If the directory already contains a valid repo, it reuses it.
func CloneAndIndexWithDir(repoURL, cloneDir string, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) ([]string, error) {
	opts.InstallHTTPTransport()

	var repo *git.Repository
	var err error

	// Try opening existing repo first; fetch to update refs
	repo, err = git.PlainOpen(cloneDir)
	if err != nil {
		repo, err = git.PlainClone(cloneDir, true, &git.CloneOptions{
			URL: repoURL,
		})
		if err != nil {
			return nil, fmt.Errorf("cloning %s: %w", repoURL, err)
		}
	} else {
		err = repo.Fetch(&git.FetchOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, fmt.Errorf("fetching %s: %w", repoURL, err)
		}
	}

	replaces := indexRefs(repo, refs, db, opts)
	return replaces, nil
}

// RefsFromTags clones (or opens) a git repo and returns a map of
// version→commit-hash derived from semver-like tags.
func RefsFromTags(repoURL, cloneDir string, opts IndexOptions) (*git.Repository, map[string]string, error) {
	opts.InstallHTTPTransport()

	var repo *git.Repository
	var err error

	repo, err = git.PlainOpen(cloneDir)
	if err != nil {
		repo, err = git.PlainClone(cloneDir, true, &git.CloneOptions{
			URL: repoURL,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("cloning %s: %w", repoURL, err)
		}
	} else {
		err = repo.Fetch(&git.FetchOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, nil, fmt.Errorf("fetching %s: %w", repoURL, err)
		}
	}

	tags, err := repo.Tags()
	if err != nil {
		return nil, nil, fmt.Errorf("listing tags: %w", err)
	}

	refs := make(map[string]string)
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		if !isVersionTag(name) {
			return nil
		}

		// Resolve annotated tags to the underlying commit
		hash := ref.Hash()
		if tagObj, tErr := repo.TagObject(hash); tErr == nil {
			// Annotated tag — follow to commit
			commit, cErr := tagObj.Commit()
			if cErr == nil {
				hash = commit.Hash
			}
		}

		refs[name] = hash.String()
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("iterating tags: %w", err)
	}

	return repo, refs, nil
}

// IndexRepo indexes an already-opened repo with the given refs.
func IndexRepo(repo *git.Repository, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) ([]string, error) {
	return indexRefs(repo, refs, db, opts), nil
}

// isVersionTag returns true if the tag name looks like a semver version:
// starts with an optional "v" followed by a digit.
func isVersionTag(name string) bool {
	s := strings.TrimPrefix(name, "v")
	if s == "" {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}

// findSubPackages scans a git tree for composer.json files in subdirectories
// and returns the sub-packages found (each with name, version, and directory).
// The root composer.json is skipped.
func findSubPackages(tree *object.Tree) []SubPackage {
	var pkgs []SubPackage
	tree.Files().ForEach(func(f *object.File) error {
		base := f.Name[strings.LastIndex(f.Name, "/")+1:]
		if base != "composer.json" || f.Name == "composer.json" {
			return nil
		}
		content, err := f.Contents()
		if err != nil {
			return nil
		}
		name := composer.ParseName([]byte(content))
		if name == "" {
			return nil
		}
		version := composer.ParseVersion([]byte(content))
		dir := f.Name[:strings.LastIndex(f.Name, "/")+1]
		pkgs = append(pkgs, SubPackage{Name: name, Version: version, Dir: dir})
		return nil
	})
	return pkgs
}

// resolveStoredPath returns the canonical vendor path for a file.
// If the file is inside a sub-package, it uses vendor/<sub-package-name>/...
// Otherwise, it falls back to the default prefix.
func resolveStoredPath(filePath string, subPkgs []SubPackage, defaultPrefix string) string {
	for _, sp := range subPkgs {
		if strings.HasPrefix(filePath, sp.Dir) {
			return "vendor/" + sp.Name + "/" + filePath[len(sp.Dir):]
		}
	}
	return defaultPrefix + filePath
}

func indexRefs(repo *git.Repository, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) []string {
	versions := slices.Collect(maps.Keys(refs))
	slices.SortFunc(versions, cmpVersionDesc)

	// Track blob hashes across versions to skip unchanged files.
	// Versions are processed newest-first; subsequent versions skip blobs
	// already hashed, avoiding redundant I/O and hashing.
	seenBlobs := make(map[plumbing.Hash]struct{})

	// Collect replace entries from composer.json across all versions.
	replaceSet := make(map[string]struct{})

	for _, version := range versions {
		tree, _, err := indexRef(repo, version, refs[version], db, opts, seenBlobs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s (%s): %v\n", version, refs[version][:minLen(refs[version], 12)], err)
			continue
		}

		// Read composer.json from the tree root to extract replace entries.
		if tree != nil {
			if f, fErr := tree.File("composer.json"); fErr == nil {
				if content, cErr := f.Contents(); cErr == nil {
					if pkgs, pErr := composer.ParseReplace([]byte(content)); pErr == nil {
						for _, pkg := range pkgs {
							replaceSet[pkg] = struct{}{}
						}
					}
				}
			}
		}
	}

	return slices.Collect(maps.Keys(replaceSet))
}

func indexRef(repo *git.Repository, version, ref string, db *hashdb.HashDB, opts IndexOptions, seenBlobs map[plumbing.Hash]struct{}) (*object.Tree, []SubPackage, error) {
	commit, err := repo.CommitObject(plumbing.NewHash(ref))
	if err != nil {
		return nil, nil, fmt.Errorf("resolving commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, nil, fmt.Errorf("getting tree: %w", err)
	}

	// Pre-scan for sub-package composer.json files to resolve canonical paths.
	var subPkgs []SubPackage
	if !opts.NoPlatform && opts.PathPrefix != "" {
		subPkgs = findSubPackages(tree)
	}

	var newHashes, totalHashes, skippedFiles int
	start := time.Now()

	err = tree.Files().ForEach(func(f *object.File) error {
		storedPath := resolveStoredPath(f.Name, subPkgs, opts.PathPrefix)
		n, t := indexFileCount(f, storedPath, db, opts, seenBlobs)
		if n == 0 && t == 0 {
			skippedFiles++
		}
		newHashes += n
		totalHashes += t
		return nil
	})

	elapsed := time.Since(start)
	rate := float64(totalHashes) / max(elapsed.Seconds(), 0.001)

	pkg := strings.TrimSuffix(strings.TrimPrefix(opts.PathPrefix, "vendor/"), "/")
	if skippedFiles > 0 {
		opts.log(1, "  indexed %s@%s (%d new, %d total, %d files skipped, %.0f hash/sec)", pkg, version, newHashes, totalHashes, skippedFiles, rate)
	} else {
		opts.log(1, "  indexed %s@%s (%d new, %d total, %.0f hash/sec)", pkg, version, newHashes, totalHashes, rate)
	}

	if err == nil && opts.OnVersionDone != nil {
		opts.OnVersionDone(version)
	}

	// Notify about sub-packages found in this version.
	if err == nil && opts.OnSubPackage != nil {
		for _, sp := range subPkgs {
			opts.OnSubPackage(sp.Name, sp.Version)
		}
	}

	if err != nil {
		return nil, nil, err
	}
	return tree, subPkgs, nil
}

func (opts IndexOptions) log(level int, format string, args ...any) {
	if opts.Verbose >= level {
		fmt.Println(fmt.Sprintf(format, args...))
	}
}

func (opts IndexOptions) httpClient() *http.Client {
	if opts.HTTP != nil {
		return opts.HTTP
	}
	return http.DefaultClient
}

// InstallHTTPTransport registers a custom HTTP client with go-git's protocol
// handlers so clone/fetch requests use the provided client. This modifies
// global state and must be called once before any concurrent git operations.
func (opts IndexOptions) InstallHTTPTransport() {
	if opts.HTTP == nil {
		return
	}
	t := githttp.NewClient(opts.HTTP)
	gitclient.InstallProtocol("https", t)
	gitclient.InstallProtocol("http", t)
}

// indexFileCount indexes a single file and returns (new hashes added, total hashes processed).
// storedPath is the canonical path used for path hashing (e.g. "vendor/magento/module-catalog/Block/Product.php").
// seenBlobs tracks git blob hashes already processed; unchanged files across versions are skipped.
func indexFileCount(f *object.File, storedPath string, db *hashdb.HashDB, opts IndexOptions, seenBlobs map[plumbing.Hash]struct{}) (int, int) {
	if !opts.AllValidText && !normalize.HasValidExt(f.Name) {
		opts.log(3, "    skip %s (no valid ext)", f.Name)
		return 0, 0
	}

	// Skip if this exact blob content was already processed in a previous version.
	if seenBlobs != nil {
		if _, seen := seenBlobs[f.Hash]; seen {
			return 0, 0
		}
	}

	// Check UTF-8 validity by reading first 8KB
	reader, err := f.Blob.Reader()
	if err != nil {
		return 0, 0 // skip unreadable files
	}

	buf := make([]byte, 8*1024)
	n, err := reader.Read(buf)
	reader.Close()
	if err != nil && err != io.EOF {
		return 0, 0
	}
	if !utf8.Valid(buf[:n]) {
		opts.log(3, "    skip %s (invalid utf8)", f.Name)
		if seenBlobs != nil {
			seenBlobs[f.Hash] = struct{}{} // don't re-check in later versions
		}
		return 0, 0
	}

	// Add path hash unless NoPlatform
	if !opts.NoPlatform {
		db.Add(normalize.PathHash(storedPath))
		opts.log(3, "    hash %s", storedPath)
	} else {
		opts.log(3, "    hash %s", f.Name)
	}

	// Re-open reader and hash all lines
	reader, err = f.Blob.Reader()
	if err != nil {
		return 0, 0
	}
	defer reader.Close()

	var lineLogf func(string, ...any)
	if opts.Verbose >= 4 {
		lineLogf = func(format string, args ...any) {
			fmt.Println(fmt.Sprintf(format, args...))
		}
	}
	added, total := normalize.HashReader(reader, db, lineLogf)

	// Mark blob as processed so subsequent versions skip it
	if seenBlobs != nil {
		seenBlobs[f.Hash] = struct{}{}
	}

	return added, total
}

func minLen(s string, n int) int {
	if len(s) < n {
		return len(s)
	}
	return n
}

// cmpVersionDesc compares two version strings in descending order.
// Splits on "." and "-", compares segments numerically when possible.
func cmpVersionDesc(a, b string) int {
	return cmpVersion(b, a) // swap for descending
}

func cmpVersion(a, b string) int {
	pa := splitVersion(a)
	pb := splitVersion(b)
	for i := range max(len(pa), len(pb)) {
		var sa, sb string
		if i < len(pa) {
			sa = pa[i]
		}
		if i < len(pb) {
			sb = pb[i]
		}
		na, errA := strconv.Atoi(sa)
		nb, errB := strconv.Atoi(sb)
		if errA == nil && errB == nil {
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
			continue
		}
		if sa != sb {
			if sa < sb {
				return -1
			}
			return 1
		}
	}
	return 0
}

func splitVersion(s string) []string {
	s = strings.TrimPrefix(s, "v")
	// Split on both "." and "-" to handle "1.2.3-beta1"
	var parts []string
	start := 0
	for i := range len(s) {
		if s[i] == '.' || s[i] == '-' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
