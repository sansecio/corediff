# Concurrency in Corediff

Corediff parallelizes package indexing — the most time-consuming operation — using a consistent worker pool pattern throughout `cmd/corediff/db_add.go`. The scanning/diffing path is entirely sequential.

## Overview

```
executeComposer / executeUpdate
│
├─ InstallHTTPTransport()          ← once, before any goroutines
│
├─ sem = make(chan struct{}, GOMAXPROCS)
│
├─ for each work item:
│   │
│   ├─ sem <- ∅                    ← block if pool full
│   │
│   └─ go ──┐
│            │  localDB := New()   ← isolated, no lock needed
│            │  clone/fetch repo
│            │  hash all files
│            │  ──── mu.Lock() ────
│            │  sharedDB.Merge()
│            │  ──── mu.Unlock() ──
│            │  mf.MarkIndexed()   ← manifest has its own mutex
│            │  <-sem              ← release slot
│            └──────────────────
│
└─ wg.Wait()
   db.Save()                       ← single goroutine, after all done
```

What runs concurrently (up to GOMAXPROCS goroutines at a time):

```
time ──────────────────────────────────────────────────────────►

main:  [setup] ──wait──────────────────────────────────── [save]

         ┌─ goroutine 1: [clone repo A] [hash files] [merge]
         ├─ goroutine 2: [clone repo B] [hash files] [merge]
sem=4    ├─ goroutine 3: [clone repo C] [hash files] [merge]
         ├─ goroutine 4: [clone repo D] [hash files] [merge]
         │                              ↑ slot freed, next starts
         ├─ goroutine 5: [clone repo E] [hash files] [merge]
         └─ ...
```

## Worker Pool Pattern

All three concurrent code paths use the same structure:

```go
var (
    mu  sync.Mutex
    wg  sync.WaitGroup
    sem = make(chan struct{}, runtime.GOMAXPROCS(0))
)

for _, item := range work {
    sem <- struct{}{}   // acquire semaphore slot (blocks if pool is full)
    wg.Add(1)
    go func() {
        defer wg.Done()
        defer func() { <-sem }()  // release slot

        localDB := hashdb.New()        // isolated per-goroutine database
        doExpensiveWork(item, localDB)  // git clone, hash files, etc.

        mu.Lock()
        sharedDB.Merge(localDB)        // merge results into shared state
        mu.Unlock()
    }()
}
wg.Wait()
```

**Bounded concurrency:** A buffered channel of size `runtime.GOMAXPROCS(0)` acts as a counting semaphore, capping active goroutines to the number of CPU cores. The send blocks when the channel is full; the deferred receive frees a slot.

**Data isolation:** Each goroutine allocates its own `hashdb.HashDB`. Only the final `Merge()` into the shared database requires a mutex. This minimizes lock contention — most CPU time is spent in unlocked git/hash work.

## Where It's Used

| Function | File | What's parallelized |
|---|---|---|
| `executeComposer` | `db_add.go:362-388` | Indexing packages from `composer.lock` |
| `executeUpdate` (Packagist) | `db_add.go:450-504` | Checking + indexing new Packagist versions |
| `executeUpdate` (Git URLs) | `db_add.go:507-521` | Fetching + indexing new tags from git repos |

The Packagist and git URL loops in `executeUpdate` share the same semaphore and WaitGroup, so they collectively stay within the concurrency limit.

## Manifest (Thread-Safe Shared State)

`internal/manifest/manifest.go` is accessed from goroutines to record indexed versions. Two layers of synchronization:

1. **`sync.Mutex`** — protects in-memory maps and file writes. Every public method acquires the lock.
2. **`syscall.Flock` (LOCK_EX)** — exclusive file lock for cross-process safety, acquired on `Load()`, released on `Close()`.

## Error Handling

Goroutine errors are logged to stderr as warnings. No `context.Context` cancellation, no `errgroup`, no early abort — all goroutines run to completion independently.

## What's Not Concurrent

- **Scanning** (`executeScan`) — sequential filesystem walk.
- **Database I/O** (`db.Save`, `db.Load`) — main goroutine only, after workers finish.
- **Single git URL indexing** (`executeGitURL`) — see below.

## Why Git Tag Processing Is Sequential

`executeGitURL` (and `indexRefs` inside `gitindex`) processes tags one at a time, newest-first. This is intentional — parallelizing across tags would be counterproductive.

The key is `seenBlobs`: a map of git blob hashes already processed. When version 2.4.7 has been indexed, version 2.4.6 skips every file whose blob hash matches (i.e. the file content is identical). For a repo like magento2 where ~95% of files are unchanged between adjacent versions, this avoids re-reading and re-hashing the same content hundreds of times.

```
sequential (current):

  v2.4.7  [hash 10,000 files]
  v2.4.6  [hash 500 changed files, skip 9,500 via seenBlobs]
  v2.4.5  [hash 300 changed files, skip 9,700 via seenBlobs]
  ...

concurrent (hypothetical, without seenBlobs):

  v2.4.7  [hash 10,000 files]  ──┐
  v2.4.6  [hash 10,000 files]  ──┼── same blobs hashed N times
  v2.4.5  [hash 10,000 files]  ──┘
  ...
```

### Why not make `seenBlobs` concurrent?

It's technically possible (`sync.Map` or mutex-protected map) — hashing is idempotent so correctness is preserved regardless of processing order. But it wouldn't help, because adjacent versions share ~95% of the same files and git trees iterate in the same alphabetical order. With 4 goroutines processing 4 versions simultaneously:

- All 4 hit `A.php` at roughly the same time
- All 4 find it's not yet in `seenBlobs`
- All 4 read and hash it, then mark it seen
- Repeat for `B.php`, `C.php`, ...

Result: ~4x the work on 4 cores ≈ same wall-clock time as sequential, but with added lock contention on every file check. The sequential approach wins because it front-loads all work onto the newest version, then subsequent versions skip 95%+ of files instantly via map lookup (no I/O).

### Other obstacles

- `HashDB` is a plain map (not thread-safe) — solvable with per-goroutine local DBs + merge (the existing pattern from `executeComposer`).
- `go-git`'s `Repository` reads through an internal packfile cache with no documented concurrency guarantees.
- The real bottleneck for single-repo indexing is the initial `clone`/`fetch` — a single network operation that can't be split across tags.
