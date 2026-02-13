package highlight

import "testing"

var dummy = []byte("lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen ")

func BenchmarkShouldHighlight(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ShouldHighlight(dummy)
	}
}
