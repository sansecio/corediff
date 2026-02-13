# Corediff v3

## Dream scenario

```bash
# Scanning — what analysts run daily
corediff scan /var/www/magento                       # scan with default DB
corediff scan -s /var/www/magento                    # suspect lines only
corediff scan -i /var/www/magento                    # ignore paths, scan everything
corediff scan -d custom.db /var/www/magento          # scan with specific DB
corediff scan -vv < suspicious-file.php          # debug: show hash for each line

# Indexing — building the hash database
corediff db index --from-composer composer.json      # index full Magento install (all deps, all versions)
corediff db index --update                           # re-run weekly, picks up new package versions only
corediff db index magento/module-catalog             # index a single package (all versions)
corediff db index --repo https://repo.magento.com \
    magento/product-enterprise-edition               # index from private repo

# Database maintenance
corediff db add -d custom.db /path/to/extension      # add a local extension to DB
corediff db merge -d all.db community.db enterprise.db  # combine databases
corediff db info                                     # print DB stats
```

## Subcommands

Single binary with two top-level subcommands. `scan` is the default
(bare `corediff /path` implies `corediff scan /path`).

**`corediff scan <path>`** — Scanner
- [ ] Scan files/dirs against hash DB
- [ ] `-vv` flag: show hash for each input line (debugging, reads from files or stdin)
- [ ] Default DB: `$XDG_DATA_HOME/corediff/default.db`. Override with `-d <path>`.
- [ ] Error on first run if no DB exists, point user to `corediff db index`.
- [ ] Single file argument: auto-skip app root check
- [ ] Exit codes: 0=clean, 1=unrecognized lines, 2=suspect lines, >2=error

**`corediff db`** — Database maintenance
- [ ] `corediff db add <path>` — Add hashes from files/dirs to DB
- [ ] `corediff db merge <db1> <db2>` — Merge multiple databases
- [ ] `corediff db index <package>` — Index a Packagist package (all versions) into the DB
- [ ] `corediff db info [db-path]` — Print DB stats (hash count, format version, file size)
- [ ] Default DB: `$XDG_DATA_HOME/corediff/default.db`. Override with `-d <path>`.
- [ ] Manifest: `$XDG_DATA_HOME/corediff/manifest.txt`

## Database

- [ ] Replace `map[uint64]struct{}` with sorted `[]uint64` + binary search (~8x memory reduction)
- [ ] Mmap the sorted database file (instant startup, OS-managed paging)
- [ ] DB file header: magic bytes, version, hash count. Reject corrupt/truncated files.

## Indexing

Pipeline: `corediff db index <package>`, `corediff db index --no-deps <package>`, `corediff db index --update`

- [ ] Fetch version list from Composer repository API (`/p2/{vendor}/{package}.json`), diff against manifest, process only new versions
- [ ] Default repo: Packagist. `--repo <url>` for additional sources (e.g. `repo.magento.com`)
- [ ] `--from-composer <composer.json>`: read repositories + root package + deps from existing project
- [ ] Worker pool (8-16 goroutines): download zip into memory → `zip.NewReader` → hash lines directly → append to DB
- [ ] Follow dependencies recursively by default. `--no-deps` to index a single package only.
- [ ] `--update` mode: re-check all previously indexed packages for new versions
- [ ] Append-only text manifest for state (`package@version` per line, one entry per indexed version)
- [ ] Read `~/.composer/auth.json` for private repo auth (e.g. repo.magento.com). Optional `--auth` flag to override.
- [ ] Final step: sort + deduplicate DB


## Scanning

- [ ] Content-defined chunking (CDC) for minified JS/JSON: use rolling hash to split long lines into ~64-byte chunks (min 32, max 256). Approximates statement-level granularity without syntax parsing. Language-agnostic, resilient to local insertions.
- [ ] Parallel file scanning (worker pool)

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


