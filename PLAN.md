# Corediff v3 — Implementation Plan

## Target architecture

```
corediff/
├── internal/
│   ├── hashdb/
│   │   ├── hashdb.go      # HashDB type: sorted []uint64, Contains, Add, Merge
│   │   ├── storage.go     # Load (mmap read-only / slice read-write), Save, DB version header
│   │   └── hashdb_test.go
│   ├── normalize/
│   │   ├── normalize.go   # NormalizeLine, Hash, PathHash, HasValidExt, IsValidUtf8
│   │   └── normalize_test.go
│   ├── path/
│   │   ├── path.go        # PathIsExcluded, IsAppRoot, appRootPaths, excludePaths
│   │   └── path_test.go
│   ├── highlight/
│   │   ├── highlight.go   # ShouldHighlight (suspect pattern detection)
│   │   └── highlight_test.go
│   └── chunker/
│       ├── chunker.go     # CDC: content-defined chunking for minified files
│       └── chunker_test.go
├── cmd/corediff/
│   ├── main.go            # CLI entry point: scan (default) and db subcommands
│   ├── scan.go            # corediff scan: walk, scan, display, -vv mode
│   ├── db.go              # corediff db: parent for db subcommands
│   ├── db_add.go          # corediff db add: hash files into DB
│   ├── db_merge.go        # corediff db merge: combine databases
│   ├── db_index.go        # corediff db index: Packagist indexing pipeline
│   └── db_info.go         # corediff db info: print DB stats
├── internal/packagist/
│   ├── client.go          # Packagist/Composer repo API client + rate limiting
│   ├── client_test.go
│   ├── auth.go            # Read ~/.composer/auth.json
│   └── auth_test.go
├── internal/manifest/
│   ├── manifest.go        # Append-only text file: package@version state
│   └── manifest_test.go
├── fixture/
├── db/
├── Justfile
├── go.mod
└── go.sum
```

Shared code lives under `internal/` to prevent unintended external imports.
The binary imports `github.com/gwillem/corediff/internal/hashdb`, etc.
The `packagist/` and `manifest/` packages are only used by `db` subcommands.

---

## Step 1 — Extract shared packages, restructure CLI

**Goal:** Restructure from single `cmd/` main package into internal packages +
single binary with subcommands. All existing tests must pass. No behavior changes.

**Files to create:**
- `internal/hashdb/hashdb.go` — Move `hashDB` type (keep as
  `map[uint64]struct{}` for now), `loadDB`, `saveDB`, `newDB` from
  `cmd/database.go`. Export as `HashDB`, `LoadDB`, `SaveDB`, `NewDB`.
  Add `Contains(uint64) bool` method.
- `internal/normalize/normalize.go` — Move `normalizeLine`, `hash`,
  `pathHash`, `hasValidExt`, `isValidUtf8`, `normalizeRx`, `skipLines`,
  `scanExts` from `cmd/path.go`. Export public API.
- `internal/path/path.go` — Move `pathIsExcluded`, `isAppRoot`,
  `pathExists`, `appRootPaths`, `excludePaths` from `cmd/path.go` and
  `cmd/scan.go`. Export as needed.
- `internal/highlight/highlight.go` — Move `shouldHighlight` and pattern
  data from `cmd/highlight.go`. Export as `ShouldHighlight`.
- `cmd/corediff/main.go` — Single binary entry point. `go-flags` with
  two top-level subcommands: `scan` (default) and `db`. Bare
  `corediff /path` implies `corediff scan /path` (same default command
  logic as current code).
- `cmd/corediff/scan.go` — Move scan logic from `cmd/scan.go`:
  `walkPath`, `parseFileWithDB`, `parseFile`, `parseFH`, `walkStats`,
  `scanArg`. `-vv` flag. Default DB:
  `$XDG_DATA_HOME/corediff/default.db`, error if missing (point user
  to `corediff db index`).
  **Single file scanning:** if argument is a file (not directory), skip
  app root check automatically.
  **Exit codes:** 0 = clean, 1 = unrecognized lines found,
  2 = suspect lines found, >2 = error.
