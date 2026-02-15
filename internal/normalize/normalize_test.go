package normalize

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/stretchr/testify/assert"
)

func TestNormLine(t *testing.T) {
	tests := []struct {
		arg  string
		want string
	}{
		{"\t'reference' => '836ce4bde75ef67a1b4b2230ea725773adca2de7',\n", ""},
		{"reference\n", "reference"},
		{"reference' => '1234567890',", "reference' => '1234567890',"},
	}
	for _, tt := range tests {
		t.Run(string(tt.arg), func(t *testing.T) {
			assert.Equal(t, tt.want, string(normLine([]byte(tt.arg))))
		})
	}
}

func TestHashReader(t *testing.T) {
	t.Run("adds hashes for code lines", func(t *testing.T) {
		db := hashdb.New()
		input := "<?php\necho 'hello';\n// comment\n\necho 'world';\n"
		var added int
		total := HashReader(strings.NewReader(input), func(h uint64, _ []byte) {
			if !db.Contains(h) {
				db.Add(h)
				added++
			}
		}, nil)
		assert.Greater(t, added, 0)
		assert.Greater(t, total, 0)
		assert.Greater(t, db.Len(), 0)
	})

	t.Run("skips empty and comment lines", func(t *testing.T) {
		db := hashdb.New()
		input := "// comment\n# another comment\n/* block comment\n\n"
		total := HashReader(strings.NewReader(input), func(h uint64, _ []byte) {
			db.Add(h)
		}, nil)
		assert.Equal(t, 0, total)
		assert.Equal(t, 0, db.Len())
	})

	t.Run("does not add duplicate hashes", func(t *testing.T) {
		db := hashdb.New()
		input := "echo 'hello';\necho 'hello';\n"
		var added int
		total := HashReader(strings.NewReader(input), func(h uint64, _ []byte) {
			if !db.Contains(h) {
				db.Add(h)
				added++
			}
		}, nil)
		// First line adds hash(es), second is duplicate

		assert.Greater(t, added, 0)
		assert.Greater(t, total, added) // total includes duplicates
		// All hashes from first line should already exist for second
		count := 0
		HashLine([]byte("echo 'hello';"), func(uint64, []byte) bool { count++; return true })
		assert.Equal(t, count, db.Len())
	})

	t.Run("returns count of new hashes", func(t *testing.T) {
		db := hashdb.New()
		input := "line1line1;\nline2line2;\n"
		var added int
		HashReader(strings.NewReader(input), func(h uint64, _ []byte) {
			if !db.Contains(h) {
				db.Add(h)
				added++
			}
		}, nil)
		expected := 0
		HashLine([]byte("line1line1;"), func(uint64, []byte) bool { expected++; return true })
		HashLine([]byte("line2line2;"), func(uint64, []byte) bool { expected++; return true })
		assert.Equal(t, expected, added)
	})
}

func TestHashSanity(t *testing.T) {
	tests := []struct {
		args []byte
		want string
	}{
		{[]byte("banaan"), "bb9aa85f787ea9ad"},
	}
	for _, tt := range tests {
		t.Run(string(tt.args), func(t *testing.T) {
			got := fmt.Sprintf("%016x", hashFunc(tt.args))
			assert.Equal(t, tt.want, got)
		})
	}
}
