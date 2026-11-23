package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDryRunAndConflictAreNonMutating(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing")
	if err := os.WriteFile(existing, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "nested", "new")
	files := map[string][]byte{existing: []byte("replace"), path: []byte("new")}
	var out bytes.Buffer
	if err := Write(files, nil, false, &out); err == nil {
		t.Fatal("expected conflict")
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatal("preflight wrote directory")
	}
	if err := Write(files, map[string]bool{existing: true}, true, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatal("dry run wrote directory")
	}
	got, err := os.ReadFile(existing)
	if err != nil || string(got) != "keep" {
		t.Fatal("dry run replaced file")
	}
}
