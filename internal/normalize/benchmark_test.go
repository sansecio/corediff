package normalize

import (
	"bufio"
	"bytes"
	"os"
	"testing"

	"github.com/gwillem/corediff/internal/chunker"
	"github.com/zeebo/xxh3"
)

func loadFixtureLines(b *testing.B) [][]byte {
	b.Helper()
	f, err := os.Open("../../fixture/typicalfile.php")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, append([]byte{}, scanner.Bytes()...))
	}
	if err := scanner.Err(); err != nil {
		b.Fatal(err)
	}
	return lines
}

// Full pipeline: normalize + chunk + hash
func BenchmarkHashLine(b *testing.B) {
	lines := loadFixtureLines(b)
	b.ResetTimer()
	for range b.N {
		for _, line := range lines {
			HashLine(line, func(uint64) bool { return true })
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(lines)), "ns/line")
}

// Stage 1: Line normalization only (TrimSpace + comment skip + regex)
func BenchmarkNormalizeLine(b *testing.B) {
	lines := loadFixtureLines(b)
	b.ResetTimer()
	for range b.N {
		for _, line := range lines {
			Line(line)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(lines)), "ns/line")
}

// Stage 1a: TrimSpace only
func BenchmarkTrimSpace(b *testing.B) {
	lines := loadFixtureLines(b)
	b.ResetTimer()
	for range b.N {
		for _, line := range lines {
			bytes.TrimSpace(line)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(lines)), "ns/line")
}

// Stage 1b: Regex replacement only (pre-trimmed input)
func BenchmarkRegex(b *testing.B) {
	lines := loadFixtureLines(b)
	var trimmed [][]byte
	for _, line := range lines {
		trimmed = append(trimmed, bytes.TrimSpace(line))
	}
	b.ResetTimer()
	for range b.N {
		for _, line := range trimmed {
			for _, rx := range normalizeRx {
				rx.ReplaceAllLiteral(line, nil)
			}
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(trimmed)), "ns/line")
}

// Stage 2: ChunkLine only (pre-normalized input)
func BenchmarkChunkLine(b *testing.B) {
	lines := loadFixtureLines(b)
	var normalized [][]byte
	for _, line := range lines {
		n := Line(line)
		if len(n) > 0 {
			normalized = append(normalized, n)
		}
	}
	b.ResetTimer()
	for range b.N {
		for _, line := range normalized {
			chunker.ChunkLine(line)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(normalized)), "ns/line")
}

// Stage 3: Raw hash comparison (pre-normalized input)
func BenchmarkHash(b *testing.B) {
	lines := loadFixtureLines(b)
	var normalized [][]byte
	for _, line := range lines {
		n := Line(line)
		if len(n) > 0 {
			normalized = append(normalized, n)
		}
	}
	b.ResetTimer()
	for range b.N {
		for _, line := range normalized {
			xxh3.Hash(line)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(normalized)), "ns/line")
}

// Allocation profile: HashLine callback
func BenchmarkHashLineAlloc(b *testing.B) {
	lines := loadFixtureLines(b)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		for _, line := range lines {
			HashLine(line, func(uint64) bool { return true })
		}
	}
}

func loadFixtureLinesT(t *testing.T) [][]byte {
	t.Helper()
	f, err := os.Open("../../fixture/typicalfile.php")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, append([]byte{}, scanner.Bytes()...))
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}

func TestLineDistribution(t *testing.T) {
	lines := loadFixtureLinesT(t)

	skips := [][]byte{
		[]byte("*"), []byte("/*"), []byte("//"), []byte("#"),
	}

	var total, empty, comment, shortCode, longCode int
	total = len(lines)

	for _, raw := range lines {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 {
			empty++
			continue
		}
		isComment := false
		for _, s := range skips {
			if bytes.HasPrefix(trimmed, s) {
				isComment = true
				break
			}
		}
		if isComment {
			comment++
			continue
		}
		if len(trimmed) < 10 {
			shortCode++
		} else {
			longCode++
		}
	}

	t.Logf("Total lines:    %d", total)
	t.Logf("Empty:          %d (%.0f%%)", empty, 100*float64(empty)/float64(total))
	t.Logf("Comment:        %d (%.0f%%)", comment, 100*float64(comment)/float64(total))
	t.Logf("Code < 10 ch:   %d (%.0f%%)", shortCode, 100*float64(shortCode)/float64(total))
	t.Logf("Code >= 10 ch:  %d (%.0f%%)", longCode, 100*float64(longCode)/float64(total))
	t.Logf("Lines hitting regex (code >= 10): %d (%.0f%%)", longCode, 100*float64(longCode)/float64(total))
}

// Benchmark: skip regex with bytes.Contains prefix guard
func BenchmarkRegexWithContainsGuard(b *testing.B) {
	lines := loadFixtureLines(b)
	var trimmed [][]byte
	for _, line := range lines {
		trimmed = append(trimmed, bytes.TrimSpace(line))
	}
	needle := []byte("'reference'")
	b.ResetTimer()
	for range b.N {
		for _, line := range trimmed {
			if !bytes.Contains(line, needle) {
				continue
			}
			for _, rx := range normalizeRx {
				rx.ReplaceAllLiteral(line, nil)
			}
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(trimmed)), "ns/line")
}

// Benchmark: skip regex for lines < 10 chars (can't match the 56-char pattern)
func BenchmarkRegexWithShortSkip(b *testing.B) {
	lines := loadFixtureLines(b)
	var trimmed [][]byte
	for _, line := range lines {
		trimmed = append(trimmed, bytes.TrimSpace(line))
	}
	b.ResetTimer()
	for range b.N {
		for _, line := range trimmed {
			if len(line) < 10 {
				continue
			}
			for _, rx := range normalizeRx {
				rx.ReplaceAllLiteral(line, nil)
			}
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(len(trimmed)), "ns/line")
}
