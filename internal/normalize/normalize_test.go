package normalize

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gwillem/corediff/internal/hashdb"
	"github.com/stretchr/testify/assert"
)

func TestLine(t *testing.T) {
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
			assert.Equal(t, tt.want, string(Line([]byte(tt.arg))))
		})
	}
}

func TestHashReader(t *testing.T) {
	t.Run("adds hashes for code lines", func(t *testing.T) {
		db := hashdb.New()
		input := "<?php\necho 'hello';\n// comment\n\necho 'world';\n"
		n := HashReader(strings.NewReader(input), db, nil)
		assert.Greater(t, n, 0)
		assert.Greater(t, db.Len(), 0)
	})

	t.Run("skips empty and comment lines", func(t *testing.T) {
		db := hashdb.New()
		input := "// comment\n# another comment\n/* block comment\n\n"
		n := HashReader(strings.NewReader(input), db, nil)
		assert.Equal(t, 0, n)
		assert.Equal(t, 0, db.Len())
	})

	t.Run("does not add duplicate hashes", func(t *testing.T) {
		db := hashdb.New()
		input := "echo 'hello';\necho 'hello';\n"
		n := HashReader(strings.NewReader(input), db, nil)
		// First line adds hash(es), second is duplicate

		assert.Greater(t, n, 0)
		// All hashes from first line should already exist for second
		assert.Equal(t, len(HashLine([]byte("echo 'hello';"))), db.Len())
	})

	t.Run("returns count of new hashes", func(t *testing.T) {
		db := hashdb.New()
		input := "line1;\nline2;\n"
		n := HashReader(strings.NewReader(input), db, nil)
		expected := len(HashLine([]byte("line1;"))) + len(HashLine([]byte("line2;")))
		assert.Equal(t, expected, n)
	})
}

func TestHash(t *testing.T) {
	tests := []struct {
		args []byte
		want string
	}{
		{[]byte("banaan"), "acfb1ff4438e39f3"},
	}
	for _, tt := range tests {
		t.Run(string(tt.args), func(t *testing.T) {
			got := fmt.Sprintf("%016x", Hash(tt.args))
			assert.Equal(t, tt.want, got)
		})
	}
}
