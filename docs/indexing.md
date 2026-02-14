# Indexing Pipeline

Corediff detects unauthorized modifications in open-source codebases by comparing file contents against a database of known-good code hashes. This document describes how the hash database is built.

## Overview

The indexer processes source code files and stores two types of xxhash64 hashes:

- **Line hashes** — each meaningful line of code, after normalization (comment stripping, whitespace normalization, regex filters)
- **Path hashes** — file paths relative to the project root (e.g. `vendor/magento/module-catalog/Model/Product.php`)

During scanning, files whose paths are in the DB are checked line-by-line. Unknown paths are skipped as "custom code" unless `--ignore-paths` is used.

## Indexing Modes

### `--packagist <vendor/package>`

Indexes a single Packagist package across all its versions.

1. Fetches version metadata from the Packagist API
2. Filters out already-indexed versions (via manifest)
3. Attempts git clone (all versions at once, with blob dedup)
4. Falls back to zip download per version if git fails
5. Records each indexed version in the manifest

Supports version pinning: `--packagist vendor/pkg:1.2.3`

### `--composer <path/to/composer.json>`

Indexes all packages from a project's `composer.json` + `composer.lock`.

1. Parses `composer.json` for repository URLs and `composer.lock` for package list
2. Filters out already-indexed packages and packages replaced by a monorepo
3. For each remaining package, indexes using source/dist from the lock file, falling back to repository API lookup
4. Packages are indexed concurrently (bounded by GOMAXPROCS)

### `--update`

Re-checks all previously indexed packages for new versions.

1. Reads package names from the manifest
2. For each package, queries Packagist for versions not yet in the manifest
3. Indexes new versions via git clone or zip download

### `<path>` (local directory)

Indexes files directly from a local directory. Used for one-off additions or non-Packagist code.

## Manifest

The manifest (`.manifest` file alongside the `.db`) tracks indexing state for incremental operation. It is an append-only text file with two entry types:

- `package@version` — marks a specific version as indexed
- `replace:package` — marks a package as replaced by a monorepo

The manifest is flock'd for cross-process safety.

## Monorepo Replace Handling

Large projects like Magento publish ~230 individual `magento/*` packages that are subtree-splits from the `magento/magento2` monorepo. Indexing the monorepo via git is more efficient (blob dedup across versions, no authenticated access needed).

When indexing via git, corediff reads `composer.json` from each version's tree root and extracts the `"replace"` section. All replaced package names are recorded in the manifest as `replace:` entries.

In `--composer` mode, packages that appear in the manifest's replace set are automatically skipped, avoiding redundant downloads.

**Recommended Magento workflow:**

```bash
# 1. Index the monorepo (covers all magento/* packages)
corediff db add --packagist magento/magento2

# 2. Index remaining third-party packages from your project
corediff db add --composer /path/to/composer.json
```

Step 2 will skip all `magento/*` packages already covered by the monorepo.

## Blob Dedup

When indexing via git clone, versions are processed newest-first. Git blob hashes are tracked across versions — if a file's blob hash was already processed in a newer version, it is skipped in older versions. This avoids redundant I/O and hashing for files that don't change between releases.

## Path Handling

File paths stored in the DB use the `vendor/` prefix convention: `vendor/<package-name>/<file-path>`. This is controlled by the `PathPrefix` option set during indexing.

- **`excludePaths`** — Patterns like `generated/**`, `var/**` prevent paths from being stored in the DB, but line contents are still hashed
- **`--no-platform`** — Don't store path hashes at all; only store line hashes
- **`--ignore-paths`** — During scanning, skip path checking and scan all files regardless

## Normalization

Before hashing, each line is normalized:

1. Leading/trailing whitespace is stripped
2. Comments are removed (single-line `//`, `#` and multi-line `/* */`)
3. Empty lines after stripping are skipped
4. Configurable regex filters can further transform lines

This ensures that formatting differences (indentation, comment style) don't cause false positives.

## File Filtering

Files are filtered before processing:

- **Extension check** — Only files with recognized code extensions (`.php`, `.js`, `.phtml`, etc.) are processed, unless `--text` is used
- **UTF-8 validation** — Binary files are detected by checking the first 8KB for valid UTF-8; invalid files are skipped
