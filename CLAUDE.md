# Corediff

Forensic tool to detect unauthorized code modifications in open-source codebases (Magento, WordPress). Compares lines of code against a database of legitimate code hashes (xxhash64) to identify tampering or malware.

## Project structure

- `cmd/` — All Go source: CLI entry point, scanning, database I/O, highlighting, path utils
- `db/` — Pre-computed Magento version hash databases (.db files)
- `fixture/` — Test fixtures (sample docroot, sample/empty databases)
- `build/` — Pre-built binaries (darwin/linux, amd64/arm64)

## Commands

- `go test ./...` — Build and run all tests (never use `go build`)
- CI runs `go test -v ./...` on ubuntu with Go 1.20

## Architecture

- Single `main` package in `cmd/`
- CLI: `go-flags` with subcommands (scan, add, merge, trace)
- Hash format: binary little-endian uint64 (xxhash64), type `hashDB = map[uint64]struct{}`
- Normalization: strips whitespace, comments, applies regex filters before hashing
- Self-update via `go-selfupdate`

## Path handling

The hash DB stores both line hashes and path hashes. In default scan mode, only files whose path is in the DB are checked — unknown paths are skipped as "custom code." This focuses output on official platform files where tampering matters most.

- `--ignore-paths` — Skip path checking entirely. Scan all files regardless of whether their path is in the DB. Useful for scanning non-standard installs or third-party code.
- `--no-platform` — Skip platform root detection (normally corediff requires the target to look like a Magento/WordPress root). When adding hashes, don't store file paths in the DB — only store line hashes.

When adding hashes with `add`, paths from `excludePaths` (e.g. `generated/**`, `var/**`) are never stored in the DB, but their line contents are still hashed.

## Key dependencies

- `cespare/xxhash/v2` — Fast hashing
- `jessevdk/go-flags` — CLI parsing
- `fatih/color` — Terminal colors
- `gobwas/glob` — Path pattern matching
- `stretchr/testify` — Test assertions
