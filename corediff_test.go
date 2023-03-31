package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func digest(b uint32) string {
	return fmt.Sprintf("%x", b)
}

func Test_parseFile(t *testing.T) {
	hdb := hashDB{}
	updateDB := true
	hits, lines := parseFile("fixture/docroot/odd-encoding.js", "n/a", hdb, updateDB)
	assert.Equal(t, 220, len(hdb))
	assert.Equal(t, 220, len(hits))
	assert.Equal(t, 471, len(lines))
}

func Test_hash(t *testing.T) {
	tests := []struct {
		args []byte
		want string
	}{
		{[]byte("banaan"), "14ac6691"},
	}
	for _, tt := range tests {
		t.Run(string(tt.args), func(t *testing.T) {
			if got := digest(hash(tt.args)); got != tt.want {
				t.Errorf("hash() = %x (%v), want %x", got, got, tt.want)
			}
		})
	}
}

func Test_vendor_bug(t *testing.T) {
	db := loadDB("fixture/sample.db")
	assert.Len(t, db, 238)
	wantHash := uint32(3333369281)
	if _, ok := db[wantHash]; !ok {
		t.Error("hash not in db")
	}
}
func Test_Corruption(t *testing.T) {
	fh, _ := os.Open("fixture/docroot/sample")
	defer fh.Close()

	lines := [][]byte{}

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		x := scanner.Bytes()
		l := make([]byte, len(x))
		copy(l, x)
		lines = append(lines, l)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}