- `cmd/corediff/db.go` — Parent subcommand for `db add`, `db merge`,
  `db index`, `db info`. Default DB: `$XDG_DATA_HOME/corediff/default.db`.
  Manifest: `$XDG_DATA_HOME/corediff/manifest.txt`.

**Files to delete:** all files in `cmd/` (`main.go`, `database.go`,
`scan.go`, `path.go`, `highlight.go`, `add.go`, `trace.go`, `merge.go`,
`logger.go`). Fold color setup into the binaries that need it.

**Move tests:** `cmd/path_test.go` → `normalize_test.go` + `path_test.go`.
`cmd/highlight_test.go` → `highlight_test.go`.

**Verify:** `go test ./...` passes, `go run ./cmd/corediff` compiles,
`corediff scan` and `corediff db` subcommands work.

---

## Step 2 — Sorted []uint64 + binary search

**Goal:** Replace `map[uint64]struct{}` with `[]uint64` (sorted). 8x memory
reduction. Split HashDB into query logic and storage.

**`internal/hashdb/hashdb.go`** — query + mutation:
- `type HashDB struct { main []uint64; buf []uint64 }` (main is sorted,
  buf is unsorted append buffer)
- `Contains(h uint64) bool`: binary search on main, linear scan on buf.
- `Add(h uint64)`: append to buf.
- `Merge(other *HashDB)`: concat all slices.
- `Compact()`: sort buf, merge into main, dedup. Called before Save and
  periodically during large Add sessions (e.g. every 100K adds).

**`internal/hashdb/storage.go`** — file I/O:
- **DB file header** (16 bytes): magic bytes `"CDDB"`, version uint32,
  hash count uint64. Validates on load — reject truncated/corrupt files.
  Version 1 = sorted uint64s. Future versions can change format without
  silent breakage.
- `OpenReadOnly(path) → *HashDB`: mmap file, interpret as sorted
  `[]uint64` (after header). Returns HashDB with main pointing into
  mmap'd memory, empty buf. `Contains` works, `Add` is not allowed
  (panics or returns error).
- `OpenReadWrite(path) → *HashDB`: read file into owned `[]uint64`
  slice. Both `Contains` and `Add` work.
- `Save(path)`: compact, write header + sorted uint64s. Atomic write
  (write to tmp, rename).
- Validate file size: `(filesize - 16) % 8 == 0`, else return error.

**Update callers:**
- `cmd/corediff/main.go`: uses `OpenReadOnly`.
- `cmd/corediff/db_add.go`: uses `OpenReadWrite`.

**Tests:**
- `hashdb_test.go`: Load/Save round-trip, Contains, Merge, Add+Compact,
  deduplication, empty DB, corrupt file detection, version mismatch error.

---

## Step 3 — Mmap (integrated into storage.go)

Mmap is already designed into Step 2's `OpenReadOnly`. This step is just
implementation and benchmarking.

**Implementation in `internal/hashdb/storage.go`:**
- `OpenReadOnly` uses `syscall.Mmap` (darwin + linux). Cast `[]byte` to
  `[]uint64` via `unsafe.Slice`. Skip the 16-byte header.
- `Close()` method to `syscall.Munmap`.
- Fallback: if mmap fails (e.g. empty file), return error.

**Tests:** Benchmark load time before/after. Test that mmap'd DB returns
same results as in-memory `OpenReadWrite`.

---

## Step 4 — Content-defined chunking (CDC)

**Goal:** Handle minified JS/JSON where single long lines break the
line-by-line hashing model.

**New file `chunker.go`:**
```go
package corediff

// Chunk splits data into content-defined chunks using a Rabin-style
// rolling hash. Average chunk size is controlled by mask (e.g. 0x1FF
// for ~512 bytes). Returns chunk boundaries.
func Chunk(data []byte, mask uint64) [][]byte
```

