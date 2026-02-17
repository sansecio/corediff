package chunker

import (
	"fmt"
	"math"
	"os"
	"sort"
	"testing"
)

// Benchmark different CDC parameter combinations on real-world data.

type cdcParams struct {
	windowSize int
	mask       uint64
	minChunk   int
	maxChunk   int
}

func (p cdcParams) String() string {
	return fmt.Sprintf("win=%d/mask=0x%X/min=%d/max=%d", p.windowSize, p.mask, p.minChunk, p.maxChunk)
}

// buzhash table and precomputed outgoing rotations per window size
var benchTable [256]uint64

func init() {
	x := uint64(0x123456789abcdef0)
	for i := range benchTable {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		benchTable[i] = x
	}
}

func rotateN(h uint64, n int) uint64 {
	for range n {
		h = (h << 1) | (h >> 63)
	}
	return h
}

func benchChunk(data []byte, p cdcParams) [][]byte {
	if len(data) <= p.minChunk {
		return [][]byte{data}
	}

	// Precompute outgoing byte table for this window size
	var outTable [256]uint64
	for i := range outTable {
		outTable[i] = rotateN(benchTable[i], p.windowSize)
	}

	var chunks [][]byte
	var hash uint64
	start := 0

	for i := range len(data) {
		hash = (hash << 1) | (hash >> 63)
		hash ^= benchTable[data[i]]

		posInChunk := i - start
		if posInChunk >= p.windowSize {
			hash ^= outTable[data[i-p.windowSize]]
		}

		chunkLen := posInChunk + 1
		if chunkLen < p.minChunk {
			continue
		}
		if chunkLen >= p.maxChunk || (hash&p.mask == 0) {
			chunks = append(chunks, data[start:i+1])
			start = i + 1
			hash = 0
		}
	}
	if start < len(data) {
		chunks = append(chunks, data[start:])
	}
	return chunks
}

type chunkStats struct {
	count   int
	avgSize float64
	medSize int
	minSize int
	maxSize int
	p10     int
	p90     int
}

func calcStats(chunks [][]byte) chunkStats {
	if len(chunks) == 0 {
		return chunkStats{}
	}
	sizes := make([]int, len(chunks))
	total := 0
	for i, c := range chunks {
		sizes[i] = len(c)
		total += len(c)
	}
	sort.Ints(sizes)
	return chunkStats{
		count:   len(chunks),
		avgSize: float64(total) / float64(len(chunks)),
		medSize: sizes[len(sizes)/2],
		minSize: sizes[0],
		maxSize: sizes[len(sizes)-1],
		p10:     sizes[len(sizes)/10],
		p90:     sizes[len(sizes)*9/10],
	}
}

func stabilityScore(data []byte, p cdcParams, modifications int) float64 {
	original := benchChunk(data, p)
	origHashes := make(map[string]int)
	for _, c := range original {
		origHashes[string(c)]++
	}

	modified := make([]byte, len(data))
	copy(modified, data)
	step := len(data) / (modifications + 1)
	for i := 0; i < modifications; i++ {
		pos := step * (i + 1)
		modified[pos] ^= 0xFF
	}

	newChunks := benchChunk(modified, p)
	newHashes := make(map[string]int)
	for _, c := range newChunks {
		newHashes[string(c)]++
	}

	unchanged := 0
	for h, count := range origHashes {
		if newCount, ok := newHashes[h]; ok {
			if newCount < count {
				unchanged += newCount
			} else {
				unchanged += count
			}
		}
	}
	total := len(original)
	if len(newChunks) > total {
		total = len(newChunks)
	}
	return float64(total-unchanged) / float64(total)
}

func TestCDCParameters(t *testing.T) {
	files := []string{
		"../../fixture/docroot/editor_plugin.js",
		"../../fixture/docroot/jquery.min.js",
		"../../fixture/docroot/knockout.min.js",
	}

	params := []cdcParams{
		{32, 0x1F, 16, 128},
		{32, 0x1F, 16, 256},
		{32, 0x3F, 16, 256},
		{32, 0x3F, 32, 256},
		{32, 0x3F, 32, 512},
		{32, 0x7F, 32, 256},
		{32, 0x7F, 32, 512},
		{32, 0x7F, 64, 512},
		{32, 0xFF, 32, 512},
		{32, 0xFF, 64, 1024},
		{32, 0x1FF, 64, 1024},
	}

	for _, fname := range files {
		data, err := os.ReadFile(fname)
		if err != nil {
			t.Logf("Skipping %s: %v", fname, err)
			continue
		}

		maxLine := 0
		lineCount := 1
		cur := 0
		for _, b := range data {
			if b == '\n' {
				if cur > maxLine {
					maxLine = cur
				}
				lineCount++
				cur = 0
			} else {
				cur++
			}
		}
		if cur > maxLine {
			maxLine = cur
		}

		t.Logf("\n=== %s (%d bytes, %d lines, longest line: %d bytes) ===", fname, len(data), lineCount, maxLine)
		t.Logf("%-38s %6s %6s %6s %6s %6s %6s %8s %8s", "params", "chunks", "avg", "med", "min", "max", "p10-90", "stab1", "stab5")

		for _, p := range params {
			chunks := benchChunk(data, p)
			s := calcStats(chunks)
			stab1 := stabilityScore(data, p, 1)
			stab5 := stabilityScore(data, p, 5)
			t.Logf("%-38s %6d %6.0f %6d %6d %6d %3d-%-3d %7.1f%% %7.1f%%",
				p, s.count, s.avgSize, s.medSize, s.minSize, s.maxSize, s.p10, s.p90,
				stab1*100, stab5*100)
		}
	}
}

func TestStabilityDetail(t *testing.T) {
	data, err := os.ReadFile("../../fixture/docroot/jquery.min.js")
	if err != nil {
		t.Skip("jquery.min.js not available")
	}

	p := cdcParams{32, 0x7F, 64, 512}
	original := benchChunk(data, p)

	modified := make([]byte, len(data))
	copy(modified, data)
	modified[len(data)/2] ^= 0xFF

	newChunks := benchChunk(modified, p)

	origSet := make(map[string]bool)
	for _, c := range original {
		origSet[string(c)] = true
	}

	changed := 0
	for _, c := range newChunks {
		if !origSet[string(c)] {
			changed++
		}
	}

	t.Logf("Params: %s", p)
	t.Logf("Original: %d chunks, Modified: %d chunks", len(original), len(newChunks))
	t.Logf("Chunks not found in original: %d (%.1f%%)", changed, float64(changed)/float64(len(newChunks))*100)

	editPos := len(data) / 2
	affected := 0
	pos := 0
	for _, c := range original {
		if pos <= editPos && editPos < pos+len(c) {
			affected++
		}
		pos += len(c)
	}
	t.Logf("Chunks spanning edit point: %d (theoretical minimum changes)", affected)
	t.Logf("Locality ratio: %.1f (lower=better, 1.0=perfect)", float64(changed)/math.Max(float64(affected), 1))
}
