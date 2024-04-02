package main

import "testing"

var dummy = []byte("lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen ")

func BenchmarkHighlight(b *testing.B) {
	for i := 0; i < b.N; i++ {
		shouldHighlight(dummy)
	}
}