- Rolling hash (Buzhash or simple polynomial) over a 32-byte window.
- Average chunk size ~64 bytes (mask `0x3F`). Roughly 2 minified JS
  statements per chunk — distinctive enough to avoid noise, small enough
  to isolate changes. Tunable constant, adjust based on real-world feedback.
- Min chunk size: 32 bytes (prevent degenerate splits on short tokens).
- Max chunk size: 256 bytes (prevent huge chunks on low-entropy data).
- Expose `ChunkLine(line []byte) [][]byte` — if `len(line) > 512`,
  apply CDC. Otherwise return `[][]byte{line}`.

**Integration in `normalize.go`:**
- `HashLine(line []byte) []uint64` — normalize, then if long, chunk and
  hash each chunk. Otherwise hash the single line. Returns 1 or more
  hashes.

**Update callers:** `parseFileWithDB` in scanner calls `HashLine` instead
of `hash(normalizeLine(line))`. `addPath` in `db add` does the same.

**Backward compatibility:** CDC produces different hashes than line-level
hashing for the same minified content. Old DBs won't match new scanner
output for long lines. This is acceptable because:
- Old DBs already produced a single fragile hash for long lines (useless).
- The DB version header (Step 2) marks the format version.
- Databases must be rebuilt with `corediff db` anyway for v3.

**Tests:**
- Deterministic: same input always produces same chunks.
- Stability: inserting bytes only affects nearby chunks.
- Short lines pass through unchanged.
- Benchmark on `fixture/docroot/editor_plugin.js`.

---

## Step 5 — Fix db subcommands (add, merge, info)

**Goal:** Get the DB operations working cleanly.

**`cmd/corediff/db_add.go`:**
- Fix broken call signature (`addPath` needs correct args).
- Walk path, hash lines (using `HashLine`), add to DB.
- Respect `--ignore-paths` and `--no-platform` flags.
- Save DB at the end.

**`cmd/corediff/db_merge.go`:**
- Accept `--database` (output) and positional args (input DBs).
- Load each input, merge into output, save.
- Report stats: input counts, output count, duplicates removed.

**`cmd/corediff/db_info.go`:**
- `corediff db info [db-path]` — print DB stats: hash count, file size,
  DB format version. If no path given, use default DB.
- `corediff db info --manifest` — print manifest stats: total packages,
  total versions indexed, last indexed timestamp.

**Tests:** Integration tests for each subcommand using fixture data.

---

## Step 6 — Packagist client + auth

**Goal:** Talk to Composer repository APIs, handle authentication.

**`internal/packagist/client.go`:**
```go
type Client struct {
    repos   []string       // repository base URLs
    auth    AuthConfig
    http    *http.Client
    limiter *rate.Limiter  // golang.org/x/time/rate
}

type PackageVersion struct {
    Name    string
    Version string
    Dist    struct {
        URL  string
        Type string // "zip" or "tar"
    }
}

func NewClient(repos []string, auth AuthConfig) *Client
// Sets User-Agent with mailto= per Packagist guidelines.
// Uses HTTP/2 client.
// Default rate limit: 5 req/s. Honors Retry-After header on 429.

func (c *Client) Init() error
// Fetch {repo}/packages.json from each repo to discover which
// packages each repo provides (full list or patterns like "magento/*").
// Build routing map: package → repo. Prevents leaking package names
// to repos that don't host them.

func (c *Client) Versions(pkg string) ([]PackageVersion, error)
// GET {repo}/p2/{vendor}/{package}.json
// Routes to the repo that claims the package (via Init routing map).
// Falls back to Packagist for anything unclaimed.

func (c *Client) DownloadZip(pv PackageVersion) (*zip.Reader, error)
// Download zip into memory (max 50MB), return zip.Reader.
// Validates Content-Length before downloading.
```

