package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func digest(b uint64) string {
	return fmt.Sprintf("%x", b)
}

func Test_parseFile(t *testing.T) {
	hdb := hashDB{}
	updateDB := true
	hits, lines := parseFile("fixture/docroot/odd-encoding.js", hdb, updateDB)
	assert.Equal(t, 220, len(hdb))
	assert.Equal(t, 220, len(hits))
	assert.Equal(t, 220, len(lines))
}

func Test_hash(t *testing.T) {
	tests := []struct {
		args []byte
		want string
	}{
		{[]byte("banaan"), "acfb1ff4438e39f3"},
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
	wantHash := uint64(3900178074848893275)
	if _, ok := db[wantHash]; !ok {
		t.Error("hash not in db")
	}
}

// Too slow to run in testing.B
// func Test_loadFile(t *testing.T) {
// 	for i := 0; i < 10; i++ {
// 		start := time.Now()
// 		loadDB("m2.db") // pre-allocating fixed map size saves 20% time
// 		fmt.Println(time.Since(start))
// 	}
// }
