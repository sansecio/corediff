// Package chunker provides content-defined chunking (CDC) for minified files.
// Lines longer than ChunkThreshold are split into variable-size chunks using a
// Buzhash rolling hash, so that small edits only affect nearby chunks.
package chunker

const (
	windowSize     = 32
	mask           = 0x3F // average chunk ~64 bytes
	minChunk       = 32
	maxChunk       = 128
	ChunkThreshold = 512 // lines <= this length are not chunked
)

// buzhash byte-to-hash table
var hashTable [256]uint64

// outTable stores pre-rotated values for the outgoing byte
var outTable [256]uint64

func init() {
	x := uint64(0x123456789abcdef0)
	for i := range hashTable {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		hashTable[i] = x
	}
	for i := range outTable {
		outTable[i] = rotateLeft(hashTable[i], windowSize)
	}
}

func rotateLeft(h uint64, n int) uint64 {
	for range n {
		h = (h << 1) | (h >> 63)
	}
	return h
}

// ChunkLine splits a line into content-defined chunks if it exceeds
// ChunkThreshold. Short lines are returned as-is in a single-element slice.
func ChunkLine(line []byte) [][]byte {
	if len(line) <= ChunkThreshold {
		return [][]byte{line}
	}
	return chunk(line)
}

func chunk(data []byte) [][]byte {
	if len(data) <= minChunk {
		return [][]byte{data}
	}

	var chunks [][]byte
	var hash uint64
	start := 0

	for i := range len(data) {
		hash = (hash << 1) | (hash >> 63)
		hash ^= hashTable[data[i]]

		posInChunk := i - start
		if posInChunk >= windowSize {
			hash ^= outTable[data[i-windowSize]]
		}

		chunkLen := posInChunk + 1
		if chunkLen < minChunk {
			continue
		}
		if chunkLen >= maxChunk || (hash&mask == 0) || data[i] == ',' {
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
