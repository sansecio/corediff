# Hash Benchmarks

Benchmarked on Apple M2, Go 1.22+, using `fixture/typicalfile.php` (949 lines of Magento PHP).

Source: `internal/normalize/benchmark_test.go`

## Full HashLine pipeline

Includes normalization (trim, comment skip, regex), chunking, and hashing.

| Benchmark                           | Per line | Allocs/line | Notes         |
| ----------------------------------- | -------- | ----------- | ------------- |
| HashLine (before optimizations)     | ~86 ns   | ~1.7        | xxhash64, no guards, slice return |
| **HashLine (after optimizations)**  | **~25 ns** | **0**     | **xxh3, Contains guard, callback API** |

## Pipeline stage breakdown (before optimizations)

| Stage                     | ns/line | Allocs/line | % of total |
| ------------------------- | ------- | ----------- | ---------- |
| TrimSpace                 | ~11     | 0           | 13%        |
| Regex (ReplaceAllLiteral) | ~73     | ~1.9        | **84%**    |
| ChunkLine                 | ~0.6    | 0           | <1%        |
| Hash (xxhash64)           | ~9.2    | 0           | 11%        |
| **Total (HashLine)**      | **~86** | **~1.7**    | **100%**   |

Note: NormalizeLine (trim + comment skip + regex) measures ~69 ns/line total. The regex dominates even though the pattern (`'reference' => '[a-f0-9]{40},'`) rarely matches — Go's regexp engine runs `ReplaceAllLiteral` on every line regardless.

## Line distribution (TypeProcessor.php)

| Category      | Lines | % of total |
| ------------- | ----- | ---------- |
| Empty         | 74    | 8%         |
| Comment       | 375   | 40%        |
| Code < 10 ch  | 175   | 18%        |
| Code >= 10 ch | 325   | 34%        |

Comments + empty lines are already skipped before regex. Of the remaining 500 code lines, 175 (35%) are shorter than 10 characters (braces, `}`, `return;`, etc.) and cannot match the 56-char regex pattern.

## Raw hash: xxhash64 vs XXH3

Pre-normalized lines only (~500 non-empty lines), pure hash call, zero allocations.

| Implementation               | Per line | Relative     |
| ---------------------------- | -------- | ------------ |
| cespare/xxhash/v2 (xxhash64) | ~9.2 ns  | baseline     |
| zeebo/xxh3 (XXH3)            | ~6.3 ns  | 1.45x faster |

## Regex guard strategies (measured)

All strategies operate on the regex stage only (~73 ns/line baseline).

| Strategy                      | ns/line | Allocs/line | Reduction |
| ----------------------------- | ------- | ----------- | --------- |
| Baseline (always run regex)   | ~73     | ~1.9        | —         |
| Skip lines < 10 chars         | ~47     | ~1.1        | 35%       |
| `bytes.Contains` prefix guard | **~8**  | **0**       | **89%**   |

The short-line skip helps (35% less regex work) but still runs the regex on 325 lines that will never match. The `bytes.Contains("'reference'")` guard is far more effective — it eliminates regex calls entirely for files that don't contain the pattern, dropping from 73 ns to 8 ns/line with zero allocations.

## Implemented optimizations

All optimizations below have been implemented. Final result: **~25 ns/line, 0 allocs** (down from ~86 ns, ~1.7 allocs).

### 1. `bytes.Contains` regex guard (biggest win)

The regex `ReplaceAllLiteral` accounted for **84%** of pipeline time (~73 ns/line, ~1.9 allocs/line) despite almost never matching. Added a `bytes.Contains(b, "'reference' =>")` guard in `Line()` — the regex only runs if the line contains the literal substring. Measured at ~8 ns/line for the regex stage (89% reduction), zero allocs.

### 2. Skip short lines (minSize = 10)

Lines shorter than 10 characters skip comment checking and regex entirely in `Line()`. `HashLine()` also returns early for short raw/normalized lines, avoiding unnecessary function calls. Combined with the Contains guard, this is redundant for regex but saves the comment-prefix checks on 18% of lines.

### 3. Callback API for HashLine (eliminated allocations)

Changed `HashLine` from returning `[]uint64` to a callback: `func HashLine(raw []byte, fn func(uint64) bool)`. This eliminates the per-line slice allocation entirely. For non-minified lines (vast majority, <= 512 bytes), the fast path hashes directly without calling `ChunkLine`.

### 4. Switch to XXH3

Replaced `cespare/xxhash/v2` with `zeebo/xxh3`. 1.45x faster raw hashing (~6.3 ns vs ~9.2 ns/line). Required DB rebuild since hash values changed.

### Summary

| Optimization                  | Measured savings                 | Status      |
| ----------------------------- | -------------------------------- | ----------- |
| `bytes.Contains` regex guard  | ~65 ns/line (75% of pipeline)    | implemented |
| Skip short lines (minSize=10) | minor (redundant with guard)     | implemented |
| Callback API (no slice alloc) | ~7 ns/line + 0 allocs            | implemented |
| Switch to XXH3                | ~3 ns/line                       | implemented |

### Why the original ~21 ns prediction was wrong

The original estimate simply subtracted the regex cost (~65 ns) from the baseline (~86 ns) to get ~21 ns. This missed:
- **Function call overhead**: `HashLine` → `Line` → `bytes.Contains` adds ~3-4 ns/line even when no regex runs
- **Contains guard cost**: `bytes.Contains` scanning each line costs ~2-3 ns/line (cheap but not free)
- **Pipeline overhead**: TrimSpace, comment prefix checks, and the callback dispatch add up
- The actual floor with all optimizations is **~25 ns/line** — a **3.4x improvement** from the original ~86 ns
