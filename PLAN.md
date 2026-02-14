# Corediff v3 — Implementation Plan

## Current architecture

```
corediff/
├── cmd/corediff/
│   ├── main.go            # CLI entry point: scan (default) and db subcommands
│   ├── scan.go            # corediff scan: walk, scan, display
│   ├── scan_test.go
│   ├── output.go          # Color helpers, verbose logging
│   ├── db.go              # corediff db: parent for db subcommands
│   ├── db_add.go          # corediff db add: local paths, --packagist, --composer, --update
│   ├── db_merge.go        # corediff db merge: combine databases
│   ├── db_info.go         # corediff db info: print DB stats
│   └── db_test.go
├── internal/
│   ├── hashdb/            # Map-based HashDB + CDDB binary format (with legacy support)
│   ├── normalize/         # NormalizeLine, Hash, PathHash, HasValidExt, IsValidUtf8
│   ├── path/              # IsExcluded, IsAppRoot, appRootPaths, excludePaths
│   ├── highlight/         # ShouldHighlight (suspect pattern detection)
│   ├── chunker/           # CDC: content-defined chunking for minified files
│   ├── gitindex/          # Git clone + tree walk, zip fallback, blob dedup across versions
│   ├── manifest/          # Append-only manifest tracking indexed package@version pairs
│   ├── composer/          # Parse composer.json/lock, auth.json
│   └── packagist/         # Packagist/Composer repository API client
├── fixture/
├── db/                    # Pre-computed Magento hash databases
└── go.mod / go.sum
```

---

## Step 7b — Dependency resolution

**Goal:** `corediff db add --packagist magento/product-enterprise-edition` automatically resolves
and indexes all transitive dependencies.

- Download one version of the root package, parse its `composer.json` `require` section, recurse.
- `--no-deps` flag to index a single package only.
- Depth limit: max 3 levels deep by default (`--depth N` to override).
- When `composer.lock` is available (via `--composer`), use its exact package list instead of recursing.

---

## Step 8 — Scanner improvements

**`corediff scan` enhancements:**
- Default DB: `$XDG_DATA_HOME/corediff/default.db`. Override with `-d <path>`.
- Error on first run if no DB exists.
- Single file argument: auto-skip app root check.
- Exit codes: 0=clean, 1=unrecognized lines, 2=suspect lines, >2=error.
- Parallel file scanning (worker pool).

---

## Step 9 — Release infrastructure

**`Justfile`:**
```just
version := `git describe --tags --always`
ldflags := "-X main.version=" + version

build:
    go build -ldflags '{{ldflags}}' -o dist/corediff ./cmd/corediff

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
- Check GitHub releases for newer version, download matching binary.
- No auto-update. User must explicitly run `corediff update`.

**DB version gate:**
- If DB version > compiled-in max supported version, exit with error:
  `"Database requires corediff vX.Y+. Run: corediff update"`.

**Remove:** `go-selfupdate` and `go-buildversion` dependencies. Replace with `-ldflags` version injection.

---

## Step 10 — LLM triage (`cdsummary`)

Separate tool that pipes corediff output through an LLM for risk prioritization.

- Two-pass approach: corediff filters locally first, only sends ambiguous files to LLM
- Skip files that are >80% unrecognized (custom code, not tampered — no LLM needed)
- Send top N suspicious files as plain `path:line content` — no JSON/markup overhead
- LLM classifies per file: high risk / suspicious / likely benign, with brief explanation
- Cluster related changes ("these 40 lines across 8 files are one custom module")

---

## Risks & mitigations

| # | Risk | Mitigation | Step |
|---|------|------------|------|
| 1 | Dependency recursion crawls all of Packagist | Depth limit (default 3). Parse `composer.lock` when available. | 7b |
| 2 | Packagist rate limiting / bans | Honor `Retry-After` on 429, `User-Agent` with mailto. | 7b |
| 3 | Zip bombs / oversized packages | Cap download at 100MB, validate `Content-Length`. | — |
| 4 | Single file scan requires `--ignore-paths` | Auto-skip app root check when argument is a file. | 8 |
| 5 | No exit codes for CI usage | 0=clean, 1=unrecognized, 2=suspect, >2=error. | 8 |
