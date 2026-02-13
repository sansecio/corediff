# Content-Defined Chunking (CDC) for Minified Files

Minified JavaScript/JSON files pack thousands of statements onto a single line.
Line-by-line hashing produces a single fragile hash that breaks on any change.
CDC splits long lines into variable-size chunks using a Buzhash rolling hash,
so small edits only affect nearby chunks.

## Chosen Parameters

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Window | 32 bytes | Buzhash rolling window size |
| Mask | 0x7F | Average chunk ~128 bytes |
| Min chunk | 64 bytes | Prevent degenerate splits on short tokens |
| Max chunk | 512 bytes | Prevent huge chunks on low-entropy data |
| Threshold | 512 bytes | Lines <= 512 bytes are not chunked |

## Benchmark Results

Tested on three real-world minified JavaScript files with `TestCDCParameters`:

### editor_plugin.js (6,894 bytes, 1 line)

```
params                                 chunks    avg    med    min    max p10-90    stab1    stab5
win=32/mask=0x1F/min=16/max=128           163     42     36     16    128  19-73      1.8%    11.7%
win=32/mask=0x1F/min=16/max=256           160     43     36     16    210  19-74      1.9%     8.8%
win=32/mask=0x3F/min=16/max=256            97     71     53     17    256  21-146     3.1%    13.4%
win=32/mask=0x3F/min=32/max=256            82     84     69     32    256  35-166     3.7%    15.9%
win=32/mask=0x3F/min=32/max=512            82     84     69     32    269  35-166     3.7%    14.6%
win=32/mask=0x7F/min=32/max=256            55    125     92     33    256  37-256     3.6%    14.5%
win=32/mask=0x7F/min=32/max=512            51    135     92     33    443  39-261     3.9%    17.6%
win=32/mask=0x7F/min=64/max=512            42    164    131     45    443  70-270     4.8%    18.6%  <-- chosen
win=32/mask=0xFF/min=32/max=512            31    222    193     22    512  64-419     6.5%    19.4%
win=32/mask=0xFF/min=64/max=1024           27    255    218     22    651  74-589     7.4%    22.2%
win=32/mask=0x1FF/min=64/max=1024          14    492    504     22   1024  64-1024     7.1%    35.7%
```

### jquery.min.js (87,533 bytes, 3 lines)

```
params                                 chunks    avg    med    min    max p10-90    stab1    stab5
win=32/mask=0x1F/min=16/max=128          1872     47     37     16    128  19-91      0.1%     0.8%
win=32/mask=0x1F/min=16/max=256          1849     47     37     16    256  19-91      0.1%     0.9%
win=32/mask=0x3F/min=16/max=256          1141     77     58     16    256  24-158     0.1%     1.0%
win=32/mask=0x3F/min=32/max=256           955     92     73     32    256  38-174     0.1%     0.8%
win=32/mask=0x3F/min=32/max=512           932     94     73     32    512  38-174     0.1%     0.9%
win=32/mask=0x7F/min=32/max=256           623    141    124     32    256  45-256     0.2%     1.3%
win=32/mask=0x7F/min=32/max=512           541    162    124     32    512  44-356     0.2%     1.3%
win=32/mask=0x7F/min=64/max=512           447    196    159     64    512  77-375     0.2%     1.3%  <-- chosen
win=32/mask=0xFF/min=32/max=512           343    255    211     32    512  58-512     0.3%     1.7%
win=32/mask=0xFF/min=64/max=1024          263    333    242     65   1024 100-697     0.4%     2.3%
win=32/mask=0x1FF/min=64/max=1024         171    512    432     69   1024 125-1024     0.6%     2.9%
```

### knockout.min.js (67,224 bytes, 1 line)

```
params                                 chunks    avg    med    min    max p10-90    stab1    stab5
win=32/mask=0x1F/min=16/max=128          1455     46     39     16    128  18-85      0.5%     1.2%
win=32/mask=0x1F/min=16/max=256          1423     47     39     16    256  19-84      0.6%     1.2%
win=32/mask=0x3F/min=16/max=256           899     75     58     10    256  21-153     0.2%     1.2%
win=32/mask=0x3F/min=32/max=256           744     90     76     32    256  37-170     0.3%     1.1%
win=32/mask=0x3F/min=32/max=512           730     92     76     32    430  37-171     0.3%     1.1%
win=32/mask=0x7F/min=32/max=256           487    138    116     32    256  44-256     0.6%     2.0%
win=32/mask=0x7F/min=32/max=512           423    159    121     32    512  45-323     0.5%     1.7%
win=32/mask=0x7F/min=64/max=512           349    193    158     64    512  79-350     0.3%     1.7%  <-- chosen
win=32/mask=0xFF/min=32/max=512           268    251    212     32    512  58-512     1.1%     2.6%
win=32/mask=0xFF/min=64/max=1024          212    317    244     65   1024  88-665     0.5%     2.4%
win=32/mask=0x1FF/min=64/max=1024         136    494    431     77   1024 138-1024     1.5%     4.4%
```

## Stability Detail (jquery.min.js, mask=0x7F/min=64/max=512)

```
Original: 447 chunks, Modified: 447 chunks
Chunks not found in original: 1 (0.2%)
Chunks spanning edit point: 1 (theoretical minimum changes)
Locality ratio: 1.0 (lower=better, 1.0=perfect)
```

A single byte flip in the middle of an 87KB file affects exactly 1 out of 447
chunks — the theoretical minimum. The rolling hash correctly re-synchronizes
after the edit.

## Column Definitions

- **chunks**: number of chunks produced
- **avg/med/min/max**: chunk size statistics in bytes
- **p10-90**: 10th and 90th percentile chunk sizes
- **stab1**: percentage of chunks affected by 1 byte modification
- **stab5**: percentage of chunks affected by 5 byte modifications
