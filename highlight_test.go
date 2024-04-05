package main

import (
	"bytes"
	"testing"
)

var dummy = []byte("lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen lorum ipsum fopen ")

func BenchmarkHighlight(b *testing.B) {
	for i := 0; i < b.N; i++ {
		shouldHighlight(dummy)
	}
}

func TestHighlightCustom(t *testing.T) {
	param := "protected $customerRepositoryFactory;"
	for _, rx := range highlightPatternsReg {
		if rx.Match([]byte(param)) {
			t.Log("Matched", rx.String())
		}
	}
	for _, p := range highlightPatternsLit {
		if bytes.Contains([]byte(param), p) {
			t.Log("Matched", string(p))
		}
	}
}
