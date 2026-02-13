package path

import "testing"

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		arg  string
		want bool
	}{
		{"generated/x/y/z.php", true},
		{"x/y/z.php", false},
		{"/vendor/x/y/z", false},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			if got := IsExcluded(tt.arg); got != tt.want {
				t.Errorf("IsExcluded() = %v, want %v", got, tt.want)
			}
		})
	}
}
