# Corediff Architecture

## Component Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                     CLI  (cmd/corediff/)                         │
│                     go-flags subcommands                         │
│                                                                  │
│   scan.go        db_add.go        db_merge.go     db_info.go    │
└──────┬───────────────┬──────────────────┬──────────────┬─────────┘
       │               │                  │              │
       ▼               ▼                  ▼              ▼
┌──────────┐  ┌────────────────┐  ┌───────────┐  ┌───────────┐
│ normalize│  │   gitindex     │  │  hashdb   │  │  hashdb   │
│ hashdb   │  │   packagist    │  │  (merge)  │  │  (stats)  │
│ path     │  │   composer     │  └───────────┘  └───────────┘
│ highlight│  │   hashdb       │
│ chunker  │  │   manifest     │
└──────────┘  └────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                  Internal Packages (internal/)                   │
├────────────┬───────────┬───────────┬──────────┬─────────────────┤
│  hashdb    │ normalize │ gitindex  │packagist │  composer       │
│  Hash DB   │ Line norm │ Git clone │ API      │  Parse json/lock│
│  CDDB I/O  │ xxhash64  │ Zip fetch │ Versions │  Auth config    │
│            │ Chunking  │ Blob dedup│          │  Repo config    │
├────────────┼───────────┼──────────┬┴──────────┼─────────────────┤
│  manifest  │ path      │ chunker  │ highlight │  output         │
│  Track     │ App root  │ Buzhash  │ Malware   │  Colors         │
│  versions  │ Excludes  │ CDC split│ patterns  │  Logging        │
└────────────┴───────────┴──────────┴───────────┴─────────────────┘
```

## Scanning Flow

Detects unauthorized modifications by comparing file contents against known-good hashes.

```
corediff scan <docroot>
       │
       ▼
 ┌─────────────┐
 │  Load hash  │  CDDB binary format: [magic][version][count][hash...]
 │  database   │  In-memory: map[uint64]struct{}
 └──────┬──────┘
        │
        ▼
 ┌─────────────┐
 │  Detect     │  Look for Magento/WordPress markers
 │  platform   │  (skip with --no-platform)
 └──────┬──────┘
        │
        ▼
 ┌─────────────┐     For each file in docroot:
 │  Walk files │─────────────────────────────────────┐
 └─────────────┘                                     │
                                                     ▼
                                              ┌──────────────┐
                                              │ Check path   │  xxhash64("path:" + relPath)
                                              │ in DB?       │  Unknown path = custom code → skip
                                              └──────┬───────┘  (override with --ignore-paths)
                                                     │ known
                                                     ▼
                                              ┌──────────────┐
                                              │ For each     │  1. Trim whitespace
                                              │ line:        │  2. Strip comments (// # /* *)
                                              │  normalize   │  3. Apply regex filters
                                              │  hash        │  4. xxhash64 (or chunk if >512B)
                                              │  lookup      │  5. db.Contains(hash)?
                                              └──────┬───────┘
                                                     │
                                                     ▼
                                              ┌──────────────┐
                                              │ Report       │  Unrecognized lines = potential tampering
                                              │ findings     │  --suspect: filter through highlight patterns
                                              └──────────────┘  (40+ regex/literal malware signatures)
```

## Indexing Flow

Builds the hash database from known-good source code. Four mutually exclusive modes:

```
corediff db add -d <db>
    │
    ├── --packagist vendor/pkg ─────────────────────────────────┐
    │                                                           │
    ├── --composer /path/composer.json ─────────────────┐       │
    │                                                   │       │
    ├── --update ───────────────────────────┐           │       │
    │                                       │           │       │
    ├── <git-url> ──────────────┐           │           │       │
    │                           │           │           │       │
    └── <local-path> ─┐        │           │           │       │
                      │        │           │           │       │
                      ▼        ▼           ▼           ▼       ▼
               ┌──────────────────────────────────────────────────┐
               │              Load manifest                       │
               │              (tracks package@version pairs)      │
               │              Skip already-indexed versions       │
               └──────────────────────────────────────────────────┘
                                     │
                                     ▼
               ┌──────────────────────────────────────────────────┐
               │              Source resolution                    │
               │                                                  │
               │  Local path:  filepath.Walk directly             │
               │  Git URL:     clone → discover version tags      │
               │  Packagist:   API → git clone or zip download    │
               │  Composer:    lock file → source/dist/repo API   │
               │  Update:      manifest pkgs → packagist API      │
               └──────────────────────────────────────────────────┘
                                     │
                                     ▼
               ┌──────────────────────────────────────────────────┐
               │              Per-version indexing                 │
               │                                                  │
               │  For each file in version:                       │
               │    1. Hash path:  xxhash64("path:" + prefix+rel)│
               │    2. Hash lines: normalize → xxhash64           │
               │    3. Blob dedup: skip unchanged files via       │
               │       git blob hash (across versions)            │
               │    4. Extract composer.json "replace" entries     │
               │       (mark sub-packages as replaced)            │
               └──────────────────────────────────────────────────┘
                                     │
                                     ▼
               ┌──────────────────────────────────────────────────┐
               │  Save: update manifest + write CDDB binary       │
               └──────────────────────────────────────────────────┘
```

## Hash Database Format (CDDB)

```
Offset  Size   Description
──────  ─────  ──────────────────────────
0       4      Magic bytes: "CDDB"
4       4      Version: 1 (uint32 LE)
8       8      Hash count N (uint64 LE)
16      N*8    Hash values (uint64 LE each)

Two types of hashes coexist in the same DB:
  - Line hashes:  xxhash64(normalized_line)
  - Path hashes:  xxhash64("path:" + relative_path)
```

## Normalization & Hashing

```
Raw source line
       │
       ▼
 ┌─────────────┐
 │ Trim space   │
 │ Strip comment│  Lines starting with // # /* * are removed
 │ Regex filter │  Version-specific refs stripped
 └──────┬──────┘
        │
        ▼
  len <= 512B?
   ┌────┴────┐
   │ yes     │ no (minified code)
   ▼         ▼
 xxhash64  Buzhash CDC
 (1 hash)  (content-defined chunking)
            ├─ window: 32 bytes
            ├─ avg chunk: ~128 bytes
            ├─ min: 64, max: 512 bytes
            └─ N hashes per line
```

## Authentication & Config

```
~/.composer/auth.json       Credentials for private repos
  ├─ http-basic             (username/password per host)
  ├─ bearer                 (token per host)
  └─ github-oauth           (GitHub token)

~/.composer/config.json     Additional Composer repositories
  └─ repositories[]         (merged with project composer.json repos)

Search order: cwd/.composer/ → parent dirs → ~/.composer/

HTTP transport chain:
  request → loggingTransport (if -vv) → authTransport → http.DefaultTransport
```

## Concurrency Model

```
Composer/Update mode:
  ┌─────────────────────────────────┐
  │  Semaphore (GOMAXPROCS workers) │
  │                                 │
  │  pkg1 ──→ pkgDB1 ──┐           │
  │  pkg2 ──→ pkgDB2 ──┤  mutex    │
  │  pkg3 ──→ pkgDB3 ──┼────────→ main DB
  │  ...                │           │
  └─────────────────────────────────┘

  - Each package indexed into its own DB
  - Merged under mutex into main DB
  - Manifest writes serialized via flock
  - go-git HTTP transport installed once globally
```
