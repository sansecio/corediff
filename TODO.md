# Corediff v3

## Dream scenario

```bash
# Scanning — what analysts run daily
corediff scan /var/www/magento                       # scan with default DB
corediff scan -s /var/www/magento                    # suspect lines only
corediff scan -i /var/www/magento                    # ignore paths, scan everything
corediff scan -d custom.db /var/www/magento          # scan with specific DB
corediff scan -vv suspicious-file.php              # debug: show hash for each line

# Indexing — building the hash database
corediff db add --composer /var/www/magento           # index full install (all deps from lock file)
corediff db add --packagist magento/module-catalog    # index a single Packagist package (all versions)
corediff db add -d custom.db /path/to/extension       # add a local extension to DB

# Future
corediff db add --update                              # re-run weekly, picks up new package versions only
corediff db add --packagist magento/product-enterprise-edition  # private repo auth via ~/.composer/auth.json

# Database maintenance
corediff db merge -d all.db community.db enterprise.db  # combine databases
corediff db info                                        # print DB stats
```

## Subcommands

Single binary with two top-level subcommands. `scan` is the default
(bare `corediff /path` implies `corediff scan /path`).

**`corediff scan <path>`** — Scanner
- [x] Scan files/dirs against hash DB
- [x] `-vv` flag: show hash for each input line (debugging, reads from files or stdin)
- [ ] Default DB: `$XDG_DATA_HOME/corediff/default.db`. Override with `-d <path>`.
- [ ] Error on first run if no DB exists.
- [ ] Single file argument: auto-skip app root check
- [ ] Exit codes: 0=clean, 1=unrecognized lines, 2=suspect lines, >2=error

**`corediff db`** — Database maintenance
- [x] `corediff db add <path>` — Add hashes from local files/dirs to DB
- [x] `corediff db add --packagist <vendor/package>` — Index a Packagist package (all versions) into the DB
- [x] `corediff db add --composer <path>` — Index all packages from composer.json + lock
- [x] `corediff db merge <db1> <db2>` — Merge multiple databases
- [x] `corediff db info [db-path]` — Print DB stats (hash count, format version, file size)
- [ ] Default DB: `$XDG_DATA_HOME/corediff/default.db`. Override with `-d <path>`.
- [x] Manifest: sibling of DB file (e.g. `corediff.db` → `corediff.manifest`)

## Database

- [x] CDDB binary file format: 16-byte header (magic, version, hash count) + sorted uint64s
- [x] Mmap for read-only access (instant startup, OS-managed paging)
- [x] Map-based in-memory lookups for O(1) Contains

## Indexing

Pipeline: `corediff db add --packagist <package>`, `corediff db add --composer <path>`

- [x] Fetch version list from Packagist API (`/p2/{vendor}/{package}.json`)
- [x] Default repo: Packagist. Custom repos discovered from composer.json `repositories` section.
- [x] `--composer <path>`: read composer.json (repos) + composer.lock (packages with source/dist)
- [x] `--packagist <vendor/package>`: index single package, supports version pin (`vendor/pkg:1.2.3`)
- [x] Parallel goroutine pool (GOMAXPROCS workers) for --composer mode
- [x] Prefer git clone + tree walk, fall back to zip download per version
- [x] Read `~/.composer/auth.json` for private repo auth (http-basic, bearer, github-oauth)
- [x] Sort + deduplicate DB before save
- [ ] Follow dependencies recursively by default. `--no-deps` to index a single package only.
- [x] `--update` mode: re-check all previously indexed packages for new versions
- [x] Append-only text manifest for state (`package@version` per line, one entry per indexed version)


## Scanning

- [x] Content-defined chunking (CDC) for minified JS/JSON: Buzhash rolling window, variable-size chunks (64-512 bytes)
- [ ] Parallel file scanning (worker pool)

## Concurrency

- [x] Worker pool (GOMAXPROCS semaphore) for `--composer` and `--update` modes
- [x] Documented in `doc/concurrency.md`
- [-] Parallel git tag processing — investigated, not viable (seenBlobs dedup across versions saves ~95% of work for monorepos like magento2; parallelizing would multiply total I/O)

## Release & update

- [ ] `corediff update` subcommand: check GitHub releases, download newer binary if available. No auto-update.
- [ ] DB version gate: if DB format version > max supported, exit with `"Database requires corediff vX.Y+. Run: corediff update"`
- [ ] `just release`: bump version, cross-compile (darwin/linux × amd64/arm64), upload to GitHub release via `gh release create`
- [ ] Consistent asset naming: `corediff-{os}-{arch}`

## LLM triage (`cdsummary`)

Separate tool that pipes corediff output through an LLM for risk prioritization.

- [ ] Two-pass approach: corediff filters locally first, only sends ambiguous files to LLM
- [ ] Skip files that are >80% unrecognized (custom code, not tampered — no LLM needed)
- [ ] Send top N suspicious files as plain `path:line content` — no JSON/markup overhead
- [ ] LLM classifies per file: high risk / suspicious / likely benign, with brief explanation
- [ ] Cluster related changes ("these 40 lines across 8 files are one custom module")



# manual 

- [ ] git indexer does not correctly identify packagist packages from monorepo (see magento2), or at least no write them to the manifest
- [ ] if packages are embedded in a monorepo, do their relative paths match the ones if they had been installed via composer? that is important since most of our target installs use composer and we only analyze a file if the relative path has been registered
- [ ] zip downloads dont end up in cache dir
- [ ] merge various logger functions