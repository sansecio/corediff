package path

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExists(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(existing, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if !Exists(existing) {
		t.Error("Exists() = false for existing file")
	}
	if Exists(filepath.Join(dir, "nope.txt")) {
		t.Error("Exists() = true for non-existing file")
	}
}