**Rate limiting strategy:**
- Default: `rate.NewLimiter(5, 1)` — 5 requests/sec, burst of 1.
- On HTTP 429: honor `Retry-After` header (seconds or HTTP-date).
  If no `Retry-After`, exponential backoff (1s, 2s, 4s, max 30s).
- On any `X-RateLimit-Remaining` header: if value is 0, sleep until
  `X-RateLimit-Reset` (Packagist doesn't document these, but honor
  them defensively if present).
- Log rate limit events at verbose level.

**Download safety:**
- Cap download size at 50MB. Reject if `Content-Length` exceeds cap.
- If no `Content-Length`, use `io.LimitReader` during download.
- Validate zip structure before returning reader.

**`internal/packagist/auth.go`:**
```go
type AuthConfig struct {
    HTTPBasic map[string]struct{ Username, Password string }
}

func LoadAuth(path string) (AuthConfig, error)
// Parse ~/.composer/auth.json, extract http-basic section.

func (a AuthConfig) ForHost(host string) (user, pass string, ok bool)
// Lookup credentials by hostname.
```

**Tests:** Test auth.json parsing. Test version list parsing against
recorded API responses (testdata/). Test rate limiter behavior on 429.

---

## Step 7 — Manifest + index command

**Goal:** `corediff db index <package>` indexes all versions of a Packagist
package into the hash DB. Incremental via text manifest.

**`manifest/manifest.go`:**
```go
type Manifest struct {
    mu      sync.Mutex
    path    string
    fh      *os.File              // open for append
    indexed map[string]struct{}   // "package@version"
}

func Load(path string) (*Manifest, error)
// Read text file, one "package@version" per line, into set.
// Opens file handle for appending. Acquires flock for cross-process safety.

func (m *Manifest) IsIndexed(pkg, version string) bool
// Lock-free: reads from in-memory set (populated at Load time).

func (m *Manifest) MarkIndexed(pkg, version string) error
// Mutex-protected: appends line to file and updates in-memory set.
// Safe for concurrent calls from worker goroutines.

func (m *Manifest) Close() error
// Release flock and close file handle.
```

**`cmd/corediff/db_index.go`:**
```go
type indexArg struct {
    Database string `short:"d" long:"database" required:"true"`
    Repo     []string `long:"repo"`
    Auth     string   `long:"auth"`
    NoDeps   bool     `long:"no-deps"`
    Update   bool     `long:"update"`
    FromComposer string `long:"from-composer"`
    Workers  int      `long:"workers" default:"8"`
}
```

Pipeline:
1. Build package list (single package, or resolve deps from composer.json).
2. For each package, fetch versions from Packagist client.
3. Filter out already-indexed (check manifest).
4. Worker pool: download zip into memory → `zip.NewReader` over
   `bytes.NewReader` (no tmpdir) → hash lines from each entry → add to
   DB → mark in manifest.
   **Path mapping:** detect zip root (could be a single root dir like
   `module-catalog-a3f2b1d/`, or `magento/`, or `.` / no root at all).
   Detect by checking if all entries share a common first path component.
   Strip it if so, then prepend `vendor/{vendor}/{package}/` to match
   real `composer install` layout. Store path hashes with this mapped path.
5. Save DB (sort + dedup).

**Dependency resolution (default):** Download one version of the root
package, parse its `composer.json`, extract `require` keys, recurse.
`--no-deps` skips this and indexes only the specified package.
**Depth limit:** max 3 levels deep by default (`--depth N` to override).
Prevents crawling the entire PHP ecosystem when indexing a metapackage.
Level 0 = root package, level 1 = direct deps, level 2 = transitive.
Log skipped packages at verbose level.

**`--update` mode:** Read all packages from manifest, re-fetch version
lists, index only new versions.

**`--from-composer`:** Parse a `composer.json` file, extract `repositories`
(as `--repo` values) and `require` (as package list), then resolve deps.
Also parse `composer.lock` if present — use its exact package list
instead of recursing, as it represents the real installed set.

