package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
)

func digest(b [16]byte) string {
	return fmt.Sprintf("%x", b)
}

func Test_parseFile(t *testing.T) {
	hits, lines := parseFile("fixture/odd-encoding.js", "n/a", hashDB{}, false)
	fmt.Println("succeeded", len(hits), len(lines))
}

func Test_hash(t *testing.T) {
	tests := []struct {
		args []byte
		want string
	}{
		{
			[]byte("banaan"),
			"31d674be46e1ba6b54388a671c09accb",
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.args), func(t *testing.T) {
			if got := digest(hash(tt.args)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("hash() = %x (%v), want %x", got, got, tt.want)
			}
		})
	}
}

func Test_Corruption(t *testing.T) {
	fh, _ := os.Open("fixture/sample")
	defer fh.Close()

	lines := [][]byte{}

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		x := scanner.Bytes()
		l := make([]byte, len(x))
		// Need to copy, underlying Scan array may change later
		copy(l, x)
		fmt.Printf("%s\n", l)
		lines = append(lines, l)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Scanning completed, lines:", len(lines))

	for _, l := range lines {
		fmt.Printf("%s\n", l)
	}
}

func Test_NoFileSource(t *testing.T) {
	lines := [][]byte{}

	for i := 0; i < 70; i++ {
		line := fmt.Sprintf("LINE %3d =======================================================", i)
		lines = append(lines, []byte(line))
	}

	fmt.Println("Scanning completed, lines:", len(lines))

	for _, l := range lines {
		fmt.Printf("%s\n", l)
	}
}
