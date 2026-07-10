package safefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileCleansPathAndReadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "..", "secret.txt")
	cleanPath := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(cleanPath, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "secret" {
		t.Fatalf("ReadFile = %q", got)
	}
}

func TestReadFileRejectsDirectory(t *testing.T) {
	if _, err := ReadFile(t.TempDir()); err == nil {
		t.Fatal("ReadFile accepted a directory")
	}
}
