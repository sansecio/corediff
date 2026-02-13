package gitindex

import (
	"fmt"
	"io"
	"net/http"
	"os"
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
	NoPlatform   bool
	AllValidText bool
	PathPrefix   string                           // prepended to file paths for path hashes (e.g. "vendor/psr/log/")
	Logf         func(format string, args ...any) // optional file-level logger (-vv)
	LineLogf     func(format string, args ...any) // optional line-level logger (-vvv)
	HTTP         *http.Client                     // optional; defaults to http.DefaultClient
}

// CloneAndIndex bare-clones repoURL, then for each version→ref pair,
// walks the git tree and hashes all eligible files into db.
func CloneAndIndex(repoURL string, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) error {
	opts.installHTTPTransport()

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

	for version, ref := range refs {
		if err := indexRef(repo, version, ref, db, opts); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s (%s): %v\n", version, ref[:minLen(ref, 12)], err)
		}
	}

	return nil
}

// CloneAndIndexWithDir is like CloneAndIndex but uses an existing directory
// for the bare clone. If the directory already contains a valid repo, it reuses it.
func CloneAndIndexWithDir(repoURL, cloneDir string, refs map[string]string, db *hashdb.HashDB, opts IndexOptions) error {
	opts.installHTTPTransport()

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

	for version, ref := range refs {
		if err := indexRef(repo, version, ref, db, opts); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s (%s): %v\n", version, ref[:minLen(ref, 12)], err)
		}
	}

	return nil
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

	fmt.Printf("  indexing %s (%s)\n", version, ref[:minLen(ref, 12)])

	return tree.Files().ForEach(func(f *object.File) error {
		return indexFile(f, db, opts)
	})
}

func (opts IndexOptions) logf(format string, args ...any) {
	if opts.Logf != nil {
		opts.Logf(format, args...)
	}
}

func (opts IndexOptions) httpClient() *http.Client {
	if opts.HTTP != nil {
		return opts.HTTP
	}
	return http.DefaultClient
}

// installHTTPTransport registers a custom HTTP client with go-git's protocol
// handlers so clone/fetch requests are logged. This is global state, so it
// should only be called once before git operations.
func (opts IndexOptions) installHTTPTransport() {
	if opts.HTTP == nil {
		return
	}
	t := githttp.NewClient(opts.HTTP)
	gitclient.InstallProtocol("https", t)
	gitclient.InstallProtocol("http", t)
}

func indexFile(f *object.File, db *hashdb.HashDB, opts IndexOptions) error {
	if !opts.AllValidText && !normalize.HasValidExt(f.Name) {
		opts.logf("    skip %s (no valid ext)", f.Name)
		return nil
	}

	// Check UTF-8 validity by reading first 8KB
	reader, err := f.Blob.Reader()
	if err != nil {
		return nil // skip unreadable files
	}

	buf := make([]byte, 8*1024)
	n, err := reader.Read(buf)
	reader.Close()
	if err != nil && err != io.EOF {
		return nil
	}
	if !utf8.Valid(buf[:n]) {
		opts.logf("    skip %s (invalid utf8)", f.Name)
		return nil
	}

	// Add path hash unless NoPlatform
	if !opts.NoPlatform {
		storedPath := opts.PathPrefix + f.Name
		db.Add(normalize.PathHash(storedPath))
		opts.logf("    hash %s", storedPath)
	} else {
		opts.logf("    hash %s", f.Name)
	}

	// Re-open reader and hash all lines
	reader, err = f.Blob.Reader()
	if err != nil {
		return nil
	}
	defer reader.Close()

	normalize.HashReader(reader, db, opts.LineLogf)
	return nil
}

func minLen(s string, n int) int {
	if len(s) < n {
		return len(s)
	}
	return n
}
