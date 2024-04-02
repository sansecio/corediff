package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
