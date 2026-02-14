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
	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/gwillem/corediff/internal/normalize"
)

// IndexOptions controls how files are indexed.
type IndexOptions struct {
	NoPlatform      bool
	AllValidText    bool
	PathPrefix      string            // prepended to file paths for path hashes (e.g. "vendor/psr/log/")
	Verbose         int               // verbosity level: 1=-v (versions), 3=-vvv (files), 4=-vvvv (lines)
	HTTP            *http.Client      // optional; defaults to http.DefaultClient
	OnVersionDone   func(version string) // called after each version is indexed (for manifest updates)
}

// CloneAndIndex bare-clones repoURL, then for each version→ref pair,
// walks the git tree and hashes all eligible files into db.
func CloneAndIndex(repoURL string, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) error {
	opts.InstallHTTPTransport()

	tmpDir, err := os.MkdirTemp("", "corediff-git-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := git.PlainClone(tmpDir, true, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		return fmt.Errorf("cloning %s: %w", repoURL, err)
	}

	indexRefs(repo, refs, db, opts)
	return nil
}

// CloneAndIndexWithDir is like CloneAndIndex but uses an existing directory
// for the bare clone. If the directory already contains a valid repo, it reuses it.
func CloneAndIndexWithDir(repoURL, cloneDir string, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) error {
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
			return fmt.Errorf("cloning %s: %w", repoURL, err)
		}
	} else {
		err = repo.Fetch(&git.FetchOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("fetching %s: %w", repoURL, err)
		}
	}

	indexRefs(repo, refs, db, opts)
	return nil
}

func indexRefs(repo *git.Repository, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) {
	versions := slices.Collect(maps.Keys(refs))
	slices.SortFunc(versions, cmpVersionDesc)
	for _, version := range versions {
		if err := indexRef(repo, version, refs[version], db, opts); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s (%s): %v\n", version, refs[version][:minLen(refs[version], 12)], err)
		}
	}
}

func indexRef(repo *git.Repository, version, ref string, db *hashdb.HashDB, opts IndexOptions) error {
	commit, err := repo.CommitObject(plumbing.NewHash(ref))
	if err != nil {
		return fmt.Errorf("resolving commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return fmt.Errorf("getting tree: %w", err)
	}

	var newHashes, totalHashes int
	start := time.Now()

	err = tree.Files().ForEach(func(f *object.File) error {
		n, t := indexFileCount(f, db, opts)
		newHashes += n
		totalHashes += t
		return nil
	})

	elapsed := time.Since(start)
	rate := float64(totalHashes) / max(elapsed.Seconds(), 0.001)

	pkg := strings.TrimSuffix(strings.TrimPrefix(opts.PathPrefix, "vendor/"), "/")
	opts.log(1, "  indexed %s@%s (%d new, %d total, %.0f hash/sec)", pkg, version, newHashes, totalHashes, rate)

	if err == nil && opts.OnVersionDone != nil {
		opts.OnVersionDone(version)
	}

	return err
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
func indexFileCount(f *object.File, db *hashdb.HashDB, opts IndexOptions) (int, int) {
	if !opts.AllValidText && !normalize.HasValidExt(f.Name) {
		opts.log(3, "    skip %s (no valid ext)", f.Name)
		return 0, 0
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
		return 0, 0
	}

	// Add path hash unless NoPlatform
	if !opts.NoPlatform {
		storedPath := opts.PathPrefix + f.Name
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
	return normalize.HashReader(reader, db, lineLogf)
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
