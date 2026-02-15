package chunker

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkLineBelowThreshold(t *testing.T) {
	short := []byte("var x = 42;")
	chunks := ChunkLine(short)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, short, chunks[0])
}

func TestChunkLineExactlyAtThreshold(t *testing.T) {
	line := bytes.Repeat([]byte("a"), ChunkThreshold)
	chunks := ChunkLine(line)
	assert.Equal(t, 1, len(chunks), "line exactly at threshold should not be chunked")
}

func TestChunkLineAboveThreshold(t *testing.T) {
	line := bytes.Repeat([]byte("x"), ChunkThreshold+1)
	chunks := ChunkLine(line)
	assert.Greater(t, len(chunks), 1, "line above threshold should be chunked")
}

func TestChunkLineDeterministic(t *testing.T) {
	line := bytes.Repeat([]byte("hello world; "), 100)
	chunks1 := ChunkLine(line)
	chunks2 := ChunkLine(line)
	assert.Equal(t, len(chunks1), len(chunks2))
	for i := range chunks1 {
		assert.Equal(t, chunks1[i], chunks2[i], "chunk %d differs", i)
	}
}

func TestChunkLineCoversAllInput(t *testing.T) {
	line := bytes.Repeat([]byte("function foo(bar,baz){return bar+baz;};"), 50)
	chunks := ChunkLine(line)
	var reassembled []byte
	for _, c := range chunks {
		reassembled = append(reassembled, c...)
	}
	assert.Equal(t, line, reassembled, "chunks must reassemble to original")
}

func TestChunkLineSizeBounds(t *testing.T) {
	line := bytes.Repeat([]byte("var x=Math.random()*100;"), 100)
	chunks := ChunkLine(line)
	for i, c := range chunks {
		if i < len(chunks)-1 {
			assert.GreaterOrEqual(t, len(c), minChunk, "chunk %d too small: %d", i, len(c))
			assert.LessOrEqual(t, len(c), maxChunk, "chunk %d too large: %d", i, len(c))
		}
		// Last chunk may be smaller than minChunk (remainder)
	}
}

func TestChunkLineStability(t *testing.T) {
	original := bytes.Repeat([]byte("var result=calculate(a,b,c);"), 100)
	modified := make([]byte, len(original))
	copy(modified, original)
	// Flip a byte in the middle
	modified[len(modified)/2] ^= 0xFF

	origChunks := ChunkLine(original)
	modChunks := ChunkLine(modified)

	origSet := make(map[string]bool)
	for _, c := range origChunks {
		origSet[string(c)] = true
	}
	changed := 0
	for _, c := range modChunks {
		if !origSet[string(c)] {
			changed++
		}
	}
	// Most chunks should survive a single byte change
	assert.Less(t, changed, len(modChunks)/2,
		"too many chunks changed: %d/%d", changed, len(modChunks))
}

func TestChunkLineEmpty(t *testing.T) {
	chunks := ChunkLine(nil)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, []byte(nil), chunks[0])
}

func TestChunkLinePreferCommas(t *testing.T) {
	data, err := os.ReadFile("testdata/original.php")
	require.NoError(t, err)

	// Find the long line (line 14, the PHP array)
	var longLine []byte
	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > ChunkThreshold {
			longLine = trimmed
			break
		}
	}
	require.NotNil(t, longLine, "no long line found in fixture")

	chunks := ChunkLine(longLine)
	t.Logf("%d chunks from %d byte line:", len(chunks), len(longLine))
	for i, c := range chunks {
		t.Logf("  chunk %2d: %3d bytes  %.60q", i, len(c), c)
	}

	// Most chunks should end with a comma (preferred boundary)
	commaEndings := 0
	for _, c := range chunks[:len(chunks)-1] { // skip last chunk (remainder)
		if c[len(c)-1] == ',' {
			commaEndings++
		}
	}
	assert.Greater(t, commaEndings, len(chunks)/2,
		"most chunks should end at comma boundaries")
}

func TestChunkLineModifiedDetection(t *testing.T) {
	original, err := os.ReadFile("testdata/original.php")
	require.NoError(t, err)
	modified, err := os.ReadFile("testdata/modified.php")
	require.NoError(t, err)

	// Find the long lines
	findLong := func(data []byte) []byte {
		for _, line := range bytes.Split(data, []byte("\n")) {
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) > ChunkThreshold {
				return trimmed
			}
		}
		return nil
	}
	origLine := findLong(original)
	modLine := findLong(modified)
	require.NotNil(t, origLine)
	require.NotNil(t, modLine)

	origChunks := ChunkLine(origLine)
	modChunks := ChunkLine(modLine)

	origSet := make(map[string]bool)
	for _, c := range origChunks {
		origSet[string(c)] = true
	}
	var mismatched []string
	for _, c := range modChunks {
		if !origSet[string(c)] {
			mismatched = append(mismatched, string(c))
		}
	}

	// Should detect exactly the chunks containing PROD-666 and 29.99
	assert.LessOrEqual(t, len(mismatched), 3, "should have few mismatched chunks")
	found := map[string]bool{"PROD-666": false, "29.99": false}
	for _, m := range mismatched {
		for key := range found {
			if bytes.Contains([]byte(m), []byte(key)) {
				found[key] = true
			}
		}
	}
	for key, ok := range found {
		assert.True(t, ok, "mismatched chunks should contain %q", key)
	}
}

func TestChunkLineRealFile(t *testing.T) {
	data, err := os.ReadFile("../../fixture/docroot/editor_plugin.js")
	if err != nil {
		t.Skip("fixture not available")
	}
	// This file is a single long minified line
	require.Greater(t, len(data), ChunkThreshold)

	chunks := ChunkLine(data)
	assert.Greater(t, len(chunks), 10, "should produce many chunks for 6KB minified file")

	// Verify reassembly
	var reassembled []byte
	for _, c := range chunks {
		reassembled = append(reassembled, c...)
	}
	assert.Equal(t, data, reassembled)
}
