package main

import "testing"

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
