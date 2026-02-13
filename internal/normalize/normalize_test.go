package normalize

import (
	"fmt"
	"testing"

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
