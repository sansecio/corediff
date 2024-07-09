package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pathIsExcluded(t *testing.T) {
	tests := []struct {
		arg  string
		want bool
	}{
		{"generated/x/y/z.php", true},
		{"x/y/z.php", false},
		{"/vendor/x/y/z", false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			if got := pathIsExcluded(tt.arg); got != tt.want {
				t.Errorf("pathIsExcluded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_normalizeLine(t *testing.T) {
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
			assert.Equal(t, tt.want, string(normalizeLine([]byte(tt.arg))))
		})
	}
}

func digest(b uint64) string {
	return fmt.Sprintf("%016x", b)
}

func Test_parseFile(t *testing.T) {
	hdb := hashDB{}
	updateDB := true
	hits, lines := parseFileWithDB("../fixture/docroot/odd-encoding.js", hdb, updateDB)
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
	db, err := loadDB("../fixture/sample.db")
	require.NoError(t, err)
	assert.Len(t, db, 238)
	wantHash := uint64(3900178074848893275)
	if _, ok := db[wantHash]; !ok {
		t.Error("hash not in db")
	}
}

// func Test_Needle(t *testing.T) {
// 	needles := []string{
// 		"path:app/code/Magedelight/GeoIp/Controller/Adminhtml/Currencymapping/Delete.php",
// 	}
// 	dbpath := "../m2.db"

// 	db, err := loadDB(dbpath)
// 	require.NoError(t, err)
// 	fmt.Println("Loaded entries:", len(db))

// 	for k := range db {
// 		fmt.Println("first entry", k)
// 		break
// 	}

// 	for _, needle := range needles {
// 		checksum := hash([]byte(needle))
// 		hash := fmt.Sprintf("%x", checksum)
// 		_, ok := db[checksum]
// 		fmt.Println(ok, hash)
// 	}
// }