**Progress output:** Print ongoing stats to stderr during indexing:
`[142/580 versions] magento/module-catalog@2.4.6 (+312 hashes)`
On completion, print summary: packages indexed, versions processed,
total hashes, new hashes added, time elapsed.

**Tests:** End-to-end test with a small real package (e.g.
`psr/log` — few versions, small files, public).

---

## Step 8 — Release infrastructure

**Goal:** Justfile for cross-compilation, GitHub releases, self-update.

**`Justfile`:**
```just
version := `git describe --tags --always`
ldflags := "-X main.version=" + version

build:
    go build -ldflags '{{ldflags}}' -o dist/corediff ./cmd/corediff
    # single binary, no cddb

release:
    #!/usr/bin/env bash
    for platform in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
        os="${platform%/*}"; arch="${platform#*/}"
        GOOS=$os GOARCH=$arch go build -ldflags '{{ldflags}}' \
            -o dist/corediff-$os-$arch ./cmd/corediff
    done

upload: release
    gh release create v{{version}} dist/corediff-*
```

**`corediff update` subcommand:**
- Check `https://api.github.com/repos/gwillem/corediff/releases/latest`
  for tag name, compare to compiled-in version string.
- If newer, download `corediff-{os}-{arch}` from release assets,
  replace current binary.
- No auto-update on scan. User must explicitly run `corediff update`.

**DB version gate:**
- DB header contains format version (see Step 2).
- On `corediff scan`: if DB version > compiled-in max supported version,
  exit with error: `"Database requires corediff vX.Y+. Run: corediff update"`.
- This ensures newer DBs built with a future `corediff db` version
  aren't silently misinterpreted by an older scanner.

**Remove:** `go-selfupdate` and `go-buildversion` dependencies. Replace
with `-ldflags` version injection.

---

## Suggested order of implementation

| Step | Session | Depends on |
|------|---------|------------|
| 1. Restructure CLI + shared packages | 1 | — |
| 2. Sorted []uint64 | 2 | Step 1 |
| 3. Mmap | 2 | Step 2 |
| 5. Fix db subcommands | 3 | Step 1 |
| 4. CDC chunker | 3 | Step 1 |
| 6. Packagist client + auth | 4 | — |
| 7. Manifest + index command | 5 | Steps 5, 6 |
| 8. Release infrastructure | 6 | Step 1 |

Steps 2+3 combine into one session (both touch hashdb.go).
Steps 4+5 combine into one session (both touch the db subcommands).
Step 6 is independent and can be done in parallel with anything after step 1.

---

## Risks & mitigations

| # | Risk | Mitigation | Step |
|---|------|------------|------|
| 1 | Dependency recursion crawls all of Packagist | Depth limit (default 3). Parse `composer.lock` when available. | 7 |
| 2 | CDC breaks backward compat with old DBs | DB version header rejects old format. v3 requires full rebuild. | 2, 4 |
| 3 | Mmap + Add conflict (can't append to mmap'd slice) | Separate `OpenReadOnly` / `OpenReadWrite` in storage.go. | 2, 3 |
| 4 | Packagist rate limiting / bans | 5 req/s default, honor `Retry-After` on 429, honor `X-RateLimit-*` if present, `User-Agent` with mailto, HTTP/2. | 6 |
| 5 | Zip bombs / oversized packages | Cap download at 50MB, validate `Content-Length`, `io.LimitReader`. | 6 |
| 6 | Corrupt/truncated DB file | Magic header + size validation on load. | 2 |
| 7 | No way to inspect DB contents | `corediff db info` subcommand. | 5 |
| 8 | Single file scan requires `--ignore-paths` | Auto-skip app root check when argument is a file. | 1 |
| 9 | No exit codes for CI usage | 0=clean, 1=unrecognized, 2=suspect, >2=error. | 1 |
| 10 | No progress during long indexing runs | Ongoing stats to stderr + completion summary. | 7 |
| 11 | Shared code importable by external packages | All shared code under `internal/`. | 1 |
